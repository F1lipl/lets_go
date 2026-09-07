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

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterLogic) Register(
	req *types.RegisterReq,
) (*types.RegisterResp, error) {
	passwordDigest, err := generatePasswordDigest(req.Password)
	if err != nil {
		code := ecode.InvalidPassword

		return &types.RegisterResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}

	_, err = l.svcCtx.UserModel.FindOneByPhoneNumber(
		l.ctx,
		req.PhoneNumber,
	)

	// 查询成功，说明电话号码已经存在
	if err == nil {
		code := ecode.PhoneAlreadyRegistered

		return &types.RegisterResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}

	// 不是“没有查询到”，说明数据库操作出现问题
	if !errors.Is(err, model.ErrNotFound) {
		code := ecode.DatabaseError
		l.Errorw(
			"query user failed",
			logx.Field("operation", "register"),
			logx.Field("stage", "check_phone_number"),
			logx.Field("errorCode", code.Int()),
			logx.Field("err", err),
		)

		return &types.RegisterResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}

	// 没有查询到用户，可以开始注册
	userID := uuid.NewString()

	user := &model.Users{
		UserId:         userID,
		Username:       req.UserName,
		PhoneNumber:    req.PhoneNumber,
		PasswordDigest: passwordDigest,
		AccountStatus:  1,
		CreatedAt:      time.Now(),
	}

	_, err = l.svcCtx.UserModel.Insert(l.ctx, user)
	if err != nil {
		code := ecode.DatabaseError
		l.Errorw(
			"insert user failed",
			logx.Field("operation", "register"),
			logx.Field("stage", "insert_user"),
			logx.Field("userId", userID),
			logx.Field("errorCode", code.Int()),
			logx.Field("err", err),
		)

		return &types.RegisterResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}

	code := ecode.Success
	l.Infow(
		"user registered",
		logx.Field("operation", "register"),
		logx.Field("result", "success"),
		logx.Field("userId", userID),
	)

	return &types.RegisterResp{
		ErrorCode: code.Int(),
		Message:   code.Message(),
		UserID:    userID,
	}, nil
}
