package respository

import (
	"contentserver/internal/domain"
	"contentserver/internal/model"
	"context"
	"errors"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type PostRepository struct {
}

func NewPostRepository() domain.PostRepositoryInterface {
	return &PostRepository{}
}

func (r *PostRepository) CreatePost(
	ctx context.Context,
	conn sqlx.SqlConn,
	post *domain.Post,
	postDraft *domain.PostDraft,
) error {
	if post == nil || postDraft == nil {
		return domain.ErrInvalidPost
	}

	if post.PostID != postDraft.PostID {
		return domain.ErrDraftPostMismatch
	}

	postRow := &model.Post{
		PostId:             post.PostID.String(),
		AuthorId:           post.AuthorID.String(),
		LifecycleStatus:    uint64(post.LifecycleStatus),
		Visibility:         uint64(post.Visibility),
		AvailabilityStatus: uint64(domain.AvailabilityNormal),
		PostVersion:        post.Version,
	}

	return conn.TransactCtx(
		ctx,
		func(ctx context.Context, session sqlx.Session) error {
			txConn := sqlx.NewSqlConnFromSession(session)

			postModel := model.NewPostModel(txConn)
			if _, err := postModel.Insert(ctx, postRow); err != nil {
				return err
			}

			draftRepository := newPostDraftRepository(txConn)
			if err := draftRepository.CreatePostDraft(ctx, postDraft); err != nil {
				return err
			}

			return nil
		},
	)
}

func (r *PostRepository) DeletePost(
	ctx context.Context,
	conn sqlx.SqlConn,
	postID domain.PostID,
	authorID domain.UserID,
	expectedVersion uint64,
) error {
	if postID.IsZero() {
		return domain.ErrInvalidPostID
	}
	if authorID.IsZero() {
		return domain.ErrInvalidUserID
	}
	if expectedVersion == 0 {
		return domain.ErrInvalidVersion
	}

	return conn.TransactCtx(
		ctx,
		func(ctx context.Context, session sqlx.Session) error {
			txConn := sqlx.NewSqlConnFromSession(session)

			postModel := model.NewPostModel(txConn)
			result, err := postModel.UpdateForDelete(
				ctx,
				authorID.String(),
				postID.String(),
				expectedVersion,
				time.Now(),
			)
			if err != nil {
				return err
			}

			affected, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if affected == 0 {
				current, err := postModel.FindOneForUpdate(ctx, postID.String())
				if err != nil {
					if errors.Is(err, model.ErrNotFound) {
						return domain.ErrPostNotFound
					}
					return err
				}

				return classifyDeleteFailure(current, authorID, expectedVersion)
			}

			cardModel := model.NewPostCardProjectionModel(txConn)
			if err := cardModel.Delete(ctx, postID.String()); err != nil {
				return err
			}

			// PostDeleted 事件也应在这个事务中写入。

			return nil
		},
	)
}

func (r *PostRepository) PublishPost(ctx context.Context, postID domain.PostID, authorID domain.UserID, expectedVersion uint64) error {
	return nil
}

func classifyDeleteFailure(current *model.Post, authorID domain.UserID, expectedVersion uint64) error {
	if current == nil {
		return domain.ErrPostNotFound
	}
	switch {
	case current.AuthorId != authorID.String():
		return domain.ErrPostOperationNotAllowed
	case current.LifecycleStatus == uint64(domain.LifecycleDeleted):
		return domain.ErrPostAlreadyDeleted
	case current.PostVersion != expectedVersion:
		return domain.ErrPostVersionConflict
	default:
		return domain.ErrPostOperationNotAllowed
	}
}
