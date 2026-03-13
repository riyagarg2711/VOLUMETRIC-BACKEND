package main

import (
	"log"
	"net/http"

	"volumetric-backend/internal/app"
	"volumetric-backend/internal/config"
	"volumetric-backend/internal/db"
	"volumetric-backend/internal/router"
)

func main() {
	cfg := config.Load()
	dbConn := db.Connect(cfg)

	if err := dbConn.Ping(); err != nil {
		log.Fatalf("Database ping failed: %v", err)
	}
	log.Println("Database connection OK")

	a := app.New(dbConn)
	r := router.Setup(a)

	log.Printf("Server running on :%s\n", cfg.ServerPort)
	log.Fatal(http.ListenAndServe(":"+cfg.ServerPort, r))
}

// (1) It starts a web server that:

// Load config

// Connect to database

// Create repo (DB layer)

// Create handlers (business logic layer)

// Setup routes (URL mapping)
