// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"
	"errors"
	"time"
	"userServer/internal/ecode"
	"userServer/internal/model"

	"userServer/internal/svc"
	"userServer/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ChangePasswordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewChangePasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChangePasswordLogic {
	return &ChangePasswordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ChangePasswordLogic) ChangePassword(req *types.ChangePasswordReq) (resp *types.ChangePasswordResp, err error) {
	userId, sessionId, err := getAccessClaims(l.ctx)
	if err != nil {
		code := ecode.AccessTokenInvalid
		return &types.ChangePasswordResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}

	err = l.svcCtx.Sqlconn.TransactCtx(l.ctx,
		func(ctx context.Context, session sqlx.Session) error {
			txConn := sqlx.NewSqlConnFromSession(session)
			userSessionModel := model.NewUserSessionsModel(txConn)
			userModel := model.NewUsersModel(txConn)
			user, err := userModel.FindOneForUpdate(ctx, userId)
			if err != nil {
				return err
			}

			// 检查账号状态
			if user.AccountStatus != 1 {
				return errors.New(ecode.AccountDisabled.String())
			}
			//验证一下密码
			if comparePassword(user.PasswordDigest, req.CurrentPassword) {
				return errors.New(ecode.InvalidPassword.String())
			}
			now := time.Now()
			revokeReason := "用户修改密码"

			newPassword, err := generatePasswordDigest(req.CurrentPassword)
			if err != nil {
				return errors.New("generate password digest fail")
			}
			user.PasswordDigest = newPassword
			user.UpdatedAt = now
			err = userModel.Update(ctx, user)
			if err != nil {
				return err
			}
			err = userSessionModel.RevokeOtherByUserID(ctx, user.UserId, sessionId, now, revokeReason)
			if err != nil {
				return err
			}
			return nil
		})
	if err != nil {
		code := ecode.PasswordChangeFailed
		l.Infow("change password error", logx.Field("err", err))
		return &types.ChangePasswordResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}
	return &types.ChangePasswordResp{
		ErrorCode: ecode.Success.Int(),
		Message:   ecode.Success.Message(),
	}, nil
}
