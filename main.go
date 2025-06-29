package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"p2p/config"
	"p2p/peer"
	"p2p/relay"
)

func main() {
	mode := flag.String("mode", "", "Mode to run: 'relay', 'peer', or 'con' (required)")
	flag.Parse()

	if *mode == "" {
		fmt.Fprintln(os.Stderr, "Error: --mode flag is required ('relay', 'peer', or 'con')")
		os.Exit(1)
	}

	cfg := config.LoadConfig()
	peerInfo := peer.PeerInfo{}

	switch *mode {
	case "relay":
		fmt.Println("Running in relay mode...")
		relay.RunRelay(cfg)
	case "peer":
		fmt.Println("Running in peer mode...")
		p, err := peer.RunPeer(cfg)
		if err != nil {
			log.Fatalf("Failed to run peer: %v", err)
		}
		peerInfo = *p
	case "con":
		fmt.Println("Connecting to peer...")
		args := flag.Args()
		if len(args) < 1 {
			log.Fatalf("targetpeerid argument is required in 'con' mode (usage: --mode=con targetpeerid)")
		}
		targetPeerID := args[0]
		if err := peerInfo.ConnectPeer(targetPeerID); err != nil {
			log.Fatalf("Failed to connect to peer: %v", err)
		}
	case "send":
		fmt.Println("Sending message to peer...")
		args := flag.Args()
		if len(args) < 1 {
			log.Fatalf("message arguments are required in 'send' mode (usage: --mode=send message)")
		}
		message := args[0]
		if err := peerInfo.SendMessage(message); err != nil {
			log.Fatalf("Failed to send message: %v", err)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown mode: %s\n", *mode)
		os.Exit(1)
	}

	select {}
}
