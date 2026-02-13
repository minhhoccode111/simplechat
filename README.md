# Simple Chat

Really simple chat app with Gin + Svelte

## Requirements

Requirements:

- [ ] just one global room chat
- [ ] users get all messages
- [ ] user send a message
- [ ] user is identified by browser
  - new tab → same user
  - new browser → new user
  - new tab in incognito → new user
  - new tab in the same incognito → same user

Extra:

- [ ] user connect/disconnect events print to chat history

## Database

- Message
  - id
  - content
  - created
  - user_id

## Todo

- [x] setup devops dir to start db fast
- [x] setup backend dir
- [x] setup database schema
- [ ] setup frontend dir
