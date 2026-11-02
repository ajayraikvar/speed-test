# Changelog

## 2.1.0 — Modern interface refresh

- Redesigned the main Speed Test dashboard with clearer navigation, controls, live test visualization, result cards, and diagnostics.
- Refreshed the host administration dashboard with more readable KPI cards, connected-client comparisons, and test history.
- Added responsive layouts for mobile and tablet widths, keyboard focus styling, reduced-motion handling, and improved print presentation.
- Kept existing application JavaScript and element IDs unchanged; no changes to throughput or latency measurement behavior.
- Clarified that admin source-address restrictions are not user authentication and that the host should remain behind trusted-LAN firewall rules.

## 2.0.0 — Complete Network Performance Analyzer

- Consolidated the v1.0–v1.4 roadmap into one release.
- Corrected dynamic Mbps/Gbps formatting across gauge, result cards, recent reports, and host dashboard.
- Added 1/2/4/8 parallel HTTP streams and sequential/bidirectional modes.
- Added 5/10/20/30/60-second test durations, live direction-specific graph, average/minimum/peak values.
- Added median/min/max HTTP round-trip latency and jitter estimates.
- Added local-versus-LAN scope detection and browser/platform/network API hints.
- Added editable client labels, active-client heartbeats, configurable client capacity, and inactive-session expiry.
- Expanded persistent host JSONL logs and host dashboard client comparison/history/CSV/JSON export.
- Added client-local recent reports, JSON/CSV download, and browser print-to-PDF.
- Improved version-aware launcher selection, SHA-256-verified updates, version-specific installation, and offline fallback.
- Added host-served corresponding source archive, full license, and notices.
- Preserved original developer attribution: Ajay Raikvar.

## 1.0.0 — Initial public release

- Self-hosted Windows host and browser-only clients.
- HTTP/TCP download/upload, HTTP round-trip latency and jitter estimates.
- Persistent host logs and optional client reports.
- Windows launcher with update check and offline bundled executable.
