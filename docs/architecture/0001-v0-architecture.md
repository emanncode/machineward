# ADR 0001: V0 application architecture

- Status: Accepted for the first implementation slice
- Date: 2026-10-18

## Context

V0 is a focused prototype for one Pop!_OS 24.04 x86_64 computer and one phone. It must show current system state remotely, explain one useful condition, and eventually support one carefully scoped process action. The existing [V0 build plan](https://linear.app/emanncode/document/v0-build-plan-251a178346c1) sets the capability and security boundaries.

## Decisions

### Repository and components

Keep one repository with independently runnable components:

- `agent`: unprivileged Go process that collects system metrics and later executes the single allowlisted action.
- `relay`: Go service that accepts outbound agent and phone connections, owns pairing/session state, and routes authorized messages.
- `mobile`: React Native application using Expo and TypeScript.
- `protocol`: versioned JSON message contract shared by all components.

Use one Go module for the agent and relay initially. This keeps shared protocol types local without adding a workspace or generated-code pipeline before the contract stabilizes.

### Mobile framework

Use Expo as the React Native framework. The app needs QR scanning and secure local storage for pairing credentials, both available through Expo modules. React Native's current getting-started guidance recommends using a framework for new apps. Keep app logic in React Native so native modules can be introduced without changing the product architecture.

### Communication

Use JSON messages over WebSockets. Both the agent and phone initiate outbound connections to the relay, so the computer does not need an inbound firewall port. Use TLS for non-loopback environments. A local development relay may run without TLS only when bound to loopback and with pairing/actions disabled.

Each message has a protocol version, message ID, type, timestamp, and type-specific payload. Validate message shape, connection role, session binding, and authorization at the relay. The first implementation slice covers typed registration and message routing only; it does not imply production authentication.

### Metrics and actions

- Publish CPU, per-core CPU, RAM, swap, and per-process CPU/memory data every two seconds, as specified in the V0 plan.
- Keep the agent unprivileged. The V0 action is limited to terminating a selected process owned by the logged-in user, after revalidating its identity to guard against PID reuse.
- Do not add arbitrary command execution, remote desktop, file access, history, alerts, or reconnection behavior to V0.

## Implementation order

1. Define the versioned protocol and local development boundaries.
2. Build a relay skeleton that registers connections and routes typed messages.
3. Build the Linux agent and live metrics collection.
4. Build a read-only mobile view against real agent data.
5. Add one-time QR pairing, session authorization, TLS, expiration, and revocation.
6. Add the process action and explicit result flow.
7. Validate the complete flow on Pop!_OS 24.04 x86_64 before claiming V0 support.

## Deferred decisions

- Public relay hosting, domain, and production certificate automation. These are needed before remote end-to-end validation, not before local component work.
- Mobile release/distribution channel. V0 development targets Android first; iOS is not part of the current acceptance target.
- Agent service installation and packaging. During development, run it manually as the logged-in user. Packaging follows the security review.

## Development environment note

The current development host is Pop!_OS 22.04 LTS. It can be used to build the early components, but it cannot serve as the V0 target-platform verification environment. That check requires Pop!_OS 24.04 LTS on x86_64.

## References

- [React Native: Get Started](https://reactnative.dev/docs/getting-started)
- [Expo Camera](https://docs.expo.dev/versions/latest/sdk/camera/)
- [Expo SecureStore](https://docs.expo.dev/versions/latest/sdk/securestore/)
