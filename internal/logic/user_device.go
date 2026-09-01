package logic

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"
	"userServer/internal/model"
	"userServer/internal/svc"

	"userServer/internal/types"

	"github.com/google/uuid"
)

var supportedPlatforms = map[string]struct{}{
	"web":     {},
	"windows": {},
	"macos":   {},
	"linux":   {},
	"android": {},
	"ios":     {},
}

func validateDeviceInfo(
	device types.DeviceInfo,
) (types.DeviceInfo, error) {
	// 去除首尾空格
	device.DeviceID = strings.TrimSpace(device.DeviceID)
	device.DeviceName = strings.TrimSpace(device.DeviceName)
	device.Platform = strings.ToLower(
		strings.TrimSpace(device.Platform),
	)
	device.AppVersion = strings.TrimSpace(device.AppVersion)

	if device.DeviceID == "" {
		return types.DeviceInfo{}, errors.New("deviceId不能为空")
	}

	if len(device.DeviceID) > 64 {
		return types.DeviceInfo{}, errors.New("deviceId长度不能超过64")
	}

	// 如果约定客户端必须使用UUID作为deviceId
	if _, err := uuid.Parse(device.DeviceID); err != nil {
		return types.DeviceInfo{}, errors.New("deviceId格式不正确")
	}

	if device.DeviceName == "" {
		return types.DeviceInfo{}, errors.New("deviceName不能为空")
	}

	if utf8.RuneCountInString(device.DeviceName) > 128 {
		return types.DeviceInfo{}, errors.New("deviceName长度不能超过128")
	}

	if _, ok := supportedPlatforms[device.Platform]; !ok {
		return types.DeviceInfo{}, errors.New("platform不受支持")
	}

	if utf8.RuneCountInString(device.AppVersion) > 32 {
		return types.DeviceInfo{}, errors.New("appVersion长度不能超过32")
	}

	return device, nil
}

func saveLoginDevice(
	ctx context.Context,
	svcCtx *svc.ServiceContext,
	userID string,
	device types.DeviceInfo,
) error {
	return svcCtx.UserDeviceModel.UpsertLogin(
		ctx,
		&model.UserDevices{
			Id:               uuid.NewString(),
			UserId:           userID,
			ClientInstanceId: device.DeviceID,
			DeviceName:       device.DeviceName,
			Platform:         device.Platform,
			AppVersion:       device.AppVersion,
		},
	)
}
