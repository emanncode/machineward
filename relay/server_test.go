package relay

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/emanncode/machineward/protocol"
)

func newTestRelay(t *testing.T) (*Server, string) {
	t.Helper()
	server := NewServer()
	httpServer := httptest.NewServer(server)
	t.Cleanup(httpServer.Close)
	return server, "ws" + strings.TrimPrefix(httpServer.URL, "http")
}

func dial(t *testing.T, endpoint string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.Dial(context.Background(), endpoint, nil)
	if err != nil {
		t.Fatalf("connect WebSocket: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test complete") })
	return conn
}

func sendEnvelope(t *testing.T, conn *websocket.Conn, messageType string, payload any) protocol.Envelope {
	t.Helper()
	message, err := protocol.NewEnvelope(messageType, payload)
	if err != nil {
		t.Fatalf("create message: %v", err)
	}
	data, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("encode message: %v", err)
	}
	if err := conn.Write(context.Background(), websocket.MessageText, data); err != nil {
		t.Fatalf("send message: %v", err)
	}
	return message
}

func readEnvelope(t *testing.T, conn *websocket.Conn) protocol.Envelope {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	typ, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read message: %v", err)
	}
	if typ != websocket.MessageText {
		t.Fatalf("message type = %v, want text", typ)
	}
	message, err := protocol.DecodeMessage(data)
	if err != nil {
		t.Fatalf("decode received message: %v", err)
	}
	return message
}

func TestServerRoutesPhoneProbeAndAgentUpdates(t *testing.T) {
	server, endpoint := newTestRelay(t)
	agent := dial(t, endpoint)
	sendEnvelope(t, agent, protocol.TypeAgentHello, protocol.AgentHello{
		AgentID: "agent-1", AgentVersion: "0.1.0", OS: "linux", Arch: "amd64",
	})
	waitFor(t, func() bool {
		server.mu.Lock()
		defer server.mu.Unlock()
		return server.agents["agent-1"] != nil
	}, "agent registration")
	phone := dial(t, endpoint)
	sendEnvelope(t, phone, protocol.TypePhoneHello, protocol.PhoneHello{PhoneID: "phone-1", AgentID: "agent-1"})
	waitFor(t, func() bool {
		server.mu.Lock()
		defer server.mu.Unlock()
		return server.phones["agent-1"] != nil
	}, "phone registration")

	probe := sendEnvelope(t, phone, protocol.TypeRelayProbe, protocol.RelayProbe{Text: "ping"})
	if got := readEnvelope(t, agent); got.ID != probe.ID || got.Type != protocol.TypeRelayProbe {
		t.Fatalf("routed probe = (%s, %s), want (%s, %s)", got.ID, got.Type, probe.ID, protocol.TypeRelayProbe)
	}

	snapshot := sendEnvelope(t, agent, protocol.TypeMetricsSnapshot, protocol.MetricsSnapshot{
		CollectedAt: time.Now().UTC(),
		CPU:         protocol.CPU{PerCorePercent: []float64{}},
		Processes:   []protocol.ProcessMetric{},
	})
	if got := readEnvelope(t, phone); got.ID != snapshot.ID || got.Type != protocol.TypeMetricsSnapshot {
		t.Fatalf("routed snapshot = (%s, %s), want (%s, %s)", got.ID, got.Type, snapshot.ID, protocol.TypeMetricsSnapshot)
	}

	ack := sendEnvelope(t, agent, protocol.TypeRelayProbeAck, protocol.RelayProbe{Text: "pong"})
	if got := readEnvelope(t, phone); got.ID != ack.ID || got.Type != protocol.TypeRelayProbeAck {
		t.Fatalf("routed acknowledgement = (%s, %s), want (%s, %s)", got.ID, got.Type, ack.ID, protocol.TypeRelayProbeAck)
	}
}

func TestPhoneMustRegisterAnOnlineAgent(t *testing.T) {
	_, endpoint := newTestRelay(t)
	phone := dial(t, endpoint)
	sendEnvelope(t, phone, protocol.TypePhoneHello, protocol.PhoneHello{PhoneID: "phone-1", AgentID: "missing"})
	if got := readEnvelope(t, phone); got.Type != protocol.TypeError {
		t.Fatalf("registration response type = %q, want error", got.Type)
	}
}

func TestServerRejectsDuplicateAgentRegistration(t *testing.T) {
	server, endpoint := newTestRelay(t)
	first := dial(t, endpoint)
	second := dial(t, endpoint)
	hello := protocol.AgentHello{AgentID: "agent-1", AgentVersion: "0.1.0", OS: "linux", Arch: "amd64"}
	sendEnvelope(t, first, protocol.TypeAgentHello, hello)
	waitFor(t, func() bool {
		server.mu.Lock()
		defer server.mu.Unlock()
		return server.agents[hello.AgentID] != nil
	}, "agent registration")
	sendEnvelope(t, second, protocol.TypeAgentHello, hello)
	if got := readEnvelope(t, second); got.Type != protocol.TypeError {
		t.Fatalf("duplicate registration response type = %q, want error", got.Type)
	}
}

func TestServerRejectsSecondPhoneForAgent(t *testing.T) {
	server, endpoint := newTestRelay(t)
	agent := dial(t, endpoint)
	sendEnvelope(t, agent, protocol.TypeAgentHello, protocol.AgentHello{
		AgentID: "agent-1", AgentVersion: "0.1.0", OS: "linux", Arch: "amd64",
	})
	waitFor(t, func() bool {
		server.mu.Lock()
		defer server.mu.Unlock()
		return server.agents["agent-1"] != nil
	}, "agent registration")
	firstPhone := dial(t, endpoint)
	sendEnvelope(t, firstPhone, protocol.TypePhoneHello, protocol.PhoneHello{PhoneID: "phone-1", AgentID: "agent-1"})
	waitFor(t, func() bool {
		server.mu.Lock()
		defer server.mu.Unlock()
		return server.phones["agent-1"] != nil
	}, "first phone registration")
	secondPhone := dial(t, endpoint)
	sendEnvelope(t, secondPhone, protocol.TypePhoneHello, protocol.PhoneHello{PhoneID: "phone-2", AgentID: "agent-1"})
	if got := readEnvelope(t, secondPhone); got.Type != protocol.TypeError {
		t.Fatalf("duplicate phone response type = %q, want error", got.Type)
	}
}

func TestServerRejectsMessagesFromWrongRole(t *testing.T) {
	server, endpoint := newTestRelay(t)
	phone := dial(t, endpoint)
	agent := dial(t, endpoint)
	sendEnvelope(t, agent, protocol.TypeAgentHello, protocol.AgentHello{
		AgentID: "agent-1", AgentVersion: "0.1.0", OS: "linux", Arch: "amd64",
	})
	waitFor(t, func() bool {
		server.mu.Lock()
		defer server.mu.Unlock()
		return server.agents["agent-1"] != nil
	}, "agent registration")
	sendEnvelope(t, phone, protocol.TypePhoneHello, protocol.PhoneHello{PhoneID: "phone-1", AgentID: "agent-1"})
	waitFor(t, func() bool {
		server.mu.Lock()
		defer server.mu.Unlock()
		return server.phones["agent-1"] != nil
	}, "phone registration")
	sendEnvelope(t, phone, protocol.TypeMetricsSnapshot, protocol.MetricsSnapshot{
		CollectedAt: time.Now().UTC(), CPU: protocol.CPU{PerCorePercent: []float64{}}, Processes: []protocol.ProcessMetric{},
	})
	if got := readEnvelope(t, phone); got.Type != protocol.TypeError {
		t.Fatalf("wrong-role response type = %q, want error", got.Type)
	}
}

func TestDisconnectReleasesAgentRegistration(t *testing.T) {
	server, endpoint := newTestRelay(t)
	agent := dial(t, endpoint)
	hello := protocol.AgentHello{AgentID: "agent-1", AgentVersion: "0.1.0", OS: "linux", Arch: "amd64"}
	sendEnvelope(t, agent, protocol.TypeAgentHello, hello)
	_ = agent.Close(websocket.StatusNormalClosure, "disconnect")

	deadline := time.Now().Add(2 * time.Second)
	for {
		server.mu.Lock()
		_, exists := server.agents[hello.AgentID]
		server.mu.Unlock()
		if !exists {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("disconnected agent remained registered")
		}
		time.Sleep(10 * time.Millisecond)
	}
	second := dial(t, endpoint)
	sendEnvelope(t, second, protocol.TypeAgentHello, hello)
	waitFor(t, func() bool {
		server.mu.Lock()
		defer server.mu.Unlock()
		return server.agents[hello.AgentID] != nil
	}, "replacement agent registration")
	phone := dial(t, endpoint)
	sendEnvelope(t, phone, protocol.TypePhoneHello, protocol.PhoneHello{PhoneID: "phone-1", AgentID: "agent-1"})
	waitFor(t, func() bool {
		server.mu.Lock()
		defer server.mu.Unlock()
		return server.phones[hello.AgentID] != nil
	}, "phone registration")
	if got := sendEnvelope(t, phone, protocol.TypeRelayProbe, protocol.RelayProbe{Text: "ping"}); got.Type != protocol.TypeRelayProbe {
		t.Fatal("probe was not created")
	}
	if got := readEnvelope(t, second); got.Type != protocol.TypeRelayProbe {
		t.Fatalf("probe recipient got %q, want %q", got.Type, protocol.TypeRelayProbe)
	}
}

func waitFor(t *testing.T, condition func() bool, description string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", description)
}

func TestCloseConnectionsClosesActiveWebSockets(t *testing.T) {
	server, endpoint := newTestRelay(t)
	agent := dial(t, endpoint)
	sendEnvelope(t, agent, protocol.TypeAgentHello, protocol.AgentHello{
		AgentID: "agent-1", AgentVersion: "0.1.0", OS: "linux", Arch: "amd64",
	})
	if err := server.CloseConnections(); err != nil {
		t.Fatalf("close relay connections: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, _, err := agent.Read(ctx); err == nil {
		t.Fatal("WebSocket remained open after relay shutdown")
	}
}

func TestServerRejectsMalformedMessage(t *testing.T) {
	_, endpoint := newTestRelay(t)
	agent := dial(t, endpoint)
	if err := agent.Write(context.Background(), websocket.MessageText, []byte(`{"bad":true}`)); err != nil {
		t.Fatalf("send malformed message: %v", err)
	}
	if got := readEnvelope(t, agent); got.Type != protocol.TypeError {
		t.Fatalf("malformed-message response type = %q, want error", got.Type)
	}
}
