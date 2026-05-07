# JUNKyard REST API Documentation

**Base URL:** `http://localhost:8080/api`

---

## Authentication

Currently, JUNKyard does not require authentication. All endpoints are public.

---

## Endpoints

### Ingestion

#### POST /api/ingest

Ingest a single log entry.

**Request Body:**
```json
{
  "host": "s1-app",
  "level": "error",
  "message": "Database connection failed",
  "source": "application",
  "timestamp": "2026-05-07T14:32:11Z"
}
```

**Response (201 Created):**
```json
{
  "status": "ok",
  "message": "Log thrown into the junkyard"
}
```

**Required Fields:**
- `host` (string): Hostname or source identifier
- `message` (string): Log message

**Optional Fields:**
- `level` (string): Log level - debug, info, warning, error (default: info)
- `source` (string): Source/facility (default: http)
- `timestamp` (ISO8601): Log timestamp (default: now)

---

#### POST /api/ingest/batch

Ingest multiple log entries in a single request (max 1000 logs).

**Request Body:**
```json
[
  {
    "host": "s1-app",
    "level": "info",
    "message": "App started",
    "source": "application"
  },
  {
    "host": "s1-app",
    "level": "warning",
    "message": "High memory usage",
    "source": "monitoring"
  }
]
```

**Response (201 Created):**
```json
{
  "status": "ok",
  "inserted": 2,
  "total": 2,
  "message": "Dumped 2 logs into the junkyard"
}
```

---

### Querying

#### GET /api/logs

Query logs with filtering and pagination.

**Query Parameters:**
- `limit` (int): Maximum results to return (default: 100, max: 10000)
- `offset` (int): Results offset for pagination (default: 0)
- `host` (string): Filter by hostname
- `source` (string): Filter by source
- `level` (string): Filter by level (debug, info, warning, error)
- `search` (string): Full-text search in message field
- `start_time` (ISO8601): Start timestamp
- `end_time` (ISO8601): End timestamp
- `order` (string): Sort order - timestamp_asc or timestamp_desc (default: timestamp_desc)

**Example:**
```
GET /api/logs?host=s1-app&level=error&limit=50
```

**Response (200 OK):**
```json
{
  "total": 1234,
  "limit": 50,
  "offset": 0,
  "results": [
    {
      "id": 1234,
      "timestamp": "2026-05-07T14:32:11Z",
      "host": "s1-app",
      "source": "application",
      "level": "error",
      "message": "Database connection failed",
      "raw": "{...}"
    }
  ]
}
```

---

#### GET /api/logs/hosts

Get list of all unique hostnames.

**Response (200 OK):**
```json
{
  "hosts": ["s1-app", "s1-db", "s2-bastion"]
}
```

---

#### GET /api/logs/sources

Get list of all unique sources.

**Response (200 OK):**
```json
{
  "sources": ["syslog", "ssh", "firewall", "application", "http"]
}
```

---

### Statistics

#### GET /api/stats

Get aggregate log statistics.

**Response (200 OK):**
```json
{
  "total": 45231,
  "by_level": {
    "debug": 5000,
    "info": 30000,
    "warning": 8000,
    "error": 2231
  },
  "by_host": {
    "s1-app": 20000,
    "s1-db": 15000,
    "s2-bastion": 10231
  },
  "by_source": {
    "syslog": 35000,
    "ssh": 5000,
    "firewall": 3000,
    "application": 2231
  },
  "db_size_mb": 285.5,
  "oldest_log": "2026-04-23T10:15:00Z",
  "newest_log": "2026-05-07T14:32:11Z",
  "last_hour": 324,
  "last_24h": 8543
}
```

---

#### GET /api/timeseries

Get time-series data (log counts grouped by time interval).

**Query Parameters:**
- `interval` (string): Group by - minute, 5min, 15min, hour (default: hour)
- `hours` (int): Historical period in hours (default: 24, max: 336 = 14 days)

**Example:**
```
GET /api/timeseries?interval=hour&hours=24
```

**Response (200 OK):**
```json
{
  "interval": "hour",
  "data": [
    {
      "timestamp": "2026-05-06T14:00:00Z",
      "count": 324
    },
    {
      "timestamp": "2026-05-06T15:00:00Z",
      "count": 356
    }
  ]
}
```

---

### Health & Status

#### GET /health

Health check endpoint.

**Response (200 OK):**
```json
{
  "status": "ok",
  "version": "1.0.0",
  "uptime_seconds": 3600,
  "database": {
    "path": "/var/lib/junkyard/logs.db",
    "size_mb": 285.5,
    "total_logs": 45231,
    "last_updated": "2026-05-07T14:32:11Z"
  }
}
```

---

## Error Responses

All errors follow this format:

```json
{
  "error": "Error message",
  "code": "ERROR_CODE"
}
```

**Common Error Codes:**
- `INVALID_JSON` - Request body is invalid JSON
- `MISSING_FIELD` - Required field is missing
- `BATCH_TOO_LARGE` - Batch size exceeds 1000 logs
- `DB_ERROR` - Database operation failed
- `INVALID_QUERY` - Query parameters are invalid

**HTTP Status Codes:**
- `200 OK` - Successful read operation
- `201 Created` - Successful write operation
- `400 Bad Request` - Invalid request
- `405 Method Not Allowed` - Wrong HTTP method
- `500 Internal Server Error` - Server error

---

## Rate Limiting

Currently, JUNKyard does not implement rate limiting. This may be added in future versions.

---

## Batch Operations

For bulk ingestion, use the `/api/ingest/batch` endpoint:

```bash
curl -X POST http://localhost:8080/api/ingest/batch \
  -H "Content-Type: application/json" \
  -d @logs.json
```

Maximum batch size is 1000 logs per request.

---

## Examples

### Single Log Ingestion (curl)

```bash
curl -X POST http://localhost:8080/api/ingest \
  -H "Content-Type: application/json" \
  -d '{
    "host": "s1-app",
    "level": "error",
    "message": "Connection timeout",
    "source": "application"
  }'
```

### Query Recent Errors

```bash
curl "http://localhost:8080/api/logs?level=error&limit=20"
```

### Search Logs

```bash
curl "http://localhost:8080/api/logs?search=database&limit=50"
```

### Get Statistics

```bash
curl "http://localhost:8080/api/stats"
```

---

## Syslog Ingestion

For Syslog ingestion, configure rsyslog on your servers:

```conf
# /etc/rsyslog.d/50-junkyard.conf
*.* @@192.168.20.X:5514
```

Then restart rsyslog:
```bash
sudo systemctl restart rsyslog
```

JUNKyard Syslog server runs on port 5514 by default (TCP).
