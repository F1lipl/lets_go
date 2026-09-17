package logic

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"contentserver/internal/domain"
	"contentserver/internal/model"
	"contentserver/internal/svc"
	"contentserver/internal/types"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// Fail after revision and post writes to verify that the whole transaction rolls back.
type failOutboxSession struct{ sqlx.Session }

func (s failOutboxSession) ExecCtx(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if strings.Contains(query, "INSERT INTO outbox_event") {
		return nil, errors.New("test outbox failure")
	}
	return s.Session.ExecCtx(ctx, query, args...)
}

type failOutboxConn struct{ sqlx.SqlConn }

func (c failOutboxConn) TransactCtx(ctx context.Context, fn func(context.Context, sqlx.Session) error) error {
	return c.SqlConn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		return fn(ctx, failOutboxSession{session})
	})
}

func TestPublishPostMySQL(t *testing.T) {
	dsn := os.Getenv("CONTENT_MODEL_TEST_DSN")
	if dsn == "" {
		t.Skip("set CONTENT_MODEL_TEST_DSN for MySQL integration")
	}
	conn := sqlx.NewMysql(dsn)
	db, err := conn.RawDB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	postID, authorID, assetID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	// Only remove rows belonging to this test's generated IDs.
	t.Cleanup(func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 10*time.Second)
		defer done()
		queries := []struct {
			sql  string
			args []any
		}{
			{"DELETE FROM outbox_event WHERE aggregate_id=?", []any{postID}},
			{"DELETE FROM content_asset_ref WHERE asset_id=?", []any{assetID}},
			{"DELETE FROM post_revision WHERE post_id=?", []any{postID}},
			{"DELETE FROM post_draft WHERE post_id=?", []any{postID}},
			{"DELETE FROM post WHERE post_id=?", []any{postID}},
			{"DELETE FROM media_asset WHERE asset_id=?", []any{assetID}},
		}
		for _, q := range queries {
			if _, err := conn.ExecCtx(cleanupCtx, q.sql, q.args...); err != nil {
				t.Errorf("fixture cleanup: %v", err)
			}
		}
	})
	if _, err := model.NewMediaAssetModel(conn).Insert(ctx, &model.MediaAsset{
		AssetId: assetID, OwnerId: authorID, StorageProvider: "test", BucketName: "test",
		ObjectKey: assetID, MimeType: "image/jpeg", FileSize: 1, Status: 3,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := model.NewPostModel(conn).Insert(ctx, &model.Post{
		PostId: postID, AuthorId: authorID, LifecycleStatus: 1, Visibility: 1, AvailabilityStatus: 1, PostVersion: 1,
	}); err != nil {
		t.Fatal(err)
	}
	document := `{"schemaVersion":1,"blocks":[{"BlockID":"b1","BlockType":"paragraph","Text":"hello"}]}`
	if _, err := model.NewPostDraftModel(conn).Insert(ctx, &model.PostDraft{
		PostId: postID, DraftVersion: 1, Title: "first", CoverAssetId: sql.NullString{String: assetID, Valid: true},
		DocumentSchemaVersion: 1, DocumentJson: document, TagNamesJson: `["travel"]`, PlainText: "hello", BlockCount: 1,
	}); err != nil {
		t.Fatal(err)
	}
	call := func(c sqlx.SqlConn, actor string, version, draftVersion uint64) (*types.PublishPostData, error) {
		requestCtx := context.WithValue(context.WithValue(ctx, "userId", actor), "sessionId", "test")
		logic := NewPublishPostLogic(requestCtx, &svc.ServiceContext{DB: c})
		return logic.PublishPost(&types.PublishPostRequest{PostId: postID, ExpectedPostVersion: version, ExpectedDraftVersion: draftVersion})
	}
	if _, err := call(conn, uuid.NewString(), 1, 1); !errors.Is(err, domain.ErrPostOperationNotAllowed) {
		t.Fatalf("different author: %v", err)
	}
	if _, err := call(conn, authorID, 1, 2); !errors.Is(err, domain.ErrDraftVersionConflict) {
		t.Fatalf("draft conflict: %v", err)
	}
	if _, err := conn.ExecCtx(ctx, "UPDATE media_asset SET status=2 WHERE asset_id=?", assetID); err != nil {
		t.Fatal(err)
	}
	if _, err := call(conn, authorID, 1, 1); !errors.Is(err, domain.ErrMediaAssetNotReady) {
		t.Fatalf("unfinished asset: %v", err)
	}
	if _, err := conn.ExecCtx(ctx, "UPDATE media_asset SET status=3 WHERE asset_id=?", assetID); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecCtx(ctx, "UPDATE post_draft SET image_count=1 WHERE post_id=?", postID); err != nil {
		t.Fatal(err)
	}
	if _, err := call(conn, authorID, 1, 1); !errors.Is(err, domain.ErrDraftContentInvalid) {
		t.Fatalf("missing image references: %v", err)
	}
	if _, err := conn.ExecCtx(ctx, "UPDATE post_draft SET image_count=0 WHERE post_id=?", postID); err != nil {
		t.Fatal(err)
	}
	if result, err := call(failOutboxConn{conn}, authorID, 1, 1); err == nil || result != nil {
		t.Fatal("failed outbox returned success")
	}
	var count int
	if err := conn.QueryRowCtx(ctx, &count, "SELECT COUNT(*) FROM post_revision WHERE post_id=?", postID); err != nil || count != 0 {
		t.Fatalf("rollback revision count=%d err=%v", count, err)
	}
	row, err := model.NewPostModel(conn).FindOne(ctx, postID)
	if err != nil || row.PostVersion != 1 || row.RevisionSequence != 0 {
		t.Fatalf("rollback post: %+v %v", row, err)
	}
	first, err := call(conn, authorID, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if first.RevisionNumber != 1 || first.PostVersion != 2 {
		t.Fatalf("first publication: %+v", first)
	}
	if _, err := call(conn, authorID, 1, 1); !errors.Is(err, domain.ErrPostVersionConflict) {
		t.Fatalf("stale publish: %v", err)
	}
	firstRow, _ := model.NewPostModel(conn).FindOne(ctx, postID)
	if _, err := conn.ExecCtx(ctx, "UPDATE post_draft SET title='second',draft_version=2 WHERE post_id=?", postID); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := call(conn, authorID, 2, 2); results <- err }()
	}
	wg.Wait()
	close(results)
	success, conflicts := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, domain.ErrPostVersionConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatalf("concurrent results: %d successes, %d conflicts", success, conflicts)
	}
	row, err = model.NewPostModel(conn).FindOne(ctx, postID)
	if err != nil || row.RevisionSequence != 2 || row.PostVersion != 3 || row.LifecycleStatus != 3 || !row.FirstPublishedAt.Time.Equal(firstRow.FirstPublishedAt.Time) {
		t.Fatalf("republish post: %+v %v", row, err)
	}
	oldRevision, err := model.NewPostRevisionModel(conn).FindOne(ctx, first.RevisionId)
	if err != nil || oldRevision.Title != "first" {
		t.Fatalf("old snapshot changed: %+v %v", oldRevision, err)
	}
	draftRow, _ := model.NewPostDraftModel(conn).FindOne(ctx, postID)
	if oldRevision.DocumentJson != draftRow.DocumentJson {
		t.Fatal("publication re-encoded document")
	}
	if err := conn.QueryRowCtx(ctx, &count, "SELECT COUNT(*) FROM outbox_event WHERE aggregate_id=? AND status=1 AND event_type='PostPublished'", postID); err != nil || count != 2 {
		t.Fatalf("outbox count=%d err=%v", count, err)
	}
	if _, err := conn.ExecCtx(ctx, "UPDATE post SET lifecycle_status=4,post_version=4 WHERE post_id=?", postID); err != nil {
		t.Fatal(err)
	}
	if _, err := call(conn, authorID, 4, 2); !errors.Is(err, domain.ErrPostAlreadyDeleted) {
		t.Fatalf("deleted post: %v", err)
	}
}
