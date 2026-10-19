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
- `metrics.snapshot`: current CPU, memory, swap, and process information
- `error`: bounded machine-readable error response

Pairing, session authentication, authorization, action requests, and action results must be specified before any command handling is implemented.
