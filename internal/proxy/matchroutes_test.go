package proxy

import (
	"net/http"
	"testing"

	"github.com/asenalabs/asena/internal/rule"
	"go.uber.org/zap/zaptest"
)

// mustParseRule is a small test helper. If the rule string fails to parse,
// that means the test itself is written wrong, so we fail right away
// instead of returning an error to check.
func mustParseRule(t *testing.T, r string) rule.Node {
	t.Helper()
	node, err := rule.ParseRule(r)
	if err != nil {
		t.Fatalf("test setup: failed to parse rule %q: %v", r, err)
	}
	return node
}

func TestMatchRouter_NoRoutersConfigured(t *testing.T) {
	pm := NewProxyManger(zaptest.NewLogger(t))

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	_, ok, err := pm.MatchRouter(req)
	if err != nil {
		t.Fatalf("did not expect error, got: %v", err)
	}
	if ok {
		t.Error("expected no match for an empty route slice")
	}
}

func TestMatchRouter_ValidHostRule(t *testing.T) {
	pm := NewProxyManger(zaptest.NewLogger(t))

	pm.RouterHolder.Store([]Route{
		{
			Name:    "api-router",
			Rule:    "Host(`example.com`)",
			Tree:    mustParseRule(t, "Host(`example.com`)"),
			Service: "api-service",
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
	if svc != "api-service" {
		t.Errorf("expected service api-service, got %s", svc)
	}
}

func TestMatchRouter_NoRouteMatches(t *testing.T) {
	pm := NewProxyManger(zaptest.NewLogger(t))

	pm.RouterHolder.Store([]Route{
		{
			Name:    "api-router",
			Rule:    "Host(`example.com`)",
			Tree:    mustParseRule(t, "Host(`example.com`)"),
			Service: "api-service",
		},
	})

	req, _ := http.NewRequest("GET", "http://other.com", nil)
	req.Host = "other.com"

	svc, ok, err := pm.MatchRouter(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected no match")
	}
	if svc != "" {
		t.Errorf("expected empty service, got %s", svc)
	}
}

// TestMatchRouter_FirstMatchWinsInStoredOrder checks one simple rule:
// MatchRouter does NOT sort or compare scores. It just trusts the order
// it was given. Here we store two routes that would both match, on
// purpose in the "wrong" order (least specific first), and check that
// MatchRouter still picks the first one. Real sorting is tested on its
// own in route_test.go.
func TestMatchRouter_FirstMatchWinsInStoredOrder(t *testing.T) {
	pm := NewProxyManger(zaptest.NewLogger(t))

	pm.RouterHolder.Store([]Route{
		{Name: "broad", Rule: "PathPrefix(`/`)", Tree: mustParseRule(t, "PathPrefix(`/`)"), Service: "broad-service"},
		{Name: "narrow", Rule: "PathPrefix(`/api`)", Tree: mustParseRule(t, "PathPrefix(`/api`)"), Service: "narrow-service"},
	})

	req, _ := http.NewRequest("GET", "http://example.com/api/users", nil)
	svc, ok, err := pm.MatchRouter(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected a match")
	}
	if svc != "broad-service" {
		t.Errorf("expected the FIRST stored route to win regardless of specificity, got %s", svc)
	}
}
