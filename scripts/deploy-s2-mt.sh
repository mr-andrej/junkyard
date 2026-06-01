#!/bin/bash
# JUNKyard Deployment Script for S2-MT (Monitoring VM)
# This script automates the installation and configuration of JUNKyard on Ubuntu 24.04 LTS

set -e  # Exit on error

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
JUNKYARD_USER="junkyard"
JUNKYARD_GROUP="junkyard"
JUNKYARD_HOME="/var/lib/junkyard"
JUNKYARD_BIN="/usr/local/bin/junkyard-server"
JUNKYARD_DB="${JUNKYARD_HOME}/logs.db"
HTTP_PORT="${HTTP_PORT:-8080}"
SYSLOG_PORT="${SYSLOG_PORT:-5514}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    log_error "This script must be run as root"
fi

log_info "Starting JUNKyard deployment on S2-MT..."

# 1. Create junkyard user and group
log_info "Creating junkyard user and group..."
if ! id -u "$JUNKYARD_USER" >/dev/null 2>&1; then
    groupadd -f "$JUNKYARD_GROUP"
    useradd -r -g "$JUNKYARD_GROUP" -d "$JUNKYARD_HOME" -s /usr/sbin/nologin "$JUNKYARD_USER"
    log_info "User and group created"
else
    log_warn "User $JUNKYARD_USER already exists"
fi

# 2. Create directories
log_info "Creating directories..."
mkdir -p "$JUNKYARD_HOME"
chown -R "$JUNKYARD_USER:$JUNKYARD_GROUP" "$JUNKYARD_HOME"
chmod 750 "$JUNKYARD_HOME"

# 3. Copy binary
log_info "Installing JUNKyard binary..."
if [ ! -f "bin/junkyard-server" ] && [ ! -f "bin/junkyard-server.exe" ]; then
    log_error "Binary not found. Please build with 'go build ./cmd/junkyard-server' first"
fi

# Find the binary (handle both Linux and cross-compiled)
BINARY=$(find . -name "junkyard-server" -o -name "junkyard-server.exe" | head -1)
if [ -z "$BINARY" ]; then
    log_error "Could not find junkyard-server binary"
fi

cp "$BINARY" "$JUNKYARD_BIN"
chmod 755 "$JUNKYARD_BIN"
log_info "Binary installed to $JUNKYARD_BIN"

# 4. Install systemd service file
log_info "Installing systemd service..."
if [ -f "systemd/junkyard.service" ]; then
    cp systemd/junkyard.service /etc/systemd/system/
    systemctl daemon-reload
    log_info "Systemd service installed"
else
    log_error "systemd/junkyard.service not found"
fi

# 5. Configure firewall (if ufw is available)
log_info "Configuring firewall..."
if command -v ufw >/dev/null 2>&1; then
    # Allow HTTP from internal networks
    ufw allow from 192.168.20.0/24 to any port "$HTTP_PORT" >/dev/null 2>&1 || true
    ufw allow from 192.168.30.0/24 to any port "$HTTP_PORT" >/dev/null 2>&1 || true  # VPN admin
    
    # Allow Syslog from all internal VMs
    ufw allow from 192.168.0.0/16 to any port "$SYSLOG_PORT" >/dev/null 2>&1 || true
    
    log_info "Firewall rules added"
else
    log_warn "UFW not found, skipping firewall configuration"
fi

# 6. Enable and start service
log_info "Enabling JUNKyard service..."
systemctl enable junkyard.service
log_info "Starting JUNKyard service..."
systemctl start junkyard.service

# 7. Verify service is running
sleep 2
if systemctl is-active --quiet junkyard; then
    log_info "✅ JUNKyard is running"
    systemctl status junkyard --no-pager | head -10
else
    log_error "Failed to start JUNKyard service. Check logs with: journalctl -u junkyard -n 50"
fi

# 8. Display next steps
log_info "Deployment complete!"
echo ""
echo -e "${GREEN}Next steps:${NC}"
echo "  - View logs:           journalctl -u junkyard -f"
echo "  - Check status:        systemctl status junkyard"
echo "  - Access Web UI:       http://$(hostname -I | awk '{print $1}'):${HTTP_PORT}"
echo "  - Query logs via CLI:  junk logs --limit 20"
echo ""
echo -e "${GREEN}Configuration:${NC}"
echo "  - Database:            $JUNKYARD_DB"
echo "  - HTTP port:           $HTTP_PORT"
echo "  - Syslog port:         $SYSLOG_PORT"
echo "  - Retention:           $RETENTION_DAYS days"
echo ""
echo -e "${GREEN}For remote log collection, configure rsyslog on source VMs:${NC}"
echo "  - Add to /etc/rsyslog.d/99-junkyard.conf:"
echo "    *.* @@$(hostname -I | awk '{print $1}'):${SYSLOG_PORT}"
echo "  - Then: systemctl restart rsyslog"
