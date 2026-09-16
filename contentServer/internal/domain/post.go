package domain

import (
	"context"
	"errors"
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
	PublishPost(ctx context.Context, postID PostID, authorID UserID, expectedVersion uint64) error
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

	postRepository PostRepositoryInterface
}

func NewPost(authorID UserID, visibility Visibility, repositoryInterface PostRepositoryInterface) (*Post, error) {
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
		postRepository:      repositoryInterface,
	}, nil
}

func (post *Post) PublishPost(ctx context.Context, conn sqlx.SqlConn, expectedVersion uint64, expectedDraftVersion uint64) error {
	revisionId, err := NewRevisionID()
	if err != nil {
		return err
	}
	postDraft, err := GetPostDraft(post.PostID, ctx, conn)
	if err != nil {
		return err
	}
	if postDraft.Version != expectedDraftVersion {
		return errors.New("draft version mismatch")
	}
	postRevision := CreateNewPostRevision(revisionId, post.PostID, post.RevisionSequence, postDraft.Title, postDraft.Summary, postDraft.Cover, postDraft.Document, time.Now())
	post.postRepository
}
