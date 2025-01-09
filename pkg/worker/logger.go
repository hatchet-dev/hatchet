package worker

import (
	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
)

type LogrusForwardHook struct {
	levels []logrus.Level
	hc     HatchetContext
}

func NewLogrusForwardHook(hc HatchetContext, levels []logrus.Level) *LogrusForwardHook {
	return &LogrusForwardHook{
		levels: levels,
		hc:     hc,
	}
}

func (h *LogrusForwardHook) Levels() []logrus.Level {
	return h.levels
}

func (h *LogrusForwardHook) Fire(entry *logrus.Entry) error {
	logMessage, err := entry.String()
	if err != nil {
		return err
	}
	h.hc.Log(logMessage)
	return nil
}

type ZerologForwardHook struct {
	hc HatchetContext
}

func (h *ZerologForwardHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	h.hc.Log(msg)
}

func (hc *hatchetContext) NewCombinedLogrusLogger(originalLogger *logrus.Logger) *logrus.Logger {
	newLogger := logrus.New()
	newLogger.Out = originalLogger.Out
	newLogger.Formatter = originalLogger.Formatter
	newLogger.Hooks = originalLogger.Hooks
	newLogger.Level = originalLogger.Level
	newLogger.ExitFunc = originalLogger.ExitFunc
	newLogger.ReportCaller = originalLogger.ReportCaller

	newLogger.Hooks = make(logrus.LevelHooks)
	for level, hooks := range originalLogger.Hooks {
		newLogger.Hooks[level] = append([]logrus.Hook{}, hooks...)
	}

	hook := NewLogrusForwardHook(hc, logrus.AllLevels)
	newLogger.AddHook(hook)
	return newLogger
}

func (hc *hatchetContext) NewCombinedZerologLogger(originalLogger zerolog.Logger) zerolog.Logger {
	newLogger := originalLogger
	hook := &ZerologForwardHook{hc: hc}
	return newLogger.Hook(hook)
}
