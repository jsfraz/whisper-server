package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/routes"
	"jsfraz/whisper-server/utils"
	"log"
	"net/http"
)

const addr = "0.0.0.0:8080"

var (
	// Mail templates
	//go:embed mailTemplates/mailTemplate.hbs
	mailTemlplate string
	//go:embed mailTemplates/verifyMail.json
	verifyMail string
	//go:embed mailTemplates/verifiedMail.json
	verifiedMail string

	// SQL scripts
	//go:embed sqlScripts/register_trigger.sql
	registerTrigger string
	//go:embed sqlScripts/verify_trigger.sql
	verifyTrigger string
)

func main() {
	// Log settings
	log.SetPrefix("whisper: ")
	log.SetFlags(log.LstdFlags | log.LUTC | log.Lmicroseconds)

	log.Printf(fmt.Sprintf("Starting on %s...", addr))

	// Setup singleton
	singleton := utils.GetSingleton()
	singleton.MailTemlplate = mailTemlplate
	singleton.VerifyMail = *utils.NewMailData(verifyMail)
	singleton.VerifiedMail = *utils.NewMailData(verifiedMail)

	// Load config
	config, err := utils.LoadConfig()
	if err != nil {
		log.Fatalln(err)
	}
	singleton.Config = *config

	// Setup database
	connStr := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=disable",
		singleton.Config.PostgresUser,
		singleton.Config.PostgresPassword,
		singleton.Config.PostgresHost,
		singleton.Config.PostgresPort,
		singleton.Config.PostgresDb)
	singleton.PostgresDb = *database.InitPostgres(connStr)

	// Get router
	router, err := routes.NewRouter()
	if err != nil {
		log.Fatal(err)
	}
	// Start HTTP server
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}
	// Start server in separated goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Panicln(err)
		}
	}()

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

	// Channel for blocking exiting main function
	waitForSignal := make(chan bool)
	<-waitForSignal
}
