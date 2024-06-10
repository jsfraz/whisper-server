package main

import (
	_ "embed"
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/routes"
	"jsfraz/whisper-server/utils"
	"log"
	"net/http"
)

// Mail templates
var (
	//go:embed static/mailTemplate.html
	mailTemlplate string
	//go:embed static/verifyMail.json
	verifyMail string
	//go:embed static/verifiedMail.json
	verifiedMail string
)

func main() {

	utils.CheckEnvs()

	singleton := utils.GetSingleton()
	singleton.MailTemlplate = mailTemlplate
	singleton.VerifyMail = *utils.NewMailData(verifyMail)
	singleton.VerifiedMail = *utils.NewMailData(verifiedMail)
	singleton.PostgresDb = *database.InitPostgres()

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
	srv.ListenAndServe()
}
