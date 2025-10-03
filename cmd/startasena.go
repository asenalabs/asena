package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/asenalabs/asena/internal/config"
	"github.com/asenalabs/asena/pkg/logger"
	"go.uber.org/zap"
)

var (
	version               = "0.0.7"
	env                   = "development" //	development | production
	asenaConfigFilePath   = "asena.yaml"
	dynamicConfigFilePath = "dynamic.yaml"
)

func StartAsena() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	//	Initialize basic logger
	logger.IntiFallbackZapLogger()
	logg := logger.Get()

	//	Load Asena Configurations
	asenaConfigService, err := config.NewAsenaConfigService(asenaConfigFilePath, logg)
	if err != nil {
		logg.Fatal("Failed to initialize Asena configurations", zap.Error(err))
	}
	asenaCfg := asenaConfigService.Get()

	//	Switch logger according to config
	logger.InitProductionZapLogger(env, asenaCfg.Log)
	logg = logger.Get()
	defer logger.Sync()

	//	Load dynamic configurations
	dynamicConfigService, err := config.NewDynamicConfigService(ctx, dynamicConfigFilePath, logg)
	if err != nil {
		logg.Fatal("Failed to initialize dynamic configurations", zap.Error(err))
	}

	go func() {
		for _ = range dynamicConfigService.Updates() {

		}
	}()

	logg.Info("Starting asena", zap.String("version", version), zap.String("env", env))
	logg.Info("Asena configuration", zap.Bool("enable https", *asenaCfg.Asena.EnableHTTPS))
}
