package logic

import (
	"context"
	"errors"
	"testing"
	"time"

	"contentserver/internal/domain"
	"contentserver/internal/respository"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type fakePostReadQueries struct {
	post                *domain.Post
	revision            *domain.PostRevision
	postReadCount       int
	revisionReadCount   int
	assetReadCount      int
	assetErr            error
	waitForMediaTimeout bool
}

var _ respository.PostReadQueries = (*fakePostReadQueries)(nil)

func (f *fakePostReadQueries) FindPost(context.Context, sqlx.SqlConn, domain.PostID) (*domain.Post, error) {
	f.postReadCount++
	return f.post, nil
}
func (f *fakePostReadQueries) FindRevision(context.Context, sqlx.SqlConn, domain.RevisionID) (*domain.PostRevision, error) {
	f.revisionReadCount++
	if f.revision == nil {
		return nil, domain.ErrRevisionNotFound
	}
	return f.revision, nil
}
func (f *fakePostReadQueries) FindReadyAssets(ctx context.Context, _ sqlx.SqlConn, _ []domain.AssetID) ([]domain.PublishedAsset, error) {
	f.assetReadCount++
	if f.waitForMediaTimeout {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return nil, f.assetErr
}
func (f *fakePostReadQueries) FindTags(context.Context, sqlx.SqlConn, domain.RevisionID) ([]domain.PublishedTag, error) {
	return nil, nil
}
func (f *fakePostReadQueries) FindCounts(context.Context, sqlx.SqlConn, domain.PostID) (*domain.PostCounts, error) {
	return nil, nil
}

func TestReadPublishedPostUsesCurrentRevision(t *testing.T) {
	postID, _ := domain.ParsePostID(uuid.NewString())
	authorID, _ := domain.ParseUserID(uuid.NewString())
	currentID, _ := domain.ParseRevisionID(uuid.NewString())
	queries := &fakePostReadQueries{
		post: &domain.Post{PostID: postID, AuthorID: authorID, LifecycleStatus: domain.LifecyclePublished,
			Visibility: domain.VisibilityPublic, AvailabilityStatus: domain.AvailabilityNormal,
			PublishedRevisionID: &currentID, Version: 3},
		revision: &domain.PostRevision{PostId: postID, RevisionId: currentID,
			Document: `{"schemaVersion":1,"blocks":[]}`, DocumentSchemaVersion: 1},
	}
	data, err := readPublishedPost(context.Background(), queries, nil, postID, time.Second)
	if err != nil || queries.postReadCount != 1 || queries.revisionReadCount != 1 || data.RevisionId != currentID.String() {
		t.Fatalf("current revision = %+v, post reads = %d, revision reads = %d, err = %v", data, queries.postReadCount, queries.revisionReadCount, err)
	}
	queries.post.Visibility = domain.VisibilityPrivate
	if _, err := readPublishedPost(context.Background(), queries, nil, postID, time.Second); !errors.Is(err, domain.ErrPostNotVisible) {
		t.Fatalf("private post = %v", err)
	}
	if queries.assetReadCount != 0 {
		t.Fatalf("media was read for a non-public post: %d", queries.assetReadCount)
	}
	if queries.revisionReadCount != 1 {
		t.Fatalf("revision was read for a non-public post: %d", queries.revisionReadCount)
	}
}

func TestReadPublishedPostRejectsMismatchedRevision(t *testing.T) {
	postID, _ := domain.ParsePostID(uuid.NewString())
	authorID, _ := domain.ParseUserID(uuid.NewString())
	currentID, _ := domain.ParseRevisionID(uuid.NewString())
	otherID, _ := domain.ParseRevisionID(uuid.NewString())
	queries := &fakePostReadQueries{
		post: &domain.Post{PostID: postID, AuthorID: authorID,
			LifecycleStatus: domain.LifecyclePublished, Visibility: domain.VisibilityPublic,
			AvailabilityStatus: domain.AvailabilityNormal, PublishedRevisionID: &currentID},
		revision: &domain.PostRevision{PostId: postID, RevisionId: otherID},
	}
	if _, err := readPublishedPost(context.Background(), queries, nil, postID, time.Second); err == nil {
		t.Fatal("mismatched current revision accepted")
	}
	if queries.assetReadCount != 0 {
		t.Fatal("loaded ancillary data for mismatched revision")
	}
}

func TestReadPublishedPostRequiresCurrentRevision(t *testing.T) {
	postID, _ := domain.ParsePostID(uuid.NewString())
	authorID, _ := domain.ParseUserID(uuid.NewString())
	currentID, _ := domain.ParseRevisionID(uuid.NewString())
	queries := &fakePostReadQueries{
		post: &domain.Post{PostID: postID, AuthorID: authorID,
			LifecycleStatus: domain.LifecyclePublished, Visibility: domain.VisibilityPublic,
			AvailabilityStatus: domain.AvailabilityNormal, PublishedRevisionID: &currentID},
	}
	if _, err := readPublishedPost(context.Background(), queries, nil, postID, time.Second); !errors.Is(err, domain.ErrRevisionNotFound) {
		t.Fatalf("missing current revision = %v", err)
	}
	if queries.assetReadCount != 0 {
		t.Fatal("loaded ancillary data without current revision")
	}
}

func TestReadPublishedPostReturnsTextWhenMediaTimesOut(t *testing.T) {
	postID, _ := domain.ParsePostID(uuid.NewString())
	authorID, _ := domain.ParseUserID(uuid.NewString())
	revisionID, _ := domain.ParseRevisionID(uuid.NewString())
	assetID, _ := domain.ParseAssetID(uuid.NewString())
	queries := &fakePostReadQueries{
		post: &domain.Post{PostID: postID, AuthorID: authorID,
			LifecycleStatus: domain.LifecyclePublished, Visibility: domain.VisibilityPublic,
			AvailabilityStatus: domain.AvailabilityNormal, PublishedRevisionID: &revisionID},
		revision: &domain.PostRevision{PostId: postID, RevisionId: revisionID,
			Cover:                 &domain.Cover{AssetID: assetID},
			Document:              `{"schemaVersion":1,"blocks":[{"blockId":"b1","blockType":"paragraph","sortOrder":1,"text":"hello","assetIds":["` + assetID.String() + `"]}]}`,
			DocumentSchemaVersion: 1},
		waitForMediaTimeout: true,
	}
	data, err := readPublishedPost(context.Background(), queries, nil, postID, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if data.Document.Blocks[0].Text != "hello" || len(data.MediaAssets) != 1 ||
		data.MediaAssets[0].AssetId != assetID.String() || data.MediaAssets[0].Status != "unavailable" {
		t.Fatalf("degraded post response = %+v", data)
	}
}

func TestPublishedPostData(t *testing.T) {
	postID, _ := domain.ParsePostID(uuid.NewString())
	authorID, _ := domain.ParseUserID(uuid.NewString())
	revisionID, _ := domain.ParseRevisionID(uuid.NewString())
	coverID, _ := domain.ParseAssetID(uuid.NewString())
	publishedAt := time.Date(2026, 10, 9, 12, 30, 0, 0, time.UTC)
	post := &domain.Post{PostID: postID, AuthorID: authorID, Version: 5}
	revision := &domain.PostRevision{
		RevisionId: revisionID, PostId: postID, RevisionNumber: 2,
		Title: "title", Summary: "summary", Cover: &domain.Cover{AssetID: coverID},
		Document:              `{"schemaVersion":1,"blocks":[{"blockId":"b1","blockType":"paragraph","sortOrder":1,"text":"hello"}]}`,
		DocumentSchemaVersion: 1, PublishedAt: publishedAt,
	}
	document, err := decodePublishedDocument(revision)
	if err != nil {
		t.Fatal(err)
	}
	ids, err := referencedAssetIDs(revision.Cover, document)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != coverID {
		t.Fatalf("deduplicated asset IDs = %v", ids)
	}
	data, err := publishedPostData(post, revision, document, ids,
		[]domain.PublishedAsset{{ID: coverID, MimeType: "image/jpeg", Width: 640, Height: 480}},
		[]domain.PublishedTag{{ID: "tag", Name: "travel"}},
		&domain.PostCounts{Likes: 3, Favorites: 2, Comments: 1}, false)
	if err != nil {
		t.Fatal(err)
	}
	if data.RevisionId != revisionID.String() || data.RevisionNumber != 2 || data.PostVersion != 5 ||
		len(data.Document.Blocks) != 1 || data.Document.Blocks[0].Text != "hello" ||
		!data.HasCover || data.Cover.Width != 640 || len(data.MediaAssets) != 1 ||
		len(data.Tags) != 1 || data.LikeCount != 3 || data.PublishedAt != publishedAt.Format(time.RFC3339Nano) {
		t.Fatalf("published response = %+v", data)
	}
}

func TestPublishedPostDataRejectsInvalidDocument(t *testing.T) {
	revisionID, _ := domain.ParseRevisionID(uuid.NewString())
	_, err := decodePublishedDocument(&domain.PostRevision{
		RevisionId: revisionID, Document: `{"schemaVersion":2,"blocks":[]}`, DocumentSchemaVersion: 1,
	})
	if err == nil {
		t.Fatal("mismatched document schema accepted")
	}
}
