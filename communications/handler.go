package communications

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/dedis/protobuf"
	"github.com/gookit/color"
	"github.com/marcozov/Peerster/messages"
	"github.com/marcozov/Peerster/peers"
	"github.com/marcozov/Peerster/routing"
	"math/rand"
	"net"
	"time"
)

func now() string {
	return color.Red.Sprintf("%s", time.Now())
}

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

		if mex.ShareFile != nil {
			// share file
			metadata := gossiper.GetMetadata(mex.ShareFile.Filename)
			gossiper.FilesDatabase.Mux.Lock()
			gossiper.FilesDatabase.Files[hex.EncodeToString(metadata.MetaHash[:])] = metadata
			fmt.Println(fmt.Sprintf("metadatabase: %s", gossiper.FilesDatabase.Files))
			gossiper.FilesDatabase.Mux.Unlock()
			continue
		}
		if mex.DataRequest != nil {
			mex.DataRequest.Origin = gossiper.Name
			gossiper.ProcessMessage(mex, senderAddress)
			continue
		}
		//if mex.Simple != nil {
		//fmt.Printf("CLIENT MESSAGE %s\n", mex.Simple.Contents)
		//}
		//if
		//fmt.Printf("PEERS %s\n", gossiper.PeersAsString())
		if mex.Simple != nil {
			mex.Simple.OriginalName = gossiper.Name
			mex.Simple.RelayPeerAddr = gossiper.PeerListenerAddress.String()
		}
		noSend := make(map[string]struct{})


		if !gossiper.Simple {
			gossiper.Counter.Mux.Lock()
			if mex.Private == nil {
				gossiper.Counter.Counter++
			}

			if mex.Simple != nil {
				mex.Rumor = &messages.RumorMessage{
					Origin: gossiper.Name,
					ID:     gossiper.Counter.Counter,
					Text:   mex.Simple.Contents,
				}
			}
			gossiper.Counter.Mux.Unlock()

			if mex.Rumor != nil {
				fmt.Printf("CLIENT MESSAGE %s\n", mex.Rumor.Text)
			}
			if mex.Private != nil {
				mex.Rumor = nil
				mex.Private.Origin = gossiper.Name
				fmt.Printf("CLIENT MESSAGE %s\n", mex.Private.Text)
			}

			mex.Simple = nil
			mex.Status = nil
			//gossiper.handleRumorMessage(mex, senderAddress)
			gossiper.ProcessMessage(mex, senderAddress)
		} else {
			fmt.Printf("CLIENT MESSAGE %s\n", mex.Simple.Contents)
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

		// always adding the sender address to the known peers, whatever the type of the message is
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
	} else if packet.Private != nil {
		gossiper.handlePrivateMessage(packet, senderAddress)
	} else if packet.DataRequest != nil {
		gossiper.handleDataRequestMessage(packet, senderAddress)
	} else if packet.DataReply != nil {
		gossiper.handleDataReplyMessage(packet)
	}

	fmt.Printf("%s: PEERS %s\n", now(), gossiper.PeersAsString())
}

// noSend: to exclude some hosts when sending a message
func (gossiper *Gossiper) sendToPeers(packet *messages.GossipPacket, peers map[string]*peers.Peer, noSend map[string]struct{}) {
	for addr, peer := range peers {
		if _, exists := noSend[addr]; !exists {
			gossiper.SendToSinglePeer(packet, peer.Address)
		}
	}
}

func (gossiper *Gossiper) SendToSinglePeer(packet *messages.GossipPacket, peer *net.UDPAddr) {
	packetBytes, err := protobuf.Encode(packet)
	if err != nil {
		panic(fmt.Sprintf("Error in encoding the message: %s", err))
	}

	/*
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
	*/

	_, err = gossiper.PeerListenerConnection.WriteToUDP(packetBytes, peer)

	if err != nil {
		panic(fmt.Sprintf("Error in sending udp data: %s", err))
	}
}

func (gossiper *Gossiper) handleSimpleMessage(packet *messages.GossipPacket) {
	//gossiper.AddPeer(packet.Simple.RelayPeerAddr)
	allPeers := gossiper.Peers.GetAllPeers()

	fmt.Printf("%s SIMPLE MESSAGE origin %s from %s contents %s\n", now(), packet.Simple.OriginalName, packet.Simple.RelayPeerAddr, packet.Simple.Contents)
	//fmt.Printf("PEERS %s\n", gossiper.PeersAsString())

	noSend := make(map[string]struct{})
	noSend[packet.Simple.RelayPeerAddr] = struct{}{}
	packet.Simple.RelayPeerAddr = gossiper.PeerListenerAddress.String()

	gossiper.sendToPeers(packet, allPeers, noSend)
}

func (gossiper *Gossiper) getRandomNeighbor() *peers.Peer {
	return nil
}

func (gossiper *Gossiper) processRumorMessage(packet *messages.GossipPacket, senderAddress *net.UDPAddr, continues bool, noSend map[string]struct{}) {
	// send a copy of the rumor to a randomly chosen peer
	allPeers := gossiper.Peers.GetAllPeers()
	filteredPeers := make(map[string]*peers.Peer)

	for addr, peer := range allPeers {
		if _, exists := noSend[addr]; !exists {
			filteredPeers[addr] = peer
		}
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
	fmt.Printf("%s: MONGERING with %s\n", now(), chosenPeer.Address.String())
	if continues {
		fmt.Printf("%s: FLIPPED COIN sending rumor to %s\n", now(), chosenPeer.Address.String())
	}
	// sending a rumor to a random peer
	gossiper.SendToSinglePeer(packet, chosenPeer.Address)

	// run an handler for STATUS that disappears after the timeout
	// the handler is related specifically to the sender of the rumor (senderAddress)

	gossiper.StatusHandlers.Mux.Lock()

	// creating a status handled for this address
	gossiper.StatusHandlers.V[chosenPeer.Address.String()] = make(chan *messages.StatusPacket)
	handler, ok := gossiper.StatusHandlers.V[chosenPeer.Address.String()]
	if !ok {
		panic(fmt.Sprintf("ERROR: handler for %s does not exist", chosenPeer.Address.String()))
	}

	gossiper.StatusHandlers.Mux.Unlock()

	fmt.Println("Handler created for ", chosenPeer.Address.String())

	select {
	// is it sure that this message is really from "chosenPeer"? Not really.. it could be any status message coming from any host
	case receivedACK := <- handler:
		fmt.Println("ACK received!")
		processedStatus := gossiper.processStatusMessage(receivedACK, chosenPeer.Address)
		gossiper.StatusHandlers.Mux.Lock()
		delete(gossiper.StatusHandlers.V, chosenPeer.Address.String())
		gossiper.StatusHandlers.Mux.Unlock()

		// if I didn't send any RUMOR/STATUS after processing the received STATUS
		// but in this case the random peer should be another one, not the senderAddress
		if processedStatus == 0 {
			noSend[chosenPeer.Address.String()] = struct{}{}
			rand.Seed(time.Now().UnixNano())
			if rand.Int() % 2 == 1 {
				gossiper.processRumorMessage(packet, senderAddress, true, noSend)
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
			gossiper.processRumorMessage(packet, senderAddress, true, noSend)
		}
	}
}

func (gossiper *Gossiper) handleRumorMessage(packet *messages.GossipPacket, senderAddress *net.UDPAddr) {
	senderPeer := gossiper.Peers.GetPeer(senderAddress.String())
	if packet.Rumor.Origin == gossiper.Name && senderPeer != nil {
		fmt.Println("discarding rumor..")
		fmt.Println(packet.Rumor)
		fmt.Println(senderAddress.String())

		//fmt.Println(gossiper.ClientListenerAddress.String())
		fmt.Println(senderPeer)
		return
	}
	if packet.Rumor.Origin != gossiper.Name {
		gossiper.processRoutingUpdate(packet.Rumor, senderAddress)
	}
	if packet.Rumor.Text != "" {
		fmt.Printf("%s: RUMOR origin %s from %s ID %d contents %s\n", now(), packet.Rumor.Origin, senderAddress.String(), packet.Rumor.ID, packet.Rumor.Text)
	}

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
		//desiredStatus = &messages.PeerStatus{Identifier: packet.Rumor.Origin, NextID: 1}
		//gossiper.Database.CurrentStatus.Want = append(gossiper.Database.CurrentStatus.Want, *desiredStatus)
		gossiper.Database.CurrentStatus.Want = append(gossiper.Database.CurrentStatus.Want, messages.PeerStatus{Identifier: packet.Rumor.Origin, NextID: 1})

		// important! otherwise the value inside Want is not really updated
		desiredStatus = &gossiper.Database.CurrentStatus.Want[len(gossiper.Database.CurrentStatus.Want)-1]
	}
	//fmt.Println("desired status: ", desiredStatus)

	// if the next message that must be received is the received one (i.e., it is not out of order)
	if desiredStatus.NextID == packet.Rumor.ID {
		// is it possible to do so with a slice?
		receivedMessages := gossiper.Database.Messages[packet.Rumor.Origin]
		gossiper.Database.Messages[packet.Rumor.Origin] = append(receivedMessages, packet.Rumor)
		//fmt.Println("DATABASE: ", gossiper.Database.Messages, "\nSTATE: ", gossiper.Database.CurrentStatus)
		//fmt.Println("status (BEFORE): ", gossiper.Database.CurrentStatus.Want)
		desiredStatus.NextID++
		//fmt.Println("status (AFTER): ", gossiper.Database.CurrentStatus.Want)
		nextMessage = true
	}

	// should a status message be sent back in any case? no, maybe it is just propagating an old message!
	// but if it is a "too new" message maybe it make sense to send back a status message, so that the sender can send old messages!

	// sends a STATUS to the sender, to ACKnowledge the RUMOR
	statusPacket := &messages.GossipPacket{
		Status: gossiper.Database.CurrentStatus,
	}

	// sending a Status message to the peer that sent the rumor.
	allPeers := gossiper.Peers.GetAllPeers()
	if _, exists := allPeers[senderAddress.String()]; exists {
		gossiper.SendToSinglePeer(statusPacket, senderAddress)
	}

	gossiper.Database.Mux.Unlock()

	// if the message is out of order, do not propagate
	//fmt.Println("next message??", nextMessage)
	if !nextMessage {
		return
	}

	noSend := make(map[string]struct{})
	noSend[senderAddress.String()] = struct{}{}
	gossiper.processRumorMessage(packet, senderAddress, false, noSend)
	//gossiper.processRumorMessage(packet, senderAddress, true, noSend)
}

func (gossiper *Gossiper) handleStatusMessage(packet *messages.GossipPacket, senderAddress *net.UDPAddr) {
	//fmt.Println("Received STATUS: ", packet.Status.Want, "from", senderAddress.String())
	toShow := fmt.Sprintf("%s: STATUS from %s", now(), senderAddress.String())
	for _, v := range packet.Status.Want {
		toShow += fmt.Sprintf(" peer %s nextID %d", v.Identifier, v.NextID)
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
		// it means that this host sent a rumor message
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
			if len(myMessages) > 0 {
				toSend.Rumor = myMessages[0]

				//fmt.Println(toSend)
				gossiper.SendToSinglePeer(toSend, senderAddress)
			}

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

		//fmt.Println(nextIDreceivedStatus, nextIDmyStatus)

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

	fmt.Printf("%s: IN SYNC WITH %s\n", now(), senderAddress.String())
	return 0
}


// updates the routing table if the sender is sending a message with sequence number higher than the current one, for that specific destination
func (gossiper *Gossiper) processRoutingUpdate(rumor *messages.RumorMessage, senderAddress *net.UDPAddr) {
	gossiper.PrefixTable.Mux.Lock()
	defer gossiper.PrefixTable.Mux.Unlock()
	gossiper.Peers.Mux.RLock()
	defer gossiper.Peers.Mux.RUnlock()

	origin := rumor.Origin
	id := rumor.ID

	// getting the entry from the current prefix table
	entry, exists := gossiper.PrefixTable.V[origin]
	if !exists {
		entry = &routing.TableEntry{
			SequenceNumber: 0,
			NextHop: "",
		}
		gossiper.PrefixTable.V[origin] = entry
	}

	_, isOriginPeer := gossiper.Peers.V[origin]

	if entry.SequenceNumber < id {
		if !isOriginPeer || senderAddress.String() == origin {
			old := entry.SequenceNumber
			entry.NextHop = senderAddress.String()
			entry.SequenceNumber = id
			fmt.Println(fmt.Sprintf("DSDV %s %s (%d %d)", origin, entry.NextHop, old, id))
		}
	}
}

func (gossiper *Gossiper) handlePrivateMessage(packet *messages.GossipPacket, senderAddress *net.UDPAddr) {
	if packet.Private.Destination == gossiper.Name {
		fmt.Println(fmt.Sprintf("PRIVATE origin %s hop-limit %d contents %s", packet.Private.Origin, packet.Private.HopLimit, packet.Private.Text))
		gossiper.PrivateDatabase.Mux.Lock()
		messagesReceived, exists := gossiper.PrivateDatabase.MessagesReceived[packet.Private.Origin]
		if !exists {
			messagesReceived = []*messages.PrivateMessage{}
		}
		gossiper.PrivateDatabase.MessagesReceived[packet.Private.Origin] = append(messagesReceived, packet.Private)

		gossiper.PrivateDatabase.Mux.Unlock()
	} else if packet.Private.HopLimit > 0 {
		// send the message to the destination
		packet.Private.HopLimit--
		//gossiper.sendPrivateMessage(packet)
		gossiper.sendDirectMessage(packet)
	}
}

/*
func (gossiper *Gossiper) sendPrivateMessage(packet *messages.GossipPacket) {
	gossiper.PrefixTable.Mux.RLock()
	defer gossiper.PrefixTable.Mux.RUnlock()

	entry, exists := gossiper.PrefixTable.V[packet.Private.Destination]
	if exists {
		nextHopUDPAddr, err := net.ResolveUDPAddr("udp4", entry.NextHop)
		if err != nil {
			panic(fmt.Sprintf("Address not valid: %s", err))
		}
		gossiper.SendToSinglePeer(packet, nextHopUDPAddr)
	}
}
*/

func (gossiper *Gossiper) sendDirectMessage(packet *messages.GossipPacket) {
	gossiper.PrefixTable.Mux.RLock()
	defer gossiper.PrefixTable.Mux.RUnlock()

	var destination string
	if packet.Private != nil {
		destination = packet.Private.Destination
	} else if packet.DataRequest != nil {
		destination = packet.DataRequest.Destination
	} else if packet.DataReply != nil {
		destination = packet.DataReply.Destination
	}

	entry, exists := gossiper.PrefixTable.V[destination]
	if exists {
		nextHopUDPAddr, err := net.ResolveUDPAddr("udp4", entry.NextHop)
		if err != nil {
			panic(fmt.Sprintf("Address not valid: %s", err))
		}
		gossiper.SendToSinglePeer(packet, nextHopUDPAddr)
	}
}

func (gossiper *Gossiper) handleDataRequestMessage(packet *messages.GossipPacket, senderAddress *net.UDPAddr) {
	// the identifier of the file
	fileID := packet.DataRequest.HashValue
	destination := packet.DataRequest.Destination

	// the request is for this host
	if destination == gossiper.Name {
		gossiper.FilesDatabase.Mux.RLock()
		file, exists := gossiper.FilesDatabase.Files[hex.EncodeToString(fileID[:])]
		if exists {
			// the file is shared by this peer
			if bytes.Compare(file.MetaHash[:], fileID) != 0 {
				panic(fmt.Sprintf("The metahash (fileID stored previously when saving) is different from the fileID (data request)"))
			}
			reply := &messages.GossipPacket{
				DataReply: &messages.DataReply{
					Origin: packet.DataRequest.Destination,
					Destination: packet.DataRequest.Origin,
					HopLimit: 20,
					HashValue: fileID,
					Data: file.MetaFile,
				},
			}
			fmt.Println(fmt.Sprintf("Sending the following reply: %s", reply))
			gossiper.sendDirectMessage(reply)
		} else {
			// the file is not shared by this peer. A message specifying this could be sent back
			reply := &messages.GossipPacket{
				DataReply: &messages.DataReply{
					Origin: packet.DataRequest.Destination,
					Destination: packet.DataRequest.Origin,
					HopLimit: 20,
					HashValue: fileID,
					Data: nil,
				},
			}
			fmt.Println(fmt.Sprintf("File does not exist, sending back an error: %s", reply))
			gossiper.sendDirectMessage(reply)
		}

		gossiper.FilesDatabase.Mux.RUnlock()
	} else {

	}
}

func (gossiper *Gossiper) handleDataReplyMessage(packet *messages.GossipPacket) {

}