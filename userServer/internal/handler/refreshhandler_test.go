package handler

import (
	"testing"

	"userServer/internal/ecode"
)

func TestShouldClearRefreshTokenCookie(t *testing.T) {
	tests := []struct {
		name string
		code ecode.Code
		want bool
	}{
		{name: "refresh token invalid", code: ecode.RefreshTokenInvalid, want: true},
		{name: "refresh token expired", code: ecode.RefreshTokenExpired, want: true},
		{name: "refresh token already used", code: ecode.RefreshTokenAlreadyUsed, want: true},
		{name: "session not found", code: ecode.SessionNotFound, want: true},
		{name: "session inactive", code: ecode.SessionInactive, want: true},
		{name: "session expired", code: ecode.SessionExpired, want: true},
		{name: "database error", code: ecode.DatabaseError, want: false},
		{name: "cache error", code: ecode.CacheError, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldClearRefreshTokenCookie(tt.code.Int()); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
