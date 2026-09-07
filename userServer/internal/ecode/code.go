// Package ecode defines stable application-level response codes.
//
// Code ranges:
//   - 0: success
//   - 100xxx: request validation
//   - 200xxx: user account
//   - 300xxx: login and token
//   - 400xxx: verification code
//   - 500xxx: user device
//   - 600xxx: user session
//   - 700xxx: password management
//   - 900xxx: internal service dependencies
package ecode

// Code is an application-level response code returned by the API.
// It is independent of the HTTP status code.
type Code int

const (
	Success Code = 0

	InvalidRequest             Code = 100001
	InvalidUsername            Code = 100002
	InvalidPhoneNumber         Code = 100003
	InvalidPassword            Code = 100004
	InvalidLoginIdentifierType Code = 100005

	UserNotFound           Code = 200001
	PhoneAlreadyRegistered Code = 200002
	UsernameAlreadyExists  Code = 200003
	AccountDisabled        Code = 200004
	AccountPending         Code = 200005

	LoginCredentialInvalid  Code = 300001
	AccessTokenInvalid      Code = 300002
	AccessTokenExpired      Code = 300003
	RefreshTokenInvalid     Code = 300004
	RefreshTokenExpired     Code = 300005
	RefreshTokenAlreadyUsed Code = 300006

	VerificationCodeInvalid     Code = 400001
	VerificationCodeExpired     Code = 400002
	VerificationCodeTooFrequent Code = 400003
	VerificationCodeSendFailed  Code = 400004

	InvalidDeviceInfo Code = 500001
	DeviceNotFound    Code = 500002
	DeviceDisabled    Code = 500003

	SessionNotFound      Code = 600001
	SessionInactive      Code = 600002
	SessionExpired       Code = 600003
	SessionLimitExceeded Code = 600004

	CurrentPasswordInvalid     Code = 700001
	NewPasswordSameAsCurrent   Code = 700002
	PasswordChangeFailed       Code = 700003
	PasswordResetTicketInvalid Code = 700004
	PasswordResetTicketExpired Code = 700005
	PasswordResetTicketUsed    Code = 700006
	PasswordResetTooFrequent   Code = 700007
	PasswordResetFailed        Code = 700008

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
	InvalidLoginIdentifierType:  "登录标识类型不正确",
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
	RefreshTokenAlreadyUsed:     "刷新凭据已被使用，请重新登录",
	VerificationCodeInvalid:     "验证码不正确",
	VerificationCodeExpired:     "验证码已过期",
	VerificationCodeTooFrequent: "验证码发送过于频繁",
	VerificationCodeSendFailed:  "验证码发送失败，请稍后重试",
	InvalidDeviceInfo:           "设备信息不正确",
	DeviceNotFound:              "设备不存在",
	DeviceDisabled:              "当前设备不可用",
	SessionNotFound:             "登录会话不存在",
	SessionInactive:             "登录会话已失效",
	SessionExpired:              "登录会话已过期",
	SessionLimitExceeded:        "登录会话数量已达上限",
	CurrentPasswordInvalid:      "当前密码不正确",
	NewPasswordSameAsCurrent:    "新密码不能与当前密码相同",
	PasswordChangeFailed:        "修改密码失败，请稍后重试",
	PasswordResetTicketInvalid:  "密码重置凭据无效",
	PasswordResetTicketExpired:  "密码重置凭据已过期",
	PasswordResetTicketUsed:     "密码重置凭据已被使用",
	PasswordResetTooFrequent:    "密码重置操作过于频繁，请稍后重试",
	PasswordResetFailed:         "重置密码失败，请稍后重试",
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
