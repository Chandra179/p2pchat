package peer

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/dgraph-io/badger/v4"
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
	db       *badger.DB
	memStore peerstore.Peerstore
}

// NewPeerStore creates a new persistent peer store
func NewPeerStore(dbPath string) (*PeerStore, error) {
	opts := badger.DefaultOptions(dbPath)
	opts.Logger = nil // Disable badger logs
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger db: %w", err)
	}

	memStore, err := pstoremem.NewPeerstore()
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create memory peerstore: %w", err)
	}

	ps := &PeerStore{
		db:       db,
		memStore: memStore,
	}

	// Load persistent peers into memory
	if err := ps.loadPersistentPeers(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to load persistent peers: %w", err)
	}

	return ps, nil
}

// AddPeer adds a peer to both memory and persistent storage
func (ps *PeerStore) AddPeer(id peer.ID, addrs []multiaddr.Multiaddr) error {
	ps.memStore.AddAddrs(id, addrs, peerstore.PermanentAddrTTL)

	// Convert to storable format
	addrStrs := make([]string, len(addrs))
	for i, addr := range addrs {
		addrStrs[i] = addr.String()
	}

	peerData := PeerData{
		ID:        id.String(),
		Addresses: addrStrs,
		Timestamp: time.Now().Unix(),
	}

	// Save to persistent storage
	return ps.savePeer(peerData)
}

// AddTempPeer adds a peer only to memory (not persistent)
func (ps *PeerStore) AddTempPeer(id peer.ID, addrs []multiaddr.Multiaddr) {
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
func (ps *PeerStore) RemovePeer(id peer.ID) error {
	ps.memStore.ClearAddrs(id)
	return ps.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte("peer:" + id.String()))
	})
}

// Close closes the persistent store
func (ps *PeerStore) Close() error {
	if err := ps.memStore.Close(); err != nil {
		log.Printf("Error closing memory store: %v", err)
	}
	return ps.db.Close()
}

// GetPeerstore returns the underlying peerstore for libp2p usage
func (ps *PeerStore) GetPeerstore() peerstore.Peerstore {
	return ps.memStore
}

// savePeer saves peer data to BadgerDB
func (ps *PeerStore) savePeer(peerData PeerData) error {
	return ps.db.Update(func(txn *badger.Txn) error {
		data, err := json.Marshal(peerData)
		if err != nil {
			return err
		}
		return txn.Set([]byte("peer:"+peerData.ID), data)
	})
}

// loadPersistentPeers loads all peers from BadgerDB into memory
func (ps *PeerStore) loadPersistentPeers() error {
	return ps.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefix := []byte("peer:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var peerData PeerData
				if err := json.Unmarshal(val, &peerData); err != nil {
					return err
				}

				// Convert back to peer.ID and multiaddrs
				id, err := peer.Decode(peerData.ID)
				if err != nil {
					return err
				}

				addrs := make([]multiaddr.Multiaddr, len(peerData.Addresses))
				for i, addrStr := range peerData.Addresses {
					addr, err := multiaddr.NewMultiaddr(addrStr)
					if err != nil {
						return err
					}
					addrs[i] = addr
				}

				// Add to memory store
				ps.memStore.AddAddrs(id, addrs, peerstore.PermanentAddrTTL)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}
