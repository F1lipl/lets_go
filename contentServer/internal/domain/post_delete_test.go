package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPostDeleteRules(t *testing.T) {
	author, _ := ParseUserID(uuid.NewString())
	other, _ := ParseUserID(uuid.NewString())
	now := time.Now().UTC().Truncate(time.Millisecond)

	for _, tc := range []struct {
		name     string
		prepare  func(*Post)
		actor    UserID
		expected uint64
		want     error
	}{
		{name: "different author", actor: other, expected: 1, want: ErrPostOperationNotAllowed},
		{name: "missing version", actor: author, expected: 0, want: ErrInvalidVersion},
		{name: "stale version", actor: author, expected: 2, want: ErrPostVersionConflict},
		{name: "already deleted", actor: author, expected: 1, prepare: func(p *Post) { p.LifecycleStatus = LifecycleDeleted }, want: ErrPostAlreadyDeleted},
		{name: "publishing", actor: author, expected: 1, prepare: func(p *Post) { p.LifecycleStatus = LifecyclePublishing }, want: ErrPostOperationNotAllowed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			post, err := NewPost(author, VisibilityPublic)
			if err != nil {
				t.Fatal(err)
			}
			if tc.prepare != nil {
				tc.prepare(post)
			}
			beforeVersion := post.Version
			beforeStatus := post.LifecycleStatus
			if err := post.Delete(tc.actor, tc.expected, now); !errors.Is(err, tc.want) {
				t.Fatalf("Delete() = %v, want %v", err, tc.want)
			}
			if post.Version != beforeVersion || post.LifecycleStatus != beforeStatus || post.DeletedAt != nil {
				t.Fatalf("failed deletion mutated post: %+v", post)
			}
		})
	}

	for _, status := range []LifecycleStatus{LifecycleDraft, LifecyclePublished} {
		post, err := NewPost(author, VisibilityPublic)
		if err != nil {
			t.Fatal(err)
		}
		post.LifecycleStatus = status
		if err := post.Delete(author, 1, now); err != nil {
			t.Fatalf("status %v: %v", status, err)
		}
		if post.Version != 2 || post.LifecycleStatus != LifecycleDeleted || post.DeletedAt == nil || !post.DeletedAt.Equal(now) || !post.UpdatedAt.Equal(now) {
			t.Fatalf("status %v: incorrect deleted state %+v", status, post)
		}
		if err := post.Delete(author, 2, now); !errors.Is(err, ErrPostAlreadyDeleted) {
			t.Fatalf("repeated delete = %v", err)
		}
	}
	var missing *Post
	if err := missing.Delete(author, 1, now); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("nil post delete = %v", err)
	}
}
