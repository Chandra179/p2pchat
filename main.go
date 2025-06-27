package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/libp2p/go-libp2p"

	"github.com/joho/godotenv"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	ma "github.com/multiformats/go-multiaddr"
)

func main() {
	relayTest()
	// peerTest()
}

func peerTest() {
	// Load .env file
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("Failed to load .env: %v", err)
		return
	}
	relayIP := os.Getenv("RELAY_IP")
	relayPort := os.Getenv("RELAY_TCP_PORT")
	relayID := os.Getenv("RELAY_ID")
	if relayIP == "" || relayPort == "" || relayID == "" {
		log.Printf("RELAY_IP, RELAY_TCP_PORT, or RELAY_ID not set in .env")
		return
	}
	unreachable1, err := libp2p.New(
		libp2p.NoListenAddrs,
		// Usually EnableRelay() is not required as it is enabled by default
		// but NoListenAddrs overrides this, so we're adding it in explicitly again.
		libp2p.EnableRelay(),
	)
	if err != nil {
		log.Printf("Failed to create unreachable1: %v", err)
		return
	}
	addr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%s", relayIP, relayPort))
	if err != nil {
		log.Printf("Failed to parse multiaddr: %v", err)
		return
	}
	relay1info := peer.AddrInfo{
		ID:    peer.ID(relayID),
		Addrs: []ma.Multiaddr{addr},
	}
	if err := unreachable1.Connect(context.Background(), relay1info); err != nil {
		log.Printf("Failed to connect unreachable1 and relay1: %v", err)
		return
	}
	select {}
}

func relayTest() {
	// Load .env file
	if err := godotenv.Load(".env"); err != nil {
		fmt.Printf("Failed to load .env: %v\n", err)
		return
	}
	publicIP := os.Getenv("PUBLIC_IP")
	relayPort := os.Getenv("RELAY_TCP_PORT")
	if publicIP == "" || relayPort == "" {
		fmt.Println("PUBLIC_IP or RELAY_TCP_PORT not set in .env")
		return
	}
	listenAddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%s", relayPort)
	advertiseAddr := fmt.Sprintf("/ip4/%s/tcp/%s", publicIP, relayPort)

	relay1, err := libp2p.New(
		libp2p.ListenAddrStrings(listenAddr),
		libp2p.AddrsFactory(func(addrs []ma.Multiaddr) []ma.Multiaddr {
			// Override with the public IP
			adv, _ := ma.NewMultiaddr(advertiseAddr)
			return []ma.Multiaddr{adv}
		}),
	)
	if err != nil {
		fmt.Printf("Failed to create relay1: %v\n", err)
		return
	}
	_, err = relay.New(relay1)
	if err != nil {
		fmt.Printf("Failed to instantiate the relay: %v\n", err)
		return
	}
	relayinfo := peer.AddrInfo{
		ID:    relay1.ID(),
		Addrs: relay1.Addrs(),
	}
	fmt.Println(relayinfo.ID)
	fmt.Println(relayinfo.Addrs)
	select {}
}
