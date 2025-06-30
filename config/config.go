package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	PeerPort     string
	RelayPort    string
	RelayID      string
	PeerID       string
	PublicIP     string
	TargetPeerID string
}

func LoadConfig() *Config {
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("Failed to load .env: %v", err)
	}
	return &Config{
		PeerPort:     os.Getenv("PEER_TCP_PORT"),
		RelayPort:    os.Getenv("RELAY_TCP_PORT"),
		RelayID:      os.Getenv("RELAY_ID"),
		PeerID:       os.Getenv("PEER_ID"),
		PublicIP:     os.Getenv("PUBLIC_IP"),
		TargetPeerID: os.Getenv("TARGET_PEER_ID"),
	}
}
