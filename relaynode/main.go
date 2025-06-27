package main

import (
	"fmt"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"

	"github.com/libp2p/go-libp2p/core/peer"
)

func main() {
	fmt.Println("aaaa")
	relay1, err := libp2p.New()
	fmt.Println("bbb")
	if err != nil {
		fmt.Printf("Failed to create relay1: %v", err)
		return
	}
	fmt.Println("ccc")
	_, err = relay.New(relay1)
	fmt.Println("ddd")
	if err != nil {
		fmt.Printf("Failed to instantiate the relay: %v", err)
		return
	}

	fmt.Println("eee")
	relay1info := peer.AddrInfo{
		ID:    relay1.ID(),
		Addrs: relay1.Addrs(),
	}
	fmt.Println("fff")
	fmt.Println(relay1info.ID)
	for _, addr := range relay1info.Addrs {
		fmt.Println(addr)
	}
	fmt.Println("ggg")
	select {}
}
