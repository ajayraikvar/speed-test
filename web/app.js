// Copyright (C) 2026 Ajay Raikvar. SPDX-License-Identifier: AGPL-3.0-or-later. See ../LICENSE and ../NOTICE.
(() => {
  'use strict';

  const $ = (id) => document.getElementById(id);
  const state = { running: false, cancel: false, chart: [], maxSpeed: 1, lastResults: null, stopCurrent: null, hostInfo: null };
  const canvas = $('speedChart');
  const ctx = canvas.getContext('2d');
  const downloadBlobSize = 64 * 1024 * 1024;
  let toastTimer = null;

  function showToast(message) {
    const node = $('toast');
    node.textContent = message;
    node.classList.add('show');
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => node.classList.remove('show'), 3800);
  }

  function formatSpeed(mbps) {
    if (!Number.isFinite(mbps)) return '—';
    return mbps >= 1000 ? (mbps / 1000).toFixed(2) : mbps.toFixed(1);
  }

  function speedUnit(mbps) { return mbps >= 1000 ? 'Gbps' : 'Mbps'; }
  function formatMBps(mbps) { return Number.isFinite(mbps) ? `${(mbps / 8).toFixed(2)} MB/s` : '—'; }

  function setSpeed(value, direction) {
    $('liveSpeed').textContent = formatSpeed(value);
    $('liveUnit').textContent = speedUnit(value);
    $('liveDirection').textContent = direction;
    $('gaugeRing').style.setProperty('--angle', `${Math.min(280, Math.max(0, (value / Math.max(state.maxSpeed, value, 1)) * 280))}deg`);
  }

  function drawChart() {
    const rect = canvas.getBoundingClientRect();
    const dpr = window.devicePixelRatio || 1;
    const width = Math.max(280, rect.width);
    const height = 92;
    canvas.width = Math.floor(width * dpr);
    canvas.height = Math.floor(height * dpr);
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, width, height);
    ctx.strokeStyle = 'rgba(143,162,189,.13)';
    ctx.lineWidth = 1;
    for (let i = 1; i <= 3; i++) {
      const y = (height - 8) * i / 4;
      ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(width, y); ctx.stroke();
    }
    if (state.chart.length < 2) return;
    const max = Math.max(1, ...state.chart.map(p => p.value)) * 1.12;
    const points = state.chart.slice(-90);
    const step = width / Math.max(1, points.length - 1);
    ctx.beginPath();
    points.forEach((p, i) => {
      const x = i * step;
      const y = height - 7 - (p.value / max) * (height - 17);
      if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
    });
    ctx.lineWidth = 2;
    ctx.strokeStyle = '#45dfbd';
    ctx.shadowColor = 'rgba(57,223,189,.38)';
    ctx.shadowBlur = 8;
    ctx.stroke();
    ctx.shadowBlur = 0;
    const last = points[points.length - 1];
    const x = (points.length - 1) * step;
    const y = height - 7 - (last.value / max) * (height - 17);
    ctx.fillStyle = '#6be8cb'; ctx.beginPath(); ctx.arc(x, y, 3, 0, Math.PI * 2); ctx.fill();
  }

  function addSample(value) {
    state.chart.push({ value, time: Date.now() });
    if (state.chart.length > 120) state.chart.shift();
    state.maxSpeed = Math.max(state.maxSpeed, value, 1);
    drawChart();
    setSpeed(value, $('liveDirection').textContent);
  }

  async function loadHostInfo() {
    try {
      const response = await fetch('/api/info', { cache: 'no-store' });
      if (!response.ok) throw new Error(`Server returned ${response.status}`);
      const info = await response.json();
      state.hostInfo = info;
      $('hostName').textContent = info.hostname || 'Host PC';
      $('hostAddress').textContent = `This server · Port ${info.port} · ${info.addresses.join(', ') || 'Host address unavailable'}`;
      $('appVersion').textContent = `v${info.version}`;
      $('connectionStatus').textContent = 'Connected to host';
      return true;
    } catch (error) {
      $('connectionStatus').textContent = 'Host connection issue';
      $('hostName').textContent = 'Could not reach the host server';
      $('hostAddress').textContent = error.message;
      $('hostName').style.color = '#f1b75a';
      return false;
    }
  }

  async function measureLatency() {
    const values = [];
    const count = 10;
    for (let i = 0; i < count; i++) {
      if (state.cancel) break;
      const started = performance.now();
      await fetch(`/api/ping?n=${i}&t=${Date.now()}`, { cache: 'no-store' });
      values.push(performance.now() - started);
      $('progressFill').style.width = `${(i + 1) / count * 8}%`;
      $('progressLabel').textContent = `${Math.round((i + 1) / count * 8)}%`;
    }
    if (!values.length) return { latency: NaN, jitter: NaN };
    const sorted = [...values].sort((a, b) => a - b);
    const median = sorted.length % 2 ? sorted[(sorted.length - 1) / 2] : (sorted[sorted.length / 2 - 1] + sorted[sorted.length / 2]) / 2;
    const deltas = values.slice(1).map((v, i) => Math.abs(v - values[i]));
    const jitter = deltas.length ? deltas.reduce((a, b) => a + b, 0) / deltas.length : 0;
    return { latency: median, jitter };
  }

  async function measureDownload(seconds, progressBase, progressSpan) {
    const controller = new AbortController();
    const start = performance.now();
    let bytes = 0;
    let lastSampleTime = start;
    let lastSampleBytes = 0;
    const abortTimer = setTimeout(() => controller.abort(), seconds * 1000);
    state.stopCurrent = () => controller.abort();
    try {
      const response = await fetch(`/api/download?run=${Date.now()}`, { cache: 'no-store', signal: controller.signal });
      if (!response.ok || !response.body) throw new Error('Download stream could not be opened.');
      const reader = response.body.getReader();
      while (!state.cancel) {
        const chunk = await reader.read();
        if (chunk.done) break;
        bytes += chunk.value.byteLength;
        const now = performance.now();
        if (now - lastSampleTime >= 180) {
          const instant = (bytes - lastSampleBytes) * 8 / ((now - lastSampleTime) / 1000) / 1000000;
          addSample(instant);
          lastSampleTime = now;
          lastSampleBytes = bytes;
        }
        const elapsed = now - start;
        const pct = Math.min(100, elapsed / (seconds * 1000) * 100);
        $('progressFill').style.width = `${progressBase + pct * progressSpan / 100}%`;
        $('progressLabel').textContent = `${Math.round(progressBase + pct * progressSpan / 100)}%`;
      }
    } catch (error) {
      if (!controller.signal.aborted && !state.cancel) throw error;
    } finally {
      clearTimeout(abortTimer);
      state.stopCurrent = null;
      controller.abort();
    }
    const elapsed = (performance.now() - start) / 1000;
    return { mbps: elapsed > 0 ? bytes * 8 / elapsed / 1000000 : 0, bytes, seconds: elapsed };
  }

  function makeUploadBlob() {
    const chunk = new Uint8Array(1024 * 1024);
    // Deterministic non-zero bytes avoid a trivially compressible all-zero payload.
    for (let i = 0; i < chunk.length; i++) chunk[i] = (i * 37 + 19) & 255;
    const parts = Array.from({ length: 64 }, () => chunk);
    return new Blob(parts, { type: 'application/octet-stream' });
  }

  async function measureUpload(seconds, progressBase, progressSpan) {
    const blob = makeUploadBlob();
    const start = performance.now();
    const deadline = start + seconds * 1000;
    let totalBytes = 0;
    let lastSampleTime = start;
    let lastSampleBytes = 0;

    while (!state.cancel && performance.now() < deadline) {
      const remaining = deadline - performance.now();
      let currentLoaded = 0;
      const result = await new Promise((resolve, reject) => {
        const xhr = new XMLHttpRequest();
        let settled = false;
        const timer = setTimeout(() => xhr.abort(), Math.max(1, remaining));
        const finish = (value) => {
          if (settled) return;
          settled = true;
          clearTimeout(timer);
          state.stopCurrent = null;
          resolve(value);
        };
        state.stopCurrent = () => xhr.abort();
        const fail = (error) => {
          if (settled) return;
          settled = true;
          clearTimeout(timer);
          state.stopCurrent = null;
          reject(error);
        };
        xhr.open('POST', `/api/upload?run=${Date.now()}`, true);
        xhr.upload.onprogress = (event) => {
          const loaded = Math.min(event.loaded, blob.size);
          totalBytes += Math.max(0, loaded - currentLoaded);
          currentLoaded = loaded;
          const now = performance.now();
          if (now - lastSampleTime >= 180) {
            const instant = (totalBytes - lastSampleBytes) * 8 / ((now - lastSampleTime) / 1000) / 1000000;
            addSample(instant);
            lastSampleTime = now;
            lastSampleBytes = totalBytes;
          }
          const pct = Math.min(100, (now - start) / (seconds * 1000) * 100);
          $('progressFill').style.width = `${progressBase + pct * progressSpan / 100}%`;
          $('progressLabel').textContent = `${Math.round(progressBase + pct * progressSpan / 100)}%`;
        };
        xhr.onload = () => {
          if (xhr.status >= 200 && xhr.status < 300) {
            totalBytes += Math.max(0, blob.size - currentLoaded);
            finish({ complete: true });
          } else {
            reject(new Error(`Upload failed: HTTP ${xhr.status}`));
          }
        };
        xhr.onerror = () => fail(new Error('Upload connection failed.'));
        xhr.onabort = () => finish({ complete: false });
        xhr.send(blob);
      });
      if (!result.complete) break;
    }
    const elapsed = (performance.now() - start) / 1000;
    return { mbps: elapsed > 0 ? totalBytes * 8 / elapsed / 1000000 : 0, bytes: totalBytes, seconds: elapsed };
  }

  function setStage(text, caption) {
    $('stageLabel').textContent = text;
    $('gaugeCaption').textContent = caption;
  }

  function downloadReport(format) {
    if (!state.lastResults) return;
    const result = state.lastResults;
    let content, type, extension;
    if (format === 'csv') {
      const fields = Object.keys(result);
      const quote = (value) => `"${String(value ?? '').replaceAll('"', '""')}"`;
      content = [fields.map(quote).join(','), fields.map(key => quote(result[key])).join(',')].join('\r\n');
      type = 'text/csv;charset=utf-8'; extension = 'csv';
    } else {
      content = JSON.stringify(result, null, 2);
      type = 'application/json'; extension = 'json';
    }
    const blob = new Blob([content], { type });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `Speed-Test-Report-${new Date(result.timestamp).toISOString().replaceAll(':', '-').replaceAll('.', '-')}.${extension}`;
    document.body.appendChild(a); a.click(); a.remove(); setTimeout(() => URL.revokeObjectURL(url), 1500);
  }

  async function saveHostResult(result) {
    try {
      const response = await fetch('/api/results', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          durationSeconds: result.durationSeconds,
          downloadMbps: result.downloadMbps,
          uploadMbps: result.uploadMbps,
          latencyMs: result.latencyMs,
          jitterMs: result.jitterMs,
          downloadBytes: result.downloadBytes,
          uploadBytes: result.uploadBytes,
          clientVersion: result.version
        })
      });
      if (!response.ok) throw new Error(`Host log save failed (${response.status})`);
      const saved = await response.json();
      result.testId = saved.testId;
      result.hostLogSaved = true;
    } catch (error) {
      result.hostLogSaved = false;
      console.warn('Could not save result to host log:', error);
    }
    state.lastResults = result;
    $('reportActions').hidden = false;
    $('resultTime').textContent = result.hostLogSaved
      ? `Saved on host · Test ID ${result.testId}`
      : `Completed ${new Date(result.timestamp).toLocaleString()} · Host log unavailable; download a local report.`;
  }

  async function checkForUpdates() {
    try {
      const response = await fetch('/api/update', { cache: 'no-store' });
      if (!response.ok) return;
      const update = await response.json();
      if (update.updateAvailable && update.latestVersion) {
        const banner = $('updateBanner');
        banner.hidden = false;
        const text = document.createElement('span');
        text.textContent = `Speed Test ${update.latestVersion} is available (you are using ${update.currentVersion}). Ask the host administrator to restart the official launcher to update. `;
        const link = document.createElement('a');
        link.href = update.releaseUrl || 'https://github.com/ajayraikvar/speed-test/releases/latest';
        link.target = '_blank'; link.rel = 'noopener noreferrer'; link.textContent = 'Release information';
        banner.replaceChildren(text, link);
      }
    } catch (_) { /* Offline hosts continue normally; update checking is optional. */ }
  }

  function setResults(download, upload, latency, jitter) {
    $('downloadResult').textContent = formatSpeed(download);
    $('downloadMBps').textContent = `${formatMBps(download)} · Host → Browser`;
    $('uploadResult').textContent = formatSpeed(upload);
    $('uploadMBps').textContent = `${formatMBps(upload)} · Browser → Host`;
    $('latencyResult').innerHTML = `${Number.isFinite(latency) ? latency.toFixed(1) : '—'} <small>ms</small>`;
    $('jitterResult').innerHTML = `${Number.isFinite(jitter) ? jitter.toFixed(1) : '—'} <small>ms</small>`;
    $('resultTime').textContent = `Completed ${new Date().toLocaleString()}`;
  }

  async function runTest() {
    if (state.running) return;
    const hostOK = await loadHostInfo();
    if (!hostOK) { showToast('The host server is not responding.'); return; }

    state.running = true;
    state.cancel = false;
    state.chart = [];
    state.maxSpeed = 1;
    drawChart();
    $('startButton').disabled = true;
    $('stopButton').disabled = false;
    $('downloadResult').textContent = '—';
    $('uploadResult').textContent = '—';
    $('latencyResult').innerHTML = '— <small>ms</small>';
    $('jitterResult').innerHTML = '— <small>ms</small>';
    $('reportActions').hidden = true;
    state.lastResults = null;
    $('progressFill').style.width = '0%';
    $('progressLabel').textContent = '0%';
    const duration = Number($('durationSelect').value);
    let latency = NaN, jitter = NaN, download = { mbps: 0, bytes: 0, seconds: 0 }, upload = { mbps: 0, bytes: 0, seconds: 0 };

    try {
      setStage('Measuring response time…', 'LATENCY CHECK');
      const ping = await measureLatency();
      latency = ping.latency; jitter = ping.jitter;
      if (state.cancel) throw new Error('Test stopped.');

      setStage('Testing host to browser…', 'DOWNLOAD TEST');
      $('liveDirection').textContent = 'Host → Browser';
      download = await measureDownload(duration, 8, 46);
      if (state.cancel) throw new Error('Test stopped.');

      setStage('Testing browser to host…', 'UPLOAD TEST');
      $('liveDirection').textContent = 'Browser → Host';
      upload = await measureUpload(duration, 54, 46);
      if (state.cancel) throw new Error('Test stopped.');

      setResults(download.mbps, upload.mbps, latency, jitter);
      const report = {
        product: 'Speed Test', version: state.hostInfo?.version || 'unknown',
        timestamp: new Date().toISOString(), durationSeconds: duration,
        downloadMbps: Number(download.mbps.toFixed(3)), uploadMbps: Number(upload.mbps.toFixed(3)),
        latencyMs: Number(latency.toFixed(3)), jitterMs: Number(jitter.toFixed(3)),
        downloadBytes: download.bytes, uploadBytes: upload.bytes,
        host: state.hostInfo?.hostname || 'unknown',
        hostAddresses: (state.hostInfo?.addresses || []).join('; '),
        userAgent: navigator.userAgent, testId: null, hostLogSaved: false
      };
      await saveHostResult(report);
      setStage('Test completed', 'TEST COMPLETE');
      $('progressFill').style.width = '100%';
      $('progressLabel').textContent = '100%';
      setSpeed(Math.max(download.mbps, upload.mbps), 'Peak measured throughput');
      showToast('Speed test completed.');
    } catch (error) {
      setStage(state.cancel ? 'Test stopped' : 'Test could not complete', state.cancel ? 'STOPPED' : 'TEST ERROR');
      showToast(error.message || 'The speed test failed.');
      if (!state.cancel) console.error(error);
    } finally {
      state.running = false;
      $('startButton').disabled = false;
      $('stopButton').disabled = true;
      if (state.cancel) {
        $('liveSpeed').textContent = '—';
        $('liveUnit').textContent = 'Mbps';
      }
    }
  }

  $('startButton').addEventListener('click', runTest);
  $('downloadJsonReport').addEventListener('click', () => downloadReport('json'));
  $('downloadCsvReport').addEventListener('click', () => downloadReport('csv'));
  $('stopButton').addEventListener('click', () => { state.cancel = true; if (state.stopCurrent) state.stopCurrent(); setStage('Stopping test…', 'STOPPING'); });
  window.addEventListener('resize', drawChart);
  drawChart();
  loadHostInfo();
  checkForUpdates();
  setInterval(checkForUpdates, 5 * 60 * 1000);
})();
