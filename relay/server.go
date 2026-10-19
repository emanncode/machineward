// Package relay provides the local development relay for Machineward.
package relay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/coder/websocket"
	"github.com/emanncode/machineward/protocol"
)

const maxMessageBytes = 1 << 20

type role uint8

const (
	agentRole role = iota + 1
	phoneRole
)

type peer struct {
	conn    *websocket.Conn
	role    role
	agentID string
	phoneID string
}

// Server tracks active local agent and phone WebSocket connections.
// Each agent may have one phone connected at a time.
type Server struct {
	mu       sync.Mutex
	writeMu  sync.Mutex
	agents   map[string]*peer
	phones   map[string]*peer // keyed by the registered agent ID
	phoneIDs map[string]*peer
	conns    map[*websocket.Conn]struct{}
}

func NewServer() *Server {
	return &Server{
		agents:   make(map[string]*peer),
		phones:   make(map[string]*peer),
		phoneIDs: make(map[string]*peer),
		conns:    make(map[*websocket.Conn]struct{}),
	}
}

// ServeHTTP accepts WebSocket connections at the handler's mounted path.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	conn.SetReadLimit(maxMessageBytes)
	s.mu.Lock()
	s.conns[conn] = struct{}{}
	s.mu.Unlock()
	defer s.remove(conn)

	ctx := r.Context()
	messageType, data, err := conn.Read(ctx)
	if err != nil {
		return
	}
	if messageType != websocket.MessageText {
		s.reject(ctx, conn, "text_required", "Only text JSON messages are supported.")
		return
	}
	message, err := protocol.DecodeMessage(data)
	if err != nil {
		s.reject(ctx, conn, "invalid_message", "The message does not match protocol v1.")
		return
	}
	registered, err := s.register(conn, message)
	if err != nil {
		s.reject(ctx, conn, "registration_failed", err.Error())
		return
	}

	for {
		messageType, data, readErr := conn.Read(ctx)
		if readErr != nil {
			return
		}
		if messageType != websocket.MessageText {
			s.reject(ctx, conn, "text_required", "Only text JSON messages are supported.")
			return
		}
		message, decodeErr := protocol.DecodeMessage(data)
		if decodeErr != nil {
			s.reject(ctx, conn, "invalid_message", "The message does not match protocol v1.")
			return
		}
		if routeErr := s.route(ctx, registered, data, message); routeErr != nil {
			s.reject(ctx, conn, "message_not_allowed", routeErr.Error())
			return
		}
	}
}

func (s *Server) register(conn *websocket.Conn, message protocol.Envelope) (*peer, error) {
	registered := &peer{conn: conn}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch message.Type {
	case protocol.TypeAgentHello:
		var hello protocol.AgentHello
		if err := json.Unmarshal(message.Payload, &hello); err != nil {
			return nil, fmt.Errorf("invalid agent registration")
		}
		if _, exists := s.agents[hello.AgentID]; exists {
			return nil, fmt.Errorf("an agent with this ID is already connected")
		}
		registered.role, registered.agentID = agentRole, hello.AgentID
		s.agents[hello.AgentID] = registered
	case protocol.TypePhoneHello:
		var hello protocol.PhoneHello
		if err := json.Unmarshal(message.Payload, &hello); err != nil {
			return nil, fmt.Errorf("invalid phone registration")
		}
		if _, exists := s.agents[hello.AgentID]; !exists {
			return nil, fmt.Errorf("the requested agent is not connected")
		}
		if _, exists := s.phones[hello.AgentID]; exists {
			return nil, fmt.Errorf("a phone is already connected to this agent")
		}
		if _, exists := s.phoneIDs[hello.PhoneID]; exists {
			return nil, fmt.Errorf("a phone with this ID is already connected")
		}
		registered.role, registered.agentID, registered.phoneID = phoneRole, hello.AgentID, hello.PhoneID
		s.phones[hello.AgentID] = registered
		s.phoneIDs[hello.PhoneID] = registered
	default:
		return nil, fmt.Errorf("the first message must register an agent or phone")
	}
	return registered, nil
}

func (s *Server) route(ctx context.Context, source *peer, data []byte, message protocol.Envelope) error {
	s.mu.Lock()
	var destination *peer
	switch source.role {
	case agentRole:
		if message.Type != protocol.TypeMetricsSnapshot && message.Type != protocol.TypeRelayProbeAck {
			s.mu.Unlock()
			return fmt.Errorf("agents may send only metrics snapshots and probe acknowledgements")
		}
		destination = s.phones[source.agentID]
	case phoneRole:
		if message.Type != protocol.TypeRelayProbe {
			s.mu.Unlock()
			return fmt.Errorf("phones may send only relay probes")
		}
		destination = s.agents[source.agentID]
	default:
		s.mu.Unlock()
		return fmt.Errorf("connection is not registered")
	}
	s.mu.Unlock()
	if destination == nil {
		return nil // No connected counterpart; the sender remains registered.
	}
	if err := s.writeText(ctx, destination, data); err != nil {
		// A counterpart can disappear between lookup and write. Remove only
		// that exact connection and keep the live sender registered.
		s.remove(destination.conn)
	}
	return nil
}

func (s *Server) writeText(ctx context.Context, destination *peer, data []byte) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return destination.conn.Write(ctx, websocket.MessageText, data)
}

func (s *Server) reject(ctx context.Context, conn *websocket.Conn, code, message string) {
	payload := protocol.ErrorPayload{Code: code, Message: message}
	envelope, err := protocol.NewEnvelope(protocol.TypeError, payload)
	if err == nil {
		data, marshalErr := json.Marshal(envelope)
		if marshalErr == nil {
			s.writeMu.Lock()
			_ = conn.Write(ctx, websocket.MessageText, data)
			s.writeMu.Unlock()
		}
	}
	_ = conn.Close(websocket.StatusPolicyViolation, "connection rejected")
}

func (s *Server) remove(conn *websocket.Conn) {
	s.mu.Lock()
	delete(s.conns, conn)
	for id, registered := range s.agents {
		if registered.conn == conn {
			delete(s.agents, id)
		}
	}
	for agentID, registered := range s.phones {
		if registered.conn == conn {
			delete(s.phones, agentID)
			delete(s.phoneIDs, registered.phoneID)
		}
	}
	s.mu.Unlock()
	_ = conn.Close(websocket.StatusNormalClosure, "connection closed")
}

// CloseConnections closes upgraded WebSockets during process shutdown.
// net/http.Server.Shutdown does not close hijacked WebSocket connections.
func (s *Server) CloseConnections() error {
	s.mu.Lock()
	connections := make([]*websocket.Conn, 0, len(s.conns))
	for conn := range s.conns {
		connections = append(connections, conn)
	}
	s.mu.Unlock()

	var firstErr error
	for _, conn := range connections {
		// CloseNow avoids waiting for a peer close handshake while the process
		// is already shutting down.
		if err := conn.CloseNow(); err != nil && !errors.Is(err, context.Canceled) && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
