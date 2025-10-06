package config

import (
	"strings"
	"testing"
)

// ============================== Static ==============================

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

// ============================== Dynamic ==============================

func TestValidateHTTPCfg(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *HTTPCfg
		wantErr string
	}{
		{"missing http", nil, "http section is missing"},
		{"missing routers", &HTTPCfg{}, "routers section is missing"},
		{"missing services", &HTTPCfg{Routers: map[string]*RoutersCfg{}}, "services section is missing"},
		{"valid config", &HTTPCfg{Routers: map[string]*RoutersCfg{}, Services: map[string]*ServiceCfg{}}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHTTPCfg(tt.cfg)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error %v", err)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestValidateServiceCfg(t *testing.T) {
	url := "http://localhost:9000"
	alg := RoundRobin

	tests := []struct {
		name    string
		cfg     *ServiceCfg
		wantErr string
	}{
		{"missing load balancer", nil, "load_balancer section is missing"},
		{"missing servers", &ServiceCfg{LoadBalancer: &LoadBalancerCfg{}}, "load_balancer.servers section is missing"},
		{"valid config", &ServiceCfg{
			LoadBalancer: &LoadBalancerCfg{
				Algorithm: &alg,
				Servers: []*ServerCfg{
					{URL: &url},
				},
			},
		}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateServiceCfg(tt.cfg)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error %v", err)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestValidateServiceAlgorithm(t *testing.T) {
	badAlg := "no-algorithm"
	goodAlg := RoundRobin

	tests := []struct {
		name    string
		alg     *string
		wantErr string
	}{
		{"nil algorithm", nil, "algorithm is not set"},
		{"unknown algorithm", &badAlg, "unknown algorithm"},
		{"valid algorithm", &goodAlg, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateServiceAlgorithm(tt.alg)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error %v", err)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestNormalizeServicesCfg(t *testing.T) {
	cfg := &ServiceCfg{} // load_balancer not defined
	normalizeServicesCfg(cfg)

	if cfg.LoadBalancer == nil {
		t.Fatal("expected LoadBalancer to be initialized, got nil")
	}
	if cfg.LoadBalancer.Algorithm == nil || *cfg.LoadBalancer.Algorithm != RoundRobin {
		t.Errorf("expected Algorithm=roundRobin, got %v", cfg.LoadBalancer.Algorithm)
	}
	if cfg.LoadBalancer.FlashInterval == nil || *cfg.LoadBalancer.FlashInterval != flashInterval {
		t.Errorf("expected FlashInterval=%v, got %v", flashInterval, cfg.LoadBalancer.FlashInterval)
	}
	if cfg.LoadBalancer.PassHostHeader == nil || *cfg.LoadBalancer.PassHostHeader != passHostHeaderFalse {
		t.Errorf("expected PassHostHeader=%v, got %v", passHostHeaderFalse, cfg.LoadBalancer.PassHostHeader)
	}
}
