package peer

import (
	"context"
	"fmt"
	"log"
	"p2p/config"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

type PeerEnv struct {
	RelayIP   string
	RelayPort string
	RelayID   string
}

func RunPeer(cfg *config.Config) {
	if cfg.RelayIP == "" || cfg.RelayPort == "" || cfg.RelayID == "" {
		log.Printf("RELAY_IP, RELAY_TCP_PORT, or RELAY_ID not set in config")
		return
	}
	unreachable1, err := libp2p.New(
		libp2p.NoListenAddrs,
		libp2p.EnableRelay(),
	)
	if err != nil {
		log.Printf("Failed to create unreachable1: %v", err)
		return
	}
	addr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%s", cfg.RelayIP, cfg.RelayPort))
	if err != nil {
		log.Printf("Failed to parse multiaddr: %v", err)
		return
	}
	ID, err := peer.Decode(cfg.RelayID)
	if err != nil {
		log.Fatalf("Invalid peer ID: %v", err)
	}
	relayinfo := peer.AddrInfo{
		ID:    ID,
		Addrs: []ma.Multiaddr{addr},
	}
	if err := unreachable1.Connect(context.Background(), relayinfo); err != nil {
		log.Printf("Failed to connect unreachable1 and relay1: %v", err)
		return
	}
	fmt.Println("Connected to relay")
	select {}
}
