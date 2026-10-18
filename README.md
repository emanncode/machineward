# Machineward

> Know what your computer is doing. Act when it matters.

Machineward is an open-source, privacy-first phone companion for understanding the state of a personal computer and taking a small set of safe remote actions.

It is being built in public as a portfolio project focused on Linux, secure communication, clear system insights, and thoughtful product engineering.

## The problem

When a long-running task is running on your computer—a build, download, render, local service, or development workflow—you may want to know:

- Is the computer still healthy?
- Is the task still making progress?
- What is consuming the system’s resources?
- Does anything need attention?
- Can I take a safe action without exposing full remote control?

Machineward aims to answer those questions from a phone without turning the project into a general-purpose remote desktop or shell.

## Current status

Machineward is in early development.

The first version is being designed and tested around:

- Pop!\_OS 24.04 LTS
- x86_64 desktop systems
- One paired phone and computer
- CPU, per-core, memory, swap, and process information
- Plain-language system insights
- A small allowlist of safe remote actions
- Secure pairing and authenticated communication

The system is not ready for external installation or production use yet.

## Planned architecture

Machineward is planned as three main components:

- **Desktop agent** — runs on the user’s computer and collects system state
- **Relay service** — helps the phone and computer communicate without requiring direct inbound access
- **Mobile app** — built with React Native and Expo; displays system health and provides bounded remote actions

The initial implementation uses Go for the desktop and relay components, and React Native with Expo for the mobile application. The agent and relay share one Go module in this monorepo. See the [V0 architecture decision](docs/architecture/0001-v0-architecture.md) and [protocol contract](protocol/README.md).

## Scope

Machineward is intentionally focused.

### In scope

- Personal computer monitoring
- Linux-first support
- Resource and process visibility
- Plain-language explanations
- Secure pairing
- Session revocation
- Safe, allowlisted remote actions
- Self-hosting documentation

### Out of scope for the initial versions

- Screen sharing
- Mouse and keyboard control
- Arbitrary shell commands
- File browsing or file transfer
- Enterprise fleet management
- Unrestricted remote administration

These boundaries keep the product understandable and reduce the security surface while the core experience is being built.

## Security direction

Security is a core part of the project.

The design includes:

- One-time pairing tokens
- Short-lived credentials
- Encrypted transport
- Per-message authentication and authorization
- Agent-bound phone sessions
- Session revocation
- Rate limiting
- An explicit command allowlist
- Auditable action results

Security requirements will be treated as a release gate before external users are invited to install or control a machine through Machineward.

## Roadmap

### V0

- Desktop agent prototype
- Secure phone-to-agent pairing
- Basic CPU, memory, swap, and process metrics
- Health summary
- Plain-language resource insights
- One safe process action
- Pop!\_OS 24.04 LTS support

### V1

- Improved health summaries
- Safe power actions
- Better session management
- Ubuntu 24.04 LTS support
- Fedora Workstation support
- Self-hosting documentation
- Security review of the complete pairing and action flow

### Later

- More Linux distributions
- Reconnection and offline handling
- History and alerts
- Additional platforms
- Broader system actions where they can be made safe and understandable

## Building in public

Development decisions, scope changes, security work, and implementation progress will be shared publicly.

The project is being built to demonstrate practical engineering skills across:

- Product definition
- Distributed systems
- Mobile development
- Linux system integration
- Secure communication
- Open-source documentation
- Iterative product design

## Contributing

The project is still being shaped. Contribution guidelines will be added as the first usable components are published.

Ideas, technical feedback, and carefully scoped issues are welcome.

## License

Machineward is intended to be released under the Apache License 2.0.

See [`LICENSE`](LICENSE) for the full license text.

## Project links

- Website: https://machineward.vercel.app
- Project planning: [Linear project](https://linear.app/emanncode/project/machineward-e87dae88c42d)
- Build updates: [X](https://x.com/)
- Author: [EmannCode](https://github.com/emanncode)

---

Machineward is early, experimental, and evolving. The goal is to build a useful personal computer companion while making the engineering decisions visible along the way.
