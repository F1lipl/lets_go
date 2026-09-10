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

func CreatePostHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CreatePostRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewCreatePostLogic(r.Context(), svcCtx)
		resp, err := l.CreatePost(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
