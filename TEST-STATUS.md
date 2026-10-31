# Test status — v1.0.0

| Check | Result |
|---|---|
| Go unit tests | PASS in GitHub Actions |
| Go vet | PASS in GitHub Actions |
| Go race detector | PASS in GitHub Actions |
| Browser JavaScript syntax | PASS in GitHub Actions |
| Windows x64 build | PASS in GitHub Actions |
| PowerShell launcher and local HTTP smoke test | RUN in GitHub Windows runner |
| Actual Windows-to-Windows LAN test on separate physical PCs | PENDING independent user validation |

## Measurement limits

The browser edition measures HTTP/TCP throughput and HTTP round-trip latency. It does not measure raw ICMP ping or UDP packet loss. Browser-based measurements include browser and application overhead.

## Release note

This is the initial public v1.0.0 release. Automated CI and Windows-runner checks are performed, but a separate physical two-PC LAN test has not been independently confirmed. Do not treat the software as a calibrated network certification instrument.
