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
		l.Infow(
			"access token claims invalid",
			logx.Field("err", err),
		)

		code := ecode.AccessTokenInvalid
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
		l.Infow(
			"user not found",
			logx.Field("userId", userID),
		)

		code := ecode.UserNotFound
		return &types.UserInfoResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}

	if err != nil {
		l.Errorw(
			"search user failed",
			logx.Field("userId", userID),
			logx.Field("err", err),
		)

		code := ecode.DatabaseError
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
		return &types.UserInfoResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	default:
		code := ecode.AccountDisabled
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
