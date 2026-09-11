// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"

	"contentserver/internal/svc"
	"contentserver/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CompleteImageUploadLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCompleteImageUploadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CompleteImageUploadLogic {
	return &CompleteImageUploadLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CompleteImageUploadLogic) CompleteImageUpload(req *types.CompleteImageUploadRequest) (resp *types.CompleteImageUploadResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
