package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"p2p/node"
	"p2p/protocol"
)

const PrivateProtocolID = "/p2p-chat/1.0.0-private"

func main() {
	ctx := context.Background()
	node, err := node.NewP2PNode(ctx)
	if err != nil {
		panic(err)
	}
	node.Info()
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
	node.SetStreamHandler(PrivateProtocolID, func(s network.Stream) {
		protocol.HandlePrivateMessage(s, node.PrivateKey)
	})

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			text := scanner.Text()

			// Get connected peers
			peers := node.Host.Network().Peers()
			if len(peers) == 0 {
				fmt.Println("❌ No peers connected.")
				continue
			}
			// Send to first connected peer
			targetPeer := peers[0]

			err := protocol.SendPrivateMessage(PrivateProtocolID, node.Host, node.PrivateKey, targetPeer, text)
			if err != nil {
				fmt.Println("❌ Failed to send:", err)
			}
		}
	}()
}
