package server

import (
	"crypto/tls"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/asenalabs/asena/pkg/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ServerConfig struct {
	Address     string
	Version     string
	EnableHTTPS bool
	CertFileTLS string
	KeyFileTLS  string
	Proxy       http.Handler
	Logg        *zap.Logger
}

func ServeHTTPS(cfg *ServerConfig) (*http.Server, error) {
	if cfg.EnableHTTPS {
		certMg, err := NewCertManager(cfg.CertFileTLS, cfg.KeyFileTLS)
		if err != nil {
			cfg.Logg.Warn("Failed to reload certificates", zap.Error(err))
			return startHTTP(cfg)
		}

		//	SIGHUP listener for reload new TLS certificate
		go func() {
			signalChan := make(chan os.Signal, 1)
			signal.Notify(signalChan, syscall.SIGHUP)

			for range signalChan {
				cfg.Logg.Info("[TLS] Reloading certificates...")
				if err := certMg.Load(cfg.CertFileTLS, cfg.KeyFileTLS); err != nil {
					cfg.Logg.Warn("Failed to reload certificates", zap.Error(err))
				}
				cfg.Logg.Info("[TLS] Certificates reloaded successfully.")
			}
		}()

		srv := &http.Server{
			Addr:    cfg.Address,
			Handler: cfg.Proxy,
			TLSConfig: &tls.Config{
				GetCertificate: certMg.GetCertificate,
			},
			ErrorLog: logger.MustZapToStdLoggerAtLevel(cfg.Logg, zapcore.WarnLevel),
		}

		go func() {
			cfg.Logg.Info("[HTTPS] Asena has started", zap.String("version", cfg.Version), zap.String("address", cfg.Address))
			if err := srv.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
				cfg.Logg.Warn("[HTTPS] Failed to start HTTPS server", zap.Error(err), zap.String("version", cfg.Version))
			}
		}()
		go startRedirectToHTTPS(cfg.Logg)

		return srv, nil

	} else {
		return startHTTP(cfg)
	}
}

func startHTTP(cfg *ServerConfig) (*http.Server, error) {
	srv := &http.Server{
		Addr:     cfg.Address,
		Handler:  cfg.Proxy,
		ErrorLog: logger.MustZapToStdLoggerAtLevel(cfg.Logg, zap.WarnLevel),
	}

	go func() {
		cfg.Logg.Info("[HTTP] Asena has started", zap.String("version", cfg.Version), zap.String("address", cfg.Address))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			cfg.Logg.Warn("[HTTP] Failed to start HTTP server", zap.Error(err), zap.String("version", cfg.Version))
		}
	}()

	return srv, nil
}

func startRedirectToHTTPS(logg *zap.Logger) {
	err := http.ListenAndServe(":80", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
	}))
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logg.Warn("[HTTP → HTTPS] Failed to start redirect server", zap.String("error", err.Error()))
	}
}
