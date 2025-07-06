package peer

import (
	"context"
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

type ConnectOption func(ctx context.Context, h host.Host) error

func (p *PeerInfo) ConnectWithFallback(ctx context.Context, h host.Host, opts ...ConnectOption) error {
	for _, opt := range opts {
		if err := opt(ctx, h); err == nil {
			return nil
		} else {
			log.Println("Fallback failed:", err)
		}
	}
	return fmt.Errorf("all connection methods failed")
}

// Direct connect using standard peer.AddrInfo
func (p *PeerInfo) WithDirect(peerInfo peer.AddrInfo) ConnectOption {
	return func(ctx context.Context, h host.Host) error {
		log.Println("Trying direct (auto hole punch + UPnP)")
		if err := h.Connect(ctx, peerInfo); err != nil {
			return fmt.Errorf("direct connect failed: %w", err)
		}
		return nil
	}
}

// Relay circuit connect using relay peer ID and target peer ID
func (p *PeerInfo) WithRelayFallback(relayID peer.ID, targetPeerID string) ConnectOption {
	return func(ctx context.Context, h host.Host) error {
		log.Println("Trying relay circuit fallback")
		addrStr := fmt.Sprintf("/p2p/%s/p2p-circuit/p2p/%s", relayID, targetPeerID)
		targetRelayaddr, err := ma.NewMultiaddr(addrStr)
		if err != nil {
			return fmt.Errorf("invalid relay multiaddr: %w", err)
		}

		targetID, err := peer.Decode(targetPeerID)
		if err != nil {
			return fmt.Errorf("invalid target peer ID: %w", err)
		}
		targetPeer := peer.AddrInfo{
			ID:    targetID,
			Addrs: []ma.Multiaddr{targetRelayaddr},
		}
		if err := h.Connect(ctx, targetPeer); err != nil {
			return fmt.Errorf("relay connect failed: %w", err)
		}
		return nil
	}
}
