# Contributing

Contributions are welcome. Please open an issue before large changes so scope and compatibility can be discussed.

## Development checks

```powershell
go test ./...
go vet ./...
go test -race ./...
node --check web/app.js
node --check web/admin.js
```

## Contribution expectations

- Keep the host self-contained and avoid unnecessary third-party runtime dependencies.
- Preserve copyright, license, and attribution notices.
- Clearly identify modified versions and do not imply that unofficial builds are official upstream releases.
- Include tests for new server behaviour and document browser measurement limitations.
- Do not commit user test logs, machine-specific settings, credentials, or generated executables.
- Contributors retain copyright in their contributions unless they separately agree otherwise. Contributions are submitted for distribution under AGPL-3.0-or-later as part of this project.
