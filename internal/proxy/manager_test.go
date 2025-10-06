package proxy

import (
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"
	"time"

	"github.com/asenalabs/asena/internal/config"
	"go.uber.org/zap/zaptest"
)

func TestNewProxyManager_Init(t *testing.T) {
	logg := zaptest.NewLogger(t)
	pm := NewProxyManger(logg)

	if pm == nil {
		t.Fatal("expected manager to be non-nil")
	}

	if _, ok := pm.ProxyHolder.Load().(map[string]*httputil.ReverseProxy); !ok {
		t.Error("expected ProxyHolder to contain map[string]*ReverseProxy")
	}

	if _, ok := pm.RouterHolder.Load().(map[string]*config.RoutersCfg); !ok {
		t.Error("expected RouterHolder to contain map[string]*RoutersCfg")
	}
}

func TestBuildReverseProxy_ValidConfig(t *testing.T) {
	logg := zaptest.NewLogger(t)

	pm := NewProxyManger(logg)

	algo := "round-robin"
	rule := "Host(`/api`)"
	flash := 50 * time.Millisecond
	passHost := true
	urlStr := "http://127.0.0.1:8080"

	cfg := &config.HTTPCfg{
		Services: map[string]*config.ServiceCfg{
			"api-service": {
				LoadBalancer: &config.LoadBalancerCfg{
					Algorithm:      &algo,
					Servers:        []*config.ServerCfg{{URL: &urlStr}},
					FlashInterval:  &flash,
					PassHostHeader: &passHost,
				},
			},
		},
		Routers: map[string]*config.RoutersCfg{
			"api-router": {
				Rule: &rule,
			},
		},
	}

	// transport config
	dialTimeout := time.Second
	keepAlive := time.Second
	forceHTTP2 := true
	maxIdle := 10
	maxIdlePerHost := 5
	idleTimeout := 30 * time.Second
	tlsMin := uint16(0x0303) // TLS1.2
	tCfg := &config.ProxyTransportCfg{
		DailTimeout:           &dialTimeout,
		DailKeepalive:         &keepAlive,
		ForceHTTP2:            &forceHTTP2,
		MaxIdleConn:           &maxIdle,
		MaxIdleConnPerHost:    &maxIdlePerHost,
		IdleConnTimeout:       &idleTimeout,
		TLSHandshakeTimeout:   &dialTimeout,
		ExpectContinueTimeout: &dialTimeout,
		TLSMinVersion:         &tlsMin,
	}

	pm.BuildReverseProxy(cfg, tCfg)

	// check proxy exists
	if rp, ok := pm.GetProxy("api-service"); !ok || rp == nil {
		t.Fatal("expected proxy for api-service to exist")
	}
}

func TestGetProxy_NotFound(t *testing.T) {
	logg := zaptest.NewLogger(t)
	pm := NewProxyManger(logg)

	_, ok := pm.GetProxy("unknown")
	if ok {
		t.Error("expected no proxy for unknown service")
	}
}

func TestReverseProxy_DirectorRewrite(t *testing.T) {
	logg := zaptest.NewLogger(t)
	pm := NewProxyManger(logg)

	algo := "round-robin"
	urlStr := "http://127.0.0.1:8080"
	flash := 50 * time.Millisecond
	passHost := true
	lb := &config.LoadBalancerCfg{
		Algorithm:      &algo,
		Servers:        []*config.ServerCfg{{URL: &urlStr}},
		FlashInterval:  &flash,
		PassHostHeader: &passHost,
	}

	dialTimeout := time.Second
	tlsMin := uint16(0x0303)
	tCfg := &config.ProxyTransportCfg{
		DailTimeout:           &dialTimeout,
		DailKeepalive:         &dialTimeout,
		ForceHTTP2:            new(bool),
		MaxIdleConn:           new(int),
		MaxIdleConnPerHost:    new(int),
		IdleConnTimeout:       &dialTimeout,
		TLSHandshakeTimeout:   &dialTimeout,
		ExpectContinueTimeout: &dialTimeout,
		TLSMinVersion:         &tlsMin,
	}

	rp, err := pm.newReverseProxy(tCfg, lb)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "http://original/path", nil)
	rp.Director(req)

	u, _ := url.Parse(urlStr)
	if req.URL.Host != u.Host {
		t.Errorf("expected host %s, got %s", u.Host, req.URL.Host)
	}
}
