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
	"p2p/cryptoutils"
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

	var foundPeers map[string][]ma.Multiaddr

	switch *mode {
	case "relay":
		fmt.Println("Running in relay mode...")
		relay.RunRelay(cfg)
	case "peer":
		fmt.Println("Running in peer mode...")
		p, err := mypeer.InitPeerHost(cfg)
		if err != nil {
			log.Fatalf("Failed to init peer: %v", err)
		}
		p.ConnectAndReserveRelay(cfg.RelayID)
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
					fmt.Println("Usage: con <targetpeerid>|all")
					continue
				}
				targetPeerID := fields[1]
				decodedPeerID, err := peer.Decode(targetPeerID)
				if err != nil {
					fmt.Printf("Invalid peer ID: %v\n", err)
					continue
				}
				relayPrivKey, err := cryptoutils.DecodeBase64Key(cfg.RelayID)
				if err != nil {
					fmt.Printf("Invalid relay ID (base64 decode): %v\n", err)
					continue
				}
				relayID, err := peer.IDFromPrivateKey(relayPrivKey)
				if err != nil {
					fmt.Printf("Invalid relay ID: %v\n", err)
					continue
				}
				peerInfo := peer.AddrInfo{
					ID:    decodedPeerID,
					Addrs: foundPeers[targetPeerID], // use foundPeers if available
				}
				if err := p.ConnectWithFallback(
					context.Background(),
					p.Host,
					p.WithDirect(peerInfo),
					p.WithRelayFallback(relayID, targetPeerID)); err != nil {
					fmt.Printf("Failed to connect to peer: %v\n", err)
				}
			case "find":
				dm, err := mypeer.InitDHT(context.Background(), p.Host)
				if err != nil {
					fmt.Printf("Failed to init DHT: %v\n", err)
				}
				err = dm.AdvertiseHost(context.Background(), "/customprotocol")
				if err != nil {
					fmt.Printf("Failed to advertise host: %v\n", err)
				}
				peers, err := dm.FindPeers(context.Background(), "/customprotocol")
				if err != nil {
					fmt.Printf("Failed to find peers: %v\n", err)
					continue
				}
				// Store found peers in memory, excluding self
				if foundPeers == nil {
					foundPeers = make(map[string][]ma.Multiaddr)
				}
				for peer := range peers {
					if peer.ID == p.Host.ID() {
						continue // skip self
					}
					foundPeers[peer.ID.String()] = peer.Addrs
					fmt.Println("Found peer:", peer.ID)
					fmt.Println("Found peer:", peer.Addrs)
				}
				fmt.Printf("Total found peers (excluding self): %d\n", len(foundPeers))
			case "send":
				if len(fields) < 3 {
					fmt.Println("Usage: send <targetpeerid> <message>")
					continue
				}
				targetPeerIDStr := fields[1]
				msg := strings.Join(fields[2:], " ")
				decodedPeerID, err := peer.Decode(targetPeerIDStr)
				if err != nil {
					fmt.Printf("Invalid peer ID: %v\n", err)
					continue
				}
				privKey, err := cryptoutils.DecodeBase64Key(cfg.PeerID)
				if err != nil {
					fmt.Printf("Failed to decode private key: %v\n", err)
					return
				}
				if err := chat.SendPrivateMessage("/customprotocol", p.Host, privKey, decodedPeerID, msg); err != nil {
					fmt.Printf("Failed to send message: %v\n", err)
				}
			case "genkey":
				key, err := cryptoutils.GenerateEd25519Key()
				if err != nil {
					fmt.Printf("Failed to generate key: %v\n", err)
				}
				fmt.Println(key)
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
