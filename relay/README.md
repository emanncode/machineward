# Local relay

The relay is the local development bridge between one Machineward Linux agent
and one phone connection. It accepts WebSockets at `ws://127.0.0.1:8765/ws`.

Start it from the repository root:

```sh
go run ./cmd/relay
```

The first message registers a connection. An agent sends `agent.hello`; a phone
sends `phone.hello` with its own ID and the agent ID it wants to connect to.
Phones can send `relay.probe`. The agent can send `relay.probe_ack` and
`metrics.snapshot`. The relay forwards those typed envelopes without changing
their IDs or payloads. Other message types are rejected for that connection.

This is a development skeleton, not a remotely usable service. It binds only to
loopback and has no pairing, authentication, TLS, persistence, reconnection, or
process actions. Do not expose it to a network or treat a phone registration as
proof of identity. Metrics include process names and resource usage, so even
local snapshots can reveal information about the machine.

The 24.04 Pop!_OS target check is still pending. Building and running the relay
on another Linux host does not establish that target-platform support.
