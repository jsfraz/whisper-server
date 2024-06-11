package errors

import "errors"

type Status int

const (
	UsernameTaken      Status = 452
	MailTaken          Status = 453
	VerificationFailed Status = 454
	UserNotVerified    Status = 455
	UserDoesntExist    Status = 456

	BadRequest          Status = 400
	Unauthorized        Status = 401
	InternalServerError Status = 500
)

// Gets status message
func (status Status) GetMessage() string {
	message := ""
	switch status {
	// Custom errors
	case UsernameTaken:
		message = "Username already taken."
	case MailTaken:
		message = "E-mail already taken."
	case VerificationFailed:
		message = "Account verification failed."
	case UserNotVerified:
		message = "User is not verified."
	case UserDoesntExist:
		message = "User doesn't exist."
	// Common errors
	case BadRequest:
		message = "Bad request."
	case Unauthorized:
		message = "Unauthorized."
	case InternalServerError:
		message = "Internal server error."
	}

	return message
}

// Gets status code
func (status Status) GetCode() int {
	return int(status)
}

// Returns error
func (status Status) GetError() error {
	return errors.New(status.GetMessage())
}
