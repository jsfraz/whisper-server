package main

import (
	_ "embed"
	"fmt"
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/routes"
	"jsfraz/whisper-server/utils"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/lib/pq"
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

	// Listeners in separated goroutines

	// Register trigger
	go func() {
		TriggerListener(connStr, "register_channel", func(s string) {
			// TODO send mail
			log.Println(s)
		})
	}()

	// Channel for blocking exiting main function
	waitForSignal := make(chan bool)
	<-waitForSignal

}

// Method for creating listener for specific triggers
//
//	@param connStr
//	@param channel
//	@param callback
func TriggerListener(connStr string, channel string, callback func(string)) {
	// Create listener
	listener := pq.NewListener(connStr, 10*time.Second, time.Minute, func(event pq.ListenerEventType, err error) {
		if err != nil {
			log.Println(err.Error())
		}
	})
	err := listener.Listen(channel)
	if err != nil {
		log.Fatal(err)
	}
	// Listen
	for {
		select {
		case notification := <-listener.Notify:
			// Change detection
			callback(notification.Extra)
		case <-time.After(90 * time.Second):
			go listener.Ping()
		}
	}
}
