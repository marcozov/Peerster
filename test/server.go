package main

import (
	"fmt"
	"net"
	"os"
)

/* A Simple function to verify error */
func checkError(err error) {
	if err  != nil {
		fmt.Println("Error: " , err)
		os.Exit(0)
	}
}

func main() {
	/* Lets prepare a address at any address at port 10001*/
	ServerAddr,err := net.ResolveUDPAddr("udp",":5000")
	checkError(err)

	/* Now listen at selected port */
	ServerConn, err := net.ListenUDP("udp", ServerAddr)
	checkError(err)
	defer ServerConn.Close()

	buf := make([]byte, 1024)

	for {
		n,addr,err := ServerConn.ReadFromUDP(buf)
		fmt.Println("Received ",string(buf[0:n]), " from ",addr)

		if err != nil {
			fmt.Println("Error: ",err)
		}
	}
}