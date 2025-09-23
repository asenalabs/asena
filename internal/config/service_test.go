package config

import (
	"os"
	"testing"

	"go.uber.org/zap"
)

func TestAsenaConfigService_Load_defaults(t *testing.T) {
	tmpFile := "test_config.yaml"
	content := []byte(`
asena: {}
log:
  lumberjack: {}
proxy_transport: {}
`)

	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Remove(tmpFile); err != nil {
			t.Fatal(err)
		}
	}()

	svc, err := NewAsenaConfigService(tmpFile, zap.NewNop())
	if err != nil {
		t.Fatalf("NewAsenaConfigService() failed: %v", err)
	}

	cfg := svc.Get()

	if cfg.Asena.Port == nil || *cfg.Asena.Port != portHTTP {
		t.Errorf("expected default port to be %s, got %v", portHTTP, *cfg.Asena.Port)
	}
	if cfg.ProxyTransport.MaxIdleConn == nil || *cfg.ProxyTransport.MaxIdleConn != ptMaxIdleConn {
		t.Errorf("expected default maxIdleConn to be %d, got %v", ptMaxIdleConn, *cfg.ProxyTransport.MaxIdleConn)
	}
}

func TestAsenaConfigService_Load_Override(t *testing.T) {
	tmpFile := "test_config.yaml"
	content := []byte(`
asena:
  enable_https: true
  tls_cert_file: "mycert.pem"
  tls_key_file: "mykey.pem"
`)

	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Remove(tmpFile); err != nil {
			t.Fatal(err)
		}
	}()

	svc, err := NewAsenaConfigService(tmpFile, zap.NewNop())
	if err != nil {
		t.Fatalf("NewAsenaConfigService() failed: %v", err)
	}

	cfg := svc.Get()

	if *cfg.Asena.Port != portHTTPS {
		t.Errorf("expected default port to be %s, got %v", portHTTP, *cfg.Asena.Port)
	}
	if *cfg.Asena.TLSCertFile != "mycert.pem" {
		t.Errorf("expected overridden TLS cert file, got %s", *cfg.Asena.TLSCertFile)
	}
}
