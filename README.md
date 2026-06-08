# JUNKyard - Centralized Log Aggregation System

> "Throw all your logs into the junkyard."

JUNKyard is a lightweight, Go-based log aggregation system designed for centralized log collection across multiple VMs in a CIA academic project environment. It provides a web UI, powerful query capabilities, and full-text search across logs from distributed sources.

![JUNKyard Dashboard](https://i.imgur.com/pzfycnd.png)

## Features

- **Centralized Collection** - Single monitoring VM aggregates logs from all infrastructure
- **High Performance** - SQLite with WAL mode for concurrent read performance during writes
- **Full-Text Search** - FTS5 integration with automatic indexing (falls back to LIKE search if FTS5 unavailable)
- **Multiple Ingestion Methods**:
  - **Syslog (RFC 3164)** - TCP and UDP on port 5514 for traditional log forwarding
  - **HTTP API** - POST to `/api/ingest` for single or batch logs
  - **Direct POST** - Application-native log submission
- **Web UI** - Dark-themed dashboard with real-time stats and filtering
- **REST API** - Comprehensive endpoints for programmatic access
- **pfSense Compatible** - UDP syslog support with automatic hostname resolution
- **Production Ready** - Systemd integration, graceful shutdown, health checks

## Architecture

JUNKyard runs exclusively on S2-MT (`192.168.20.1`) and receives logs from all other VMs.

| VM | IP | Method | Protocol |
|----|----|--------|----------|
| S1-APP | `10.0.10.1` | rsyslog | TCP |
| S1-DB | `10.0.20.1` | rsyslog | TCP |
| S1-FW | `172.16.0.2` (tunnel) | pfSense remote syslog | UDP |
| S2-FW | `192.168.20.254` | pfSense remote syslog | UDP |
| S2-JS | `192.168.10.10` | rsyslog | TCP |

### System Diagram

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│    S1-APP VM    │     │    S1-DB VM     │     │    S1-FW VM     │     │    S2-FW VM     │     │    S2-JS VM     │
│   (rsyslog)     │     │   (rsyslog)     │     │   (pfSense)     │     │   (pfSense)     │     │   (rsyslog)     │
└────────┬────────┘     └────────┬────────┘     └────────┬────────┘     └────────┬────────┘     └────────┬────────┘
         │                       │                       │                       │                       │
         │ TCP 5514              │ TCP 5514              │ UDP 5514              │ UDP 5514              │ TCP 5514
         └───────────────────────┼───────────────────────┼───────────────────────┼───────────────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │  S2-MT (192.168.20.1)   │
                    │  JUNKyard Server        │
                    │  - TCP+UDP Syslog :5514 │
                    │  - HTTP API       :8080 │
                    │  - SQLite Database      │
                    │  - Web UI Dashboard     │
                    └────────────┬────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │   Operators/Users       │
                    │   - Web UI              │
                    │   - REST API            │
                    │   - CLI Tool            │
                    └─────────────────────────┘
```

---

## Installation Guide

### 1. Clone and Build

```bash
git clone https://github.com/mr-andrej/junkyard.git
cd junkyard

go build -o bin/junkyard-server ./cmd/junkyard-server
go build -o bin/junk ./cmd/junkyard-cli
```

### 2. Deploy on S2-MT

```bash
chmod +x scripts/deploy-s2-mt.sh
sudo bash scripts/deploy-s2-mt.sh
```

The script will automatically:
- Create the `junkyard` user and directories
- Install the binary to `/usr/local/bin`
- Set up the systemd service
- Configure firewall rules (UFW)
- Start the service

### 3. Verify It's Running

```bash
sudo systemctl status junkyard.service
junk health
journalctl -u junkyard.service -f
```

A healthy `junk health` response:

```json
{
  "status": "ok",
  "version": "1.0.0",
  "hostname": "s2-mt",
  "uptime_seconds": 3600,
  "database": {
    "path": "/var/lib/junkyard/logs.db",
    "size_mb": 0.1,
    "total_logs": 0
  }
}
```

### 4. Configure Source VMs

#### Linux VMs (S1-APP, S1-DB, S2-JS)

```bash
sudo tee /etc/rsyslog.d/99-junkyard.conf > /dev/null <<'EOF'
*.* @@192.168.20.1:5514
EOF

sudo systemctl restart rsyslog
sudo netstat -tn | grep 5514
# Should show ESTABLISHED
```

> The double `@@` means TCP. A single `@` would use UDP.

> If `netstat` is not found: `sudo apt-get install net-tools`

#### pfSense VMs (S1-FW, S2-FW)

**Status → System Logs → Settings → Remote Logging**

- **Enable Remote Logging**: checked
- **Remote log servers**: `192.168.20.1:5514`
- **Remote Syslog Contents**: `Everything`
- Click **Save**

> pfSense sends syslog over UDP even when a custom port is specified. JUNKyard handles this natively.

### 5. Test Log Ingestion

```bash
# From any Linux source VM:
logger -t test-vm -p user.info "Test log from $(hostname)"

# Then on S2-MT:
junk logs --limit 5
```

### 6. Access the Web UI

```
http://192.168.20.1:8080
```

Access requires VPN. No direct WAN exposure.

---

## Required Firewall Rules

### Site 1 pfSense (`5.196.50.51`)

| Interface | Protocol | Source | Destination | Port | Purpose |
|-----------|----------|--------|-------------|------|---------|
| OPT1 | IPv4 TCP | `10.0.10.0/24` | `192.168.20.0/24` | `*` | S1-APP syslog |
| OPT2 | IPv4 TCP | `10.0.20.0/24` | `192.168.20.1/32` | `5514` | S1-DB syslog |
| OpenVPN | IPv4 TCP/UDP | `172.16.0.0/30` | `192.168.20.1/32` | `5514` | S1-FW syslog (tunnel IP) |

### Site 2 pfSense (`5.196.45.7`)

| Interface | Protocol | Source | Destination | Port | Purpose |
|-----------|----------|--------|-------------|------|---------|
| OPT1 | IPv4 TCP | `192.168.10.0/24` | `192.168.20.1/32` | `5514` | S2-JS syslog |
| OpenVPN | IPv4 TCP/UDP | `172.16.0.0/30` | `192.168.20.1/32` | `5514` | S1-FW tunnel replies |

---

## Usage

### CLI

```
🗑️  JUNKyard CLI v1.0.0
Usage: junk <command> [options]

Commands:
  logs    - Display recent logs with filtering
  stream  - Stream logs in real-time (polls every 5s)
  stats   - Show log statistics and breakdown
  search  - Full-text search across logs
  graph   - Display log trends as ASCII graph
  health  - Check server health

Global Options:
  --host      Filter by hostname
  --source    Filter by source (syslog, http, etc.)
  --level     Filter by level (debug, info, warning, error)
  --limit     Limit number of results (default: 100)
  --hours     Time range in hours (default: 24)
  --server    API server address (default: http://localhost:8080)

Examples:
  junk logs                              # Last 100 logs
  junk logs --host s1-app --level error  # Errors on S1-APP
  junk search "database"                 # Full-text search
  junk graph --hours 24                  # Last 24 hours
  junk stream                            # Real-time streaming
  junk stats                             # Statistics
  junk health                            # Server status
```

### Web UI

Navigate to `http://192.168.20.1:8080`.

Features:
- Real-time statistics (total logs, errors, warnings, database size)
- Full-text search across log messages
- Filter by host, source, or log level
- Auto-refresh every 5 seconds
- Pagination (50–1000 logs per view)
- Color-coded severity indicators

### REST API

```bash
# Recent logs
curl http://localhost:8080/api/logs?limit=100

# Filter by host and level
curl 'http://localhost:8080/api/logs?host=s1-app&level=error'

# Full-text search
curl 'http://localhost:8080/api/logs?search=connection%20timeout'

# Time range
curl 'http://localhost:8080/api/logs?start=2026-05-07T00:00:00Z&end=2026-05-07T23:59:59Z'

# Stats
curl http://localhost:8080/api/stats

# Health
curl http://localhost:8080/health

# Ingest single log
curl -X POST http://localhost:8080/api/ingest \
  -H "Content-Type: application/json" \
  -d '{"host":"s1-app","source":"http","level":"error","message":"test"}'

# Ingest batch
curl -X POST http://localhost:8080/api/ingest/batch \
  -H "Content-Type: application/json" \
  -d '[{"host":"s1-app","message":"log 1"},{"host":"s1-db","message":"log 2"}]'
```

---

## Configuration

### Command Line Flags

```
junkyard-server [flags]

  -http-addr string      HTTP server address (default ":8080")
  -syslog-addr string    Syslog server address (default ":5514")
  -db-path string        SQLite database path (default "./junkyard.db")
  -retention-days int    Log retention in days (default 14)
  -version               Print version and exit
```

### Environment Variables

```bash
export HTTP_ADDR=":8080"
export SYSLOG_ADDR=":5514"
export DB_PATH="/var/lib/junkyard/junkyard.db"
export RETENTION_DAYS="14"
```

### Systemd Service

Edit `/etc/systemd/system/junkyard.service` to customize resource limits:

```ini
[Service]
MemoryMax=500M
CPUQuota=100%
Restart=on-failure
RestartSec=10s
User=junkyard
Group=junkyard
```

> Use `MemoryMax` instead of `MemoryLimit` — `MemoryLimit` is deprecated in newer systemd versions.

---

## Troubleshooting

**SYN_SENT instead of ESTABLISHED** — firewall rule missing. Check rules above and test with:
```bash
nc -zv 192.168.20.1 5514
```

**pfSense logs not appearing** — verify UDP listener is active:
```bash
ss -ulnp | grep 5514
```

**Logs show IP instead of hostname** — update the `hostMap` in `internal/ingestion/syslog.go`.

**Service won't start:**
```bash
sudo journalctl -u junkyard.service -n 50 --no-pager
sudo systemctl daemon-reload
```

**Database corrupted:**
```bash
sudo systemctl stop junkyard.service
sudo cp /var/lib/junkyard/logs.db /var/lib/junkyard/logs.db.bak
sudo rm /var/lib/junkyard/logs.db
sudo systemctl start junkyard.service
```

---

## Alternative Installation Methods

### Docker
```bash
docker build -t junkyard:latest .
docker run -d \
  -p 8080:8080 \
  -p 5514:5514/tcp \
  -p 5514:5514/udp \
  -v junkyard-data:/data \
  junkyard:latest
```

### Manual (Ubuntu 24.04 LTS)

```bash
sudo apt-get update && sudo apt-get install -y golang-1.21 git sqlite3
make build-linux
sudo useradd -r -s /bin/false junkyard
sudo cp bin/junkyard-server /usr/local/bin/
sudo mkdir -p /var/lib/junkyard && sudo chown junkyard:junkyard /var/lib/junkyard
sudo cp systemd/junkyard.service /etc/systemd/system/
sudo systemctl daemon-reload && sudo systemctl enable junkyard && sudo systemctl start junkyard
```

---

## Performance Benchmarks

Tested on Ubuntu 24.04 LTS, 2GB RAM VM:

| Metric | Value |
|--------|-------|
| RAM Usage | ~95 MB |
| Database Size (1M logs) | ~450 MB |
| Query Time (full table) | < 500ms |
| Full-Text Search (10K phrase) | < 200ms |
| Ingest Rate (batch) | 5K logs/sec |
| Concurrent Syslog Connections | 100+ |

---

## Development

### Building

```bash
make build          # Build all binaries
make build-server   # Server only
make build-cli      # CLI only
make build-linux    # Cross-compile for Linux
make docker-build   # Build Docker image
```

### Project Structure

```
junkyard/
├── cmd/
│   ├── junkyard-server/     # Main server executable
│   └── junkyard-cli/        # CLI client tool
├── internal/
│   ├── api/handlers.go      # REST API endpoints
│   ├── ingestion/
│   │   ├── http.go          # HTTP ingestion handler
│   │   └── syslog.go        # Syslog server (RFC 3164, TCP+UDP)
│   ├── storage/
│   │   ├── db.go            # SQLite operations
│   │   └── queries.go       # Query layer with filtering
│   └── web/ui.go            # Web UI assets
├── pkg/models/log.go        # Shared data models
├── configs/
│   └── junkyard-remote.conf # rsyslog forwarding template
├── systemd/junkyard.service # Systemd service unit
├── scripts/
│   ├── deploy-s2-mt.sh      # Automated deployment script
│   ├── build.sh             # Build script
│   └── install.sh           # Ubuntu installation script
├── docs/
│   ├── API.md
│   ├── INSTALL.md
│   └── USAGE.md
├── Makefile
├── Dockerfile
├── go.mod
└── go.sum
```

### Running Tests

```bash
make test
go test -cover ./...
make lint && make fmt
```

---

## License

MIT — see LICENSE file.

## Support

- Issues: https://github.com/mr-andrej/junkyard/issues
- Docs: https://github.com/mr-andrej/junkyard/wiki

---

*Made with hopes and prayers for the CIA academic project* 🗑️
