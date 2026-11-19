package database

import (
	"database/sql"
	"errors"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"time"

	"gorm.io/gorm"
)

// eventColumns is the physical order a scanned event expects. The table name is
// spelled out because message_events joins messages, which also has sent_at.
const eventColumns = `message_events.event_seq, message_events.message_id, message_events.message_seq,
	message_events.conversation, message_events.sent_at, message_events.kind, message_events.actor_id,
	message_events.content, message_events.wrapped_key, message_events.nonce, message_events.mac,
	message_events.pinned, message_events.created_at`

// MessageRow is the part of an archived message a mutation needs in order to
// authorize the caller and to stamp the log entry.
type MessageRow struct {
	Id           string
	Seq          int64
	Conversation string
	SenderId     string
	ReceiverId   string
	SentAt       time.Time
	DeletedAt    *time.Time
}

// GetMessageRow loads one archived message, found=false when there is no such
// row. A tombstone is still found: mutations of a deleted message are refused
// by the handler, not by a missing row.
//
//	@param messageId
//	@return MessageRow
//	@return bool
//	@return error
func GetMessageRow(messageId string) (MessageRow, bool, error) {
	var row MessageRow
	err := utils.GetSingleton().Db.Raw(`SELECT id, seq, conversation, sender_id, receiver_id, sent_at, deleted_at
		FROM messages WHERE id = ?`, messageId).Row().
		Scan(&row.Id, &row.Seq, &row.Conversation, &row.SenderId, &row.ReceiverId, &row.SentAt, &row.DeletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return MessageRow{}, false, nil
	}
	if err != nil {
		return MessageRow{}, false, err
	}
	return row, true, nil
}

// insertEvent appends one mutation to the log and fills its event_seq.
func insertEvent(tx *gorm.DB, event *models.MessageEvent) error {
	return tx.Raw(`INSERT INTO message_events
		(message_id, message_seq, conversation, sent_at, kind, actor_id, content, wrapped_key, nonce, mac, pinned, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING event_seq`,
		event.MessageId, event.MessageSeq, event.Conversation, event.SentAt, event.Kind,
		event.ActorId, event.Content, event.Key, event.Nonce, event.Mac, event.Pinned, event.CreatedAt).
		Scan(&event.EventSeq).Error
}

// EditMessage replaces the ciphertext of an own message and logs the change, so
// a device that already holds the old row learns about it on the next sync.
//
//	@param row
//	@param cipher
//	@param editedAt
//	@return *models.MessageEvent
//	@return error
func EditMessage(row MessageRow, cipher models.MessageCiphertext, editedAt time.Time) (*models.MessageEvent, error) {
	event := &models.MessageEvent{
		MessageId:    row.Id,
		MessageSeq:   row.Seq,
		Conversation: row.Conversation,
		SentAt:       row.SentAt,
		Kind:         models.MessageEventEdit,
		ActorId:      row.SenderId,
		Content:      cipher.Content,
		Key:          cipher.Key,
		Nonce:        cipher.Nonce,
		Mac:          cipher.Mac,
		CreatedAt:    editedAt,
	}
	err := utils.GetSingleton().Db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`UPDATE messages
			SET content = ?, wrapped_key = ?, nonce = ?, mac = ?, edited_at = ?
			WHERE id = ?`,
			cipher.Content, cipher.Key, cipher.Nonce, cipher.Mac, editedAt, row.Id).Error; err != nil {
			return err
		}
		return insertEvent(tx, event)
	})
	if err != nil {
		return nil, err
	}
	return event, nil
}

// DeleteMessageForEveryone drops the ciphertext of an own message for both
// sides, unpins it and logs the tombstone. The row keeps its slot so the peer
// learns about it through the log instead of a silent gap.
//
//	@param row
//	@param deletedAt
//	@return *models.MessageEvent
//	@return error
func DeleteMessageForEveryone(row MessageRow, deletedAt time.Time) (*models.MessageEvent, error) {
	event := &models.MessageEvent{
		MessageId:    row.Id,
		MessageSeq:   row.Seq,
		Conversation: row.Conversation,
		SentAt:       row.SentAt,
		Kind:         models.MessageEventDelete,
		ActorId:      row.SenderId,
		CreatedAt:    deletedAt,
	}
	err := utils.GetSingleton().Db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`UPDATE messages
			SET content = NULL, wrapped_key = NULL, nonce = NULL, mac = NULL,
			    pinned_at = NULL, pinned_by = NULL, deleted_at = COALESCE(deleted_at, ?)
			WHERE id = ?`, deletedAt, row.Id).Error; err != nil {
			return err
		}
		return insertEvent(tx, event)
	})
	if err != nil {
		return nil, err
	}
	return event, nil
}

// SetReaction logs one reaction of one account on a message. A nil cipher is a
// removal; the fold on the client is last-write-wins per actor.
//
//	@param row
//	@param actorId
//	@param cipher
//	@param createdAt
//	@return *models.MessageEvent
//	@return error
func SetReaction(row MessageRow, actorId string, cipher *models.MessageCiphertext, createdAt time.Time) (*models.MessageEvent, error) {
	event := &models.MessageEvent{
		MessageId:    row.Id,
		MessageSeq:   row.Seq,
		Conversation: row.Conversation,
		SentAt:       row.SentAt,
		Kind:         models.MessageEventReaction,
		ActorId:      actorId,
		CreatedAt:    createdAt,
	}
	if cipher != nil {
		event.Content = cipher.Content
		event.Key = cipher.Key
		event.Nonce = cipher.Nonce
		event.Mac = cipher.Mac
	}
	if err := insertEvent(utils.GetSingleton().Db, event); err != nil {
		return nil, err
	}
	return event, nil
}

// SetPinned pins or unpins a message for both participants. It reports
// changed=false when the message already was in that state, so a retry does not
// append a duplicate log entry.
//
//	@param row
//	@param actorId
//	@param pinned
//	@param now
//	@return *models.MessageEvent
//	@return bool
//	@return error
func SetPinned(row MessageRow, actorId string, pinned bool, now time.Time) (*models.MessageEvent, bool, error) {
	var result *gorm.DB
	if pinned {
		result = utils.GetSingleton().Db.Exec(`UPDATE messages
			SET pinned_at = COALESCE(pinned_at, ?), pinned_by = COALESCE(pinned_by, ?)
			WHERE id = ? AND pinned_at IS NULL`, now, actorId, row.Id)
	} else {
		result = utils.GetSingleton().Db.Exec(`UPDATE messages
			SET pinned_at = NULL, pinned_by = NULL
			WHERE id = ? AND pinned_at IS NOT NULL`, row.Id)
	}
	if result.Error != nil {
		return nil, false, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, false, nil
	}
	event := &models.MessageEvent{
		MessageId:    row.Id,
		MessageSeq:   row.Seq,
		Conversation: row.Conversation,
		SentAt:       row.SentAt,
		Kind:         models.MessageEventPin,
		ActorId:      actorId,
		Pinned:       &pinned,
		CreatedAt:    now,
	}
	if err := insertEvent(utils.GetSingleton().Db, event); err != nil {
		return nil, false, err
	}
	return event, true, nil
}

// EventsSince returns the mutations a device missed, oldest first. An event is
// held back until the device has the message it talks about, so a page can
// never advance the event cursor past a message still in flight.
//
//	@param userId
//	@param sinceEventSeq
//	@param syncedThrough highest message seq the device now holds
//	@param limit
//	@return []models.MessageEvent
//	@return error
func EventsSince(userId string, sinceEventSeq int64, syncedThrough int64, limit int) ([]models.MessageEvent, error) {
	events := []models.MessageEvent{}
	err := utils.GetSingleton().Db.Raw(`SELECT `+eventColumns+`
		FROM message_events
		JOIN messages ON messages.id = message_events.message_id
		WHERE message_events.event_seq > ? AND message_events.message_seq <= ?
		  AND (messages.sender_id = ? OR messages.receiver_id = ?)
		  AND messages.expires_at > ? AND`+visibleTo+`
		ORDER BY message_events.event_seq LIMIT ?`,
		sinceEventSeq, syncedThrough, userId, userId, time.Now(), userId, userId, limit).
		Scan(&events).Error
	if err != nil {
		return nil, err
	}
	return events, nil
}

// ClearedFor returns every whole-chat cut of one account, so sync can empty a
// chat on a device that was offline while it was cleared.
//
//	@param userId
//	@return []models.ConversationCleared
//	@return error
func ClearedFor(userId string) ([]models.ConversationCleared, error) {
	var rows []struct {
		Conversation string
		ClearedAt    time.Time
	}
	if err := utils.GetSingleton().Db.Raw(`SELECT conversation, cleared_at FROM conversation_hides
		WHERE user_id = ? ORDER BY cleared_at DESC`, userId).Scan(&rows).Error; err != nil {
		return nil, err
	}
	cleared := make([]models.ConversationCleared, 0, len(rows))
	for _, row := range rows {
		cleared = append(cleared, models.ConversationCleared{
			PeerId:    models.PeerOfConversation(row.Conversation, userId),
			ClearedAt: row.ClearedAt,
		})
	}
	return cleared, nil
}

// PinsForPeer returns the pinned messages of one conversation, newest pin
// first.
//
//	@param userId
//	@param peerId
//	@return []models.PinnedMessage
//	@return error
func PinsForPeer(userId string, peerId string) ([]models.PinnedMessage, error) {
	pins := []models.PinnedMessage{}
	err := utils.GetSingleton().Db.Raw(`SELECT id AS message_id, pinned_by, pinned_at FROM messages
		WHERE conversation = ? AND pinned_at IS NOT NULL AND expires_at > ? AND deleted_at IS NULL AND`+visibleTo+`
		ORDER BY pinned_at DESC`,
		models.ConversationKey(userId, peerId), time.Now(), userId, userId).Scan(&pins).Error
	if err != nil {
		return nil, err
	}
	return pins, nil
}
