package logging

import "time"

type AuditEntry struct {
	ActorID   string                 `json:"actor_id"`
	Action    string                 `json:"action"`
	Target    string                 `json:"target"`
	Timestamp time.Time              `json:"timestamp"`
	Signature string                 `json:"signature,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

func (l *Logger) Audit(entry AuditEntry) {
	l.Info("audit",
		"actor_id", entry.ActorID,
		"action", entry.Action,
		"target", entry.Target,
		"timestamp", entry.Timestamp.UTC().Format(time.RFC3339Nano),
		"details", entry.Details,
	)
}