package database

import (
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
)

// Check if username is taken.
//
//	@param username
//	@return bool
//	@return error
func IsUsernameTaken(username string) (bool, error) {
	var count int64
	err := utils.GetSingleton().PostgresDb.Model(&models.User{}).Where("username = ?", username).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

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

// Insert user to database.
//
//	@param user
//	@return error
func InsertUser(user models.User) error {
	err := utils.GetSingleton().PostgresDb.Create(&user).Error
	if err != nil {
		return err
	}
	return nil
}

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
	err := utils.GetSingleton().PostgresDb.Model(&models.User{}).Where("username = ?", username).Attrs(models.User{}).FirstOrInit(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
