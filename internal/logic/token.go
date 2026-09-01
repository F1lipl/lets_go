package logic

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

func generateAccessToken(secret string, expire int64, userId string, sessionId string) (string, error) {
	now := time.Now().Unix()
	claims := jwt.MapClaims{
		"iat":       now,
		"exp":       now + expire,
		"sub":       userId,
		"userId":    userId,
		"sessionId": sessionId,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

type RefreshTokenPayload struct {
	Version   uint8  `json:"v"`
	SessionID string `json:"sid"`
	Counter   uint64 `json:"ctr"`
}

type parsedRefreshToken struct {
	Payload        RefreshTokenPayload
	encodedPayload string
	signature      []byte
}

const (
	refreshTokenVersion = uint8(1)
	refreshTokenKeySize = 32
	maxRefreshTokenSize = 1024
)

var ErrInvalidRefreshToken = errors.New("refresh token invalid")

// 每个session创建的时候创建一个唯一的key
func newRefreshTokenKey() ([]byte, error) {
	key := make([]byte, refreshTokenKeySize)

	if _, err := rand.Read(key); err != nil {
		return nil, err
	}

	return key, nil
}

// 生成token字符串
func encodeRefreshToken(
	payload RefreshTokenPayload,
	key []byte,
) (string, error) {
	if len(key) < refreshTokenKeySize {
		return "", errors.New("refresh token key长度不足")
	}

	if payload.Version != refreshTokenVersion {
		return "", errors.New("refresh token版本不正确")
	}

	if _, err := uuid.Parse(payload.SessionID); err != nil {
		return "", errors.New("sessionId格式不正确")
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(
		payloadBytes,
	)

	mac := hmac.New(sha256.New, key)

	if _, err := mac.Write([]byte(encodedPayload)); err != nil {
		return "", err
	}

	signature := mac.Sum(nil)

	encodedSignature := base64.RawURLEncoding.EncodeToString(
		signature,
	)

	return encodedPayload + "." + encodedSignature, nil
}

// 对token进行解码，提取出数据
func decodeRefreshToken(
	token string,
) (*parsedRefreshToken, error) {
	if token == "" || len(token) > maxRefreshTokenSize {
		return nil, ErrInvalidRefreshToken
	}

	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, ErrInvalidRefreshToken
	}

	encodedPayload := parts[0]
	encodedSignature := parts[1]

	payloadBytes, err := base64.RawURLEncoding.DecodeString(
		encodedPayload,
	)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	signature, err := base64.RawURLEncoding.DecodeString(
		encodedSignature,
	)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	if len(signature) != sha256.Size {
		return nil, ErrInvalidRefreshToken
	}

	var payload RefreshTokenPayload

	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, ErrInvalidRefreshToken
	}

	if payload.Version != refreshTokenVersion {
		return nil, ErrInvalidRefreshToken
	}

	if _, err := uuid.Parse(payload.SessionID); err != nil {
		return nil, ErrInvalidRefreshToken
	}

	return &parsedRefreshToken{
		Payload:        payload,
		encodedPayload: encodedPayload,
		signature:      signature,
	}, nil
}
