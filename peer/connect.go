package peer

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

func (p *HostInfo) Connect(ctx context.Context, h host.Host, peerInfo peer.AddrInfo, relayID peer.ID) error {
	if err := h.Connect(ctx, peerInfo); err != nil {
		return fmt.Errorf("direct connect failed: %w", err)
	}
	addrStr := fmt.Sprintf("/p2p/%s/p2p-circuit/p2p/%s", relayID, peerInfo.ID)
	targetRelayaddr, err := ma.NewMultiaddr(addrStr)
	if err != nil {
		return fmt.Errorf("invalid relay multiaddr: %w", err)
	}
	targetPeer := peer.AddrInfo{
		ID:    peerInfo.ID,
		Addrs: []ma.Multiaddr{targetRelayaddr},
	}
	if err := h.Connect(ctx, targetPeer); err != nil {
		return fmt.Errorf("relay connect failed: %w", err)
	}
	fmt.Println("success connection")
	return nil
}
