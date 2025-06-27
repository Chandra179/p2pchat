package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	RelayIP   string
	RelayPort string
	RelayID   string
	PublicIP  string
}

func LoadConfig() *Config {
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("Failed to load .env: %v", err)
	}
	return &Config{
		RelayIP:   os.Getenv("RELAY_IP"),
		RelayPort: os.Getenv("RELAY_TCP_PORT"),
		RelayID:   os.Getenv("RELAY_ID"),
		PublicIP:  os.Getenv("PUBLIC_IP"),
	}
}
