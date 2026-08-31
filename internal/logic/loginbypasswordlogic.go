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

type LoginByPasswordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginByPasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginByPasswordLogic {
	return &LoginByPasswordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginByPasswordLogic) LoginByPassword(
	req *types.LoginByPasswordReq,
) (*types.LoginResp, error) {
	var (
		user *model.Users
		err  error
	)

	switch req.IdentifierType {
	case "phoneNumber":
		user, err = l.svcCtx.UserModel.FindOneByPhoneNumber(
			l.ctx,
			req.Identifier,
		)

	case "username":
		user, err = l.svcCtx.UserModel.FindOneByUsername(
			l.ctx,
			req.Identifier,
		)

	default:
		code := ecode.InvalidRequest
		l.Infow("使用了错误的登陆方式",
			logx.Field("identifier", req.Identifier),
		)
		return &types.LoginResp{
			ErrorCode: code.Int(),
			Message:   "identifierType 只能是 phoneNumber 或 username",
		}, nil
	}

	// 用户不存在
	if errors.Is(err, model.ErrNotFound) {
		code := ecode.LoginCredentialInvalid
		l.Infow("登陆的用户不存在",
			logx.Field("identifier", req.Identifier),
			logx.Field("username", req.Identifier))
		return &types.LoginResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}

	// 查询出现其他问题
	if err != nil {
		l.Errorf(
			"查询数据库时出现错误",
			logx.Field("identifier", req.Identifier),
			logx.Field("identifierType", req.IdentifierType),
			logx.Field("error", err.Error()),
		)

		code := ecode.DatabaseError

		return &types.LoginResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}

	// 手机号登录和用户名登录都必须执行密码比对
	if !comparePassword(user.PasswordDigest, req.Password) {
		code := ecode.LoginCredentialInvalid
		l.Infow("密码错误",
			logx.Field("identifier", req.Identifier),
			logx.Field("username", req.Identifier))
		return &types.LoginResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}

	// 检查账号状态
	switch user.AccountStatus {
	case 0, 3:
		code := ecode.AccountDisabled

		return &types.LoginResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil

	case 2:
		code := ecode.AccountPending

		return &types.LoginResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil

	case 1:
		// 正常状态，继续登录

	default:
		code := ecode.AccountDisabled

		return &types.LoginResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}

	accessToken, err := generateAccessToken(
		l.svcCtx.Config.Auth.AccessSecret,
		l.svcCtx.Config.Auth.AccessExpire,
		user.UserId,
		"access",
	)
	if err != nil {
		l.Errorf("generate access token failed: %v", err)

		code := ecode.InternalError
		return &types.LoginResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}

	refreshToken, err := generateAccessToken(
		l.svcCtx.Config.Auth.RefreshSecret,
		l.svcCtx.Config.Auth.RefreshExpire,
		user.UserId,
		"refresh",
	)
	if err != nil {
		l.Errorf("generate refresh token failed: %v", err)

		code := ecode.InternalError
		return &types.LoginResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}

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
