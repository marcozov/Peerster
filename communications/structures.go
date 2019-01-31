package communications

import (
	"bytes"
	"fmt"
	"github.com/marcozov/Peerster/messages"
	"github.com/marcozov/Peerster/peers"
	"net"
	"strings"
	"sync"
	"time"
)

// add the peers to the gossiper structure
type Gossiper struct {
	ClientListenerAddress *net.UDPAddr // peerAddress on which client sends messages
	ClientListenerConnection *net.UDPConn	 // listener (to receive data from client)
	PeerListenerAddress *net.UDPAddr // peerAddress on which peers send messages
	PeerListenerConnection *net.UDPConn // listener (to receive data from peers)
	Name string			 // name of the gossiper
	//Peers map[string]Peer
	Peers    *peers.SafeMapPeers
	Database *MessagesDatabase
	Simple bool

	//StatusHandlers map[string]chan *messages.StatusPacket
	StatusHandlers *SafeMapStatusHandlers

	Counter CounterWrapper
}

type SafeMapStatusHandlers struct {
	V   map[string]chan *messages.StatusPacket
	Mux sync.RWMutex
}

type CounterWrapper struct {
	Counter uint32
	Mux sync.RWMutex
}

// should the database contain both the received messages and the current status?
// their state should be consistent, but there might be cases in which it doesn't make sense to lock both the structures
type MessagesDatabase struct {
	Messages map[string][]*messages.RumorMessage // name -> list of messages
	CurrentStatus *messages.StatusPacket
	Mux sync.RWMutex
}


func NewGossiperComplete(clientAddress, myAddress, name, peersList string) *Gossiper {
	clientUDPAddr, err := net.ResolveUDPAddr("udp4", clientAddress)
	if err != nil {
		panic (fmt.Sprintf("Address not valid: %s", err))
	}

	clientUDPConn, err := net.ListenUDP("udp4", clientUDPAddr)
	if err != nil {
		panic (fmt.Sprintf("Error in opening UDP listener: %s", err))
	}

	peerUDPAddr, err := net.ResolveUDPAddr("udp4", myAddress)
	if err != nil {
		panic (fmt.Sprintf("Address not valid: %s", err))
	}

	peerUDPConn, err := net.ListenUDP("udp4", peerUDPAddr)
	if err != nil {
		panic (fmt.Sprintf("Error in opening UDP listener: %s", err))
	}

	gossiper := &Gossiper{
		ClientListenerAddress: clientUDPAddr,
		ClientListenerConnection: clientUDPConn,
		PeerListenerAddress: peerUDPAddr,
		PeerListenerConnection: peerUDPConn,
		Name: name,
		Peers: peers.InitPeersMap(),
		//StatusHandlers: make(map[string]chan *messages.StatusPacket),
		StatusHandlers: &SafeMapStatusHandlers{
			V: make(map[string]chan *messages.StatusPacket),
		},

		Database: &MessagesDatabase {
			Messages: make(map[string][]*messages.RumorMessage),
			CurrentStatus: &messages.StatusPacket{
				Want: []messages.PeerStatus{},
			},
		},
		Counter: CounterWrapper{
			Counter: 0,
		},
	}

	for _, peer := range strings.Split(peersList, ",") {
		gossiper.addPeer(peer)
	}

	return gossiper
}

func (gossiper *Gossiper) addPeer(addr string) {
	peerAddress, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		panic(fmt.Sprintf("Error in parsing the UDP peerAddress: %s", err))
	}

	gossiper.Peers.AddPeer(&peers.Peer{Address: peerAddress})
}

func (gossiper *Gossiper) PeersAsString() string {
	allPeers := gossiper.Peers.V
	keys := make([]string, 0, len(allPeers))
	for key := range allPeers {
		keys = append(keys, key)
	}

	return strings.Join(keys, ",")
}

func (myGossiper *Gossiper) String() string {
	var gossiper bytes.Buffer

	gossiper.WriteString(fmt.Sprintf("Name: %s\n", myGossiper.Name))
	gossiper.WriteString(fmt.Sprintf("UDP listener peerAddress: %s\n", myGossiper.ClientListenerAddress.String()))

	return gossiper.String()
}

func (gossiper *Gossiper) PeriodicStatusPropagation() {
	ticker := time.NewTicker(1 * time.Second)

	for {
		select {
		case <- ticker.C:
			gossiper.Database.Mux.RLock()
			gossiper.Peers.Mux.RLock()
			packet := &messages.GossipPacket{
				Status: gossiper.Database.CurrentStatus,
			}

			noSend := make(map[string]struct{})
			gossiper.sendToPeers(packet, gossiper.Peers.V, noSend)

			gossiper.Database.Mux.RUnlock()
			gossiper.Peers.Mux.RUnlock()
		}
	}
}