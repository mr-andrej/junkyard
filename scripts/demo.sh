#!/usr/bin/env bash
# Junkyard v2 live demo: seeds OTLP logs and queries them back.
# Runs fully on localhost — no network needed. Safe to run repeatedly.
set -euo pipefail
cd "$(dirname "$0")/.."

DEMO_DIR=".demo"
BIN="$DEMO_DIR/junkyard-server.exe"
HTTP="http://localhost:8080"
OTLP="http://localhost:4318"

mkdir -p "$DEMO_DIR"
rm -f "$DEMO_DIR"/*.db*

# Go may be installed but missing from a stale terminal's PATH
if ! command -v go >/dev/null 2>&1; then
  for candidate in "/c/Program Files/Go/bin" "$HOME/go/bin" "$HOME/scoop/shims"; do
    [ -x "$candidate/go" ] || [ -x "$candidate/go.exe" ] && export PATH="$PATH:$candidate" && break
  done
fi
command -v go >/dev/null 2>&1 || { echo "error: go not found on PATH"; exit 1; }

echo "==> Building junkyard-server"
go build -o "$BIN" ./cmd/junkyard-server

echo "==> Starting server (OTLP gRPC :4317, OTLP HTTP :4318, query API :8080)"
"$BIN" -db-path "$DEMO_DIR/v1.db" -v2-db-path "$DEMO_DIR/v2.db" &
SERVER_PID=$!
trap 'kill "$SERVER_PID" 2>/dev/null || true' EXIT

for i in $(seq 1 30); do
  curl -sf "$HTTP/api/v1/logs?limit=1" >/dev/null 2>&1 && break
  sleep 0.3
done

now_ns() { echo "$(date +%s)$1"; }

seed() {
  local service="$1" sev="$2" body="$3" ts="$4"
  cat <<EOF
{"resourceLogs":[{"resource":{"attributes":[{"key":"service.name","value":{"stringValue":"$service"}}]},"scopeLogs":[{"logRecords":[{"timeUnixNano":"$ts","severityText":"$sev","body":{"stringValue":"$body"}}]}]}]}
EOF
}

echo "==> Sending OTLP logs to :4318/v1/logs (this is the ingestion path)"
i=0
while IFS='|' read -r service sev body; do
  i=$((i + 1))
  curl -s -X POST "$OTLP/v1/logs" -H "Content-Type: application/json" \
    -d "$(seed "$service" "$sev" "$body" "$(now_ns "$(printf '%09d' "$i")")")"
  echo "  <- $service [$sev] $body"
done <<'LOGS'
checkout|INFO|order 8841 placed, total EUR 129.90
api-gateway|INFO|GET /api/products 200 in 23ms
payment|INFO|stripe charge ch_3Px2 succeeded
checkout|WARN|cart 5541 abandoned at payment step
api-gateway|ERROR|GET /api/orders 502 upstream timeout after 3000ms
payment|ERROR|stripe charge ch_3Px9 declined: insufficient_funds
worker|INFO|resized 41 product images in 812ms
api-gateway|INFO|POST /api/checkout 201 in 141ms
LOGS
echo

echo "==> Waiting for the batcher to flush (500 records or 1s, whichever first)"
sleep 1.5

query() {
  echo
  echo "\$ curl \"$HTTP/api/v1/logs?$1\""
  curl -s "$HTTP/api/v1/logs?$1"
  echo
}

echo
echo "==> Querying it back (this is /api/v1/logs on :8080)"
query "limit=8"
query "service=payment"
query "severity=ERROR"
query "q=timeout"

echo
echo "==> Demo over. Server still running — try your own:"
echo "    curl \"$HTTP/api/v1/logs?service=checkout&severity=WARN\""
echo "    Ctrl+C stops it (graceful shutdown drains the pipeline first)."
wait "$SERVER_PID"
