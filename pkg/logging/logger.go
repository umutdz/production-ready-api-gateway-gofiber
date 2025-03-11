package logging

import (
    "fmt"
    "os"

    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

// Logger is a wrapper around zap.Logger
type Logger struct {
    *zap.Logger
}

// NewLogger creates a new logger with default settings
func NewLogger() (*Logger, error) {
    encoderCfg := zap.NewProductionEncoderConfig()
    encoderCfg.TimeKey = "timestamp"
    encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
    encoderCfg.MessageKey = "msg"

    config := zap.Config{
        Level:             zap.NewAtomicLevelAt(zapcore.InfoLevel),
        Development:       false,
        DisableCaller:     false,
        DisableStacktrace: false,
        Sampling:          nil,
        Encoding:          "json",
        EncoderConfig:     encoderCfg,
        OutputPaths:       []string{"stdout"}, // Varsayılan stdout
        ErrorOutputPaths:  []string{"stdout"},
        InitialFields: map[string]interface{}{
            "pid": os.Getpid(),
        },
    }

    zapLogger, err := config.Build(zap.AddCaller())
    if err != nil {
        return nil, fmt.Errorf("failed to build logger: %w", err)
    }

    return &Logger{zapLogger}, nil
}

// NewLoggerWithConfig creates a new logger with specified configuration
func NewLoggerWithConfig(level, format, outputPath string) (*Logger, error) {
    // Parse log level
    var zapLevel zapcore.Level
    if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
        zapLevel = zapcore.InfoLevel
    }

    // Encoder configuration
    encoderCfg := zap.NewProductionEncoderConfig()
    encoderCfg.TimeKey = "timestamp"
    encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
    encoderCfg.MessageKey = "msg"

    // Output configuration
    outputPaths := []string{outputPath}
    if outputPath == "" {
        outputPaths = []string{"stdout"} // Varsayılan stdout
    }

    // Config
    config := zap.Config{
        Level:             zap.NewAtomicLevelAt(zapLevel),
        Development:       false,
        DisableCaller:     false,
        DisableStacktrace: false,
        Sampling:          nil,
        Encoding:          format,
        EncoderConfig:     encoderCfg,
        OutputPaths:       outputPaths,
        ErrorOutputPaths:  outputPaths,
        InitialFields: map[string]interface{}{
            "pid": os.Getpid(),
        },
    }

    zapLogger, err := config.Build(zap.AddCaller())
    if err != nil {
        return nil, fmt.Errorf("failed to build logger: %w", err)
    }

    return &Logger{zapLogger}, nil
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
    return l.Logger.Sync()
}
