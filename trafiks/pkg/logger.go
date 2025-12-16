package pkg

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/trafiks/trafiks/cfg"
)

type LoggerClient interface {
	Info(log ...interface{})
	Infof(format string, log ...interface{})
	Errorf(format string, log ...interface{})
	Error(msg string, err error, tags ...zap.Field)
	Fatalf(format string, log ...interface{})
	Panicf(format string, log ...interface{})
	LogWithFields(fields map[string]interface{}) *Logger
	Warn(log ...interface{})
	Warnf(format string, log ...interface{})
}

type Logger struct {
	log *zap.Logger
}

func NewLogger() LoggerClient {
	l := Logger{}

	logConfig := zap.Config{
		OutputPaths: []string{getOutput()},
		Level:       zap.NewAtomicLevelAt(getLevel()),
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			LevelKey:      "level",
			TimeKey:       "time",
			MessageKey:    "msg",
			EncodeTime:    zapcore.ISO8601TimeEncoder,
			EncodeLevel:   zapcore.LowercaseLevelEncoder,
			EncodeCaller:  zapcore.ShortCallerEncoder,
			StacktraceKey: "stacktrace",
		},
		DisableStacktrace: false,
	}

	logger, err := logConfig.Build()
	if err != nil {
		panic(err)
	}

	l.log = logger

	return &l
}

func getLevel() zapcore.Level {
	conf := cfg.GetConf()

	switch strings.ToLower(conf.LogLevel) {
	case "debug":
		return zap.DebugLevel
	case "info":
		return zap.InfoLevel
	case "warn":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	default:
		return zap.InfoLevel
	}
}

func getOutput() string {
	conf := cfg.GetConf()

	output := conf.LogOutput
	if output == "" {
		return "stdout"
	}

	return output
}

func (l *Logger) Info(log ...interface{}) {
	l.log.Info(fmt.Sprint(log...))
}

func (l *Logger) Infof(format string, log ...interface{}) {
	if len(log) == 0 {
		l.log.Info(format)
	} else {
		l.log.Info(fmt.Sprintf(format, log...))
	}
}

func (l *Logger) Errorf(format string, log ...interface{}) {
	msg := fmt.Sprintf(format, log...)
	l.log.Info(msg)
}

func (l *Logger) Error(msg string, err error, tags ...zap.Field) {
	tags = append(tags, zap.NamedError("error", err))
	l.log.Error(msg, tags...)
	l.log.Sync() //nolint:errcheck
}

func (l *Logger) Fatalf(format string, log ...interface{}) {
	msg := fmt.Sprintf(format, log...)
	l.log.Fatal(msg)
}

func (l *Logger) Panicf(format string, log ...interface{}) {
	msg := fmt.Sprintf(format, log...)
	l.log.Info(msg)
	panic(msg)
}

func (l *Logger) Warn(log ...interface{}) {
	l.log.Warn(fmt.Sprint(log...))
}

func (l *Logger) Warnf(format string, log ...interface{}) {
	msg := fmt.Sprintf(format, log...)
	l.log.Warn(msg)
}

func (l *Logger) LogWithFields(fields map[string]interface{}) *Logger {
	zapFields := make([]zap.Field, 0, len(fields))

	for key, value := range fields {
		switch v := value.(type) {
		case string:
			zapFields = append(zapFields, zap.String(key, v))
		case int:
			zapFields = append(zapFields, zap.Int(key, v))
		case int64:
			zapFields = append(zapFields, zap.Int64(key, v))
		case float64:
			zapFields = append(zapFields, zap.Float64(key, v))
		case bool:
			zapFields = append(zapFields, zap.Bool(key, v))
		default:
			zapFields = append(zapFields, zap.Any(key, v))
		}
	}

	return &Logger{
		log: l.log.With(zapFields...),
	}
}
