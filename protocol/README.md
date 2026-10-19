# Machineward protocol

`v1/schema.json` defines the JSON envelope shared by the Go agent, Go relay, and mobile app.

The Go decoder embeds and compiles this schema once, then validates every incoming message against it before decoding the typed payload. Keep required fields, types, enums, bounds, and unknown-field rules in the schema so validation does not drift between Go and other clients. Go's timestamp parsing and UUID check enforce those formats because the selected schema validator treats `format` as an annotation.

The first contract slice includes agent registration, typed relay probes, and the V0 metrics snapshot. It is for local development only. It does not define authentication, pairing, command, or action-result messages yet. Do not expose a relay using this contract to an untrusted network.

## Envelope

Every message has:

- `version`: protocol version, currently `1`
- `id`: UUID for message correlation
- `type`: namespaced message type
- `sent_at`: RFC 3339 timestamp
- `payload`: object validated against the message type

## Initial message types

- `agent.hello`: agent identity and platform information
- `relay.probe`: local routing smoke message
- `relay.probe_ack`: relay response for a probe
- `metrics.snapshot`: current CPU, memory, swap, and process information. `processes_skipped` reports per-PID inspection failures encountered by the agent. Process exits during collection are expected and are not counted. The OS can still hide processes from enumeration, so zero does not guarantee that every system process was visible.
- `error`: bounded machine-readable error response

### Metrics sampling notes

- The agent uses one collector and calls it serially. Host CPU deltas use gopsutil's package-level baseline, so another caller in the same process would advance that baseline too.
- Host total and per-core CPU values are sampled in separate reads. They are close in time, but not one atomic snapshot.
- The agent emits a snapshot immediately, then targets a two-second cadence. Slow collection can make the actual interval longer.
- A process's first successful CPU read establishes its baseline and reports `0%`. Later reads report CPU use relative to one logical CPU, so a multi-threaded process can exceed `100%`.
- Process rows are best effort. If a PID exits during a read, its row is omitted without incrementing `processes_skipped`. Other per-PID inspection failures omit the row and increment the field. Rows are sorted by CPU use and capped at 25; intentional top-25 truncation does not increment the skipped count.

Pairing, session authentication, authorization, action requests, and action results must be specified before any command handling is implemented.
