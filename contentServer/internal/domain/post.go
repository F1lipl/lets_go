package domain

import (
	"time"
)

type LifecycleStatus uint8

const (
	LifecycleDraft LifecycleStatus = iota + 1
	LifecyclePublishing
	LifecyclePublished
	LifecycleDeleted
)

func (status LifecycleStatus) Valid() bool {
	return status >= LifecycleDraft && status <= LifecycleDeleted
}

type Visibility uint8

const (
	VisibilityPublic Visibility = iota + 1
	VisibilityFollowers
	VisibilityPrivate
)

func (visibility Visibility) Valid() bool {
	return visibility >= VisibilityPublic && visibility <= VisibilityPrivate
}

type Post struct {
	postID   PostID
	authorID UserID

	lifecycleStatus LifecycleStatus
	visibility      Visibility

	publishedRevisionID *RevisionID
	revisionSequence    uint64
	version             uint64

	firstPublishedAt *time.Time
	lastPublishedAt  *time.Time

	createdAt time.Time
	updatedAt time.Time
	deletedAt *time.Time
}

func NewPost(authorID UserID, visibility Visibility) (*Post, error) {
	postId, err := NewPostID()
	if err != nil {
		return nil, err
	}
	if visibility.Valid() {
		return nil, ErrInvalidVisibility
	}
	if authorID.IsZero() {
		return nil, ErrInvalidUserID
	}
	now := time.Now()
	return &Post{
		postID:              postId,
		authorID:            authorID,
		lifecycleStatus:     LifecycleDraft,
		visibility:          visibility,
		publishedRevisionID: nil,
		revisionSequence:    0,
		version:             1,
		firstPublishedAt:    nil,
		lastPublishedAt:     nil,
		createdAt:           now,
		updatedAt:           now,
		deletedAt:           nil,
	}, nil
}

func (post *Post) GetID() PostID {
	return post.postID
}
func (post *Post) GetAuthorID() UserID {
	return post.authorID
}
