# Test status — Speed Test v2.0.0

Build and automated verification performed in the current build environment:

| Check | Result |
|---|---|
| `go test ./...` | PASS |
| `go vet ./...` | PASS |
| `go test -race ./...` | PASS |
| `node --check web/app.js` | PASS |
| `node --check web/admin.js` | PASS |
| Linux amd64 static host build | PASS |
| Windows x64 cross-build | PASS (PE32+ executable verified) |
| Local HTTP API / transfer / source-license / multi-client smoke test | PASS |
| Persistent host JSONL log and admin JSON/CSV export | PASS |
| PowerShell launcher execution on Windows | NOT RUN in this environment |
| Actual Windows-to-Windows LAN test on separate physical PCs | PENDING user validation |
| Browser UI end-to-end compatibility on Edge/Chrome/Firefox | PENDING user validation |

## Smoke test coverage

The local host smoke test verified host info/version, embedded UI serving, corresponding source ZIP download, full license and notice serving, HTTP ping, two-client capacity enforcement, upload acceptance, download streaming, structured result persistence, host admin JSON/CSV export, and persistent JSONL log creation.

## Measurement limits

The browser edition measures HTTP/TCP throughput and HTTP round-trip latency/jitter estimates. It does not measure raw ICMP ping or UDP packet loss. The browser network type is a best-effort API hint, not a guaranteed Ethernet/Wi-Fi classification. Browser timings include browser and application overhead.

## Release note

The v2.0.0 source tests, race checks, vet, JavaScript syntax checks, Linux build, Windows x64 cross-build, and local host API smoke test passed. A physical two-PC Windows LAN test, browser compatibility test, and PowerShell launcher execution on Windows remain unverified here. Test on your own Windows PCs before treating the results as production measurements. This is not a calibrated network certification instrument.
