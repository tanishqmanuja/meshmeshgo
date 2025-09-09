package logger

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

type Fields logrus.Fields

var log = &logrus.Logger{
	Out:       os.Stderr,
	Formatter: &logrus.TextFormatter{
		DisableTimestamp: false,
		FullTimestamp: true,
		ForceColors: true,
		TimestampFormat: time.TimeOnly,
		DisableColors: false},
	Hooks:     make(logrus.LevelHooks),
	Level:     logrus.WarnLevel,
}

func SetLevel(level logrus.Level) {
	log.SetLevel(level)
}

func Log() *logrus.Logger {
	return log
}

func Error(args ...interface{}) {
	log.Error(args...)
}

func Info(args ...interface{}) {
	log.Info(args...)
}

func Debug(args ...interface{}) {
	log.Debug(args...)
}

func Fatal(args ...interface{}) {
	log.Fatal(args...)
}

func Printf(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func WithField(key string, value interface{}) *logrus.Entry {
	return log.WithField(key, value)
}

func WithFields(fields Fields) *logrus.Entry {
	return log.WithFields(logrus.Fields(fields))
}


func IsInfo() (bool) {
	return log.Level == logrus.InfoLevel || IsDebug()
}

func IsDebug() (bool) {
	return log.Level == logrus.DebugLevel || IsTrace()
}

func IsTrace() (bool) {
	return log.Level == logrus.TraceLevel
}
