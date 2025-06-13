package main

import (
	"bufio"
	"context"
	"log"
	"os"

	"p2p/node"
	"p2p/pubsub"
)

func main() {
	ctx := context.Background()

	p2pNode, err := node.NewP2PNode(ctx)
	if err != nil {
		panic(err)
	}
	p2pNode.Info()
	// p2pNode.SetStreamHandler(protocol.ProtocolID, protocol.ChatStreamHandler)

	pubsubService, err := pubsub.NewPubSubService(ctx, p2pNode.Host)
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
			pubsubService.Publish(topicName, []byte(text))
		}
	}()

	// if len(os.Args) > 1 {
	// 	// Dial a peer
	// 	addr, _ := multiaddr.NewMultiaddr(os.Args[1])
	// 	pi, _ := peer.AddrInfoFromP2pAddr(addr)
	// 	p2pNode.Host.Connect(ctx, *pi)
	// 	s, _ := p2pNode.Host.NewStream(ctx, pi.ID, protocol.ProtocolID)
	// 	protocol.ChatStreamHandler(s)
	// }

	select {}
}
