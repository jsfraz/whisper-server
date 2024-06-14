package errors

import "errors"

type Status string

const (
	UsernameTaken      Status = "username already taken"
	MailTaken          Status = "mail already taken"
	VerificationFailed Status = "account verification failed"
	UserNotVerified    Status = "user is not verified"
	UserDoesntExist    Status = "user doesn't exist"
)

// Returns error
func (status Status) Error() error {
	return errors.New(string(status))
}
