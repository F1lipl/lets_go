package domain

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
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

type PostRepositoryInterface interface {
	CreatePost(ctx context.Context, conn sqlx.SqlConn, post *Post, postDraft *PostDraft) error
	DeletePost(ctx context.Context, conn sqlx.SqlConn, postID PostID, authorID UserID, expectedVersion uint64) error
	PublishPost(ctx context.Context, conn sqlx.SqlConn, postID PostID, apply func(*Post, *PostDraft) (*PostRevision, error)) (*PublicationResult, error)
}

type Post struct {
	PostID   PostID
	AuthorID UserID

	LifecycleStatus    LifecycleStatus
	Visibility         Visibility
	AvailabilityStatus uint64

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
		AvailabilityStatus:  AvailabilityNormal,
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

// Publish evaluates domain rules on request-local state, without database access.
func (post *Post) Publish(actorID UserID, draft *PostDraft, revisionID RevisionID, now time.Time) (*PostRevision, error) {
	if actorID.IsZero() {
		return nil, ErrInvalidUserID
	}
	if post.AuthorID != actorID {
		return nil, ErrPostOperationNotAllowed
	}
	if post.LifecycleStatus == LifecycleDeleted || post.DeletedAt != nil {
		return nil, ErrPostAlreadyDeleted
	}
	if post.LifecycleStatus != LifecycleDraft && post.LifecycleStatus != LifecyclePublished {
		return nil, ErrPublishNotAllowed
	}
	if !post.Visibility.Valid() || post.AvailabilityStatus != AvailabilityNormal {
		return nil, ErrPublishNotAllowed
	}
	if draft == nil {
		return nil, ErrDraftNotFound
	}
	if draft.PostID != post.PostID {
		return nil, ErrDraftPostMismatch
	}
	if err := draft.ValidateForPublish(); err != nil {
		return nil, err
	}
	if revisionID.IsZero() {
		return nil, ErrInvalidRevisionID
	}
	if post.Version == ^uint64(0) || post.RevisionSequence == ^uint64(0) {
		return nil, ErrInvalidVersion
	}
	revision := CreateNewPostRevision(revisionID, post.PostID, post.RevisionSequence+1, draft.Title, draft.Summary, draft.Cover, draft.Document, now)
	revision.SourceDraftVersion = draft.Version
	revision.DocumentSchemaVersion = draft.DocumentSchemaVersion
	revision.PlainText = draft.PlainText
	revision.BlockCount = draft.BlockCount
	revision.ImageCount = draft.ImageCount
	revision.TagNames = append([]string(nil), draft.TagNames...)
	if post.FirstPublishedAt == nil {
		post.FirstPublishedAt = &now
	}
	post.PublishedRevisionID = &revisionID
	post.RevisionSequence = revision.RevisionNumber
	post.Version++
	post.LifecycleStatus = LifecyclePublished
	post.LastPublishedAt = &now
	post.UpdatedAt = now
	return revision, nil
}

type PublicationResult struct {
	PostID         PostID
	RevisionID     RevisionID
	RevisionNumber uint64
	PostVersion    uint64
	PublishedAt    time.Time
}
