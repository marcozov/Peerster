package web

import (
	"encoding/json"
	"fmt"
	"github.com/marcozov/Peerster/client"
	"github.com/marcozov/Peerster/communications"
	"github.com/marcozov/Peerster/messages"
	"io/ioutil"
	"net/http"
	"strconv"
)

type Webserver struct {
	port     string
	router   *http.ServeMux
	gossiper *communications.Gossiper
}

type MessageLogEntry struct {
	FromNode 	string
	SeqID 		uint32
	Content 	string
}

func New(port string, g *communications.Gossiper) *Webserver {
	return &Webserver{port: port, router: http.NewServeMux(), gossiper: g}
}

func (w *Webserver) Start() {
	w.router.HandleFunc("/message", w.MessageHandler)
	w.router.HandleFunc("/node", w.NodeHandler)
	w.router.HandleFunc("/id", w.IdHandler)
	w.router.HandleFunc("/routes", w.handleRoutes)
	w.router.HandleFunc("/privateMessage", w.PrivateMessageHandler)
	w.router.HandleFunc("/file", w.FileHandler)
	w.router.HandleFunc("/privateFile", w.PrivateFileHandler)

	w.router.Handle("/", http.FileServer(http.Dir("client")))
	http.ListenAndServe("localhost:"+w.port, w.router)

}

func safeDecode(w http.ResponseWriter, r *http.Request, out interface{}) error {
	data, err := ioutil.ReadAll(r.Body)

	if err != nil {w.WriteHeader(http.StatusBadRequest)
		return err
	}

	err = json.Unmarshal(data, out)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return err
	}

	return nil
}

func (w *Webserver) ConvertMessageFormat(m *messages.RumorMessage) *MessageLogEntry {
	return &MessageLogEntry{
		FromNode: m.Origin,
		SeqID: m.ID,
		Content: m.Text,
	}
}

func (w *Webserver) ConvertPrivateMessageFormat(m *messages.PrivateMessage) *MessageLogEntry {
	return &MessageLogEntry{
		FromNode: m.Origin,
		SeqID: m.ID,
		Content: m.Text,
		//HopLimit: m.HopLimit,
	}
}

func (w *Webserver) FileHandler(wr http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
	case "POST":
		var data string
		err := safeDecode(wr, r, &data)
		if err != nil {
			wr.WriteHeader(http.StatusInternalServerError)
			return
		}

		//client := client.NewClient("127.0.0.1", strconv.Itoa(w.gossiper.ClientListenerAddress.Port))
		//
		//messageWrapper := &messages.GossipPacket{
		//	DataRequest: &messages.DataRequest {
		//		Origin: "",
		//		Destination: "",
		//		HopLimit: 15,
		//		HashValue: []byte{},
		//	},
		//}
		fmt.Println("file handler data: ", data)
		//client.SendMessage(messageWrapper)

		wr.WriteHeader(http.StatusOK)
	default:
		wr.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (w *Webserver) PrivateFileHandler(wr http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
	case "POST":
		type PrivateMessage struct {
			Destination string
			Content		string
		}

		var data PrivateMessage
		err := safeDecode(wr, r, &data)
		if err != nil {
			wr.WriteHeader(http.StatusInternalServerError)
			return
		}

		destination := data.Destination
		content := data.Content
		fmt.Println("private file handler data: ", destination, content)
	default:
		wr.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (w *Webserver) MessageHandler(wr http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		wr.WriteHeader(http.StatusOK)
		var log []*MessageLogEntry
		w.gossiper.Database.Mux.RLock()
		messagesDB := w.gossiper.Database.Messages

		for _, messagesPerPeer := range messagesDB {
			for _, m := range messagesPerPeer {
				fmt.Println("m: ", m)
				log = append(log, w.ConvertMessageFormat(m))
			}
		}
		w.gossiper.Database.Mux.RUnlock()

		//fmt.Println("log: ", log)
		data, err := json.Marshal(log)

		if err != nil {
			wr.WriteHeader(http.StatusInternalServerError)
			return
		}
		wr.Write(data)

	case "POST":
		var data string
		err := safeDecode(wr, r, &data)
		if err != nil {
			wr.WriteHeader(http.StatusInternalServerError)
			return
		}

		client := client.NewClient("127.0.0.1", strconv.Itoa(w.gossiper.ClientListenerAddress.Port))

		messageWrapper := &messages.GossipPacket{
			Simple: &messages.SimpleMessage {
				OriginalName: "",
				RelayPeerAddr: "",
				Contents: data,
			},
		}
		client.SendMessage(messageWrapper)

		wr.WriteHeader(http.StatusOK)
	default:
		wr.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (w *Webserver) PrivateMessageHandler(wr http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		wr.WriteHeader(http.StatusOK)
		var log []*MessageLogEntry
		w.gossiper.Database.Mux.RLock()
		privateMessagesDB := w.gossiper.PrivateDatabase

		origin := r.URL.Query().Get("name")
		//for _, messagesPerPeer := range privateMessagesDB.MessagesReceived {
		//	for _, m := range messagesPerPeer {
		for _, m := range privateMessagesDB.MessagesReceived[origin] {
			//fmt.Println(fmt.Sprintf("received private m (from %s): %s", origin, m))
			log = append(log, w.ConvertPrivateMessageFormat(m))
		}

		w.gossiper.Database.Mux.RUnlock()

			//for _, m := range privateMessagesDB.MessagesSent[origin] {
			//	fmt.Println(fmt.Sprintf("sent private m (to %s): %s", origin, m))
			//	log = append(log, w.ConvertPrivateMessageFormat(m))
			//}
		//}
		//log =

		//fmt.Println("log: ", log)
		data, err := json.Marshal(log)

		if err != nil {
			wr.WriteHeader(http.StatusInternalServerError)
			return
		}
		wr.Write(data)

	case "POST":
		type PrivateMessage struct {
			Destination string
			Content		string
		}

		var data PrivateMessage
		err := safeDecode(wr, r, &data)
		if err != nil {
			wr.WriteHeader(http.StatusInternalServerError)
			return
		}

		client := client.NewClient("127.0.0.1", strconv.Itoa(w.gossiper.ClientListenerAddress.Port))

		//messageWrapper := &messages.GossipPacket{
		//	Simple: &messages.SimpleMessage {
		//		OriginalName: "",
		//		RelayPeerAddr: "",
		//		Contents: data,
		//	},
		//}
		messageWrapper := &messages.GossipPacket{
			Private: &messages.PrivateMessage{
				Origin: "",
				ID: 0,
				Text: data.Content,
				Destination: data.Destination,
				HopLimit: 15,
			},
		}
		client.SendMessage(messageWrapper)

		wr.WriteHeader(http.StatusOK)
	default:
		wr.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (w *Webserver) NodeHandler(wr http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		peers := w.gossiper.Peers.GetAllPeers()
		peersArray := make([]string, len(peers))
		i := 0
		for _, peer := range peers {
			peersArray[i] = peer.Address.String()
			i++
		}

		data, err := json.Marshal(peersArray)
		if err != nil {
			wr.WriteHeader(http.StatusInternalServerError)
			return
		}

		wr.WriteHeader(http.StatusOK)
		wr.Write(data)

	case "POST":
		var data string
		err := safeDecode(wr, r, &data)
		if err != nil {
			wr.WriteHeader(http.StatusInternalServerError)
			return
		}

		fmt.Println("peer: ", data)
		//w.gossiper.AddPeer(data)
		w.gossiper.AddDeletePeer(data)

		wr.WriteHeader(http.StatusOK)
	default:
		wr.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (w *Webserver) IdHandler(wr http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		wr.WriteHeader(http.StatusOK)
		data, err := json.Marshal(w.gossiper.Name)

		if err != nil {
			wr.WriteHeader(http.StatusInternalServerError)
			return
		}

		wr.Write(data)
	default:
		wr.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (w *Webserver) handleRoutes(wr http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		wr.WriteHeader(http.StatusOK)
		w.gossiper.Database.Mux.RLock()
		allStatuses := w.gossiper.Database.CurrentStatus.Want
		w.gossiper.Database.Mux.RUnlock()

		nodeList := make([]string, len(allStatuses))

		for i, status := range allStatuses {
			nodeList[i] = status.Identifier
		}

		data, err := json.Marshal(nodeList)
		if err != nil {
			wr.WriteHeader(http.StatusInternalServerError)
			return
		}

		wr.Write(data)
	default:
		wr.WriteHeader(http.StatusMethodNotAllowed)
	}
}