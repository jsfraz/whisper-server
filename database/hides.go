package database

import (
	"database/sql"
	"errors"
	"jsfraz/shept-server/models"
	"jsfraz/shept-server/utils"
	"time"
)

// ClearConversation hides everything sent at or before before from one account.
// The archive row is untouched, so the other participant keeps the whole chat.
// A later call with an older instant changes nothing, which keeps the call safe
// to retry.
//
//	@param userId
//	@param peerId
//	@param before
//	@return time.Time the cut now in force
//	@return error
func ClearConversation(userId string, peerId string, before time.Time) (time.Time, error) {
	var clearedAt time.Time
	err := utils.GetSingleton().Db.Raw(`INSERT INTO conversation_hides (user_id, conversation, cleared_at)
		VALUES (?, ?, ?)
		ON CONFLICT (user_id, conversation)
		DO UPDATE SET cleared_at = GREATEST(conversation_hides.cleared_at, EXCLUDED.cleared_at)
		RETURNING cleared_at`,
		userId, models.ConversationKey(userId, peerId), before).Scan(&clearedAt).Error
	return clearedAt, err
}

// ConversationClearedAt returns the whole-chat cut of one account, nil when the
// conversation was never cleared.
//
//	@param userId
//	@param peerId
//	@return *time.Time
//	@return error
func ConversationClearedAt(userId string, peerId string) (*time.Time, error) {
	var clearedAt *time.Time
	err := utils.GetSingleton().Db.Raw(`SELECT cleared_at FROM conversation_hides
		WHERE user_id = ? AND conversation = ?`,
		userId, models.ConversationKey(userId, peerId)).Scan(&clearedAt).Error
	return clearedAt, err
}

// HideMessage removes one message from one account only. Idempotent.
//
//	@param userId
//	@param messageId
//	@return error
func HideMessage(userId string, messageId string) error {
	return utils.GetSingleton().Db.Exec(`INSERT INTO message_hides (user_id, message_id)
		VALUES (?, ?) ON CONFLICT DO NOTHING`, userId, messageId).Error
}

// MessageParticipants returns both sides of an archived message, found=false
// when there is no such row.
//
//	@param messageId
//	@return string sender
//	@return string receiver
//	@return bool
//	@return error
func MessageParticipants(messageId string) (string, string, bool, error) {
	var senderId, receiverId string
	err := utils.GetSingleton().Db.Raw(`SELECT sender_id, receiver_id FROM messages WHERE id = ?`,
		messageId).Row().Scan(&senderId, &receiverId)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, err
	}
	return senderId, receiverId, true, nil
}
