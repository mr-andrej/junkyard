# JUNKyard Installation Guide

## Prerequisites

- Linux 2GB+ RAM VM (Ubuntu 20.04 LTS or later recommended)
- 500MB+ available disk space (for 14 days of logs)
- Root or sudo access

## Installation Methods

### Method 1: Using Pre-built Binaries (Recommended)

1. Download the latest binaries from GitHub Releases:

```bash
cd /tmp
wget https://github.com/mr-andrej/junkyard/releases/download/v1.0.0/junkyard-server
wget https://github.com/mr-andrej/junkyard/releases/download/v1.0.0/junk
```

2. Make executable and move to system PATH:

```bash
chmod +x junkyard-server junk
sudo mv junkyard-server /usr/local/bin/
sudo mv junk /usr/local/bin/
```

3. Verify installation:

```bash
junkyard-server --version
junk --version
```

### Method 2: Building from Source

1. Clone the repository:

```bash
git clone https://github.com/mr-andrej/junkyard.git
cd junkyard
```

2. Build (requires Docker for cross-compilation):

```bash
make build-linux
```

Or for local build:

```bash
make build
```

3. Install:

```bash
make install
```

### Method 3: Docker

1. Build Docker image:

```bash
docker build -t junkyard:1.0.0 .
```

2. Run container:

```bash
docker run -d \
  --name junkyard \
  -p 8080:8080 \
  -p 5514:5514 \
  -v junkyard-data:/var/lib/junkyard \
  junkyard:1.0.0
```

---

## System Setup

### Create JUNKyard User

```bash
sudo useradd -r -s /bin/false -m -d /var/lib/junkyard junkyard
sudo mkdir -p /var/lib/junkyard
sudo chown junkyard:junkyard /var/lib/junkyard
sudo chmod 700 /var/lib/junkyard
```

### Setup Systemd Service

1. Copy service file:

```bash
sudo cp systemd/junkyard.service /etc/systemd/system/
```

2. Reload systemd:

```bash
sudo systemctl daemon-reload
```

3. Enable and start service:

```bash
sudo systemctl enable junkyard
sudo systemctl start junkyard
```

4. Check status:

```bash
sudo systemctl status junkyard
```

### Configuration

Create `/etc/junkyard/config` (optional):

```bash
sudo mkdir -p /etc/junkyard
sudo tee /etc/junkyard/config > /dev/null <<EOF
JUNKYARD_HTTP_ADDR=:8080
JUNKYARD_SYSLOG_ADDR=:5514
JUNKYARD_DB_PATH=/var/lib/junkyard/logs.db
JUNKYARD_RETENTION_DAYS=14
JUNKYARD_LOG_LEVEL=info
EOF
```

### Firewall Configuration

Allow inbound traffic to JUNKyard:

```bash
# UFW (Ubuntu)
sudo ufw allow 8080/tcp comment "JUNKyard HTTP"
sudo ufw allow 5514/tcp comment "JUNKyard Syslog"

# firewalld (CentOS/RHEL)
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --permanent --add-port=5514/tcp
sudo firewall-cmd --reload
```

---

## Sending Logs to JUNKyard

### Method 1: HTTP API

```bash
curl -X POST http://localhost:8080/api/ingest \
  -H "Content-Type: application/json" \
  -d '{
    "host": "my-server",
    "level": "info",
    "message": "Application started successfully"
  }'
```

### Method 2: Rsyslog (Recommended for VM logs)

1. Create rsyslog config on each VM:

```bash
sudo tee /etc/rsyslog.d/50-junkyard.conf > /dev/null <<EOF
# Forward all logs to JUNKyard server
*.* @@192.168.20.X:5514
EOF
```

Replace `192.168.20.X` with your JUNKyard server IP.

2. Restart rsyslog:

```bash
sudo systemctl restart rsyslog
```

3. Verify logs are being received:

```bash
# On JUNKyard server
junk logs --limit 10
```

---

## Verification

1. Check server is running:

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "ok",
  "version": "1.0.0",
  "database": {
    "total_logs": 0
  }
}
```

2. Send a test log:

```bash
curl -X POST http://localhost:8080/api/ingest \
  -H "Content-Type: application/json" \
  -d '{
    "host": "test-server",
    "message": "Test log entry"
  }'
```

3. Query logs:

```bash
curl http://localhost:8080/api/logs
```

---

## Troubleshooting

### Port Already in Use

If port 8080 or 5514 is already in use:

```bash
# Find process using port
sudo lsof -i :8080
sudo lsof -i :5514

# Kill process
sudo kill -9 <PID>
```

Or use different ports:

```bash
junkyard-server --http-addr :8081 --syslog-addr :5515
```

### Database Locked

If you see "database is locked" errors:

```bash
# This usually means SQLite WAL file is corrupted
rm -f /var/lib/junkyard/logs.db-wal
rm -f /var/lib/junkyard/logs.db-shm
```

### Permission Denied

If you get permission errors:

```bash
# Fix permissions
sudo chown -R junkyard:junkyard /var/lib/junkyard
sudo chmod 700 /var/lib/junkyard
```

### Disk Space Issues

Check database size:

```bash
df -h /var/lib/junkyard
du -sh /var/lib/junkyard/logs.db
```

Manual cleanup:

```bash
# On server - delete logs older than 7 days
curl -X DELETE "http://localhost:8080/api/logs?days=7"
```

---

## Performance Tuning

### Increase Memory Limit (Systemd)

Edit `/etc/systemd/system/junkyard.service`:

```
MemoryLimit=400M
```

### Increase Query Limit

Set larger query limits in production (careful with memory):

```bash
# Max query size (in API)
junkyard-server --max-query-limit 50000
```

### Database Optimization

Periodically optimize database:

```bash
curl -X POST http://localhost:8080/api/optimize
```

---

## Uninstallation

```bash
# Stop service
sudo systemctl stop junkyard
sudo systemctl disable junkyard

# Remove service file
sudo rm /etc/systemd/system/junkyard.service
sudo systemctl daemon-reload

# Remove binaries
sudo rm /usr/local/bin/junkyard-server /usr/local/bin/junk

# Remove user and data
sudo userdel -r junkyard
```

---

## Next Steps

- See [docs/USAGE.md](USAGE.md) for CLI usage
- See [docs/API.md](API.md) for REST API documentation
- Check [README.md](../README.md) for overview
