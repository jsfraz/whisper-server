package database

import (
	"context"
	"fmt"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"strconv"
)

// Check if user exists by username.
//
//	@param username
//	@return bool
//	@return error
func UserExistsByUsername(username string) (bool, error) {
	var count int64
	err := utils.GetSingleton().Sqlite.Model(&models.User{}).Where("username = ?", username).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

// Insert user to database and delete invite code.
// Invite code is deleted AFTER successful commit to avoid data loss.
//
//	@param user
//	@param inviteCode
//	@return error
func InsertUser(user *models.User, inviteCode string) error {
	tx := utils.GetSingleton().Sqlite.Begin()
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
	err := utils.GetSingleton().Sqlite.Model(&models.User{}).Where("admin = ?", true).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

// Check if user exists by ID.
//
//	@param userId
//	@return bool
//	@return error
func UserExistsById(userId uint64) (bool, error) {
	var count int64
	err := utils.GetSingleton().Sqlite.Model(&models.User{}).Where("id = ?", userId).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

// Returns user by ID.
//
//	@param userId
//	@return *models.User
//	@return error
func GetUserById(userId uint64) (*models.User, error) {
	toDelete, err := GetAllUsersToDelete()
	if err != nil {
		return nil, err
	}
	var user models.User
	if len(toDelete) != 0 {
		err = utils.GetSingleton().Sqlite.Where("id = ? AND id NOT IN ?", userId, toDelete).First(&user).Error
	} else {
		err = utils.GetSingleton().Sqlite.Where("id = ?", userId).First(&user).Error
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
func GetAllUsersExceptUser(userId uint64) (*[]models.User, error) {
	toDelete, err := GetAllUsersToDelete()
	if err != nil {
		return nil, err
	}
	var users []models.User = []models.User{}
	if len(toDelete) != 0 {
		err = utils.GetSingleton().Sqlite.Where("id != ? AND id NOT IN ?", userId, toDelete).Find(&users).Error
	} else {
		err = utils.GetSingleton().Sqlite.Where("id != ?", userId).Find(&users).Error
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
func DeleteUserById(userId uint64) error {
	return utils.GetSingleton().Sqlite.Where("id = ?", userId).Delete(&models.User{}).Error
}

// Checks if user with given ID is admin.
//
//	@param userId
//	@return bool
//	@return error
func IsAdmin(userId uint64) (bool, error) {
	var isAdmin bool
	err := utils.GetSingleton().Sqlite.Model(&models.User{}).Select("admin").Where("id = ?", userId).Scan(&isAdmin).Error
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
func GetUserPublicKey(userId uint64) (string, error) {
	var publicKey string
	err := utils.GetSingleton().Sqlite.Model(&models.User{}).Select("public_key").Where("id = ?", userId).Scan(&publicKey).Error
	if err != nil {
		return "", err
	}
	return publicKey, nil
}

// Search users by username. Return all except the user.
//
//	@param username
//	@param userId
//	@return *[]models.User
//	@return error
func SearchUsersByUsername(username string, userId uint64) (*[]models.User, error) {
	toDelete, err := GetAllUsersToDelete()
	if err != nil {
		return nil, err
	}
	var users []models.User

	skipUser := true
	if utils.GetSingleton().Config.GinMode != "release" {
		skipUser = false
	}

	if len(toDelete) != 0 {
		if skipUser {
			err = utils.GetSingleton().Sqlite.Where("username LIKE ? AND id != ? AND id NOT IN ?", "%"+username+"%", userId, toDelete).Find(&users).Error
		} else {
			err = utils.GetSingleton().Sqlite.Where("username LIKE ? AND id NOT IN ?", "%"+username+"%", toDelete).Find(&users).Error
		}
	} else {
		if skipUser {
			err = utils.GetSingleton().Sqlite.Where("username LIKE ? AND id != ?", "%"+username+"%", userId).Find(&users).Error
		} else {
			err = utils.GetSingleton().Sqlite.Where("username LIKE ?", "%"+username+"%").Find(&users).Error
		}
	}
	if err != nil {
		return nil, err
	}
	return &users, nil
}

// Push users to delete SET and clean up their messages.
//
//	@param ids
//	@return error
func PushUsersToDelete(ids []uint64) error {
	client := utils.GetSingleton().Valkey
	// Add all user IDs to the delete SET
	members := make([]string, len(ids))
	for i, userId := range ids {
		members[i] = strconv.FormatUint(userId, 10)
	}
	if err := client.Do(context.Background(), client.B().Sadd().Key("del:users").Member(members...).Build()).Error(); err != nil {
		return err
	}
	// Delete messages for these users
	for _, userId := range ids {
		// Use SCAN to find keys matching pattern
		pattern := fmt.Sprintf("msg:%d_*", userId)
		keys := []string{}

		// Scan for matching keys
		iter := client.B().Scan().Cursor(0).Match(pattern).Count(100).Build()
		for {
			result, err := client.Do(context.Background(), iter).AsScanEntry()
			if err != nil {
				return err
			}
			keys = append(keys, result.Elements...)
			if result.Cursor == 0 {
				break
			}
			iter = client.B().Scan().Cursor(result.Cursor).Match(pattern).Count(100).Build()
		}

		// Delete found keys in batches
		if len(keys) > 0 {
			if err := client.Do(context.Background(), client.B().Del().Key(keys...).Build()).Error(); err != nil {
				return err
			}
		}
	}
	return nil
}

// Get all users to delete from SET.
//
//	@return []uint64
//	@return error
func GetAllUsersToDelete() ([]uint64, error) {
	client := utils.GetSingleton().Valkey
	members, err := client.Do(context.Background(), client.B().Smembers().Key("del:users").Build()).AsStrSlice()
	if err != nil {
		return nil, err
	}
	result := make([]uint64, 0, len(members))
	for _, m := range members {
		id, err := strconv.ParseUint(m, 10, 64)
		if err != nil {
			continue
		}
		result = append(result, id)
	}
	return result, nil
}

// Check if user is in to delete SET (O(1) lookup).
//
//	@param userId
//	@return bool
//	@return error
func WillUserBeDeleted(userId uint64) (bool, error) {
	client := utils.GetSingleton().Valkey
	result, err := client.Do(context.Background(), client.B().Sismember().Key("del:users").Member(strconv.FormatUint(userId, 10)).Build()).AsBool()
	if err != nil {
		return false, err
	}
	return result, nil
}

// Remove deleted user's ID from delete SET.
//
//	@param userId
//	@return error
func RemoveDeletedUserId(userId uint64) error {
	client := utils.GetSingleton().Valkey
	return client.Do(context.Background(), client.B().Srem().Key("del:users").Member(strconv.FormatUint(userId, 10)).Build()).Error()
}
