# WebSocket + Go — Lecture Slides

---

## Slide 1: Why WebSocket?

| HTTP               | WebSocket                       |
| ------------------ | ------------------------------- |
| Request-response   | Full-duplex                     |
| Client asks first  | Server pushes anytime           |
| Headers every time | Minimal framing after handshake |
| Stateless          | Persistent connection           |

**Chat problem**: With HTTP, each new message needs a new request (polling). With WebSocket, server sends messages as they arrive.

---

## Slide 2: The Handshake (HTTP → WS)

```
Client                          Server
  |-------- HTTP GET ---------->|
  |   Upgrade: websocket        |
  |   Connection: Upgrade       |
  |   Sec-WebSocket-Key: ...    |
  |                             |
  |<-- 101 Switching Protocols -|
  |   Upgrade: websocket        |
  |   Sec-WebSocket-Accept: ... |
  |                             |
  |======= TCP socket open =====|
  |   bidirectional frames      |
```

- Starts as HTTP, then "upgrades" to WebSocket
- After `101`, same TCP socket carries frames both ways
- `gorilla/websocket` handles this via `upgrader.Upgrade()`

---

## Slide 3: WebSocket Frame Structure

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-------+-+-------------+-------------------------------+
|F|R|R|R| opcode|M| Payload len |    Extended payload length    |
|I|S|S|S|  (4)  |A|     (7)     |             (16/64)           |
|N|V|V|V|       |S|             |                               |
| |1|2|3|       |K|             |                               |
+-+-+-+-+-------+-+-------------+-------------------------------+
```

**Key opcodes**: `0x1` (text), `0x8` (close), `0x9` (ping), `0xA` (pong)

**Don't need to memorize** — library abstracts this.

---

## Slide 4: gorilla/websocket at a Glance

```go
import "github.com/gorilla/websocket"
```

Three core types:

| Type       | Role                                      |
| ---------- | ----------------------------------------- |
| `Upgrader` | Upgrades HTTP → WS on server              |
| `Conn`     | Represents one WS connection              |
| `Dialer`   | Client-side connect (not needed for chat) |

**Server flow**: HTTP handler → `upgrader.Upgrade(w, r, nil)` → `Conn`

---

## Slide 5: Upgrade Pattern

```go
var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        return true // allow all (dev only!)
    },
}

func handleWS(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Print("upgrade:", err)
        return
    }
    // conn is now ready for read/write
}
```

- `Upgrade()` sends the `101` response
- Returns `*websocket.Conn` — your handle to the socket

---

## Slide 6: Read & Write

```go
// Read JSON message from client
var msg Message
err := conn.ReadJSON(&msg)

// Write JSON message to client
err := conn.WriteJSON(msg)
```

**Both block** the calling goroutine.

**Consequence**: Need **two goroutines per client** — one reading, one writing.

---

## Slide 7: The Blocking Problem

```
Time →   readPump goroutine          writePump goroutine
         ┌──────────────────┐        ┌──────────────────┐
         │ ReadJSON (block) │        │ ← msg from chan  │
         │   ...waiting...  │        │ WriteJSON        │
         │   ← msg arrives  │        │ ← msg from chan  │
         │ Process msg      │        │ WriteJSON        │
         │ ReadJSON (block) │        │ ← msg from chan  │
         └──────────────────┘        └──────────────────┘
```

**One goroutine reads** (receives from browser), **one writes** (sends to browser).

They share the `Conn` — gorilla handles concurrent access with locks.

---

## Slide 8: Concurrency Model — Hub

```
                        ┌───────────┐
              register→│           │
  Client ──────────────→│           │
                        │   Hub     │──broadcast→ all clients
  Client ──────────────→│  (goroutine)│
            unregister→│           │
  Client ──────────────→│           │
                        └───────────┘
```

- Hub is a **goroutine** that owns client state
- All mutation happens inside one `select` loop
- Channels are the only way in/out

---

## Slide 9: Hub Internals

```go
type Hub struct {
    clients    map[*Client]bool
    broadcast  chan Message
    register   chan *Client
    unregister chan *Client
    mu         sync.RWMutex   // protect clients map
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.clients[client] = true
        case client := <-h.unregister:
            delete(h.clients, client)
            close(client.send)
        case msg := <-h.broadcast:
            for client := range h.clients {
                select {
                case client.send <- msg:
                default:
                    // client too slow → disconnect
                    delete(h.clients, client)
                    close(client.send)
                }
            }
        }
    }
}
```

**Pattern**: Actor model — one goroutine owns the state, communicates via channels.

---

## Slide 10: Client Struct

```go
type Client struct {
    hub      *Hub
    conn     *websocket.Conn
    send     chan Message    // buffered channel for outgoing
    username string
}
```

**Key**: `send` channel buffers messages so we don't block the hub if client is slow.

---

## Slide 11: readPump — Incoming

```go
func (c *Client) readPump() {
    defer func() {
        c.hub.unregister <- c    // clean up on exit
        c.conn.Close()
    }()

    for {
        var msg Message
        err := c.conn.ReadJSON(&msg)
        if err != nil {
            break   // client disconnected or error
        }
        msg.Username = c.username
        msg.Type = "message"
        c.hub.broadcast <- msg   // send to hub
    }
}
```

- Runs in its own goroutine
- Blocks on `ReadJSON`
- On any error (close, timeout, bad data) → cleanup

---

## Slide 12: writePump — Outgoing

```go
func (c *Client) writePump() {
    defer c.conn.Close()

    for msg := range c.send {
        err := c.conn.WriteJSON(msg)
        if err != nil {
            break
        }
    }
}
```

- Blocks on range over channel
- Each message from hub is written to the WebSocket
- Channel close → loop exits → conn closes

---

## Slide 13: Why the Buffered Channel?

```go
send chan Message  // unbuffered — bad
send chan Message  // cap: 256 — good
```

**Scenario**: Hub broadcasts to 100 clients. One client is slow (bad network).

- **Unbuffered**: Hub blocks on that client, delaying everyone
- **Buffered (256)**: Hub writes to channel and moves on. Slow client gets a backlog. If channel fills → `default` case drops them

This is **backpressure** — protect the fast from the slow.

---

## Slide 14: Message Flow End-to-End

```
Browser A                Server                   Browser B
   │                       │                         │
   │──WriteJSON("hi")─────→│                         │
   │                       │ ReadJSON → "hi"         │
   │                       │ hub.broadcast ← "hi"    │
   │                       │ client.send ← "hi"      │
   │                       │    for each B's chan    │
   │                       │ WriteJSON("hi")─────────→│
   │                       │                         │ Display "hi"
   │                       │                         │
```

**Latency**: 1 network round trip. No polling delay.

---

## Slide 15: Message Type Discipline

```go
type Message struct {
    Type     string `json:"type"`
    Username string `json:"username,omitempty"`
    Content  string `json:"content"`
}
```

Use `Type` field to distinguish:

| Type             | Meaning                       |
| ---------------- | ----------------------------- |
| `"message"`      | Chat message from user        |
| `"system"`       | Join/leave notification       |
| `"set-username"` | Tell client its assigned name |

The browser switches on `msg.type` to decide how to render.

---

## Slide 16: main() — Wiring

```go
func main() {
    hub := NewHub()
    go hub.Run()           // start hub goroutine

    http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
        serveWs(hub, w, r)
    })
    http.HandleFunc("/", serveIndex)

    log.Fatal(http.ListenAndServe(":8000", nil))
}
```

**Only two routes**: `/` for HTML page, `/ws` for WebSocket.

---

## Slide 17: serveWs — Client Lifecycle

```go
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
    conn, _ := upgrader.Upgrade(w, r, nil)

    client := &Client{
        hub:      hub,
        conn:     conn,
        send:     make(chan Message, 256),
        username: generateName(),
    }
    hub.register <- client

    go client.writePump()
    go client.readPump()

    client.send <- Message{Type: "set-username", Content: client.username}
}
```

**Per connection**: 1 HTTP upgrade + 2 goroutines + 1 buffered channel.

**Three concurrent things per client**:

- Hub (shared, one for all)
- readPump goroutine
- writePump goroutine

---

## Slide 18: What Could Go Wrong

| Problem            | Symptom             | Fix                                      |
| ------------------ | ------------------- | ---------------------------------------- |
| Slow client        | Hub blocks on write | Buffered chan + `select/default`         |
| Client disconnects | `ReadJSON` error    | Break → unregister                       |
| writePump blocked  | Goroutine leak      | `close(client.send)` triggers range exit |
| Concurrent write   | Race condition      | One writer goroutine per conn            |
| Client never reads | Buffer fills        | `default` case drops them                |

---

## Slide 19: Key Go Concepts Used

| Concept           | Where                                            |
| ----------------- | ------------------------------------------------ |
| `goroutine`       | One per client (2) + one for hub                 |
| `chan`            | register, unregister, broadcast, per-client send |
| `select`          | Hub multiplexes register/unregister/broadcast    |
| `range over chan` | writePump loops over send channel                |
| `close(chan)`     | Signals writePump to exit                        |
| `defer`           | Cleanup on function exit                         |
| `sync.RWMutex`    | Protect clients map (alternatives exist)         |

---

## Slide 20: Building Order (Suggested)

1. **`main.go`**: HTTP server, `/ws` handler, `upgrader`
2. **`Message` struct**: Define JSON shape
3. **`Client` struct**: `conn`, `send` chan, `readPump()`, `writePump()`
4. **`Hub` struct**: `clients` map, `Run()` with `select`
5. **Wire it**: `serveWs` → upgrade → create client → register → launch goroutines
6. **Test**: Open two browser tabs, type messages

Start with **one client** and `ReadJSON`/`WriteJSON` directly (no hub). Add hub after you see two goroutines working.
