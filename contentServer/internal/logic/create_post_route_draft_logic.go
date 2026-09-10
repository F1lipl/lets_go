// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"

	"contentserver/internal/svc"
	"contentserver/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreatePostRouteDraftLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreatePostRouteDraftLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreatePostRouteDraftLogic {
	return &CreatePostRouteDraftLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreatePostRouteDraftLogic) CreatePostRouteDraft(req *types.CreatePostRouteDraftRequest) (resp *types.CreatePostRouteDraftResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
