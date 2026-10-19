package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emanncode/machineward/relay"
)

const listenAddress = "127.0.0.1:8765"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	relayServer := relay.NewServer()
	mux := http.NewServeMux()
	mux.Handle("/ws", relayServer)
	server := &http.Server{
		Addr:              listenAddress,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	listener, err := listen(listenAddress, os.Stdout)
	if err != nil {
		return err
	}
	defer listener.Close()

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(listener)
	}()

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("run local relay: %w", err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	shutdownErr := server.Shutdown(shutdownCtx)
	connectionsErr := relayServer.CloseConnections()
	if shutdownErr != nil {
		return fmt.Errorf("shut down HTTP server: %w", shutdownErr)
	}
	if connectionsErr != nil {
		return fmt.Errorf("close relay connections: %w", connectionsErr)
	}
	return nil
}

func listen(address string, output io.Writer) (net.Listener, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("run local relay: %w", err)
	}
	fmt.Fprintf(output, "Machineward local relay listening at ws://%s/ws (loopback only; no pairing or authentication)\n", address)
	return listener, nil
}
