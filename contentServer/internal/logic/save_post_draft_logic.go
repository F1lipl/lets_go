// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"

	"contentserver/internal/svc"
	"contentserver/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SavePostDraftLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSavePostDraftLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SavePostDraftLogic {
	return &SavePostDraftLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SavePostDraftLogic) SavePostDraft(req *types.SavePostDraftRequest) (resp *types.SavePostDraftResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
