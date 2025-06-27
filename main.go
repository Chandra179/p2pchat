package main

import (
	"fmt"
	"os"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"

	"github.com/joho/godotenv"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

func main() {
	// Load .env file
	if err := godotenv.Load(".env"); err != nil {
		fmt.Printf("Failed to load .env: %v\n", err)
		return
	}
	publicIP := os.Getenv("PUBLIC_IP")
	if publicIP == "" {
		fmt.Println("PUBLIC_IP not set in .env")
		return
	}
	listenAddr := "/ip4/0.0.0.0/tcp/9000"
	advertiseAddr := "/ip4/" + publicIP + "/tcp/9000"

	relay1, err := libp2p.New(
		libp2p.ListenAddrStrings(listenAddr),
		libp2p.AddrsFactory(func(addrs []ma.Multiaddr) []ma.Multiaddr {
			// Override with the public IP
			adv, _ := ma.NewMultiaddr(advertiseAddr)
			return []ma.Multiaddr{adv}
		}),
	)
	if err != nil {
		fmt.Printf("Failed to create relay1: %v", err)
		return
	}
	_, err = relay.New(relay1)
	if err != nil {
		fmt.Printf("Failed to instantiate the relay: %v", err)
		return
	}

	relay1info := peer.AddrInfo{
		ID:    relay1.ID(),
		Addrs: relay1.Addrs(),
	}
	fmt.Println(relay1info.ID)
	for _, addr := range relay1info.Addrs {
		fmt.Println(addr)
	}
	select {}
}
