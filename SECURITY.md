# Security Policy

Machineward is early-stage software and is not ready for external installation or production use.

## Reporting a vulnerability

Please do not open a public issue for a security vulnerability. Report it privately through GitHub's security advisory feature for this repository. Include:

- A clear description of the issue
- The affected component and version or commit
- Reproduction steps or a proof of concept
- The potential impact
- Any suggested mitigation

I will acknowledge reports as soon as practical and keep the reporter informed while the issue is investigated.

## Security expectations

Machineward is designed around encrypted transport, short-lived pairing credentials, session revocation, authorization checks, rate limits, and an explicit action allowlist. These controls are part of the release gate before external users are invited to connect a phone to a computer.

Please do not use development builds to control a machine containing sensitive data.
