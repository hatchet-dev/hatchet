package logger

import (
	"context"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

type Logger struct {
	zerolog.Logger
}

func New(l *zerolog.Logger) *Logger {
	return &Logger{Logger: *l}
}

func From(l zerolog.Logger) *Logger {
	return &Logger{Logger: l}
}

func (l *Logger) Ctx(ctx context.Context) *zerolog.Logger {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		logger := l.Logger.With().
			Str("trace_id", span.SpanContext().TraceID().String()).
			Str("span_id", span.SpanContext().SpanID().String()).
			Logger()
		return &logger
	}
	return &l.Logger
}

func (l *Logger) With() zerolog.Context {
	return l.Logger.With()
}

func (l *Logger) Level(lvl zerolog.Level) *Logger {
	return &Logger{Logger: l.Logger.Level(lvl)}
}

func (l *Logger) Sample(s zerolog.Sampler) *Logger {
	return &Logger{Logger: l.Logger.Sample(s)}
}

func (l *Logger) Hook(h zerolog.Hook) *Logger {
	return &Logger{Logger: l.Logger.Hook(h)}
}

func (l *Logger) Trace() *zerolog.Event {
	return l.Logger.Trace()
}

func (l *Logger) Debug() *zerolog.Event {
	return l.Logger.Debug()
}

func (l *Logger) Info() *zerolog.Event {
	return l.Logger.Info()
}

func (l *Logger) Warn() *zerolog.Event {
	return l.Logger.Warn()
}

func (l *Logger) Error() *zerolog.Event {
	return l.Logger.Error()
}

func (l *Logger) Err(err error) *zerolog.Event {
	return l.Logger.Err(err)
}

func (l *Logger) Fatal() *zerolog.Event {
	return l.Logger.Fatal()
}

func (l *Logger) Panic() *zerolog.Event {
	return l.Logger.Panic()
}

func (l *Logger) WithLevel(level zerolog.Level) *zerolog.Event {
	return l.Logger.WithLevel(level)
}

func (l *Logger) Log() *zerolog.Event {
	return l.Logger.Log()
}

func (l *Logger) Print(v ...interface{}) {
	l.Logger.Print(v...)
}

func (l *Logger) Printf(format string, v ...interface{}) {
	l.Logger.Printf(format, v...)
}

func (l *Logger) Println(v ...interface{}) {
	l.Logger.Println(v...)
}
