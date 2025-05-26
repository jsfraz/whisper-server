package main

import (
	"context"
	_ "embed"
	"fmt"
	"jsfraz/whisper-server/database"
	"jsfraz/whisper-server/routes"
	"jsfraz/whisper-server/utils"
	"log"
	"net/http"

	firebase "firebase.google.com/go/v4"
)

const addr = "0.0.0.0:8080"

func main() {
	// Log settings
	log.SetPrefix("whisper: ")
	log.SetFlags(log.LstdFlags | log.LUTC | log.Lmicroseconds)

	log.Printf("starting on %s...", addr)

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

	// Initialize Firebase app
	firebaseApp, err := firebase.NewApp(context.Background(), nil)
	if err != nil {
		log.Panicln(fmt.Errorf("failed to initialize Firebase: %v", err))
	}
	firebaseMsg, err := firebaseApp.Messaging(context.Background())
	if err != nil {
		log.Panicln(fmt.Errorf("failed to initialize Firebase Messaging: %v", err))
	}
	singleton.FirebaseMsg = firebaseMsg

	// Initialize Hub
	singleton.Hub = utils.NewHub()
	go singleton.Hub.Run()

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

	// Create PostgreSQL triggers
	database.CreatePostgresTriggers("./sqlScripts/create_user_trigger.sql")
	if err != nil {
		log.Panicln(err)
	}

	// Send mail on new invite creation
	go func() {
		database.SubscribeNewInvites()
	}()

	// Create admin invite if admin does not exist
	err = database.CreateAdminInvite()
	if err != nil {
		log.Panicln(err)
	}

	// Channel for blocking exiting main function
	waitForSignal := make(chan bool)
	<-waitForSignal
}
