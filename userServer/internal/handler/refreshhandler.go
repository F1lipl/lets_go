// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package handler

import (
	"errors"
	"net/http"
	"strings"
	"userServer/internal/ecode"
	"userServer/internal/types"

	"userServer/internal/logic"
	"userServer/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func RefreshHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("refresh_token")
		if errors.Is(err, http.ErrNoCookie) {
			code := ecode.RefreshTokenInvalid

			httpx.OkJsonCtx(
				r.Context(),
				w,
				&types.RefreshResp{
					ErrorCode: code.Int(),
					Message:   code.Message(),
				},
			)
			return
		}
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		refreshToken := strings.TrimSpace(cookie.Value)
		if refreshToken == "" {
			code := ecode.RefreshTokenInvalid

			httpx.OkJsonCtx(
				r.Context(),
				w,
				&types.RefreshResp{
					ErrorCode: code.Int(),
					Message:   code.Message(),
				})
			return
		}
		l := logic.NewRefreshLogic(r.Context(), svcCtx)
		result, err := l.Refresh(refreshToken)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		if result == nil || result.Response == nil {
			httpx.ErrorCtx(r.Context(), w, errors.New("refresh result is nil"))
			return
		}

		if result.RefreshToken != "" {
			writeRefreshTokenCookie(
				w,
				result.RefreshToken,
				result.RefreshExpireSeconds,
				svcCtx.Config.Auth.CookieSecure,
			)
		} else if shouldClearRefreshTokenCookie(result.Response.ErrorCode) {
			clearRefreshTokenCookie(
				w,
				svcCtx.Config.Auth.CookieSecure,
			)
		}

		httpx.OkJsonCtx(r.Context(), w, result.Response)
	}
}

func shouldClearRefreshTokenCookie(errorCode int) bool {
	switch ecode.Code(errorCode) {
	case ecode.RefreshTokenInvalid,
		ecode.RefreshTokenExpired,
		ecode.RefreshTokenAlreadyUsed,
		ecode.SessionNotFound,
		ecode.SessionInactive,
		ecode.SessionExpired:
		return true
	default:
		return false
	}
}
