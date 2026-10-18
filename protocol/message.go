package protocol

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"time"
)

const Version = 1

const (
	TypeAgentHello      = "agent.hello"
	TypeRelayProbe      = "relay.probe"
	TypeRelayProbeAck   = "relay.probe_ack"
	TypeMetricsSnapshot = "metrics.snapshot"
	TypeError           = "error"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// Envelope is the versioned outer message used by all Machineward components.
type Envelope struct {
	Version int             `json:"version"`
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	SentAt  time.Time       `json:"sent_at"`
	Payload json.RawMessage `json:"payload"`
}

type AgentHello struct {
	AgentID      string `json:"agent_id"`
	AgentVersion string `json:"agent_version"`
	OS           string `json:"os"`
	Arch         string `json:"arch"`
}

type RelayProbe struct {
	Text string `json:"text"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Memory struct {
	TotalBytes  uint64  `json:"total_bytes"`
	UsedBytes   uint64  `json:"used_bytes"`
	UsedPercent float64 `json:"used_percent"`
}

type CPU struct {
	TotalPercent   float64   `json:"total_percent"`
	PerCorePercent []float64 `json:"per_core_percent"`
}

type ProcessMetric struct {
	PID             int32   `json:"pid"`
	Name            string  `json:"name"`
	CPUPercent      float64 `json:"cpu_percent"`
	MemoryBytes     uint64  `json:"memory_bytes"`
	StartedAtUnixMS int64   `json:"started_at_unix_ms"`
}

type MetricsSnapshot struct {
	CollectedAt time.Time       `json:"collected_at"`
	CPU         CPU             `json:"cpu"`
	Memory      Memory          `json:"memory"`
	Swap        Memory          `json:"swap"`
	Processes   []ProcessMetric `json:"processes"`
}

// NewEnvelope creates a message with a random UUID and the current UTC time.
func NewEnvelope(messageType string, payload any) (Envelope, error) {
	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, fmt.Errorf("encode %s payload: %w", messageType, err)
	}

	var randomBytes [16]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		return Envelope{}, fmt.Errorf("create message ID: %w", err)
	}
	randomBytes[6] = (randomBytes[6] & 0x0f) | 0x40
	randomBytes[8] = (randomBytes[8] & 0x3f) | 0x80
	encodedID := make([]byte, hex.EncodedLen(len(randomBytes)))
	hex.Encode(encodedID, randomBytes[:])
	id := fmt.Sprintf("%s-%s-%s-%s-%s", encodedID[0:8], encodedID[8:12], encodedID[12:16], encodedID[16:20], encodedID[20:32])

	return Envelope{
		Version: Version,
		ID:      id,
		Type:    messageType,
		SentAt:  time.Now().UTC(),
		Payload: encodedPayload,
	}, nil
}

// DecodeMessage parses and validates the versioned envelope and payload.
func DecodeMessage(data []byte) (Envelope, error) {
	var message Envelope
	if err := decodeStrict(data, &message); err != nil {
		return Envelope{}, fmt.Errorf("decode envelope: %w", err)
	}
	if message.Version != Version {
		return Envelope{}, fmt.Errorf("unsupported protocol version %d", message.Version)
	}
	if !uuidPattern.MatchString(message.ID) {
		return Envelope{}, fmt.Errorf("invalid message ID")
	}
	if message.SentAt.IsZero() {
		return Envelope{}, fmt.Errorf("missing sent_at timestamp")
	}
	if len(message.Payload) == 0 || !json.Valid(message.Payload) {
		return Envelope{}, fmt.Errorf("missing or invalid payload")
	}

	var payload any
	switch message.Type {
	case TypeAgentHello:
		payload = &AgentHello{}
	case TypeRelayProbe, TypeRelayProbeAck:
		payload = &RelayProbe{}
	case TypeMetricsSnapshot:
		payload = &MetricsSnapshot{}
	case TypeError:
		payload = &ErrorPayload{}
	default:
		return Envelope{}, fmt.Errorf("unknown message type %q", message.Type)
	}
	if err := decodeStrict(message.Payload, payload); err != nil {
		return Envelope{}, fmt.Errorf("decode %s payload: %w", message.Type, err)
	}

	return message, nil
}

func decodeStrict(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON value")
		}
		return err
	}
	return nil
}
