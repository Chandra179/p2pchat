package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"p2p/config"
	"p2p/peer"
	"p2p/relay"
	"strings"
)

func main() {
	mode := flag.String("mode", "", "Mode to run: 'relay' or 'peer' (required for startup)")
	flag.Parse()

	if *mode == "" {
		fmt.Fprintln(os.Stderr, "Error: --mode flag is required ('relay' or 'peer')")
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
				if err := peerInfo.ConnectPeer(targetPeerID); err != nil {
					fmt.Printf("Failed to connect to peer: %v\n", err)
				}
			case "send":
				if len(fields) < 2 {
					fmt.Println("Usage: send <message>")
					continue
				}
				msg := strings.Join(fields[1:], " ")
				if err := peerInfo.SendMessage(msg); err != nil {
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
