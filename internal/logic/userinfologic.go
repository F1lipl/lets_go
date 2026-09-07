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

type UserInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserInfoLogic {
	return &UserInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UserInfoLogic) UserInfo() (
	*types.UserInfoResp,
	error,
) {
	userID, _, err := getAccessClaims(l.ctx)
	if err != nil {
		code := ecode.AccessTokenInvalid
		l.Infow(
			"get user info rejected",
			logx.Field("operation", "get_user_info"),
			logx.Field("reason", "invalid_access_claims"),
			logx.Field("errorCode", code.Int()),
			logx.Field("err", err),
		)

		return &types.UserInfoResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}

	user, err := l.svcCtx.UserModel.FindOne(
		l.ctx,
		userID,
	)

	if errors.Is(err, model.ErrNotFound) {
		code := ecode.UserNotFound
		l.Errorw(
			"user not found",
			logx.Field("operation", "get_user_info"),
			logx.Field("stage", "query_user"),
			logx.Field("userId", userID),
			logx.Field("errorCode", code.Int()),
		)

		return &types.UserInfoResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}

	if err != nil {
		code := ecode.DatabaseError
		l.Errorw(
			"search user failed",
			logx.Field("operation", "get_user_info"),
			logx.Field("stage", "query_user"),
			logx.Field("userId", userID),
			logx.Field("errorCode", code.Int()),
			logx.Field("err", err),
		)

		return &types.UserInfoResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}

	switch user.AccountStatus {
	case 1:
		// 正常状态，继续返回资料。
	case 2:
		code := ecode.AccountPending
		l.Infow(
			"get user info rejected",
			logx.Field("operation", "get_user_info"),
			logx.Field("userId", user.UserId),
			logx.Field("accountStatus", user.AccountStatus),
			logx.Field("reason", "account_pending"),
			logx.Field("errorCode", code.Int()),
		)
		return &types.UserInfoResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	case 0, 3:
		code := ecode.AccountDisabled
		l.Infow(
			"get user info rejected",
			logx.Field("operation", "get_user_info"),
			logx.Field("userId", user.UserId),
			logx.Field("accountStatus", user.AccountStatus),
			logx.Field("reason", "account_disabled"),
			logx.Field("errorCode", code.Int()),
		)
		return &types.UserInfoResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	default:
		code := ecode.AccountDisabled
		l.Errorw(
			"unexpected account status",
			logx.Field("operation", "get_user_info"),
			logx.Field("userId", user.UserId),
			logx.Field("accountStatus", user.AccountStatus),
			logx.Field("errorCode", code.Int()),
		)
		return &types.UserInfoResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}

	return &types.UserInfoResp{
		ErrorCode: ecode.Success.Int(),
		Message:   ecode.Success.Message(),
		UserID:    user.UserId,
		Username:  user.Username,
		Nickname:  user.Nickname.String,
		AvatarURL: user.AvatarUrl.String,
	}, nil
}
