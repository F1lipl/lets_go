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

func (l *LoginByPhoneLogic) LoginByPhone(req *types.LoginByPhoneReq) (resp *types.LoginResp, err error) {
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
		return &types.LoginResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}
	//查询失败
	if err != nil {
		l.Errorw(
			"query user failed",
			logx.Field("err", err),
		)
		code := ecode.DatabaseError
		return &types.LoginResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
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

		return &types.LoginResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil

	case 2:
		l.Infow(
			"phone login rejected",
			logx.Field("userId", user.UserId),
			logx.Field("accountStatus", user.AccountStatus),
		)
		code := ecode.AccountPending

		return &types.LoginResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
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

		return &types.LoginResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
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
		return &types.LoginResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}

	matched, ok := result.(int64)
	if !ok {
		l.Errorw(
			"unexpected redis result type",
			logx.Field("result", result),
		)
		code := ecode.CacheError
		return &types.LoginResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}

	switch matched {
	case -1:
		l.Infow(
			"phone login rejected",
			logx.Field("reason", "verification_code_missing"),
		)
		code := ecode.VerificationCodeExpired
		return &types.LoginResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil

	case 0:
		l.Infow(
			"phone login rejected",
			logx.Field("reason", "verification_code_mismatch"),
		)
		code := ecode.VerificationCodeInvalid
		return &types.LoginResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil

	case 1:
		// 验证码正确且已经删除，继续登录。

	default:
		l.Errorw(
			"unexpected redis result value",
			logx.Field("result", matched),
		)
		code := ecode.CacheError
		return &types.LoginResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}

	accessToken, err := generateAccessToken(l.svcCtx.Config.Auth.AccessSecret, l.svcCtx.Config.Auth.AccessExpire, user.UserId, "access")
	if err != nil {
		l.Errorw("generate access token failed", logx.Field("err", err), logx.Field("userId", user.UserId))

		code := ecode.InternalError
		return &types.LoginResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}
	refreshToken, err := generateAccessToken(l.svcCtx.Config.Auth.RefreshSecret, l.svcCtx.Config.Auth.RefreshExpire, user.UserId, "refresh")
	if err != nil {
		l.Errorw("generate refresh token failed", logx.Field("err", err), logx.Field("userId", user.UserId))
		code := ecode.InternalError
		return &types.LoginResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil

	}
	l.Infow(
		"phone login succeeded",
		logx.Field("userId", user.UserId),
	)
	return &types.LoginResp{
		ErrorCode:        ecode.Success.Int(),
		Message:          ecode.Success.Message(),
		UserID:           user.UserId,
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresIn:  l.svcCtx.Config.Auth.AccessExpire,
		RefreshExpiresIn: l.svcCtx.Config.Auth.RefreshExpire,
	}, nil
}
