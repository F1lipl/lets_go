// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"
	"time"
	"userServer/internal/ecode"

	"userServer/internal/svc"
	"userServer/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RefreshLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRefreshLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefreshLogic {
	return &RefreshLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RefreshLogic) Refresh(refreshToken string) (resp *types.RefreshResp, err error) {
	parsedRefreshToken, err := decodeRefreshToken(refreshToken)
	if err != nil {
		code := ecode.RefreshTokenInvalid
		l.Infow("refreshToken error", logx.Field("err", err), logx.Field("refreshToken", refreshToken))
		return &types.RefreshResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}
	sessionId := parsedRefreshToken.Payload.SessionID
	userSession, err := l.svcCtx.UserSessionModel.FindOne(l.ctx, sessionId)
	if err != nil {
		code := ecode.RefreshTokenInvalid
		l.Infow("refreshToken error", logx.Field("err", err), logx.Field("sessionId", sessionId))
		return &types.RefreshResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}
	if !parsedRefreshToken.Verify([]byte(userSession.Id)) {
		code := ecode.RefreshTokenInvalid
		l.Infow("refreshToken error", logx.Field("err", err), logx.Field("sessionId", sessionId))
		return &types.RefreshResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}

	//session已经失效了
	if userSession.Status == 0 {
		code := ecode.RefreshTokenInvalid
		l.Infow("refreshToken error", logx.Field("err", err), logx.Field("sessionId", sessionId))
		return &types.RefreshResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil

	}

	//查找一下用户
	user, err := l.svcCtx.UserModel.FindOne(l.ctx, userSession.UserId)
	if err != nil {
		code := ecode.RefreshTokenInvalid
		l.Infow("refreshToken error", logx.Field("err", err), logx.Field("sessionId", sessionId))
		return &types.RefreshResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}

	//检查一下用户账户的状态
	switch user.AccountStatus {
	case 0, 3:
		l.Infow(
			"phone login rejected",
			logx.Field("userId", user.UserId),
			logx.Field("accountStatus", user.AccountStatus),
		)
		code := ecode.AccountDisabled

		return &types.RefreshResp{
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

		return &types.RefreshResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil

	case 1:
		// 正常状态，继续刷新

	default:
		l.Errorw(
			"unexpected account status",
			logx.Field("userId", user.UserId),
			logx.Field("accountStatus", user.AccountStatus),
		)
		code := ecode.AccountDisabled

		return &types.RefreshResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}

	//看refreshToken过没过期
	now := time.Now()
	//过期了
	if now.Before(userSession.NotAfter) {
		l.Infow("session expired",logx.Field("sessionId", sessionId))
	}

	//签发accessToken
	accessToken, err := generateAccessToken(l.svcCtx.Config.Auth.AccessSecret, l.svcCtx.Config.Auth.AccessExpire, user.UserId, userSession.Id)
	if err != nil {
		code := ecode.AccessTokenInvalid
		l.Infow("refreshToken error", logx.Field("err", err), logx.Field("sessionId", sessionId))
		return &types.RefreshResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}

	//更新refreshToken
	userSession.UpdatedAt = now
	userSession.Status = 1
	userSession.RefreshTokenCounter+=1
	paylod:=RefreshTokenPayload{
		Version:   parsedRefreshToken.Payload.Version,
		SessionID: userSession.Id,
		Counter:   userSession.RefreshTokenCounter,
	}
	refreshToken,err=encodeRefreshToken(paylod,[]byte(userSession.RefreshTokenKey))
	if err != nil {
		code := ecode.RefreshTokenInvalid
		l.Infow("refreshToken error", logx.Field("err", err), logx.Field("sessionId", sessionId))
		return &types.RefreshResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}

	return &types.RefreshResp{
		ErrorCode: ecode.Success.Int(),
		Message:ecode.Success.Message(),
		AccessToken: accessToken,
	},nil
}
