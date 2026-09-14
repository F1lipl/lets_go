// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package handler

import (
	"net/http"

	"contentserver/internal/logic"
	"contentserver/internal/svc"
	"contentserver/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ReadinessHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewReadinessLogic(r.Context(), svcCtx)
		resp, err := l.Readiness()
		if err != nil {
			logx.WithContext(r.Context()).Errorw("readiness check failed", logx.Field("detail", err.Error()))
			httpx.WriteJsonCtx(r.Context(), w, http.StatusServiceUnavailable, &types.ReadinessResponse{
				Status:   "not_ready",
				Database: "unavailable",
			})
			return
		}

		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
