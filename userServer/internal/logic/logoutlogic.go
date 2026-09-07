// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"
	"database/sql"
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

type LogoutLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLogoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogoutLogic {
	return &LogoutLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LogoutLogic) Logout() (*types.LogoutResp, error) {
	userID, sessionID, err := getAccessClaims(l.ctx)
	if err != nil {
		code := ecode.AccessTokenInvalid
		l.Infow(
			"logout rejected",
			logx.Field("operation", "logout"),
			logx.Field("reason", "invalid_access_claims"),
			logx.Field("errorCode", code.Int()),
			logx.Field("err", err),
		)

		return logoutResponse(code), nil
	}

	var resultCode = ecode.Success

	err = l.svcCtx.Sqlconn.TransactCtx(
		l.ctx,
		func(ctx context.Context, txSession sqlx.Session) error {
			txConn := sqlx.NewSqlConnFromSession(txSession)
			sessionModel := model.NewUserSessionsModel(txConn)

			userSession, findErr :=
				sessionModel.FindOneForUpdate(ctx, sessionID)

			if errors.Is(findErr, model.ErrNotFound) {
				// Session已经不存在，按照重复登出处理。
				return nil
			}

			if findErr != nil {
				return fmt.Errorf(
					"find session for update: %w",
					findErr,
				)
			}

			if userSession.UserId != userID {
				resultCode = ecode.SessionNotFound
				return nil
			}

			// 已经退出，直接成功。
			if userSession.Status != 1 {
				return nil
			}

			now := time.Now()

			userSession.Status = 0
			userSession.RevokedAt = sql.NullTime{
				Time:  now,
				Valid: true,
			}
			userSession.RevokeReason = sql.NullString{
				String: "logout",
				Valid:  true,
			}

			if updateErr := sessionModel.Update(
				ctx,
				userSession,
			); updateErr != nil {
				return fmt.Errorf(
					"update session: %w",
					updateErr,
				)
			}

			return nil
		},
	)

	if err != nil {
		l.Errorw(
			"logout transaction failed",
			logx.Field("operation", "logout"),
			logx.Field("stage", "transaction"),
			logx.Field("userId", userID),
			logx.Field("sessionId", sessionID),
			logx.Field("errorCode", ecode.DatabaseError.Int()),
			logx.Field("err", err),
		)

		return logoutResponse(ecode.DatabaseError), nil
	}

	if resultCode == ecode.Success {
		l.Infow(
			"logout completed",
			logx.Field("operation", "logout"),
			logx.Field("result", "success"),
			logx.Field("userId", userID),
			logx.Field("sessionId", sessionID),
		)
	} else {
		l.Infow(
			"logout rejected",
			logx.Field("operation", "logout"),
			logx.Field("result", "rejected"),
			logx.Field("userId", userID),
			logx.Field("sessionId", sessionID),
			logx.Field("reason", "session_not_found"),
			logx.Field("errorCode", resultCode.Int()),
		)
	}

	return logoutResponse(resultCode), nil
}

func logoutResponse(code ecode.Code) *types.LogoutResp {
	return &types.LogoutResp{
		ErrorCode: code.Int(),
		Message:   code.Message(),
	}
}
