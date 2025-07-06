package config

import (
	"bufio"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	PeerPort       string
	RelayPort      string
	RelayID        string
	PeerID         string
	RelayIP        string
	BootstrapAddrs []string
}

func LoadConfig() *Config {
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("Failed to load .env: %v", err)
	}

	// Read bootstrap addresses from CSV
	bootstrapAddrs := []string{}
	file, err := os.Open("bootstrappeer.csv")
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			addr := scanner.Text()
			if addr != "" {
				bootstrapAddrs = append(bootstrapAddrs, addr)
			}
		}
		if err := scanner.Err(); err != nil {
			log.Printf("Error reading bootstrappeer.csv: %v", err)
		}
	} else {
		log.Printf("Could not open bootstrappeer.csv: %v", err)
	}

	return &Config{
		PeerPort:       os.Getenv("PEER_TCP_PORT"),
		RelayPort:      os.Getenv("RELAY_TCP_PORT"),
		RelayID:        os.Getenv("RELAY_ID"),
		PeerID:         os.Getenv("PEER_ID"),
		RelayIP:        os.Getenv("RELAY_IP"),
		BootstrapAddrs: bootstrapAddrs,
	}
}
