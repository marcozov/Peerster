package messages

import "fmt"

type SimpleMessage struct {
	OriginalName string
	RelayPeerAddr string
	Contents string
}

type GossipPacket struct {
	Simple *SimpleMessage
	Rumor *RumorMessage
	Status *StatusPacket
}

type RumorMessage struct {
	Origin string
	ID uint32
	Text string
}

type PeerStatus struct {
	Identifier string
	NextID uint32
}

type StatusPacket struct {
	Want []PeerStatus
}

func (packet *GossipPacket) String() string {
	return packet.Simple.String()
}

func (message *SimpleMessage) String() string {
	return fmt.Sprintf("OriginalName: %s\nRelayPeerAddr: %s\nContents: %s", message.OriginalName, message.RelayPeerAddr, message.Contents)
}