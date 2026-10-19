package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/coder/websocket"
	"github.com/emanncode/machineward/agent/metrics"
	"github.com/emanncode/machineward/protocol"
)

const sampleInterval = 2 * time.Second
const agentVersion = "0.1.0"

type snapshotCollector interface {
	Collect(context.Context) (protocol.MetricsSnapshot, error)
}

func main() {
	var relayURL string
	var agentID string
	flag.StringVar(&relayURL, "relay-url", "", "connect to the relay at this WebSocket URL (for example ws://127.0.0.1:8765/ws)")
	flag.StringVar(&agentID, "agent-id", "", "agent identity to register when using relay mode")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	collector := metrics.NewCollector()
	var err error
	if relayURL != "" {
		if agentID == "" {
			fmt.Fprintln(os.Stderr, "-agent-id is required when -relay-url is set")
			os.Exit(2)
		}
		err = runRelay(ctx, collector, relayURL, agentID)
	} else if agentID != "" {
		fmt.Fprintln(os.Stderr, "-agent-id requires -relay-url")
		os.Exit(2)
	} else {
		err = run(ctx, collector, os.Stdout)
	}
	if err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, collector snapshotCollector, output io.Writer) error {
	ticker := time.NewTicker(sampleInterval)
	defer ticker.Stop()

	for {
		if err := emitSnapshot(ctx, collector, output); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func emitSnapshot(ctx context.Context, collector snapshotCollector, output io.Writer) error {
	snapshot, err := collector.Collect(ctx)
	if err != nil {
		return err
	}
	message, err := protocol.NewEnvelope(protocol.TypeMetricsSnapshot, snapshot)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(output)
	return encoder.Encode(message)
}

func runRelay(ctx context.Context, collector snapshotCollector, relayURL, agentID string) error {
	conn, _, err := websocket.Dial(ctx, relayURL, nil)
	if err != nil {
		return fmt.Errorf("connect to relay %q: %w", relayURL, err)
	}
	defer conn.CloseNow()

	hello, err := protocol.NewEnvelope(protocol.TypeAgentHello, protocol.AgentHello{
		AgentID:      agentID,
		AgentVersion: agentVersion,
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
	})
	if err != nil {
		return err
	}
	if err := writeEnvelope(ctx, conn, hello); err != nil {
		return fmt.Errorf("register agent with relay: %w", err)
	}

	probes := make(chan protocol.Envelope, 4)
	readErrors := make(chan error, 1)
	go readRelayMessages(ctx, conn, probes, readErrors)

	ticker := time.NewTicker(sampleInterval)
	defer ticker.Stop()
	if err := sendMetricsSnapshot(ctx, collector, conn); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case readErr := <-readErrors:
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("read relay message: %w", readErr)
		case probe := <-probes:
			ack, ackErr := newProbeAck(probe)
			if ackErr != nil {
				return ackErr
			}
			if writeErr := writeEnvelope(ctx, conn, ack); writeErr != nil {
				return fmt.Errorf("send probe acknowledgement: %w", writeErr)
			}
		case <-ticker.C:
			if err := sendMetricsSnapshot(ctx, collector, conn); err != nil {
				return err
			}
		}
	}
}

func sendMetricsSnapshot(ctx context.Context, collector snapshotCollector, conn *websocket.Conn) error {
	snapshot, err := collector.Collect(ctx)
	if err != nil {
		return fmt.Errorf("collect metrics: %w", err)
	}
	message, err := protocol.NewEnvelope(protocol.TypeMetricsSnapshot, snapshot)
	if err != nil {
		return err
	}
	if err := writeEnvelope(ctx, conn, message); err != nil {
		return fmt.Errorf("send metrics snapshot: %w", err)
	}
	return nil
}

func newProbeAck(probe protocol.Envelope) (protocol.Envelope, error) {
	payload, err := json.Marshal(protocol.RelayProbe{Text: "pong"})
	if err != nil {
		return protocol.Envelope{}, fmt.Errorf("encode probe acknowledgement: %w", err)
	}
	return protocol.Envelope{
		Version: protocol.Version,
		ID:      probe.ID,
		Type:    protocol.TypeRelayProbeAck,
		SentAt:  time.Now().UTC(),
		Payload: payload,
	}, nil
}

func readRelayMessages(ctx context.Context, conn *websocket.Conn, probes chan<- protocol.Envelope, readErrors chan<- error) {
	for {
		messageType, data, err := conn.Read(ctx)
		if err != nil {
			select {
			case readErrors <- err:
			case <-ctx.Done():
			}
			return
		}
		if messageType != websocket.MessageText {
			select {
			case readErrors <- fmt.Errorf("relay sent a non-text message"):
			case <-ctx.Done():
			}
			return
		}
		message, err := protocol.DecodeMessage(data)
		if err != nil {
			select {
			case readErrors <- fmt.Errorf("relay sent an invalid protocol message: %w", err):
			case <-ctx.Done():
			}
			return
		}
		if message.Type != protocol.TypeRelayProbe {
			select {
			case readErrors <- fmt.Errorf("relay sent unsupported message type %q", message.Type):
			case <-ctx.Done():
			}
			return
		}
		select {
		case probes <- message:
		case <-ctx.Done():
			return
		}
	}
}

func writeEnvelope(ctx context.Context, conn *websocket.Conn, message protocol.Envelope) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("encode protocol message: %w", err)
	}
	return conn.Write(ctx, websocket.MessageText, data)
}
