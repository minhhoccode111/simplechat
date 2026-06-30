# Simple Chat

Multi-user WebSocket chat server with gorilla/websocket, Hub fan-out pattern,
random guest nicknames, inline HTML

## Concepts Learned

- Golang + Websocket
- Simple deploy script that copy the binary to production vps

## QA

Q: what is realtime

A:

realtime = server pushes data to client immediately when event happens, no waiting/polling.

For web: traditional HTTP request-response is synchronous (ask → wait → get).

Realtime flips it — connection stays open, server sends messages as they occur.

Core patterns:

- WebSocket — persistent TCP connection, full-duplex
- Server-Sent Events — one-way server→client over HTTP
- Long-polling — hack before real protocols existed

---

Q: what is full-duplex

A:

Both sides send data simultaneously, no waiting turns.

Think walkie-talkie (half-duplex) vs phone call (full-duplex):

- `Half-duplex` — push-to-talk, one speaks at a time
- `Full-duplex` — both talk and listen at same time

HTTP is half-duplex: client sends request, server sends response, connection idle. WebSocket is full-duplex: client sends message while server sends message in same instant, both reading/writing independently on same connection.

In Go WebSocket: `ReadMessage` in one goroutine, `WriteMessage` in another — they don't block each other. That's full-duplex in practice.

---

Q: is server send event simplex

A:

Yes. SSE is simplex — server → client only, one direction.

Client opens event stream via EventSource, server sends data: lines.

Client can't send anything back through that channel. If client needs to send data, it uses separate HTTP requests.

- Simplex = one way (radio broadcast)
- Half-duplex = one at a time (walkie-talkie)
- Full-duplex = both simultaneously (phone call, WebSocket)

---

Q: how do i serve file for a route in a go server

A:

```go
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})
```

---

Q: how do i define a websocket endpoint in a Go server

A:

Use net/http + gorilla/websocket (most common) or the newer nhooyr.io/websocket.

Core pattern:

1. Register handler via `http.HandleFunc("/ws", handler)`
2. In handler, upgrade HTTP connection with `upgrader.Upgrade(w, r, nil)` → returns `*websocket.Conn`
3. Read/write with `conn.ReadMessage()` / `conn.WriteMessage()` in goroutines

`gorilla/websocket` upgrader struct:

```go
var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
}
```

Key methods: `upgrader.Upgrade(w, r, nil)` upgrades HTTP→WS, `conn.ReadJSON(v)` / `conn.WriteJSON(v)` for JSON, `conn.ReadMessage()` for raw bytes.

Need `gorilla/websocket` — `go get github.com/gorilla/websocket`.

---

Q: what is the meaning of 'upgrade' a connection?

A:

HTTP/1.1 has an Upgrade mechanism that lets a client-server switch from HTTP to a different protocol mid-connection.

WebSocket starts as HTTP — client sends a `GET /ws` request with headers:

```
Upgrade: websocket
Connection: Upgrade
```

Server responds with `101 Switching Protocols`, then the TCP socket transitions from HTTP to the WebSocket framing protocol. No more HTTP request/response — now it's full-duplex message frames.

`upgrader.Upgrade()` does that negotiation server-side: validates the handshake headers, writes the 101 response, and gives you a `*websocket.Conn` over the same underlying TCP connection.

---

Q: what is the `messageType` returned from `ReadMessage()`

A:

messageType is an int from package-level constants:

- websocket.TextMessage = 1 — UTF-8 text
- websocket.BinaryMessage = 2 — raw binary
- websocket.CloseMessage = 8 — close frame
- websocket.PingMessage = 9 — ping
- websocket.PongMessage = 10 — pong

Maps to WebSocket opcode (https://developer.rfc-editor.org/rfc/rfc6455#section-11.8). 99% of the time you see TextMessage or BinaryMessage.

---
