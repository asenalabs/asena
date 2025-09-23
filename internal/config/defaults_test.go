package config

import "testing"

func TestNormalizeAsenaCfg_defaults(t *testing.T) {
	cfg := &AsenaCfg{}
	normalizeAsenaCfg(cfg)

	if cfg.Port == nil || *cfg.Port != portHTTP {
		t.Errorf("expected default port %s, got %v", portHTTP, cfg.Port)
	}
	if cfg.TLSCertFile == nil || *cfg.TLSCertFile != certFile {
		t.Errorf("expected default TLS cert file %s, got %v", certFile, cfg.TLSCertFile)
	}
	if cfg.TLSKeyFile == nil || *cfg.TLSKeyFile != keyFile {
		t.Errorf("expected default TLS key file %s, got %v", keyFile, cfg.TLSKeyFile)
	}
}

func TestNormalizeLogCfg_defaults(t *testing.T) {
	cfg := &LogCfg{Lumberjack: &LumberjackCfg{}}
	normalizeLogCfg(cfg)

	if cfg.Lumberjack.Path == nil || *cfg.Lumberjack.Path != llPath {
		t.Errorf("expected default path %s, got %v", llPath, cfg.Lumberjack.Path)
	}
	if cfg.Lumberjack.MaxSize == nil || *cfg.Lumberjack.MaxSize != llMaxSize {
		t.Errorf("expected default max size %v, got %v", llMaxSize, cfg.Lumberjack.MaxSize)
	}
}

func TestNormalizeProxyTransportCfg_defaults(t *testing.T) {
	cfg := &ProxyTransportCfg{}
	normalizeProxyTransportCfg(cfg)

	if cfg.DailTimeout == nil || *cfg.DailTimeout != ptDailTimeout {
		t.Errorf("expected default dail timeout %s, got %v", ptDailTimeout, cfg.DailTimeout)
	}
	if cfg.TLSHandshakeTimeout == nil || *cfg.TLSHandshakeTimeout != ptTLSHandshakeTimeout {
		t.Errorf("expected dafault TLS handshake timeout %s, got %v", ptTLSHandshakeTimeout, cfg.TLSHandshakeTimeout)
	}
}
