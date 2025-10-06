package handler

import (
	"net/http"

	"github.com/asenalabs/asena/internal/proxy"
	"go.uber.org/zap"
)

func RegisterRoutes(pm *proxy.Manager, mux *http.ServeMux, logg *zap.Logger) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serviceName, ok, err := pm.MatchRouter(r)
		if err != nil {
			logg.Error("failed to match router", zap.Error(err))
			http.Error(w, "router error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if !ok {
			logg.Warn("No router found", zap.String("path", r.URL.Path), zap.Error(err))
			http.NotFound(w, r)
			return
		}

		targetProxy, ok := pm.GetProxy(serviceName)
		if !ok {
			logg.Warn("No routing rule found for service", zap.String("service", serviceName))
			http.Error(w, "404 page not found", http.StatusNotFound)
			return
		}

		targetProxy.ServeHTTP(w, r)
	})
}
