// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package handler

import (
	"net/http"

	"contentserver/internal/logic"
	"contentserver/internal/svc"
	"contentserver/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetPostHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetPostRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewGetPostLogic(r.Context(), svcCtx)
		resp, err := l.GetPost(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
