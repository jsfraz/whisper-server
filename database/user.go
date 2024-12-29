package database

import (
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"log"
	"strconv"

	"github.com/aymerick/raymond"
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

/*
// Returns user by username.
//
//	@param username
//	@return *models.User
//	@return error
func GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := utils.GetSingleton().Postgres.Model(&models.User{}).Where("username = ?", username).Attrs(models.User{}).First(&user).Error
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
	err := utils.GetSingleton().Postgres.Model(&models.User{}).Where("id = ?", userId).Attrs(models.User{}).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Subscribe for new users and send mail.
func SubscribeNewUsers() {
	PostgresTriggerListener(utils.GetSingleton().GetPostgresConnStr(), "create_user_channel", func(s string) {
		// Get user
		newUserId, _ := strconv.ParseUint(s, 10, 64)
		newUser, err := GetUserById(newUserId)
		if err != nil {
			log.Println(err)
			return
		}
		// Load template
		template, err := utils.ReadFile("./mailTemplates/userCreated.hbs")
		if err != nil {
			log.Println(err)
			return
		}
		// Render template
		content, err := raymond.Render(
			*template,
			map[string]string{
				"username": newUser.Username,
				"footer":   utils.GetMailFooter(),
			},
		)
		// Send mail
		err = utils.SendMail(newUser.Mail, "Account successfully created", content)
		if err != nil {
			log.Println(err)
			return
		}
	})
}

// Get all users except the user.
//
//	@param userId
//	@return *[]models.User
//	@return error
func GetAllUsersExceptUser(userId uint64) (*[]models.User, error) {
	var users []models.User = []models.User{}
	err := utils.GetSingleton().Postgres.Model(&models.User{}).Find(&users).Where("id != ?", userId).Error
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
