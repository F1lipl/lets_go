package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPostCurrentPublicRevision(t *testing.T) {
	postID, _ := ParsePostID(uuid.NewString())
	authorID, _ := ParseUserID(uuid.NewString())
	currentID, _ := ParseRevisionID(uuid.NewString())
	base := Post{
		PostID: postID, AuthorID: authorID, LifecycleStatus: LifecyclePublished,
		Visibility: VisibilityPublic, AvailabilityStatus: AvailabilityNormal,
		PublishedRevisionID: &currentID, Version: 3,
	}
	got, err := base.CurrentPublicRevision()
	if err != nil || got != currentID {
		t.Fatalf("CurrentPublicRevision() = %v, %v", got, err)
	}
	private := base
	private.Visibility = VisibilityPrivate
	if _, err := private.CurrentPublicRevision(); !errors.Is(err, ErrPostNotVisible) {
		t.Fatalf("private post: %v", err)
	}
	deleted := base
	deleted.LifecycleStatus = LifecycleDeleted
	now := time.Now()
	deleted.DeletedAt = &now
	if _, err := deleted.CurrentPublicRevision(); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("deleted post: %v", err)
	}
	draft := base
	draft.LifecycleStatus = LifecycleDraft
	if _, err := draft.CurrentPublicRevision(); !errors.Is(err, ErrPostNotPublished) {
		t.Fatalf("draft post: %v", err)
	}
}
