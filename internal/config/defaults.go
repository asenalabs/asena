package config

import (
	"crypto/tls"
	"time"
)

var (
	portHTTP                = ":80"
	portHTTPS               = ":443"
	disableHTTPS            = false
	certFile                = "/etc/letsencrypt/live/example.com/cert.pem"
	keyFile                 = "/etc/letsencrypt/live/example.com/privkey.pem"
	llPath                  = "access.log"
	llMaxSize               = 100 // MB
	llMaxBackups            = 7
	llMaxAge                = 30 // days
	llCompress              = true
	ptDailTimeout           = 30 * time.Second
	ptDailKeepalive         = 30 * time.Second
	ptForceHTTP2            = true
	ptMaxIdleConn           = 100
	ptMaxIdleConnPerHost    = 10
	ptIdleConnTimeout       = 90 * time.Second
	ptTLSHandshakeTimeout   = 10 * time.Second
	ptExpectContinueTimeout = 1 * time.Second
	ptTLSMinVersion         = uint16(tls.VersionTLS12)
)

func setAsenaConfigs(cfg *AsenaConfig) {
	if cfg.Asena == nil {
		cfg.Asena = &AsenaCfg{}
	}
	if cfg.Log == nil {
		cfg.Log = &LogCfg{}
	}
	if cfg.Log.Lumberjack == nil {
		cfg.Log.Lumberjack = &LumberjackCfg{}
	}
	if cfg.ProxyTransport == nil {
		cfg.ProxyTransport = &ProxyTransportCfg{}
	}

	normalizeAsenaCfg(cfg.Asena)
	normalizeLogCfg(cfg.Log)
	normalizeProxyTransportCfg(cfg.ProxyTransport)
}

func normalizeAsenaCfg(cfg *AsenaCfg) {
	if cfg.EnableHTTPS == nil {
		cfg.EnableHTTPS = &disableHTTPS
	}
	if !*cfg.EnableHTTPS {
		cfg.Port = &portHTTP
	} else {
		cfg.Port = &portHTTPS
	}
	if cfg.TLSCertFile == nil {
		cfg.TLSCertFile = &certFile
	}
	if cfg.TLSKeyFile == nil {
		cfg.TLSKeyFile = &keyFile
	}
}

func normalizeLogCfg(cfg *LogCfg) {
	if cfg.Lumberjack.Path == nil {
		cfg.Lumberjack.Path = &llPath
	}
	if cfg.Lumberjack.MaxSize == nil {
		cfg.Lumberjack.MaxSize = &llMaxSize
	}
	if cfg.Lumberjack.MaxBackups == nil {
		cfg.Lumberjack.MaxBackups = &llMaxBackups
	}
	if cfg.Lumberjack.MaxAge == nil {
		cfg.Lumberjack.MaxAge = &llMaxAge
	}
	if cfg.Lumberjack.Compress == nil {
		cfg.Lumberjack.Compress = &llCompress
	}
}

func normalizeProxyTransportCfg(cfg *ProxyTransportCfg) {
	if cfg.DailTimeout == nil {
		cfg.DailTimeout = &ptDailTimeout
	}
	if cfg.DailKeepalive == nil {
		cfg.DailKeepalive = &ptDailKeepalive
	}
	if cfg.ForceHTTP2 == nil {
		cfg.ForceHTTP2 = &ptForceHTTP2
	}
	if cfg.MaxIdleConn == nil {
		cfg.MaxIdleConn = &ptMaxIdleConn
	}
	if cfg.MaxIdleConnPerHost == nil {
		cfg.MaxIdleConnPerHost = &ptMaxIdleConnPerHost
	}
	if cfg.IdleConnTimeout == nil {
		cfg.IdleConnTimeout = &ptIdleConnTimeout
	}
	if cfg.TLSHandshakeTimeout == nil {
		cfg.TLSHandshakeTimeout = &ptTLSHandshakeTimeout
	}
	if cfg.ExpectContinueTimeout == nil {
		cfg.ExpectContinueTimeout = &ptExpectContinueTimeout
	}
	if cfg.TLSMinVersion == nil {
		cfg.TLSMinVersion = &ptTLSMinVersion
	}
}
