# Linux agent

The agent is an early local development program. It collects system metrics and writes one JSON `metrics.snapshot` envelope to stdout immediately, then targets a two-second interval. Collection can take long enough that the actual interval is longer.

## Requirements

- Linux
- Go 1.26 or newer

## Run from the repository root

```sh
go run ./cmd/agent
```

To build a standalone executable:

```sh
go build -o machineward-agent ./cmd/agent
./machineward-agent
```

Press Ctrl+C to stop. The process also handles SIGTERM. Both signals stop the sampling loop through its context. Collection or encoding failures are written to stderr and cause a non-zero exit.

## Configuration and data handling

There are no command-line flags, environment-variable settings, configuration files, network listeners, or remote commands in this version. The sampling interval is fixed at two seconds in `cmd/agent/main.go`.

Stdout is a metrics stream, not a diagnostic log. It includes CPU, memory, swap, and process data. Process records can include names, PIDs, memory use, and CPU use. Treat captured output as sensitive machine data. Avoid posting it publicly or sharing it without reviewing and redacting it first. Diagnostic errors go to stderr; the agent does not write credentials.

## Platform verification

The V0 target is Pop!_OS 24.04 LTS on x86_64. The development host observed on 2026-10-19 is Pop!_OS 22.04 LTS on x86_64. A successful build or local run on the development host does not establish compatibility with the Pop!_OS 24.04 target. Record a separate runtime check on that target before claiming support.

The agent is not ready for external installation or production use. Pairing, transport, authentication, and remote actions are not implemented.
