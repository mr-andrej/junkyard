package web

import "net/http"

// ServeV2LogsUI serves the v2 OTLP log viewer, backed by /api/v1/logs.
func ServeV2LogsUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(V2LogsHTML))
}

const V2LogsHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>JUNKyard v2 - OTLP Logs</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }

        body {
            font-family: 'Segoe UI', 'Roboto', 'Consolas', monospace;
            background: linear-gradient(135deg, #0d1117 0%, #161b22 100%);
            color: #c9d1d9;
            min-height: 100vh;
            padding: 20px;
        }

        .container { max-width: 1200px; margin: 0 auto; }

        header {
            background: rgba(22, 27, 34, 0.8);
            padding: 24px 30px;
            border-radius: 8px;
            margin-bottom: 20px;
            border: 1px solid rgba(48, 54, 61, 0.5);
        }

        h1 { color: #58a6ff; font-size: 28px; margin-bottom: 4px; }
        h1 .v2 { color: #a371f7; }
        .tagline { color: #8b949e; font-size: 13px; font-style: italic; }
        .tagline code { color: #79c0ff; font-style: normal; }

        .controls {
            background: rgba(22, 27, 34, 0.8);
            padding: 16px 20px;
            border-radius: 8px;
            margin-bottom: 20px;
            border: 1px solid rgba(48, 54, 61, 0.5);
            display: flex;
            gap: 10px;
            flex-wrap: wrap;
            align-items: center;
        }

        input, select, button {
            background: rgba(13, 17, 23, 0.8);
            color: #c9d1d9;
            border: 1px solid rgba(48, 54, 61, 0.8);
            padding: 9px 12px;
            border-radius: 6px;
            font-family: inherit;
            font-size: 13px;
        }

        input:focus, select:focus { outline: none; border-color: #58a6ff; }

        button { background: #238636; cursor: pointer; font-weight: bold; border: 1px solid #2ea043; }
        button:hover { background: #2ea043; }
        button.secondary { background: transparent; color: #8b949e; }
        button.secondary.active { background: rgba(88, 166, 255, 0.1); border-color: #58a6ff; color: #58a6ff; }

        .status { color: #6e7681; font-size: 12px; margin-left: auto; }

        .logs-container {
            background: rgba(13, 17, 23, 0.8);
            border: 1px solid rgba(48, 54, 61, 0.5);
            border-radius: 8px;
            overflow: hidden;
        }

        .log-entry {
            padding: 10px 16px;
            border-bottom: 1px solid rgba(48, 54, 61, 0.3);
            font-size: 13px;
            line-height: 1.6;
            border-left: 4px solid transparent;
        }

        .log-entry.error { border-left-color: #f85149; }
        .log-entry.warn { border-left-color: #d29922; }
        .log-entry.info { border-left-color: #58a6ff; }
        .log-entry.debug { border-left-color: #8b949e; }

        .log-timestamp { color: #8b949e; margin-right: 12px; font-size: 12px; }
        .log-service { color: #79c0ff; font-weight: bold; margin-right: 10px; }

        .log-level {
            display: inline-block;
            padding: 2px 8px;
            border-radius: 4px;
            font-size: 11px;
            font-weight: bold;
            margin-right: 10px;
            min-width: 46px;
            text-align: center;
        }

        .log-level.error { background: rgba(248, 81, 73, 0.2); color: #f85149; }
        .log-level.warn { background: rgba(210, 153, 34, 0.2); color: #d29922; }
        .log-level.info { background: rgba(88, 166, 255, 0.2); color: #58a6ff; }
        .log-level.debug { background: rgba(139, 148, 158, 0.2); color: #8b949e; }

        .log-trace { color: #a371f7; font-size: 11px; margin-left: 10px; }
        .empty-state { text-align: center; padding: 60px 20px; color: #8b949e; font-style: italic; }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>JUNKyard <span class="v2">v2</span> / OTLP logs</h1>
            <p class="tagline">Ingested via OTLP on <code>:4317</code> (gRPC) / <code>:4318</code> (HTTP), queried from <code>/api/v1/logs</code>. <a href="/" style="color:#58a6ff">v1 UI</a></p>
        </header>

        <div class="controls">
            <input type="text" id="serviceFilter" placeholder="service name...">
            <select id="severityFilter">
                <option value="">All severities</option>
                <option value="ERROR">ERROR+</option>
                <option value="WARN">WARN+</option>
                <option value="INFO">INFO+</option>
                <option value="DEBUG">DEBUG+</option>
                <option value="TRACE">TRACE+</option>
            </select>
            <input type="text" id="searchInput" placeholder="search body..." style="flex: 1; min-width: 180px;">
            <select id="limitSelect">
                <option value="50">50</option>
                <option value="100" selected>100</option>
                <option value="500">500</option>
            </select>
            <button onclick="loadLogs()">Search</button>
            <button class="secondary active" id="liveBtn" onclick="toggleLive()">Live</button>
            <span class="status" id="statusLine"></span>
        </div>

        <div class="logs-container" id="logsContainer">
            <div class="empty-state">No OTLP logs yet. POST some to :4318/v1/logs.</div>
        </div>
    </div>

    <script>
        let liveInterval = null;

        function severityClass(e) {
            const t = (e.severity_text || '').toUpperCase();
            if (t === 'ERROR' || t === 'FATAL') return 'error';
            if (t === 'WARN' || t === 'WARNING') return 'warn';
            if (t === 'DEBUG' || t === 'TRACE') return 'debug';
            return 'info';
        }

        function escapeHtml(text) {
            if (!text) return '';
            const map = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#039;' };
            return String(text).replace(/[&<>"']/g, m => map[m]);
        }

        async function loadLogs() {
            const params = new URLSearchParams();
            const service = document.getElementById('serviceFilter').value;
            const severity = document.getElementById('severityFilter').value;
            const search = document.getElementById('searchInput').value;
            if (service) params.append('service', service);
            if (severity) params.append('severity', severity);
            if (search) params.append('q', search);
            params.append('limit', document.getElementById('limitSelect').value);

            try {
                const res = await fetch('/api/v1/logs?' + params.toString());
                if (!res.ok) throw new Error('query failed: ' + res.status);
                const logs = await res.json();
                const container = document.getElementById('logsContainer');
                if (!logs || logs.length === 0) {
                    container.innerHTML = '<div class="empty-state">No logs match. OTLP in, query out — that is the whole trick.</div>';
                } else {
                    container.innerHTML = logs.map(e => {
                        const cls = severityClass(e);
                        const ts = new Date(e.timestamp).toLocaleTimeString('en-GB');
                        const trace = e.trace_id ? '<span class="log-trace">trace ' + escapeHtml(e.trace_id.slice(0, 12)) + '</span>' : '';
                        return '<div class="log-entry ' + cls + '">' +
                            '<span class="log-timestamp">' + ts + '</span>' +
                            '<span class="log-level ' + cls + '">' + escapeHtml(e.severity_text || 'INFO') + '</span>' +
                            '<span class="log-service">' + escapeHtml(e.service || 'unknown') + '</span>' +
                            '<span>' + escapeHtml(e.body) + '</span>' + trace +
                            '</div>';
                    }).join('');
                }
                document.getElementById('statusLine').textContent =
                    (logs ? logs.length : 0) + ' logs · refreshed ' + new Date().toLocaleTimeString('en-GB');
            } catch (e) {
                document.getElementById('statusLine').textContent = e.message;
            }
        }

        function toggleLive() {
            const btn = document.getElementById('liveBtn');
            if (liveInterval) {
                clearInterval(liveInterval);
                liveInterval = null;
                btn.classList.remove('active');
            } else {
                liveInterval = setInterval(loadLogs, 2000);
                btn.classList.add('active');
            }
        }

        document.getElementById('searchInput').addEventListener('keypress', e => {
            if (e.key === 'Enter') loadLogs();
        });
        document.getElementById('serviceFilter').addEventListener('keypress', e => {
            if (e.key === 'Enter') loadLogs();
        });

        loadLogs();
        liveInterval = setInterval(loadLogs, 2000);
    </script>
</body>
</html>`
