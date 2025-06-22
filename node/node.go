package node

import (
	"context"
	"log"

	libp2p "github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	host "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
)

var publicRelays = []string{
	"/ip4/147.75.83.83/tcp/4001/p2p/12D3KooWH7JKykg5AMhGug5U5uNRt3U9B3SNRshHXY5Ue8bnzzSY",   // Protocol Labs relay
	"/ip4/147.75.109.245/tcp/4001/p2p/12D3KooWHkXoJ6hd9Uet4UtHoDDAcP9v7SGcXz8S2vC1A8e1tWwv", // Protocol Labs
}

type P2PNode struct {
	Host       host.Host
	Ctx        context.Context
	PrivateKey crypto.PrivKey
	PublicKey  crypto.PubKey
	DHT        *dht.IpfsDHT
}

func NewP2PNode(ctx context.Context) (*P2PNode, error) {
	var kadDHT *dht.IpfsDHT

	h, err := libp2p.New(
		libp2p.DefaultEnableRelay,
	)
	if err != nil {
		return nil, err
	}

	return &P2PNode{
		Host:       h,
		Ctx:        ctx,
		PrivateKey: h.Peerstore().PrivKey(h.ID()),
		PublicKey:  h.Peerstore().PubKey(h.ID()),
		DHT:        kadDHT,
	}, nil
}

// CheckRelayReachability attempts to connect to each relay address
func (n *P2PNode) CheckRelayReachability() {
	publicRelays := []string{
		"/ip4/147.75.80.110/tcp/4001/p2p/QmbFgm5zan8P6eWWmeyfncR5feYEMPbht5b1FW1C37aQ7aQ",
		"/ip4/147.75.195.153/tcp/4001/p2p/QmW9m57aiBDHAkKj9nmFSEn7QrcF1fZS4bipsTCHburei",
		"/ip4/147.75.70.221/tcp/4001/p2p/Qme8g49gm3q4Acp7xWBKg3nAa9fxZ1YmyDJdyGgoG6LsXh",
	}

	for _, addrStr := range publicRelays {
		addr, err := multiaddr.NewMultiaddr(addrStr)
		if err != nil {
			log.Printf("❌ Invalid multiaddr: %s", err)
			continue
		}

		pi, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			log.Printf("❌ Failed to parse peer info: %s", err)
			continue
		}

		log.Printf("🔌 Trying to connect to relay %s...", pi.ID.String())

		if err := n.Host.Connect(n.Ctx, *pi); err != nil {
			log.Printf("❌ Relay %s unreachable: %s", pi.ID.String(), err)
		} else {
			log.Printf("✅ Relay %s is reachable", pi.ID.String())
		}
	}
}

func (n *P2PNode) SetStreamHandler(protocolID protocol.ID, handler network.StreamHandler) {
	n.Host.SetStreamHandler(protocolID, handler)
}
