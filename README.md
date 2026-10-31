# Speed Test

**A self-hosted network performance analyzer for Windows.**

One Windows PC runs the host server. Other PCs connect using a modern browser; no client installation is needed.

- **Original author and upstream maintainer:** Ajay Raikvar
- **Website:** https://www.ajayraikvar.com
- **Repository:** https://github.com/ajayraikvar/speed-test
- **License:** GNU AGPL v3 or later (AGPL-3.0-or-later)

## Release candidate

This repository is being published as a **release candidate for local testing**, not as a production-certified network diagnostic tool. A real Windows-to-Windows LAN test is still required.

## Quick start (Windows host)

1. Download and extract the complete Windows offline package from the [repository files](https://github.com/ajayraikvar/speed-test).
2. Double-click `Start-Speed-Test.bat`.
3. Enter a host port (default: `80`).
4. Open the printed URL in a browser on another PC on the same LAN.
5. Keep the host console open. Press `Ctrl+C` to stop the host.

Client PCs need only a modern browser. No browser extension, client app, or client-side installer is required.

## Features

- Browser-based HTTP/TCP download and upload throughput tests
- HTTP round-trip latency and jitter estimates
- Live throughput graph
- Automatic host IPv4 address detection and configurable port
- Host-side JSONL test history and local admin dashboard
- Optional client JSON/CSV report download
- GitHub release update check, checksum verification, and offline fallback
- Standalone Windows x64 server executable

## Important measurement limits

This browser edition reports HTTP/TCP throughput and HTTP round-trip latency. It does **not** claim to measure raw ICMP ping or UDP packet loss. Browser timing includes browser and application overhead; results are useful for comparing the tested path, not as a calibrated certification instrument.

## Offline and updates

The complete offline package includes the server executable and can start without internet. A first-time offline installation must use that complete package. The online launcher checks GitHub Releases and verifies the downloaded ZIP using the release SHA-256 checksum before installing it in a version-specific directory. Host logs are stored separately from application binaries.

Connected clients receive update notices from the host; clients do not need to contact GitHub themselves.

## Build and test from source

Requires Go 1.21 or later for development:

```powershell
go test ./...
go vet ./...
go test -race ./...
node --check web/app.js
node --check web/admin.js
```

The Go server uses only the Go standard library. The provided Windows launcher uses PowerShell included with supported Windows installations.

## Security

The server is intended for a trusted local network. Do not expose its port to the public internet or configure router port-forwarding. Allow Windows Firewall access only on trusted Private networks. The host log dashboard is restricted to loopback requests.

## Attribution and contributions

The original project identity, upstream repository, and official release channel are maintained by **Ajay Raikvar**. Preserve copyright, license, and attribution notices. Identify modified versions clearly and do not represent unofficial forks as official Ajay Raikvar releases.

Contributors retain copyright in their own contributions unless separately agreed. Contributions are accepted under the project license; see `CONTRIBUTING.md`.

## Release history policy

The official launcher targets the latest stable release. Historical Git commits and public forks are retained for transparency and cannot be reliably revoked; the project does not promise that old source snapshots can be made inaccessible.

---

© 2026 Ajay Raikvar · Developed by [www.ajayraikvar.com](https://www.ajayraikvar.com)
