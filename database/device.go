package database

import (
	"errors"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"sync"
	"time"

	"gorm.io/gorm/clause"
)

// DeviceActive caches "this device is alive" answers: every authenticated
// request asks, and a stale entry only delays a revocation by a moment.
const deviceActiveTtl = 5 * time.Second

var (
	deviceActiveMu    sync.Mutex
	deviceActiveCache = map[string]time.Time{}
)

func deviceActiveKey(userId string, deviceId string) string {
	return userId + "|" + deviceId
}

// InvalidateDeviceCache drops the cached answer after a revocation change, so
// the next request sees it immediately.
//
//	@param userId
//	@param deviceId
func InvalidateDeviceCache(userId string, deviceId string) {
	deviceActiveMu.Lock()
	delete(deviceActiveCache, deviceActiveKey(userId, deviceId))
	deviceActiveMu.Unlock()
}

// RegisterDevice records a login from a device. An unknown device id is
// inserted, a known one is touched, a revoked one is refused.
//
//	@param userId
//	@param deviceId
//	@param label
//	@param platform
//	@return bool created
//	@return bool revoked
//	@return error
func RegisterDevice(userId string, deviceId string, label string, platform string) (bool, bool, error) {
	db := utils.GetSingleton().Db
	var device models.Device
	err := db.Where("id = ?", deviceId).First(&device).Error
	if err == nil {
		// The id belongs to another account: a device id is not a shared name.
		if device.UserId != userId {
			return false, false, errors.New(utils.ErrorInvalidDevice)
		}
		if device.RevokedAt != nil {
			return false, true, nil
		}
		err = db.Model(&models.Device{}).Where("id = ?", deviceId).
			Updates(map[string]interface{}{
				"label":        label,
				"platform":     platform,
				"last_seen_at": time.Now(),
			}).Error
		if err != nil {
			return false, false, err
		}
		InvalidateDeviceCache(userId, deviceId)
		return false, false, nil
	}
	now := time.Now()
	created := models.Device{
		Id:         deviceId,
		UserId:     userId,
		Label:      label,
		Platform:   platform,
		CreatedAt:  now,
		LastSeenAt: now,
	}
	// Two logins of one installation can race; the loser just touches the row.
	result := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&created)
	if result.Error != nil {
		return false, false, result.Error
	}
	if result.RowsAffected == 0 {
		var existing models.Device
		if err := db.Where("id = ?", deviceId).First(&existing).Error; err != nil {
			return false, false, err
		}
		if existing.UserId != userId {
			return false, false, errors.New(utils.ErrorInvalidDevice)
		}
		return false, existing.RevokedAt != nil, nil
	}
	return true, false, nil
}

// IsDeviceActive reports whether a device exists for the user and is not
// revoked. The answer is cached for a few seconds.
//
//	@param userId
//	@param deviceId
//	@return bool
//	@return error
func IsDeviceActive(userId string, deviceId string) (bool, error) {
	key := deviceActiveKey(userId, deviceId)
	deviceActiveMu.Lock()
	until, cached := deviceActiveCache[key]
	deviceActiveMu.Unlock()
	if cached && time.Now().Before(until) {
		return true, nil
	}
	var count int64
	err := utils.GetSingleton().Db.Model(&models.Device{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL", deviceId, userId).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	if count > 0 {
		deviceActiveMu.Lock()
		deviceActiveCache[key] = time.Now().Add(deviceActiveTtl)
		deviceActiveMu.Unlock()
	}
	return count > 0, nil
}

// GetDevices returns the devices of an account, newest last seen first.
//
//	@param userId
//	@return *[]models.Device
//	@return error
func GetDevices(userId string) (*[]models.Device, error) {
	devices := []models.Device{}
	err := utils.GetSingleton().Db.Where("user_id = ?", userId).
		Order("last_seen_at DESC").Find(&devices).Error
	if err != nil {
		return nil, err
	}
	return &devices, nil
}

// RevokeDevice locks one device out. The caller also revokes its refresh
// sessions and drops its live socket, so the lockout is immediate.
//
//	@param userId
//	@param deviceId
//	@return error
func RevokeDevice(userId string, deviceId string) error {
	now := time.Now()
	result := utils.GetSingleton().Db.Model(&models.Device{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL", deviceId, userId).
		Update("revoked_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New(utils.ErrorDeviceNotFound)
	}
	InvalidateDeviceCache(userId, deviceId)
	return nil
}

// UnrevokeDevice lets a revoked device sign in again. Only a live device of the
// same account may do this.
//
//	@param userId
//	@param deviceId
//	@return error
func UnrevokeDevice(userId string, deviceId string) error {
	result := utils.GetSingleton().Db.Model(&models.Device{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NOT NULL", deviceId, userId).
		Updates(map[string]interface{}{"revoked_at": nil})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New(utils.ErrorDeviceNotFound)
	}
	InvalidateDeviceCache(userId, deviceId)
	return nil
}

// DeleteUserDevices drops the registry rows of a deleted account.
//
//	@param userId
//	@return error
func DeleteUserDevices(userId string) error {
	return utils.GetSingleton().Db.Where("user_id = ?", userId).Delete(&models.Device{}).Error
}
