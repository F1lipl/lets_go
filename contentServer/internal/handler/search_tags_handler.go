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

func SearchTagsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SearchTagsRequest
		if err := httpx.Parse(r, &req); err != nil {
			writeInvalidRequest(r.Context(), w, err)
			return
		}

		l := logic.NewSearchTagsLogic(r.Context(), svcCtx)
		data, err := l.SearchTags(&req)
		writeBusinessResponse(r.Context(), w, data, err)
	}
}
