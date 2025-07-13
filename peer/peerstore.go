package peer

import (
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	"github.com/multiformats/go-multiaddr"
)

type PeerData struct {
	ID        string   `json:"id"`
	Addresses []string `json:"addresses"`
	Timestamp int64    `json:"timestamp"`
}

type PeerStore struct {
	memStore peerstore.Peerstore
}

func NewPeerStore() (*PeerStore, error) {
	memStore, err := pstoremem.NewPeerstore()
	if err != nil {
		return nil, fmt.Errorf("failed to create peerstore: %w", err)
	}
	ps := &PeerStore{
		memStore: memStore,
	}
	return ps, nil
}

func (ps *PeerStore) AddPeer(id peer.ID, addrs []multiaddr.Multiaddr) {
	ps.memStore.AddAddrs(id, addrs, peerstore.TempAddrTTL)
}

func (ps *PeerStore) GetPeer(id peer.ID) []multiaddr.Multiaddr {
	return ps.memStore.Addrs(id)
}

func (ps *PeerStore) GetAllPeers() []peer.ID {
	return ps.memStore.Peers()
}

func (ps *PeerStore) RemovePeer(id peer.ID) {
	ps.memStore.ClearAddrs(id)
}

func (ps *PeerStore) Close() {
	if err := ps.memStore.Close(); err != nil {
		fmt.Printf("Failed to close peerstore: %v\n", err)
	}
}
