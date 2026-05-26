package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nomados/nomados/packages/logging"

	natsgo "github.com/nats-io/nats.go"
)

// Event represents a parsed NATS event with metadata.
type Event struct {
	Subject   string          `json:"subject"`
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time        `json:"timestamp"`
	Source    string          `json:"source,omitempty"`
}

// EventProcessor is the interface that the aggregator implements
// to receive events from the subscriber.
type EventProcessor interface {
	ProcessEvent(subject string, data []byte)
	IncrementErrors(service string)
}

// Subscriber subscribes to NATS subjects and forwards events
// to the aggregator for metric processing.
type Subscriber struct {
	conn   *natsgo.Conn
	agg    EventProcessor
	logger *logging.Logger
	subs   []*natsgo.Subscription
}

// NewSubscriber connects to NATS and returns a new Subscriber.
func NewSubscriber(natsURL string, agg EventProcessor, logger *logging.Logger) (*Subscriber, error) {
	conn, err := natsgo.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return &Subscriber{
		conn:   conn,
		agg:    agg,
		logger: logger,
	}, nil
}

// SubscribeAll sets up subscriptions for all event subjects.
func (s *Subscriber) SubscribeAll(ctx context.Context) error {
	subjects := map[string]string{
		"audit":     "audit.*",
		"session":   "session.*",
		"workspace": "workspace.*",
		"vault":     "vault.*",
	}

	for name, subject := range subjects {
		sub, err := s.conn.Subscribe(subject, func(msg *natsgo.Msg) {
			s.processEvent(msg.Subject, msg.Data)
		})
		if err != nil {
			return fmt.Errorf("failed to subscribe to %s (%s): %w", name, subject, err)
		}
		s.subs = append(s.subs, sub)
	}

	s.logger.Info("subscribed to all event subjects",
		"subjects", "audit.*, session.*, workspace.*, vault.*")

	return nil
}

// processEvent handles an incoming NATS message by parsing it,
// logging a structured entry, and forwarding it to the aggregator.
func (s *Subscriber) processEvent(subject string, data []byte) {
	s.logger.Info("received event",
		"subject", subject,
		"data_length", len(data),
	)

	// Try to parse the event for structured logging
	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		// If data is not valid JSON, still process the event
		// but log the parse error
		s.logger.Warn("failed to parse event data as structured event",
			"subject", subject,
			"error", err,
		)
	}

	// Forward to aggregator for metric processing
	s.agg.ProcessEvent(subject, data)
}

// Close unsubscribes from all subjects and closes the NATS connection.
func (s *Subscriber) Close() {
	for _, sub := range s.subs {
		if err := sub.Unsubscribe(); err != nil {
			s.logger.Warn("failed to unsubscribe", "error", err)
		}
	}
	s.subs = nil

	if s.conn != nil && !s.conn.IsClosed() {
		s.conn.Close()
	}
}

// ParseEvent is a helper function to parse raw JSON data into an Event struct.
// It is exported for use in tests and other packages.
func ParseEvent(data []byte) (*Event, error) {
	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to parse event: %w", err)
	}
	return &event, nil
}