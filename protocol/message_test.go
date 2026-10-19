package protocol

import (
	"fmt"
	"strings"
	"testing"
)

const testMessageID = "d9428888-122b-4f8f-8f0c-1c6f4f0d8f0a"
const testTimestamp = "2026-10-19T12:00:00Z"

func encodeTestMessage(messageType, payload string) []byte {
	return []byte(fmt.Sprintf(`{"version":1,"id":%q,"type":%q,"sent_at":%q,"payload":%s}`, testMessageID, messageType, testTimestamp, payload))
}

func TestDecodeMessageAcceptsEveryV1MessageType(t *testing.T) {
	tests := []struct {
		name        string
		messageType string
		payload     string
	}{
		{"agent hello", TypeAgentHello, `{"agent_id":"agent-1","agent_version":"0.1.0","os":"linux","arch":"amd64"}`},
		{"relay probe", TypeRelayProbe, `{"text":"ping"}`},
		{"relay probe acknowledgement", TypeRelayProbeAck, `{"text":"pong"}`},
		{"metrics snapshot", TypeMetricsSnapshot, `{"collected_at":"2026-10-19T12:00:00Z","cpu":{"total_percent":12.5,"per_core_percent":[10,15]},"memory":{"total_bytes":16000,"used_bytes":8000,"used_percent":50},"swap":{"total_bytes":4000,"used_bytes":0,"used_percent":0},"processes":[{"pid":42,"name":"editor","cpu_percent":4.25,"memory_bytes":2000,"started_at_unix_ms":1729340000000}]}`},
		{"error", TypeError, `{"code":"unavailable","message":"The process is no longer running."}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := DecodeMessage(encodeTestMessage(test.messageType, test.payload)); err != nil {
				t.Fatalf("DecodeMessage() error = %v", err)
			}
		})
	}
}

func TestDecodeMessageRejectsInvalidV1Payloads(t *testing.T) {
	tests := []struct {
		name        string
		messageType string
		payload     string
	}{
		{"agent hello missing required field", TypeAgentHello, `{"agent_id":"agent-1","agent_version":"0.1.0","os":"linux"}`},
		{"agent hello invalid platform enum", TypeAgentHello, `{"agent_id":"agent-1","agent_version":"0.1.0","os":"windows","arch":"amd64"}`},
		{"agent hello string exceeds maximum", TypeAgentHello, `{"agent_id":"` + repeat("a", 129) + `","agent_version":"0.1.0","os":"linux","arch":"amd64"}`},
		{"agent version exceeds maximum", TypeAgentHello, `{"agent_id":"agent-1","agent_version":"` + repeat("a", 65) + `","os":"linux","arch":"amd64"}`},
		{"agent hello unknown field", TypeAgentHello, `{"agent_id":"agent-1","agent_version":"0.1.0","os":"linux","arch":"amd64","extra":true}`},
		{"probe missing required text", TypeRelayProbe, `{}`},
		{"probe text exceeds maximum", TypeRelayProbe, `{"text":"` + repeat("a", 257) + `"}`},
		{"probe acknowledgement wrong field type", TypeRelayProbeAck, `{"text":42}`},
		{"metrics missing required process list", TypeMetricsSnapshot, `{"collected_at":"2026-10-19T12:00:00Z","cpu":{"total_percent":10,"per_core_percent":[]},"memory":{"total_bytes":100,"used_bytes":50,"used_percent":50},"swap":{"total_bytes":0,"used_bytes":0,"used_percent":0}}`},
		{"metrics CPU above percentage bound", TypeMetricsSnapshot, `{"collected_at":"2026-10-19T12:00:00Z","cpu":{"total_percent":101,"per_core_percent":[]},"memory":{"total_bytes":100,"used_bytes":50,"used_percent":50},"swap":{"total_bytes":0,"used_bytes":0,"used_percent":0},"processes":[]}`},
		{"metrics per-core CPU below percentage bound", TypeMetricsSnapshot, `{"collected_at":"2026-10-19T12:00:00Z","cpu":{"total_percent":10,"per_core_percent":[-1]},"memory":{"total_bytes":100,"used_bytes":50,"used_percent":50},"swap":{"total_bytes":0,"used_bytes":0,"used_percent":0},"processes":[]}`},
		{"metrics memory above percentage bound", TypeMetricsSnapshot, `{"collected_at":"2026-10-19T12:00:00Z","cpu":{"total_percent":10,"per_core_percent":[]},"memory":{"total_bytes":100,"used_bytes":50,"used_percent":101},"swap":{"total_bytes":0,"used_bytes":0,"used_percent":0},"processes":[]}`},
		{"metrics process PID below minimum", TypeMetricsSnapshot, `{"collected_at":"2026-10-19T12:00:00Z","cpu":{"total_percent":10,"per_core_percent":[]},"memory":{"total_bytes":100,"used_bytes":50,"used_percent":50},"swap":{"total_bytes":0,"used_bytes":0,"used_percent":0},"processes":[{"pid":0,"name":"editor","cpu_percent":0,"memory_bytes":10,"started_at_unix_ms":1}]}`},
		{"metrics process negative start time", TypeMetricsSnapshot, `{"collected_at":"2026-10-19T12:00:00Z","cpu":{"total_percent":10,"per_core_percent":[]},"memory":{"total_bytes":100,"used_bytes":50,"used_percent":50},"swap":{"total_bytes":0,"used_bytes":0,"used_percent":0},"processes":[{"pid":42,"name":"editor","cpu_percent":0,"memory_bytes":10,"started_at_unix_ms":-1}]}`},
		{"metrics process name exceeds maximum", TypeMetricsSnapshot, `{"collected_at":"2026-10-19T12:00:00Z","cpu":{"total_percent":10,"per_core_percent":[]},"memory":{"total_bytes":100,"used_bytes":50,"used_percent":50},"swap":{"total_bytes":0,"used_bytes":0,"used_percent":0},"processes":[{"pid":42,"name":"` + repeat("a", 257) + `","cpu_percent":0,"memory_bytes":10,"started_at_unix_ms":1}]}`},
		{"metrics invalid collection timestamp", TypeMetricsSnapshot, `{"collected_at":"sometime","cpu":{"total_percent":10,"per_core_percent":[]},"memory":{"total_bytes":100,"used_bytes":50,"used_percent":50},"swap":{"total_bytes":0,"used_bytes":0,"used_percent":0},"processes":[]}`},
		{"error empty code", TypeError, `{"code":"","message":"unavailable"}`},
		{"error message exceeds maximum", TypeError, `{"code":"unavailable","message":"` + repeat("a", 513) + `"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := DecodeMessage(encodeTestMessage(test.messageType, test.payload)); err == nil {
				t.Fatal("DecodeMessage() accepted invalid payload")
			}
		})
	}
}

func TestDecodeMessageRejectsInvalidEnvelope(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"invalid UUID", []byte(fmt.Sprintf(`{"version":1,"id":"not-a-uuid","type":%q,"sent_at":%q,"payload":{"text":"ping"}}`, TypeRelayProbe, testTimestamp))},
		{"UUID does not match lowercase contract", []byte(fmt.Sprintf(`{"version":1,"id":"D9428888-122B-4F8F-8F0C-1C6F4F0D8F0A","type":%q,"sent_at":%q,"payload":{"text":"ping"}}`, TypeRelayProbe, testTimestamp))},
		{"invalid timestamp format", []byte(fmt.Sprintf(`{"version":1,"id":%q,"type":%q,"sent_at":"yesterday","payload":{"text":"ping"}}`, testMessageID, TypeRelayProbe))},
		{"unsupported version", []byte(fmt.Sprintf(`{"version":2,"id":%q,"type":%q,"sent_at":%q,"payload":{"text":"ping"}}`, testMessageID, TypeRelayProbe, testTimestamp))},
		{"unknown message type", []byte(fmt.Sprintf(`{"version":1,"id":%q,"type":"unknown","sent_at":%q,"payload":{"text":"ping"}}`, testMessageID, testTimestamp))},
		{"unknown envelope field", []byte(fmt.Sprintf(`{"version":1,"id":%q,"type":%q,"sent_at":%q,"payload":{"text":"ping"},"extra":true}`, testMessageID, TypeRelayProbe, testTimestamp))},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := DecodeMessage(test.data); err == nil {
				t.Fatal("DecodeMessage() accepted invalid envelope")
			}
		})
	}
}

func repeat(value string, count int) string {
	return strings.Repeat(value, count)
}
