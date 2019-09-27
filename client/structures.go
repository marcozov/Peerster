package client

import (
	"bytes"
	"fmt"
	"github.com/dedis/protobuf"
	"github.com/marcozov/Peerster/messages"
	"net"
)

//var gossiper = "127.0.0.1"

type Client struct {
	gossipUDPAddress *net.UDPAddr // gossiper to connect for sending a message
	gossipUDPConnection *net.UDPConn
}

func NewClient(gossiperAddress, UIPort string) *Client {
	var udpAddressBuilder bytes.Buffer

	//udpAddressBuilder.WriteString(GetGossiperAddress())
	udpAddressBuilder.WriteString(gossiperAddress)
	udpAddressBuilder.WriteString(":")
	udpAddressBuilder.WriteString(UIPort)

	serverAddress, err := net.ResolveUDPAddr("udp4", udpAddressBuilder.String())
	if err != nil {
		panic(fmt.Sprintf("Error in parsing the UDP address: %s", err))
	}

	udpConnection, err := net.DialUDP("udp", nil, serverAddress)

	if err != nil {
		panic(fmt.Sprintf("Error in creating udp connection: %s", err))
	}
	
	return &Client {
		gossipUDPAddress: serverAddress,
		gossipUDPConnection: udpConnection,
	}
}

func (client *Client) String() string {
	var c bytes.Buffer

	c.WriteString(fmt.Sprintf("UDP address: %s\n", client.gossipUDPAddress.String()))

	return c.String()
}

func (client *Client) SendMessage(messageWrapper *messages.GossipPacket) {
	packetBytes, err := protobuf.Encode(messageWrapper)

	if err != nil {
		panic(fmt.Sprintf("Error in encoding the message: %s", err))
	}

	_, err = client.gossipUDPConnection.Write(packetBytes)

	if err != nil {
		panic(fmt.Sprintf("Error in sending udp data: %s", err))
	}
}