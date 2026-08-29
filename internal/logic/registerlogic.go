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
		l.Errorf(
			"query user failed, phoneNumber=%s, err=%v",
			req.PhoneNumber,
			err,
		)

		code := ecode.DatabaseError

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
	}

	_, err = l.svcCtx.UserModel.Insert(l.ctx, user)
	if err != nil {
		l.Errorf(
			"insert user failed, phoneNumber=%s, err=%v",
			req.PhoneNumber,
			err,
		)

		code := ecode.DatabaseError

		return &types.RegisterResp{
			ErrorCode: code.Int(),
			Message:   code.Message(),
		}, nil
	}

	code := ecode.Success

	return &types.RegisterResp{
		ErrorCode: code.Int(),
		Message:   code.Message(),
		UserID:    userID,
	}, nil
}
