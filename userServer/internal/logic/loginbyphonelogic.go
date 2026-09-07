// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"
	"errors"
	"userServer/internal/ecode"
	"userServer/internal/model"

	"userServer/internal/svc"
	"userServer/internal/types"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type LoginByPhoneLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginByPhoneLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginByPhoneLogic {
	return &LoginByPhoneLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

const consumeLoginVerificationCodeScript = `
local value = redis.call("GET", KEYS[1])
if not value then
    return -1
end
if value ~= ARGV[1] then
    return 0
end
redis.call("DEL", KEYS[1])
return 1
`

func loginVerificationCodeKey(phoneNumber string) string {
	return "verification:login:" + phoneNumber
}

func (l *LoginByPhoneLogic) LoginByPhone(req *types.LoginByPhoneReq) (resp *LoginResult, err error) {
	phoneNumber := req.PhoneNumber
	verificationCode := req.VerificationCode

	user, err := l.svcCtx.UserModel.FindOneByPhoneNumber(l.ctx, phoneNumber)
	//用户不存在
	if errors.Is(err, model.ErrNotFound) {
		code := ecode.UserNotFound
		l.Infow(
			"phone login rejected",
			logx.Field("operation", "login_by_phone"),
			logx.Field("reason", "user_not_found"),
			logx.Field("errorCode", code.Int()),
		)
		return &LoginResult{
			RefreshToken: "",
			Response: &types.LoginResp{
				ErrorCode: code.Int(),
				Message:   code.String(),
			},
		}, nil
	}
	//查询失败
	if err != nil {
		code := ecode.DatabaseError
		l.Errorw(
			"query user failed",
			logx.Field("operation", "login_by_phone"),
			logx.Field("stage", "query_user"),
			logx.Field("errorCode", code.Int()),
			logx.Field("err", err),
		)
		return &LoginResult{
			RefreshToken: "",
			Response: &types.LoginResp{
				ErrorCode: code.Int(),
				Message:   code.String(),
			},
		}, nil
	}

	// 检查账号状态
	switch user.AccountStatus {
	case 0, 3:
		code := ecode.AccountDisabled
		l.Infow(
			"phone login rejected",
			logx.Field("operation", "login_by_phone"),
			logx.Field("userId", user.UserId),
			logx.Field("accountStatus", user.AccountStatus),
			logx.Field("reason", "account_disabled"),
			logx.Field("errorCode", code.Int()),
		)

		return &LoginResult{
			RefreshToken: "",
			Response: &types.LoginResp{
				ErrorCode: code.Int(),
				Message:   code.String(),
			},
		}, nil

	case 2:
		code := ecode.AccountPending
		l.Infow(
			"phone login rejected",
			logx.Field("operation", "login_by_phone"),
			logx.Field("userId", user.UserId),
			logx.Field("accountStatus", user.AccountStatus),
			logx.Field("reason", "account_pending"),
			logx.Field("errorCode", code.Int()),
		)

		return &LoginResult{
			RefreshToken: "",
			Response: &types.LoginResp{
				ErrorCode: code.Int(),
				Message:   code.String(),
			},
		}, nil

	case 1:
		// 正常状态，继续登录

	default:
		code := ecode.AccountDisabled
		l.Errorw(
			"unexpected account status",
			logx.Field("operation", "login_by_phone"),
			logx.Field("userId", user.UserId),
			logx.Field("accountStatus", user.AccountStatus),
			logx.Field("errorCode", code.Int()),
		)

		return &LoginResult{
			RefreshToken: "",
			Response: &types.LoginResp{
				ErrorCode: code.Int(),
				Message:   code.String(),
			},
		}, nil
	}

	// Redis 一次完成读取、比较和删除，确保验证码只能成功使用一次。
	result, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		consumeLoginVerificationCodeScript,
		[]string{loginVerificationCodeKey(phoneNumber)},
		verificationCode,
	)
	if err != nil {
		code := ecode.CacheError
		l.Errorw(
			"verify code from redis failed",
			logx.Field("operation", "login_by_phone"),
			logx.Field("stage", "consume_verification_code"),
			logx.Field("userId", user.UserId),
			logx.Field("errorCode", code.Int()),
			logx.Field("err", err),
		)
		return &LoginResult{
			RefreshToken: "",
			Response: &types.LoginResp{
				ErrorCode: code.Int(),
				Message:   code.String(),
			},
		}, nil
	}

	matched, ok := result.(int64)
	if !ok {
		code := ecode.CacheError
		l.Errorw(
			"unexpected redis result type",
			logx.Field("operation", "login_by_phone"),
			logx.Field("stage", "consume_verification_code"),
			logx.Field("userId", user.UserId),
			logx.Field("errorCode", code.Int()),
		)
		return &LoginResult{
			RefreshToken: "",
			Response: &types.LoginResp{
				ErrorCode: code.Int(),
				Message:   code.String(),
			},
		}, nil
	}

	switch matched {
	case -1:
		code := ecode.VerificationCodeExpired
		l.Infow(
			"phone login rejected",
			logx.Field("operation", "login_by_phone"),
			logx.Field("userId", user.UserId),
			logx.Field("reason", "verification_code_missing"),
			logx.Field("errorCode", code.Int()),
		)
		return &LoginResult{
			RefreshToken: "",
			Response: &types.LoginResp{
				ErrorCode: code.Int(),
				Message:   code.String(),
			},
		}, nil

	case 0:
		code := ecode.VerificationCodeInvalid
		l.Infow(
			"phone login rejected",
			logx.Field("operation", "login_by_phone"),
			logx.Field("userId", user.UserId),
			logx.Field("reason", "verification_code_mismatch"),
			logx.Field("errorCode", code.Int()),
		)
		return &LoginResult{
			RefreshToken: "",
			Response: &types.LoginResp{
				ErrorCode: code.Int(),
				Message:   code.String(),
			},
		}, nil

	case 1:
		// 验证码正确且已经删除，继续登录。

	default:
		code := ecode.CacheError
		l.Errorw(
			"unexpected redis result value",
			logx.Field("operation", "login_by_phone"),
			logx.Field("stage", "consume_verification_code"),
			logx.Field("userId", user.UserId),
			logx.Field("result", matched),
			logx.Field("errorCode", code.Int()),
		)
		return &LoginResult{
			RefreshToken: "",
			Response: &types.LoginResp{
				ErrorCode: code.Int(),
				Message:   code.String(),
			},
		}, nil
	}
	//判断设备信息
	deviceid := &req.Device.DeviceID
	if *deviceid == "" {
		*deviceid = uuid.NewString()
	}
	err = validateDeviceInfo(&req.Device)
	if err != nil {
		code := ecode.InvalidDeviceInfo
		l.Infow(
			"device info rejected",
			logx.Field("operation", "login_by_phone"),
			logx.Field("stage", "validate_device"),
			logx.Field("userId", user.UserId),
			logx.Field("reason", "invalid_device_info"),
			logx.Field("detail", err.Error()),
			logx.Field("errorCode", code.Int()),
		)
		return &LoginResult{
			RefreshToken: "",
			Response: &types.LoginResp{
				ErrorCode: code.Int(),
				Message:   code.String(),
				DeviceID:  *deviceid,
			},
		}, nil
	}
	Login, err := finalizeLogin(l.svcCtx, l.ctx, user, &req.Device)
	if err != nil {
		code := ecode.InternalError
		if errors.Is(err, errDeviceDisabled) {
			code = ecode.DeviceDisabled
			l.Infow(
				"login rejected",
				logx.Field("operation", "login_by_phone"),
				logx.Field("userId", user.UserId),
				logx.Field("deviceId", *deviceid),
				logx.Field("reason", "device_disabled"),
				logx.Field("errorCode", code.Int()),
			)
		} else {
			l.Errorw(
				"finalize login failed",
				logx.Field("operation", "login_by_phone"),
				logx.Field("stage", "finalize_login"),
				logx.Field("userId", user.UserId),
				logx.Field("deviceId", *deviceid),
				logx.Field("errorCode", code.Int()),
				logx.Field("err", err),
			)
		}
		return &LoginResult{
			RefreshToken: "",
			Response: &types.LoginResp{
				ErrorCode: code.Int(),
				Message:   code.String(),
				DeviceID:  *deviceid,
			},
		}, nil
	}
	l.Infow(
		"login succeeded",
		logx.Field("operation", "login_by_phone"),
		logx.Field("result", "success"),
		logx.Field("userId", user.UserId),
		logx.Field("deviceId", Login.Response.DeviceID),
	)
	return Login, nil
}
