# Speed Test

**A self-hosted network performance analyzer for Windows.** One PC runs the host server; other PCs use a modern browser. No client application, external speed-test service, internet measurement endpoint, or third-party runtime is required.

**Created and maintained by [Ajay Raikvar](https://www.ajayraikvar.com)**  
**Project:** https://github.com/ajayraikvar/speed-test  
**License:** GNU Affero General Public License v3.0 or later (`AGPL-3.0-or-later`)

## Quick start

1. Extract the complete Windows package to the host PC.
2. Double-click `Start-Speed-Test.bat`.
3. Enter a host port (default `80`).
4. Open one of the printed URLs from another PC in Edge, Chrome, Firefox, or another modern browser.
5. Keep the host console open while tests are running. Press `Ctrl+C` to stop the server.

The client PCs do not need a BAT file, EXE, extension, or installation.

## Online updates and offline operation

The launcher (`Start-Speed-Test.bat` + `Start-Speed-Test.ps1`) checks the public GitHub Releases API for the latest stable release. If a newer release exists, it downloads the Windows ZIP and its checksum file, verifies SHA-256, extracts the release into a version-specific folder, and starts it.

If GitHub has no published release yet (the Releases API returns HTTP 404), the launcher treats that as a normal pre-release state and starts the bundled executable. The complete offline ZIP contains `speed-test.exe`. If the host has no internet access, the launcher uses that bundled executable or a previously installed local version. A first-time offline installation must use the complete offline ZIP; a standalone launcher without a cached/bundled executable cannot download an application while offline.

The host checks for updates in the background approximately every 15 minutes; connected browser clients poll the host for a cached notice every five minutes. Connected browser clients read the host's cached update status and show a notice when a newer release is available. Clients do not independently contact GitHub.

## Host logs and client reports

Completed test summaries are appended to a host-side JSON Lines file:

`%LOCALAPPDATA%\AjayRaikvar\SpeedTest\logs\tests.jsonl`

The host-only dashboard is available locally at `http://127.0.0.1:<port>/admin`. It can view recent results and export CSV or JSON. The admin page and its log API reject non-loopback requests; regular clients cannot browse the host log.

After a test, the client may optionally download its own report as JSON or CSV. The server stores summary metrics only, not the generated test payloads.

Each host log can include: test ID, UTC timestamp, client IP address, browser user-agent, duration, download/upload throughput, latency, jitter, transferred byte counts, client version, and completion status. The client IP and user-agent are collected for local troubleshooting; keep host logs private and clear them using normal file-management tools when appropriate.

## What's measured

- **Download:** host to browser throughput over HTTP/TCP.
- **Upload:** browser to host throughput over HTTP/TCP.
- **Latency:** repeated HTTP round-trip time, summarized by median.
- **Jitter:** average absolute difference between successive HTTP round-trip samples.
- **Live throughput graph:** rolling transfer-rate samples.

This browser-based edition does not claim to measure raw ICMP ping or UDP packet loss. Those require a separate native diagnostic path and are not inferred from browser HTTP measurements.

## Runtime requirements

- **Host:** Windows 10/11 x64 for the provided release package. The Go server is standalone; Go, Node.js, Python, Java, PHP, PowerShell, and a separate web server are not runtime dependencies. PowerShell is used by the update launcher and is included with supported Windows installations.
- **Client:** any modern browser with streaming Fetch and XMLHttpRequest support.
- **Network:** host and clients must be able to reach one another over the LAN.

Windows Defender Firewall may ask whether to allow the server. Permit it on **Private networks only** for a trusted network. The app does not create firewall rules or modify Windows configuration automatically.

## Persistent data locations

Application versions are installed under:

`%LOCALAPPDATA%\AjayRaikvar\SpeedTest\versions\`

Host logs are stored separately under:

`%LOCALAPPDATA%\AjayRaikvar\SpeedTest\logs\`

Updating the executable does not overwrite or clear logs. The launcher stages a new version separately and only starts it after the executable has been extracted and verified.

## Build from source

Install Go 1.21 or later, then run:

```powershell
go test ./...
go vet ./...
go test -race ./...
$env:GOOS='windows'; $env:GOARCH='amd64'; go build -trimpath -ldflags='-s -w' -o dist/speed-test.exe .
```

The application currently uses only the Go standard library. No third-party Go modules are required.

## Attribution and contributions

The original project, product identity, upstream repository, and official release channel are maintained by **Ajay Raikvar**. Preserve existing copyright, license, and attribution notices when copying or modifying the project. Clearly identify modified versions and their maintainers; do not imply that an unofficial fork is the official upstream build.

Contributors retain copyright in their own contributions unless they separately agree otherwise. Contributions are accepted under the project license so the project can be distributed as a combined AGPL-licensed work. See [CONTRIBUTING.md](CONTRIBUTING.md).

## Release policy

The launcher follows the latest stable GitHub Release. The project may promote one release as the current official download, but public Git history, tags, forks, caches, and already-downloaded files cannot be reliably revoked. Previous source commits are intentionally retained for transparency and reproducibility. See [docs/RELEASE-POLICY.md](docs/RELEASE-POLICY.md).

## Security and exposure

This server is designed for a trusted local network. It listens on all IPv4 interfaces so LAN clients can connect. It does not provide user authentication or TLS by default. Do not expose the host port to the public internet or configure router port-forwarding. The `/admin` dashboard and host logs are restricted to loopback requests on the host PC.

## Project status

Version 1.0.0 is the initial public release. Automated Go tests, race checks, static checks, the Windows build, and a Windows-runner launcher smoke test are run by GitHub Actions. A physical Windows-to-Windows LAN test has not yet been independently confirmed; validate your own network before production use.

---

© 2026 Ajay Raikvar · Developed by [www.ajayraikvar.com](https://www.ajayraikvar.com)
