package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emanncode/machineward/agent/metrics"
	"github.com/emanncode/machineward/protocol"
)

const sampleInterval = 2 * time.Second

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	collector := metrics.NewCollector()
	if err := run(ctx, collector, os.Stdout); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, collector *metrics.Collector, output io.Writer) error {
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

func emitSnapshot(ctx context.Context, collector *metrics.Collector, output io.Writer) error {
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
