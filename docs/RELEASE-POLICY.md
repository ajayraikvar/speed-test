# Release and latest-only download policy

The public launcher resolves the newest stable GitHub Release and verifies the Windows package against the SHA-256 checksum file before installing it.

The official download page should prominently link to `/releases/latest` and the latest launcher/offline package. When a new stable version is released, the latest release becomes the default official download. Users on older versions receive an in-app warning from the host's cached GitHub update check.

## Important limitation

A public Git repository cannot guarantee that older source or binaries become impossible to obtain. Git commit history, tags, forks, third-party mirrors, caches, and copies already downloaded by users may continue to exist. Deleting or rewriting public Git history is not a reliable revocation mechanism and damages transparency/reproducibility.

For that reason, this project does not automatically delete Git history, tags, or prior release records. If the maintainer later chooses to remove old official release assets, that must be a deliberate manual release-management action after keeping a backup. The latest-only policy applies to the official download path, not to all copies on the internet.

## Release asset contract

Every stable Windows release must publish:

- `Speed-Test-Windows-x64.zip` — offline-capable package containing `speed-test.exe`, `Start-Speed-Test.bat`, `Start-Speed-Test.ps1`, `README.md`, `LICENSE`, and `NOTICE`.
- `SHA256SUMS.txt` — SHA-256 hash for the ZIP.

The PowerShell launcher must reject a package if its checksum is missing or does not match. Failed downloads/extractions must not replace a working installed version or touch host logs.
