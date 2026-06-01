package logging

import (
	"io"
	"log/slog"
	"os"
	"sync"
)

type Logger struct {
	service  string
	fields   []interface{}
	writer   io.Writer
	mu       sync.Mutex
	internal *slog.Logger
}

func NewLogger(service string, w io.Writer) *Logger {
	if w == nil {
		w = os.Stdout
	}
	l := &Logger{
		service: service,
		writer:  w,
	}
	l.rebuild()
	return l
}

func (l *Logger) rebuild() {
	handler := slog.NewJSONHandler(l.writer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	l.internal = slog.New(handler).With("service", l.service)
	if len(l.fields) > 0 {
		l.internal = l.internal.With(l.fields...)
	}
}

func (l *Logger) With(keysAndValues ...interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	newFields := append(l.fields, keysAndValues...)
	return &Logger{
		service:  l.service,
		fields:   newFields,
		writer:   l.writer,
		internal: l.internal.With(keysAndValues...),
	}
}

func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	l.internal.Info(msg, keysAndValues...)
}

func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	l.internal.Error(msg, keysAndValues...)
}

func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	l.internal.Debug(msg, keysAndValues...)
}

func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	l.internal.Warn(msg, keysAndValues...)
}

// WithRequestID returns a new Logger with the request_id field set.
func (l *Logger) WithRequestID(id string) *Logger {
	return l.With("request_id", id)
}

// WithMethod returns a new Logger with the method field set.
func (l *Logger) WithMethod(method string) *Logger {
	return l.With("method", method)
}

// WithPath returns a new Logger with the path field set.
func (l *Logger) WithPath(path string) *Logger {
	return l.With("path", path)
}