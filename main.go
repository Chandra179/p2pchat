package main

import (
	"p2p/config"
	"p2p/p2p"
)

func main() {
	cfg := config.LoadConfig()
	p2p.RunRelay(cfg)
	// p2p.RunPeer(cfg)
}
