package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// ============================== Static ==============================

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

// ============================== Dynamic ==============================

func TestReload_ConfigFileMissing(t *testing.T) {
	dcs := &DynamicConfigService{
		configFilePath: "does-not-exist.yaml",
		logg:           zap.NewNop(),
		updates:        make(chan *DynamicConfig, 1),
	}

	if err := dcs.reload(); err == nil {
		t.Error("expected error when config file is missing")
	}
}

func TestReload_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.yaml")
	if err := os.WriteFile(path, []byte("::not-valid-yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	dcs := &DynamicConfigService{
		configFilePath: path,
		logg:           zap.NewNop(),
		updates:        make(chan *DynamicConfig, 1),
	}

	if err := dcs.reload(); err == nil {
		t.Error("expected yaml parse error, got nil")
	}
}

func TestReload_ValidConfigAndHashCheck(t *testing.T) {
	dir := t.TempDir()
	url := "http://localhost:9000"
	cfg := &DynamicConfig{
		HTTP: &HTTPCfg{
			Routers: map[string]*RoutersCfg{"r1": {}},
			Services: map[string]*ServiceCfg{
				"s1": {
					LoadBalancer: &LoadBalancerCfg{
						Servers: []*ServerCfg{{URL: &url}},
					},
				},
			},
		},
	}
	path := writeTempConfig(t, dir, cfg)

	dcs := &DynamicConfigService{
		configFilePath: path,
		logg:           zap.NewNop(),
		updates:        make(chan *DynamicConfig, 1),
	}

	//	first load
	if err := dcs.reload(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := dcs.Get(); got == nil || got.HTTP == nil {
		t.Error("expected config loaded, got nil")
	}

	select {
	case <-dcs.Updates():
		//	ok
	default:
		t.Errorf("expected update on first reload")
	}

	//	second load with same content -> should not push update
	if err := dcs.reload(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case <-dcs.Updates():
		t.Error("did not expect update when config unchanged")
	default:
		//	ok
	}
}

func writeTempConfig(t *testing.T, dir string, cfg *DynamicConfig) string {
	t.Helper()

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal yaml: %v", err)
	}
	path := filepath.Join(dir, "dynamic.yaml")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	return path
}

func TestWatch_FileChangeTriggerReload(t *testing.T) {
	dir := t.TempDir()
	url := "http://localhost:9000"
	cfg := &DynamicConfig{
		HTTP: &HTTPCfg{
			Routers: map[string]*RoutersCfg{"r1": {}},
			Services: map[string]*ServiceCfg{
				"s1": {
					LoadBalancer: &LoadBalancerCfg{
						Servers: []*ServerCfg{{URL: &url}},
					},
				},
			},
		},
	}
	path := writeTempConfig(t, dir, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dcs, err := NewDynamicConfigService(ctx, path, zap.NewNop())
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	// modify file → expect update
	cfg.HTTP.Services["s1"] = &ServiceCfg{}
	writeTempConfig(t, dir, cfg)

	select {
	case <-dcs.Updates():
		// ok
	case <-time.After(2 * time.Second):
		t.Errorf("expected reload after file change")
	}
}
