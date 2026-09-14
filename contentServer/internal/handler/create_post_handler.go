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
			writeInvalidRequest(r.Context(), w, err)
			return
		}

		l := logic.NewCreatePostLogic(r.Context(), svcCtx)
		data, err := l.CreatePost(&req)
		writeBusinessResponse(r.Context(), w, data, err)
	}
}
