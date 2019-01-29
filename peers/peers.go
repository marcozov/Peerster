package peers

import (
	"net"
	"sync"
)

type Peer struct {
	Address *net.UDPAddr
}

type SafeMapPeers struct {
	V   map[string]*Peer
	Mux sync.RWMutex
}

func InitPeersMap() *SafeMapPeers {
	return &SafeMapPeers{V: make(map[string]*Peer)}
}

func (peers *SafeMapPeers) AddPeer(peer *Peer) {
	peers.Mux.Lock()
	defer peers.Mux.Unlock()

	if _, exists := peers.V[peer.Address.String()]; !exists {
		peers.V[peer.Address.String()] = peer
	}
}

func (peers *SafeMapPeers) GetPeer(address string) *Peer {
	peers.Mux.RLock()
	defer peers.Mux.RUnlock()

	if _, exists := peers.V[address]; !exists {
		return peers.V[address]
	}

	return nil
}

func (peers *SafeMapPeers) GetAllPeers() map[string]*Peer {
	peers.Mux.RLock()
	defer peers.Mux.RUnlock()

	return peers.V
}