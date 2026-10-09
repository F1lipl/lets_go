package respository

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"contentserver/internal/domain"
	"contentserver/internal/model"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

func TestPostReadQueriesMySQL(t *testing.T) {
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
	postIDString, authorID, assetID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	firstID, secondID := uuid.NewString(), uuid.NewString()
	postID, err := domain.ParsePostID(postIDString)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 10*time.Second)
		defer done()
		for _, query := range []string{
			"DELETE FROM post_card_projection WHERE post_id=?",
			"DELETE FROM post_stats WHERE post_id=?",
			"DELETE FROM post_revision WHERE post_id=?",
			"DELETE FROM post WHERE post_id=?",
		} {
			if _, err := conn.ExecCtx(cleanupCtx, query, postIDString); err != nil {
				t.Errorf("fixture cleanup: %v", err)
			}
		}
		if _, err := conn.ExecCtx(cleanupCtx, "DELETE FROM media_asset WHERE asset_id=?", assetID); err != nil {
			t.Errorf("fixture cleanup: %v", err)
		}
	})
	if _, err := model.NewMediaAssetModel(conn).Insert(ctx, &model.MediaAsset{
		AssetId: assetID, OwnerId: authorID, StorageProvider: "test", BucketName: "test",
		ObjectKey: assetID, MimeType: "image/jpeg", FileSize: 1, Status: 3,
		Width: sql.NullInt64{Int64: 640, Valid: true}, Height: sql.NullInt64{Int64: 480, Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := model.NewPostModel(conn).Insert(ctx, &model.Post{
		PostId: postIDString, AuthorId: authorID, LifecycleStatus: 3,
		Visibility: 1, AvailabilityStatus: 1,
		PublishedRevisionId: sql.NullString{String: firstID, Valid: true},
		RevisionSequence:    1, PostVersion: 2,
	}); err != nil {
		t.Fatal(err)
	}
	first := &model.PostRevision{
		RevisionId: firstID, PostId: postIDString, RevisionNumber: 1,
		SourceDraftVersion: 1, Title: "first", Summary: "summary",
		CoverAssetId:          sql.NullString{String: assetID, Valid: true},
		DocumentSchemaVersion: 1, DocumentJson: `{"schemaVersion":1,"blocks":[]}`,
		PublishedAt: time.Now().UTC().Truncate(time.Millisecond),
	}
	if _, err := model.NewPostRevisionModel(conn).Insert(ctx, first); err != nil {
		t.Fatal(err)
	}
	if _, err := model.NewPostCardProjectionModel(conn).Insert(ctx, &model.PostCardProjection{
		PostId: postIDString, RevisionId: firstID, AuthorId: authorID,
		Title: "first", CoverAssetId: sql.NullString{String: assetID, Valid: true},
		Visibility: 1, AvailabilityStatus: 1,
		PublishedAt: first.PublishedAt, SourcePostVersion: 2,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := model.NewPostStatsModel(conn).Insert(ctx, &model.PostStats{
		PostId: postIDString, LikeCount: 8, FavoriteCount: 3, CommentCount: 2,
	}); err != nil {
		t.Fatal(err)
	}
	repo := NewPostRepository()
	read := func() (*domain.PostRevision, *domain.PostCounts, []domain.PublishedAsset, error) {
		var revision *domain.PostRevision
		var counts *domain.PostCounts
		var assets []domain.PublishedAsset
		post, err := repo.FindPost(ctx, conn, postID)
		if err != nil {
			return nil, nil, nil, err
		}
		revisionID, err := post.CurrentPublicRevision()
		if err != nil {
			return nil, nil, nil, err
		}
		currentRevision, err := repo.FindRevision(ctx, conn, revisionID)
		if err != nil {
			return nil, nil, nil, err
		}
		if currentRevision.RevisionId != revisionID || !currentRevision.BelongsTo(postID) {
			return nil, nil, nil, domain.ErrRevisionNotFound
		}
		revision = currentRevision
		assets, err = repo.FindReadyAssets(ctx, conn, []domain.AssetID{revision.Cover.AssetID})
		if err != nil {
			return nil, nil, nil, err
		}
		counts, err = repo.FindCounts(ctx, conn, postID)
		return revision, counts, assets, err
	}
	got, counts, assets, err := read()
	if err != nil || got.RevisionId.String() != firstID || got.Title != "first" || counts.Likes != 8 ||
		len(assets) != 1 || assets[0].ID.String() != assetID || assets[0].Width != 640 {
		t.Fatalf("post to current revision = %+v, counts = %+v, assets = %+v, err = %v", got, counts, assets, err)
	}
	second := *first
	second.RevisionId, second.RevisionNumber, second.Title = secondID, 2, "second"
	second.PublishedAt = first.PublishedAt.Add(time.Second)
	if _, err := model.NewPostRevisionModel(conn).Insert(ctx, &second); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecCtx(ctx,
		"UPDATE post SET published_revision_id=?,revision_sequence=2,post_version=3 WHERE post_id=?",
		secondID, postIDString); err != nil {
		t.Fatal(err)
	}
	got, _, _, err = read()
	if err != nil || got.RevisionId.String() != secondID || got.Title != "second" {
		t.Fatalf("read current revision while card lags = %+v, err = %v", got, err)
	}
	if err := model.NewPostCardProjectionModel(conn).Delete(ctx, postIDString); err != nil {
		t.Fatal(err)
	}
	got, _, _, err = read()
	if err != nil || got.RevisionId.String() != secondID {
		t.Fatalf("read current revision without card = %+v, err = %v", got, err)
	}
	if _, err := conn.ExecCtx(ctx, "UPDATE post SET visibility=3,post_version=4 WHERE post_id=?", postIDString); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := read(); !errors.Is(err, domain.ErrPostNotVisible) {
		t.Fatalf("private post = %v", err)
	}
	if _, err := conn.ExecCtx(ctx, "UPDATE post SET lifecycle_status=4,deleted_at=NOW(3),post_version=5 WHERE post_id=?", postIDString); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := read(); !errors.Is(err, domain.ErrPostNotFound) {
		t.Fatalf("deleted post = %v", err)
	}
}
