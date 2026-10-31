# Contributing

Thank you for considering a contribution to Speed Test.

## Before submitting

- Open an issue for significant changes so the design can be discussed first.
- Keep the server dependency-free where practical; the current Go application uses only the standard library.
- Never add telemetry to third-party services, remote analytics, tracking pixels, or external CDNs.
- Keep host logs local and protect administrative log access.
- Add or update tests for server behaviour and input validation.
- Clearly mark material changes in the changelog and UI where appropriate.

## Development checks

```powershell
go test ./...
go vet ./...
go test -race ./...
```

## Attribution and licensing

By submitting a contribution, you confirm that you have the right to submit it and agree that it may be distributed as part of Speed Test under AGPL-3.0-or-later. You retain copyright in your contribution unless you separately agree otherwise. Existing Ajay Raikvar copyright, author, website, and upstream project notices must remain intact. Add your name to the contributor credits when the contribution is accepted.

Do not submit secrets, personal data, private logs, or files from an employer/client environment.
