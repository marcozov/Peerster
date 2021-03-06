package communications

import (
	"bytes"
	"fmt"
	"github.com/marcozov/Peerster/messages"
	"github.com/marcozov/Peerster/peers"
	"github.com/marcozov/Peerster/routing"
	"net"
	"strings"
	"sync"
	"time"
)

// add the peers to the gossiper structure
type Gossiper struct {
	ClientListenerAddress 		*net.UDPAddr // peerAddress on which client sends messages
	ClientListenerConnection 	*net.UDPConn	 // listener (to receive data from client)
	PeerListenerAddress 		*net.UDPAddr // peerAddress on which peers send messages
	PeerListenerConnection 		*net.UDPConn // listener (to receive data from peers)
	Name string			 		// name of the gossiper
	//Peers map[string]Peer
	Peers    					*peers.SafeMapPeers
	FilesDatabase				*FilesDatabase
	Database 					*MessagesDatabase
	PrivateDatabase 			*PrivateMessagesDatabase
	Simple 						bool
	//StatusHandlers map[string]chan *messages.StatusPacket
	StatusHandlers 				*SafeMapStatusHandlers
	Counter 					CounterWrapper
	PrefixTable 				*routing.PrefixTable
}

type SafeMapStatusHandlers struct {
	V   map[string]chan *messages.StatusPacket
	Mux sync.RWMutex
}

type CounterWrapper struct {
	Counter uint32
	Mux sync.RWMutex
}

type FilesDatabase struct {
	Files map[string]*Metadata
	Mux sync.RWMutex
}

// should the database contain both the received messages and the current status?
// their state should be consistent, but there might be cases in which it doesn't make sense to lock both the structures
type MessagesDatabase struct {
	Messages map[string][]*messages.RumorMessage // name -> list of messages
	CurrentStatus *messages.StatusPacket
	Mux sync.RWMutex
}

// database that contains the private messages
// the messages sent and received are both stored here
type PrivateMessagesDatabase struct {
	MessagesSent map[string][]*messages.PrivateMessage
	MessagesReceived map[string][]*messages.PrivateMessage
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

		FilesDatabase: &FilesDatabase{
			Files: make(map[string]*Metadata),
		},
		Database: &MessagesDatabase {
			Messages: make(map[string][]*messages.RumorMessage),
			CurrentStatus: &messages.StatusPacket{
				Want: []messages.PeerStatus{},
			},
		},
		PrivateDatabase: &PrivateMessagesDatabase{
			MessagesSent: make(map[string][]*messages.PrivateMessage),
			MessagesReceived: make(map[string][]*messages.PrivateMessage),
		},
		Counter: CounterWrapper{
			Counter: 0,
		},
		PrefixTable: &routing.PrefixTable{
			V: make(map[string]*routing.TableEntry),
		},
	}

	for _, peer := range strings.Split(peersList, ",") {
		gossiper.AddPeer(peer)
	}

	return gossiper
}

func (gossiper *Gossiper) AddPeer(addr string) {
	peerAddress, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		panic(fmt.Sprintf("Error in parsing the UDP peerAddress: %s", err))
	}

	gossiper.Peers.AddPeer(&peers.Peer{Address: peerAddress})
}

func (gossiper *Gossiper) AddDeletePeer(addr string) {
	gossiper.Peers.Mux.Lock()
	defer gossiper.Peers.Mux.Unlock()

	peerAddress, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		panic(fmt.Sprintf("Error in parsing the UDP peerAddress: %s", err))
	}

	peer := &peers.Peer{Address: peerAddress}

	if _, exists := gossiper.Peers.V[peer.Address.String()]; !exists {
		gossiper.Peers.V[peer.Address.String()] = peer
	} else {
		delete(gossiper.Peers.V, peer.Address.String())
	}
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

// anti-entropy mechanism
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

// anti-entropy mechanism: should the host get the sequence number from the database of rumors or from the prefix table?
// cannot get it from the prefix table!
func (gossiper *Gossiper) PeriodicRouteRumorPropagation(rtimer int) {
	if rtimer <= 0 {
		return
	}
	ticker := time.NewTicker(time.Duration(rtimer) * time.Second)
	noSend := make(map[string]struct{})

	for {
		select {
		case <- ticker.C:
			myMessages := gossiper.Database.Messages[gossiper.Name]
			if len(myMessages) > 0 {
				if gossiper.Name != myMessages[len(myMessages)-1].Origin {
					panic("The two names should be the same")
				}

				packet := &messages.GossipPacket{
					Rumor: &messages.RumorMessage{
						Origin: gossiper.Name,
						ID: myMessages[len(myMessages)-1].ID,
						Text: "",
					},
				}
				gossiper.Peers.Mux.RLock()
				gossiper.sendToPeers(packet, gossiper.Peers.V, noSend)
				gossiper.Peers.Mux.RUnlock()
			}
		}
	}
}