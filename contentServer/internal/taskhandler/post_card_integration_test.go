package taskhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"contentserver/internal/event"
	"contentserver/internal/model"
	"contentserver/internal/respository"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

func TestPostCardHandlerMySQL(t *testing.T) {
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
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	postID, authorID, assetID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	firstRevisionID, secondRevisionID := uuid.NewString(), uuid.NewString()
	now := time.Now().UTC().Truncate(time.Millisecond)
	rollback := errors.New("rollback card fixtures")
	err = conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		tx := sqlx.NewSqlConnFromSession(session)
		if _, err := model.NewMediaAssetModel(tx).Insert(ctx, &model.MediaAsset{
			AssetId: assetID, OwnerId: authorID, StorageProvider: "test", BucketName: "test",
			ObjectKey: assetID, MimeType: "image/jpeg", FileSize: 1, Status: 3,
		}); err != nil {
			return err
		}
		if _, err := model.NewPostModel(tx).Insert(ctx, &model.Post{
			PostId: postID, AuthorId: authorID, LifecycleStatus: 3, Visibility: 1,
			AvailabilityStatus: 1, PublishedRevisionId: sql.NullString{String: firstRevisionID, Valid: true},
			RevisionSequence: 1, PostVersion: 2,
		}); err != nil {
			return err
		}
		revisions := model.NewPostRevisionModel(tx)
		first := &model.PostRevision{
			RevisionId: firstRevisionID, PostId: postID, RevisionNumber: 1,
			SourceDraftVersion: 1, Title: "first", Summary: "first summary",
			CoverAssetId:          sql.NullString{String: assetID, Valid: true},
			DocumentSchemaVersion: 1, DocumentJson: `{"blocks":[]}`, PublishedAt: now,
		}
		if _, err := revisions.Insert(ctx, first); err != nil {
			return err
		}
		cards := model.NewPostCardProjectionModel(tx)
		handler := NewPostCardHandler(respository.NewPostCardRepository())
		input := func(revisionID string, version uint64) *event.TaskContext {
			payload, err := json.Marshal(postPublishedPayload{
				SchemaVersion: 1, PostID: postID, AuthorID: authorID,
				RevisionID: revisionID, PostVersion: version,
			})
			if err != nil {
				t.Fatal(err)
			}
			return &event.TaskContext{EventType: "PostPublished", SchemaVersion: 1, Payload: payload}
		}
		if err := handler.Handle(ctx, session, input(firstRevisionID, 2)); err != nil {
			return err
		}
		card, err := cards.FindOne(ctx, postID)
		if err != nil || card.Title != "first" || card.RevisionId != firstRevisionID ||
			!card.CoverAssetId.Valid || card.CoverAssetId.String != assetID || card.SourcePostVersion != 2 {
			t.Fatalf("first card = %+v, err = %v", card, err)
		}
		if err := handler.Handle(ctx, session, input(firstRevisionID, 2)); err != nil {
			t.Fatalf("duplicate delivery: %v", err)
		}

		second := *first
		second.RevisionId, second.RevisionNumber, second.Title = secondRevisionID, 2, "second"
		second.PublishedAt = now.Add(time.Second)
		if _, err := revisions.Insert(ctx, &second); err != nil {
			return err
		}
		if _, err := tx.ExecCtx(ctx,
			"UPDATE post SET published_revision_id=?,revision_sequence=2,post_version=3 WHERE post_id=?",
			secondRevisionID, postID); err != nil {
			return err
		}
		if err := handler.Handle(ctx, session, input(firstRevisionID, 2)); !errors.Is(err, event.ErrSuperseded) {
			t.Fatalf("old event after republish = %v", err)
		}
		if err := handler.Handle(ctx, session, input(secondRevisionID, 3)); err != nil {
			return err
		}
		card, err = cards.FindOne(ctx, postID)
		if err != nil || card.Title != "second" || card.RevisionId != secondRevisionID || card.SourcePostVersion != 3 {
			t.Fatalf("new card = %+v, err = %v", card, err)
		}

		if _, err := tx.ExecCtx(ctx,
			"UPDATE post SET lifecycle_status=4,deleted_at=NOW(3),post_version=4 WHERE post_id=?", postID); err != nil {
			return err
		}
		if err := cards.Delete(ctx, postID); err != nil {
			return err
		}
		if err := handler.Handle(ctx, session, input(secondRevisionID, 3)); !errors.Is(err, event.ErrSuperseded) {
			t.Fatalf("event after deletion = %v", err)
		}
		if _, err := cards.FindOne(ctx, postID); !errors.Is(err, model.ErrNotFound) {
			t.Fatalf("deleted card was recreated: %v", err)
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatalf("post card transaction: %v", err)
	}
}
