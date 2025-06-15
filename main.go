// main.go
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"p2p/node"
	"p2p/protocol"
)

const PrivateProtocolID = "/p2p-chat/1.0.0-private-ephemeral"

func main() {
	ctx := context.Background()
	node, err := node.NewP2PNode(ctx)
	if err != nil {
		panic(err)
	}

	node.Info()
	fmt.Println("🔐 Starting encrypted P2P chat with ephemeral session keys...")
	fmt.Println("📝 Commands:")
	fmt.Println("  - Type messages to send to connected peers")
	fmt.Println("  - '/connect <multiaddr>' to connect to a peer")
	fmt.Println("  - '/peers' to list connected peers")
	fmt.Println("  - '/quit' to exit")
	fmt.Println()

	RunPrivateChat(ctx, node)
	select {}
}

func RunPubSub(ctx context.Context, node *node.P2PNode) {
	pubsubService, err := protocol.NewPubSubService(ctx, node.Host)
	if err != nil {
		log.Fatal(err)
	}
	topicName := "chatroom"
	_, sub, err := pubsubService.JoinTopic(topicName)
	if err != nil {
		log.Fatal(err)
	}
	pubsubService.ListenToTopic(sub)

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			text := scanner.Text()
			msg := protocol.NewChatMessage(node.Host.ID().String(), text)
			err := msg.Sign(node.Host.Peerstore().PrivKey(node.Host.ID()))
			if err != nil {
				log.Println("Failed to sign message:", err)
				return
			}
			data, _ := msg.Marshal()
			pubsubService.Publish(topicName, data)
		}
	}()
}

func RunSimpleChat(ctx context.Context, node *node.P2PNode) {
	node.SetStreamHandler(protocol.ProtocolID, protocol.ChatStreamHandler)

	if len(os.Args) > 1 {
		// Dial a peer
		addr, _ := multiaddr.NewMultiaddr(os.Args[1])
		pi, _ := peer.AddrInfoFromP2pAddr(addr)
		node.Host.Connect(ctx, *pi)
		s, _ := node.Host.NewStream(ctx, pi.ID, protocol.ProtocolID)
		protocol.ChatStreamHandler(s)
	}
}

func RunPrivateChat(ctx context.Context, node *node.P2PNode) {
	// Set up the stream handler for encrypted messages
	node.SetStreamHandler(PrivateProtocolID, func(s network.Stream) {
		protocol.HandlePrivateMessage(s, node.PrivateKey)
	})

	// Handle user input
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			text := strings.TrimSpace(scanner.Text())

			if text == "" {
				continue
			}

			// Handle commands
			if strings.HasPrefix(text, "/") {
				handleCommand(ctx, node, text)
				continue
			}

			// Send message to connected peers
			peers := node.Host.Network().Peers()
			if len(peers) == 0 {
				fmt.Println("❌ No peers connected. Use '/connect <multiaddr>' to connect to a peer.")
				continue
			}

			// Send to all connected peers
			for _, targetPeer := range peers {
				go func(peerID peer.ID) {
					err := protocol.SendPrivateMessage(PrivateProtocolID, node.Host, node.PrivateKey, peerID, text)
					if err != nil {
						fmt.Printf("❌ Failed to send to %s: %v\n", peerID.String()[:8], err)
					}
				}(targetPeer)
			}
		}
	}()
}

func handleCommand(ctx context.Context, node *node.P2PNode, command string) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "/connect":
		if len(parts) < 2 {
			fmt.Println("❌ Usage: /connect <multiaddr>")
			return
		}

		addr, err := multiaddr.NewMultiaddr(parts[1])
		if err != nil {
			fmt.Printf("❌ Invalid multiaddr: %v\n", err)
			return
		}

		pi, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			fmt.Printf("❌ Failed to parse peer info: %v\n", err)
			return
		}

		fmt.Printf("🔄 Connecting to %s...\n", pi.ID.String()[:8])
		if err := node.Host.Connect(ctx, *pi); err != nil {
			fmt.Printf("❌ Failed to connect: %v\n", err)
			return
		}

		fmt.Printf("✅ Connected to %s\n", pi.ID.String()[:8])

	case "/peers":
		peers := node.Host.Network().Peers()
		if len(peers) == 0 {
			fmt.Println("📭 No peers connected")
			return
		}

		fmt.Printf("👥 Connected peers (%d):\n", len(peers))
		for i, peerID := range peers {
			fmt.Printf("  %d. %s\n", i+1, peerID.String()[:8])
		}

	case "/quit":
		fmt.Println("👋 Goodbye!")
		os.Exit(0)

	default:
		fmt.Printf("❌ Unknown command: %s\n", parts[0])
		fmt.Println("📝 Available commands:")
		fmt.Println("  - /connect <multiaddr>")
		fmt.Println("  - /peers")
		fmt.Println("  - /quit")
	}
}
