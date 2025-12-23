package main

import (
	"log"
	"net/http"

	"volumetric-backend/internal/config"
	"volumetric-backend/internal/db"
	"volumetric-backend/internal/handler"
	"volumetric-backend/internal/router"
)

func main() {
	cfg := config.Load()

	dbConn := db.Connect(cfg)

	h := &handler.CoordinateHandler{
		DB: dbConn,
	}

	r := router.Setup(h)

	log.Printf("Server running on :%s\n", cfg.ServerPort)
	log.Fatal(http.ListenAndServe(":"+cfg.ServerPort, r))
}
