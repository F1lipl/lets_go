package logic

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
	"userServer/internal/model"
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

func createLoginDevice(
	ctx context.Context,
	deviceModel model.UserDevicesModel,
	userID string,
	info types.DeviceInfo,
	now time.Time,
) (*model.UserDevices, error) {
	device := &model.UserDevices{
		Id:           uuid.NewString(),
		UserId:       userID,
		DeviceName:   info.DeviceName,
		Platform:     info.Platform,
		AppVersion:   info.AppVersion,
		FirstLoginAt: now,
		LastLoginAt:  now,
		Status:       1,
	}

	if _, err := deviceModel.Insert(ctx, device); err != nil {
		return nil, fmt.Errorf("insert device: %w", err)
	}

	return device, nil
}

func validateDeviceInfo(
	device *types.DeviceInfo,
) error {
	if device == nil {
		return errors.New("设备信息不能为空")
	}

	// 去除首尾空格
	device.DeviceID = strings.TrimSpace(device.DeviceID)
	device.DeviceName = strings.TrimSpace(device.DeviceName)
	device.Platform = strings.ToLower(
		strings.TrimSpace(device.Platform),
	)
	device.AppVersion = strings.TrimSpace(device.AppVersion)

	// 首次登录允许不传deviceId；非空时必须是后端签发的标准UUID。
	if device.DeviceID != "" {
		if len(device.DeviceID) != 36 {
			return errors.New("deviceId长度不正确")
		}

		if _, err := uuid.Parse(device.DeviceID); err != nil {
			return errors.New("deviceId格式不正确")
		}
	}

	if device.DeviceName == "" {
		return errors.New("deviceName不能为空")
	}

	if utf8.RuneCountInString(device.DeviceName) > 128 {
		return errors.New("deviceName长度不能超过128")
	}

	if _, ok := supportedPlatforms[device.Platform]; !ok {
		return errors.New("platform不受支持")
	}

	if utf8.RuneCountInString(device.AppVersion) > 32 {
		return errors.New("appVersion长度不能超过32")
	}

	return nil
}

var errDeviceDisabled = errors.New("device disabled")

func resolveLoginDevice(
	ctx context.Context,
	deviceModel model.UserDevicesModel,
	userID string,
	info types.DeviceInfo,
) (*model.UserDevices, error) {
	info.DeviceID = strings.TrimSpace(info.DeviceID)
	now := time.Now()

	// 首次登录，由后端创建设备ID
	if info.DeviceID == "" {
		return createLoginDevice(
			ctx,
			deviceModel,
			userID,
			info,
			now,
		)
	}

	device, err := deviceModel.FindOneByIdUserId(
		ctx,
		info.DeviceID,
		userID,
	)

	if errors.Is(err, model.ErrNotFound) {
		// 收到的ID已经无法对应设备记录，签发新的ID
		return createLoginDevice(
			ctx,
			deviceModel,
			userID,
			info,
			now,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("find device: %w", err)
	}

	if device.Status != 1 {
		return nil, errDeviceDisabled
	}

	device.DeviceName = info.DeviceName
	device.Platform = info.Platform
	device.AppVersion = info.AppVersion
	device.LastLoginAt = now

	if err := deviceModel.Update(ctx, device); err != nil {
		return nil, fmt.Errorf("update device: %w", err)
	}
	return device, nil
}
