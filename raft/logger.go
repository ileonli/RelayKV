package raft

import (
	"github.com/sirupsen/logrus"
	"io"
)

// Logger used to log information
type Logger interface {
	Debug(v ...interface{})
	Debugf(format string, v ...interface{})

	Error(v ...interface{})
	Errorf(format string, v ...interface{})

	Info(v ...interface{})
	Infof(format string, v ...interface{})

	Warning(v ...interface{})
	Warningf(format string, v ...interface{})

	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})

	Panic(v ...interface{})
	Panicf(format string, v ...interface{})
}

type LogrusLogger struct {
	log *logrus.Logger
}

// NewLogrusLogger return a new logger using logrus as a default logger.
// logrus's official repository: https://github.com/Sirupsen/logrus
func NewLogrusLogger(output io.Writer, level logrus.Level) Logger {

	logger := logrus.New()

	// Log as JSON instead of the default ASCII formatter.
	logger.SetFormatter(&logrus.TextFormatter{
		DisableColors:   true,
		TimestampFormat: "2006-01-02 15:04:05.000",
	})

	logger.SetOutput(output)

	logger.SetLevel(level)
	return &LogrusLogger{log: logger}
}

func (l *LogrusLogger) Debug(v ...interface{}) {
	l.log.Debug(v...)
}

func (l *LogrusLogger) Debugf(format string, v ...interface{}) {
	l.log.Debugf(format, v...)
}

func (l *LogrusLogger) Error(v ...interface{}) {
	l.log.Error(v...)
}

func (l *LogrusLogger) Errorf(format string, v ...interface{}) {
	l.log.Errorf(format, v...)
}

func (l *LogrusLogger) Info(v ...interface{}) {
	l.log.Info(v...)
}

func (l *LogrusLogger) Infof(format string, v ...interface{}) {
	l.log.Infof(format, v...)
}

func (l *LogrusLogger) Warning(v ...interface{}) {
	l.log.Warning(v...)
}

func (l LogrusLogger) Warningf(format string, v ...interface{}) {
	l.log.Warningf(format, v...)
}

func (l *LogrusLogger) Fatal(v ...interface{}) {
	l.log.Fatal(v...)
}

func (l *LogrusLogger) Fatalf(format string, v ...interface{}) {
	l.log.Fatalf(format, v...)
}

func (l *LogrusLogger) Panic(v ...interface{}) {
	l.log.Panic(v...)
}

func (l *LogrusLogger) Panicf(format string, v ...interface{}) {
	l.log.Panicf(format, v...)
}
