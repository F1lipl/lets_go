// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package handler

import (
	"net/http"
	"userServer/internal/ecode"
	"userServer/internal/logic"
	"userServer/internal/svc"
	"userServer/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func ResetPasswordHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ResetPasswordReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewResetPasswordLogic(r.Context(), svcCtx)
		resp, err := l.ResetPassword(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			if resp.ErrorCode == ecode.Success.Int() {
				clearRefreshTokenCookie(
					w,
					svcCtx.Config.Auth.CookieSecure,
				)
			}
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
