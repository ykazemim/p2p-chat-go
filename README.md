# P2P Chat System

A peer-to-peer chat system using HTTP (STUN server) and TCP (direct peer communication).

## Features

- **STUN Server**: HTTP server for peer registration and discovery
- **P2P Messaging**: Direct TCP communication between peers
- **File Transfer**: Send files between peers
- **TLS Encryption**: Secure peer-to-peer connections
- **Redis Cache**: Optional Redis backend for peer storage
- **GUI**: Desktop application using Fyne
- **Docker**: Containerized deployment

## Project Structure

```
├── cmd/
│   ├── stun-server/     # STUN server entry point
│   └── peer/            # Peer client entry point
├── internal/
│   ├── stun/            # STUN server implementation
│   ├── peer/            # Peer TCP client/server
│   ├── protocol/        # Shared message types
│   └── ui/              # Fyne GUI
├── docker/              # Docker configuration
└── README.md
```

## Requirements

- Go 1.21+
- Docker & Docker Compose (for containerized deployment)
- For GUI: X11 or Wayland (Linux), or native support (macOS/Windows)

## Quick Start

### Option 1: Run Locally (using Make)

```bash
# Terminal 1: Start STUN server
make run-stun

# Terminal 2: Start first peer
make run-peer

# Terminal 3: Start second peer
make run-peer
```

### Option 2: Using Docker (Redis + STUN)

```bash
# Start Redis + STUN server in background
make docker-up
# Or directly with docker-compose:
cd docker && docker-compose up -d

# Run peer clients locally
make run-peer    # Terminal 1
make run-peer    # Terminal 2

# Stop when done
make docker-down
# Or directly:
cd docker && docker-compose down
```

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make run-stun` | Run STUN server locally |
| `make run-peer` | Run peer client |
| `make docker-up` | Start Docker (Redis + STUN) |
| `make docker-down` | Stop Docker containers |
| `make docker-logs` | View Docker logs |
| `make build` | Build binaries to `bin/` |
| `make clean` | Clean up binaries and Docker |

## Command Line Options

### STUN Server
```
-addr    Server address (default: ":8080")
-redis   Redis address (e.g., "localhost:6379")
```

### Peer Client
```
-stun    STUN server URL (default: "http://localhost:8080")
-ip      Local IP to advertise (default: "127.0.0.1")
-tls     Use TLS for peer connections (default: true)
```

## API Endpoints

### POST /register
Register a peer with the STUN server.
```json
{
  "username": "alice",
  "ip": "192.168.1.100",
  "port": 9000
}
```

### GET /peers
Get list of all registered peers.

### GET /peerinfo?username=alice
Get information for a specific peer.

## Architecture

```
┌─────────┐     HTTP      ┌─────────────┐     HTTP      ┌─────────┐
│  Peer A │◄─────────────►│ STUN Server │◄─────────────►│  Peer B │
└─────────┘               └─────────────┘               └─────────┘
     │                                                       │
     │                      TCP (TLS)                        │
     └───────────────────────────────────────────────────────┘
                        Direct P2P Connection
```

## License

MIT
