package node

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	host "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/core/routing"
	routingdisc "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	autorelay "github.com/libp2p/go-libp2p/p2p/host/autorelay"
	connmgr "github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/multiformats/go-multiaddr"
)

var bootstrapPeers = []string{
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
	// "/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
	// "/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
	// "/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
	// "/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
}

type P2PNode struct {
	Host       host.Host
	Ctx        context.Context
	PrivateKey crypto.PrivKey
	PublicKey  crypto.PubKey
	DHT        *dht.IpfsDHT
}

type ConnectionStats struct {
	UniqueIPs        []string
	TotalConnections int
	RelayConnections int
	ConnectedPeers   []peer.ID
}

func NewP2PNode(ctx context.Context) (*P2PNode, error) {
	cm, err := connmgr.NewConnManager(
		1, // Low watermark
		3, // High watermark
		connmgr.WithGracePeriod(time.Minute),
	)
	if err != nil {
		return nil, err
	}
	var kadDHT *dht.IpfsDHT
	h, err := libp2p.New(
		libp2p.EnableAutoRelayWithPeerSource(
			func(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
				ch := make(chan peer.AddrInfo, numPeers)
				go func() {
					defer close(ch)
					// Use DHT-discovered peers as relay candidates
					// The libp2p will automatically find suitable relay peers
				}()
				return ch
			},
			autorelay.WithMaxCandidates(4),
		),
		libp2p.EnableNATService(),
		libp2p.EnableHolePunching(),
		libp2p.DefaultListenAddrs,
		libp2p.ConnectionManager(cm),
		libp2p.DefaultTransports,
		libp2p.DefaultMuxers,
		libp2p.DefaultSecurity,
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			d, err := dht.New(ctx, h, dht.Mode(dht.ModeAuto))
			kadDHT = d
			return d, err
		}),
	)
	if err != nil {
		return nil, err
	}

	for _, addr := range bootstrapPeers {
		ma, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			log.Printf("Failed to parse multiaddr %s: %v", addr, err)
			continue
		}

		pi, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			log.Printf("Failed to get peer info from addr %s: %v", addr, err)
			continue
		}

		go func(peerInfo peer.AddrInfo) {
			if err := h.Connect(ctx, peerInfo); err != nil {
				log.Printf("Failed to connect to bootstrap node %s: %v", peerInfo.ID, err)
			} else {
				log.Printf("Successfully connected to bootstrap node %s", peerInfo.ID)
			}
		}(*pi)
	}

	go func() {
		if err := kadDHT.Bootstrap(ctx); err != nil {
			log.Printf("DHT bootstrap error: %v", err)
		} else {
			log.Println("DHT bootstrap complete")
		}
	}()

	rendezvousString := "my-app-rendezvous"
	routingDiscovery := routingdisc.NewRoutingDiscovery(kadDHT)
	go func() {
		// Advertise presence on DHT
		_, err := discovery.Advertise(ctx, routingDiscovery, rendezvousString)
		if err != nil {
			log.Printf("DHT advertise failed: %v", err)
		} else {
			log.Printf("Successfully advertised rendezvous: %s", rendezvousString)
		}
	}()

	return &P2PNode{
		Host:       h,
		Ctx:        ctx,
		PrivateKey: h.Peerstore().PrivKey(h.ID()),
		PublicKey:  h.Peerstore().PubKey(h.ID()),
		DHT:        kadDHT,
	}, nil
}

// PerformConnectivityTest tests connectivity to a specific peer
func (p *P2PNode) PerformConnectivityTest(peerID peer.ID) error {
	ctx, cancel := context.WithTimeout(p.Ctx, 10*time.Second)
	defer cancel()

	// Try to ping the peer
	// Note: You might need to implement a ping protocol or use DHT queries
	return p.Host.Connect(ctx, peer.AddrInfo{ID: peerID})
}

// ReconnectBootstraps attempts to reconnect to bootstrap peers
func (p *P2PNode) ReconnectBootstraps() {
	for _, addr := range bootstrapPeers {
		ma, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			continue
		}

		pi, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			continue
		}

		// Only try to reconnect if not currently connected
		if p.Host.Network().Connectedness(pi.ID) != network.Connected {
			go func(peerInfo peer.AddrInfo) {
				ctx, cancel := context.WithTimeout(p.Ctx, 15*time.Second)
				defer cancel()

				if err := p.Host.Connect(ctx, peerInfo); err != nil {
					log.Printf("Reconnection failed for %s: %v", peerInfo.ID, err)
				} else {
					log.Printf("Successfully reconnected to bootstrap node %s", peerInfo.ID)
				}
			}(*pi)
		}
	}
}

// CheckUniqueConnections analyzes connected peers and returns statistics
func (n *P2PNode) CheckUniqueConnections() (*ConnectionStats, error) {
	stats := &ConnectionStats{
		UniqueIPs:        []string{},
		ConnectedPeers:   []peer.ID{},
		TotalConnections: 0,
		RelayConnections: 0,
	}

	// Get all connected peers
	connectedPeers := n.Host.Network().Peers()
	stats.TotalConnections = len(connectedPeers)
	stats.ConnectedPeers = connectedPeers

	// Track unique IP addresses
	ipSet := make(map[string]bool)
	bootstrapPeerIDs := n.getBootstrapPeerIDs()

	for _, peerID := range connectedPeers {
		// Skip bootstrap peers
		if bootstrapPeerIDs[peerID] {
			continue
		}

		// Get connections to this peer
		conns := n.Host.Network().ConnsToPeer(peerID)
		for _, conn := range conns {
			// Check if this is a relay connection
			if n.isRelayConnection(conn) {
				stats.RelayConnections++
				continue
			}

			// Extract IP address from remote multiaddr
			remoteAddr := conn.RemoteMultiaddr()
			ip := n.extractIPFromMultiaddr(remoteAddr)
			if ip != "" && !ipSet[ip] {
				ipSet[ip] = true
				stats.UniqueIPs = append(stats.UniqueIPs, ip)
			}
		}
	}

	return stats, nil
}

// getBootstrapPeerIDs returns a map of bootstrap peer IDs for quick lookup
func (n *P2PNode) getBootstrapPeerIDs() map[peer.ID]bool {
	bootstrapIDs := make(map[peer.ID]bool)

	for _, addr := range bootstrapPeers {
		ma, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			continue
		}

		pi, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			continue
		}

		bootstrapIDs[pi.ID] = true
	}

	return bootstrapIDs
}

// isRelayConnection checks if a connection is through a relay
func (n *P2PNode) isRelayConnection(conn network.Conn) bool {
	// Check if the connection uses relay transport
	remoteAddr := conn.RemoteMultiaddr()
	localAddr := conn.LocalMultiaddr()

	// Look for relay protocol indicators in the multiaddr
	remoteStr := remoteAddr.String()
	localStr := localAddr.String()

	return strings.Contains(remoteStr, "/p2p-circuit") ||
		strings.Contains(localStr, "/p2p-circuit") ||
		strings.Contains(remoteStr, "/relay") ||
		strings.Contains(localStr, "/relay")
}

// extractIPFromMultiaddr extracts IP address from a multiaddr
func (n *P2PNode) extractIPFromMultiaddr(ma multiaddr.Multiaddr) string {
	// Split the multiaddr into components
	protocols := ma.Protocols()

	for _, proto := range protocols {
		if proto.Code == multiaddr.P_IP4 || proto.Code == multiaddr.P_IP6 {
			// Get the value for this protocol
			value, err := ma.ValueForProtocol(proto.Code)
			if err == nil {
				// Validate that it's a valid IP
				if ip := net.ParseIP(value); ip != nil {
					return value
				}
			}
		}
	}

	return ""
}

// PrintConnectionStats prints detailed connection statistics
func (n *P2PNode) PrintConnectionStats() {
	stats, err := n.CheckUniqueConnections()
	if err != nil {
		log.Printf("Error checking connections: %v", err)
		return
	}

	fmt.Printf("\n=== P2P Connection Statistics ===\n")
	fmt.Printf("Total Connections: %d\n", stats.TotalConnections)
	fmt.Printf("Relay Connections: %d\n", stats.RelayConnections)
	fmt.Printf("Direct Connections: %d\n", stats.TotalConnections-stats.RelayConnections)
	fmt.Printf("Unique IP Addresses: %d\n", len(stats.UniqueIPs))

	if len(stats.UniqueIPs) > 0 {
		fmt.Printf("\nConnected IP Addresses:\n")
		for i, ip := range stats.UniqueIPs {
			fmt.Printf("  %d. %s\n", i+1, ip)
		}
	}

	fmt.Printf("\nConnected Peer IDs:\n")
	for i, peerID := range stats.ConnectedPeers {
		fmt.Printf("  %d. %s\n", i+1, peerID.String())
	}
	fmt.Printf("================================\n\n")
}

// StartPeriodicConnectionCheck starts a goroutine that periodically checks connections
func (n *P2PNode) StartPeriodicConnectionCheck(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				n.PrintConnectionStats()
			case <-n.Ctx.Done():
				return
			}
		}
	}()
}

func (n *P2PNode) SetStreamHandler(protocolID protocol.ID, handler network.StreamHandler) {
	n.Host.SetStreamHandler(protocolID, handler)
}
