package logic

import (
	"context"
	"fmt"
	"time"
	"userServer/internal/ecode"
	"userServer/internal/model"
	"userServer/internal/svc"
	"userServer/internal/types"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type LoginResult struct {
	Response     *types.LoginResp
	RefreshToken string
}

//到这里说明，user是存在并且状态是正常的，deviceInfo是正常的,然后开启事务，更新设备表，更新用户表，创建一个session，然后签发token，和refreshToken

func finalizeLogin(SvcCtx *svc.ServiceContext, ctx context.Context, user *model.Users, info *types.DeviceInfo) (*LoginResult, error) {
	sessionID := uuid.NewString()
	now := time.Now()

	accessToken, err := generateAccessToken(SvcCtx.Config.Auth.AccessSecret, SvcCtx.Config.Auth.AccessExpire, user.UserId, sessionID)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}
	refreshTokenKey, err := newRefreshTokenKey()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token key: %w", err)
	}
	refreshCounter := uint64(0)

	refreshToken, err := encodeRefreshToken(
		RefreshTokenPayload{
			Version:   refreshTokenVersion,
			SessionID: sessionID,
			Counter:   refreshCounter,
		},
		refreshTokenKey,
	)
	if err != nil {
		return nil, fmt.Errorf("encode refresh token: %w", err)
	}
	var loginDevice *model.UserDevices

	err = SvcCtx.Sqlconn.TransactCtx(ctx, func(ctx context.Context, txSession sqlx.Session) error {
		txConn := sqlx.NewSqlConnFromSession(txSession)
		deviceModel := model.NewUserDevicesModel(txConn)
		sessionModel := model.NewUserSessionsModel(txConn)
		userModel := model.NewUsersModel(txConn)

		// 查询、插入或更新设备
		loginDevice, err = resolveLoginDevice(
			ctx,
			deviceModel,
			user.UserId,
			*info,
		)
		if err != nil {
			return fmt.Errorf("resolve login device: %w", err)
		}

		//创建session
		loginSession := &model.UserSessions{
			Id:          sessionID,
			UserId:      user.UserId,
			DeviceId:    loginDevice.Id,
			Status:      1,
			RefreshedAt: now,
			NotAfter: now.Add(
				time.Duration(
					SvcCtx.Config.Auth.RefreshExpire,
				) * time.Second,
			),
			RefreshTokenKey:     string(refreshTokenKey),
			RefreshTokenCounter: refreshCounter,
		}
		_, err = sessionModel.Insert(ctx, loginSession)
		if err != nil {
			return fmt.Errorf("insert login session: %w", err)
		}
		// 只更新最后登录时间
		if err := userModel.UpdateLastLoginAt(
			ctx,
			user.UserId,
			now,
		); err != nil {
			return fmt.Errorf(
				"update last login time: %w",
				err,
			)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &LoginResult{
		RefreshToken: refreshToken,
		Response: &types.LoginResp{
			DeviceID:    loginDevice.Id,
			AccessToken: accessToken,
			ErrorCode:   ecode.Success.Int(),
			Message:     ecode.Success.String(),
		},
	}, nil
}
