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
	"os"
	"os/signal"
	"syscall"
	"time"

	firebase "firebase.google.com/go/v4"
)

const addr = "0.0.0.0:8080"

//go:embed mailTemplates/registerAdmin.hbs
var registerAdminTemplate string

//go:embed mailTemplates/registerInvite.hbs
var registerInviteTemplate string

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

	// Store embedded mail templates
	singleton.RegisterAdminTemplate = registerAdminTemplate
	singleton.RegisterInviteTemplate = registerInviteTemplate

	// Setup SQLite
	database.InitSqlite()
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
	// Start HTTP server with timeouts
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	// Start server in separated goroutine, panic on error
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Panicln(err)
		}
	}()

	// Send mail on new invite creation
	go func() {
		database.SubscribeNewInvites()
	}()

	// Create admin invite if admin does not exist
	err = database.CreateAdminInvite()
	if err != nil {
		log.Panicln(err)
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("server forced to shutdown: %v", err)
	}

	// Close Valkey client
	singleton.Valkey.Close()

	log.Println("server exited")
}
