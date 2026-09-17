// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"contentserver/internal/domain"
	"contentserver/internal/respository"
	"context"
	"time"

	"contentserver/internal/ecode"
	"contentserver/internal/identity"
	"contentserver/internal/svc"
	"contentserver/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PublishPostLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPublishPostLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishPostLogic {
	return &PublishPostLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PublishPostLogic) PublishPost(req *types.PublishPostRequest) (*types.PublishPostData, error) {
	if req == nil {
		return nil, ecode.New(ecode.InvalidRequest)
	}
	currentUser, err := identity.FromContext(l.ctx)
	if err != nil {
		return nil, ecode.Wrap(ecode.RequestIdentityInvalid, err)
	}
	actorID, err := domain.ParseUserID(currentUser.UserID)
	if err != nil {
		return nil, err
	}
	postID, err := domain.ParsePostID(req.PostId)
	if err != nil {
		return nil, err
	}
	if req.ExpectedPostVersion == 0 || req.ExpectedDraftVersion == 0 {
		return nil, domain.ErrInvalidVersion
	}
	revisionID, err := domain.NewRevisionID()
	if err != nil {
		return nil, err
	}
	repo := respository.NewPostRepository()
	result, err := repo.PublishPost(l.ctx, l.svcCtx.DB, postID,
		func(post *domain.Post, draft *domain.PostDraft) (*domain.PostRevision, error) {
			if post.AuthorID != actorID {
				return nil, domain.ErrPostOperationNotAllowed
			}
			if post.Version != req.ExpectedPostVersion {
				return nil, domain.ErrPostVersionConflict
			}
			if draft.Version != req.ExpectedDraftVersion {
				return nil, domain.ErrDraftVersionConflict
			}
			return post.Publish(actorID, draft, revisionID, time.Now().UTC().Truncate(time.Millisecond))
		})
	if err != nil {
		return nil, err
	}
	return &types.PublishPostData{
		PostId: result.PostID.String(), RevisionId: result.RevisionID.String(),
		RevisionNumber: result.RevisionNumber, PostVersion: result.PostVersion,
		Status: "published", PublishedAt: result.PublishedAt.Format(time.RFC3339Nano),
	}, nil
}
