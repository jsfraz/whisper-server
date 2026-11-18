package database

import (
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// messageColumns is the physical order a scanned message expects.
const messageColumns = "id, seq, sender_id, receiver_id, content, wrapped_key, nonce, mac, sent_at, expires_at, deleted_at"

// visibleTo keeps only the rows one account may still see: nothing covered by
// its whole-chat cut and nothing hidden for it alone. It binds two arguments,
// the viewer id twice, and expects the messages table to be in scope.
const visibleTo = ` NOT EXISTS (SELECT 1 FROM conversation_hides h
		WHERE h.user_id = ? AND h.conversation = messages.conversation
		  AND messages.sent_at <= h.cleared_at)
	AND NOT EXISTS (SELECT 1 FROM message_hides mh
		WHERE mh.user_id = ? AND mh.message_id = messages.id)`

// InsertMessage writes one ciphertext message to the archive and fills its id
// and seq. A resend of the same message (same sender, receiver and timestamp)
// keeps the original row instead of creating a second copy.
//
//	@param message
//	@param retentionDays
//	@return error
func InsertMessage(message *models.PrivateMessage, retentionDays int) error {
	db := utils.GetSingleton().Db
	if message.Id == "" {
		message.Id = uuid.NewString()
	}
	message.Conversation = models.ConversationKey(message.SenderId, message.ReceiverId)
	message.ExpiresAt = message.SentAt.AddDate(0, 0, retentionDays)
	var seq int64
	err := db.Raw(`INSERT INTO messages
		(id, conversation, sender_id, receiver_id, content, wrapped_key, nonce, mac, sent_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (sender_id, receiver_id, sent_at) DO NOTHING
		RETURNING seq`,
		message.Id, message.Conversation, message.SenderId, message.ReceiverId,
		message.Content, message.Key, message.Nonce, message.Mac, message.SentAt, message.ExpiresAt).
		Scan(&seq).Error
	if err != nil {
		return err
	}
	if seq != 0 {
		message.Seq = seq
		return nil
	}
	// The row existed: a resend after a lost confirmation.
	return db.Raw(`SELECT id, seq FROM messages WHERE sender_id = ? AND receiver_id = ? AND sent_at = ?`,
		message.SenderId, message.ReceiverId, message.SentAt).
		Row().Scan(&message.Id, &message.Seq)
}

// HistoryForPeer returns one page of a conversation, newest first.
//
//	@param userId
//	@param peerId
//	@param before
//	@param limit
//	@return []models.PrivateMessage
//	@return error
func HistoryForPeer(userId string, peerId string, before *utils.Cursor, limit int) ([]models.PrivateMessage, error) {
	db := utils.GetSingleton().Db
	args := []interface{}{models.ConversationKey(userId, peerId), time.Now(), userId, userId}
	query := `SELECT ` + messageColumns + ` FROM messages
		WHERE conversation = ? AND expires_at > ? AND` + visibleTo
	if before != nil {
		query += ` AND (sent_at, id) < (?, ?)`
		args = append(args, before.SentAt, before.Id)
	}
	query += ` ORDER BY sent_at DESC, id DESC LIMIT ?`
	args = append(args, limit)
	messages := []models.PrivateMessage{}
	if err := db.Raw(query, args...).Scan(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

// SyncSince returns the archive delta a device missed, oldest first.
//
//	@param userId
//	@param sinceSeq
//	@param limit
//	@return []models.PrivateMessage
//	@return error
func SyncSince(userId string, sinceSeq int64, limit int) ([]models.PrivateMessage, error) {
	messages := []models.PrivateMessage{}
	err := utils.GetSingleton().Db.Raw(`SELECT `+messageColumns+` FROM messages
		WHERE (sender_id = ? OR receiver_id = ?) AND seq > ? AND expires_at > ? AND`+visibleTo+`
		ORDER BY seq LIMIT ?`, userId, userId, sinceSeq, time.Now(), userId, userId, limit).
		Scan(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// PruneExpired deletes one batch of messages past their retention. The sweeper
// repeats until the batch comes back short.
//
//	@param batch
//	@return int64
//	@return error
func PruneExpired(batch int) (int64, error) {
	result := utils.GetSingleton().Db.Exec(`DELETE FROM messages WHERE seq IN (
		SELECT seq FROM messages WHERE expires_at < ? LIMIT ?)`, time.Now(), batch)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// DeleteUserMessages drops the archive of a deleted account. The foreign keys
// would cascade, but account deletion has to be explicit about it.
//
//	@param userId
//	@return error
func DeleteUserMessages(userId string) error {
	return utils.GetSingleton().Db.Exec(`DELETE FROM messages WHERE sender_id = ? OR receiver_id = ?`, userId, userId).Error
}

// GetUsersByIds loads the accounts referenced by a batch, keyed by id.
//
//	@param ids
//	@return map[string]models.User
//	@return error
func GetUsersByIds(ids []string) (map[string]models.User, error) {
	users := map[string]models.User{}
	unique := map[string]struct{}{}
	for _, id := range ids {
		if id != "" {
			unique[id] = struct{}{}
		}
	}
	if len(unique) == 0 {
		return users, nil
	}
	keys := make([]string, 0, len(unique))
	for id := range unique {
		keys = append(keys, id)
	}
	var rows []models.User
	if err := utils.GetSingleton().Db.Where("id IN ?", keys).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, user := range rows {
		users[user.Id] = user
	}
	return users, nil
}

// RetentionDays returns the archive retention of a sender and a recipient: the
// shorter of the two wins, because a message is visible to both accounts.
//
//	@param senderRetention
//	@param receiverRetention
//	@return int
func ShortestRetention(senderRetention int, receiverRetention int) int {
	if senderRetention <= 0 {
		senderRetention = models.DefaultRetentionDays
	}
	if receiverRetention <= 0 {
		receiverRetention = models.DefaultRetentionDays
	}
	if receiverRetention < senderRetention {
		return receiverRetention
	}
	return senderRetention
}

// ConversationsOf collects the peer ids of a message batch as one viewer sees them.
//
//	@param messages
//	@param viewerId
//	@return []string
func ConversationsOf(messages []models.PrivateMessage, viewerId string) []string {
	peers := make([]string, 0, len(messages))
	for _, message := range messages {
		if message.SenderId == viewerId {
			peers = append(peers, message.ReceiverId)
		} else {
			peers = append(peers, message.SenderId)
		}
	}
	return peers
}

// RunMessageSweeper deletes messages past their retention. It runs in batches
// so a large backlog does not hold one long write transaction.
//
//	@param interval
func RunMessageSweeper(interval time.Duration) {
	for {
		for {
			removed, err := PruneExpired(1000)
			if err != nil {
				slog.Error("message archive sweep failed", "error", err)
				break
			}
			if removed == 0 {
				break
			}
		}
		time.Sleep(interval)
	}
}
