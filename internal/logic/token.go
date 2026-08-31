package logic

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func generateAccessToken(secret string, expire int64, userId string) (string, error) {
	now := time.Now().Unix()
	claims := jwt.MapClaims{
		"iat":    now,
		"exp":    now + expire,
		"sub":    userId,
		"userId": userId,
		//"sessionId": sessionId,
		"tokenType": "access",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func generateRefreshToken(secret string, expire int64, userId string, sessionId string, refreshId string) (string, error) {
	now := time.Now().Unix()
	claims := jwt.MapClaims{
		"iat":       now,
		"exp":       now + expire,
		"sub":       userId,
		"userId":    userId,
		"sessionId": sessionId,
		"jti":       refreshId,
		"tokenType": "refresh",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
