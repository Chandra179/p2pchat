package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"p2p/chat"
	"p2p/config"
	mypeer "p2p/peer"
	"p2p/relay"
	"strings"

	ma "github.com/multiformats/go-multiaddr"

	"github.com/libp2p/go-libp2p/core/peer"
)

func main() {
	mode := flag.String("mode", "", "Mode to run: 'relay' or 'peer' (required for startup)")
	flag.Parse()

	if *mode == "" {
		fmt.Fprintln(os.Stderr, "Error: --mode flag is required ('relay' or 'peer')")
		os.Exit(1)
	}

	cfg := config.LoadConfig()

	switch *mode {
	case "relay":
		fmt.Println("Running in relay mode...")
		relay.RunRelay(cfg)
	case "peer":
		fmt.Println("Running in peer mode...")
		p, err := mypeer.InitPeer(cfg)
		if err != nil {
			log.Fatalf("Failed to init peer: %v", err)
		}
		// Enter REPL for commands
		fmt.Println("Peer started. Enter commands: 'con <targetpeerid>', 'send <message>', or 'exit'.")
		scanner := bufio.NewScanner(os.Stdin)
		for {
			fmt.Print("> ")
			if !scanner.Scan() {
				break
			}
			input := scanner.Text()
			fields := strings.Fields(input)
			if len(fields) == 0 {
				continue
			}
			switch fields[0] {
			case "con":
				if len(fields) < 2 {
					fmt.Println("Usage: con <targetpeerid>")
					continue
				}
				targetPeerID := fields[1]
				decodedPeerID, err := peer.Decode(targetPeerID)
				if err != nil {
					fmt.Printf("Invalid peer ID: %v\n", err)
					continue
				}
				peerInfo := peer.AddrInfo{
					ID:    decodedPeerID,
					Addrs: []ma.Multiaddr{},
				}
				if err := p.ConnectWithFallback(
					context.Background(),
					p.Host,
					p.WithDirect(peerInfo),
					p.WithRelayFallback(cfg.RelayID, targetPeerID)); err != nil {
					fmt.Printf("Failed to connect to peer: %v\n", err)
				}
			case "find":
				dm, err := mypeer.InitDHT(context.Background(), p.Host, cfg.BootstrapAddrs)
				if err != nil {
					fmt.Printf("Failed to init DHT: %v\n", err)
				}
				peers, err := dm.FindPeers(context.Background(), "/customprotocol")
				if err != nil {
					fmt.Printf("Failed to find peers: %v\n", err)
					continue
				}
				for peer := range peers {
					fmt.Println("Found peer:", peer.ID)
				}
			case "send":
				if len(fields) < 2 {
					fmt.Println("Usage: send <message>")
					continue
				}
				msg := strings.Join(fields[1:], " ")
				if err := chat.SendPrivateMessage("/customprotocol", p.Host, nil, p.TargetPeerID, msg); err != nil {
					fmt.Printf("Failed to send message: %v\n", err)
				}
			case "exit":
				fmt.Println("Exiting peer...")
				os.Exit(0)
			default:
				fmt.Println("Unknown command. Use 'con <targetpeerid>', 'send <message>', or 'exit'.")
			}
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown mode: %s\n", *mode)
		os.Exit(1)
	}

	select {}
}
