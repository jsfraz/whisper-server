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
	err := utils.GetSingleton().PostgresDb.Model(&models.User{}).Where("username = ?", username).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

/*
// Check if mail is taken.
//
//	@param mail
//	@return bool
//	@return error
func IsMailTaken(mail string) (bool, error) {
	var count int64
	err := utils.GetSingleton().PostgresDb.Model(&models.User{}).Where("mail = ?", mail).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count == 1, nil
}
*/

// Insert user to database and delete invite code.
//
//	@param user
//	@param inviteCode
//	@return error
func InsertUser(user models.User, inviteCode string) error {
	tx := utils.GetSingleton().PostgresDb.Begin()
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
	err := utils.GetSingleton().PostgresDb.Model(&models.User{}).Where("admin = ?", true).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

/*
// Verify user.
//
//	@param verificationCode
//	@return *models.User
//	@return error
func VerifyUser(verificationCode string) (*models.User, error) {
	// Find the user
	var user models.User
	err := utils.GetSingleton().PostgresDb.Model(&models.User{}).Where("verification_code = ? AND is_verified = ?", verificationCode, false).Attrs(models.User{}).First(&user).Error
	if err != nil {
		return nil, err
	}
	// Update user
	user.IsVerified = true
	return &user, utils.GetSingleton().PostgresDb.Save(&user).Error
}

// Returns user by username.
//
//	@param username
//	@return *models.User
//	@return error
func GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := utils.GetSingleton().PostgresDb.Model(&models.User{}).Where("username = ?", username).Attrs(models.User{}).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
*/

// Check if user exists by ID.
//
//	@param userId
//	@return bool
//	@return error
func UserExistsById(userId uint64) (bool, error) {
	var count int64
	err := utils.GetSingleton().PostgresDb.Model(&models.User{}).Where("id = ?", userId).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

/*
// Returns user by ID.
//
//	@param userId
//	@return *models.User
//	@return error
func GetUserById(userId uint64) (*models.User, error) {
	var user models.User
	err := utils.GetSingleton().PostgresDb.Model(&models.User{}).Where("id = ?", userId).Attrs(models.User{}).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
*/
