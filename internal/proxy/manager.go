package proxy

import (
	"crypto/tls"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"

	"github.com/asenalabs/asena/internal/config"
	"github.com/asenalabs/asena/internal/proxy/balancer"
	"github.com/asenalabs/asena/pkg/logger"
	"go.uber.org/zap"
)

type Manager struct {
	ProxyHolder  atomic.Value
	RouterHolder atomic.Value
	mu           sync.RWMutex
	logg         *zap.Logger
}

func NewProxyManger(logg *zap.Logger) *Manager {
	pm := &Manager{
		logg: logg,
	}
	pm.ProxyHolder.Store(make(map[string]*httputil.ReverseProxy))
	pm.RouterHolder.Store(make(map[string]*config.RoutersCfg))

	return pm
}

func (pm *Manager) BuildReverseProxy(cfg *config.HTTPCfg, t *config.ProxyTransportCfg) {
	if cfg == nil || t == nil {
		pm.logg.Error("Proxy transport config is nil")
		return
	}

	newProxies := make(map[string]*httputil.ReverseProxy)
	for name, group := range cfg.Services {
		rp, err := pm.newReverseProxy(t, group.LoadBalancer)
		if err != nil {
			pm.logg.Error("Failed to build reverse proxy", zap.String("service", name), zap.Error(err))
		}

		newProxies[name] = rp
		pm.logg.Info("Reverse proxy built", zap.String("service", name), zap.String("algorithm", *group.LoadBalancer.Algorithm), zap.Int("services_count", len(group.LoadBalancer.Servers)))
	}

	newRouters := make(map[string]*config.RoutersCfg)
	for name, router := range cfg.Routers {
		newRouters[name] = router
		pm.logg.Info("Router registered", zap.String("router", name), zap.String("rule", *router.Rule))
	}

	pm.mu.Lock()
	pm.ProxyHolder.Store(newProxies)
	pm.RouterHolder.Store(newRouters)
	pm.mu.Unlock()
}

func (pm *Manager) newReverseProxy(t *config.ProxyTransportCfg, l *config.LoadBalancerCfg) (*httputil.ReverseProxy, error) {
	bl := balancer.New(*l.Algorithm, l.Servers)

	rp := &httputil.ReverseProxy{
		Transport:     newProxyTransport(t),
		FlushInterval: *l.FlashInterval,
		ErrorLog:      logger.MustZapToStdLoggerAtLevel(pm.logg, zap.WarnLevel),
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, e error) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.WriteHeader(http.StatusBadGateway) // 502 Bad Gateway

			resp := map[string]interface{}{
				"error":   "Service not available",
				"code":    http.StatusBadGateway,
				"message": "Please try again later.",
			}

			_ = json.NewEncoder(w).Encode(resp)
		},
	}

	rp.Director = func(req *http.Request) {
		server := bl.Next()
		if server == nil || server.URL == nil {
			pm.logg.Warn("No server available for proxy", zap.String("url", req.URL.String()))
		}

		target, err := url.Parse(*server.URL)
		if err != nil {
			pm.logg.Warn("Invalid server URL", zap.String("url", *server.URL), zap.Error(err))
			return
		}

		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host

		if l.PassHostHeader != nil && *l.PassHostHeader {
			req.Host = target.Host
		}
	}

	rp.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Set("X-Content-Type-Options", "nosniff")
		resp.Header.Set("X-Frame-Options", "DENY")

		if resp.StatusCode >= http.StatusBadRequest {
			pm.logg.Warn("Proxy response error", zap.Int("status_code", resp.StatusCode), zap.String("service", resp.Request.URL.Host), zap.String("url", resp.Request.URL.String()))
		}

		return nil
	}

	return rp, nil
}

func newProxyTransport(t *config.ProxyTransportCfg) *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   *t.DailTimeout,
			KeepAlive: *t.DailKeepalive,
		}).DialContext,
		ForceAttemptHTTP2:     *t.ForceHTTP2,
		MaxIdleConns:          *t.MaxIdleConn,
		MaxConnsPerHost:       *t.MaxIdleConnPerHost,
		IdleConnTimeout:       *t.IdleConnTimeout,
		TLSHandshakeTimeout:   *t.TLSHandshakeTimeout,
		ExpectContinueTimeout: *t.ExpectContinueTimeout,
		TLSClientConfig: &tls.Config{
			MinVersion: *t.TLSMinVersion,
		},
	}
}

func (pm *Manager) GetProxy(serviceName string) (*httputil.ReverseProxy, bool) {
	value := pm.ProxyHolder.Load()
	proxies, ok := value.(map[string]*httputil.ReverseProxy)
	if !ok || proxies == nil {
		return nil, false
	}

	rp, exists := proxies[serviceName]
	return rp, exists
}
