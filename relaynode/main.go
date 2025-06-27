package main

import (
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"

	"github.com/libp2p/go-libp2p/core/peer"
)

func main() {
	relay1, err := libp2p.New()
	if err != nil {
		log.Printf("Failed to create relay1: %v", err)
		return
	}
	_, err = relay.New(relay1)
	if err != nil {
		log.Printf("Failed to instantiate the relay: %v", err)
		return
	}
	relay1info := peer.AddrInfo{
		ID: relay1.ID(),
	}
	fmt.Println(relay1info.ID)
	select {}
}
