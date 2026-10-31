// Copyright (C) 2026 Ajay Raikvar. SPDX-License-Identifier: AGPL-3.0-or-later. See ../LICENSE and ../NOTICE.
(() => {
  'use strict';
  const $ = (id) => document.getElementById(id);
  const fmt = (n) => Number.isFinite(Number(n)) ? Number(n).toFixed(1) : '—';
  async function load() {
    $('status').textContent = 'Loading host logs…';
    try {
      const response = await fetch('/api/admin/logs', { cache: 'no-store' });
      if (!response.ok) throw new Error(`HTTP ${response.status}: ${await response.text()}`);
      const data = await response.json();
      const items = Array.isArray(data.items) ? data.items.slice().reverse() : [];
      $('totalCount').textContent = String(data.count ?? items.length);
      $('latestTime').textContent = items[0]?.timestamp ? new Date(items[0].timestamp).toLocaleString() : 'No tests yet';
      const body = $('logRows'); body.replaceChildren();
      if (!items.length) {
        const tr = document.createElement('tr'); const td = document.createElement('td'); td.colSpan = 8; td.textContent = 'No completed tests have been recorded yet.'; tr.append(td); body.append(tr);
      }
      for (const item of items) {
        const tr = document.createElement('tr');
        const values = [item.timestamp ? new Date(item.timestamp).toLocaleString() : '—', item.clientIp || '—', `${fmt(item.downloadMbps)} Mbps`, `${fmt(item.uploadMbps)} Mbps`, `${fmt(item.latencyMs)} ms`, `${fmt(item.jitterMs)} ms`, `${fmt(item.durationSeconds)} s`, item.id || '—'];
        for (const value of values) { const td = document.createElement('td'); td.textContent = value; tr.append(td); }
        body.append(tr);
      }
      $('status').textContent = `Showing ${items.length} most recent test records. Full history remains in the host JSONL log.`;
    } catch (error) { $('status').textContent = `Could not load host logs: ${error.message}`; }
  }
  $('refreshButton').addEventListener('click', load);
  load();
})();
