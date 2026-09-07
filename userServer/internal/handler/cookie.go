package handler

import (
	"net/http"
	"time"
)

const refreshTokenCookieName = "refresh_token"

/*
HttpOnly
    → 前端代码不能直接读取refresh token

Secure
    → 只通过HTTPS发送
    → 本地开发可以暂时设为false

SameSite=Lax
    → 适合同站点前后端

Path=/
    → refresh和logout接口都可以收到Cookie

*/

func writeRefreshTokenCookie(w http.ResponseWriter,
	refreshToken string,
	expireSeconds int64,
	secure bool) {
	maxAge := int(expireSeconds)
	expires := time.Now().Add(
		time.Duration(expireSeconds) * time.Second,
	)

	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    refreshToken,
		Path:     "/",
		MaxAge:   maxAge,
		Expires:  expires,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearRefreshTokenCookie(
	w http.ResponseWriter,
	secure bool,
) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
