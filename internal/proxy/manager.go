package proxy

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/asenalabs/asena/internal/config"
	"github.com/asenalabs/asena/internal/proxy/balancer"
	"github.com/asenalabs/asena/pkg/logger"
	"go.uber.org/zap"
)

// balancerResultKey is the context key for balancerResult. It's an unexported
// empty struct type, so nothing outside this package can collide with it or
// read/write it directly, only the helpers below can.
type balancerResultKey struct{}

// balancerResult is a small mutable "box" carried on the request context.
//
// Why a box instead of just storing values with context.WithValue at the point we
// learn them? Because httputil.ReverseProxy clones the request *before*
// calling Rewrite, and its ErrorHandler is later called with the ORIGINAL request,
// not the clone. A new context value created inside Rewrite would only be visible
// on the clone, so the error path would never see it. A box created up front and
// shared by both is visible everywhere, and Rewrite just fills in its fields.
type balancerResult struct {
	server    *config.ServerCfg
	startTime time.Time
}

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
	pm.RouterHolder.Store([]Route{})

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

	// Read and sort all rules once, here, at reload time.
	newRouters := compileRoutes(cfg.Routers, pm.logg)

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

	rp.Rewrite = func(preq *httputil.ProxyRequest) {
		server := bl.Next(preq.In)
		if server == nil || server.URL == nil {
			pm.logg.Warn("No server available for proxy",
				zap.String("url", preq.In.URL.String()))
			return
		}

		// If ServeProxy set up a result box on this request, record which server we picked and when,
		// so ModifyResponse/ErrorHandler can report back to the balancer once the request finishes.
		if result, ok := preq.In.Context().Value(balancerResultKey{}).(*balancerResult); ok {
			result.server = server
			result.startTime = time.Now()
		}

		target, err := url.Parse(*server.URL)
		if err != nil {
			pm.logg.Warn("Invalid server URL",
				zap.String("url", *server.URL),
				zap.Error(err))
			return
		}

		preq.SetURL(target)

		if l.PassHostHeader != nil && *l.PassHostHeader {
			preq.Out.Host = target.Host
		}

		preq.SetXForwarded()
	}

	rp.ModifyResponse = func(resp *http.Response) error {
		reportDone(bl, resp.Request, nil)
		applyStickyCookie(bl, resp)

		resp.Header.Set("X-Content-Type-Options", "nosniff")
		resp.Header.Set("X-Frame-Options", "DENY")

		if resp.StatusCode >= http.StatusBadRequest {
			pm.logg.Warn("Proxy response error", zap.Int("status_code", resp.StatusCode), zap.String("service", resp.Request.URL.Host), zap.String("url", resp.Request.URL.String()))
		}

		return nil
	}

	return rp, nil
}

// reportDone reads the balancer result box of r's context (if ServeProxy set one up)
// and reports the outcome back to the balancer. It's called from both ModifyResponse
// (success) and ErrorHandler (failure), so every request reports exactly once, no
// matter how it ends.
func reportDone(bl balancer.Balancer, r *http.Request, err error) {
	result, ok := r.Context().Value(balancerResultKey{}).(*balancerResult)
	if !ok || result.server == nil {
		return
	}

	bl.Done(result.server, time.Since(result.startTime), err)
}

// applyStickyCookie lets a balancer write something onto a successful response - currently
// only used by Sticky Sessions, to set the cookie that pins a client to the server that
// just handled their request.
//
// This only runs from ModifyResponse (the success path), not ErrorHandler:
// we don't want to freshly pin a client to a server that just failed.
func applyStickyCookie(bl balancer.Balancer, resp *http.Response) {
	setter, ok := bl.(balancer.StickyCookieSetter)
	if !ok {
		return
	}

	result, ok := resp.Request.Context().Value(balancerResultKey{}).(*balancerResult)
	if !ok || result.server == nil {
		return
	}

	setter.SetStickyCookie(resp.Header, result.server)
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

// ServeProxy serves r through the named service's reverse proxy.
//
// This is the only place we set up the balancer result box, and we do it BEFORE calling ServeHTTP.
//
//	That matters: httputil.ReverseProxy clones the request internally, and the clone starts out
//
// sharing the same context as the original. Setting the box up front, before the clone happens, is what
// lets Rewrite (which sees the clone) and ErrorHandler (which sees the original) both reach the same
// box. See balancerResult's doc comment for the full story.
//
// Callers that used to do `rp, _ := pm.GetProxy(name); rp.ServeHTTP(w, r)` should call this instead.
func (pm *Manager) ServeProxy(serviceName string, w http.ResponseWriter, r *http.Request) bool {
	rp, ok := pm.GetProxy(serviceName)
	if !ok || rp == nil {
		return false
	}

	ctx := context.WithValue(r.Context(), balancerResultKey{}, &balancerResult{})
	rp.ServeHTTP(w, r.WithContext(ctx))
	return true
}
