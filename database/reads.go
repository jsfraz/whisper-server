package database

import (
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"time"
)

// SetReadCursor records how far one account has read a conversation. The cursor
// only moves forward and never past now, so a client clock ahead of the server
// cannot mark unread messages as read.
//
//	@param userId
//	@param peerId
//	@param readAt
//	@return time.Time the cursor now in force
//	@return error
func SetReadCursor(userId string, peerId string, readAt time.Time) (time.Time, error) {
	// A message is stamped with the sender's clock, accepted within
	// MaxSentAtSkew. Clamping the cursor to now would leave a message from a
	// clock that runs a little ahead permanently unread, so the same skew is
	// allowed here.
	limit := time.Now().UTC().Add(models.MaxSentAtSkew)
	if readAt.After(limit) {
		readAt = limit
	}
	var stored time.Time
	err := utils.GetSingleton().Db.Raw(`INSERT INTO conversation_reads
		(user_id, conversation, peer_id, read_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (user_id, conversation)
		DO UPDATE SET read_at = GREATEST(conversation_reads.read_at, EXCLUDED.read_at)
		RETURNING read_at`,
		userId, models.ConversationKey(userId, peerId), peerId, readAt).Scan(&stored).Error
	return stored, err
}

// ReadCursorsFor returns the cursors other accounts set in the viewer's chats,
// so a device that was offline while a message was read catches up on sync.
//
//	@param userId
//	@return []models.ReadReceipt
//	@return error
func ReadCursorsFor(userId string) ([]models.ReadReceipt, error) {
	receipts := []models.ReadReceipt{}
	// The row belongs to the account that read; its peer_id is the viewer, so
	// the owner is what the viewer needs to see.
	err := utils.GetSingleton().Db.Raw(`SELECT user_id AS peer_id, read_at FROM conversation_reads
		WHERE peer_id = ? ORDER BY read_at DESC`, userId).Scan(&receipts).Error
	if err != nil {
		return nil, err
	}
	for i := range receipts {
		receipts[i].V = 1
	}
	return receipts, nil
}
