package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/emanncode/machineward/protocol"
)

type testCollector struct {
	snapshot protocol.MetricsSnapshot
	calls    int
	onCall   func()
}

func (c *testCollector) Collect(context.Context) (protocol.MetricsSnapshot, error) {
	c.calls++
	if c.onCall != nil {
		c.onCall()
	}
	return c.snapshot, nil
}

func TestRunPreservesStandaloneJSONOutput(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	collector := &testCollector{snapshot: protocol.MetricsSnapshot{
		CollectedAt: time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC),
		CPU:         protocol.CPU{TotalPercent: 12, PerCorePercent: []float64{12}},
		Processes:   []protocol.ProcessMetric{},
	}, onCall: cancel}
	var output bytes.Buffer
	err := run(ctx, collector, &output)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("run() error = %v, want context.Canceled", err)
	}
	var message protocol.Envelope
	if err := json.Unmarshal(output.Bytes(), &message); err != nil {
		t.Fatalf("stdout is not JSON: %v", err)
	}
	if message.Type != protocol.TypeMetricsSnapshot {
		t.Fatalf("stdout message type = %q, want %q", message.Type, protocol.TypeMetricsSnapshot)
	}
	if collector.calls != 1 {
		t.Fatalf("collector calls = %d, want 1", collector.calls)
	}
}

func TestRunRelayRegistersSendsMetricsAndAcknowledgesProbe(t *testing.T) {
	type observed struct {
		hello   protocol.AgentHello
		metrics protocol.Envelope
		next    protocol.Envelope
		probeID string
		ack     protocol.Envelope
		err     error
	}
	observedMessages := make(chan observed, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			observedMessages <- observed{err: err}
			return
		}
		defer conn.CloseNow()
		read := func() (protocol.Envelope, error) {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			messageType, data, err := conn.Read(ctx)
			if err != nil {
				return protocol.Envelope{}, err
			}
			if messageType != websocket.MessageText {
				return protocol.Envelope{}, errors.New("expected text message")
			}
			return protocol.DecodeMessage(data)
		}
		first, err := read()
		if err != nil {
			observedMessages <- observed{err: err}
			return
		}
		var hello protocol.AgentHello
		if err := json.Unmarshal(first.Payload, &hello); err != nil {
			observedMessages <- observed{err: err}
			return
		}
		metrics, err := read()
		if err != nil {
			observedMessages <- observed{err: err}
			return
		}
		probe, err := protocol.NewEnvelope(protocol.TypeRelayProbe, protocol.RelayProbe{Text: "health check"})
		if err != nil {
			observedMessages <- observed{err: err}
			return
		}
		probeBytes, err := json.Marshal(probe)
		if err != nil {
			observedMessages <- observed{err: err}
			return
		}
		if err := conn.Write(r.Context(), websocket.MessageText, probeBytes); err != nil {
			observedMessages <- observed{err: err}
			return
		}
		ack, err := read()
		if err != nil {
			observedMessages <- observed{err: err}
			return
		}
		next, err := read()
		observedMessages <- observed{hello: hello, metrics: metrics, next: next, probeID: probe.ID, ack: ack, err: err}
		<-r.Context().Done()
	}))
	defer server.Close()
	endpoint := "ws" + strings.TrimPrefix(server.URL, "http")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	collector := &testCollector{snapshot: protocol.MetricsSnapshot{
		CollectedAt: time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC),
		CPU:         protocol.CPU{TotalPercent: 20, PerCorePercent: []float64{20}},
		Processes:   []protocol.ProcessMetric{},
	}}
	runErr := make(chan error, 1)
	go func() { runErr <- runRelay(ctx, collector, endpoint, "machine-1") }()

	select {
	case got := <-observedMessages:
		if got.err != nil {
			t.Fatalf("relay exchange: %v", got.err)
		}
		if got.hello.AgentID != "machine-1" || got.hello.OS != runtime.GOOS || got.hello.Arch != runtime.GOARCH || got.hello.AgentVersion != agentVersion {
			t.Fatalf("agent registration = %+v", got.hello)
		}
		if got.metrics.Type != protocol.TypeMetricsSnapshot {
			t.Fatalf("first data message = %q, want metrics snapshot", got.metrics.Type)
		}
		if got.next.Type != protocol.TypeMetricsSnapshot || got.next.ID == got.metrics.ID {
			t.Fatalf("next data message = (%q, %q), want a new metrics snapshot", got.next.Type, got.next.ID)
		}
		if got.ack.Type != protocol.TypeRelayProbeAck || got.ack.ID != got.probeID {
			t.Fatalf("probe acknowledgement = (%q, %q), want correlation ID %q", got.ack.Type, got.ack.ID, got.probeID)
		}
		cancel()
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for agent relay exchange")
	}
	select {
	case err := <-runErr:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("runRelay() error = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runRelay did not stop after cancellation")
	}
}

func TestRunRelayReportsConnectionFailure(t *testing.T) {
	collector := &testCollector{}
	err := runRelay(context.Background(), collector, "ws://127.0.0.1:1/ws", "machine-1")
	if err == nil || !strings.Contains(err.Error(), "connect to relay") {
		t.Fatalf("runRelay() error = %v, want connection error", err)
	}
	if collector.calls != 0 {
		t.Fatalf("collector calls = %d, want 0 after connection failure", collector.calls)
	}
}

func TestRunRelayReturnsWhenServerClosesConnection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		for range 2 { // agent.hello followed by its first metrics snapshot
			if _, _, err := conn.Read(ctx); err != nil {
				return
			}
		}
	}))
	defer server.Close()
	endpoint := "ws" + strings.TrimPrefix(server.URL, "http")
	collector := &testCollector{snapshot: protocol.MetricsSnapshot{
		CollectedAt: time.Now().UTC(), CPU: protocol.CPU{PerCorePercent: []float64{}}, Processes: []protocol.ProcessMetric{},
	}}
	err := runRelay(context.Background(), collector, endpoint, "machine-1")
	if err == nil || !strings.Contains(err.Error(), "read relay message") {
		t.Fatalf("runRelay() error = %v, want relay-closure error", err)
	}
}
