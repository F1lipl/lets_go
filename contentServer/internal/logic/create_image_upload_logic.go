// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"

	"contentserver/internal/svc"
	"contentserver/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateImageUploadLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateImageUploadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateImageUploadLogic {
	return &CreateImageUploadLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateImageUploadLogic) CreateImageUpload(req *types.CreateImageUploadRequest) (resp *types.CreateImageUploadResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
