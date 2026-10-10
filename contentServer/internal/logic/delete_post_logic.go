// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"
	"time"

	"contentserver/internal/domain"
	"contentserver/internal/ecode"
	"contentserver/internal/identity"
	"contentserver/internal/respository"
	"contentserver/internal/svc"
	"contentserver/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeletePostLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeletePostLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeletePostLogic {
	return &DeletePostLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeletePostLogic) DeletePost(req *types.DeletePostRequest) (*types.DeletePostData, error) {
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
	if req.ExpectedPostVersion == 0 {
		return nil, domain.ErrInvalidVersion
	}
	deleted, err := respository.NewPostRepository().DeletePost(l.ctx, l.svcCtx.DB, postID,
		func(post *domain.Post) error {
			return post.Delete(actorID, req.ExpectedPostVersion, time.Now().UTC().Truncate(time.Millisecond))
		})
	if err != nil {
		return nil, err
	}
	logCommittedPostOperation(l.ctx, "delete_post", deleted.PostID.String(), actorID.String(),
		logx.Field("postVersion", deleted.PostVersion))
	return &types.DeletePostData{
		PostId: deleted.PostID.String(), Status: "deleted", PostVersion: deleted.PostVersion,
		DeletedAt: deleted.DeletedAt.Format(time.RFC3339Nano),
	}, nil
}
