package log

import (
	"errors"
	"go.uber.org/zap"
)

func InitLoggerAnalyser() {
	var logger *zap.Logger
	var err error

	if logger, err = zap.NewProduction(); err != nil {
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
