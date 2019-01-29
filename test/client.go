package main

import (
	"fmt"
	"net"
	"time"
	//"strconv"
)

func CheckError(err error) {
	if err  != nil {
		fmt.Println("Error: " , err)
	}
}

func main() {
	ServerAddr,err := net.ResolveUDPAddr("udp","127.0.0.1:5000")
	CheckError(err)

	//LocalAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	//CheckError(err)

	Conn, err := net.DialUDP("udp", nil, ServerAddr)
	CheckError(err)

	defer Conn.Close()
	i := 0
	for {
		msg := "asdlolxd"
		i++
		buf := []byte(msg)
		_,err := Conn.Write(buf)
		if err != nil {
			fmt.Println(msg, err)
		}
		time.Sleep(time.Second * 1)
	}
}