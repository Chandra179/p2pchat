package config

import (
	"fmt"
	"log"
	"os"
	"p2p/cryptoutils"

	"github.com/joho/godotenv"
	"github.com/libp2p/go-libp2p/core/peer"
)

type Config struct {
	PeerPort  string
	RelayPort string
	RelayID   peer.ID
	PeerID    string
	RelayIP   string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("Failed to load .env: %v", err)
	}

	relayPrivKey, err := cryptoutils.DecodeBase64Key(os.Getenv("RELAY_ID"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode relay id: %w", err)
	}
	relayID, err := peer.IDFromPrivateKey(relayPrivKey)
	if err != nil {
		return nil, fmt.Errorf("failed to extract id from priv key: %w", err)
	}

	return &Config{
		PeerPort:  os.Getenv("PEER_TCP_PORT"),
		RelayPort: os.Getenv("RELAY_TCP_PORT"),
		RelayID:   relayID,
		PeerID:    os.Getenv("PEER_ID"),
		RelayIP:   os.Getenv("RELAY_IP"),
	}, nil
}
