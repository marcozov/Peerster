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

		fmt.Printf("CLIENT MESSAGE %s\n", mex.Simple.Contents)
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

		//fmt.Println("DATABASE: ", gossiper.Database.Messages, "\nSTATE: ", gossiper.Database.CurrentStatus)

		fmt.Printf("PEERS %s\n", gossiper.PeersAsString())
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

		gossiper.AddPeer(senderAddress.String())
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

	fmt.Printf("PEERS %s\n", gossiper.PeersAsString())
}

func (gossiper *Gossiper) sendToPeers(packet *messages.GossipPacket, peers map[string]*peers.Peer, noSend map[string]struct{}) {
	for addr, peer := range peers {
		if _, exists := noSend[addr]; !exists {
			fmt.Println("addr: ", addr, noSend)
			gossiper.SendToSinglePeer(packet, peer.Address)
		}
	}
}

func (gossiper *Gossiper) SendToSinglePeer(packet *messages.GossipPacket, peer *net.UDPAddr) {
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
	//gossiper.AddPeer(packet.Simple.RelayPeerAddr)
	allPeers := gossiper.Peers.GetAllPeers()

	fmt.Printf("SIMPLE MESSAGE origin %s from %s contents %s\n", packet.Simple.OriginalName, packet.Simple.RelayPeerAddr, packet.Simple.Contents)
	fmt.Printf("PEERS %s\n", gossiper.PeersAsString())

	noSend := make(map[string]struct{})
	noSend[packet.Simple.RelayPeerAddr] = struct{}{}
	packet.Simple.RelayPeerAddr = gossiper.PeerListenerAddress.String()

	gossiper.sendToPeers(packet, allPeers, noSend)
}

func (gossiper *Gossiper) processRumorMessage(packet *messages.GossipPacket, senderAddress *net.UDPAddr, continues bool) {
	// send a copy of the rumor to a randomly chosen peer
	allPeers := gossiper.Peers.GetAllPeers()
	filteredPeers := make(map[string]*peers.Peer)

	for key, value := range allPeers {
		//if key != senderAddress.String() {
			filteredPeers[key] = value
		//}
	}

	// get random neighbor
	rand.Seed(time.Now().UnixNano())
	nFilteredPeers := len(filteredPeers)
	if nFilteredPeers < 1 {
		return
	}

	n := rand.Intn(nFilteredPeers)
	var chosenPeer *peers.Peer

	// should avoid to pick senderAddress as peer for sending the RUMOR: it makes no sense to send it back!
	for _, chosenPeer = range filteredPeers {
		if n == 0 {
			break
		}

		n--
	}

	// MONGERING
	fmt.Printf("MONGERING with %s\n", chosenPeer.Address.String())
	if continues {
		fmt.Printf("FLIPPED COIN sending rumor to %s\n", chosenPeer.Address.String())
	}
	gossiper.SendToSinglePeer(packet, chosenPeer.Address)

	// run an handler for STATUS that disappears after the timeout
	// the handler is related specifically to the sender of the rumor (senderAddress)

	gossiper.StatusHandlers.Mux.Lock()

	gossiper.StatusHandlers.V[chosenPeer.Address.String()] = make(chan *messages.StatusPacket)
	handler, ok := gossiper.StatusHandlers.V[chosenPeer.Address.String()]
	if !ok {
		panic(fmt.Sprintf("ERROR: handler for %s does not exist", chosenPeer.Address.String()))
	}

	gossiper.StatusHandlers.Mux.Unlock()

	fmt.Println("Handler created for ", chosenPeer.Address.String())

	select {
	case receivedACK := <- handler:
		fmt.Println("ACK received!")
		processedStatus := gossiper.processStatusMessage(receivedACK, chosenPeer.Address)
		gossiper.StatusHandlers.Mux.Lock()
		delete(gossiper.StatusHandlers.V, chosenPeer.Address.String())
		gossiper.StatusHandlers.Mux.Unlock()

		// if I didn't send any RUMOR/STATUS after processing the received STATUS
		if processedStatus == 0 {
			rand.Seed(time.Now().UnixNano())
			if rand.Int() % 2 == 1 {
				gossiper.processRumorMessage(packet, senderAddress, true)
			}
		}
	case <- time.After(1 * time.Second):
		fmt.Println("*** TIMEOUT ***")
		// destroy the handler
		gossiper.StatusHandlers.Mux.Lock()
		_, exists := gossiper.StatusHandlers.V[chosenPeer.Address.String()]
		if exists {
			delete(gossiper.StatusHandlers.V, chosenPeer.Address.String())
		}
		gossiper.StatusHandlers.Mux.Unlock()

		// flip coin, choose another random peer R', send the RUMOR message to R' (just call this function but without sending back the ACK?)
		// ....
		rand.Seed(time.Now().UnixNano())
		if rand.Int() % 2 == 1 {
			gossiper.processRumorMessage(packet, senderAddress, true)
		}
	}
}

func (gossiper *Gossiper) handleRumorMessage(packet *messages.GossipPacket, senderAddress *net.UDPAddr) {
	fmt.Printf("RUMOR origin %s from %s ID %d contents %s\n", packet.Rumor.Origin, senderAddress.String(), packet.Rumor.ID, packet.Rumor.Text)

	// TODO: check whether the message has already been received
	nextMessage := false
	gossiper.Database.Mux.Lock()

	// see if I already have some messages from Rumor.Origin
	//fmt.Println("received RUMOR packet: ", packet.Rumor)
	var desiredStatus *messages.PeerStatus
	desiredStatus = nil
	for i, _ := range gossiper.Database.CurrentStatus.Want {
		if gossiper.Database.CurrentStatus.Want[i].Identifier == packet.Rumor.Origin {
			desiredStatus = &gossiper.Database.CurrentStatus.Want[i]

			break
		}
	}

	// *** ADDING message to the database
	// if no message was received from the host specified by the RUMOR packet
	if desiredStatus == nil {
		gossiper.Database.Messages[packet.Rumor.Origin] = []*messages.RumorMessage{}
		desiredStatus = &messages.PeerStatus{Identifier: packet.Rumor.Origin, NextID: 1}
		gossiper.Database.CurrentStatus.Want = append(gossiper.Database.CurrentStatus.Want, *desiredStatus)

		desiredStatus = &gossiper.Database.CurrentStatus.Want[len(gossiper.Database.CurrentStatus.Want)-1]
	}
	fmt.Println("desired status: ", desiredStatus)

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

	gossiper.SendToSinglePeer(statusPacket, senderAddress)
	gossiper.Database.Mux.Unlock()

	// if the message is out of order, do not propagate
	fmt.Println("next message??", nextMessage)
	if !nextMessage {
		return
	}

	gossiper.processRumorMessage(packet, senderAddress, false)
}

func (gossiper *Gossiper) handleStatusMessage(packet *messages.GossipPacket, senderAddress *net.UDPAddr) {
	//fmt.Println("Received STATUS: ", packet.Status.Want, "from", senderAddress.String())
	toShow := fmt.Sprintf("STATUS from %s ", senderAddress.String())
	for _, v := range packet.Status.Want {
		toShow += fmt.Sprintf("peer %s nextID %d", v.Identifier, v.NextID)
	}
	fmt.Println(toShow)

	// add lock
	// is it possible that a status message comes from "senderAddress" but
	// it is not related to any RUMOR sent? It could just be a status message sent through the anti-entropy mechanism
	// ---> but it is not a problem!
	gossiper.StatusHandlers.Mux.RLock()
	handler, exists := gossiper.StatusHandlers.V[senderAddress.String()]
	gossiper.StatusHandlers.Mux.RUnlock()
	if exists {
		fmt.Println("HANDLER EXISTS!! for ", senderAddress.String())
		// this will still call the processStatusMessage function, but it will let the processRumorMessage function continue
		handler <- packet.Status
	} else {
		fmt.Println("no handler exists for ", senderAddress.String())
		gossiper.processStatusMessage(packet.Status, senderAddress)
	}
}

// returns 1 if this peer appears to have some newer messages to send to senderAddress
// returns -1 if senderAddress appears to have some newer messages to send me
// returns 0 if neither peer appears to have any new messages the other has not yet seen
func (gossiper *Gossiper) processStatusMessage(status *messages.StatusPacket, senderAddress *net.UDPAddr) int {
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

	toSend := &messages.GossipPacket{}

	// trying to see whether I have to send messages to senderAddress (that sent me the STATUS)
	for identifierMyStatus, nextIDmyStatus := range currentStatusMap {
		nextIDreceivedStatus, exists := receivedStatusMap[identifierMyStatus]

		if !exists {
			// send all the messages of nextID to senderAddress.. well, just send the first one, this will trigger the whole mechanism of RUMOR-STATUS!
			myMessages, ok := gossiper.Database.Messages[identifierMyStatus]
			if !ok {
				panic(fmt.Sprintf("Discrepancies between the current status and the database: \n%s\n%s", gossiper.Database.Messages, gossiper.Database.CurrentStatus))
			}
			toSend.Rumor = myMessages[0]

			//fmt.Println(toSend)
			gossiper.SendToSinglePeer(toSend, senderAddress)

			return 1
		} else {
			if nextIDmyStatus < nextIDreceivedStatus {
				// send my status to senderAddress
				toSend.Status = gossiper.Database.CurrentStatus
				gossiper.SendToSinglePeer(toSend, senderAddress)

				return -1
			} else if nextIDmyStatus > nextIDreceivedStatus {
				// send the message with nextIDreceivedStatus
				myMessages, ok := gossiper.Database.Messages[identifierMyStatus]
				if !ok {
					panic(fmt.Sprintf("Discrepancies between the current status and the database: \n%s\n%s", gossiper.Database.Messages, gossiper.Database.CurrentStatus))
				}

				toSend.Rumor = myMessages[nextIDreceivedStatus-1]

				gossiper.SendToSinglePeer(toSend, senderAddress)
				return 1
			}
		}
	}

	for identifierReceivedStatus, nextIDreceivedStatus := range receivedStatusMap {
		nextIDmyStatus, exists := currentStatusMap[identifierReceivedStatus]

		fmt.Println(nextIDreceivedStatus, nextIDmyStatus)

		if !exists {
			// send my status to senderAddress, I need some messages!
			toSend.Status = gossiper.Database.CurrentStatus
			gossiper.SendToSinglePeer(toSend, senderAddress)

			return -1
		} else {
			if nextIDreceivedStatus > nextIDmyStatus {
				// send my status to senderAddress, I need some messages
				toSend.Status = gossiper.Database.CurrentStatus
				gossiper.SendToSinglePeer(toSend, senderAddress)

				return -1
			} else if nextIDreceivedStatus < nextIDmyStatus {
				// send messages to senderAddress: it needs my messages
				// ---> send only one message! (repeat the rumormorgening process by sending one of those messages)

				myMessages, ok := gossiper.Database.Messages[identifierReceivedStatus]
				if !ok {
					panic(fmt.Sprintf("Discrepancies between the current status and the database: \n%s\n%s", gossiper.Database.Messages, gossiper.Database.CurrentStatus))
				}

				toSend.Rumor = myMessages[nextIDreceivedStatus-1]

				gossiper.SendToSinglePeer(toSend, senderAddress)
				return 1
			}
		}
	}

	fmt.Printf("IN SYNC WITH %s\n", senderAddress.String())
	return 0
}