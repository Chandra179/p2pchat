package node

import (
	"context"
	"fmt"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	host "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

type P2PNode struct {
	Host       host.Host
	Ctx        context.Context
	PrivateKey crypto.PrivKey
}

func NewP2PNode(ctx context.Context) (*P2PNode, error) {
	h, err := libp2p.New()
	if err != nil {
		return nil, err
	}

	node := &P2PNode{
		Host:       h,
		Ctx:        ctx,
		PrivateKey: h.Peerstore().PrivKey(h.ID()),
	}

	err = setupMDNS(h, &discoveryNotifee{Host: h})
	if err != nil {
		fmt.Println("⚠️ mDNS discovery failed:", err)
	}

	return node, nil
}

func (n *P2PNode) SetStreamHandler(protocolID protocol.ID, handler network.StreamHandler) {
	n.Host.SetStreamHandler(protocolID, handler)
}

func (n *P2PNode) Info() {
	fmt.Println("🧭 Peer ID:", n.Host.ID())
	for _, addr := range n.Host.Addrs() {
		fmt.Printf("🌐 Addr: %s/p2p/%s\n", addr, n.Host.ID())
	}
}

// --- mDNS Support ---

type discoveryNotifee struct {
	Host host.Host
}

func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	fmt.Println("🔍 Found peer:", pi.ID)
	if err := n.Host.Connect(context.Background(), pi); err != nil {
		fmt.Println("❌ Failed to connect:", err)
	}
}

func setupMDNS(h host.Host, notifee mdns.Notifee) error {
	service := mdns.NewMdnsService(h, "p2p-mdns", notifee)
	return service.Start()
}
