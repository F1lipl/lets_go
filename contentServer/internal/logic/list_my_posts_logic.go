// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"

	"contentserver/internal/svc"
	"contentserver/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListMyPostsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListMyPostsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMyPostsLogic {
	return &ListMyPostsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListMyPostsLogic) ListMyPosts(req *types.ListMyPostsRequest) (resp *types.ListMyPostsResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
