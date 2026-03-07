# Simple Chat

Really simple chat app with Gin + Gorilla WebSockets + PostgreSQL + Sqlc + Svelte

## Makefile

Run `make help` to see available commands. \

Start/stop database

- `make db-up` / `make db-down`

## Scripts

- `./logs.sh` - check logs db service
- `./exec.sh` - execute shell in db service

## Backend

## Frontend

## Requirements

- one global chat room
- every actions get stored in the database
- user:
  - first visit
  - redirect to `/config`
  - user choose a name and submit
  - redirect to `/`
  - load the last 20 messages (paginated), scroll to bottom (scroll up to load older messages)
  - `username has joined the chat`
  - send/receive messages
  - `username has left the chat`

## Database

- Event
  - id
  - userid
  - type (message, join, leave)
  - content
  - created
- User
  - id
  - username
  - created

## Todo

- [ ] Add migrations setup
- [ ] Add db models
- [ ] Add `sqlc`
- [ ] Add `air`
