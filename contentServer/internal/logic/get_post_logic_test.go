package logic

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"contentserver/internal/domain"
	"contentserver/internal/respository"
	"contentserver/internal/types"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type fakePostReadQueries struct {
	post           *domain.Post
	revision       *domain.PostRevision
	postReads      int
	revisionReads  int
	tagReads       int
	countReads     int
	tags           []domain.PublishedTag
	counts         *domain.PostCounts
	tagErr         error
	countErr       error
	waitForTimeout bool
}

var _ respository.PostReadQueries = (*fakePostReadQueries)(nil)

func (f *fakePostReadQueries) FindPost(context.Context, sqlx.SqlConn, domain.PostID) (*domain.Post, error) {
	f.postReads++
	return f.post, nil
}
func (f *fakePostReadQueries) FindRevision(context.Context, sqlx.SqlConn, domain.RevisionID) (*domain.PostRevision, error) {
	f.revisionReads++
	if f.revision == nil {
		return nil, domain.ErrRevisionNotFound
	}
	return f.revision, nil
}
func (f *fakePostReadQueries) FindTags(ctx context.Context, _ sqlx.SqlConn, _ domain.RevisionID) ([]domain.PublishedTag, error) {
	f.tagReads++
	if f.waitForTimeout {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return f.tags, f.tagErr
}
func (f *fakePostReadQueries) FindCounts(context.Context, sqlx.SqlConn, domain.PostID) (*domain.PostCounts, error) {
	f.countReads++
	return f.counts, f.countErr
}

func publishedFixture() (domain.PostID, domain.RevisionID, *fakePostReadQueries) {
	postID, _ := domain.ParsePostID(uuid.NewString())
	authorID, _ := domain.ParseUserID(uuid.NewString())
	revisionID, _ := domain.ParseRevisionID(uuid.NewString())
	return postID, revisionID, &fakePostReadQueries{
		post: &domain.Post{PostID: postID, AuthorID: authorID,
			LifecycleStatus: domain.LifecyclePublished, Visibility: domain.VisibilityPublic,
			AvailabilityStatus: domain.AvailabilityNormal, PublishedRevisionID: &revisionID, Version: 3},
		revision: &domain.PostRevision{PostId: postID, RevisionId: revisionID,
			Document:              `{"schemaVersion":1,"blocks":[{"blockId":"b1","text":"hello"}]}`,
			DocumentSchemaVersion: 1, PublishedAt: time.Now().UTC()},
	}
}

func TestReadPublishedPostUsesCurrentRevision(t *testing.T) {
	postID, revisionID, queries := publishedFixture()
	queries.tags = []domain.PublishedTag{{ID: "tag", Name: "travel"}}
	queries.counts = &domain.PostCounts{Likes: 3, Favorites: 2, Comments: 1}
	data, err := readPublishedPost(context.Background(), queries, nil, postID, time.Second)
	if err != nil || queries.postReads != 1 || queries.revisionReads != 1 || data.RevisionId != revisionID.String() ||
		data.TagsStatus != "ready" || data.StatsStatus != "ready" || data.LikeCount != 3 || len(data.Tags) != 1 {
		t.Fatalf("read = %+v, queries = %+v, err = %v", data, queries, err)
	}
	wire, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(wire, &body); err != nil {
		t.Fatal(err)
	}
	if string(body["document"]) != queries.revision.Document {
		t.Fatalf("document was not embedded as JSON: %s", wire)
	}
	queries.post.Visibility = domain.VisibilityPrivate
	if _, err := readPublishedPost(context.Background(), queries, nil, postID, time.Second); !errors.Is(err, domain.ErrPostNotVisible) {
		t.Fatalf("private post = %v", err)
	}
	if queries.revisionReads != 1 || queries.tagReads != 1 {
		t.Fatal("loaded revision or supplements for a non-public post")
	}
}

func TestReadPublishedPostRejectsMismatchedRevision(t *testing.T) {
	postID, _, queries := publishedFixture()
	queries.revision.RevisionId, _ = domain.ParseRevisionID(uuid.NewString())
	if _, err := readPublishedPost(context.Background(), queries, nil, postID, time.Second); err == nil {
		t.Fatal("mismatched current revision accepted")
	}
	if queries.tagReads != 0 || queries.countReads != 0 {
		t.Fatal("loaded supplements for mismatched revision")
	}
}

func TestReadPublishedPostRequiresCurrentRevision(t *testing.T) {
	postID, _, queries := publishedFixture()
	queries.revision = nil
	if _, err := readPublishedPost(context.Background(), queries, nil, postID, time.Second); !errors.Is(err, domain.ErrRevisionNotFound) {
		t.Fatalf("missing current revision = %v", err)
	}
}

func TestReadPublishedPostReturnsTextWhenSupplementsFail(t *testing.T) {
	postID, _, queries := publishedFixture()
	assetID, _ := domain.ParseAssetID(uuid.NewString())
	queries.revision.Cover = &domain.Cover{AssetID: assetID}
	queries.tagErr = errors.New("unavailable")
	queries.countErr = errors.New("unavailable")
	data, err := readPublishedPost(context.Background(), queries, nil, postID, time.Second)
	if err != nil || data.TagsStatus != "unavailable" || data.StatsStatus != "unavailable" ||
		data.Cover.AssetId != assetID.String() || data.Document == nil {
		t.Fatalf("degraded post response = %+v, err = %v", data, err)
	}
}

func TestReadPublishedPostBoundsSupplementLatency(t *testing.T) {
	postID, _, queries := publishedFixture()
	queries.waitForTimeout = true
	data, err := readPublishedPost(context.Background(), queries, nil, postID, time.Millisecond)
	if err != nil || data.TagsStatus != "unavailable" || data.StatsStatus != "unavailable" || queries.countReads != 0 {
		t.Fatalf("timed out supplement = %+v, queries = %+v, err = %v", data, queries, err)
	}
}

func TestGeneratedGetPostDataCanCarryRawDocument(t *testing.T) {
	view := &types.GetPostData{Document: json.RawMessage(`{"schemaVersion":1,"blocks":[]}`)}
	wire, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(wire, &body); err != nil {
		t.Fatal(err)
	}
	if string(body["document"]) != `{"schemaVersion":1,"blocks":[]}` {
		t.Fatalf("document was not embedded as JSON: %s", wire)
	}
}
