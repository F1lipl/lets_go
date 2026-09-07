package logic

import (
	"context"
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

/**
### 生成过程

先准备 payload：

```go
payload := RefreshTokenPayload{
    Version:   1,
    SessionID: sessionID,
    Counter:   0,
}
```

转换成 JSON：

```json
{
  "v": 1,
  "sid": "session-uuid",
  "ctr": 0
}
```

进行 Base64URL 编码：

```text
encodedPayload = Base64URL(payloadJSON)
```

然后使用 session 对应的 key 计算 HMAC：

```text
signature = HMAC-SHA256(
    sessionTokenKey,
    encodedPayload
)
```

再把签名编码：

```text
encodedSignature = Base64URL(signature)
```

最后拼接：

```text
refreshToken =
encodedPayload + "." + encodedSignature
```

结构就是：

```text
payload.signature
```

需要注意，不要自己简单拼接：

```go
sha256(key + payload)
```

而应该使用标准的 HMAC：

```go
mac := hmac.New(
    sha256.New,
    sessionTokenKey,
)

mac.Write([]byte(encodedPayload))

signature := mac.Sum(nil)
```

### 检查过程

收到：

```text
encodedPayload.encodedSignature
```

第一步，拆分：

```go
parts := strings.Split(token, ".")

encodedPayload := parts[0]
encodedSignature := parts[1]
```

第二步，解码 payload：

```go
payloadBytes, err :=
    base64.RawURLEncoding.DecodeString(encodedPayload)
```

得到：

```json
{
  "v": 1,
  "sid": "session-uuid",
  "ctr": 0
}
```

注意，此时解码出来的数据还不能直接使用，只能先拿 `sessionId` 查询数据库：

```sql
SELECT *
FROM user_sessions
WHERE id = ?;
```

第三步，从数据库取得：

```text
refresh_token_key
```

第四步，使用数据库里的 key 和收到的原始 `encodedPayload` 再计算一次 HMAC：

```go
mac := hmac.New(
    sha256.New,
    []byte(session.RefreshTokenKey),
)

mac.Write([]byte(encodedPayload))

expectedSignature := mac.Sum(nil)
```

第五步，解码 token 中原有的签名：

```go
receivedSignature, err :=
    base64.RawURLEncoding.DecodeString(
        encodedSignature,
    )
```

第六步，比较两个签名：

```go
if !hmac.Equal(
    expectedSignature,
    receivedSignature,
) {
    return ErrInvalidRefreshToken
}
```

比较的是：

```text
重新计算出的signature
        和
token中携带的signature
```

不是比较两个完整 token 字符串。

完整过程可以理解为：

```text
生成：

payload
  ↓ JSON
payloadJSON
  ↓ Base64URL
encodedPayload
  ↓ 使用sessionKey计算HMAC
signature
  ↓ Base64URL
encodedSignature
  ↓ 拼接
encodedPayload.encodedSignature
```

检查：

```text
收到token
  ↓ 拆分
encodedPayload + encodedSignature
  ↓ 解码payload
获得sessionId
  ↓ 查询数据库
获得sessionKey
  ↓ 重新计算HMAC
expectedSignature
  ↓ 与token中的signature比较
相同：payload保持原样
不同：拒绝
```

所以没有“解密”的过程：

- Base64URL 可以直接解码。
- `sessionTokenKey` 不会写入 token。
- HMAC 不能反向还原 key。
- 服务端只是使用相同的 key 和 payload，再计算一次并比较结果。

/
*/

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

// Verify 使用sessionKey对token进行校验
func (t *parsedRefreshToken) Verify(key []byte) bool {
	if t == nil || len(key) < refreshTokenKeySize {
		return false
	}

	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(t.encodedPayload))

	expectedSignature := mac.Sum(nil)

	return hmac.Equal(
		expectedSignature,
		t.signature,
	)
}
func getAccessClaims(context context.Context) (userId string, sessionId string, err error) {
	userId, ok := context.Value("userId").(string)
	if !ok || userId == "" {
		return "", "", errors.New("access token中缺少userId")
	}
	sessionId, ok = context.Value("sessionId").(string)
	if !ok || sessionId == "" {
		return "", "", errors.New("access token中缺少userId")
	}
	return userId, sessionId, nil
}
