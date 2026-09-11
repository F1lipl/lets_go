// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"

	"contentserver/internal/svc"
	"contentserver/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteMediaAssetLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteMediaAssetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteMediaAssetLogic {
	return &DeleteMediaAssetLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteMediaAssetLogic) DeleteMediaAsset(req *types.DeleteMediaAssetRequest) (resp *types.DeleteMediaAssetResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
