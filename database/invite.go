package database

import (
	"context"
	"errors"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"log"
	"time"

	"github.com/aymerick/raymond"
	"github.com/valkey-io/valkey-go"
)

var valkeyErr *valkey.ValkeyError

// Push new invite record to Valkey and send notification message to newInvite channel.
//
//	@param code
//	@param invite
//	@param ttl
//	@return error
func PushInvite(code string, invite models.Invite, ttl int) error {
	// Marshall JSON
	i, err := invite.MarshalBinary()
	if err != nil {
		return err
	}
	// Push
	client := utils.GetSingleton().ValkeyInvite
	err = client.Do(context.Background(), client.B().Set().Key(code).Value(string(i)).ExSeconds(int64(ttl)).Build()).Error()
	if err != nil {
		return err
	}
	return client.Do(context.Background(), client.B().Publish().Channel("newInvite").Message(code).Build()).Error()
}

// Subscribe for new invites and send mail.
func SubscribeNewInvites() {
	c, cancel := utils.GetSingleton().ValkeyInvite.Dedicate()
	defer cancel()
	wait := c.SetPubSubHooks(valkey.PubSubHooks{
		OnMessage: func(m valkey.PubSubMessage) {
			// Get invite from Valkey
			client := utils.GetSingleton().ValkeyInvite
			result, err := client.Do(context.Background(), client.B().Get().Key(m.Message).Build()).AsBytes()
			// Return error except if is Valkey error
			if err != nil && !errors.As(err, &valkeyErr) {
				log.Println(err)
				return
			}
			// Get inviteData from JSON
			inviteData, err := models.InviteFromJson(result)
			if err != nil {
				log.Println(err)
				return
			}
			// Mail variables
			var template *string
			var content string
			var subject string
			// Load template and set variables
			if inviteData.Admin {
				template, err = utils.ReadFile("./mailTemplates/registerAdmin.hbs")
				if err != nil {
					log.Println(err)
					return
				}
				subject = "Admin registration"
			} else {
				template, err = utils.ReadFile("./mailTemplates/registerInvite.hbs")
				if err != nil {
					log.Println(err)
					return
				}
				subject = "Registration invite"
			}
			// Generate QR code
			inviteJsonBytes, err := models.NewInviteData(utils.GetSingleton().Config.ServerUrl, m.Message, inviteData.ValidUntil).MarshalBinary()
			if err != nil {
				log.Println(err)
				return
			}
			qrBase64, err := utils.GetQrBytesFromJson(string(inviteJsonBytes))
			if err != nil {
				log.Println(err)
				return
			}
			// Render template
			content, err = raymond.Render(
				*template,
				map[string]string{
					"qrBase64":   *qrBase64,
					"validUntil": inviteData.ValidUntil.Format("2.1. 2006 15:04:05"),
					"footer":     utils.GetMailFooter(),
				},
			)
			if err != nil {
				log.Println(err)
				return
			}
			// Send mail
			err = utils.SendMail(inviteData.Mail, subject, content)
			if err != nil {
				log.Println(err)
				return
			}
			log.Printf("Invite sent to %s, admin: %t", inviteData.Mail, inviteData.Admin)
		},
	})
	c.Do(context.Background(), c.B().Subscribe().Channel("newInvite").Build())
	<-wait
}

// Return invite by code from Valkey.
//
//	@param code
//	@return bool
//	@return []byte
//	@return error
func GetInviteDataByCode(code string) (bool, []byte, error) {
	client := utils.GetSingleton().ValkeyInvite
	result, err := client.Do(context.Background(), client.B().Get().Key(code).Build()).AsBytes()
	// Return error except if is Valkey error
	if err != nil && !errors.As(err, &valkeyErr) {
		return false, []byte{}, err
	}
	// Return result
	if result != nil {
		return true, result, nil
	}
	return false, []byte{}, nil
}

// Delete invite by code.
//
//	@param code
//	@return error
func DeleteInviteDataByCode(code string) error {
	client := utils.GetSingleton().ValkeyInvite
	return client.Do(context.Background(), client.B().Del().Key(code).Build()).Error()
}

// Check if admin invite exists
//
//	@return bool
//	@return error
func AdminInviteExists() (bool, error) {
	client := utils.GetSingleton().ValkeyInvite

	// Use SCAN instead of KEYS
	var cursor uint64 = 0
	for {
		// Scan batch of keys
		result, err := client.Do(context.Background(), client.B().Scan().Cursor(cursor).Match("*").Count(100).Build()).AsScanEntry()
		if err != nil {
			return false, err
		}

		// Check each key in the batch
		for _, k := range result.Elements {
			inviteDataBytes, err := client.Do(context.Background(), client.B().Get().Key(k).Build()).AsBytes()
			if err != nil && !errors.As(err, &valkeyErr) {
				log.Println(err)
				continue
			}
			if inviteDataBytes != nil {
				// Unmarshall invite data
				inviteData, err := models.InviteFromJson(inviteDataBytes)
				if err != nil {
					return false, err
				}
				// Check if invite is for admin
				if inviteData.Admin {
					return true, nil
				}
			}
		}

		// Break if cursor is 0 (scan completed)
		if result.Cursor == 0 {
			break
		}
		cursor = result.Cursor
	}
	return false, nil
}

// Create admin invite if admin does not exist
//
//	@return error
func CreateAdminInvite() error {
	// Check if admin account exists
	adminExists, err := AdminExists()
	if err != nil {
		return err
	}
	// Check if admin invite exists
	adminInviteExists, err := AdminInviteExists()
	if err != nil {
		return err
	}
	// Send admin invite
	if !adminExists && !adminInviteExists {
		ttl := utils.GetSingleton().Config.AdminInviteTtl
		err = PushInvite(utils.RandomASCIIString(64), *models.NewInvite(utils.GetSingleton().Config.AdminMail, true, time.Now().Add(time.Duration(ttl)*time.Second)), ttl)
		if err != nil {
			return err
		}
	}
	return nil
}

// Get all invites
//
//	@return *[]models.Invite
//	@return error
func GetAllInvites() (*[]models.Invite, error) {
	var invites []models.Invite = []models.Invite{}
	client := utils.GetSingleton().ValkeyInvite

	// Use SCAN instead of KEYS
	var cursor uint64 = 0
	var keys []string
	for {
		// Scan batch of keys
		var batch []string
		result, err := client.Do(context.Background(), client.B().Scan().Cursor(cursor).Match("*").Count(100).Build()).AsScanEntry()
		if err != nil {
			return nil, err
		}
		cursor = result.Cursor
		batch = result.Elements

		// Append batch to all keys
		keys = append(keys, batch...)

		// Break if cursor is 0 (scan completed)
		if cursor == 0 {
			break
		}
	}

	// Zero keys
	if len(keys) == 0 {
		return &invites, nil
	}

	// Get all invites as JSON
	invitesJson, err := client.Do(context.Background(), client.B().Mget().Key(keys...).Build()).AsStrSlice()
	if err != nil {
		return nil, err
	}
	// Unmarshall JSON to Invite
	for _, i := range invitesJson {
		invite, err := models.InviteFromJson([]byte(i))
		if err != nil {
			return nil, err
		}
		invites = append(invites, *invite)
	}
	return &invites, nil
}
