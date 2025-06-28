package p2p

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
	privKey, err := decodePrivateKey(cfg.RelayID)
	if err != nil {
		fmt.Printf("Failed to decode private key: %v\n", err)
		return
	}
	node, err := libp2p.New(
		libp2p.NoListenAddrs,
		libp2p.EnableRelay(),
	)
	if err != nil {
		log.Printf("Failed to create node: %v", err)
		return
	}
	addr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%s", cfg.RelayIP, cfg.RelayPort))
	if err != nil {
		log.Printf("Failed to parse multiaddr: %v", err)
		return
	}
	peerID, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		log.Printf("Failed to derive peer ID from private key: %v", err)
		return
	}
	relayinfo := peer.AddrInfo{
		ID:    peerID,
		Addrs: []ma.Multiaddr{addr},
	}
	if err := node.Connect(context.Background(), relayinfo); err != nil {
		log.Printf("Failed to connect unreachable1 and relay1: %v", err)
		return
	}
	fmt.Println("Connected to relay")
	select {}
}
