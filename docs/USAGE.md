# JUNKyard CLI Usage Guide

## Installation

```bash
# Install from binary
junk --version

# Or build from source
make build-cli
./bin/junk --version
```

---

## Commands

### junk logs

Display recent logs (default: last 100 logs, newest first).

**Usage:**
```bash
junk logs [options]
```

**Options:**
- `--host STRING` - Filter by hostname
- `--source STRING` - Filter by source (syslog, ssh, firewall, etc.)
- `--level STRING` - Filter by level (debug, info, warning, error)
- `--limit INT` - Number of logs to display (default: 100)
- `--offset INT` - Pagination offset (default: 0)
- `--server URL` - JUNKyard server URL (default: http://localhost:8080)

**Examples:**
```bash
# Recent 100 logs
junk logs

# Recent 20 error logs from s1-app
junk logs --host s1-app --level error --limit 20

# All firewall logs
junk logs --source firewall --limit 1000

# Paginate through results
junk logs --limit 50 --offset 0
junk logs --limit 50 --offset 50
```

**Output:**
```
ID     TIMESTAMP                HOST      LEVEL      MESSAGE
1234   2026-05-07 14:32:11     s1-app    ERROR      Database connection failed
1233   2026-05-07 14:31:45     s1-app    WARNING    High memory usage: 85%
1232   2026-05-07 14:30:22     s1-db     INFO       Backup completed
```

---

### junk stream

Stream logs in real-time (live tail).

**Usage:**
```bash
junk stream [options]
```

**Options:**
- `--host STRING` - Filter by hostname
- `--source STRING` - Filter by source
- `--level STRING` - Filter by level
- `--server URL` - JUNKyard server URL

**Examples:**
```bash
# Stream all logs
junk stream

# Stream only errors from s1-app
junk stream --host s1-app --level error

# Stream SSH logs from all servers
junk stream --source ssh
```

**Output:**
```
[14:35:22] s1-app    ERROR   Database connection failed
[14:35:23] s1-db     INFO    Query executed in 125ms
[14:35:24] s2-bastion WARNING High CPU usage detected
...
```

Press `Ctrl+C` to stop streaming.

---

### junk stats

Display aggregate statistics.

**Usage:**
```bash
junk stats [options]
```

**Options:**
- `--server URL` - JUNKyard server URL (default: http://localhost:8080)

**Example:**
```bash
junk stats
```

**Output:**
```
🗑️  JUNKyard Statistics
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Total Logs:     45,231
Database Size:  285.5 MB

By Level:
  ERROR     2,231 (4.9%)
  WARNING   8,000 (17.7%)
  INFO      30,000 (66.3%)
  DEBUG     5,000 (11.0%)

By Host:
  s1-app       20,000 (44.2%)
  s1-db        15,000 (33.1%)
  s2-bastion   10,231 (22.6%)

By Source:
  syslog        35,000 (77.4%)
  ssh           5,000 (11.0%)
  firewall      3,000 (6.6%)
  application   2,231 (4.9%)

Time Range:
  Oldest: 2026-04-23 10:15:00
  Newest: 2026-05-07 14:32:11

Recent Activity:
  Last Hour:  324 logs
  Last 24h:   8,543 logs
```

---

### junk search

Full-text search logs.

**Usage:**
```bash
junk search QUERY [options]
```

**Options:**
- `--limit INT` - Number of results (default: 50)
- `--host STRING` - Filter by hostname
- `--source STRING` - Filter by source
- `--level STRING` - Filter by level
- `--server URL` - JUNKyard server URL

**Examples:**
```bash
# Search for "database"
junk search "database"

# Search for "error" in s1-app logs
junk search "error" --host s1-app

# Search for "timeout" in error logs
junk search "timeout" --level error

# Get more results
junk search "connection" --limit 200
```

**Output:**
```
Found 42 matches for "database"

ID     TIMESTAMP                HOST      LEVEL   MESSAGE
1234   2026-05-07 14:32:11     s1-app    ERROR   Database connection failed
1220   2026-05-07 13:45:22     s1-app    ERROR   Database timeout after 30s
1198   2026-05-07 12:10:15     s1-db     INFO    Database backup started
...
```

---

### junk graph

Display log trends as ASCII graphs.

**Usage:**
```bash
junk graph [options]
```

**Options:**
- `--hours INT` - Historical period (default: 24, max: 336)
- `--interval STRING` - Grouping interval - minute, 5min, 15min, hour (default: hour)
- `--host STRING` - Filter by hostname
- `--source STRING` - Filter by source
- `--level STRING` - Filter by level
- `--server URL` - JUNKyard server URL

**Examples:**
```bash
# Last 24 hours by hour
junk graph --hours 24

# Last 7 days by day
junk graph --hours 168

# Last 6 hours by 15 minutes
junk graph --hours 6 --interval 15min

# Errors in last 24 hours
junk graph --level error
```

**Output:**
```
📊 JUNKyard Log Trend (Last 24 Hours)

    400 │     ╭─╮
    350 │     │ ╰─╮
    300 │   ╭─╯   ╰─╮
    250 │   │       ╰╮
    200 │ ╭─╯        ╰─
    150 │ │
    100 │ ╰─────╮
     50 │       ╰─────
      0 └─────────────────────────────
        00:00  06:00  12:00  18:00  24:00
        
Max: 412 logs/hour
Min: 15 logs/hour
Avg: 189 logs/hour
```

---

## Connection Options

### Specify Server URL

All commands support `--server` flag:

```bash
junk logs --server http://192.168.20.100:8080
junk stats --server http://logs.example.com
```

Or set environment variable:

```bash
export JUNKYARD_SERVER=http://192.168.20.100:8080
junk logs      # Uses http://192.168.20.100:8080
junk stats     # Uses http://192.168.20.100:8080
```

---

## Output Formatting

### Colors

- **RED** - Error logs
- **YELLOW** - Warning logs
- **GREEN** - Success/Info logs
- **CYAN** - Debug logs

### Quiet Mode

Some commands support `--quiet` or `-q` for minimal output:

```bash
junk search "error" -q   # Just print matching logs
```

### JSON Output

Export results as JSON:

```bash
junk logs --json > logs.json
junk stats --json > stats.json
```

---

## Piping and Integration

### Count matching logs

```bash
junk logs --level error --limit 1000 | grep "timeout" | wc -l
```

### Export to file

```bash
junk logs --host s1-app > s1-app-logs.txt
junk stats --json > backup.json
```

### Watch mode (continuous updates)

```bash
watch -n 5 "junk stats"   # Refresh every 5 seconds
```

---

## Tips & Tricks

### Grep Integration

```bash
# Find logs with specific pattern
junk logs | grep "ERROR"

# Count log types
junk logs --limit 1000 | cut -d' ' -f6 | sort | uniq -c
```

### Export for Analysis

```bash
# Export last 24 hours of errors
junk logs --level error --limit 10000 > errors-24h.txt

# Create CSV export
junk logs --json | jq -r '.results[] | [.timestamp, .host, .level, .message] | @csv' > logs.csv
```

### Monitor Specific Host

```bash
# Watch s1-app continuously
watch -n 2 'junk logs --host s1-app --limit 20'
```

### Follow Latest Activity

```bash
# Similar to "tail -f" but for JUNKyard
while true; do
  junk stream --level error
  sleep 1
done
```

---

## Troubleshooting

### Connection Refused

If you get "connection refused":

```bash
# Check server is running
curl http://localhost:8080/health

# Or check logs
sudo journalctl -u junkyard -n 50
```

### No Logs Found

```bash
# Check if logs exist
junk stats

# List all hosts
junk logs --limit 1 | grep HOST

# Check time range
junk logs --limit 1
```

### Slow Queries

For better performance:

```bash
# Use smaller --limit
junk logs --limit 100

# Filter by host/level
junk logs --host s1-app --level error

# Use --server flag to connect to specific server
junk logs --server http://s2-mt.local:8080
```

---

## Configuration File (Future)

```bash
# ~/.junkyard/config
JUNKYARD_SERVER=http://logs.example.com:8080
JUNKYARD_DEFAULT_LIMIT=50
JUNKYARD_COLORS=true
```

---

## Examples By Use Case

### Production Troubleshooting

```bash
# Check what happened in the last hour
junk logs --limit 200 | head -20

# Find errors
junk search "error" --hours 1

# Check specific host
junk logs --host s1-app --level error
```

### Performance Analysis

```bash
# Log volume trend
junk graph --hours 24

# Top hosts by volume
junk stats
```

### Security Review

```bash
# SSH attempts
junk logs --source ssh --limit 100

# Failed connections
junk search "failed" --limit 1000
```

### Audit Trail

```bash
# Export all logs for archival
junk logs --limit 100000 --json > audit-trail-$(date +%Y%m%d).json
```

---

For more info, see the [API documentation](API.md) or [main README](../README.md).
