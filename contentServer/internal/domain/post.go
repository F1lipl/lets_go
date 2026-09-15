package domain

import (
	"time"
)

type LifecycleStatus uint64

const (
	LifecycleDraft LifecycleStatus = iota + 1
	LifecyclePublishing
	LifecyclePublished
	LifecycleDeleted
)

func (status LifecycleStatus) Valid() bool {
	return status >= LifecycleDraft && status <= LifecycleDeleted
}

type Visibility uint64

const (
	VisibilityPublic Visibility = iota + 1
	VisibilityFollowers
	VisibilityPrivate
)

const (
	AvailabilityNormal  = 1 // 正常
	AvailabilityPending = 2 // 等待处理
	AvailabilityHidden  = 3 // 停止展示
)

func (visibility Visibility) Valid() bool {
	return visibility >= VisibilityPublic && visibility <= VisibilityPrivate
}

type Post struct {
	PostID   PostID
	AuthorID UserID

	LifecycleStatus LifecycleStatus
	Visibility      Visibility

	PublishedRevisionID *RevisionID
	RevisionSequence    uint64
	Version             uint64

	FirstPublishedAt *time.Time
	LastPublishedAt  *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

func NewPost(authorID UserID, visibility Visibility) (*Post, error) {
	postId, err := NewPostID()
	if err != nil {
		return nil, err
	}
	if !visibility.Valid() {
		return nil, ErrInvalidVisibility
	}
	if authorID.IsZero() {
		return nil, ErrInvalidUserID
	}
	now := time.Now()
	return &Post{
		PostID:              postId,
		AuthorID:            authorID,
		LifecycleStatus:     LifecycleDraft,
		Visibility:          visibility,
		PublishedRevisionID: nil,
		RevisionSequence:    0,
		Version:             1,
		FirstPublishedAt:    nil,
		LastPublishedAt:     nil,
		CreatedAt:           now,
		UpdatedAt:           now,
		DeletedAt:           nil,
	}, nil
}
