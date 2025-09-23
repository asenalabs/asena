package cmd

import (
	"github.com/asenalabs/asena/pkg/logger"
	"go.uber.org/zap"
)

var (
	version = "0.0.1"
	env     = "development" //	development | production
)

func StartAsena() {
	//	Initialize basic logger
	logger.IntiFallbackZapLogger()
	logg := logger.Get()

	logg.Info("Starting asena", zap.String("version", version), zap.String("env", env))
}
