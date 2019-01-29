package communications

import (
	"fmt"
	"github.com/dedis/protobuf"
	"github.com/marcozov/Peerster/messages"
	"github.com/marcozov/Peerster/peers"
	"math/rand"
	"net"
	"time"
)

var peerMessage = "SIMPLE MESSAGE origin %s from %s contents %s\n"
var clientMessage = "CLIENT MESSAGE %s\n"
var listPeers = "PEERS %s\n"

func (gossiper *Gossiper) HandleClientConnection() {
	for {
		udpBufferClient := make([]byte, 1024)
		mex := &messages.GossipPacket{}
		n, senderAddress, err := gossiper.ClientListenerConnection.ReadFromUDP(udpBufferClient)

		if err != nil {
			panic(fmt.Sprintf("error in reading UDP data: %s.\nudpBuffer: %v\nsenderAddress: %s\nn bytes: %d", err, udpBufferClient, senderAddress.String(), n))
		}

		udpBufferClient = udpBufferClient[:n]
		err = protobuf.Decode(udpBufferClient, mex)

		if err != nil {
			panic(fmt.Sprintf("error in decoding UDP data: %s\nudpBuffer: %v\nsenderAddress: %s\npacket: %s\nn bytes: %d", err, udpBufferClient, senderAddress.String(), mex, n))
		}

		fmt.Printf(clientMessage, mex.Simple.Contents)
		fmt.Printf("PEERS %s\n", gossiper.PeersAsString())

		mex.Simple.OriginalName = gossiper.Name
		mex.Simple.RelayPeerAddr = gossiper.PeerListenerAddress.String()
		noSend := make(map[string]struct{})

		if !gossiper.Simple {
			gossiper.Counter.Mux.Lock()
			gossiper.Counter.Counter++
			gossiper.Counter.Mux.Unlock()

			mex.Rumor = &messages.RumorMessage{
				Origin: gossiper.Name,
				ID: gossiper.Counter.Counter,
				Text: mex.Simple.Contents,
			}
			mex.Simple = nil

			//gossiper.handleRumorMessage(mex, senderAddress)
			gossiper.ProcessMessage(mex, senderAddress)
		} else {
			gossiper.sendToPeers(mex, gossiper.Peers.GetAllPeers(), noSend)
		}

		fmt.Println("DATABASE: ", gossiper.Database.Messages, "\nSTATE: ", gossiper.Database.CurrentStatus)

		//gossiper.sendToPeers(mex, gossiper.Peers.GetAllPeers(), noSend)
	}
}

func (gossiper *Gossiper) HandleIncomingPeerMessages() {
	for {
		udpBuffer := make([]byte, 1024)
		mex := &messages.GossipPacket{}
		n, senderAddress, err := gossiper.PeerListenerConnection.ReadFromUDP(udpBuffer)

		if err != nil {
			panic(fmt.Sprintf("error in reading UDP data: %s.\nudpBuffer: %v\nsenderAddress: %s\nn bytes: %d", err, udpBuffer, senderAddress.String(), n))
		}

		udpBuffer = udpBuffer[:n]
		err = protobuf.Decode(udpBuffer, mex)

		if err != nil {
			panic(fmt.Sprintf("error in decoding UDP data: %s\nudpBuffer: %v\nsenderAddress: %s\npacket: %s\nn bytes: %d", err, udpBuffer, senderAddress.String(), mex, n))
		}

		gossiper.addPeer(senderAddress.String())
		go gossiper.ProcessMessage(mex, senderAddress)
	}
}

func (gossiper *Gossiper) ProcessMessage(packet *messages.GossipPacket, senderAddress *net.UDPAddr) {
	//gossiper.Peers.Mux.Lock()

	//defer gossiper.Peers.Mux.Unlock()

	if packet.Simple != nil {
		gossiper.handleSimpleMessage(packet)
	} else if packet.Rumor != nil {
		gossiper.handleRumorMessage(packet, senderAddress)
	} else if packet.Status != nil {
		gossiper.handleStatusMessage(packet, senderAddress)
	}

	fmt.Printf(listPeers, gossiper.PeersAsString())
}

func (gossiper *Gossiper) sendToPeers(packet *messages.GossipPacket, peers map[string]*peers.Peer, noSend map[string]struct{}) {
	for addr, peer := range peers {
		if _, exists := noSend[addr]; !exists {
			fmt.Println("addr: ", addr, noSend)
			gossiper.sendToSinglePeer(packet, peer.Address)
		}
	}
}

func (gossiper *Gossiper) sendToSinglePeer(packet *messages.GossipPacket, peer *net.UDPAddr) {
	packetBytes, err := protobuf.Encode(packet)
	if err != nil {
		panic(fmt.Sprintf("Error in encoding the message: %s", err))
	}

	//fmt.Println("Sending [", packet,"] to: ", peer.String())
	if packet.Rumor != nil {
		fmt.Println("Sending RUMOR [", packet.Rumor.Origin, packet.Rumor.ID, packet.Rumor.Text, "] to: ", peer.String())
	}
	if packet.Simple != nil {
		fmt.Println("Sending SIMPLE [", packet.Simple.OriginalName, packet.Simple.RelayPeerAddr, packet.Simple.Contents, "] to: ", peer.String())
	}
	if packet.Status != nil {
		fmt.Println("Sending STATUS [", packet.Status.Want, "] to ", peer.String())
	}

	_, err = gossiper.PeerListenerConnection.WriteToUDP(packetBytes, peer)

	if err != nil {
		panic(fmt.Sprintf("Error in sending udp data: %s", err))
	}
}

func (gossiper *Gossiper) handleSimpleMessage(packet *messages.GossipPacket) {
	//gossiper.addPeer(packet.Simple.RelayPeerAddr)
	allPeers := gossiper.Peers.GetAllPeers()

	fmt.Printf(peerMessage, packet.Simple.OriginalName, packet.Simple.RelayPeerAddr, packet.Simple.Contents)
	fmt.Printf(listPeers, gossiper.PeersAsString())

	noSend := make(map[string]struct{})
	noSend[packet.Simple.RelayPeerAddr] = struct{}{}
	packet.Simple.RelayPeerAddr = gossiper.PeerListenerAddress.String()

	gossiper.sendToPeers(packet, allPeers, noSend)
}

func (gossiper *Gossiper) processRumorMessage(packet *messages.GossipPacket, senderAddress *net.UDPAddr) {
	// senderAddress not really needed

	// send a copy of the rumor to a randomly chosen peer
	allPeers := gossiper.Peers.GetAllPeers()
	// get random neighbor
	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(len(allPeers))
	var chosenPeer *peers.Peer
	for _, chosenPeer = range allPeers {
		if n == 0 {
			break
		}

		n--
	}

	gossiper.sendToSinglePeer(packet, chosenPeer.Address)

	// what if there is a different socket for these status messages?
	// it could be a socket that handles only acknowledgements for rumors
	// --> compatibility issues with other Peerster software

	// run an handler for STATUS that disappears after the timeout
	// the handler is related specifically to the sender of the rumor (senderAddress)

	gossiper.StatusHandlers[chosenPeer.Address.String()] = make(chan *messages.StatusPacket)
	handler, ok := gossiper.StatusHandlers[chosenPeer.Address.String()]
	if !ok {
		panic(fmt.Sprintf("ERROR: handler for %s does not exist", chosenPeer.Address.String()))
	}

	fmt.Println("Handler created for ", chosenPeer.Address.String())

	select {
	case receivedACK := <- handler:
		fmt.Println("ACK received!")
		gossiper.processStatusMessage(receivedACK, chosenPeer.Address)
		delete(gossiper.StatusHandlers, chosenPeer.Address.String())
	case <- time.After(1 * time.Second):
		fmt.Println("*** TIMEOUT ***")
		// destroy the handler
		_, exists := gossiper.StatusHandlers[chosenPeer.Address.String()]
		if exists {
			delete(gossiper.StatusHandlers, chosenPeer.Address.String())
		}

		// flip coin, choose another random peer R', send the RUMOR message to R' (just call this function but without sending back the ACK?)
		// ....
		rand.Seed(time.Now().UnixNano())
		if rand.Int() % 2 == 1 {
			gossiper.processRumorMessage(packet, senderAddress)
		}
	}
}

func (gossiper *Gossiper) handleRumorMessage(packet *messages.GossipPacket, senderAddress *net.UDPAddr) {
	// TODO: check whether the message has already been received
	nextMessage := false
	gossiper.Database.Mux.Lock()

	fmt.Println("received RUMOR packet: ", packet.Rumor)
	var desiredStatus *messages.PeerStatus
	desiredStatus = nil
	for i, _ := range gossiper.Database.CurrentStatus.Want {
		if gossiper.Database.CurrentStatus.Want[i].Identifier == packet.Rumor.Origin {
			desiredStatus = &gossiper.Database.CurrentStatus.Want[i]

			break
		}
	}

	// if no message was received from the host specified by the RUMOR packet
	if desiredStatus == nil {
		gossiper.Database.Messages[packet.Rumor.Origin] = []*messages.RumorMessage{}
		desiredStatus = &messages.PeerStatus{Identifier: packet.Rumor.Origin, NextID: 1}
		gossiper.Database.CurrentStatus.Want = append(gossiper.Database.CurrentStatus.Want, *desiredStatus)

		desiredStatus = &gossiper.Database.CurrentStatus.Want[len(gossiper.Database.CurrentStatus.Want)-1]
	}
	fmt.Println("desired status: ", desiredStatus)

	//if int(packet.Rumor.ID) == len(receivedMessages)+1 {
	// if the next message that must be received is the received one (i.e., it is not out of order)
	if desiredStatus.NextID == packet.Rumor.ID {
		// is it possible to do so with a slice?
		receivedMessages := gossiper.Database.Messages[packet.Rumor.Origin]
		gossiper.Database.Messages[packet.Rumor.Origin] = append(receivedMessages, packet.Rumor)
		fmt.Println("DATABASE: ", gossiper.Database.Messages, "\nSTATE: ", gossiper.Database.CurrentStatus)
		fmt.Println("status (BEFORE): ", gossiper.Database.CurrentStatus.Want)
		desiredStatus.NextID++
		fmt.Println("status (AFTER): ", gossiper.Database.CurrentStatus.Want)
		nextMessage = true
	}

	// should a status message be sent back in any case? no, maybe it is just propagating an old message!
	// but if it is a "too new" message maybe it make sense to send back a status message, so that the sender can send old messages!

	// sends a STATUS to the sender, to ACKnowledge the RUMOR
	statusPacket := &messages.GossipPacket{
		Status: gossiper.Database.CurrentStatus,
	}

	gossiper.sendToSinglePeer(statusPacket, senderAddress)
	gossiper.Database.Mux.Unlock()

	// if the message is out of order, do not propagate
	fmt.Println("next message??", nextMessage)
	if !nextMessage {
		return
	}

	gossiper.processRumorMessage(packet, senderAddress)
}

func (gossiper *Gossiper) handleStatusMessage(packet *messages.GossipPacket, senderAddress *net.UDPAddr) {
	fmt.Println("Received STATUS: ", packet.Status.Want, "from", senderAddress.String())

	// add lock
	// is it possible that a status message comes from "senderAddress" but
	// it is not related to any RUMOR sent? It could just be a status message sent through the anti-entropy mechanism
	// ---> but it is not a problem!
	handler, exists := gossiper.StatusHandlers[senderAddress.String()]
	if exists {
		fmt.Println("HANDLER EXISTS!! for ", senderAddress.String())
		// this will still call the processStatusMessage function, but it will let the processRumorMessage function continue
		handler <- packet.Status
	} else {
		fmt.Println("no handler exists for ", senderAddress.String())
		gossiper.processStatusMessage(packet.Status, senderAddress)
	}
}

func (gossiper *Gossiper) processStatusMessage(status *messages.StatusPacket, senderAddress *net.UDPAddr) {
	gossiper.Database.Mux.Lock()
	defer gossiper.Database.Mux.Unlock()

	currentStatusMap := make(map[string]uint32)
	currentStatus := gossiper.Database.CurrentStatus.Want

	receivedStatusMap := make(map[string]uint32)

	for _, myPeerStatus := range currentStatus {
		currentStatusMap[myPeerStatus.Identifier] = myPeerStatus.NextID
	}

	for _, receivedPeerStatus := range status.Want {
		receivedStatusMap[receivedPeerStatus.Identifier] = receivedPeerStatus.NextID
	}

	// trying to see whether I have to send messages to senderAddress (that sent me the STATUS)
	for identifierMyStatus, nextIDmyStatus := range currentStatusMap {
		nextIDreceivedStatus, exists := receivedStatusMap[identifierMyStatus]

		if !exists {
			// send all the messages of nextID to senderAddress.. well, just send the first one, this will trigger the whole mechanism of RUMOR-STATUS!
		} else {
			if nextIDmyStatus < nextIDreceivedStatus {
				// send my status to senderAddress
			} else if nextIDmyStatus > nextIDreceivedStatus {
				// send the message with nextIDreceivedStatus
			}
		}
	}

	for identifierReceivedStatus, nextIDreceivedStatus :=  range receivedStatusMap {
		nextIDmyStatus, exists := currentStatusMap[identifierReceivedStatus]

		fmt.Println(nextIDreceivedStatus, nextIDmyStatus)

		if !exists {
			// send my status to senderAddress
		} else {

		}
	}
}