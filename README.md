#Real - Time Billion Checkbox Backend

This project is a high-performance backend server for a massively multiplayer web application: a page with **one billion checkboxes**. It's built with **Go** and uses **WebSockets** for real-time communication and a **Redis Cluster** for scalable, persistent state management.

The goal is to efficiently handle a large number of concurrent users checking boxes and synchronize the state across all clients in real-time.

## ✨ Features

- **Massively Scalable**: Designed to handle up to 1,000,000,000 checkboxes.
- **Real-Time Updates**: Uses WebSockets and Redis Pub/Sub to instantly broadcast updates to all connected clients.
- **High Performance**: Built with Go for speed and concurrency. Leverages a Redis Cluster for fast, distributed data storage.
- **Efficient Data Protocol**: Communicates with clients using a compact binary protocol to minimize network latency.
- **Rate Limiting**: Protects the server from abuse with a configurable token bucket rate limiter for incoming messages.
- **Resilient**: Includes logic for WebSocket ping/pong keep-alives and connection timeouts.
- **Configurable**: All key parameters are easily configured using environment variables.

## 🛠️ Tech Stack

- **Backend**: Go
- **Real-Time Communication**: Gorilla WebSocket
- **Database/State Management**: Redis Cluster (using go-redis)
- **Configuration**: godotenv, env

## 🚀 Getting Started

Follow these instructions to get the server up and running on your local machine.

### Prerequisites

- **Go**: Version 1.18 or higher.
- **Redis Cluster**: A running Redis Cluster instance. For local development, you can use Docker.
- **Git**: To clone the repository.

### Installation & Setup

Clone the repository:

```bash
git clone https://github.com/amasanelli/onebillioncheckboxes
cd ./onebillioncheckboxes
```

Install dependencies:

```bash
go mod tidy
```

Configure environment variables:
Create a `.env` file in the root and fill in the required values. See the Configuration section below for details.

### Running the Server

Start the server with the following command:

```bash
go run .
```

The server will start, connect to the Redis Cluster, and begin listening for HTTP and WebSocket connections on the address specified in your `.env` file. You should see the output:

```
running...
```

## ⚙️ Configuration

The application is configured using environment variables defined in a `.env` file.

| Variable              | Type               | Description                                                                                                                                         | Example                                                                                          |     |     |     |     |
| --------------------- | ------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------ | --- | --- | --- | --- |
| SERVER_ADDRESS        | string             | Required. The host and port for the HTTP server to listen on.                                                                                       | :8080                                                                                            |     |     |     |     |
| REDIS_ADDRESSES       | \[]string          | Required. A comma-separated list of the Redis Cluster node addresses.                                                                               | "redis-node1:6379,redis-node2:6379,redis-node3:6379"                                             |     |     |     |     |
| REDIS_ADDRESSES_REMAP | map\[string]string | Optional. Remaps Redis addresses, useful for Docker/NAT environments. Format: \`external_address\|internal_address\`. Separate entries with commas. | "172.28.0.101:6379\|:6371,172.28.0.102:6379\|:6372,172.28.0.103:6379\|:6373"                     |
| ME_URL                | string             | Required. URL for your personal website/portfolio, passed to the HTML template.                                                                     | "[https://example.com](https://example.com)"                                                     |     |     |     |     |
| BUY_ME_A_COFFEE_URL   | string             | Required. URL for a "Buy Me a Coffee" or similar donation link, passed to the HTML template.                                                        | "[https://www.buymeacoffee.com/yourname](https://www.buymeacoffee.com/yourname)"                 |     |     |     |     |
| WEBSOCKET_URL         | string             | Required. The public-facing WebSocket URL for the client to connect to.                                                                             | "ws\://localhost:8080/ws"                                                                        |     |     |     |     |
| LIMITER_LIMIT         | int                | Required. The number of events allowed per second for the rate limiter.                                                                             | 10                                                                                               |     |     |     |     |
| LIMITER_BURST         | int                | Required. The maximum burst size for the rate limiter.                                                                                              | 20                                                                                               |     |     |     |     |
| ALLOWED_ORIGINS       | \[]string          | Required. A comma-separated list of allowed origins for WebSocket connections (CORS).                                                               | "[http://localhost:8080,https://your-domain.com](http://localhost:8080,https://your-domain.com)" |     |     |     |     |

### .env.example

```env
#Server Configuration
SERVER_ADDRESS=":8080"
WEBSOCKET_URL="ws://localhost:8080/ws"

#Redis Cluster Configuration
REDIS_ADDRESSES="localhost:7001,localhost:7002,localhost:7003"
REDIS_ADDRESSES_REMAP=""

#Frontend Template Data
ME_URL="https://github.com/your-username"
BUY_ME_A_COFFEE_URL="https://www.buymeacoffee.com/your-username"

#Security &Performance
LIMITER_LIMIT=10
LIMITER_BURST=20
ALLOWED_ORIGINS="http://localhost:8080,http://127.0.0.1:8080"
```

## 🔌 WebSocket API Protocol

The server communicates with clients over a WebSocket connection using a simple and efficient binary protocol. All multi-byte integers are encoded in **Little Endian** format.

**Endpoint:** `/ws`

### Server -> Client Messages

**Initial State (On Connect)**

- Sends a single 4-byte message containing the current total count of checked boxes.
- Payload: `[4 bytes]` - Total checked count as `uint32`.

**Broadcast Update**

- When any client checks a new box, broadcasts an 8-byte message.
- Payload: `[8 bytes]`

  - Bytes 0-3: The new total checked count (`uint32`)
  - Bytes 4-7: The ID of the checkbox just checked (`uint32`)

**Range Response**

- Response to a client's range request.
- Payload: `[4 + N bytes]`

  - Bytes 0-3: Starting checkbox ID of the requested range (`uint32`)
  - Bytes 4...N-1: Bitmask representing the checked status.

### Client -> Server Messages

**Check a Box**

- Payload: `[4 bytes]` - Checkbox ID (`uint32`) from 1 to 1,000,000,000

**Request a Range of Boxes**

- Payload: `[8 bytes]`

  - Bytes 0-3: Starting checkbox ID (`uint32`)
  - Bytes 4-7: Ending checkbox ID (`uint32`)

- Maximum range: `MAX_CHECKBOXES_PER_REQUEST` (100,000)

## 📁 Project Structure

```
.
├── go.mod          # Go module definitions
├── go.sum          # Go module checksums
├── main.go         # Main application entry point, server setup, and graceful shutdown
├── config.go       # Application constants and configuration values
├── handlers.go     # HTTP handlers for serving the HTML page and static assets
├── websocket.go    # Core WebSocket logic for handling connections, messages, and Redis communication
├── public/         # Directory for static assets (CSS, JS, images)
└── templates/
    └── index.html  # HTML template for the main page
```
