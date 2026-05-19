package audit

import (
	"log/slog"
	"time"
)

type Event struct {
	ID       string    `json:"id"`
	At       time.Time `json:"at"`
	Actor    string    `json:"actor"`
	Action   string    `json:"action"`
	Target   string    `json:"target"`
	Approved bool      `json:"approved"`
	Allowed  bool      `json:"allowed"`
	Reason   string    `json:"reason"`
}

type Recorder interface {
	Record(Event)
	List() []Event
}

type Logger struct {
	logger *slog.Logger
	events []Event
}

func NewLogger(logger *slog.Logger) *Logger {
	return &Logger{logger: logger, events: make([]Event, 0, 32)}
}

func (l *Logger) Record(event Event) {
	l.events = append(l.events, event)
	l.logger.Info("audit", "id", event.ID, "actor", event.Actor, "action", event.Action, "target", event.Target, "approved", event.Approved, "allowed", event.Allowed, "reason", event.Reason)
}

func (l *Logger) List() []Event {
	out := make([]Event, len(l.events))
	copy(out, l.events)
	return out
}
