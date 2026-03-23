package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Logger
}

type Fields map[string]interface{}

/* VARIABLES FOR ROOT LOGGER */
var rootLogger *Logger

func Init() {
	rootLogger = DefaultNewLogger()
}

func DefaultNewLogger() *Logger {
	log := logrus.New()

	log.SetLevel(logrus.InfoLevel)
	log.SetOutput(os.Stdout)
	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})

	return &Logger{log}
}

func NewLoggerWithConfig(level string, formatter string, output io.Writer) *Logger {
	log := logrus.New()

	// Set log level
	parseLevel, err := logrus.ParseLevel(level)
	if err != nil {
		parseLevel = logrus.InfoLevel
	}
	log.SetLevel(parseLevel)

	// Set log formatter
	switch formatter {
	case "json":
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	case "text":
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	default:
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	}

	// Set log output
	if output != nil {
		log.SetOutput(output)
	} else {
		log.SetOutput(os.Stdout)
	}

	return &Logger{log}
}

func (l *Logger) WithField(key string, value interface{}) *logrus.Entry {
	return l.Logger.WithField(key, value)
}

func (l *Logger) WithFields(fields Fields) *logrus.Entry {
	return l.Logger.WithFields(logrus.Fields(fields))
}

func (l *Logger) WithError(err error) *logrus.Entry {
	return l.Logger.WithError(err)
}

func (l *Logger) WithContext(ctx context.Context) *logrus.Entry {
	return l.Logger.WithContext(ctx)
}

func (l *Logger) WithTime(t time.Time) *logrus.Entry {
	return l.Logger.WithTime(t)
}

func (l *Logger) Info(msg string, fields ...Fields) {
	entry := l.Logger.WithFields(mergeFields(fields...))
	entry.Info(msg)
}

func (l *Logger) Debug(msg string, fields ...Fields) {
	entry := l.Logger.WithFields(mergeFields(fields...))
	entry.Debug(msg)
}

func (l *Logger) Warn(msg string, fields ...Fields) {
	entry := l.Logger.WithFields(mergeFields(fields...))
	entry.Warn(msg)
}

func (l *Logger) Error(msg string, err error, fields ...Fields) {
	f := mergeFields(fields...)
	if err != nil {
		f["error"] = err.Error()
	}
	entry := l.Logger.WithFields(f)
	entry.Error(msg)
}

func (l *Logger) Fatal(msg string, err error, fields ...Fields) {
	f := mergeFields(fields...)
	if err != nil {
		f["error"] = err.Error()
	}
	entry := l.Logger.WithFields(f)
	entry.Fatal(msg)
}

func (l *Logger) Panic(msg string, err error, fields ...Fields) {
	f := mergeFields(fields...)
	if err != nil {
		f["error"] = err.Error()
	}
	entry := l.Logger.WithFields(f)
	entry.Panic(msg)
}

/* HELPER METHODS */
func mergeFields(fields ...Fields) logrus.Fields {
	mergeRes := logrus.Fields{}
	for _, f := range fields {
		for k, v := range f {
			mergeRes[k] = v
		}
	}
	return mergeRes
}

/* METHODS FOR ROOT LOGGER */
func GetRootLogger() *Logger {
	return rootLogger
}

func SetRootLogger(logger *Logger) {
	rootLogger = logger
}

func Info(msg string, fields ...Fields) {
	rootLogger.Info(msg, fields...)
}

func Debug(msg string, fields ...Fields) {
	rootLogger.Debug(msg, fields...)
}

func Warn(msg string, fields ...Fields) {
	rootLogger.Warn(msg, fields...)
}

func Error(msg string, err error, fields ...Fields) {
	rootLogger.Error(msg, err, fields...)
}

func Fatal(msg string, err error, fields ...Fields) {
	rootLogger.Fatal(msg, err, fields...)
}

func Panic(msg string, err error, fields ...Fields) {
	rootLogger.Panic(msg, err, fields...)
}
