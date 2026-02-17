package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	ServerPort string
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil {
		
		log.Println("No .env file found, using system env")
	}

	return &Config{
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
		DBSSLMode:  os.Getenv("DB_SSLMODE"),
		ServerPort: os.Getenv("SERVER_PORT"),
	}
}

// (2) This file is responsible for loading configuration settings (like database details and server port).

// Defines a Config structure

// Reads values from .env or system environment

// Stores them in a struct

// Returns the config to be used in the app