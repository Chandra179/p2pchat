package config

import (
	"fmt"
	"os"
	"p2p/cryptoutils"

	"github.com/joho/godotenv"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

type Config struct {
	PeerPort    string
	PeerID      string
	PeerPrivKey crypto.PrivKey // Will be used as peer ID
	//
	RelayPort    string
	RelayID      peer.ID
	RelayPrivKey crypto.PrivKey // WIll be used as peer ID
	RelayIP      string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(".env"); err != nil {
		fmt.Printf("Failed to load .env: %v", err)
	}
	peerPrivKey, err := cryptoutils.DecodeBase64Key(os.Getenv("PEER_ID"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode relay id: %w", err)
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
		PeerPort:    os.Getenv("PEER_TCP_PORT"),
		PeerID:      os.Getenv("PEER_ID"),
		PeerPrivKey: peerPrivKey,
		//
		RelayPort:    os.Getenv("RELAY_TCP_PORT"),
		RelayID:      relayID,
		RelayPrivKey: relayPrivKey,
		RelayIP:      os.Getenv("RELAY_IP"),
	}, nil
}
