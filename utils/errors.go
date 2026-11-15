package utils

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"
)

// Client-facing error codes. Everything the server puts into an "error" field is
// one of these names: the clients translate them by name (i18n), so a code is
// camelCase and never carries a sentence, punctuation or a value.
const (
	ErrorInternal        = "internalError"
	ErrorInvalidRequest  = "invalidRequest"
	ErrorNotAuthorized   = "notAuthorized"
	ErrorTooManyRequests = "tooManyRequests"

	ErrorAccountWillBeDeleted    = "accountWillBeDeleted"
	ErrorCurrentUserDoesNotExist = "currentUserDoesNotExist"
	ErrorUserNotFound            = "userNotFound"
	ErrorUserAlreadyInDeleteList = "userAlreadyInDeleteList"
	ErrorCannotDeleteSelf        = "cannotDeleteSelf"
	ErrorUsernameRequired        = "usernameRequired"
	ErrorUsernameAlreadyTaken    = "usernameAlreadyTaken"

	ErrorAccessTokenRequired      = "accessTokenRequired"
	ErrorAccessTokenNotFound      = "accessTokenNotFound"
	ErrorAccessTokenMismatch      = "accessTokenMismatch"
	ErrorInvalidToken             = "invalidToken"
	ErrorLoginChallengeMissing    = "loginChallengeMissing"
	ErrorLoginChallengeUsed       = "loginChallengeUsed"
	ErrorRefreshTokenNotRevocable = "refreshTokenNotRevocable"
	ErrorRefreshSessionRevoked    = "refreshSessionRevoked"

	ErrorInviteDoesNotExist   = "inviteDoesNotExist"
	ErrorInvalidEmailAddress  = "invalidEmailAddress"
	ErrorTooManyActiveInvites = "tooManyActiveInvites"

	ErrorMediaFileTooLarge        = "mediaFileTooLarge"
	ErrorMediaNotFound            = "mediaNotFound"
	ErrorInvalidMediaId           = "invalidMediaId"
	ErrorTooManyPendingMediaFiles = "tooManyPendingMediaFiles"

	ErrorCannotSendToSelf       = "cannotSendToSelf"
	ErrorCannotSendToUser       = "cannotSendToUser"
	ErrorReceiverQueueFull      = "receiverQueueFull"
	ErrorAccountInactive        = "accountInactive"
	ErrorTooManyMessages        = "tooManyMessages"
	ErrorInvalidMessage         = "invalidMessage"
	ErrorInvalidMessageType     = "invalidMessageType"
	ErrorUnsupportedMessageType = "unsupportedMessageType"
	ErrorImplausibleTimestamp   = "implausibleTimestamp"
)

// errorCodePattern matches a translatable code and nothing else.
var errorCodePattern = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)

// IsErrorCode reports whether the message is a client-facing code rather than a
// server side failure.
func IsErrorCode(message string) bool {
	return errorCodePattern.MatchString(message)
}

// ErrorCode maps err to a client-facing code. Only a code passes through; a
// server side failure becomes ErrorInternal, so internal detail stays in the log.
func ErrorCode(err error) string {
	if err != nil && IsErrorCode(err.Error()) {
		return err.Error()
	}
	return ErrorInternal
}

// TonicErrorHook keeps the error body translatable for the handlers tonic wraps:
// a code is returned as is, anything else (a binding, database or framework
// error) is replaced by a generic code and left in the request log.
func TonicErrorHook(c *gin.Context, err error) (int, interface{}) {
	if _, isBindError := err.(tonic.BindError); isBindError {
		return http.StatusBadRequest, gin.H{"error": ErrorInvalidRequest}
	}
	return http.StatusBadRequest, gin.H{"error": ErrorCode(err)}
}
