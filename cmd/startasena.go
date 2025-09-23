package cmd

import (
	"github.com/asenalabs/asena/internal/config"
	"github.com/asenalabs/asena/pkg/logger"
	"go.uber.org/zap"
)

var (
	version             = "0.0.3"
	env                 = "development" //	development | production
	asenaConfigFilePath = "asena.yaml"
)

func StartAsena() {
	//	Initialize basic logger
	logger.IntiFallbackZapLogger()
	logg := logger.Get()

	//	Load Asena Configurations
	asenaConfigService, err := config.NewAsenaConfigService(asenaConfigFilePath, logg)
	if err != nil {
		logg.Fatal("Failed to initialize Asena configurations", zap.Error(err))
	}
	asenaCfg := asenaConfigService.Get()

	logg.Info("Starting asena", zap.String("version", version), zap.String("env", env))
	logg.Info("Asena configuration", zap.Bool("enable https", *asenaCfg.Asena.EnableHTTPS))
}
