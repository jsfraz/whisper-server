package push

import (
	"context"
	"errors"
	"jsfraz/shept-server/database"
	"jsfraz/shept-server/utils"

	messaging "firebase.google.com/go/v4/messaging"
)

// Send notifies every device of an account. A message to an offline user is
// what wakes the app; the content itself stays end-to-end encrypted.
//
//	@param userId
//	@param title
//	@param body
//	@return error
func Send(userId string, title string, body string) error {
	tokens, err := database.FirebaseTokensOf(userId)
	if err != nil {
		return err
	}
	fids := make([]string, 0, len(tokens))
	for _, token := range tokens {
		fids = append(fids, token)
	}
	return sendToFids(fids, title, body)
}

// SendToOtherDevices warns the devices that already existed about a new login,
// without notifying the device that just signed in.
//
//	@param userId
//	@param exceptDeviceId
//	@param title
//	@param body
//	@return error
func SendToOtherDevices(userId string, exceptDeviceId string, title string, body string) error {
	tokens, err := database.FirebaseTokensOf(userId)
	if err != nil {
		return err
	}
	fids := make([]string, 0, len(tokens))
	for deviceId, token := range tokens {
		if deviceId == exceptDeviceId {
			continue
		}
		fids = append(fids, token)
	}
	return sendToFids(fids, title, body)
}

func sendToFids(fids []string, title string, body string) error {
	if len(fids) == 0 {
		return nil
	}
	// Message.Token accepts only FCM tokens, FIDs must go through multicast.
	resp, err := utils.GetSingleton().FirebaseMsg.SendEachForMulticast(context.Background(), &messaging.MulticastMessage{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Fids: fids,
	})
	if err != nil {
		return err
	}
	if resp.FailureCount > 0 && len(resp.Responses) > 0 {
		return errors.New("push delivery failed")
	}
	return nil
}
