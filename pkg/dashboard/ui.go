package dashboard

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>KubeGuardian Dashboard</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f7fa; color: #333; padding: 20px; }
        .container { max-width: 1400px; margin: 0 auto; }
        header { background: #1a1a2e; color: #fff; padding: 20px; border-radius: 8px; margin-bottom: 20px; }
        header h1 { font-size: 24px; margin-bottom: 8px; }
        header p { opacity: 0.8; font-size: 14px; }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; margin-bottom: 20px; }
        .card { background: #fff; border-radius: 8px; padding: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .card h2 { font-size: 18px; margin-bottom: 16px; color: #1a1a2e; border-bottom: 2px solid #e94560; padding-bottom: 8px; }
        .stat { font-size: 32px; font-weight: bold; color: #16213e; margin: 8px 0; }
        .stat-label { font-size: 14px; color: #666; text-transform: uppercase; letter-spacing: 1px; }
        table { width: 100%; border-collapse: collapse; margin-top: 16px; }
        th { text-align: left; padding: 12px; background: #f8f9fa; color: #1a1a2e; font-size: 13px; text-transform: uppercase; }
        td { padding: 12px; border-bottom: 1px solid #e9ecef; font-size: 14px; }
        tr:hover { background: #f8f9fa; }
        .badge { display: inline-block; padding: 4px 8px; border-radius: 4px; font-size: 12px; font-weight: bold; }
        .badge-critical { background: #e94560; color: #fff; }
        .badge-high { background: #f39c12; color: #fff; }
        .badge-medium { background: #f1c40f; color: #333; }
        .badge-low { background: #2ecc71; color: #fff; }
        .badge-success { background: #27ae60; color: #fff; }
        .badge-failed { background: #e74c3c; color: #fff; }
        .badge-blocked { background: #95a5a6; color: #fff; }
        .empty { text-align: center; color: #888; padding: 20px; font-style: italic; }
        .btn { background: #1a1a2e; color: #fff; border: none; padding: 8px 16px; border-radius: 4px; cursor: pointer; font-size: 14px; }
        .btn:hover { background: #16213e; }
        .section { margin-top: 30px; }
        .refresh { float: right; margin-top: -10px; }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>&#127952; KubeGuardian Dashboard</h1>
            <p>Kubernetes Cluster Monitoring &amp; Automated Remediation</p>
        </header>

        <div class="grid">
            <div class="card">
                <h2>Remediation Stats</h2>
                <div id="stats-container" class="empty">Loading...</div>
            </div>
            <div class="card">
                <h2>Resource Recommendations</h2>
                <div id="recommendations-container" class="empty">Loading...</div>
            </div>
        </div>

        <div class="section">
            <h2 style="margin-bottom: 16px;">Remediation History</h2>
            <button class="btn refresh" onclick="loadHistory()">&#8635; Refresh</button>
            <table id="history-table">
                <thead>
                    <tr>
                        <th>Action</th>
                        <th>Resource</th>
                        <th>Namespace</th>
                        <th>Status</th>
                        <th>Message</th>
                        <th>Time</th>
                    </tr>
                </thead>
                <tbody id="history-body"></tbody>
            </table>
        </div>

        <div class="section" style="margin-top: 30px;">
            <h2>API Endpoints</h2>
            <ul style="list-style: none; padding: 0;">
                <li style="margin: 8px 0;"><code>GET /api/v1/dashboard/summary</code> - Dashboard summary</li>
                <li style="margin: 8px 0;"><code>GET /api/v1/dashboard/history</code> - Remediation history</li>
                <li style="margin: 8px 0;"><code>GET /api/v1/dashboard/stats</code> - Remediation statistics</li>
                <li style="margin: 8px 0;"><code>GET /api/v1/dashboard/recommendations</code> - Resource recommendations</li>
                <li style="margin: 8px 0;"><code>GET /api/v1/dashboard/silence</code> - Silence rules</li>
                <li style="margin: 8px 0;"><code>GET /api/v1/dashboard/maintenance</code> - Maintenance windows</li>
                <li style="margin: 8px 0;"><code>GET /api/v1/remediation/history</code> - Detailed history (with filters)</li>
                <li style="margin: 8px 0;"><code>GET /api/v1/remediation/stats</code> - Detailed statistics</li>
            </ul>
        </div>
    </div>

    <script>
        async function loadStats() {
            try {
                const res = await fetch('/api/v1/dashboard/stats');
                const data = await res.json();
                const container = document.getElementById('stats-container');
                var total = data.total || 0;
                var successful = data.successful || 0;
                var failed = data.failed || 0;
                var rate = (data.successRate || 0).toFixed(1);
                container.innerHTML =
                    '<div class="stat">' + total + '</div>' +
                    '<div class="stat-label">Total Remediations</div>' +
                    '<div class="stat" style="color: #27ae60;">' + successful + '</div>' +
                    '<div class="stat-label">Successful</div>' +
                    '<div class="stat" style="color: #e74c3c;">' + failed + '</div>' +
                    '<div class="stat-label">Failed</div>' +
                    '<div class="stat" style="color: #f39c12;">' + rate + '%</div>' +
                    '<div class="stat-label">Success Rate</div>';
            } catch (e) {
                document.getElementById('stats-container').innerHTML = '<div class="error">Failed to load stats</div>';
            }
        }

        async function loadRecommendations() {
            try {
                const res = await fetch('/api/v1/dashboard/recommendations');
                const data = await res.json();
                const container = document.getElementById('recommendations-container');
                if (data.count === 0) {
                    container.innerHTML = '<div class="empty">No resource recommendations available</div>';
                    return;
                }
                var html = '<table><thead><tr><th>Pod</th><th>Container</th><th>Namespace</th><th>CPU Request</th><th>Mem Request</th><th>Confidence</th></tr></thead><tbody>';
                data.recommendations.forEach(function(r) {
                    var badge = r.confidence > 0.8 ? 'badge-success' : r.confidence > 0.5 ? 'badge-medium' : 'badge-low';
                    html += '<tr>' +
                        '<td>' + r.podName + '</td>' +
                        '<td>' + r.containerName + '</td>' +
                        '<td>' + r.namespace + '</td>' +
                        '<td>' + r.cpuRequest + '</td>' +
                        '<td>' + r.memoryRequest + '</td>' +
                        '<td><span class="badge ' + badge + '">' + (r.confidence * 100).toFixed(0) + '%</span></td>' +
                        '</tr>';
                });
                html += '</tbody></table>';
                container.innerHTML = html;
            } catch (e) {
                document.getElementById('recommendations-container').innerHTML = '<div class="error">Failed to load recommendations</div>';
            }
        }

        async function loadHistory() {
            try {
                const res = await fetch('/api/v1/dashboard/history?limit=20');
                const data = await res.json();
                const body = document.getElementById('history-body');
                body.innerHTML = '';
                if (data.count === 0) {
                    body.innerHTML = '<tr><td colspan="6" class="empty">No remediation history</td></tr>';
                    return;
                }
                data.history.forEach(function(h) {
                    var badge = h.success ? 'badge-success' : 'badge-failed';
                    var ns = h.namespace || '-';
                    var date = new Date(h.executedAt).toLocaleString();
                    body.innerHTML += '<tr>' +
                        '<td>' + h.action + '</td>' +
                        '<td>' + h.resource + '</td>' +
                        '<td>' + ns + '</td>' +
                        '<td><span class="badge ' + badge + '">' + (h.success ? 'Success' : 'Failed') + '</span></td>' +
                        '<td>' + h.message + '</td>' +
                        '<td>' + date + '</td>' +
                        '</tr>';
                });
            } catch (e) {
                document.getElementById('history-body').innerHTML = '<tr><td colspan="6" class="error">Failed to load history</td></tr>';
            }
        }

        loadStats();
        loadRecommendations();
        loadHistory();
        setInterval(loadStats, 30000);
        setInterval(loadHistory, 30000);
    </script>
</body>
</html>`;
