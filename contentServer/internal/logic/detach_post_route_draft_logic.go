// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"

	"contentserver/internal/svc"
	"contentserver/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DetachPostRouteDraftLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDetachPostRouteDraftLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DetachPostRouteDraftLogic {
	return &DetachPostRouteDraftLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DetachPostRouteDraftLogic) DetachPostRouteDraft(req *types.DetachPostRouteDraftRequest) (resp *types.DetachPostRouteDraftResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
