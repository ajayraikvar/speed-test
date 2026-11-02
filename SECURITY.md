# Security Policy

## Scope

Speed Test is designed for trusted private networks. It is not intended to be exposed directly to the public internet or used as an authenticated multi-tenant service.

## Reporting

Please report suspected vulnerabilities privately through the contact options on https://www.ajayraikvar.com. Avoid publishing exploit details before a fix is available.

## Deployment precautions

- Do not configure router port forwarding to the host.
- Permit Windows Firewall access only on trusted Private networks.
- Use a conservative maximum client count on shared networks.
- Keep the host-only dashboard on the host PC; its APIs reject remote requests.
- Review local host logs before sharing them because they may contain client IP addresses, device labels, and browser information.
- The tool measures throughput, not security posture; it does not replace firewall, endpoint, or network security controls.
