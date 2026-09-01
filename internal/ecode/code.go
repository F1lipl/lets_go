// Package ecode defines stable application-level response codes.
//
// Code ranges:
//   - 0: success
//   - 100xxx: request validation
//   - 200xxx: user account
//   - 300xxx: login and token
//   - 400xxx: verification code
//   - 500xxx: user device
//   - 900xxx: internal service dependencies
package ecode

// Code is an application-level response code returned by the API.
// It is independent of the HTTP status code.
type Code int

const (
	Success Code = 0

	InvalidRequest     Code = 100001
	InvalidUsername    Code = 100002
	InvalidPhoneNumber Code = 100003
	InvalidPassword    Code = 100004

	UserNotFound           Code = 200001
	PhoneAlreadyRegistered Code = 200002
	UsernameAlreadyExists  Code = 200003
	AccountDisabled        Code = 200004
	AccountPending         Code = 200005

	LoginCredentialInvalid Code = 300001
	AccessTokenInvalid     Code = 300002
	AccessTokenExpired     Code = 300003
	RefreshTokenInvalid    Code = 300004
	RefreshTokenExpired    Code = 300005

	VerificationCodeInvalid     Code = 400001
	VerificationCodeExpired     Code = 400002
	VerificationCodeTooFrequent Code = 400003

	InvalidDeviceInfo Code = 500001
	DeviceNotFound    Code = 500002
	DeviceDisabled    Code = 500003

	InternalError Code = 900001
	DatabaseError Code = 900002
	CacheError    Code = 900003
)

var messages = map[Code]string{
	Success:                     "操作成功",
	InvalidRequest:              "请求参数不正确",
	InvalidUsername:             "用户名格式不正确",
	InvalidPhoneNumber:          "电话号码格式不正确",
	InvalidPassword:             "密码格式不正确",
	UserNotFound:                "用户不存在",
	PhoneAlreadyRegistered:      "该电话号码已经注册",
	UsernameAlreadyExists:       "该用户名已经存在",
	AccountDisabled:             "当前账号不可用",
	AccountPending:              "当前账号尚未确认",
	LoginCredentialInvalid:      "手机号或密码不正确",
	AccessTokenInvalid:          "登录凭据无效",
	AccessTokenExpired:          "登录凭据已过期",
	RefreshTokenInvalid:         "刷新凭据无效",
	RefreshTokenExpired:         "刷新凭据已过期",
	VerificationCodeInvalid:     "验证码不正确",
	VerificationCodeExpired:     "验证码已过期",
	VerificationCodeTooFrequent: "验证码发送过于频繁",
	InvalidDeviceInfo:           "设备信息不正确",
	DeviceNotFound:              "设备不存在",
	DeviceDisabled:              "当前设备不可用",
	InternalError:               "服务暂时不可用",
	DatabaseError:               "数据处理失败",
	CacheError:                  "临时数据处理失败",
}

// Int converts an application code to the int type used by API responses.
func (c Code) Int() int {
	return int(c)
}

// Message returns the stable client-facing message for the code.
// Unknown codes fall back to the generic internal error message.
func (c Code) Message() string {
	if message, ok := messages[c]; ok {
		return message
	}

	return messages[InternalError]
}

// String implements fmt.Stringer.
func (c Code) String() string {
	return c.Message()
}
