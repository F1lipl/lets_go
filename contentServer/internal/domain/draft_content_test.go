package domain

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewDraftContentDerivesMetadata(t *testing.T) {
	assetID, _ := ParseAssetID(uuid.NewString())
	document := fmt.Sprintf(`{"schemaVersion":1,"blocks":[{"blockId":"a","blockType":"paragraph","sortOrder":0,"title":"首日","text":"出发"},{"blockId":"b","parentBlockId":"a","blockType":"image","sortOrder":1,"assetIds":[%q]}]}`, assetID.String())
	content, err := NewDraftContent("标题", "摘要", &Cover{AssetID: assetID, FocusX: 0.5}, document, []string{"#旅行", "旅行"})
	if err != nil {
		t.Fatal(err)
	}
	if content.BlockCount != 2 || content.ImageCount != 1 || len(content.AssetRefs) != 1 ||
		content.AssetRefs[0].BlockID != "b" || content.AssetRefs[0].AssetID != assetID ||
		content.PlainText != "首日\n出发\n" || len(content.TagNames) != 1 || content.TagNames[0] != "旅行" {
		t.Fatalf("derived content = %+v", content)
	}
	postID, _ := NewPostID()
	draft, err := CreateNewPostDraft(postID, content, time.Now())
	if err != nil || draft.Version != 1 || draft.ImageCount != 1 || draft.Document != document {
		t.Fatalf("draft = %+v, err = %v", draft, err)
	}
	content.Cover.FocusX = 0.9
	if draft.Cover.FocusX != 0.5 {
		t.Fatal("draft cover aliases input")
	}
}

func TestNewDraftContentRejectsBrokenDocument(t *testing.T) {
	for _, document := range []string{
		`{"schemaVersion":0,"blocks":[]}`,
		`{"schemaVersion":1,"blocks":[{"blockId":"a","blockType":"wrong"}]}`,
		`{"schemaVersion":1,"blocks":[{"blockId":"a","blockType":"paragraph"},{"blockId":"a","blockType":"paragraph"}]}`,
		`{"schemaVersion":1,"blocks":[{"blockId":"a","parentBlockId":"b","blockType":"paragraph"},{"blockId":"b","parentBlockId":"a","blockType":"paragraph"}]}`,
		`{"schemaVersion":1,"blocks":[{"blockId":"a","parentBlockId":"missing","blockType":"paragraph"}]}`,
		`{"schemaVersion":1,"blocks":[{"blockId":"a","blockType":"image","assetIds":["bad"]}]}`,
	} {
		if _, err := NewDraftContent("", "", nil, document, nil); !errors.Is(err, ErrDraftContentInvalid) {
			t.Fatalf("document %s: %v", document, err)
		}
	}
	if _, err := NewDraftContent(strings.Repeat("a", 121), "", nil, `{"schemaVersion":1,"blocks":[]}`, nil); !errors.Is(err, ErrDraftContentInvalid) {
		t.Fatalf("long title: %v", err)
	}
}

func TestPostReplaceDraftRules(t *testing.T) {
	authorID, _ := ParseUserID(uuid.NewString())
	otherID, _ := ParseUserID(uuid.NewString())
	post, _ := NewPost(authorID, VisibilityPublic)
	content, _ := NewDraftContent("title", "", nil, `{"schemaVersion":1,"blocks":[]}`, nil)
	now := time.Now()
	if _, err := post.ReplaceDraft(otherID, 1, 1, content, now); !errors.Is(err, ErrPostOperationNotAllowed) {
		t.Fatalf("other author: %v", err)
	}
	if _, err := post.ReplaceDraft(authorID, 2, 1, content, now); !errors.Is(err, ErrDraftVersionConflict) {
		t.Fatalf("stale version: %v", err)
	}
	draft, err := post.ReplaceDraft(authorID, 1, 1, content, now)
	if err != nil || draft.Version != 2 || post.Version != 1 {
		t.Fatalf("replacement = %+v, post = %+v, err = %v", draft, post, err)
	}
	post.LifecycleStatus = LifecycleDeleted
	if _, err := post.ReplaceDraft(authorID, 1, 1, content, now); !errors.Is(err, ErrPostAlreadyDeleted) {
		t.Fatalf("deleted post: %v", err)
	}
}

func TestDraftAssetOwnershipAndState(t *testing.T) {
	ownerID, _ := ParseUserID(uuid.NewString())
	otherID, _ := ParseUserID(uuid.NewString())
	assetID, _ := ParseAssetID(uuid.NewString())
	draft := &PostDraft{Cover: &Cover{AssetID: assetID}, AssetRefs: []DraftAssetRef{{AssetID: assetID}}}
	if len(draft.ReferencedAssetIDs()) != 1 {
		t.Fatal("duplicate cover and body asset was not collapsed")
	}
	if err := draft.ValidateAssetStates(ownerID, []DraftMediaAssetState{{ID: assetID, OwnerID: ownerID, Status: 2}}); err != nil {
		t.Fatalf("processing image in draft: %v", err)
	}
	if err := draft.ValidateAssetStates(ownerID, nil); !errors.Is(err, ErrMediaAssetNotFound) {
		t.Fatalf("missing image: %v", err)
	}
	if err := draft.ValidateAssetStates(ownerID, []DraftMediaAssetState{{ID: assetID, OwnerID: otherID, Status: 3}}); !errors.Is(err, ErrMediaAssetNotFound) {
		t.Fatalf("other user's image: %v", err)
	}
	if err := draft.ValidateAssetStates(ownerID, []DraftMediaAssetState{{ID: assetID, OwnerID: ownerID, Status: 4}}); !errors.Is(err, ErrMediaAssetNotReady) {
		t.Fatalf("failed image: %v", err)
	}
}
