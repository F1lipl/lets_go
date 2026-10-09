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

type failDraftRefSession struct{ sqlx.Session }

func (s failDraftRefSession) ExecCtx(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if strings.Contains(strings.ToLower(query), "insert into `content_asset_ref`") {
		return nil, errors.New("test reference write failure")
	}
	return s.Session.ExecCtx(ctx, query, args...)
}

type failDraftRefConn struct{ sqlx.SqlConn }

func (c failDraftRefConn) TransactCtx(ctx context.Context, fn func(context.Context, sqlx.Session) error) error {
	return c.SqlConn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		return fn(ctx, failDraftRefSession{session})
	})
}

func TestCreateAndSavePostDraftMySQL(t *testing.T) {
	dsn := os.Getenv("CONTENT_MODEL_TEST_DSN")
	if dsn == "" {
		t.Skip("set CONTENT_MODEL_TEST_DSN for MySQL integration")
	}
	conn := sqlx.NewMysql(dsn)
	// sqlx caches MySQL connections for this DSN; other integration cases in
	// this package reuse it, so a single case must not close the shared pool.
	if _, err := conn.RawDB(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	authorID, coverID, imageID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	postID := ""
	t.Cleanup(func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 10*time.Second)
		defer done()
		if postID != "" {
			for _, query := range []string{
				"DELETE FROM content_asset_ref WHERE owner_type=1 AND owner_id=?",
				"DELETE FROM post_draft WHERE post_id=?",
				"DELETE FROM post WHERE post_id=?",
			} {
				if _, err := conn.ExecCtx(cleanupCtx, query, postID); err != nil {
					t.Errorf("fixture cleanup: %v", err)
				}
			}
		}
		for _, id := range []string{coverID, imageID} {
			if _, err := conn.ExecCtx(cleanupCtx, "DELETE FROM media_asset WHERE asset_id=?", id); err != nil {
				t.Errorf("fixture cleanup media: %v", err)
			}
		}
	})
	for _, id := range []string{coverID, imageID} {
		if _, err := model.NewMediaAssetModel(conn).Insert(ctx, &model.MediaAsset{
			AssetId: id, OwnerId: authorID, StorageProvider: "test", BucketName: "test",
			ObjectKey: id, MimeType: "image/jpeg", FileSize: 1, Status: 3,
		}); err != nil {
			t.Fatal(err)
		}
	}
	requestCtx := context.WithValue(context.WithValue(ctx, "userId", authorID), "sessionId", "test")
	createReq := &types.CreatePostRequest{
		Visibility: "public", Title: "first", Cover: types.CoverInput{AssetId: coverID},
		Document: types.PostDocument{SchemaVersion: 1, Blocks: []types.ContentBlock{
			{BlockId: "image-1", BlockType: "image", AssetIds: []string{imageID}},
		}}, TagNames: []string{"#旅行"},
	}
	var before int
	if err := conn.QueryRowCtx(ctx, &before, "SELECT COUNT(*) FROM post WHERE author_id=?", authorID); err != nil {
		t.Fatal(err)
	}
	if result, err := NewCreatePostLogic(requestCtx, &svc.ServiceContext{DB: failDraftRefConn{conn}}).CreatePost(createReq); err == nil || result != nil {
		t.Fatalf("failed reference write returned result=%+v err=%v", result, err)
	}
	var after int
	if err := conn.QueryRowCtx(ctx, &after, "SELECT COUNT(*) FROM post WHERE author_id=?", authorID); err != nil || after != before {
		t.Fatalf("failed create left a post: before=%d after=%d err=%v", before, after, err)
	}
	created, err := NewCreatePostLogic(requestCtx, &svc.ServiceContext{DB: conn}).CreatePost(createReq)
	if err != nil {
		t.Fatal(err)
	}
	postID = created.PostId
	if created.PostVersion != 1 || created.DraftVersion != 1 || created.Status != "draft" {
		t.Fatalf("created = %+v", created)
	}
	var refs int
	if err := conn.QueryRowCtx(ctx, &refs, "SELECT COUNT(*) FROM content_asset_ref WHERE owner_type=1 AND owner_id=? AND source_version=1", postID); err != nil || refs != 2 {
		t.Fatalf("initial refs=%d err=%v", refs, err)
	}
	newDocument := types.PostDocument{SchemaVersion: 1, Blocks: []types.ContentBlock{
		{BlockId: "text-1", BlockType: "paragraph", Text: "updated"},
	}}
	saveReq := &types.SavePostDraftRequest{PostId: postID, ExpectedDraftVersion: 1, Title: "second", Document: newDocument}
	otherCtx := context.WithValue(context.WithValue(ctx, "userId", uuid.NewString()), "sessionId", "test")
	if _, err := NewSavePostDraftLogic(otherCtx, &svc.ServiceContext{DB: conn}).SavePostDraft(saveReq); !errors.Is(err, domain.ErrPostOperationNotAllowed) {
		t.Fatalf("other author save: %v", err)
	}
	unknownAssetReq := *saveReq
	unknownAssetReq.Cover = types.CoverInput{AssetId: uuid.NewString()}
	if _, err := NewSavePostDraftLogic(requestCtx, &svc.ServiceContext{DB: conn}).SavePostDraft(&unknownAssetReq); !errors.Is(err, domain.ErrMediaAssetNotFound) {
		t.Fatalf("unknown image: %v", err)
	}
	failSaveReq := *saveReq
	failSaveReq.Cover = types.CoverInput{AssetId: coverID}
	if result, err := NewSavePostDraftLogic(requestCtx, &svc.ServiceContext{DB: failDraftRefConn{conn}}).SavePostDraft(&failSaveReq); err == nil || result != nil {
		t.Fatalf("failed ref update returned result=%+v err=%v", result, err)
	}
	unchanged, err := model.NewPostDraftModel(conn).FindOne(ctx, postID)
	if err != nil || unchanged.DraftVersion != 1 || unchanged.Title != "first" {
		t.Fatalf("failed save changed draft: %+v err=%v", unchanged, err)
	}
	saved, err := NewSavePostDraftLogic(requestCtx, &svc.ServiceContext{DB: conn}).SavePostDraft(saveReq)
	if err != nil || saved.DraftVersion != 2 || saved.PostId != postID || saved.UpdatedAt == "" {
		t.Fatalf("saved = %+v err=%v", saved, err)
	}
	row, err := model.NewPostDraftModel(conn).FindOne(ctx, postID)
	if err != nil || row.DraftVersion != 2 || row.PlainText != "updated\n" || row.CoverAssetId.Valid || row.ImageCount != 0 {
		t.Fatalf("stored draft = %+v err=%v", row, err)
	}
	if err := conn.QueryRowCtx(ctx, &refs, "SELECT COUNT(*) FROM content_asset_ref WHERE owner_type=1 AND owner_id=?", postID); err != nil || refs != 0 {
		t.Fatalf("replacement refs=%d err=%v", refs, err)
	}
	if _, err := NewSavePostDraftLogic(requestCtx, &svc.ServiceContext{DB: conn}).SavePostDraft(saveReq); !errors.Is(err, domain.ErrDraftVersionConflict) {
		t.Fatalf("stale save: %v", err)
	}
	concurrent := *saveReq
	concurrent.ExpectedDraftVersion = 2
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := NewSavePostDraftLogic(requestCtx, &svc.ServiceContext{DB: conn}).SavePostDraft(&concurrent)
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	successes, conflicts := 0, 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, domain.ErrDraftVersionConflict):
			conflicts++
		default:
			t.Fatal(err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent saves: success=%d conflict=%d", successes, conflicts)
	}
}
