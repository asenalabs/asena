package proxy

import (
	"net/http"
	"testing"

	"github.com/asenalabs/asena/internal/config"
	"go.uber.org/zap/zaptest"
)

func TestMatchRouter_NoRoutersConfigured(t *testing.T) {
	pm := NewProxyManger(zaptest.NewLogger(t))

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	_, ok, err := pm.MatchRouter(req)
	if err != nil {
		t.Fatalf("did not expect error, got: %v", err)
	}
	if ok {
		t.Error("expected no match for empty router map")
	}
}

func TestMatchRouter_ValidHostRule(t *testing.T) {
	pm := NewProxyManger(zaptest.NewLogger(t))

	serviceName := "api-service"
	rule := "Host(`example.com`)"
	pm.RouterHolder.Store(map[string]*config.RoutersCfg{
		"api-router": {
			Rule:    &rule,
			Service: &serviceName,
		},
	})

	req, _ := http.NewRequest("GET", "http://example.com/path", nil)
	req.Host = "example.com"

	svc, ok, err := pm.MatchRouter(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected rule to match")
	}
	if svc != serviceName {
		t.Errorf("expected service %s, got %s", serviceName, svc)
	}
}

func TestMatchRouter_RuleInvalid(t *testing.T) {
	pm := NewProxyManger(zaptest.NewLogger(t))

	rule := "InvalidRule"
	service := "api"
	pm.RouterHolder.Store(map[string]*config.RoutersCfg{
		"bad-router": {
			Rule:    &rule,
			Service: &service,
		},
	})

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	_, ok, err := pm.MatchRouter(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected no match for invalid rule")
	}
}

func TestMatchRouter_ServiceNil(t *testing.T) {
	pm := NewProxyManger(zaptest.NewLogger(t))

	rule := "Host(`example.com`)"
	pm.RouterHolder.Store(map[string]*config.RoutersCfg{
		"nil-service-router": {
			Rule: &rule,
			// Service deliberately nil
		},
	})

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	svc, ok, err := pm.MatchRouter(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected match to fail since service is nil")
	}
	if svc != "" {
		t.Errorf("expected empty service, got %s", svc)
	}
}

func TestEvaluateConditions(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req.Host = "example.com"

	ok, err := evaluateConditions("Host(`example.com`)", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected condition to be true")
	}

	// invalid format
	_, err = evaluateConditions("Host example.com", req)
	if err == nil {
		t.Error("expected error for invalid format")
	}

	// unsupported matcher
	_, err = evaluateConditions("Path(`/api`)", req)
	if err == nil {
		t.Error("expected error for unsupported matcher")
	}
}

func TestHostMatcher(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req.Host = "example.com"

	if !hostMatcher("Host(`example.com`)", req) {
		t.Error("expected host to match")
	}

	// port bilan
	req.Host = "example.com:8080"
	if !hostMatcher("Host(`example.com`)", req) {
		t.Error("expected host to match even with port")
	}

	// mismatch
	req.Host = "wrong.com"
	if hostMatcher("Host(`example.com`)", req) {
		t.Error("expected host mismatch")
	}
}

func TestExtractArgument(t *testing.T) {
	arg := extractArgument("Host(`example.com`)")
	if arg != "example.com" {
		t.Errorf("expected 'example.com', got %s", arg)
	}

	// invalid format
	if extractArgument("Host(example.com)") != "" {
		t.Error("expected empty string for invalid format")
	}
}
