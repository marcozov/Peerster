package web

import (
	"encoding/json"
	"fmt"
	"github.com/marcozov/Peerster/communications"
	"io/ioutil"
	"net/http"
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

	if err != nil {
		fmt.Println("error: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return err

	}

	fmt.Println("data: ", data)
	err = json.Unmarshal(data, out)
	if err != nil {
		fmt.Println("error2: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return err
	}

	//w.WriteHeader(http.StatusBadRequest)
	return nil

	//if err == nil {
	//	err := json.Unmarshal(data, out)
	//	if err == nil {
	//		return nil
	//	} else {
	//		w.WriteHeader(http.StatusBadRequest)
	//		return err
	//	}
	//} else {
	//	w.WriteHeader(http.StatusBadRequest)
	//	return err

	//}
}

func (w *Webserver) MessageHandler(wr http.ResponseWriter, r *http.Request) {
	var data string
	err := safeDecode(wr, r, &data)
	if err != nil {
		fmt.Println("Error (message handler): ", err)
		return
	}

	fmt.Println("wat")

	switch r.Method {
	case "GET":
		fmt.Println("GET!")
		wr.WriteHeader(http.StatusOK)

	case "POST":
		fmt.Println("POST@")
		wr.WriteHeader(http.StatusOK)
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