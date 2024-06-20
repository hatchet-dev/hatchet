package logger

import (
	"io"
	"os"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/config/shared"

	"time"
)

func init() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
}

func NewDefaultLogger(service string) zerolog.Logger {
	return NewStdErr(&shared.LoggerConfigFile{}, service)
}

func NewStdErr(cf *shared.LoggerConfigFile, service string) zerolog.Logger {
	lvl := zerolog.DebugLevel
	var err error

	if cf.Level != "" {
		lvl, err = zerolog.ParseLevel(cf.Level)

		if err != nil {
			panic(err)
		}
	}

	var out io.Writer = os.Stderr

	if cf.Format == "console" {
		out = zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "2006-01-02T15:04:05.999Z07:00",
		}
	}

	l := zerolog.New(out).Level(lvl)
	l = l.With().Timestamp().Logger()
	if service != "" {
		l = l.With().Str("service", service).Logger()
	}

	return l
}
