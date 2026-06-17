package main

import (
	"log"
	"math/rand"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type     string `json:"type"`
	Username string `json:"username,omitempty"`
	Content  string `json:"content"`
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Client connected: %s", client.username)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("Client disconnected: %s", client.username)

		case msg := <-h.broadcast:
			var toRemove []*Client
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- msg:
				default:
					toRemove = append(toRemove, client)
				}
			}
			h.mu.RUnlock()
			if len(toRemove) > 0 {
				h.mu.Lock()
				for _, client := range toRemove {
					if _, ok := h.clients[client]; ok {
						delete(h.clients, client)
						close(client.send)
					}
				}
				h.mu.Unlock()
			}
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan Message
	username string
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			break
		}
		msg.Username = c.username
		msg.Type = "message"
		c.hub.broadcast <- msg
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()

	for msg := range c.send {
		err := c.conn.WriteJSON(msg)
		if err != nil {
			break
		}
	}
}

func main() {
	hub := NewHub()
	go hub.Run()

	http.HandleFunc("/", serveIndex)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	log.Println("Server starting on :8000")
	log.Fatal(http.ListenAndServe(":8000", nil))
}

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Simple Chat</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: system-ui, sans-serif; background: #1a1a2e; color: #eee; height: 100vh; display: flex; flex-direction: column; }
        #messages { flex: 1; overflow-y: auto; padding: 1rem; }
        .message { margin-bottom: 0.5rem; }
        .message .username { color: #e94560; font-weight: bold; }
        .message .content { color: #eee; }
        .system { color: #888; font-style: italic; margin-bottom: 0.5rem; }
        #form { display: flex; padding: 1rem; background: #16213e; }
        #input { flex: 1; padding: 0.75rem; border: none; border-radius: 4px; font-size: 1rem; background: #0f3460; color: #eee; }
        #input::placeholder { color: #888; }
        button { margin-left: 0.5rem; padding: 0.75rem 1.5rem; border: none; border-radius: 4px; background: #e94560; color: #fff; font-size: 1rem; cursor: pointer; }
        button:hover { background: #c73e54; }
    </style>
</head>
<body>
    <div id="messages"></div>
    <form id="form">
        <input id="input" autocomplete="off" placeholder="Type a message..." autofocus />
        <button type="submit">Send</button>
    </form>
    <script>
        const messages = document.getElementById('messages');
        const form = document.getElementById('form');
        const input = document.getElementById('input');
        const ws = new WebSocket('ws://' + location.host + '/ws');

        ws.onmessage = function(event) {
            const msg = JSON.parse(event.data);
            const div = document.createElement('div');
            if (msg.type === 'system') {
                div.className = 'system';
                div.textContent = msg.content;
            } else {
                div.className = 'message';
                div.innerHTML = '<span class="username">' + msg.username + ':</span> <span class="content">' + msg.content + '</span>';
            }
            messages.appendChild(div);
            messages.scrollTop = messages.scrollHeight;
        };

        form.onsubmit = function(e) {
            e.preventDefault();
            if (input.value.trim()) {
                ws.send(JSON.stringify({ content: input.value.trim() }));
                input.value = '';
            }
        };
    </script>
</body>
</html>`

func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(indexHTML))
}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Upgrade error: %v", err)
		return
	}

	username := "Guest" + string(
		rune('0'+rand.Intn(9)),
	) + string(
		rune('0'+rand.Intn(10)),
	) + string(
		rune('0'+rand.Intn(10)),
	)

	client := &Client{hub: hub, conn: conn, send: make(chan Message, 256), username: username}
	hub.register <- client

	go client.writePump()
	go client.readPump()
}

