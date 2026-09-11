// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"

	"contentserver/internal/svc"
	"contentserver/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchTagsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSearchTagsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchTagsLogic {
	return &SearchTagsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SearchTagsLogic) SearchTags(req *types.SearchTagsRequest) (resp *types.SearchTagsResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
