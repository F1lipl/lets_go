package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// TestContentModelsAgainstMySQL exercises generated SQL against the migrated
// schema. All fixture writes use one transaction that is always rolled back.
func TestContentModelsAgainstMySQL(t *testing.T) {
	dsn := os.Getenv("CONTENT_MODEL_TEST_DSN")
	if dsn == "" {
		t.Skip("set CONTENT_MODEL_TEST_DSN to run the MySQL integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	conn := sqlx.NewMysql(dsn)
	db, err := conn.RawDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var obsolete int
	err = conn.QueryRowCtx(ctx, &obsolete, "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND (COLUMN_NAME LIKE 'route\\_%' OR COLUMN_NAME='presentation_mode' OR COLUMN_NAME IN ('create_request_id','publish_request_id','upload_request_id') OR TABLE_NAME='post_revision_place')")
	if err != nil || obsolete != 0 {
		t.Fatalf("obsolete schema fields = %d, error = %v", obsolete, err)
	}

	postID, revisionID, authorID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	rollback := errors.New("rollback integration fixtures")
	err = conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		tx := sqlx.NewSqlConnFromSession(session)
		_, err := NewPostModel(tx).Insert(ctx, &Post{
			PostId: postID, AuthorId: authorID,
			LifecycleStatus: 1, Visibility: 3, AvailabilityStatus: 1, PostVersion: 1,
		})
		if err != nil {
			return fmt.Errorf("insert post: %w", err)
		}
		drafts := NewPostDraftModel(tx)
		draft := &PostDraft{
			PostId: postID, DraftVersion: 1,
			Title: "A text and image post", Summary: "summary",
			DocumentSchemaVersion: 1, DocumentJson: `{"schemaVersion":1,"blocks":[{"blockId":"b1","blockType":"text","sortOrder":0,"text":"hello"}]}`,
			TagNamesJson: "[]", PlainText: "hello", BlockCount: 1,
		}
		if _, err := drafts.Insert(ctx, draft); err != nil {
			return fmt.Errorf("insert draft: %w", err)
		}
		draft.Title = "Updated title"
		if err := drafts.UpdateWithVersion(ctx, draft, 1); err != nil {
			return fmt.Errorf("update draft: %w", err)
		}
		if err := drafts.UpdateWithVersion(ctx, draft, 1); !errors.Is(err, ErrVersionConflict) {
			return fmt.Errorf("stale draft update error = %v, want %v", err, ErrVersionConflict)
		}
		gotDraft, err := drafts.FindOne(ctx, postID)
		if err != nil {
			return fmt.Errorf("read draft: %w", err)
		}
		if gotDraft.Title != draft.Title || gotDraft.DraftVersion != 2 || !json.Valid([]byte(gotDraft.DocumentJson)) {
			return errors.New("draft did not round-trip")
		}
		revisions := NewPostRevisionModel(tx)
		now := time.Now().UTC().Truncate(time.Millisecond)
		revision := &PostRevision{
			RevisionId: revisionID, PostId: postID, RevisionNumber: 1,
			SourceDraftVersion: 2,
			Title:              draft.Title, Summary: draft.Summary, DocumentSchemaVersion: 1,
			DocumentJson: draft.DocumentJson, PlainText: draft.PlainText,
			BlockCount: 1, PublishedAt: now,
		}
		if _, err := revisions.Insert(ctx, revision); err != nil {
			return fmt.Errorf("insert revision: %w", err)
		}
		gotRevision, err := revisions.FindOneByPostIdRevisionNumber(ctx, postID, revision.RevisionNumber)
		if err != nil {
			return fmt.Errorf("read revision: %w", err)
		}
		if gotRevision.RevisionId != revisionID || gotRevision.Title != draft.Title {
			return errors.New("revision did not round-trip")
		}
		cards := NewPostCardProjectionModel(tx)
		card := &PostCardProjection{
			PostId: postID, RevisionId: revisionID, AuthorId: authorID,
			Title: revision.Title, Summary: revision.Summary, Visibility: 1,
			AvailabilityStatus: 1, PublishedAt: now, SourcePostVersion: 2,
		}
		if _, err := cards.Insert(ctx, card); err != nil {
			return fmt.Errorf("insert card: %w", err)
		}
		card.Visibility, card.SourcePostVersion = 3, 3
		if err := cards.Update(ctx, card); err != nil {
			return fmt.Errorf("update card: %w", err)
		}
		gotCard, err := cards.FindOne(ctx, postID)
		if err != nil {
			return fmt.Errorf("read card: %w", err)
		}
		if gotCard.Visibility != 3 || gotCard.RevisionId != revisionID || gotCard.SourcePostVersion != 3 {
			return errors.New("card did not round-trip")
		}
		if gotCard.CoverAssetId != (sql.NullString{}) {
			return errors.New("nullable cover did not round-trip")
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatalf("model round-trip failed: %v", err)
	}
	if _, err := NewPostModel(conn).FindOne(ctx, postID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("fixtures were not rolled back: %v", err)
	}
}
