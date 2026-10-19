# Linux agent

The agent is an early local development program. It collects system metrics and writes one JSON `metrics.snapshot` envelope to stdout immediately, then targets a two-second interval. Collection can take long enough that the actual interval is longer.

## Requirements

- Linux
- Go 1.26 or newer

## Run from the repository root

```sh
go run ./cmd/agent
```

To connect the agent to a relay running at the local development address, use a stable ID for this development session:

```sh
go run ./cmd/agent -relay-url ws://127.0.0.1:8765/ws -agent-id machineward-dev
```

Start `go run ./cmd/relay` in another terminal first. The agent registers, sends a metrics snapshot immediately, then targets the same two-second interval. It also answers the relay's typed probe messages. In relay mode, stdout is left empty and errors go to stderr.

To build a standalone executable:

```sh
go build -o machineward-agent ./cmd/agent
./machineward-agent
```

Press Ctrl+C to stop. The process also handles SIGTERM. Both signals stop the sampling loop through its context. Collection or encoding failures are written to stderr and cause a non-zero exit.

## Configuration and data handling

Standalone mode has no command-line flags, environment-variable settings, or configuration files. Relay mode accepts `-relay-url` and `-agent-id`; retry/reconnection is not implemented. The sampling interval is fixed at two seconds in `cmd/agent/main.go`.

Stdout is a metrics stream, not a diagnostic log. It includes CPU, memory, swap, and process data. Process records can include names, PIDs, memory use, and CPU use. Treat captured output as sensitive machine data. Avoid posting it publicly or sharing it without reviewing and redacting it first. Diagnostic errors go to stderr; the agent does not write credentials.

## Platform verification

The V0 target is Pop!_OS 24.04 LTS on x86_64. The development host observed on 2026-10-19 is Pop!_OS 22.04 LTS on x86_64. A successful build or local run on the development host does not establish compatibility with the Pop!_OS 24.04 target. Record a separate runtime check on that target before claiming support.

The relay connection is development-only. The relay binds to loopback and does not pair or authenticate clients. Do not expose it beyond the local machine. The agent is not ready for external installation or production use. TLS, pairing, authentication, reconnection, and remote actions are not implemented.
