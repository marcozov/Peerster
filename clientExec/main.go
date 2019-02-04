package main

import (
	"flag"
	"github.com/marcozov/Peerster/client"
	"github.com/marcozov/Peerster/messages"
)

var gossiper = "127.0.0.1"

func main() {
	port := flag.String("UIPort", "", "port for the UI client")
	msg := flag.String("msg", "", "message to be sent")

	flag.Parse()

	if *msg == "" {
		panic("-msg cannot be omitted")
	}
	if *port == "" {
		panic("-UIPort cannot be omitted")
	}

	//fmt.Println("UIPort:", *port)
	//fmt.Println("msg:", *msg)

	client := client.NewClient(gossiper, *port)

	//fmt.Println(client.String())

	messageWrapper := &messages.GossipPacket{
		Simple: &messages.SimpleMessage {
			OriginalName: "",
			RelayPeerAddr: "",
			Contents: *msg,
		},
	}
	client.SendMessage(messageWrapper)
}
