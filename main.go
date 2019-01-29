package main

/*
	Is it necessary to put the Relay Peer's address in the message? Where
	else could the receiving peer obtain the relay peer's address from?
		- from the IP header?
		- but if a message is received from another peer and it has to be sent
			to other peers, how can the original address be preserved?
 */

import (
	"flag"
	"fmt"
	"github.com/marcozov/Peerster/communications"
	//"time"
)

var gossiper = "127.0.0.1"

func main() {
	port := flag.String("UIPort", "8080", "port for the UI client")
	gossipAddr := flag.String("gossipAddr", "127.0.0.1:5000", "ip:port for the gossiper")
	name := flag.String("name", "", "name of the gossiper")
	peers := flag.String("peers", "", "comma separated list of peers of the form ip:port")
	simple := flag.Bool("simple", false, "run gossiperClient in simple broadcast mode")

	flag.Parse()

	fmt.Println("UIPort:", *port)
	fmt.Println("gossipAddr:", *gossipAddr)
	fmt.Println("name:", *name)
	fmt.Println("peers:", *peers)
	fmt.Println("simple:", *simple)
	fmt.Println("tail:", flag.Args())

	if *name == "" {
		panic("-name cannot be an empty string")
	}

	clientListenerAddress := fmt.Sprintf("%s:%s", gossiper, *port)
	gossiper := communications.NewGossiperComplete(clientListenerAddress, *gossipAddr, *name, *peers)
	gossiper.Simple = *simple

	go gossiper.HandleClientConnection()
	gossiper.HandleIncomingPeerMessages()
}
