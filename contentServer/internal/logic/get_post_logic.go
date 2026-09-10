// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"

	"contentserver/internal/svc"
	"contentserver/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPostLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetPostLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPostLogic {
	return &GetPostLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPostLogic) GetPost(req *types.GetPostRequest) (resp *types.GetPostResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
