package main

import (
	"math/rand"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type message struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

var conns []*websocket.Conn

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil)
		id := strconv.Itoa(rand.Intn(99999))
		conns = append(conns, conn)
		go func() {
			conn.WriteJSON(message{Type: "system", Content: "Welcome user " + id})
			conn.WriteJSON(message{Type: "set-username", Content: id})
		}()
	})
	http.ListenAndServe(":8000", nil)
}
