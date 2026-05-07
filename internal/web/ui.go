package web

import "net/http"

// ServeUI serves the JUNKyard web UI
func ServeUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(HTML))
}

const HTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>🗑️ JUNKyard - Log Aggregator</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: 'Segoe UI', 'Roboto', 'Consolas', monospace;
            background: linear-gradient(135deg, #0d1117 0%, #161b22 100%);
            color: #c9d1d9;
            min-height: 100vh;
            padding: 20px;
        }

        .container {
            max-width: 1400px;
            margin: 0 auto;
        }

        header {
            background: rgba(22, 27, 34, 0.8);
            backdrop-filter: blur(10px);
            padding: 30px;
            border-radius: 8px;
            margin-bottom: 20px;
            border: 1px solid rgba(48, 54, 61, 0.5);
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
        }

        h1 {
            color: #58a6ff;
            font-size: 32px;
            margin-bottom: 5px;
            display: flex;
            align-items: center;
            gap: 10px;
        }

        .tagline {
            color: #8b949e;
            font-size: 14px;
            font-style: italic;
            margin-bottom: 20px;
        }

        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
            gap: 15px;
            margin-top: 20px;
        }

        .stat-card {
            background: rgba(13, 17, 23, 0.5);
            padding: 20px;
            border-radius: 6px;
            border: 1px solid rgba(48, 54, 61, 0.5);
            transition: all 0.3s ease;
        }

        .stat-card:hover {
            border-color: #58a6ff;
            background: rgba(88, 166, 255, 0.05);
        }

        .stat-value {
            font-size: 28px;
            font-weight: bold;
            color: #58a6ff;
            font-family: 'Courier New', monospace;
        }

        .stat-label {
            font-size: 11px;
            color: #8b949e;
            text-transform: uppercase;
            margin-top: 8px;
            letter-spacing: 1px;
        }

        .controls {
            background: rgba(22, 27, 34, 0.8);
            backdrop-filter: blur(10px);
            padding: 20px;
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
            padding: 10px 12px;
            border-radius: 6px;
            font-family: inherit;
            font-size: 13px;
            transition: all 0.2s ease;
        }

        input:focus, select:focus {
            outline: none;
            border-color: #58a6ff;
            background: rgba(13, 17, 23, 1);
            box-shadow: 0 0 0 3px rgba(88, 166, 255, 0.1);
        }

        button {
            background: #238636;
            cursor: pointer;
            font-weight: bold;
            border: 1px solid #2ea043;
            padding: 10px 16px;
        }

        button:hover {
            background: #2ea043;
            border-color: #3fb950;
        }

        button.secondary {
            background: transparent;
            border: 1px solid rgba(48, 54, 61, 0.8);
            color: #8b949e;
        }

        button.secondary:hover {
            background: rgba(48, 54, 61, 0.5);
            border-color: #58a6ff;
            color: #58a6ff;
        }

        button.secondary.active {
            background: rgba(88, 166, 255, 0.1);
            border-color: #58a6ff;
            color: #58a6ff;
        }

        .logs-container {
            background: rgba(13, 17, 23, 0.8);
            border: 1px solid rgba(48, 54, 61, 0.5);
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
        }

        .log-entry {
            padding: 12px 16px;
            border-bottom: 1px solid rgba(48, 54, 61, 0.3);
            font-size: 13px;
            line-height: 1.6;
            transition: all 0.2s ease;
            border-left: 4px solid transparent;
        }

        .log-entry:hover {
            background: rgba(48, 54, 61, 0.2);
        }

        .log-entry.error { border-left-color: #f85149; }
        .log-entry.warning { border-left-color: #d29922; }
        .log-entry.info { border-left-color: #58a6ff; }
        .log-entry.debug { border-left-color: #8b949e; }

        .log-timestamp { color: #8b949e; margin-right: 12px; font-size: 12px; }
        .log-host { color: #79c0ff; font-weight: bold; margin-right: 10px; }
        .log-source { color: #a371f7; margin-right: 10px; font-size: 11px; }

        .log-level {
            display: inline-block;
            padding: 3px 8px;
            border-radius: 4px;
            font-size: 11px;
            font-weight: bold;
            text-transform: uppercase;
            margin-right: 10px;
            min-width: 50px;
            text-align: center;
        }

        .log-level.error { background: rgba(248, 81, 73, 0.2); color: #f85149; }
        .log-level.warning { background: rgba(210, 153, 34, 0.2); color: #d29922; }
        .log-level.info { background: rgba(88, 166, 255, 0.2); color: #58a6ff; }
        .log-level.debug { background: rgba(139, 148, 158, 0.2); color: #8b949e; }

        .log-message { color: #c9d1d9; }

        .loading {
            text-align: center;
            padding: 60px 20px;
            color: #8b949e;
            font-style: italic;
        }

        .spinner {
            display: inline-block;
            width: 20px;
            height: 20px;
            border: 2px solid rgba(88, 166, 255, 0.3);
            border-top-color: #58a6ff;
            border-radius: 50%;
            animation: spin 0.8s linear infinite;
        }

        @keyframes spin {
            to { transform: rotate(360deg); }
        }

        .stats-container {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 20px;
            margin-bottom: 20px;
        }

        .chart-card {
            background: rgba(22, 27, 34, 0.8);
            padding: 20px;
            border-radius: 8px;
            border: 1px solid rgba(48, 54, 61, 0.5);
        }

        .chart-title {
            color: #58a6ff;
            font-weight: bold;
            margin-bottom: 15px;
            font-size: 14px;
        }

        .bar-chart {
            display: flex;
            flex-direction: column;
            gap: 10px;
        }

        .bar-item {
            display: flex;
            align-items: center;
            gap: 10px;
        }

        .bar-label {
            min-width: 100px;
            color: #8b949e;
            font-size: 12px;
        }

        .bar {
            flex: 1;
            height: 20px;
            background: linear-gradient(90deg, #58a6ff, #79c0ff);
            border-radius: 3px;
            opacity: 0.7;
        }

        .bar-value {
            min-width: 60px;
            text-align: right;
            color: #c9d1d9;
            font-size: 12px;
            font-weight: bold;
        }

        .empty-state {
            text-align: center;
            padding: 60px 20px;
            color: #8b949e;
        }

        .empty-icon {
            font-size: 48px;
            margin-bottom: 10px;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>🗑️ JUNKyard</h1>
            <p class="tagline">"Throw all your logs into the junkyard."</p>
            <div class="stats-grid" id="statsGrid"></div>
        </header>

        <div class="controls">
            <input type="text" id="searchInput" placeholder="🔍 Search logs..." style="flex: 1; min-width: 200px;">
            <input type="text" id="hostFilter" placeholder="Filter by host...">
            <select id="sourceFilter">
                <option value="">All Sources</option>
            </select>
            <select id="levelFilter">
                <option value="">All Levels</option>
                <option value="error">Error</option>
                <option value="warning">Warning</option>
                <option value="info">Info</option>
                <option value="debug">Debug</option>
            </select>
            <select id="limitSelect">
                <option value="50">50 logs</option>
                <option value="100" selected>100 logs</option>
                <option value="500">500 logs</option>
                <option value="1000">1000 logs</option>
            </select>
            <button onclick="loadLogs()">🔍 Search</button>
            <button class="secondary" id="autoRefreshBtn" onclick="toggleAutoRefresh()">⏸ Auto-refresh</button>
        </div>

        <div class="logs-container" id="logsContainer">
            <div class="loading"><span class="spinner"></span> Loading logs...</div>
        </div>
    </div>

    <script>
        let autoRefreshInterval = null;
        const API_BASE = window.location.protocol + '//' + window.location.host;

        async function loadStats() {
            try {
                const res = await fetch(API_BASE + '/api/stats');
                if (!res.ok) throw new Error('Failed to load stats');
                const stats = await res.json();

                const total = stats.total || 0;
                const errors = (stats.by_level && stats.by_level.error) || 0;
                const warnings = (stats.by_level && stats.by_level.warning) || 0;

                let html = '<div class="stat-card"><div class="stat-value">' + total.toLocaleString() + '</div><div class="stat-label">Total Logs</div></div>';
                html += '<div class="stat-card"><div class="stat-value" style="color: #f85149">' + errors.toLocaleString() + '</div><div class="stat-label">Errors</div></div>';
                html += '<div class="stat-card"><div class="stat-value" style="color: #d29922">' + warnings.toLocaleString() + '</div><div class="stat-label">Warnings</div></div>';
                html += '<div class="stat-card"><div class="stat-value">' + (stats.db_size_mb || 0).toFixed(1) + '</div><div class="stat-label">Database (MB)</div></div>';
                html += '<div class="stat-card"><div class="stat-value">' + ((stats.last_hour || 0).toLocaleString()) + '</div><div class="stat-label">Last Hour</div></div>';
                html += '<div class="stat-card"><div class="stat-value">' + ((stats.last_24h || 0).toLocaleString()) + '</div><div class="stat-label">Last 24h</div></div>';

                document.getElementById('statsGrid').innerHTML = html;
            } catch (e) {
                console.error('Failed to load stats:', e);
            }
        }

        async function loadSources() {
            try {
                const res = await fetch(API_BASE + '/api/sources');
                if (!res.ok) throw new Error('Failed to load sources');
                const sources = await res.json();

                const select = document.getElementById('sourceFilter');
                select.innerHTML = '<option value="">All Sources</option>';
                if (sources && Array.isArray(sources)) {
                    sources.forEach(source => {
                        const option = document.createElement('option');
                        option.value = source;
                        option.textContent = source;
                        select.appendChild(option);
                    });
                }
            } catch (e) {
                console.error('Failed to load sources:', e);
            }
        }

        async function loadLogs() {
            const search = document.getElementById('searchInput').value;
            const host = document.getElementById('hostFilter').value;
            const level = document.getElementById('levelFilter').value;
            const source = document.getElementById('sourceFilter').value;
            const limit = document.getElementById('limitSelect').value;

            const params = new URLSearchParams();
            if (search) params.append('search', search);
            if (host) params.append('host', host);
            if (level) params.append('level', level);
            if (source) params.append('source', source);
            params.append('limit', limit);

            const container = document.getElementById('logsContainer');
            container.innerHTML = '<div class="loading"><span class="spinner"></span> Loading logs...</div>';

            try {
                const res = await fetch(API_BASE + '/api/logs?' + params.toString());
                if (!res.ok) throw new Error('Failed to load logs');
                const logs = await res.json();

                if (!logs || logs.length === 0) {
                    container.innerHTML = '<div class="empty-state"><div class="empty-icon">No logs</div></div>';
                    return;
                }

                let logsHTML = '';
                logs.forEach(log => {
                    const timestamp = new Date(log.timestamp).toLocaleString('en-US', {
                        year: 'numeric', month: '2-digit', day: '2-digit',
                        hour: '2-digit', minute: '2-digit', second: '2-digit'
                    });
                    logsHTML += '<div class="log-entry ' + log.level + '">';
                    logsHTML += '<span class="log-timestamp">' + timestamp + '</span>';
                    logsHTML += '<span class="log-host">[' + escapeHtml(log.host) + ']</span>';
                    logsHTML += '<span class="log-source">(' + escapeHtml(log.source) + ')</span>';
                    logsHTML += '<span class="log-level ' + log.level + '">' + (log.level || 'info').toUpperCase() + '</span>';
                    logsHTML += '<span class="log-message">' + escapeHtml(log.message) + '</span>';
                    logsHTML += '</div>';
                });

                container.innerHTML = logsHTML;
            } catch (e) {
                container.innerHTML = '<div class="empty-state"><div class="empty-icon">Error</div></div>';
                console.error('Failed to load logs:', e);
            }
        }

        function toggleAutoRefresh() {
            const btn = document.getElementById('autoRefreshBtn');
            if (autoRefreshInterval) {
                clearInterval(autoRefreshInterval);
                autoRefreshInterval = null;
                btn.textContent = 'Auto-refresh';
                btn.classList.remove('active');
            } else {
                autoRefreshInterval = setInterval(() => {
                    loadLogs();
                    loadStats();
                }, 5000);
                btn.textContent = 'Auto-refresh (ON)';
                btn.classList.add('active');
                loadLogs();
            }
        }

        function escapeHtml(text) {
            if (!text) return '';
            const map = {'&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#039;'};
            return String(text).replace(/[&<>"']/g, m => map[m]);
        }

        // Initialize
        loadStats();
        loadSources();
        loadLogs();

        // Refresh stats every 30 seconds
        setInterval(loadStats, 30000);

        // Allow Enter key to search
        document.getElementById('searchInput').addEventListener('keypress', e => {
            if (e.key === 'Enter') loadLogs();
        });
    </script>
</body>
</html>`
