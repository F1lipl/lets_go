// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package handler

import (
	"errors"
	"net/http"

	"userServer/internal/logic"
	"userServer/internal/svc"
	"userServer/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func LoginByPhoneHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LoginByPhoneReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewLoginByPhoneLogic(r.Context(), svcCtx)
		result, err := l.LoginByPhone(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		if result == nil || result.Response == nil {
			httpx.ErrorCtx(
				r.Context(),
				w,
				errors.New("login result is nil"),
			)
			return
		}

		if result.RefreshToken != "" {
			writeRefreshTokenCookie(
				w,
				result.RefreshToken,
				svcCtx.Config.Auth.RefreshExpire,
				svcCtx.Config.Auth.CookieSecure,
			)
		}

		httpx.OkJsonCtx(r.Context(), w, result.Response)
	}
}
