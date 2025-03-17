/*
TODO: REMOVE THIS COMMENT. Logger package is a simple logger that uses zap.
It introduces a logger than is formatted in JSON and is easier to read.
I added a development mode to avoid having a lot of noise in the logs.
*/
package logger

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.Logger
}

func New(development bool) (*Logger, error) {
	level := zapcore.DebugLevel
	if !development {
		level = zapcore.InfoLevel
	}

	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	zapConfig := zap.Config{
		Level:             zap.NewAtomicLevelAt(level),
		Development:       development,
		DisableCaller:     false,
		DisableStacktrace: !development,
		Sampling:          nil,
		Encoding:          "json",
		EncoderConfig:     encoderCfg,
		OutputPaths:       []string{"stdout"},
		ErrorOutputPaths:  []string{"stderr"},
	}

	logger, err := zapConfig.Build(
		zap.AddCallerSkip(1),
	)
	if err != nil {
		return nil, err
	}

	return &Logger{logger}, nil
}

func (l *Logger) With(fields ...zapcore.Field) *Logger {
	return &Logger{l.Logger.With(fields...)}
}

func (l *Logger) WithField(key string, value any) *Logger {
	return &Logger{l.Logger.With(zap.Any(key, value))}
}

func (l *Logger) WithError(err error) *Logger {
	return &Logger{l.Logger.With(zap.Error(err))}
}

func (l *Logger) WithContext(ctx context.Context) *Logger {
	return l
}

func (l *Logger) Sync() {
	_ = l.Logger.Sync()
}
