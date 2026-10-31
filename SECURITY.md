# Security Policy

## Scope

Speed Test is designed for trusted local networks. It is not intended to be exposed directly to the public internet.

## Reporting

Please report suspected vulnerabilities privately to the project maintainer through the contact options listed at https://www.ajayraikvar.com. Avoid publishing exploit details before a fix is available.

## Deployment precautions

- Do not configure router port forwarding to the host.
- Permit Windows Firewall access only on trusted Private networks.
- Keep the host log dashboard bound to loopback.
- Do not treat this application as an authenticated multi-tenant service.
- Review logs before sharing reports; client IP addresses and browser user-agent strings may be present.
