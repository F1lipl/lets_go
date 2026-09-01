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
) (*LoginResult, error) {
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
		return &LoginResult{
			RefreshToken: "",
			Response: &types.LoginResp{
				ErrorCode: code.Int(),
				Message:   code.String(),
			},
		}, nil
	}

	// 用户不存在
	if errors.Is(err, model.ErrNotFound) {
		code := ecode.LoginCredentialInvalid
		l.Infow("登陆的用户不存在",
			logx.Field("identifier", req.Identifier),
			logx.Field("username", req.Identifier))
		return &LoginResult{
			RefreshToken: "",
			Response: &types.LoginResp{
				ErrorCode: code.Int(),
				Message:   code.String(),
			},
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

		return &LoginResult{
			RefreshToken: "",
			Response: &types.LoginResp{
				ErrorCode: code.Int(),
				Message:   code.String(),
			},
		}, nil
	}

	// 手机号登录和用户名登录都必须执行密码比对
	if !comparePassword(user.PasswordDigest, req.Password) {
		code := ecode.LoginCredentialInvalid
		l.Infow("密码错误",
			logx.Field("identifier", req.Identifier),
			logx.Field("username", req.Identifier))
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

		return &LoginResult{
			RefreshToken: "",
			Response: &types.LoginResp{
				ErrorCode: code.Int(),
				Message:   code.String(),
			},
		}, nil

	case 2:
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
		code := ecode.AccountDisabled

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
		code := ecode.LoginCredentialInvalid
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
