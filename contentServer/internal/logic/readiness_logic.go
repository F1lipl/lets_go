// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"
	"fmt"
	"time"

	"contentserver/internal/svc"
	"contentserver/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReadinessLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewReadinessLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadinessLogic {
	return &ReadinessLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ReadinessLogic) Readiness() (resp *types.ReadinessResponse, err error) {
	ctx, cancel := context.WithTimeout(l.ctx, 2*time.Second)
	defer cancel()

	var probe int
	if err := l.svcCtx.DB.QueryRowCtx(ctx, &probe, "SELECT 1"); err != nil {
		return nil, fmt.Errorf("database readiness check failed: %w", err)
	}

	return &types.ReadinessResponse{
		Status:   "ready",
		Database: "ready",
	}, nil
}
