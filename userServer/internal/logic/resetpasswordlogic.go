// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"userServer/internal/ecode"
	"userServer/internal/model"

	"userServer/internal/svc"
	"userServer/internal/types"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

const (
	resetTicketLockSeconds = 30
	resetTicketMissing     = "__ticket_missing__"
	resetTicketBusy        = "__ticket_busy__"
)

const claimResetTicketScript = `
local userId = redis.call("GET", KEYS[1])
if not userId then
    return "__ticket_missing__"
end

local acquired = redis.call("SET", KEYS[2], ARGV[1], "NX", "EX", ARGV[2])
if not acquired then
    return "__ticket_busy__"
end

return userId
`

const releaseResetTicketScript = `
if redis.call("GET", KEYS[1]) ~= ARGV[1] then
    return 0
end

return redis.call("DEL", KEYS[1])
`

const consumeResetTicketScript = `
if redis.call("GET", KEYS[2]) ~= ARGV[1] then
    return 0
end

if not redis.call("GET", KEYS[1]) then
    redis.call("DEL", KEYS[2])
    return 0
end

redis.call("DEL", KEYS[1])
redis.call("DEL", KEYS[2])
return 1
`

type ResetPasswordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewResetPasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResetPasswordLogic {
	return &ResetPasswordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func resetTicketDigest(ticket string) string {
	digest := sha256.Sum256([]byte(ticket))
	return hex.EncodeToString(digest[:])
}

func resetTicketKey(ticket string) string {
	return "password-reset:ticket:" + resetTicketDigest(ticket)
}

func resetTicketLockKey(ticket string) string {
	return "password-reset:ticket-lock:" + resetTicketDigest(ticket)
}

func (l *ResetPasswordLogic) ResetPassword(
	req *types.ResetPasswordReq,
) (*types.ResetPasswordResp, error) {
	if req == nil {
		return resetPasswordResponse(ecode.InvalidRequest), nil
	}

	ticket := strings.TrimSpace(req.ResetTicket)
	if ticket == "" {
		return resetPasswordResponse(ecode.PasswordResetTicketInvalid), nil
	}

	if err := validatePassword(req.NewPassword); err != nil {
		code := ecode.InvalidPassword
		l.Infow(
			"reset password rejected",
			logx.Field("operation", "reset_password"),
			logx.Field("reason", "invalid_new_password"),
			logx.Field("detail", err.Error()),
			logx.Field("errorCode", code.Int()),
		)
		return resetPasswordResponse(code), nil
	}

	ticketKey := resetTicketKey(ticket)
	lockKey := resetTicketLockKey(ticket)
	lockOwner := uuid.NewString()

	claimResult, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		claimResetTicketScript,
		[]string{ticketKey, lockKey},
		lockOwner,
		resetTicketLockSeconds,
	)
	if err != nil {
		code := ecode.CacheError
		l.Errorw(
			"claim reset ticket failed",
			logx.Field("operation", "reset_password"),
			logx.Field("stage", "claim_ticket"),
			logx.Field("errorCode", code.Int()),
			logx.Field("err", err),
		)
		return resetPasswordResponse(code), nil
	}

	userID, ok := claimResult.(string)
	if !ok || userID == resetTicketMissing || userID == "" {
		code := ecode.PasswordResetTicketInvalid
		l.Infow(
			"reset password rejected",
			logx.Field("operation", "reset_password"),
			logx.Field("reason", "reset_ticket_invalid"),
			logx.Field("errorCode", code.Int()),
		)
		return resetPasswordResponse(code), nil
	}
	if userID == resetTicketBusy {
		code := ecode.PasswordResetTicketUsed
		l.Infow(
			"reset password rejected",
			logx.Field("operation", "reset_password"),
			logx.Field("reason", "reset_ticket_in_use"),
			logx.Field("errorCode", code.Int()),
		)
		return resetPasswordResponse(code), nil
	}

	lockHeld := true
	defer func() {
		if !lockHeld {
			return
		}

		if _, releaseErr := l.svcCtx.Redis.EvalCtx(
			l.ctx,
			releaseResetTicketScript,
			[]string{lockKey},
			lockOwner,
		); releaseErr != nil {
			l.Errorw(
				"release reset ticket lock failed",
				logx.Field("operation", "reset_password"),
				logx.Field("stage", "release_ticket_lock"),
				logx.Field("userId", userID),
				logx.Field("errorCode", ecode.CacheError.Int()),
				logx.Field("err", releaseErr),
			)
		}
	}()

	passwordDigest, err := generatePasswordDigest(req.NewPassword)
	if err != nil {
		code := ecode.PasswordResetFailed
		l.Errorw(
			"generate password digest failed",
			logx.Field("operation", "reset_password"),
			logx.Field("stage", "generate_password_digest"),
			logx.Field("userId", userID),
			logx.Field("errorCode", code.Int()),
			logx.Field("err", err),
		)
		return resetPasswordResponse(code), nil
	}

	resultCode := ecode.Success
	err = l.svcCtx.Sqlconn.TransactCtx(
		l.ctx,
		func(ctx context.Context, session sqlx.Session) error {
			txConn := sqlx.NewSqlConnFromSession(session)
			userModel := model.NewUsersModel(txConn)
			userSessionModel := model.NewUserSessionsModel(txConn)

			user, findErr := userModel.FindOneForUpdate(ctx, userID)
			if errors.Is(findErr, model.ErrNotFound) {
				resultCode = ecode.PasswordResetTicketInvalid
				return nil
			}
			if findErr != nil {
				return fmt.Errorf("find reset user for update: %w", findErr)
			}

			switch user.AccountStatus {
			case 1:
				// 正常状态，继续重置密码。
			case 2:
				resultCode = ecode.AccountPending
				return nil
			default:
				resultCode = ecode.AccountDisabled
				return nil
			}

			now := time.Now()
			user.PasswordDigest = passwordDigest
			user.UpdatedAt = now
			if updateErr := userModel.Update(ctx, user); updateErr != nil {
				return fmt.Errorf("update reset user: %w", updateErr)
			}

			if revokeErr := userSessionModel.RevokeAllByUserID(
				ctx,
				userID,
				now,
				"password_reset",
			); revokeErr != nil {
				return fmt.Errorf("revoke reset user sessions: %w", revokeErr)
			}

			return nil
		},
	)
	if err != nil {
		code := ecode.PasswordResetFailed
		l.Errorw(
			"reset password transaction failed",
			logx.Field("operation", "reset_password"),
			logx.Field("stage", "transaction"),
			logx.Field("userId", userID),
			logx.Field("errorCode", code.Int()),
			logx.Field("err", err),
		)
		return resetPasswordResponse(code), nil
	}

	if resultCode != ecode.Success {
		l.Infow(
			"reset password rejected",
			logx.Field("operation", "reset_password"),
			logx.Field("userId", userID),
			logx.Field("reason", "user_state_rejected"),
			logx.Field("errorCode", resultCode.Int()),
		)
		return resetPasswordResponse(resultCode), nil
	}

	consumeResult, consumeErr := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		consumeResetTicketScript,
		[]string{ticketKey, lockKey},
		lockOwner,
	)
	if consumeErr != nil {
		l.Errorw(
			"consume reset ticket after password update failed",
			logx.Field("operation", "reset_password"),
			logx.Field("stage", "consume_ticket"),
			logx.Field("userId", userID),
			logx.Field("errorCode", ecode.CacheError.Int()),
			logx.Field("err", consumeErr),
		)
	} else if consumed, valid := consumeResult.(int64); !valid || consumed != 1 {
		l.Errorw(
			"reset ticket ownership lost after password update",
			logx.Field("operation", "reset_password"),
			logx.Field("stage", "consume_ticket"),
			logx.Field("userId", userID),
			logx.Field("errorCode", ecode.CacheError.Int()),
		)
	} else {
		lockHeld = false
	}

	l.Infow(
		"password reset completed",
		logx.Field("operation", "reset_password"),
		logx.Field("result", "success"),
		logx.Field("userId", userID),
	)
	return resetPasswordResponse(ecode.Success), nil
}

func resetPasswordResponse(code ecode.Code) *types.ResetPasswordResp {
	return &types.ResetPasswordResp{
		ErrorCode: code.Int(),
		Message:   code.Message(),
	}
}
