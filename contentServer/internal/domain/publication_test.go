package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPublishSnapshotAndRules(t *testing.T) {
	author, _ := ParseUserID(uuid.NewString())
	post, _ := NewPost(author, VisibilityPublic)
	asset, _ := NewAssetID()
	draft := &PostDraft{PostID: post.PostID, Version: 3, Cover: &Cover{AssetID: asset}, Document: `{"blocks":[]}`, DocumentSchemaVersion: 1}
	id, _ := NewRevisionID()
	now := time.Now()
	revision, err := post.Publish(author, draft, id, now)
	if err != nil {
		t.Fatal(err)
	}
	if post.Version != 2 || post.RevisionSequence != 1 || revision.SourceDraftVersion != 3 || post.LifecycleStatus != LifecyclePublished {
		t.Fatal("incorrect publication state")
	}
	draft.Cover.FocusX = 0.75
	draft.Document = "changed"
	if revision.Cover.FocusX != 0 || revision.Document != "{\"blocks\":[]}" {
		t.Fatal("snapshot changed with draft")
	}
	first := *post.FirstPublishedAt
	secondID, _ := NewRevisionID()
	draft.Document = "{\"blocks\":[]}"
	second, err := post.Publish(author, draft, secondID, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if second.RevisionNumber != 2 || !post.FirstPublishedAt.Equal(first) {
		t.Fatal("republish did not preserve first publication")
	}
	post.LifecycleStatus = LifecycleDeleted
	if _, err := post.Publish(author, draft, secondID, now); !errors.Is(err, ErrPostAlreadyDeleted) {
		t.Fatalf("deleted post: %v", err)
	}
}
