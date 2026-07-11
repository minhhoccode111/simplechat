package main

import (
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const pongWait = 60 * time.Second
const pingPeriod = pongWait / 10 * 9
const writeWait = 10 * time.Second

type Message struct {
	Type     string `json:"type"`
	Content  string `json:"content"`
	Username string `json:"username"`
}

func NewMessage(messageType, content, username string) Message {
	return Message{Type: messageType, Content: content, Username: username}
}

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan Message
	username string
}

func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan Message, 256),
		username: generateUsername(),
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregisterCh <- c
		c.conn.Close()
	}()

	err := c.conn.SetReadDeadline(time.Now().Add(pongWait))
	if err != nil {
		return
	}

	c.conn.SetPongHandler(func(appData string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	done := make(chan struct{})
	defer close(done)
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-ticker.C:
				c.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait))
			case <-done:
				return
			}
		}
	}()

	for {
		var msg Message

		err := c.conn.ReadJSON(&msg)
		if err != nil {
			break
		}

		c.hub.broadcastCh <- msg
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()

	for m := range c.send {
		err := c.conn.WriteJSON(m)
		if err != nil {
			break
		}
	}
}

type Hub struct {
	clients      map[*Client]bool
	registerCh   chan *Client
	unregisterCh chan *Client
	broadcastCh  chan Message
	mu           sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:      make(map[*Client]bool),
		registerCh:   make(chan *Client),
		unregisterCh: make(chan *Client),
		broadcastCh:  make(chan Message, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.registerCh:
			h.clients[c] = true
			h.broadcastCh <- NewMessage("system", "Welcome "+c.username, "")
		case c := <-h.unregisterCh:
			delete(h.clients, c)
			close(c.send)
			h.broadcastCh <- NewMessage("system", "Goodbye "+c.username, "")
		case m := <-h.broadcastCh:
			for c := range h.clients {
				select {
				// try to send to client's send channel
				case c.send <- m:
				// if client is too slow (buffer 256 full), default to drop it
				default:
					delete(h.clients, c)
					close(c.send)
					h.broadcastCh <- NewMessage("system", "Goodbye "+c.username, "")
				}
			}
		}
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8088"
	}

	var upgrader = websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024}
	hub := NewHub()
	go hub.Run()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("error upgrading connection: %v", err)
			return
		}

		c := NewClient(hub, conn)
		hub.registerCh <- c
		go c.readPump()
		go c.writePump()

		c.send <- NewMessage("set-username", c.username, "")
	})

	err := http.ListenAndServe("127.0.0.1:"+port, nil)
	if err != nil {
		log.Printf("error ListenAndServe: %v", err)
	}
}

func generateUsername() string { return uuid.NewString()[:5] }
