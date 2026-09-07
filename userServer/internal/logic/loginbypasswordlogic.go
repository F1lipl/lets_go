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
		code := ecode.InvalidLoginIdentifierType
		l.Infow(
			"login rejected",
			logx.Field("operation", "login_by_password"),
			logx.Field("identifierType", req.IdentifierType),
			logx.Field("reason", "invalid_identifier_type"),
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

	// 用户不存在
	if errors.Is(err, model.ErrNotFound) {
		code := ecode.LoginCredentialInvalid
		l.Infow(
			"login rejected",
			logx.Field("operation", "login_by_password"),
			logx.Field("identifierType", req.IdentifierType),
			logx.Field("reason", "credential_mismatch"),
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

	// 查询出现其他问题
	if err != nil {
		code := ecode.DatabaseError
		l.Errorw(
			"query user failed",
			logx.Field("operation", "login_by_password"),
			logx.Field("stage", "query_user"),
			logx.Field("identifierType", req.IdentifierType),
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

	// 手机号登录和用户名登录都必须执行密码比对
	if !comparePassword(user.PasswordDigest, req.Password) {
		code := ecode.LoginCredentialInvalid
		l.Infow(
			"login rejected",
			logx.Field("operation", "login_by_password"),
			logx.Field("identifierType", req.IdentifierType),
			logx.Field("reason", "credential_mismatch"),
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

	// 检查账号状态
	switch user.AccountStatus {
	case 0, 3:
		code := ecode.AccountDisabled
		l.Infow(
			"login rejected",
			logx.Field("operation", "login_by_password"),
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
			"login rejected",
			logx.Field("operation", "login_by_password"),
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
			logx.Field("operation", "login_by_password"),
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
			logx.Field("operation", "login_by_password"),
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
				logx.Field("operation", "login_by_password"),
				logx.Field("userId", user.UserId),
				logx.Field("deviceId", *deviceid),
				logx.Field("reason", "device_disabled"),
				logx.Field("errorCode", code.Int()),
			)
		} else {
			l.Errorw(
				"finalize login failed",
				logx.Field("operation", "login_by_password"),
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
		logx.Field("operation", "login_by_password"),
		logx.Field("result", "success"),
		logx.Field("userId", user.UserId),
		logx.Field("deviceId", Login.Response.DeviceID),
	)
	return Login, nil
}
