package main

import (
	"p2p/config"
	"p2p/relay"
)

func main() {
	cfg := config.LoadConfig()
	relay.RunRelay(cfg)
	// peer.RunPeer(cfg)
}
