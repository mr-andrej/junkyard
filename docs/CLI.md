# JUNKyard CLI Quick Reference

## Installation

```bash
# Copy CLI binary to standard location
sudo cp bin/junk /usr/local/bin/
sudo chmod +x /usr/local/bin/junk

# Or add to PATH
export PATH="./bin:$PATH"
```

## Configuration

### Server Connection

Default: `http://localhost:8080`

Override with environment variable:

```bash
export JUNKYARD_SERVER=http://192.168.20.5:8080
junk health
```

Or per-command:

```bash
JUNKYARD_SERVER=http://remote-junkyard:8080 junk logs
```

## Commands

### Health Check

```bash
junk health
```

**Output**: Server status, version, uptime, database info

**Use case**: Verify connectivity and server health

**Example**:
```bash
$ junk health
Status:     ok
Version:    1.0.0
Hostname:   junkrunner
Uptime:     1234s

Database:
  Path: /var/lib/junkyard/logs.db
  Size: 45.2 MB
  Total Logs: 125432
```

---

### Query Logs

```bash
junk logs [OPTIONS]
```

**Options**:
- `--host=VALUE`: Filter by hostname
- `--source=VALUE`: Filter by source (apache, mysql, nginx, etc.)
- `--level=VALUE`: Filter by level (error, warning, info, debug)
- `--limit=N`: Maximum logs to display (default: 100, max: 10000)

**Output**: Table with timestamp, host, source, level, message

**Use cases**:
- Monitor recent logs
- Debug specific host issues
- Identify error patterns

**Examples**:
```bash
# Last 50 logs
junk logs --limit 50

# All errors
junk logs --level error

# Errors from web-01
junk logs --host web-01 --level error

# Nginx logs only
junk logs --source nginx

# Recent database logs
junk logs --source mysql --limit 20
```

---

### Real-time Stream

```bash
junk stream
```

**Output**: New logs as they arrive, updated every 5 seconds

**Use case**: Monitor live log activity

**Example**:
```bash
$ junk stream
Monitoring for new logs (updating every 5s, press Ctrl+C to stop)...
[2026-06-01 10:35:37] web-01 (apache) [ERROR] Database connection failed
[2026-06-01 10:35:38] cache-01 (redis) [INFO] Cache eviction triggered
```

---

### Statistics

```bash
junk stats
```

**Output**: Breakdown by level, host, and source with percentages

**Use case**: Get high-level overview of log distribution

**Example**:
```bash
$ junk stats

Total Logs: 125432
Last Hour:  1234
Last 24 Hours: 45678
Database Size: 45.2 MB

Log Levels:
LEVEL        COUNT      PERCENTAGE
error        56789      45.2%
warning      34567      27.5%
info         28901      23.0%
debug         5175       4.1%

Top Hosts:
HOST            COUNT      PERCENTAGE
web-01          45678      36.3%
db-01           23456      18.7%
cache-01        18901      15.0%
web-02          15678      12.5%
firewall-01      9876       7.9%

Log Sources:
SOURCE          COUNT      PERCENTAGE
apache          45678      36.3%
mysql           28901      23.0%
redis           18901      15.0%
nginx           15678      12.5%
sshd             9876       7.9%
```

---

### Search

```bash
junk search TERM
```

**Output**: Logs matching search term

**Use case**: Find logs containing specific keywords

**Notes**: Uses LIKE pattern matching (or FTS5 if available)

**Examples**:
```bash
# Find connection errors
junk search "connection"

# Find authentication issues
junk search "auth"

# Find failed operations
junk search "failed"
```

---

### Graph

```bash
junk graph
```

**Output**: ASCII bar chart of log volume over last 24 hours (hourly)

**Use case**: Visualize log volume trends

**Example**:
```bash
$ junk graph
Log volume over the last 24 hours:

08:00 ████████ 234
09:00 ██████████████ 456
10:00 ████████████████████ 678
11:00 ██████ 189
...

Total: 5432 logs
```

---

## Common Workflows

### Investigate Recent Errors

```bash
# See recent errors
junk logs --level error --limit 20

# Search for specific error message
junk search "connection timeout"

# Get stats on error sources
junk stats | grep "Log Levels" -A 10
```

### Monitor Specific Host

```bash
# Recent logs from web-01
junk logs --host web-01 --limit 50

# Errors on web-01
junk logs --host web-01 --level error

# Real-time monitoring
junk stream | grep web-01
```

### Check Service Health

```bash
# Server health
junk health

# Recent log activity
junk stats

# Graph trending
junk graph
```

### Troubleshoot High Error Rate

```bash
# Get error breakdown
junk logs --level error --limit 100

# Find common error messages
junk search "error" | sort | uniq -c | sort -rn

# Check which hosts are affected
junk logs --level error | awk '{print $3}' | sort | uniq -c
```

---

## Performance Tips

1. **Use filters**: Reduce result set with `--host`, `--source`, `--level`
2. **Limit results**: Use `--limit` to avoid overwhelming output
3. **Stream for live data**: Use `junk stream` instead of repeated `junk logs`
4. **Search for patterns**: Use `junk search` instead of `junk logs` followed by grep

---

## Troubleshooting

### Cannot connect to server

```bash
# Check server is running
junk health

# Verify IP/port
export JUNKYARD_SERVER=http://192.168.20.5:8080
junk health

# Check firewall
sudo ufw status | grep 8080
```

### No results found

```bash
# Check logs exist
junk stats

# Verify filter is correct
junk logs  # without filters

# Check spelling
junk logs --host=hostname-exactly-as-stored
```

### Slow queries

```bash
# Reduce limit
junk logs --limit 50  # instead of --limit 10000

# Use filters
junk logs --host web-01  # instead of all logs

# Check database size
junk health | grep Size
```

---

## Environment Variables

```bash
# Server connection
export JUNKYARD_SERVER=http://192.168.20.5:8080

# Use in scripts
JUNKYARD_SERVER=http://remote-server:8080 junk logs > report.txt
```

---

## Integration Examples

### Log to File

```bash
junk logs --limit 1000 > logs_$(date +%Y%m%d_%H%M%S).txt
```

### Monitor and Alert

```bash
#!/bin/bash
# Alert if more than 10 errors in last hour
error_count=$(junk logs --level error | wc -l)
if [ $error_count -gt 10 ]; then
    echo "High error rate: $error_count errors" | mail -s "Alert" admin@example.com
fi
```

### Export for Analysis

```bash
# Export recent errors as CSV
junk logs --level error --limit 500 | tail -n +3 | awk '{print $1","$2","$3","$4}' > errors.csv
```

### Backup Statistics

```bash
# Daily stats snapshot
junk stats > stats_$(date +%Y%m%d).txt
```
