# Architecture

- **Host runtime:** Go HTTP server with embedded HTML/CSS/JavaScript. No external runtime modules are required.
- **Client:** modern browser only; client identity is a browser-generated opaque ID with an editable device label.
- **Data plane:** HTTP/TCP download and upload streams. Parallel streams are independent HTTP requests; bidirectional mode runs both directions at once.
- **Control plane:** HTTP endpoints for host info, heartbeat/session registration, updates, and result submission.
- **Session capacity:** configurable maximum clients (1–128); sessions expire after 35 seconds without a heartbeat.
- **Persistence:** append-only JSON Lines host history outside the application directory under `%LOCALAPPDATA%\AjayRaikvar\SpeedTest`.
- **Admin:** loopback/host-interface-only dashboard and read/export APIs.
- **Updates:** launcher checks the latest stable GitHub Release, validates SHA-256, and installs new versions in separate folders. Existing logs and configuration remain outside version folders.
- **Offline:** bundled server executable and source archive are included in the complete Windows package.

## Network measurement boundaries

HTTP/TCP data transfer and HTTP request/response round trips can be measured in a regular browser. Raw ICMP and arbitrary UDP diagnostics require native capabilities unavailable to ordinary browser JavaScript and are not claimed by this edition.
