# Trafiks

Trafiks is a modern API Gateway and Proxy Service built with Go and React. It provides intelligent request routing, caching, monitoring, and management capabilities for your microservices and APIs.

## Features

- **Multi-Source Service Discovery**: Support for database-backed/external domain services, Docker containers, and Kubernetes services
- **HTTPS/TLS Support**: Automatic certificate management with SNI (Server Name Indication) support
- **Request Caching**: Configurable caching with TTL support for improved performance
- **Request Logging & Replay**: Comprehensive request logging with replay capabilities
- **Real-time Metrics**: Live metrics streaming via Server-Sent Events (SSE)
- **Webhooks**: Account-level webhooks for event notifications
- **API Key Management**: Secure API key authentication
- **Modern Dashboard**: Built-in React dashboard for service management
- **Docker Compose Support**: Easy local development setup
- **Kubernetes Ready**: Helm chart for production deployments

## Architecture

Trafiks consists of two main components:

- **Backend**: API server that handles proxy requests, service management, and API endpoints
- **Agent**: Background worker that processes webhook deliveries and other async tasks

## Quick Start

### Prerequisites

- Go 1.19 or higher
- PostgreSQL 12 or higher
- Redis 6 or higher
- Node.js 18+ (for UI development)

### Local Development

1. **Clone the repository:**
```bash
git clone <repository-url>
cd trafiks
```

2. **Set up configuration:**
```bash
# Copy example config
cp config.example.json $HOME/.config/trafiks/config.json

# Edit config.json with your database and Redis settings
```

3. **Start dependencies:**
```bash
make startdb
```

4. **Run the application:**
```bash
# Development mode with hot reload
make dev

# Or run directly
go run ./cmd/server/main.go
```

5. **Access the dashboard:**
- Open `http://localhost:8889/dashboard`
- Login with default credentials: `admin@trafiks.local` / `admin123`

### Docker Compose

For a complete local setup with all services:

```bash
# Start all services (backend, agent, postgres, redis)
make start

# Or build and start
make start-app

# Stop all services
make stop

# Clean up everything including volumes
make clean
```

## Configuration

Configuration is stored in `$HOME/.config/trafiks/config.json`. See `config.example.json` for all available options.

### Key Configuration Options

```json
{
  "server_port": "8889",           // HTTP server port
  "tls_port": "8443",              // HTTPS server port
  "app_base_url": "http://localhost:8889",
  "environment": "local",
  
  "database": {
    "host": "localhost",
    "port": 5432,
    "user": "admin",
    "password": "password",
    "name": "trafiks",
    "ssl_mode": "disable",
    "log_enabled": false           // Disable PostgreSQL query logging
  },
  
  "redis": {
    "host": "localhost:6379",
    "username": "",
    "password": "",
    "db": 0
  },
  
  "docker": {
    "socket_path": "unix:///var/run/docker.sock"  // For Docker source (optional)
  },
  
  "dashboard": {
    "enabled": true                // Enable/disable dashboard UI
  },
  
  "bootstrap": {
    "user": {
      "email": "admin@trafiks.local",
      "password": "admin123",
      "first_name": "admin",
      "last_name": "user"
    }
  }
}
```

## Service Sources

Trafiks supports multiple service discovery mechanisms:

### 1. Trafiks Source (Database)
Default source where services are stored in the database. Users provide:
- Proxy URL (domain)
- Target Backend URL
- Configuration (cache, headers, query params)

### 2. Docker Source
Discovers services from Docker containers using labels:
- Configure `docker.socket_path` in config
- Services use Docker labels to match containers
- Auto-detects network and container addresses

### 3. Kubernetes Source
Discovers services from Kubernetes Ingress/Service resources using annotations.

## HTTPS/TLS Support

Trafiks supports HTTPS with automatic certificate management:

- **Self-Signed Certificates**: Automatically generated for development
- **SNI Support**: Multiple domains on the same port
- **Certificate Details**: View certificate info in the dashboard
- **HTTP to HTTPS Redirect**: Configurable per service

### Enabling HTTPS for a Service

1. Create/update a service with `scheme: "https"`
2. Trafiks automatically generates a self-signed certificate
3. Access via `https://your-domain.local:8443`

## Building

### Build UI and Binary

```bash
# Build UI and embed into Go binary
make build

# Or separately
make build-ui        # Build React UI
go build -o trafiks ./cmd/server/main.go
```


## Development

### Running Tests

```bash
go test ./...
```

### Development Mode

```bash
# Backend with hot reload
make dev

# Agent with hot reload
make dev-agent
```

### Building UI

```bash
cd web/trafiks-ui
npm install
npm run dev        # Development server
npm run build      # Production build
```

## License

This project is licensed under the MIT License.
