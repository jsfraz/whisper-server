package database

import (
	"context"
	"fmt"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
)

// Retrieves and deletes all private messages for given user.
//
//	@param userId
//	@return *[]models.PrivateMessage
//	@return error
func GetUserPrivateMessages(userId uint64) (*[]models.PrivateMessage, error) {
	var messages []models.PrivateMessage = []models.PrivateMessage{}
	client := utils.GetSingleton().ValkeyMessage
	// Create pattern for user's messages
	pattern := fmt.Sprintf("%d_*", userId)
	// Use SCAN to get all matching keys
	var cursor uint64 = 0
	var keys []string
	for {
		result, err := client.Do(context.Background(), client.B().Scan().Cursor(cursor).Match(pattern).Count(100).Build()).AsScanEntry()
		if err != nil {
			return nil, err
		}
		cursor = result.Cursor
		keys = append(keys, result.Elements...)

		if cursor == 0 {
			break
		}
	}
	// No messages found
	if len(keys) == 0 {
		return &messages, nil
	}
	// Get all messages
	messagesJson, err := client.Do(context.Background(), client.B().Mget().Key(keys...).Build()).AsStrSlice()
	if err != nil {
		return nil, err
	}
	// Parse messages
	for _, m := range messagesJson {
		message, err := models.PrivateMessageFromJson([]byte(m))
		if err != nil {
			return nil, err
		}
		messages = append(messages, *message)
	}
	// Delete all retrieved keys
	if len(keys) > 0 {
		err = client.Do(context.Background(), client.B().Del().Key(keys...).Build()).Error()
		if err != nil {
			return nil, err
		}
	}
	return &messages, nil
}
