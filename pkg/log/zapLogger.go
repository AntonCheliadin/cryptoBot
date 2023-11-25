package log

import (
	"errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func InitLoggerAnalyser() {
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	loggerConfig.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	logger, err := loggerConfig.Build()

	if err != nil {
		panic(errors.New("Fatal error during create logger" + err.Error()))
	}

	zap.ReplaceGlobals(logger)
}

func InitLogger() {
	var logger *zap.Logger
	var err error

	if logger, err = zap.NewDevelopment(); err != nil {
		panic(errors.New("Fatal error during create logger" + err.Error()))
	}
	zap.ReplaceGlobals(logger)
}
