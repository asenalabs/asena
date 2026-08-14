package proxy

import (
	"net/http"
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

	if _, ok := pm.RouterHolder.Load().([]Route); !ok {
		t.Error("expected RouterHolder to contain map[string]*RoutersCfg")
	}
}

func TestBuildReverseProxy_ValidConfig(t *testing.T) {
	logg := zaptest.NewLogger(t)

	pm := NewProxyManger(logg)

	algo := "round-robin"
	rule := "Host(`a.com`)"
	service := "api-service"
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
				Rule:    &rule,
				Service: &service,
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

	// check route was compiled and stored
	routes, ok := pm.RouterHolder.Load().([]Route)
	if !ok {
		t.Fatalf("expected 1 compiled route, got %d", len(routes))
	}
	if routes[0].Name != "api-router" || routes[0].Service != "api-service" {
		t.Errorf("unexpected compiled route: %+v", routes[0])
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

	inReq := httptest.NewRequest("GET", "http://original/path", nil)
	outerReq := inReq.Clone(inReq.Context())
	preq := &httputil.ProxyRequest{In: inReq, Out: outerReq}

	rp.Rewrite(preq)

	u, _ := url.Parse(urlStr)
	if preq.Out.URL.Host != u.Host {
		t.Errorf("expected host %s, got %s", u.Host, preq.Out.URL.Host)
	}
	if passHost && preq.Out.Host != u.Host {
		t.Errorf("expected Out.Host %s (PassHostHeader), got %s", u.Host, preq.Out.Host)
	}
}

func TestServeProxy_NotFound(t *testing.T) {
	logg := zaptest.NewLogger(t)
	pm := NewProxyManger(logg)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "http://example.com/", nil)

	if ok := pm.ServeProxy("unknown-service", w, r); ok {
		t.Error("expected ServeProxy to return false for an unknown service")
	}
}

func TestServeProxy_RoundTripsToBackend(t *testing.T) {
	// A real backend, not a mock: ServeProxy has to run the whole chain
	// (Rewrite -> RoundTrip -> ModifyResponse -> reportDone) for real, so
	// we want a real HTTP server on the other end to prove none of those
	// steps panics or gets skipped.
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	logg := zaptest.NewLogger(t)
	pm := NewProxyManger(logg)

	algo := "round-robin"
	rule := "Host(`a.com`)"
	service := "api-service"
	flash := 50 * time.Millisecond
	backendURL := backend.URL

	cfg := &config.HTTPCfg{
		Services: map[string]*config.ServiceCfg{
			service: {
				LoadBalancer: &config.LoadBalancerCfg{
					Algorithm:     &algo,
					Servers:       []*config.ServerCfg{{URL: &backendURL}},
					FlashInterval: &flash,
				},
			},
		},
		Routers: map[string]*config.RoutersCfg{
			"api-router": {Rule: &rule, Service: &service},
		},
	}

	dialTimeout := time.Second
	tlsMin := uint16(0x0303) // TLS1.2
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

	pm.BuildReverseProxy(cfg, tCfg)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "http://a.com/", nil)

	if ok := pm.ServeProxy(service, w, r); !ok {
		t.Fatal("expected ServeProxy to find the service")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 from backend, got %d", w.Code)
	}
}
