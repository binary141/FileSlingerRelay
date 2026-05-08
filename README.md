# FileSlingerServer

A lightweight WebSocket relay server that bridges HTTP file uploads to connected WebSocket clients. A receiver opens a named session over WebSocket, and a sender POSTs file data to the same token — the server streams the bytes directly to the receiver in real time.

## How it works

1. **Receiver** connects via WebSocket at `GET /session/{token}` and waits.
2. **Sender** POSTs file data to `POST /upload/{token}` with the same token.
3. The server forwards the raw bytes as a binary WebSocket message to the receiver.

Tokens are arbitrary strings. Only one receiver may hold a token at a time; a second connection attempt with the same token is rejected.

## API

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/ping` | Health check — returns `pong` |
| `GET` | `/session/{token}` | Open a WebSocket session (receiver) |
| `POST` | `/upload/{token}` | Upload data to an open session (sender) |

## Running

### Go

```bash
go run .
```

Listens on `:8080` by default. Override with the `PORT` environment variable:

```bash
PORT=9090 go run .
```

### Docker

```bash
docker compose up --build
```

## Building

```bash
go build -o server .
```
