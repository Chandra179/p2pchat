package peer

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

func (p *PeerInfo) Connect(ctx context.Context, peerInfo peer.AddrInfo, relayID peer.ID) error {
	// Try direct connection first
	if err := p.Host.Connect(ctx, peerInfo); err == nil {
		fmt.Println("success direct connection")
		return nil
	} else {
		fmt.Println("direct connect failed, trying relay...")
	}

	// If direct fails, try relay connection
	addrStr := fmt.Sprintf("/p2p/%s/p2p-circuit/p2p/%s", relayID, peerInfo.ID)
	targetRelayaddr, err := ma.NewMultiaddr(addrStr)
	if err != nil {
		return fmt.Errorf("invalid relay multiaddr: %w", err)
	}
	targetPeer := peer.AddrInfo{
		ID:    peerInfo.ID,
		Addrs: []ma.Multiaddr{targetRelayaddr},
	}
	if err := p.Host.Connect(ctx, targetPeer); err != nil {
		return fmt.Errorf("relay connect failed: %w", err)
	}
	fmt.Println("success relay connection")
	return nil
}
