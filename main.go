package main

import (
	"flag"
	"fmt"
	"os"
	"p2p/config"
	"p2p/peer"
	"p2p/relay"
)

func main() {
	mode := flag.String("mode", "", "Mode to run: 'relay' or 'peer' (required)")
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
		peer.RunPeer(cfg)
	default:
		fmt.Fprintf(os.Stderr, "Unknown mode: %s\n", *mode)
		os.Exit(1)
	}

	select {}
}
