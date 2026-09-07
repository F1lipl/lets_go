// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package handler

import (
	"net/http"
	"userServer/internal/ecode"

	"userServer/internal/logic"
	"userServer/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func LogoutHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewLogoutLogic(r.Context(), svcCtx)
		resp, err := l.Logout()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			if resp.ErrorCode == ecode.Success.Int() {
				clearRefreshTokenCookie(w, svcCtx.Config.Auth.CookieSecure)
			}
			httpx.OkJsonCtx(r.Context(), w, resp)
		}

	}
}
