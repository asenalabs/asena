package proxy

import (
	"testing"

	"github.com/asenalabs/asena/internal/config"
	"go.uber.org/zap/zaptest"
)

func strPtr(s string) *string { return &s }

func TestCompileRoutes_SkipsNilRule(t *testing.T) {
	routers := map[string]*config.RoutersCfg{
		"bad": {Service: strPtr("svc")}, // Rule left nil
	}
	routes := compileRoutes(routers, zaptest.NewLogger(t))
	if len(routes) != 0 {
		t.Errorf("expected a nil-Rule router to be skipped, got %d routes", len(routes))
	}
}

func TestCompileRoutes_SkipsNilService(t *testing.T) {
	routers := map[string]*config.RoutersCfg{
		"bad": {Rule: strPtr("Host(`a.com`)")}, // Service left nil
	}
	routes := compileRoutes(routers, zaptest.NewLogger(t))
	if len(routes) != 0 {
		t.Errorf("expected a nil-Service router to be skipped, got %d routes", len(routes))
	}
}

func TestCompileRoutes_SkipsInvalidRuleButKeepsOthers(t *testing.T) {
	// The important behavior here: ONE bad rule in the config must not prevent the other,
	// valid routers from being compiled and served.
	routers := map[string]*config.RoutersCfg{
		"bad":  {Rule: strPtr("NotAMatcher(`x`)"), Service: strPtr("svc-bad")},
		"good": {Rule: strPtr("Host(`a.com`)"), Service: strPtr("svc-good")},
	}
	routes := compileRoutes(routers, zaptest.NewLogger(t))
	if len(routes) != 1 {
		t.Fatalf("expected exactly 1 compiled route, got %d", len(routes))
	}
	if routes[0].Name != "good" {
		t.Errorf("expected the surviving route to be 'good', got %q", routes[0].Name)
	}
}

func TestCompileRoutes_SortedMostSpecificFirst(t *testing.T) {
	routers := map[string]*config.RoutersCfg{
		"host-only":       {Rule: strPtr("Host(`a.com`)"), Service: strPtr("svc-1")},                  // spec 15
		"host-and-method": {Rule: strPtr("Host(`a.com`) && Method(`GET`)"), Service: strPtr("svc-2")}, // spec 15+25+10=50
		"header-only":     {Rule: strPtr("Header(`X-Key`, `v`)"), Service: strPtr("svc-3")},           // spec 30
	}
	routes := compileRoutes(routers, zaptest.NewLogger(t))
	if len(routes) != 3 {
		t.Fatalf("expected 3 compiled routes, got %d", len(routes))
	}

	// Descending specificity: host-and-method (50) > header-only (30) > host-only (15).
	wantOrder := []string{"host-and-method", "header-only", "host-only"}
	for i, name := range wantOrder {
		if routes[i].Name != name {
			t.Errorf("position %d: expected %q, got %q (full order: %+v)", i, name, routes[i].Name, routeNames(routes))
		}
	}
}

func TestCompileRoutes_TieBreaksBySortedName(t *testing.T) {
	// Both rules score the same (a single Host matcher, 15 either way). So the only thing that
	// decides the order is the name tie-break. We run this 20 times to make sure Go's random
	// map order can't sneak in and change the result.
	routers := map[string]*config.RoutersCfg{
		"zebra": {Rule: strPtr("Host(`z.com`)"), Service: strPtr("svc-z")},
		"alpha": {Rule: strPtr("Host(`a.com`)"), Service: strPtr("svc-a")},
	}

	for i := 0; i < 20; i++ {
		routes := compileRoutes(routers, zaptest.NewLogger(t))
		if len(routes) != 2 || routes[0].Name != "alpha" || routes[1].Name != "zebra" {
			t.Fatalf("iteration %d: expected deterministic order [alpha, zebra], got %+v", i, routeNames(routes))
		}
	}
}

func routeNames(routes []Route) []string {
	names := make([]string, len(routes))
	for i, r := range routes {
		names[i] = r.Name
	}
	return names
}
