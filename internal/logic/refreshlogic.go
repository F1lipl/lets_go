// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"
	"errors"
	"fmt"
	"time"
	"userServer/internal/ecode"
	"userServer/internal/model"

	"userServer/internal/svc"
	"userServer/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type RefreshLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

type RefreshResult struct {
	Response             *types.RefreshResp
	RefreshToken         string
	RefreshExpireSeconds int64
}

func NewRefreshLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefreshLogic {
	return &RefreshLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RefreshLogic) Refresh(refreshToken string) (*RefreshResult, error) {
	parsedRefreshToken, err := decodeRefreshToken(refreshToken)
	if err != nil {
		l.Infow("invalid refresh token", logx.Field("err", err))
		return newRefreshResult(ecode.RefreshTokenInvalid), nil
	}

	sessionID := parsedRefreshToken.Payload.SessionID
	var result *RefreshResult

	err = l.svcCtx.Sqlconn.TransactCtx(
		l.ctx,
		func(ctx context.Context, txSession sqlx.Session) error {
			txConn := sqlx.NewSqlConnFromSession(txSession)
			sessionModel := model.NewUserSessionsModel(txConn)
			userModel := model.NewUsersModel(txConn)

			userSession, err := sessionModel.FindOneForUpdate(ctx, sessionID)
			if errors.Is(err, model.ErrNotFound) {
				result = newRefreshResult(ecode.SessionNotFound)
				return nil
			}
			if err != nil {
				return fmt.Errorf("find session for update: %w", err)
			}

			if !parsedRefreshToken.Verify([]byte(userSession.RefreshTokenKey)) ||
				parsedRefreshToken.Payload.Counter != userSession.RefreshTokenCounter {
				l.Infow("refresh token does not match session", logx.Field("sessionId", sessionID))
				result = newRefreshResult(ecode.RefreshTokenInvalid)
				return nil
			}

			if userSession.Status != 1 {
				l.Infow("session is inactive", logx.Field("sessionId", sessionID))
				result = newRefreshResult(ecode.SessionInactive)
				return nil
			}

			now := time.Now()
			if !now.Before(userSession.NotAfter) {
				l.Infow("session expired", logx.Field("sessionId", sessionID))
				result = newRefreshResult(ecode.SessionExpired)
				return nil
			}

			user, err := userModel.FindOne(ctx, userSession.UserId)
			if errors.Is(err, model.ErrNotFound) {
				result = newRefreshResult(ecode.RefreshTokenInvalid)
				return nil
			}
			if err != nil {
				return fmt.Errorf("find session user: %w", err)
			}

			switch user.AccountStatus {
			case 1:
				// 正常状态，继续刷新。
			case 2:
				result = newRefreshResult(ecode.AccountPending)
				return nil
			default:
				result = newRefreshResult(ecode.AccountDisabled)
				return nil
			}

			accessToken, err := generateAccessToken(
				l.svcCtx.Config.Auth.AccessSecret,
				l.svcCtx.Config.Auth.AccessExpire,
				user.UserId,
				userSession.Id,
			)
			if err != nil {
				return fmt.Errorf("generate access token: %w", err)
			}

			userSession.RefreshTokenCounter++
			userSession.RefreshedAt = now
			newRefreshToken, err := encodeRefreshToken(
				RefreshTokenPayload{
					Version:   refreshTokenVersion,
					SessionID: userSession.Id,
					Counter:   userSession.RefreshTokenCounter,
				},
				[]byte(userSession.RefreshTokenKey),
			)
			if err != nil {
				return fmt.Errorf("encode refresh token: %w", err)
			}

			if err := sessionModel.Update(ctx, userSession); err != nil {
				return fmt.Errorf("update session: %w", err)
			}

			refreshExpireSeconds := int64(userSession.NotAfter.Sub(now) / time.Second)
			if refreshExpireSeconds < 1 {
				refreshExpireSeconds = 1
			}

			result = &RefreshResult{
				RefreshToken:         newRefreshToken,
				RefreshExpireSeconds: refreshExpireSeconds,
				Response: &types.RefreshResp{
					ErrorCode:   ecode.Success.Int(),
					Message:     ecode.Success.Message(),
					AccessToken: accessToken,
				},
			}
			return nil
		},
	)
	if err != nil {
		l.Errorw("refresh transaction failed", logx.Field("err", err), logx.Field("sessionId", sessionID))
		return nil, err
	}

	if result == nil {
		return nil, errors.New("refresh result is nil")
	}
	return result, nil
}

func newRefreshResult(code ecode.Code) *RefreshResult {
	return &RefreshResult{
		Response: &types.RefreshResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		},
	}
}
