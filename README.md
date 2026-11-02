# Speed Test v2.1.0

**A self-hosted network performance analyzer for Windows.**

One Windows PC runs the host. Other PCs use a modern browser; no client application, browser extension, internet speed-test service, external CDN, or third-party runtime is required.

- **Original project author and maintainer:** Ajay Raikvar
- **Website:** https://www.ajayraikvar.com
- **Official repository:** https://github.com/ajayraikvar/speed-test
- **License:** GNU Affero General Public License v3 or later (`AGPL-3.0-or-later`)
- **Version:** 2.1.0

## Quick start — Windows host

1. Download and extract `Speed-Test-Windows-x64.zip`.
2. Double-click `Start-Speed-Test.bat`.
3. Enter the host port (default `80`) and maximum browser clients (default `16`).
4. Open one of the printed `http://HOST-IP:PORT` addresses in Edge, Chrome, Firefox, or another current browser on each client PC.
5. Keep the host console open while tests are running. Press `Ctrl+C` to stop the server.

Only the host PC runs the BAT file and executable. Client PCs use a browser only.

## What's included in v2.0

### v1.0 foundation

- Rebranded Speed Test application, original author attribution, AGPL-3.0-or-later licensing, and a self-hosted Windows host/browser-client architecture.
- Automatic host IPv4 address detection and configurable host port.
- Memory-based transfer tests; no test payload is written to disk.

### v1.1 speed testing and diagnostics

- HTTP/TCP download and upload throughput measurements.
- 1, 2, 4, or 8 parallel streams.
- Sequential and simultaneous bidirectional test modes.
- 5, 10, 20, 30, or 60-second transfer durations.
- Live download/upload graph, average, minimum sample, and peak throughput.
- HTTP round-trip latency (median/min/max) and jitter estimate.
- Local-host versus remote-LAN scope indicator.
- Browser, platform, and best-effort browser network API hints.

### v1.2 history and reporting

- Persistent host-side JSON Lines test history.
- Host administration dashboard with connected clients, history, performance comparison, and CSV/JSON exports. Source-address checks are not user authentication; keep access restricted with the host firewall.
- Optional client-side JSON/CSV report download and browser-local recent reports.
- Print-to-PDF report option using the browser's print dialog.

### v1.3 updates and offline use

- Launcher checks the latest stable GitHub Release when internet is available.
- Downloaded packages are SHA-256 verified before installation.
- Each downloaded version is installed in its own version-specific folder.
- Existing logs and settings remain outside version folders and are not modified by updates.
- Offline startup uses the bundled server or the newest locally cached version.
- Connected browser clients receive the host's cached update notice; clients do not contact GitHub themselves.

### v1.4 multiple clients

- Multiple browser clients can be connected at once (host-configurable limit, 1–128).
- Clients identify themselves with a browser-generated session ID and an editable device label.
- Host dashboard shows active clients and compares recent results by device and IP.
- Inactive sessions expire after 35 seconds without a heartbeat.

### v2.0 network analyzer dashboard

- Combined host/client overview and network diagnostics.
- Consistent Mbps/Gbps conversion across gauge, result cards, chart, recent reports, and host dashboard.
- Per-test metadata: device label, IP, browser/platform, test mode, stream count, duration, throughput, HTTP RTT, jitter, and byte counts.
- Local corresponding-source ZIP and license notice are offered from the running host (`/source.zip`, `/LICENSE`, `/NOTICE`).

## v2.1 interface refresh

- Modernized the main Speed Test screen and host dashboard with a cleaner light theme and responsive layouts.
- Improved visual hierarchy for test controls, the live gauge, throughput chart, result cards, diagnostics, client comparison, and history.
- Added clearer keyboard focus states, reduced-motion handling, and responsive layouts for smaller screens.
- The v2.1.0 update is a presentation refresh; it does not change the HTTP/TCP measurement method or add admin login/authentication.

## Measurement scope and limitations

This is a browser-based HTTP/TCP throughput analyzer. Download and upload rates are measured over the connection between the browser and this host. Multi-stream mode opens parallel HTTP connections; bidirectional mode runs upload and download transfers at the same time.

Latency and jitter are estimated with repeated HTTP round trips to this host. They are **not raw ICMP ping measurements**. Standard browser JavaScript cannot send arbitrary raw UDP packets or ICMP echo requests, so this release does **not** claim to measure UDP packet loss. Browser APIs do not reliably reveal the physical Ethernet/Wi-Fi link speed or link type on every platform; the displayed network-type field is only a best-effort browser hint. Host interface labels are detected from operating-system interface names and may be generic on some systems.

Browser timing includes browser scheduling, JavaScript, HTTP, and server overhead. Results are useful for comparing the same LAN path under consistent conditions, but they are not a calibrated network certification instrument. For best comparison, use wired Ethernet on both PCs, close other heavy transfers, and keep duration/stream settings consistent.

## Host data and privacy

Host data is stored outside the application/version folder:

`%LOCALAPPDATA%\AjayRaikvar\SpeedTest\logs\tests.jsonl`

The JSONL file is append-only from the application; it is not cleared during updates. The host dashboard and its APIs do not provide user login. Source-address checks are not a substitute for authentication or firewall rules; keep the service on a trusted private LAN and restrict its port to trusted devices. Host logs may contain client IP addresses, device labels, browser user-agent details, and test statistics. Review them before sharing. The transfer payloads are discarded in memory and are not stored as files.

The client browser keeps a short list of recent reports in that browser's local storage. It can also download an individual JSON/CSV report or print it to PDF. The host keeps its own central history independently.

## Updating and offline use

Run `Start-Speed-Test.bat` to check for a newer official stable release. The launcher compares the release version against both the bundled package and locally cached versions. It downloads only if GitHub has a newer version, verifies the published SHA-256 checksum, and extracts to a new version-specific folder. If update checking or download fails, it uses the newest available local copy.

A first-time offline setup must use the complete Windows ZIP, which includes the executable, launchers, source ZIP, license, and documentation. A standalone BAT or PowerShell script alone is not sufficient for a first-time offline install.

## Build from source

Development prerequisites: Go 1.23+ and Node.js (for JavaScript syntax checks). Runtime dependencies: none beyond Windows PowerShell and the browser clients.

```powershell
go test ./...
go vet ./...
go test -race ./...
node --check web/app.js
node --check web/admin.js
```

To build a Windows x64 executable manually:

```powershell
$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
$env:CGO_ENABLED = '0'
go build -trimpath -ldflags "-s -w -X main.appVersion=2.0.0" -o dist/speed-test.exe .
```

The complete build/package helper is `scripts/build-package.ps1`.

## Security guidance

- Intended for a trusted private LAN. Do not expose the host port to the public internet or configure router port forwarding.
- When Windows Firewall asks, allow access only on trusted Private networks.
- The host binds to all local interfaces so LAN clients can connect; firewall policy is the network boundary.
- The admin page and APIs apply source-address checks but do not provide authenticated user accounts; do not rely on these checks instead of Windows Firewall restrictions.
- Configure a sensible maximum client count; each browser can open multiple streams.
- Do not use the application as an authenticated, multi-tenant internet service.

See [SECURITY.md](SECURITY.md) for reporting guidance.

## Attribution and contributions

The original upstream project, application identity, and official release channel are maintained by **Ajay Raikvar**. Preserve copyright, license, and branding notices. Modified versions must be clearly identified as modified and must not imply that an unofficial fork is an official Ajay Raikvar release. Contributors retain copyright in their own contributions unless they separately agree otherwise; accepted contributions are distributed under the project license.

## Version 2.0 roadmap completion

| Planned version | Delivered in v2.0.0 |
|---|---|
| v1.0 — Rebrand, licensing, stable host/client | Included |
| v1.1 — Improved speed tests, multi-stream, diagnostics | Included, with browser measurement limitations documented |
| v1.2 — Persistent logs, client reports | Included |
| v1.3 — Auto-updater, offline caching, notifications | Included in launcher and host/client notification flow |
| v1.4 — Multiple clients and comparisons | Included |
| v2.0 — Network performance analyzer | Included as a browser-based HTTP/TCP analyzer; raw ICMP/UDP tests remain outside browser capability |

---

© 2026 Ajay Raikvar · Developed by [www.ajayraikvar.com](https://www.ajayraikvar.com) · You may use, modify, and redistribute under AGPL-3.0-or-later · No warranty
