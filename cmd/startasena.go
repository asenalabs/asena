package cmd

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asenalabs/asena/internal/config"
	"github.com/asenalabs/asena/internal/handler"
	"github.com/asenalabs/asena/internal/proxy"
	"github.com/asenalabs/asena/internal/server"
	"github.com/asenalabs/asena/pkg/logger"
	"go.uber.org/zap"
)

var (
	version               = "0.1.3"
	env                   = "development" //	development | production
	asenaConfigFilePath   = "asena.yaml"
	dynamicConfigFilePath = "dynamic.yaml"
	gracefulShutdownTime  = 5 * time.Second
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

	pm := proxy.NewProxyManger(logg)

	go func() {
		for newDCfg := range dynamicConfigService.Updates() {
			pm.BuildReverseProxy(newDCfg.HTTP, asenaCfg.ProxyTransport)
		}
	}()

	//	Local mux
	mux := http.NewServeMux()
	handler.RegisterRoutes(pm, mux, logg)

	wrappedMux := handler.LoggingMiddleware(logg, mux)

	//	server configurations
	srvCfg := server.ServerConfig{
		Address:     *asenaCfg.Asena.Port,
		Version:     version,
		EnableHTTPS: *asenaCfg.Asena.EnableHTTPS,
		CertFileTLS: *asenaCfg.Asena.TLSCertFile,
		KeyFileTLS:  *asenaCfg.Asena.TLSKeyFile,
		Proxy:       wrappedMux,
		Logg:        logg,
	}

	srv, err := server.ServeHTTPS(&srvCfg)
	if err != nil {
		logg.Fatal("Failed to start server", zap.Error(err))
	}

	// Graceful shutdown
	<-ctx.Done()
	logg.Info("Asena server shutting down", zap.String("version", version), zap.Duration("timeout", gracefulShutdownTime))

	shutDownCtx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTime)
	defer cancel()

	if err := srv.Shutdown(shutDownCtx); err != nil {
		logg.Warn("Asena server forced to shutdown", zap.String("version", version), zap.Error(err))
	}

	logg.Info("Asena server gracefully shutdown", zap.String("version", version))
}
