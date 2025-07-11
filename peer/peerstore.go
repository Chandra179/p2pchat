package peer

import (
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	"github.com/multiformats/go-multiaddr"
)

// PeerData represents the data we store for each peer
type PeerData struct {
	ID        string   `json:"id"`
	Addresses []string `json:"addresses"`
	Timestamp int64    `json:"timestamp"`
}

// PeerStore combines BadgerDB for persistence with in-memory peerstore
type PeerStore struct {
	memStore peerstore.Peerstore
}

// NewPeerStore creates a new persistent peer store
func NewPeerStore(dbPath string) (*PeerStore, error) {
	memStore, err := pstoremem.NewPeerstore()
	if err != nil {
		return nil, fmt.Errorf("failed to create memory peerstore: %w", err)
	}
	ps := &PeerStore{
		memStore: memStore,
	}
	return ps, nil
}

// AddPeer adds a peer to both memory and persistent storage
func (ps *PeerStore) AddPeer(id peer.ID, addrs []multiaddr.Multiaddr) {
	ps.memStore.AddAddrs(id, addrs, peerstore.TempAddrTTL)
}

// GetPeer retrieves peer addresses from memory store
func (ps *PeerStore) GetPeer(id peer.ID) []multiaddr.Multiaddr {
	return ps.memStore.Addrs(id)
}

// GetAllPeers returns all peers from memory store
func (ps *PeerStore) GetAllPeers() []peer.ID {
	return ps.memStore.Peers()
}

// RemovePeer removes a peer from both memory and persistent storage
func (ps *PeerStore) RemovePeer(id peer.ID) {
	ps.memStore.ClearAddrs(id)
}

// Close closes the persistent store
func (ps *PeerStore) Close() {
	if err := ps.memStore.Close(); err != nil {
		log.Printf("Error closing memory store: %v", err)
	}
}
