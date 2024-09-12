package database

import (
	"context"
	"errors"
	"fmt"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/utils"
	"time"

	"github.com/aymerick/raymond"
	"github.com/valkey-io/valkey-go"
)

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
			var valkeyErr *valkey.ValkeyError
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
			var ttl int
			var content string
			var subject string
			// Load template and set variables
			if inviteData.Admin {
				template, err = utils.ReadFile("./mailTemplates/registerAdmin.hbs")
				if err != nil {
					fmt.Println(err)
					return
				}
				ttl = utils.GetSingleton().Config.AdminInviteTtl
				subject = "Admin registration"
			} else {
				template, err = utils.ReadFile("./mailTemplates/registerInvite.hbs")
				if err != nil {
					fmt.Println(err)
					return
				}
				ttl = utils.GetSingleton().Config.InviteTtl
				subject = "Registration invite"
			}
			// Generate QR code
			inviteJsonBytes, err := models.NewInvite(utils.GetSingleton().Config.ServerUrl, m.Message).MarshalBinary()
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
					"validUntil": time.Now().Add(time.Duration(ttl) * time.Second).Format("2.1. 2006 15:04"),
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
		},
	})
	c.Do(context.Background(), c.B().Subscribe().Channel("newInvite").Build())
	<-wait
}
