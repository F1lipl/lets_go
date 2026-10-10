package respository

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"contentserver/internal/domain"
	"contentserver/internal/model"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type PostRepository struct {
}

func NewPostRepository() *PostRepository {
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
	if post.PostID.IsZero() || post.AuthorID.IsZero() || post.Version != 1 ||
		post.LifecycleStatus != domain.LifecycleDraft || postDraft.Version != 1 ||
		postDraft.DocumentSchemaVersion != 1 || postDraft.ImageCount != uint64(len(postDraft.AssetRefs)) {
		return domain.ErrInvalidPost
	}
	draftData, err := draftRow(postDraft)
	if err != nil {
		return err
	}

	postRow := &model.Post{
		PostId:             post.PostID.String(),
		AuthorId:           post.AuthorID.String(),
		LifecycleStatus:    uint64(post.LifecycleStatus),
		Visibility:         uint64(post.Visibility),
		AvailabilityStatus: post.AvailabilityStatus,
		PostVersion:        post.Version,
	}

	return conn.TransactCtx(
		ctx,
		func(ctx context.Context, session sqlx.Session) error {
			txConn := sqlx.NewSqlConnFromSession(session)
			if err := validateDraftAssets(ctx, txConn, postDraft, post.AuthorID); err != nil {
				return err
			}

			postModel := model.NewPostModel(txConn)
			if _, err := postModel.Insert(ctx, postRow); err != nil {
				return err
			}

			if _, err := model.NewPostDraftModel(txConn).Insert(ctx, draftData); err != nil {
				return err
			}
			if err := replaceDraftRefs(ctx, txConn, postDraft); err != nil {
				return err
			}
			stored, err := postModel.FindOne(ctx, post.PostID.String())
			if err != nil {
				return err
			}
			post.CreatedAt = stored.CreatedAt
			return nil
		},
	)
}

// SavePostDraft locks Post before Draft, matching the publication lock order.
// The draft row and its derived asset index commit or roll back together.
func (r *PostRepository) SavePostDraft(
	ctx context.Context, conn sqlx.SqlConn, postID domain.PostID,
	apply func(*domain.Post, uint64) (*domain.PostDraft, error),
) (*domain.PostDraft, error) {
	if postID.IsZero() {
		return nil, domain.ErrInvalidPostID
	}
	if apply == nil {
		return nil, fmt.Errorf("draft update callback is nil")
	}
	var saved *domain.PostDraft
	err := conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		txConn := sqlx.NewSqlConnFromSession(session)
		postModel := model.NewPostModel(txConn)
		postRow, err := postModel.FindOneForUpdate(ctx, postID.String())
		if errors.Is(err, model.ErrNotFound) {
			return domain.ErrPostNotFound
		}
		if err != nil {
			return err
		}
		draftModel := model.NewPostDraftModel(txConn)
		currentVersion, err := draftModel.FindVersionForUpdate(ctx, postID.String())
		if errors.Is(err, model.ErrNotFound) {
			return domain.ErrDraftNotFound
		}
		if err != nil {
			return err
		}
		post, err := restorePost(postRow)
		if err != nil {
			return err
		}
		draft, err := apply(post, currentVersion)
		if err != nil {
			return err
		}
		if draft == nil || draft.PostID != postID || draft.Version != currentVersion+1 ||
			draft.DocumentSchemaVersion != 1 || draft.ImageCount != uint64(len(draft.AssetRefs)) {
			return fmt.Errorf("invalid draft update callback result")
		}
		if err := validateDraftAssets(ctx, txConn, draft, post.AuthorID); err != nil {
			return err
		}
		row, err := draftRow(draft)
		if err != nil {
			return err
		}
		if err := draftModel.UpdateWithVersion(ctx, row, currentVersion); err != nil {
			if errors.Is(err, model.ErrVersionConflict) {
				return domain.ErrDraftVersionConflict
			}
			return err
		}
		if err := replaceDraftRefs(ctx, txConn, draft); err != nil {
			return err
		}
		updatedAt, err := draftModel.FindUpdatedAt(ctx, postID.String())
		if err != nil {
			return err
		}
		draft.UpdatedAt = updatedAt
		saved = draft
		return nil
	})
	if err != nil {
		return nil, err
	}
	return saved, nil
}

func validateDraftAssets(ctx context.Context, conn sqlx.SqlConn, draft *domain.PostDraft, authorID domain.UserID) error {
	ids := draft.ReferencedAssetIDs()
	if len(ids) == 0 {
		return nil
	}
	ordered := make([]string, 0, len(ids))
	for _, id := range ids {
		ordered = append(ordered, id.String())
	}
	sort.Strings(ordered)
	rows, err := model.NewMediaAssetModel(conn).FindByIDsForShare(ctx, ordered)
	if err != nil {
		return err
	}
	states := make([]domain.DraftMediaAssetState, 0, len(rows))
	for _, row := range rows {
		id, err := domain.ParseAssetID(row.AssetId)
		if err != nil {
			return err
		}
		owner, err := domain.ParseUserID(row.OwnerId)
		if err != nil {
			return err
		}
		states = append(states, domain.DraftMediaAssetState{
			ID: id, OwnerID: owner, Status: row.Status, IsDeleted: row.DeletedAt.Valid,
		})
	}
	return draft.ValidateAssetStates(authorID, states)
}

func replaceDraftRefs(ctx context.Context, conn sqlx.SqlConn, draft *domain.PostDraft) error {
	refs := model.NewContentAssetRefModel(conn)
	if err := refs.DeleteByOwner(ctx, 1, draft.PostID.String()); err != nil {
		return err
	}
	if draft.Cover != nil {
		if _, err := refs.Insert(ctx, &model.ContentAssetRef{
			OwnerType: 1, OwnerId: draft.PostID.String(), UsageType: 1,
			AssetId: draft.Cover.AssetID.String(), SourceVersion: draft.Version,
		}); err != nil {
			return err
		}
	}
	for _, ref := range draft.AssetRefs {
		if _, err := refs.Insert(ctx, &model.ContentAssetRef{
			OwnerType: 1, OwnerId: draft.PostID.String(), UsageType: 2,
			BlockId: ref.BlockID, AssetId: ref.AssetID.String(), SortOrder: ref.SortOrder,
			SourceVersion: draft.Version,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r *PostRepository) DeletePost(
	ctx context.Context,
	conn sqlx.SqlConn,
	postID domain.PostID,
	apply func(*domain.Post) error,
) (*domain.PostDeletionResult, error) {
	if postID.IsZero() {
		return nil, domain.ErrInvalidPostID
	}
	if apply == nil {
		return nil, fmt.Errorf("post deletion callback is nil")
	}

	var deleted *domain.PostDeletionResult
	err := conn.TransactCtx(
		ctx,
		func(ctx context.Context, session sqlx.Session) error {
			txConn := sqlx.NewSqlConnFromSession(session)
			postModel := model.NewPostModel(txConn)
			current, err := postModel.FindOneForUpdate(ctx, postID.String())
			if errors.Is(err, model.ErrNotFound) {
				return domain.ErrPostNotFound
			}
			if err != nil {
				return err
			}
			post, err := restorePost(current)
			if err != nil {
				return err
			}
			if err := apply(post); err != nil {
				return err
			}
			if post.PostID != postID || post.AuthorID.String() != current.AuthorId ||
				post.Version != current.PostVersion+1 || post.LifecycleStatus != domain.LifecycleDeleted ||
				post.DeletedAt == nil {
				return fmt.Errorf("invalid post deletion callback result")
			}
			result, err := postModel.UpdateForDelete(
				ctx,
				current.AuthorId,
				postID.String(),
				current.PostVersion,
				*post.DeletedAt,
			)
			if err != nil {
				return err
			}

			affected, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if affected == 0 {
				return domain.ErrPostVersionConflict
			}

			cardModel := model.NewPostCardProjectionModel(txConn)
			if err := cardModel.Delete(ctx, postID.String()); err != nil {
				return err
			}

			deleted = &domain.PostDeletionResult{
				PostID: postID, PostVersion: post.Version, DeletedAt: *post.DeletedAt,
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	return deleted, nil
}
