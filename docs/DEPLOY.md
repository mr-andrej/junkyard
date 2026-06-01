# JUNKyard Phase 7: Systemd Deployment Guide

This guide covers deploying JUNKyard on the **S2-MT** (Monitoring VM) for centralized log aggregation across the infrastructure.

## Architecture Overview

```
SOURCE VMs (S1-APP, S1-DB, S1-FW, S2-FW, S2-JS)
         ↓ rsyslog (TCP :5514)
      [S2-MT]
    ┌──────────┐
    │ JUNKyard │
    ├──────────┤
    │ HTTP:8080│ ← CLI and Web UI clients
    │ Syslog:5514│ ← Log ingestion
    │ SQLite DB│ ← Log storage
    └──────────┘
```

## Prerequisites

- **VM**: S2-MT (Monitoring VM, 192.168.20.X)
- **OS**: Ubuntu 24.04 LTS
- **RAM**: 2GB minimum (JUNKyard uses <100MB)
- **Storage**: 10GB+ for logs
- **Root access**: Required for installation

## Installation Steps

### 1. Build the Binary

On your build machine (or S2-MT if Go is installed):

```bash
cd /path/to/junkyard
go build -o bin/junkyard-server ./cmd/junkyard-server
go build -o bin/junk ./cmd/junkyard-cli
```

### 2. Automated Deployment (Recommended)

Transfer the built binary and deployment script to S2-MT:

```bash
# From build machine
scp -r bin/ deploy-user@s2-mt:/tmp/junkyard/
scp scripts/deploy-s2-mt.sh deploy-user@s2-mt:/tmp/
scp systemd/junkyard.service deploy-user@s2-mt:/tmp/

# On S2-MT
ssh deploy-user@s2-mt
sudo bash /tmp/deploy-s2-mt.sh
```

### 3. Manual Deployment

If you prefer manual steps:

```bash
# Connect to S2-MT as root or sudo user
ssh root@s2-mt

# 1. Create junkyard user
useradd -r -s /usr/sbin/nologin -d /var/lib/junkyard junkyard
mkdir -p /var/lib/junkyard
chown junkyard:junkyard /var/lib/junkyard
chmod 750 /var/lib/junkyard

# 2. Copy binary
sudo cp bin/junkyard-server /usr/local/bin/
chmod 755 /usr/local/bin/junkyard-server

# 3. Copy systemd service
sudo cp systemd/junkyard.service /etc/systemd/system/
sudo systemctl daemon-reload

# 4. Enable and start
sudo systemctl enable junkyard.service
sudo systemctl start junkyard.service

# 5. Verify
sudo systemctl status junkyard
```

## Configuration

### Systemd Service File

Location: `/etc/systemd/system/junkyard.service`

Key configuration options:

```ini
[Service]
ExecStart=/usr/local/bin/junkyard-server \
    --http-addr :8080 \
    --syslog-addr :5514 \
    --db-path /var/lib/junkyard/logs.db \
    --retention-days 30

# Resource limits
MemoryLimit=200M
CPUQuota=50%

# Auto-restart on failure
Restart=on-failure
RestartSec=10
```

### Environment Variables

Create `/etc/default/junkyard` for environment-specific settings:

```bash
# Database retention (days)
RETENTION_DAYS=30

# Memory limit
MEMORY_LIMIT=200M

# Enable debug logging
DEBUG=false
```

Then update systemd service to source it:

```ini
EnvironmentFile=/etc/default/junkyard
```

## Firewall Configuration

### UFW Rules (for S2-MT)

```bash
# Allow HTTP from monitoring VLAN
ufw allow from 192.168.20.0/24 to any port 8080

# Allow HTTP from VPN admin network
ufw allow from 192.168.30.0/24 to any port 8080

# Allow syslog from all internal VMs (192.168.0.0/16)
ufw allow from 192.168.0.0/16 to any port 5514
```

### UFW Rules (for Source VMs)

On each VM sending logs (S1-APP, S1-DB, etc.):

```bash
# Allow rsyslog to S2-MT
ufw allow out 192.168.20.X/32 port 5514 proto tcp
```

## Post-Installation Verification

### Check Service Status

```bash
# View status
systemctl status junkyard

# View logs
journalctl -u junkyard -f

# Check uptime
systemctl show junkyard -p ExecMainStartTimestamp
```

### Health Check from CLI

```bash
# On S2-MT or any authorized host
junk health

# Expected output:
# Status:     ok
# Version:    1.0.0
# Hostname:   junkrunner
# Uptime:     XXs
# Database:
#   Path: /var/lib/junkyard/logs.db
#   Size: X.X MB
#   Total Logs: XXX
```

### Test Log Ingestion

```bash
# Send a test log via HTTP API
curl -X POST http://192.168.20.X:8080/api/ingest \
  -H "Content-Type: application/json" \
  -d '{
    "host": "test-host",
    "source": "test",
    "level": "info",
    "message": "Test log from deployment"
  }'

# Query via CLI
junk logs --host test-host
```

## Configure Remote Syslog (Source VMs)

On each VM that should forward logs to S2-MT:

### 1. Create Forwarding Configuration

```bash
sudo tee /etc/rsyslog.d/99-junkyard.conf > /dev/null <<EOF
# Forward all logs to JUNKyard on S2-MT
*.* @@192.168.20.X:5514
EOF
```

Replace `192.168.20.X` with actual S2-MT IP address.

### 2. Restart Rsyslog

```bash
sudo systemctl restart rsyslog
sudo systemctl status rsyslog
```

### 3. Verify Forwarding

```bash
# Generate a test log
logger -t test-vm -p user.info "Forwarding test from $(hostname)"

# On S2-MT, check if log appears
junk logs --host $(hostname)
```

## Maintenance

### View Recent Logs

```bash
# Last 20 logs
junk logs --limit 20

# Errors only
junk logs --level error

# From specific host
junk logs --host web-01

# Real-time stream
junk stream
```

### Check Statistics

```bash
junk stats
```

### Database Management

```bash
# Database file
ls -lh /var/lib/junkyard/logs.db*

# Database size
du -h /var/lib/junkyard/

# Manual backup
sudo cp /var/lib/junkyard/logs.db /backup/junkyard-$(date +%Y%m%d).db
```

### Restart Service

```bash
# Restart
sudo systemctl restart junkyard

# Reload configuration
sudo systemctl reload junkyard
```

## Troubleshooting

### Service Won't Start

```bash
# Check logs
journalctl -u junkyard -n 50 --no-pager

# Verify binary exists
ls -la /usr/local/bin/junkyard-server

# Check permissions
ls -la /var/lib/junkyard/
```

### High Memory Usage

Check and adjust limits:

```bash
# Current memory usage
systemctl status junkyard | grep Memory

# Update systemd service
sudo systemctl edit junkyard
# Change MemoryLimit to desired value
# Restart: systemctl restart junkyard
```

### Database File Corruption

```bash
# Backup and reinitialize
sudo systemctl stop junkyard
sudo mv /var/lib/junkyard/logs.db /var/lib/junkyard/logs.db.backup
sudo systemctl start junkyard
# New database will be created on startup
```

### Port Already in Use

```bash
# Check what's using port 8080 or 5514
sudo lsof -i :8080
sudo lsof -i :5514

# Kill conflicting process (if safe)
sudo kill -9 <PID>

# Or change ports in systemd service:
sudo systemctl edit junkyard
# Update --http-addr and --syslog-addr
```

## Performance Tuning

### Database Optimization

Current configuration:
- **Mode**: WAL (Write-Ahead Logging) for concurrent reads
- **Sync**: NORMAL (balance between safety and speed)
- **Cache**: Shared
- **Connections**: Single connection pooling

For higher load:
1. Increase memory allocation: `MemoryLimit=500M`
2. Adjust CPU quota: `CPUQuota=100%`
3. Consider log rotation if storage is limited

### Retention Policy

Automatic cleanup runs daily:
- Default: 30 days
- Adjust via `--retention-days` flag
- Example: Keep 60 days of logs:
  ```bash
  # Edit systemd service
  --retention-days 60
  ```

## NetBox Integration

Register S2-MT and JUNKyard service in NetBox:

1. **VM**: S2-MT
   - IP: 192.168.20.X
   - Role: Monitoring
   - Services: JUNKyard (Port 8080, 5514)

2. **Service**: JUNKyard Log Aggregator
   - Type: Application
   - Protocol: HTTP/TCP
   - Ports: 8080, 5514

## Disaster Recovery

### Backup Strategy

```bash
# Daily backup
0 2 * * * /usr/local/bin/backup-junkyard.sh

# Create backup script
sudo tee /usr/local/bin/backup-junkyard.sh > /dev/null <<'EOF'
#!/bin/bash
BACKUP_DIR="/backup/junkyard"
mkdir -p "$BACKUP_DIR"
cp /var/lib/junkyard/logs.db "$BACKUP_DIR/logs-$(date +%Y%m%d-%H%M%S).db"
# Keep last 30 days of backups
find "$BACKUP_DIR" -mtime +30 -delete
EOF
sudo chmod +x /usr/local/bin/backup-junkyard.sh
```

### Restore from Backup

```bash
# Stop service
sudo systemctl stop junkyard

# Restore backup
sudo cp /backup/junkyard/logs-YYYYMMDD-HHMMSS.db /var/lib/junkyard/logs.db
sudo chown junkyard:junkyard /var/lib/junkyard/logs.db

# Start service
sudo systemctl start junkyard
```

## Next Steps

- [x] Install JUNKyard on S2-MT
- [x] Configure systemd service
- [x] Verify logs are being ingested
- [ ] Configure rsyslog on all source VMs
- [ ] Set up automated backups
- [ ] Monitor disk usage
- [ ] Plan for log rotation and archival

For Docker deployment, see [docs/DOCKER.md](../docs/DOCKER.md)
For advanced features, see [docs/API.md](../docs/API.md)
