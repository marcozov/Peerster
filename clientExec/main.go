package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/marcozov/Peerster/client"
	"github.com/marcozov/Peerster/messages"
)

var gossiper = "127.0.0.1"

func main() {
	port := flag.String("UIPort", "", "port for the UI client")
	dest := flag.String("dest", "", "destination for the private message")
	file := flag.String("file", "", "file to be indexed by the gossiper (from the shared folder), or filename of the requested file (name used for saving the requested file)")
	msg := flag.String("msg", "", "message to be sent")

	// if -request is specified, then a file is downloaded. Otherwise, the file is shared
	request := flag.String("request", "", "request a file with the specified metahash (hash of the metafile)")

	flag.Parse()

	if *msg == "" && *file == "" {
		panic("-msg and -file cannot be both omitted")
	}
	//if *dest == "" {
	//	panic("-dest cannot be omitted")
	//}
	if *port == "" {
		panic("-UIPort cannot be omitted")
	}
	client := client.NewClient(gossiper, *port)
	var messageWrapper *messages.GossipPacket

	if *file != "" {
		// share or download file
		if *request == "" {
			// share file
			// no peer should be specified..
			messageWrapper = &messages.GossipPacket{
				ShareFile: &messages.ShareFile{
					Filename: *file,
				},
			}
		} else {
			// download file
			decoded, err := hex.DecodeString(*request)
			if err != nil {
				panic(fmt.Sprintf("Error in decoding the requested hex string in bytes: %s\n", err))
			}

			messageWrapper = &messages.GossipPacket{
				DataRequest: &messages.DataRequest{
					//Origin: "",
					Destination: *dest,
					HopLimit: 20,
					//HashValue: []byte{},
					HashValue: decoded,
				},
			}
		}
		//return
	} else if *dest == "" {
		messageWrapper = &messages.GossipPacket{
			Simple: &messages.SimpleMessage {
				OriginalName: "",
				RelayPeerAddr: "",
				Contents: *msg,
			},
		}
	} else {
		messageWrapper = &messages.GossipPacket{
			Private: &messages.PrivateMessage{
				Origin: "",
				ID: 0,
				Text: *msg,
				Destination: *dest,
				HopLimit: 15,
			},
		}
	}
	client.SendMessage(messageWrapper)
}
