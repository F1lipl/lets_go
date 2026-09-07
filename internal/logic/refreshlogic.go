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
		code := ecode.RefreshTokenInvalid
		l.Infow(
			"refresh rejected",
			logx.Field("operation", "refresh_session"),
			logx.Field("reason", "invalid_refresh_token"),
			logx.Field("errorCode", code.Int()),
			logx.Field("err", err),
		)
		return newRefreshResult(code), nil
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
				code := ecode.SessionNotFound
				l.Infow(
					"refresh rejected",
					logx.Field("operation", "refresh_session"),
					logx.Field("sessionId", sessionID),
					logx.Field("reason", "session_not_found"),
					logx.Field("errorCode", code.Int()),
				)
				result = newRefreshResult(code)
				return nil
			}
			if err != nil {
				return fmt.Errorf("find session for update: %w", err)
			}
			if !parsedRefreshToken.Verify([]byte(userSession.RefreshTokenKey)) {
				code := ecode.RefreshTokenInvalid
				l.Infow(
					"refresh rejected",
					logx.Field("operation", "refresh_session"),
					logx.Field("sessionId", sessionID),
					logx.Field("reason", "refresh_token_mismatch"),
					logx.Field("errorCode", code.Int()),
				)
				result = newRefreshResult(code)
				return nil
			}
			if parsedRefreshToken.Payload.Counter != userSession.RefreshTokenCounter {
				code := ecode.RefreshTokenAlreadyUsed
				l.Infow(
					"refresh rejected",
					logx.Field("operation", "refresh_session"),
					logx.Field("sessionId", sessionID),
					logx.Field("reason", "refresh_token_already_used"),
					logx.Field("errorCode", code.Int()),
				)
				result = newRefreshResult(code)
				return nil
			}

			if userSession.Status != 1 {
				code := ecode.SessionInactive
				l.Infow(
					"refresh rejected",
					logx.Field("operation", "refresh_session"),
					logx.Field("sessionId", sessionID),
					logx.Field("reason", "session_inactive"),
					logx.Field("errorCode", code.Int()),
				)
				result = newRefreshResult(code)
				return nil
			}

			now := time.Now()
			if !now.Before(userSession.NotAfter) {
				code := ecode.SessionExpired
				l.Infow(
					"refresh rejected",
					logx.Field("operation", "refresh_session"),
					logx.Field("sessionId", sessionID),
					logx.Field("reason", "session_expired"),
					logx.Field("errorCode", code.Int()),
				)
				result = newRefreshResult(code)
				return nil
			}

			user, err := userModel.FindOne(ctx, userSession.UserId)
			if errors.Is(err, model.ErrNotFound) {
				code := ecode.RefreshTokenInvalid
				l.Errorw(
					"session user not found",
					logx.Field("operation", "refresh_session"),
					logx.Field("stage", "query_user"),
					logx.Field("userId", userSession.UserId),
					logx.Field("sessionId", sessionID),
					logx.Field("errorCode", code.Int()),
				)
				result = newRefreshResult(code)
				return nil
			}
			if err != nil {
				return fmt.Errorf("find session user: %w", err)
			}

			switch user.AccountStatus {
			case 1:
				// 正常状态，继续刷新。
			case 2:
				code := ecode.AccountPending
				l.Infow(
					"refresh rejected",
					logx.Field("operation", "refresh_session"),
					logx.Field("userId", user.UserId),
					logx.Field("sessionId", sessionID),
					logx.Field("accountStatus", user.AccountStatus),
					logx.Field("reason", "account_pending"),
					logx.Field("errorCode", code.Int()),
				)
				result = newRefreshResult(code)
				return nil
			case 0, 3:
				code := ecode.AccountDisabled
				l.Infow(
					"refresh rejected",
					logx.Field("operation", "refresh_session"),
					logx.Field("userId", user.UserId),
					logx.Field("sessionId", sessionID),
					logx.Field("accountStatus", user.AccountStatus),
					logx.Field("reason", "account_disabled"),
					logx.Field("errorCode", code.Int()),
				)
				result = newRefreshResult(code)
				return nil
			default:
				code := ecode.AccountDisabled
				l.Errorw(
					"unexpected account status",
					logx.Field("operation", "refresh_session"),
					logx.Field("userId", user.UserId),
					logx.Field("sessionId", sessionID),
					logx.Field("accountStatus", user.AccountStatus),
					logx.Field("errorCode", code.Int()),
				)
				result = newRefreshResult(code)
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
		l.Errorw(
			"refresh transaction failed",
			logx.Field("operation", "refresh_session"),
			logx.Field("stage", "transaction"),
			logx.Field("sessionId", sessionID),
			logx.Field("errorCode", ecode.InternalError.Int()),
			logx.Field("err", err),
		)
		return nil, err
	}

	if result == nil {
		err := errors.New("refresh result is nil")
		l.Errorw(
			"refresh result missing",
			logx.Field("operation", "refresh_session"),
			logx.Field("stage", "build_response"),
			logx.Field("sessionId", sessionID),
			logx.Field("errorCode", ecode.InternalError.Int()),
			logx.Field("err", err),
		)
		return nil, err
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
