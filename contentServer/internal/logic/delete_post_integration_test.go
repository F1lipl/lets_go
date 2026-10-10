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

type failCardDeleteSession struct{ sqlx.Session }

func (s failCardDeleteSession) ExecCtx(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if strings.Contains(strings.ToLower(query), "delete from `post_card_projection`") {
		return nil, errors.New("test card deletion failure")
	}
	return s.Session.ExecCtx(ctx, query, args...)
}

type failCardDeleteConn struct{ sqlx.SqlConn }

func (c failCardDeleteConn) TransactCtx(ctx context.Context, fn func(context.Context, sqlx.Session) error) error {
	return c.SqlConn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		return fn(ctx, failCardDeleteSession{session})
	})
}

func TestDeletePostMySQL(t *testing.T) {
	dsn := os.Getenv("CONTENT_MODEL_TEST_DSN")
	if dsn == "" {
		t.Skip("set CONTENT_MODEL_TEST_DSN for MySQL integration")
	}
	conn := sqlx.NewMysql(dsn)
	if _, err := conn.RawDB(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	postID, authorID, revisionID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	t.Cleanup(func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 10*time.Second)
		defer done()
		for _, query := range []string{
			"DELETE FROM post_card_projection WHERE post_id=?",
			"DELETE FROM post WHERE post_id=?",
		} {
			if _, err := conn.ExecCtx(cleanupCtx, query, postID); err != nil {
				t.Errorf("fixture cleanup: %v", err)
			}
		}
	})
	now := time.Now().UTC().Truncate(time.Millisecond)
	if _, err := model.NewPostModel(conn).Insert(ctx, &model.Post{
		PostId: postID, AuthorId: authorID, LifecycleStatus: uint64(domain.LifecyclePublished),
		Visibility: uint64(domain.VisibilityPublic), AvailabilityStatus: domain.AvailabilityNormal,
		PublishedRevisionId: sql.NullString{String: revisionID, Valid: true}, RevisionSequence: 1,
		PostVersion: 2, FirstPublishedAt: sql.NullTime{Time: now, Valid: true}, LastPublishedAt: sql.NullTime{Time: now, Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := model.NewPostCardProjectionModel(conn).Insert(ctx, &model.PostCardProjection{
		PostId: postID, RevisionId: revisionID, AuthorId: authorID, Title: "test post",
		Visibility: uint64(domain.VisibilityPublic), AvailabilityStatus: domain.AvailabilityNormal,
		PublishedAt: now, SourcePostVersion: 2,
	}); err != nil {
		t.Fatal(err)
	}
	call := func(db sqlx.SqlConn, actor string, version uint64) (*types.DeletePostData, error) {
		requestCtx := context.WithValue(context.WithValue(ctx, "userId", actor), "sessionId", "test")
		return NewDeletePostLogic(requestCtx, &svc.ServiceContext{DB: db}).DeletePost(
			&types.DeletePostRequest{PostId: postID, ExpectedPostVersion: version})
	}
	if _, err := call(conn, uuid.NewString(), 2); !errors.Is(err, domain.ErrPostOperationNotAllowed) {
		t.Fatalf("different author: %v", err)
	}
	if _, err := call(conn, authorID, 1); !errors.Is(err, domain.ErrPostVersionConflict) {
		t.Fatalf("stale version: %v", err)
	}
	if result, err := call(failCardDeleteConn{conn}, authorID, 2); err == nil || result != nil {
		t.Fatalf("card deletion failure committed: result=%+v err=%v", result, err)
	}
	row, err := model.NewPostModel(conn).FindOne(ctx, postID)
	if err != nil || row.PostVersion != 2 || row.DeletedAt.Valid {
		t.Fatalf("failed transaction changed post: %+v err=%v", row, err)
	}
	if _, err := model.NewPostCardProjectionModel(conn).FindOne(ctx, postID); err != nil {
		t.Fatalf("failed transaction removed card: %v", err)
	}

	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := call(conn, authorID, 2)
			if err == nil && (result == nil || result.PostVersion != 3 || result.Status != "deleted" || result.DeletedAt == "") {
				err = errors.New("invalid successful delete result")
			}
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	successes, alreadyDeleted := 0, 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, domain.ErrPostAlreadyDeleted):
			alreadyDeleted++
		default:
			t.Fatalf("concurrent delete: %v", err)
		}
	}
	if successes != 1 || alreadyDeleted != 1 {
		t.Fatalf("concurrent delete successes=%d alreadyDeleted=%d", successes, alreadyDeleted)
	}
	row, err = model.NewPostModel(conn).FindOne(ctx, postID)
	if err != nil || row.PostVersion != 3 || row.LifecycleStatus != uint64(domain.LifecycleDeleted) || !row.DeletedAt.Valid || !row.PublishedRevisionId.Valid {
		t.Fatalf("deleted row = %+v err=%v", row, err)
	}
	if _, err := model.NewPostCardProjectionModel(conn).FindOne(ctx, postID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("card still visible: %v", err)
	}
	if _, err := NewGetPostLogic(ctx, &svc.ServiceContext{DB: conn}).GetPost(&types.GetPostRequest{PostId: postID}); !errors.Is(err, domain.ErrPostNotFound) {
		t.Fatalf("deleted post still readable: %v", err)
	}
}
