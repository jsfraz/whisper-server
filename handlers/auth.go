package handlers

import (
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/errors"
	"jsfraz/whisper-server/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Creates new user.
//
//	@param c
//	@param register
//	@return error
func RegisterUser(c *gin.Context, register *models.Register) error {
	// Check if username is taken
	usernameTaken, _ := database.IsUsernameTaken(register.Username)
	if usernameTaken {
		status := errors.UsernameTaken
		c.AbortWithStatus(status.GetCode())
		return status.GetError()
	}
	// Check if mail is taken
	mailTaken, err := database.IsMailTaken(register.Mail)
	if err != nil {
		c.AbortWithStatus(errors.InternalServerError.GetCode())
		return err
	}
	if mailTaken {
		status := errors.MailTaken
		c.AbortWithStatus(status.GetCode())
		return status.GetError()
	}
	// Create new user
	newUser, err := models.NewUser(*register)
	if err != nil {
		c.AbortWithStatus(errors.InternalServerError.GetCode())
		return err
	}
	// Insert to database
	err = database.InsertUser(*newUser)
	if err != nil {
		c.AbortWithStatus(errors.InternalServerError.GetCode())
		return err
	}

	return nil
}

// Verifies user account.
//
//	@param c
//	@param verify
//	@return error
func VerifyUser(c *gin.Context, verify *models.Verify) error {
	// Verify user
	user, err := database.VerifyUser(verify.Code)
	if err != nil && err != gorm.ErrRecordNotFound {
		c.AbortWithStatus(errors.InternalServerError.GetCode())
		return err
	}
	// If user is empty
	if user == nil {
		status := errors.VerificationFailed
		c.AbortWithStatus(status.GetCode())
		return status.GetError()
	}
	// TODO send mail
	/*
		utils.SendVerifiedMail(user.Mail, user.Username)
		// TODO log error
	*/

	return nil
}

/*
// User login
func LoginUser(c *gin.Context, login *models.Login) (*models.AuthResponse, error) {
	// get user from database
	user, verified, err := database.GetUserLoginDataByUsername(login.Username)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.AbortWithStatus(errors.Unauthorized.GetCode())
			// TODO log error
			return nil, err
		} else {
			c.AbortWithStatus(errors.InternalServerError.GetCode())
			// TODO log error
			return nil, err
		}
	}
	// check if user is verified
	if !verified {
		status := errors.UserNotVerified
		c.AbortWithStatus(status.GetCode())
		return nil, status.GetError()
	}
	// check hash
	hashBytes, _ := base64.StdEncoding.DecodeString(user.PasswordHash)
	err = bcrypt.CompareHashAndPassword(hashBytes, []byte(login.Password))
	if err != nil {
		// incorrect password
		if err == bcrypt.ErrMismatchedHashAndPassword {
			c.AbortWithStatus(errors.Unauthorized.GetCode())
			// TODO log error
			return nil, err
		} else {
			// internal error
			c.AbortWithStatus(errors.InternalServerError.GetCode())
			// TODO log error
			return nil, err
		}
	}
	// generate access token
	accessToken, err := utils.GenerateToken(user.Id, os.Getenv("ACCESS_TOKEN_LIFESPAN"), os.Getenv("ACCESS_TOKEN_SECRET"))
	if err != nil {
		c.AbortWithStatus(errors.InternalServerError.GetCode())
		// TODO log error
		return nil, err
	}
	// generate refresh token
	refreshToken, err := utils.GenerateToken(user.Id, os.Getenv("REFRESH_TOKEN_LIFESPAN"), os.Getenv("REFRESH_TOKEN_SECRET"))
	if err != nil {
		c.AbortWithStatus(errors.InternalServerError.GetCode())
		// TODO log error
		return nil, err
	}
	return models.NewAuth(accessToken, refreshToken, user), nil
}

// Refresh access token
func RefreshUserAccessToken(c *gin.Context, refresh *models.Refresh) (*models.RefreshResponse, error) {
	// validate token and get user id
	userId, err := utils.TokenValid(refresh.RefreshToken, os.Getenv("REFRESH_TOKEN_SECRET"))
	if err != nil {
		c.AbortWithStatus(errors.Unauthorized.GetCode())
		return nil, err
	}
	// check if user exists
	exists, err := database.UserExists(userId)
	if err != nil {
		c.AbortWithStatus(errors.InternalServerError.GetCode())
		// TODO log error
		return nil, err
	}
	if exists {
		// generate access token
		accessToken, err := utils.GenerateToken(userId, os.Getenv("ACCESS_TOKEN_LIFESPAN"), os.Getenv("ACCESS_TOKEN_SECRET"))
		if err != nil {
			c.AbortWithStatus(errors.InternalServerError.GetCode())
			// TODO log error
			return nil, err
		}
		return models.NewRefreshResponse(accessToken), nil
	} else {
		// unauthorized
		c.AbortWithStatus(errors.Unauthorized.GetCode())
		return nil, err
	}
}
*/
