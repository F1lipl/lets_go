// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"

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

func (l *PublishPostLogic) PublishPost(req *types.PublishPostRequest) (data *types.PublishPostData, err error) {
	// todo: add your logic here and delete this line
	if _, err := identity.FromContext(l.ctx); err != nil {
		return nil, ecode.Wrap(ecode.RequestIdentityInvalid, err)
	}

	return
}
