package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Settings that change between environments. In Phase 3 these become
// environment variables (PORT, BASE_URL) with these values as defaults.

type Config struct {
	Port    string
	Host    string
	BaseURL string
}

func NewConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	return &Config{
		Port:    os.Getenv("PORT"),
		Host:    os.Getenv("HOST"),
		BaseURL: os.Getenv("BASE_URL"),
	}
}
