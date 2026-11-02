# Contributing

Contributions are welcome. Open an issue before large changes so the scope and compatibility impact can be discussed.

## Development checks

```powershell
go test ./...
go vet ./...
go test -race ./...
node --check web/app.js
node --check web/admin.js
```

## Expectations

- Keep the host self-contained and use the Go standard library unless a dependency is clearly justified.
- Preserve copyright, license, warranty disclaimer, and attribution notices.
- Clearly identify modified versions; do not imply unofficial builds are official upstream releases.
- Add server tests for new endpoints and browser tests/checks for new client behaviour.
- Do not commit real host logs, local settings, credentials, or generated test payloads.
- Contributions are distributed under AGPL-3.0-or-later as part of this project; contributors retain their own copyright unless they separately agree otherwise.
