package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
)

type message struct {
	Type     string `json:"type"`
	Content  string `json:"content"`
	Username string `json:"username"`
}

type hub struct {
	n            int
	clients      map[string]*websocket.Conn
	registerCh   chan *websocket.Conn
	unregisterCh chan *websocket.Conn
	broadcastCh  chan message
	closeCh      chan struct{}
}

func newHub() *hub {
	return &hub{
		clients:      make(map[string]*websocket.Conn),
		registerCh:   make(chan *websocket.Conn, 10),
		unregisterCh: make(chan *websocket.Conn, 10),
		broadcastCh:  make(chan message, 10),
		closeCh:      make(chan struct{}),
	}
}

func (h *hub) run() {
	go func() {
		for {
			select {
			case c := <-h.registerCh:
				h.register(c)
			case c := <-h.unregisterCh:
				h.unregister(c)
			case m := <-h.broadcastCh:
				h.broadcast(m)
			case <-h.closeCh:
				log.Println("exiting hub.run()")
				return
			}
		}
	}()
}

func (h *hub) register(c *websocket.Conn) {
	h.n++
	id := strconv.Itoa(h.n)
	h.clients[id] = c
	// selft
	c.WriteJSON(message{Type: "set-username", Content: id, Username: id})
	// everyone, call h.broadcast directly to prevent deadlock
	h.broadcast(message{Type: "system", Content: "Welcome user " + id})
}

func (h *hub) unregister(c *websocket.Conn) {
	var id string
	for k, v := range h.clients {
		if v == c {
			id = k
		}
	}
	delete(h.clients, id)
	// everyone, call h.broadcast directly to prevent deadlock
	h.broadcast(message{Type: "system", Content: "Goodbye user " + id})
}

func (h *hub) broadcast(m message) {
	for k, v := range h.clients {
		err := v.WriteJSON(m)
		if err != nil {
			log.Printf("error writing to %s client", k)
		}
	}
}

func main() {
	var upgrader = websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024}

	h := newHub()
	h.run()
	defer close(h.closeCh)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("error upgrading connection: %v", err)
			return
		}

		defer func() {
			h.unregisterCh <- conn
		}()

		h.registerCh <- conn

		for {
			var m message

			err := conn.ReadJSON(&m)
			if err != nil {
				log.Printf("error reading JSON: %v", err)
				break
			}

			h.broadcastCh <- m
		}
	})

	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Printf("error ListenAndServe: %v", err)
	}
}
