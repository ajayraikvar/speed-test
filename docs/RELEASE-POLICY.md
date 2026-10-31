# Release policy

- The official launcher checks the latest stable GitHub Release.
- Release ZIP downloads must be validated against the published SHA-256 checksum before installation.
- New versions are installed in separate version-specific directories.
- Host logs and configuration remain outside version directories and must survive updates.
- Connected clients receive update notices from the host; clients do not independently contact GitHub.
- A complete offline ZIP must include the executable and all files needed for a first-time offline run.
- Historical Git commits, public forks, mirrors, caches, and copies already downloaded by users cannot be reliably revoked. The project therefore retains public source history for transparency rather than promising deletion of old source versions.
- Stable releases must not be published until real Windows-to-Windows LAN testing and Windows launcher validation are complete.
