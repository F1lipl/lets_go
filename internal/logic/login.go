package logic

import (
	"context"
	"fmt"
	"userServer/internal/model"
	"userServer/internal/svc"
	"userServer/internal/types"

	"github.com/google/uuid"
)

//到这里说明，user是存在并且状态是正常的，deviceInfo是正常的,然后开启事务，更新设备表，更新用户表，创建一个session，然后签发token，和refreshToken

func finalizeLogin(SvcCtx *svc.ServiceContext, ctx2 context.Context, user *model.Users, info *types.DeviceInfo) (*types.LoginResp, error) {
	sessionID := uuid.NewString()
	accessToken, err := generateAccessToken(SvcCtx.Config.Auth.AccessSecret, SvcCtx.Config.Auth.AccessExpire, user.UserId, sessionID)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}
	
}
