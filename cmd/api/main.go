package main

import (
	"log"
	"net/http"

	authh "volumetric-backend/internal/auth/handler" //  alias "authh" for AuthHandler
	authr "volumetric-backend/internal/auth/repo"    //  alias for repo
	"volumetric-backend/internal/handler"
	// domain "volumetric-backend/internal/handler" //  alias "domain" for your original ScanHandler etc.
	"volumetric-backend/internal/repo"

	"volumetric-backend/internal/config"
	"volumetric-backend/internal/db"
	"volumetric-backend/internal/router"
)

func main() {
	cfg := config.Load()      //It reads env file, or environment variables, or config files
	dbConn := db.Connect(cfg) //gateway to run db queries

	//  DB check
	if err := dbConn.Ping(); err != nil {
		log.Fatalf("Database ping failed: %v", err)
	}
	log.Println("Database connection OK")

	// Repositories(Repo is layer that directly talks to db)
	authRepo := authr.NewAuthRepo(dbConn)
	scanRepo := repo.NewScanRepo(dbConn)
	coordRepo := repo.NewCoordinateRepo(dbConn)
	calc := handler.NewMockVolumeCalculator()

	// Handlers(function that handles incoming HTTP requests) — used aliases to avoid conflict
	// scanHandler := &domain.ScanHandler{DB: dbConn}
	authHandler := authh.NewAuthHandler(authRepo)
	scanHandler := handler.NewScanHandler(scanRepo)
	coordHandler := handler.NewCoordinateHandler(coordRepo, scanRepo)
	volumeHandler := handler.NewVolumeHandler(scanRepo, coordRepo, calc)

	// Router decides which URL path maps to which handler
	r := router.Setup(scanHandler, authHandler, authRepo, coordHandler, volumeHandler)

	log.Printf("Server running on :%s\n", cfg.ServerPort)
	log.Fatal(http.ListenAndServe(":"+cfg.ServerPort, r))

	
}

// (1) It starts a web server that:

// Load config

// Connect to database

// Create repo (DB layer)

// Create handlers (business logic layer)

// Setup routes (URL mapping)

// Start server
