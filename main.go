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
	"os"
)

var (
	// Mail templates
	//go:embed mailTemplates/mailTemplate.html
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

	// Check ENVs and set singleton values

	utils.CheckEnvs()

	singleton := utils.GetSingleton()
	singleton.MailTemlplate = mailTemlplate
	singleton.VerifyMail = *utils.NewMailData(verifyMail)
	singleton.VerifiedMail = *utils.NewMailData(verifiedMail)
	connStr := "postgresql://" + os.Getenv("POSTGRES_USER") + ":" + os.Getenv("POSTGRES_PASSWORD") + "@" + os.Getenv("POSTGRES_SERVER") + ":" + os.Getenv("POSTGRES_PORT") + "/" + os.Getenv("POSTGRES_DB") + "?sslmode=disable"
	singleton.PostgresDb = *database.InitPostgres(connStr)

	// Get router
	router, err := routes.NewRouter()
	if err != nil {
		log.Fatal(err)
	}
	// Start HTTP server
	srv := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: router,
	}
	// Start server in separated goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			fmt.Println("Error starting the server:", err)
		}
	}()

	// Create triggers

	// Register trigger
	err = singleton.PostgresDb.Exec(registerTrigger).Error
	if err != nil {
		panic(err)
	}

	// Verify trigger
	err = singleton.PostgresDb.Exec(verifyTrigger).Error
	if err != nil {
		panic(err)
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
