package main

import (
	_ "embed"
	"fmt"
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/models"
	"jsfraz/whisper-server/routes"
	"jsfraz/whisper-server/utils"
	"log"
	"net/http"
)

const addr = "0.0.0.0:8080"

func main() {
	// Log settings
	log.SetPrefix("whisper: ")
	log.SetFlags(log.LstdFlags | log.LUTC | log.Lmicroseconds)

	log.Printf(fmt.Sprintf("Starting on %s...", addr))

	// Setup singleton
	singleton := utils.GetSingleton()

	// Load config or panic
	config, err := utils.LoadConfig()
	if err != nil {
		log.Panicln(fmt.Errorf("failed to load config: %v", err))
	}
	singleton.Config = *config

	// Setup PostgreSQL
	database.InitPostgres()
	// Setup Valkey
	database.InitValkey()

	// Get router or panic
	router, err := routes.NewRouter()
	if err != nil {
		log.Panicln(err)
	}
	// Start HTTP server
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}
	// Start server in separated goroutine, panic on error
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Panicln(err)
		}
	}()

	// Subscribe invite creation
	go func() {
		database.SubscribeInvites()
	}()

	// Create admin account
	adminExists, err := database.AdminExists()
	if err != nil {
		log.Panicln(err)
	}
	// Check if code with admin = true exists
	if !adminExists {
		err = database.PushInvite(utils.RandomString(64), *models.NewInviteData(singleton.Config.AdminMail, true), singleton.Config.AdminInviteTtl)
		if err != nil {
			log.Panicln(err)
		}
	}

	/*
			// Create triggers

			// Register trigger
			err = singleton.PostgresDb.Exec(registerTrigger).Error
			if err != nil {
				log.Panicln(err)
			}

			// Verify trigger
			err = singleton.PostgresDb.Exec(verifyTrigger).Error
			if err != nil {
				log.Panicln(err)
			}

		// Listeners in separated goroutines

		// Register trigger
		go func() {
			database.TriggerListener(connStr, "register_channel", func(s string) {
				// Parse to JSON
				var userInfo utils.NotifyUserInfo
				err = json.Unmarshal([]byte(s), &userInfo)
				if err != nil {
					log.Println(err)
					return
				}
				// Send mail
				err = utils.SendMail(*utils.NewMailData(verifyMail), userInfo.Mail, userInfo.Username, userInfo.VerificationCode)
				if err != nil {
					log.Println(err)
				}
			})
		}()

		// Verify trigger
		go func() {
			database.TriggerListener(connStr, "verify_channel", func(s string) {
				// Parse to JSON
				var userInfo utils.NotifyUserInfo
				err = json.Unmarshal([]byte(s), &userInfo)
				if err != nil {
					log.Println(err)
					return
				}
				// Send mail
				err = utils.SendMail(*utils.NewMailData(verifiedMail), userInfo.Mail, userInfo.Username, "")
				if err != nil {
					log.Println(err)
				}
			})
		}()
	*/

	// Channel for blocking exiting main function
	waitForSignal := make(chan bool)
	<-waitForSignal
}
