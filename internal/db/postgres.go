package db

import (
	"database/sql"
	"fmt"
	"log"

	"volumetric-backend/internal/config"

	_ "github.com/lib/pq"
)

func Connect(cfg *config.Config) *sql.DB {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBSSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	log.Println("Postgres connected")
	return db
}


// (3) This file is responsible for connecting your Go app to a PostgreSQL database.

// Takes database settings

// Builds a connection string

// Connects to PostgreSQL

// Checks if connection works

// Returns database connection