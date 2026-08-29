package logic

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func generateToken(secret string, expire int64, userId string, tokenType string) (string, error) {
	now := time.Now().Unix()
	claims := jwt.MapClaims{
		"exp":  now + expire,
		"iat":  now,
		"sub":  userId,
		"type": tokenType,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
