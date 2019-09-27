package messages

import "fmt"

type SimpleMessage struct {
	OriginalName 	string
	RelayPeerAddr 	string
	Contents 		string
}

type GossipPacket struct {
	Simple 		*SimpleMessage
	Rumor 		*RumorMessage
	Status 		*StatusPacket
	Private 	*PrivateMessage
	DataRequest *DataRequest
	DataReply	*DataReply
	ShareFile	*ShareFile
}

type RumorMessage struct {
	Origin	string
	ID 		uint32
	Text 	string
}

type PrivateMessage struct {
	Origin 		string
	ID			uint32
	Text		string
	Destination string
	HopLimit	uint32
}

type DataRequest struct {
	Origin		string
	Destination	string
	HopLimit	uint32
	HashValue	[]byte // hash of the requested chunk or the MetaHash (if the metafile is requested)
}

type DataReply struct {
	Origin		string
	Destination	string
	HopLimit	uint32
	HashValue	[]byte // hash of the data that is sent with this message
	Data		[]byte // the actual data that is sent (chunk or the metafile)
}

type ShareFile struct {
	Filename	string
}

type PeerStatus struct {
	Identifier	string
	NextID 		uint32
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