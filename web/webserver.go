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

func New(port string, g *communications.Gossiper) *Webserver {
	return &Webserver{port: port, router: http.NewServeMux(), gossiper: g}
}

func (w *Webserver) Start() {
	w.router.HandleFunc("/message", w.MessageHandler)
	w.router.HandleFunc("/node", w.NodeHandler)
	w.router.HandleFunc("/id", w.IdHandler)
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

func (w *Webserver) MessageHandler(wr http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		fmt.Println("GET!")
		wr.WriteHeader(http.StatusOK)

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

func (w *Webserver) NodeHandler(wr http.ResponseWriter, r *http.Request) {
	//var data string
	//err := safeDecode(wr, r, data)
	//if err != nil {
	//	fmt.Println("Error (node handler): ", err)
	//	//return
	//}
}

func (w *Webserver) IdHandler(wr http.ResponseWriter, r *http.Request) {
	//var data string
	//err := safeDecode(wr, r, data)
	//if err != nil {
	//	fmt.Println("Error (id handler): ", err)
	//	//return
	//}
}