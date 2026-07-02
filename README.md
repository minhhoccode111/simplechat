# Simple Chat

Multi-user WebSocket chat server with gorilla/websocket, Hub fan-out pattern,
random guest nicknames, inline HTML

## Concepts Learned

- Golang + Websocket
- Simple deploy script that copy the binary to production vps
- QA.md

## Bugs Encountered

### Hub deadlock: goroutine sending to itself via unbuffered channel

**Problem:** The hub runs a single goroutine that reads from three channels
(register, unregister, broadcast) via `select`. When a new client connects,
`register()` runs inside this goroutine. Inside `register()`, the code sends a
"Welcome" message to the broadcast channel. But the broadcast channel is
unbuffered — a send blocks until someone reads. The only goroutine that reads
from the broadcast channel is the hub goroutine itself, which is currently busy
running `register()`. So the send blocks forever, waiting for itself.

It is like mailing yourself a letter and standing by the mailbox waiting for
the mailman to deliver it — you are the one inside delivering the mail, so
nobody else will ever bring it to you.

**Symptom:** First client connects OK and receives their username assignment,
then the hub freezes. No "Welcome" message reaches anyone, user messages go
nowhere, and new clients cannot connect. When the frustrated user closes the
browser tab, the WebSocket sends a close frame (code 1001, "going away"),
which the server logs as an error. That log is a symptom of the deadlock, not
the root cause.

**Lesson:** An unbuffered Go channel requires the sender and receiver to be
in DIFFERENT goroutines. Sending through a channel from within the same
goroutine that is supposed to receive from it will deadlock.

## Architecture Evolution

### V1: Blunt approach

First version stored `*websocket.Conn` directly in hub's client map, keyed by
auto-increment string ID. Hub's `broadcast()` called `conn.WriteJSON` directly
in a loop. Single goroutine per client (the HTTP handler) did both reading and
writing — blocking on `ReadJSON`, then sending to hub's broadcast channel,
then blocking on that send until hub processed it.

Problems:

- Hub blocks on slowest client — one slow reader holds up every broadcast
- `register()` deadlocked on `broadcastCh` because hub sent to its own channel
- No per-client buffering, no backpressure

### V2: Client struct + per-client send channel

Refactored to introduce a `Client` struct that wraps connection + context:

```
Client {
    conn     *websocket.Conn
    send     chan Message      // buffered (cap 256)
    username string
}
```

Two goroutines per client:

- `readPump()` — reads from conn, sends to hub
- `writePump()` — reads from `send` channel, writes to conn

Hub never calls `WriteJSON` directly. Hub sends to each client's `send`
channel instead. The `writePump` goroutine is the sole writer to `conn`,
eliminating races and blocking.

Per-client buffered channel solves the slow client problem — hub enqueues and
moves on. If channel fills up, `select/default` drops the slow client.

**Key insight:** Separate the concerns — hub manages fan-out, each client
manages its own socket. Channels between them keep things decoupled and
non-blocking.
