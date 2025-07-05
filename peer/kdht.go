package peer

import (
	"context"
	"fmt"
	"strings"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	discovery "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/multiformats/go-multiaddr"
)

type DHTManager struct {
	DHT       *dht.IpfsDHT
	Discovery *discovery.RoutingDiscovery
}

// ParseMultiAddr parses a string multiaddress into AddrInfo
func ParseMultiAddr(addr string) (*peer.AddrInfo, error) {
	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return nil, err
	}
	ai, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return nil, err
	}
	return ai, nil
}

// InitDHT initializes Kademlia DHT with bootstrap peers
func InitDHT(ctx context.Context, h host.Host, bootstrapAddrs []string) (*DHTManager, error) {
	kademliaDHT, err := dht.New(ctx, h, dht.Mode(dht.ModeAuto))
	if err != nil {
		return nil, err
	}
	for _, addr := range bootstrapAddrs {
		if strings.TrimSpace(addr) == "" {
			continue
		}
		peerInfo, err := ParseMultiAddr(addr)
		if err != nil {
			fmt.Println("Invalid bootstrap addr:", addr, err)
			continue
		}
		err = h.Connect(ctx, *peerInfo)
		if err != nil {
			fmt.Println("Failed to connect to bootstrap peer:", peerInfo.ID, err)
			continue
		}
	}
	if err := kademliaDHT.Bootstrap(ctx); err != nil {
		return nil, err
	}
	return &DHTManager{
		DHT:       kademliaDHT,
		Discovery: discovery.NewRoutingDiscovery(kademliaDHT),
	}, nil
}

// AdvertiseHost makes the host discoverable on the DHT under the given tag
func (d *DHTManager) AdvertiseHost(ctx context.Context, rendezvous string) error {
	ttl, err := d.Discovery.Advertise(ctx, rendezvous)
	if err != nil {
		return err
	}
	fmt.Println("Successfully advertised. TTL:", ttl)
	return nil
}

// FindPeers discovers peers by rendezvous string
func (d *DHTManager) FindPeers(ctx context.Context, rendezvous string) (<-chan peer.AddrInfo, error) {
	peerChan, err := d.Discovery.FindPeers(ctx, rendezvous)
	if err != nil {
		return nil, err
	}
	return peerChan, nil
}

// Expose routing interface to host options
func (d *DHTManager) Routing() routing.Routing {
	return d.DHT
}
