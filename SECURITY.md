# Security Policy

## Supported versions

Security fixes are made against the latest stable release and the current default branch.

## Reporting a vulnerability

Please do not publish exploitable details in a public issue. Contact the project maintainer through the contact details on https://www.ajayraikvar.com and include a minimal reproduction, affected version, impact, and suggested mitigation where possible.

## Deployment guidance

- Use only on a trusted private network.
- Do not port-forward the host port or expose it directly to the public internet.
- Allow Windows Firewall access on Private networks only.
- The host-only `/admin` and log endpoints are restricted to loopback requests; keep the host machine account secure.
- Host logs contain client IP addresses and browser user-agent strings. Protect the host profile and its `%LOCALAPPDATA%\AjayRaikvar\SpeedTest\logs` directory.
