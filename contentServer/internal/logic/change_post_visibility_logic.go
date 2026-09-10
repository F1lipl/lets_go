// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"

	"contentserver/internal/svc"
	"contentserver/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ChangePostVisibilityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewChangePostVisibilityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChangePostVisibilityLogic {
	return &ChangePostVisibilityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ChangePostVisibilityLogic) ChangePostVisibility(req *types.ChangePostVisibilityRequest) (resp *types.ChangePostVisibilityResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
