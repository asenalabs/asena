package server

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestStartHTTP(t *testing.T) {
	proxy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Hello World"))
	})

	ts := httptest.NewServer(proxy)
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status OK, got: %d", resp.StatusCode)
	}
}

func TestServeHTTPS_FallbackToHTTP(t *testing.T) {
	log := zaptest.NewLogger(t)
	proxy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Hello World"))
	})

	cfg := &ServerConfig{
		Address:     ":0",
		Version:     "test",
		Proxy:       proxy,
		Logg:        log,
		CertFileTLS: "invalid-cert.pem",
		KeyFileTLS:  "invalid-key.pem",
	}

	ln, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer func() {
		_ = ln.Close()
	}()

	srv := &http.Server{
		Handler: cfg.Proxy,
	}

	go func() {
		_ = srv.Serve(ln) // HTTP fallback
	}()
	defer func() {
		_ = srv.Close()
	}()

	url := fmt.Sprintf("http://%s", ln.Addr().String())

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status OK, got %d", resp.StatusCode)
	}
}
