package database

import (
	"context"
	"jsfraz/shept-server/models"
	"jsfraz/shept-server/utils"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Check if user exists by username.
//
//	@param username
//	@return bool
//	@return error
func UserExistsByUsername(username string) (bool, error) {
	var count int64
	err := utils.GetSingleton().Db.Model(&models.User{}).Where("username = ?", username).Count(&count).Error
	if err != nil {
		return false, err
	}
	// A duplicate row must still count as taken; == 1 would report a taken name
	// as free.
	return count > 0, nil
}

// Insert user to database and delete invite code.
// Invite code is deleted AFTER successful commit to avoid data loss.
//
//	@param user
//	@param inviteCode
//	@return error
func InsertUser(user *models.User, inviteCode string) error {
	tx := utils.GetSingleton().Db.Begin()
	err := tx.Create(&user).Error
	if err != nil {
		tx.Rollback()
		return err
	}
	err = tx.Commit().Error
	if err != nil {
		return err
	}
	// Delete code from Valkey after successful commit
	return DeleteInviteDataByCode(inviteCode)
}

// Check if admin exists.
//
//	@return bool
//	@return error
func AdminExists() (bool, error) {
	var count int64
	err := utils.GetSingleton().Db.Model(&models.User{}).Where("admin = ?", true).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Check if user exists by ID.
//
//	@param userId
//	@return bool
//	@return error
func UserExistsById(userId string) (bool, error) {
	var count int64
	err := utils.GetSingleton().Db.Model(&models.User{}).Where("id = ?", userId).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Returns user by ID.
//
//	@param userId
//	@return *models.User
//	@return error
func GetUserById(userId string) (*models.User, error) {
	toDelete, err := GetAllUsersToDelete()
	if err != nil {
		return nil, err
	}
	var user models.User
	if len(toDelete) != 0 {
		err = utils.GetSingleton().Db.Where("id = ? AND id NOT IN ?", userId, toDelete).First(&user).Error
	} else {
		err = utils.GetSingleton().Db.Where("id = ?", userId).First(&user).Error
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Get all users except the user.
//
//	@param userId
//	@return *[]models.User
//	@return error
func GetAllUsersExceptUser(userId string) (*[]models.User, error) {
	toDelete, err := GetAllUsersToDelete()
	if err != nil {
		return nil, err
	}
	var users []models.User = []models.User{}
	if len(toDelete) != 0 {
		err = utils.GetSingleton().Db.Where("id != ? AND id NOT IN ?", userId, toDelete).Find(&users).Error
	} else {
		err = utils.GetSingleton().Db.Where("id != ?", userId).Find(&users).Error
	}
	if err != nil {
		return nil, err
	}
	return &users, nil
}

// Delete user by ID.
//
//	@param userId
//	@return error
func DeleteUserById(userId string) error {
	return utils.GetSingleton().Db.Where("id = ?", userId).Delete(&models.User{}).Error
}

// Checks if user with given ID is admin.
//
//	@param userId
//	@return bool
//	@return error
func IsAdmin(userId string) (bool, error) {
	var isAdmin bool
	err := utils.GetSingleton().Db.Model(&models.User{}).Select("admin").Where("id = ?", userId).Scan(&isAdmin).Error
	if err != nil {
		return false, err
	}
	return isAdmin, nil
}

// Get user public key by ID.
//
//	@param userId
//	@return string
//	@return error
func GetUserPublicKey(userId string) (string, error) {
	var publicKey string
	err := utils.GetSingleton().Db.Model(&models.User{}).Select("public_key").Where("id = ?", userId).Scan(&publicKey).Error
	if err != nil {
		return "", err
	}
	return publicKey, nil
}

// Escape LIKE wildcards so user input is matched literally.
func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	return strings.ReplaceAll(value, `_`, `\_`)
}

// Self-chat is a debug-only feature.
func searchExcludesSelf() bool {
	return utils.GetSingleton().Config.GinMode == "release"
}

// Search users by username. Return at most 50, except the user.
//
//	@param username
//	@param userId
//	@return *[]models.User
//	@return error
func SearchUsersByUsername(username string, userId string) (*[]models.User, error) {
	toDelete, err := GetAllUsersToDelete()
	if err != nil {
		return nil, err
	}
	var users []models.User

	query := utils.GetSingleton().Db.Model(&models.User{}).
		Where("username LIKE ? ESCAPE '\\'", "%"+escapeLike(username)+"%")
	if searchExcludesSelf() {
		query = query.Where("id != ?", userId)
	}
	if len(toDelete) != 0 {
		query = query.Where("id NOT IN ?", toDelete)
	}
	if err = query.Order("username").Limit(50).Find(&users).Error; err != nil {
		return nil, err
	}
	return &users, nil
}

// Push users to delete SET and clean up their messages.
//
//	@param ids
//	@return error
func PushUsersToDelete(ids []string) error {
	client := utils.GetSingleton().Valkey
	// Add all user IDs to the delete SET
	if err := client.Do(context.Background(), client.B().Sadd().Key("del:users").Member(ids...).Build()).Error(); err != nil {
		return err
	}
	invalidateToDeleteCache()
	// The archive goes with the account: the ciphertext is useless without a
	// recipient, and keeping it would outlive the deletion.
	for _, userId := range ids {
		if err := DeleteUserMessages(userId); err != nil {
			return err
		}
	}
	return nil
}

// Get all users to delete from SET.
//
// The answer is cached for a couple of seconds because it is read on every user
// lookup, and a stale entry only delays a deletion confirmation.
//
//	@return []string
//	@return error
func GetAllUsersToDelete() ([]string, error) {
	toDeleteMu.Lock()
	if time.Now().Before(toDeleteUntil) {
		cached := toDeleteCache
		toDeleteMu.Unlock()
		return cached, nil
	}
	toDeleteMu.Unlock()

	client := utils.GetSingleton().Valkey
	members, err := client.Do(context.Background(), client.B().Smembers().Key("del:users").Build()).AsStrSlice()
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(members))
	for _, m := range members {
		if _, err := uuid.Parse(m); err != nil {
			continue
		}
		result = append(result, m)
	}

	toDeleteMu.Lock()
	toDeleteCache = result
	toDeleteUntil = time.Now().Add(toDeleteCacheTtl)
	toDeleteMu.Unlock()
	return result, nil
}

// toDeleteCacheTtl bounds how long a pending deletion may go unnoticed by the
// read-only helpers.
const toDeleteCacheTtl = 2 * time.Second

var (
	toDeleteMu    sync.Mutex
	toDeleteCache []string
	toDeleteUntil time.Time
)

// invalidateToDeleteCache makes the next read go to Valkey.
func invalidateToDeleteCache() {
	toDeleteMu.Lock()
	toDeleteCache = nil
	toDeleteUntil = time.Time{}
	toDeleteMu.Unlock()
}

// Check if user is in to delete SET (O(1) lookup).
//
//	@param userId
//	@return bool
//	@return error
func WillUserBeDeleted(userId string) (bool, error) {
	client := utils.GetSingleton().Valkey
	result, err := client.Do(context.Background(), client.B().Sismember().Key("del:users").Member(userId).Build()).AsBool()
	if err != nil {
		return false, err
	}
	return result, nil
}

// Remove deleted user's ID from delete SET.
//
//	@param userId
//	@return error
func RemoveDeletedUserId(userId string) error {
	client := utils.GetSingleton().Valkey
	err := client.Do(context.Background(), client.B().Srem().Key("del:users").Member(userId).Build()).Error()
	invalidateToDeleteCache()
	return err
}

// GetUserRetentionDays returns the archive retention of a user in days.
//
//	@param userId
//	@return int
//	@return error
func GetUserRetentionDays(userId string) (int, error) {
	var days int
	err := utils.GetSingleton().Db.Model(&models.User{}).
		Select("retention_days").Where("id = ?", userId).Scan(&days).Error
	if err != nil {
		return 0, err
	}
	if days == 0 {
		days = models.DefaultRetentionDays
	}
	return days, nil
}

// SetUserRetentionDays stores the archive retention of a user.
//
//	@param userId
//	@param days
//	@return error
func SetUserRetentionDays(userId string, days int) error {
	return utils.GetSingleton().Db.Model(&models.User{}).
		Where("id = ?", userId).Update("retention_days", days).Error
}

// GetRetentionDaysOf returns the retention both participants agreed to: the
// shorter setting wins, because both accounts can read the message.
//
//	@param firstId
//	@param secondId
//	@return int
//	@return error
func GetRetentionDaysOf(firstId string, secondId string) (int, error) {
	var rows []struct {
		Id            string
		RetentionDays int
	}
	err := utils.GetSingleton().Db.Model(&models.User{}).
		Select("id, retention_days").Where("id IN ?", []string{firstId, secondId}).
		Scan(&rows).Error
	if err != nil {
		return 0, err
	}
	days := map[string]int{}
	for _, row := range rows {
		value := row.RetentionDays
		if value == 0 {
			value = models.DefaultRetentionDays
		}
		days[row.Id] = value
	}
	return ShortestRetention(days[firstId], days[secondId]), nil
}
