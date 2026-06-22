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
    <script src="https://cdnjs.cloudflare.com/ajax/libs/Chart.js/4.4.0/chart.umd.min.js"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }

        body {
            font-family: 'Segoe UI', 'Roboto', 'Consolas', monospace;
            background: linear-gradient(135deg, #0d1117 0%, #161b22 100%);
            color: #c9d1d9;
            min-height: 100vh;
            padding: 20px;
        }

        .container { max-width: 1400px; margin: 0 auto; }

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

        .tagline { color: #8b949e; font-size: 14px; font-style: italic; margin-bottom: 20px; }

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

        .stat-card:hover { border-color: #58a6ff; background: rgba(88, 166, 255, 0.05); }

        .stat-value { font-size: 28px; font-weight: bold; color: #58a6ff; font-family: 'Courier New', monospace; }
        .stat-label { font-size: 11px; color: #8b949e; text-transform: uppercase; margin-top: 8px; letter-spacing: 1px; }

        /* Trend controls */
        .trend-controls {
            display: flex;
            align-items: center;
            gap: 10px;
            margin-bottom: 16px;
            flex-wrap: wrap;
        }

        .trend-label { color: #8b949e; font-size: 12px; text-transform: uppercase; letter-spacing: 1px; }

        .range-btn {
            background: transparent;
            border: 1px solid rgba(48, 54, 61, 0.8);
            color: #8b949e;
            padding: 6px 14px;
            border-radius: 6px;
            font-family: inherit;
            font-size: 12px;
            cursor: pointer;
            transition: all 0.2s ease;
        }

        .range-btn:hover { border-color: #58a6ff; color: #58a6ff; }
        .range-btn.active { background: rgba(88, 166, 255, 0.1); border-color: #58a6ff; color: #58a6ff; font-weight: bold; }

        /* Charts grid */
        .charts-grid {
            display: grid;
            grid-template-columns: repeat(2, 1fr);
            gap: 20px;
            margin-bottom: 20px;
        }

        .chart-card {
            background: rgba(22, 27, 34, 0.8);
            backdrop-filter: blur(10px);
            padding: 24px;
            border-radius: 8px;
            border: 1px solid rgba(48, 54, 61, 0.5);
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
        }

        .chart-card.full { grid-column: 1 / -1; }

        .chart-title {
            color: #58a6ff;
            font-size: 13px;
            font-weight: bold;
            text-transform: uppercase;
            letter-spacing: 1px;
            margin-bottom: 16px;
        }

        .chart-body { position: relative; height: 240px; }
        .chart-card.full .chart-body { height: 280px; }

        @media (max-width: 820px) {
            .charts-grid { grid-template-columns: 1fr; }
        }

        /* Controls */
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

        button { background: #238636; cursor: pointer; font-weight: bold; border: 1px solid #2ea043; padding: 10px 16px; }
        button:hover { background: #2ea043; border-color: #3fb950; }

        button.secondary { background: transparent; border: 1px solid rgba(48, 54, 61, 0.8); color: #8b949e; }
        button.secondary:hover { background: rgba(48, 54, 61, 0.5); border-color: #58a6ff; color: #58a6ff; }
        button.secondary.active { background: rgba(88, 166, 255, 0.1); border-color: #58a6ff; color: #58a6ff; }

        /* Quick queries bar */
        .quick-queries {
            background: rgba(22, 27, 34, 0.8);
            backdrop-filter: blur(10px);
            padding: 14px 20px;
            border-radius: 8px;
            margin-bottom: 20px;
            border: 1px solid rgba(48, 54, 61, 0.5);
            display: flex;
            gap: 10px;
            flex-wrap: wrap;
            align-items: center;
        }
        .quick-label { color: #8b949e; font-size: 12px; text-transform: uppercase; letter-spacing: 1px; margin-right: 4px; }
        .quick-btn {
            background: transparent;
            border: 1px solid rgba(48, 54, 61, 0.8);
            color: #8b949e;
            padding: 8px 14px;
            border-radius: 6px;
            font-family: inherit;
            font-size: 12px;
            cursor: pointer;
            transition: all 0.2s ease;
        }
        .quick-btn:hover { border-color: #a371f7; color: #a371f7; background: rgba(163, 113, 247, 0.08); }
        .quick-btn .desc { display: block; font-size: 10px; color: #6e7681; margin-top: 2px; }

        /* Logs */
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

        .log-entry:hover { background: rgba(48, 54, 61, 0.2); }
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

        .loading { text-align: center; padding: 60px 20px; color: #8b949e; font-style: italic; }

        .spinner {
            display: inline-block;
            width: 20px;
            height: 20px;
            border: 2px solid rgba(88, 166, 255, 0.3);
            border-top-color: #58a6ff;
            border-radius: 50%;
            animation: spin 0.8s linear infinite;
        }

        @keyframes spin { to { transform: rotate(360deg); } }

        .empty-state { text-align: center; padding: 60px 20px; color: #8b949e; }
        .empty-icon { font-size: 48px; margin-bottom: 10px; }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>🗑️ JUNKyard</h1>
            <p class="tagline">"Throw all your logs into the junkyard."</p>
            <div class="stats-grid" id="statsGrid"></div>
        </header>

        <div class="trend-controls">
            <span class="trend-label">Trends:</span>
            <button class="range-btn" data-hours="1" data-interval="5min">1h</button>
            <button class="range-btn" data-hours="6" data-interval="15min">6h</button>
            <button class="range-btn active" data-hours="24" data-interval="hour">24h</button>
            <button class="range-btn" data-hours="168" data-interval="hour">7d</button>
        </div>

        <div class="charts-grid">
            <div class="chart-card full">
                <div class="chart-title">📈 Log Volume Over Time</div>
                <div class="chart-body"><canvas id="chartVolume"></canvas></div>
            </div>
            <div class="chart-card">
                <div class="chart-title">📊 Severity Over Time</div>
                <div class="chart-body"><canvas id="chartStacked"></canvas></div>
            </div>
            <div class="chart-card">
                <div class="chart-title">⚠️ Errors by Host</div>
                <div class="chart-body"><canvas id="chartErrorsByHost"></canvas></div>
            </div>
            <div class="chart-card">
                <div class="chart-title">🔴 Logs by Severity</div>
                <div class="chart-body"><canvas id="chartByLevel"></canvas></div>
            </div>
            <div class="chart-card">
                <div class="chart-title">🖥️ Top Hosts by Volume</div>
                <div class="chart-body"><canvas id="chartByHost"></canvas></div>
            </div>
        </div>

        <div class="quick-queries">
            <span class="quick-label">Quick queries:</span>
            <button class="quick-btn" onclick="runQuickQuery('Accepted publickey')">
                SSH Logins
                <span class="desc">Successful SSH authentications</span>
            </button>
            <button class="quick-btn" onclick="runQuickQuery('Read error from remote host')">
                SSH Logouts
                <span class="desc">SSH sessions ended/disconnected</span>
            </button>
            <button class="quick-btn" onclick="runQuickQuery('Failed')">
                Failed Auth
                <span class="desc">Failed login attempts</span>
            </button>
        </div>

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
        let chartByLevel = null, chartByHost = null, chartVolume = null, chartStacked = null, chartErrorsByHost = null;
        let currentHours = 24, currentInterval = 'hour';

        const API_BASE = window.location.protocol + '//' + window.location.host;
        const VALID_HOST = /^s[12]-/i;            // only real infra hosts: s1-* / s2-*
        const LEVELS = ['error', 'warning', 'info', 'debug'];
        const LEVEL_COLORS = { error: '#f85149', warning: '#d29922', info: '#58a6ff', debug: '#8b949e' };

        const GRID = 'rgba(48,54,61,0.4)';
        const TICK = '#8b949e';
        function axis(stacked) {
            return {
                x: { stacked: !!stacked, ticks: { color: TICK, autoSkip: true, maxTicksLimit: 12, maxRotation: 0 }, grid: { color: GRID } },
                y: { stacked: !!stacked, beginAtZero: true, ticks: { color: TICK }, grid: { color: GRID } }
            };
        }
        function cap(s) { return s.charAt(0).toUpperCase() + s.slice(1); }

        function fmtLabel(iso, hours) {
            const d = new Date(iso);
            if (isNaN(d.getTime())) return iso;
            const hh = String(d.getHours()).padStart(2, '0');
            const mm = String(d.getMinutes()).padStart(2, '0');
            if (hours > 48) return (d.getMonth() + 1) + '/' + d.getDate() + ' ' + hh + ':' + mm;
            return hh + ':' + mm;
        }

        // ---------- Stats cards + severity doughnut + top hosts bar ----------
        async function loadStats() {
            try {
                const res = await fetch(API_BASE + '/api/stats');
                if (!res.ok) throw new Error('stats');
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

                // severity doughnut
                const byLevel = stats.by_level || {};
                const levelData = LEVELS.map(l => byLevel[l] || 0);
                const levelColors = LEVELS.map(l => LEVEL_COLORS[l]);
                if (chartByLevel) {
                    chartByLevel.data.datasets[0].data = levelData;
                    chartByLevel.update();
                } else {
                    chartByLevel = new Chart(document.getElementById('chartByLevel'), {
                        type: 'doughnut',
                        data: { labels: LEVELS.map(cap), datasets: [{ data: levelData, backgroundColor: levelColors.map(c => c + '33'), borderColor: levelColors, borderWidth: 2 }] },
                        options: { responsive: true, maintainAspectRatio: false, plugins: { legend: { labels: { color: TICK, font: { family: 'Consolas, monospace', size: 12 } } } } }
                    });
                }

                // top hosts (whitelisted)
                const byHost = stats.by_host || {};
                const hostLabels = Object.keys(byHost).filter(h => VALID_HOST.test(h)).sort((a, b) => byHost[b] - byHost[a]);
                const hostData = hostLabels.map(h => byHost[h]);
                if (chartByHost) {
                    chartByHost.data.labels = hostLabels;
                    chartByHost.data.datasets[0].data = hostData;
                    chartByHost.update();
                } else {
                    chartByHost = new Chart(document.getElementById('chartByHost'), {
                        type: 'bar',
                        data: { labels: hostLabels, datasets: [{ label: 'Log entries', data: hostData, backgroundColor: 'rgba(88, 166, 255, 0.2)', borderColor: '#58a6ff', borderWidth: 2, borderRadius: 4 }] },
                        options: { responsive: true, maintainAspectRatio: false, plugins: { legend: { display: false } }, scales: axis(false) }
                    });
                }
            } catch (e) { console.error('stats failed:', e); }
        }

        // ---------- Volume over time (line) ----------
        async function loadVolume() {
            try {
                const res = await fetch(API_BASE + '/api/timeseries?interval=' + currentInterval + '&hours=' + currentHours);
                if (!res.ok) throw new Error('timeseries');
                const json = await res.json();
                const points = json.data || [];
                const labels = points.map(p => fmtLabel(p.timestamp, currentHours));
                const data = points.map(p => p.count);
                if (chartVolume) {
                    chartVolume.data.labels = labels;
                    chartVolume.data.datasets[0].data = data;
                    chartVolume.update();
                } else {
                    chartVolume = new Chart(document.getElementById('chartVolume'), {
                        type: 'line',
                        data: { labels: labels, datasets: [{ label: 'Logs', data: data, borderColor: '#58a6ff', backgroundColor: 'rgba(88,166,255,0.15)', fill: true, tension: 0.3, pointRadius: 2, borderWidth: 2 }] },
                        options: { responsive: true, maintainAspectRatio: false, plugins: { legend: { display: false } }, scales: axis(false) }
                    });
                }
            } catch (e) { console.error('volume failed:', e); }
        }

        // ---------- Severity stacked over time ----------
        async function loadStacked() {
            try {
                const res = await fetch(API_BASE + '/api/timeseries/levels?interval=' + currentInterval + '&hours=' + currentHours);
                if (!res.ok) throw new Error('timeseries/levels');
                const json = await res.json();
                const rows = json.data || [];          // [{timestamp, level, count}]
                const labelSet = Array.from(new Set(rows.map(r => r.timestamp))).sort();
                const byKey = {};
                rows.forEach(r => { byKey[r.timestamp + '|' + r.level] = r.count; });
                const labels = labelSet.map(ts => fmtLabel(ts, currentHours));
                const datasets = LEVELS.map(lvl => ({
                    label: cap(lvl),
                    data: labelSet.map(ts => byKey[ts + '|' + lvl] || 0),
                    backgroundColor: LEVEL_COLORS[lvl] + 'b3',
                    borderColor: LEVEL_COLORS[lvl],
                    borderWidth: 1
                }));
                if (chartStacked) {
                    chartStacked.data.labels = labels;
                    chartStacked.data.datasets = datasets;
                    chartStacked.update();
                } else {
                    chartStacked = new Chart(document.getElementById('chartStacked'), {
                        type: 'bar',
                        data: { labels: labels, datasets: datasets },
                        options: { responsive: true, maintainAspectRatio: false, plugins: { legend: { labels: { color: TICK, font: { size: 11 } } } }, scales: axis(true) }
                    });
                }
            } catch (e) { console.error('stacked failed:', e); }
        }

        // ---------- Errors by host (horizontal bar, whitelisted) ----------
        async function loadErrorsByHost() {
            try {
                const res = await fetch(API_BASE + '/api/errors-by-host?limit=10');
                if (!res.ok) throw new Error('errors-by-host');
                const rows = await res.json();          // [{host, count}]
                const filtered = (rows || []).filter(r => VALID_HOST.test(r.host));
                const labels = filtered.map(r => r.host);
                const data = filtered.map(r => r.count);
                if (chartErrorsByHost) {
                    chartErrorsByHost.data.labels = labels;
                    chartErrorsByHost.data.datasets[0].data = data;
                    chartErrorsByHost.update();
                } else {
                    chartErrorsByHost = new Chart(document.getElementById('chartErrorsByHost'), {
                        type: 'bar',
                        data: { labels: labels, datasets: [{ label: 'Errors', data: data, backgroundColor: 'rgba(248,81,73,0.3)', borderColor: '#f85149', borderWidth: 2, borderRadius: 4 }] },
                        options: { indexAxis: 'y', responsive: true, maintainAspectRatio: false, plugins: { legend: { display: false } }, scales: { x: { beginAtZero: true, ticks: { color: TICK }, grid: { color: GRID } }, y: { ticks: { color: TICK }, grid: { display: false } } } }
                    });
                }
            } catch (e) { console.error('errors-by-host failed:', e); }
        }

        function loadTrends() { loadVolume(); loadStacked(); }

        // ---------- Sources dropdown ----------
        async function loadSources() {
            try {
                const res = await fetch(API_BASE + '/api/sources');
                if (!res.ok) throw new Error('sources');
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
            } catch (e) { console.error('sources failed:', e); }
        }

        // ---------- Log stream ----------
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
                if (!res.ok) throw new Error('logs');
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
                console.error('logs failed:', e);
            }
        }

        function runQuickQuery(query) {
            document.getElementById('searchInput').value = query;
            document.getElementById('hostFilter').value = '';
            document.getElementById('levelFilter').value = '';
            document.getElementById('sourceFilter').value = '';
            loadLogs();
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
                    loadLogs(); loadStats(); loadTrends(); loadErrorsByHost();
                }, 5000);
                btn.textContent = 'Auto-refresh (ON)';
                btn.classList.add('active');
                loadLogs();
            }
        }

        function escapeHtml(text) {
            if (!text) return '';
            const map = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#039;' };
            return String(text).replace(/[&<>"']/g, m => map[m]);
        }

        // Range selector
        document.querySelectorAll('.range-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                document.querySelectorAll('.range-btn').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                currentHours = parseInt(btn.dataset.hours, 10);
                currentInterval = btn.dataset.interval;
                loadTrends();
            });
        });

        // Initialize
        loadStats();
        loadSources();
        loadLogs();
        loadErrorsByHost();
        loadTrends();

        // Periodic background refresh
        setInterval(loadStats, 30000);
        setInterval(loadErrorsByHost, 30000);
        setInterval(loadTrends, 30000);

        document.getElementById('searchInput').addEventListener('keypress', e => {
            if (e.key === 'Enter') loadLogs();
        });
    </script>
</body>
</html>`
