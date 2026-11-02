# Test status — Speed Test v2.1.0

v2.1.0 is a UI-focused update. Go server logic and browser JavaScript are unchanged from v2.0.0. The release workflow re-runs the Go tests, vet, race detector, JavaScript syntax checks, and Windows package build against this exact release branch. The results of that GitHub Actions run are authoritative for this release.

| Check | Result |
|---|---|
| `go test ./...` | PASS in v2.0.0 baseline; re-run in v2.1.0 release workflow |
| `go vet ./...` | PASS in v2.0.0 baseline; re-run in v2.1.0 release workflow |
| `go test -race ./...` | PASS in v2.0.0 baseline; re-run in v2.1.0 release workflow |
| `node --check web/app.js` | PASS in v2.0.0 baseline; re-run in v2.1.0 release workflow |
| `node --check web/admin.js` | PASS in v2.0.0 baseline; re-run in v2.1.0 release workflow |
| Linux amd64 static host build | PASS |
| Windows x64 cross-build | PASS (PE32+ executable verified) |
| Local HTTP API / transfer / source-license / multi-client smoke test | PASS |
| Persistent host JSONL log and admin JSON/CSV export | PASS |
| PowerShell launcher execution on Windows | NOT RUN in this environment |
| Actual Windows-to-Windows LAN test on separate physical PCs | PENDING user validation |
| Main/Admin DOM ID compatibility and CSS brace-balance checks | PASS (source-level checks) |
| Browser UI visual/end-to-end compatibility on Edge/Chrome/Firefox | PENDING user validation |

## v2.1 UI validation

The redesigned CSS preserves the element IDs consumed by `app.js` and `admin.js`; the updated admin page has no duplicate IDs. Both CSS files have balanced braces. Actual browser rendering, resizing, and interaction checks in Edge, Chrome, and Firefox have not been performed in this environment and remain pending user validation.

## Smoke test coverage

The local host smoke test verified host info/version, embedded UI serving, corresponding source ZIP download, full license and notice serving, HTTP ping, two-client capacity enforcement, upload acceptance, download streaming, structured result persistence, host admin JSON/CSV export, and persistent JSONL log creation.

## Measurement limits

The browser edition measures HTTP/TCP throughput and HTTP round-trip latency/jitter estimates. It does not measure raw ICMP ping or UDP packet loss. The browser network type is a best-effort API hint, not a guaranteed Ethernet/Wi-Fi classification. Browser timings include browser and application overhead.

## Release note

The v2.0.0 source tests, race checks, vet, JavaScript syntax checks, Linux build, Windows x64 cross-build, and local host API smoke test passed. A physical two-PC Windows LAN test, browser compatibility test, and PowerShell launcher execution on Windows remain unverified here. Test on your own Windows PCs before treating the results as production measurements. This is not a calibrated network certification instrument.
