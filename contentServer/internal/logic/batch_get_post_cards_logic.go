// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"

	"contentserver/internal/svc"
	"contentserver/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchGetPostCardsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBatchGetPostCardsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchGetPostCardsLogic {
	return &BatchGetPostCardsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BatchGetPostCardsLogic) BatchGetPostCards(req *types.BatchGetPostCardsRequest) (resp *types.BatchGetPostCardsResponse, err error) {
	
	return
}
