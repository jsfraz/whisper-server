package database

import (
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
)

// Check if user exists by username.
//
//	@param username
//	@return bool
//	@return error
func UserExistsByUsername(username string) (bool, error) {
	var count int64
	err := utils.GetSingleton().Postgres.Model(&models.User{}).Where("username = ?", username).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

// Insert user to database and delete invite code.
//
//	@param user
//	@param inviteCode
//	@return error
func InsertUser(user *models.User, inviteCode string) error {
	tx := utils.GetSingleton().Postgres.Begin()
	err := tx.Create(&user).Error
	if err != nil {
		tx.Rollback()
		return err
	}
	// Delete code from Valekey
	err = DeleteInviteDataByCode(inviteCode)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

// Check if admin exists.
//
//	@return bool
//	@return error
func AdminExists() (bool, error) {
	var count int64
	err := utils.GetSingleton().Postgres.Model(&models.User{}).Where("admin = ?", true).Count(&count).Error
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
	err := utils.GetSingleton().Postgres.Model(&models.User{}).Where("id = ?", userId).Count(&count).Error
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
	var user models.User
	err := utils.GetSingleton().Postgres.Where("id = ?", userId).First(&user).Error
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
	var users []models.User = []models.User{}
	err := utils.GetSingleton().Postgres.Where("id != ?", userId).Find(&users).Error
	if err != nil {
		return nil, err
	}
	return &users, nil
}

// Delete user by ID.
//
//	@param userId
//	@return error
func DeleteUsersById(userId []uint64) error {
	return utils.GetSingleton().Postgres.Where("id IN ?", userId).Delete(&models.User{}).Error
}

// Checks if user with given ID is admin.
//
//	@param userId
//	@return bool
//	@return error
func IsAdmin(userId uint64) (bool, error) {
	var isAdmin bool
	err := utils.GetSingleton().Postgres.Model(&models.User{}).Select("admin").Where("id = ?", userId).Scan(&isAdmin).Error
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
	err := utils.GetSingleton().Postgres.Model(&models.User{}).Select("public_key").Where("id = ?", userId).Scan(&publicKey).Error
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
	var users []models.User
	err := utils.GetSingleton().Postgres.Where("username LIKE ? AND id != ?", "%"+username+"%", userId).Find(&users).Error
	if err != nil {
		return nil, err
	}
	return &users, nil
}
