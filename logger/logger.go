package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

type Fields logrus.Fields

var log = &logrus.Logger{
	Out:       os.Stderr,
	Formatter: &logrus.TextFormatter{DisableTimestamp: false, FullTimestamp: true, DisableColors: false},
	Hooks:     make(logrus.LevelHooks),
	Level:     logrus.WarnLevel,
}

func SetLevel(level logrus.Level) {
	log.SetLevel(level)
}

func Log() *logrus.Logger {
	return log
}

func WithField(key string, value interface{}) *logrus.Entry {
	return log.WithField(key, value)
}

func WithFields(fields Fields) *logrus.Entry {
	return log.WithFields(logrus.Fields(fields))
}
