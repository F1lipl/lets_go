// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"

	"contentserver/internal/svc"
	"contentserver/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPostDraftLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetPostDraftLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPostDraftLogic {
	return &GetPostDraftLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPostDraftLogic) GetPostDraft(req *types.GetPostDraftRequest) (resp *types.GetPostDraftResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
