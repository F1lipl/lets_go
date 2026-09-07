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
		l.Infow(
			"phone login rejected",
			logx.Field("reason", "user_not_found"),
		)
		code := ecode.UserNotFound
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
		l.Errorw(
			"query user failed",
			logx.Field("err", err),
		)
		code := ecode.DatabaseError
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
		l.Infow(
			"phone login rejected",
			logx.Field("userId", user.UserId),
			logx.Field("accountStatus", user.AccountStatus),
		)
		code := ecode.AccountDisabled

		return &LoginResult{
			RefreshToken: "",
			Response: &types.LoginResp{
				ErrorCode: code.Int(),
				Message:   code.String(),
			},
		}, nil

	case 2:
		l.Infow(
			"phone login rejected",
			logx.Field("userId", user.UserId),
			logx.Field("accountStatus", user.AccountStatus),
		)
		code := ecode.AccountPending

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
		l.Errorw(
			"unexpected account status",
			logx.Field("userId", user.UserId),
			logx.Field("accountStatus", user.AccountStatus),
		)
		code := ecode.AccountDisabled

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
		l.Errorw(
			"verify code from redis failed",
			logx.Field("err", err),
		)
		code := ecode.CacheError
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
		l.Errorw(
			"unexpected redis result type",
			logx.Field("result", result),
		)
		code := ecode.CacheError
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
		l.Infow(
			"phone login rejected",
			logx.Field("reason", "verification_code_missing"),
		)
		code := ecode.VerificationCodeExpired
		return &LoginResult{
			RefreshToken: "",
			Response: &types.LoginResp{
				ErrorCode: code.Int(),
				Message:   code.String(),
			},
		}, nil

	case 0:
		l.Infow(
			"phone login rejected",
			logx.Field("reason", "verification_code_mismatch"),
		)
		code := ecode.VerificationCodeInvalid
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
		l.Errorw(
			"unexpected redis result value",
			logx.Field("result", matched),
		)
		code := ecode.CacheError
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
		l.Infow("device info is validate failed", logx.Field("err", err), logx.Field("userId", user.UserId), logx.Field("deviceInfo", req.Device))
		code := ecode.InvalidDeviceInfo
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
		l.Infow("login error", logx.Field("err", err))
		code := ecode.InternalError
		if errors.Is(err, errDeviceDisabled) {
			code = ecode.DeviceDisabled
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
	return Login, nil
}
