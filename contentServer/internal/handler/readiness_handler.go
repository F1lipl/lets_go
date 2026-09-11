// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package handler

import (
	"net/http"

	"contentserver/internal/logic"
	"contentserver/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ReadinessHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewReadinessLogic(r.Context(), svcCtx)
		resp, err := l.Readiness()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
