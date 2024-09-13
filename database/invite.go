package database

import (
	"context"
	"errors"
	"fmt"
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
func PushInvite(code string, invite models.InviteData, ttl int) error {
	// Marshall JSON
	i, err := invite.MarshalBinary()
	if err != nil {
		return err
	}
	// Push
	client := utils.GetSingleton().Valkey
	err = client.Do(context.Background(), client.B().Set().Key(code).Value(string(i)).ExSeconds(int64(ttl)).Build()).Error()
	if err != nil {
		return err
	}
	return client.Do(context.Background(), client.B().Publish().Channel("newInvite").Message(code).Build()).Error()
}

// Subscribe for new invites and send mail.
func SubscribeInvites() {
	c, cancel := utils.GetSingleton().Valkey.Dedicate()
	defer cancel()
	wait := c.SetPubSubHooks(valkey.PubSubHooks{
		OnMessage: func(m valkey.PubSubMessage) {
			// Get invite from Valkey
			client := utils.GetSingleton().Valkey
			result, err := client.Do(context.Background(), client.B().Get().Key(m.Message).Build()).AsBytes()
			// Return error except if is Valkey error
			if err != nil && !errors.As(err, &valkeyErr) {
				fmt.Println(err)
				return
			}
			// Get inviteData from JSON
			inviteData, err := models.InviteDataFromJson(result)
			if err != nil {
				fmt.Println(err)
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
					fmt.Println(err)
					return
				}
				subject = "Admin registration"
			} else {
				template, err = utils.ReadFile("./mailTemplates/registerInvite.hbs")
				if err != nil {
					fmt.Println(err)
					return
				}
				subject = "Registration invite"
			}
			// Generate QR code
			inviteJsonBytes, err := models.NewInvite(utils.GetSingleton().Config.ServerUrl, m.Message, inviteData.ValidUntil).MarshalBinary()
			if err != nil {
				fmt.Println(err)
				return
			}
			qrBase64, err := utils.GetQrBytesFromJson(string(inviteJsonBytes))
			if err != nil {
				fmt.Println(err)
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
				fmt.Println(err)
				return
			}
			// Send mail
			err = utils.SendMail(inviteData.Mail, subject, content)
			if err != nil {
				fmt.Println(err)
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
	client := utils.GetSingleton().Valkey
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
	client := utils.GetSingleton().Valkey
	return client.Do(context.Background(), client.B().Del().Key(code).Build()).Error()
}

// Check if admin invite exists
//
//	@return bool
//	@return error
func AdminInviteExists() (bool, error) {
	client := utils.GetSingleton().Valkey
	// Get all keys
	keys, err := client.Do(context.Background(), client.B().Keys().Pattern("*").Build()).AsStrSlice()
	if err != nil {
		return false, err
	}
	// Cycle trough keys
	for _, k := range keys {
		inviteDataBytes, err := client.Do(context.Background(), client.B().Get().Key(k).Build()).AsBytes()
		if err != nil && !errors.As(err, &valkeyErr) {
			log.Println(err)
			continue
		}
		if inviteDataBytes != nil {
			// Unmarshall invite data
			inviteData, err := models.InviteDataFromJson(inviteDataBytes)
			if err != nil {
				return false, err
			}
			// Check if invite is for admin
			if inviteData.Admin {
				return true, nil
			}
		}
	}
	return false, nil
}

// Create admin invite if admin does not exist
//
//	@return error
func SendAdminInvite() error {
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
		err = PushInvite(utils.RandomASCIIString(64), *models.NewInviteData(utils.GetSingleton().Config.AdminMail, true, time.Now().Add(time.Duration(ttl)*time.Second)), ttl)
		if err != nil {
			return err
		}
	}
	return nil
}
