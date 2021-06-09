package logger

import (
	"go.uber.org/zap"
)

func New() (*zap.Logger, error) {
	logger, err := zap.NewDevelopment()
	// logger, err := zap.NewProduction()
	return logger, err
}
