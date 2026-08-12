package rule

import (
	"net/http"
	"testing"
)

func req(t *testing.T, method, host, path string) *http.Request {
	t.Helper()
	r, err := http.NewRequest(method, "http://"+host+path, nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	return r
}

func TestParseRule_SingleMatcher(t *testing.T) {
	node, err := ParseRule("Host(`example.com`)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !node.Match(req(t, "GET", "example.com", "/")) {
		t.Error("expected example.com to match")
	}
	if node.Match(req(t, "GET", "other.com", "/")) {
		t.Error("expected other.com NOT to match")
	}
}

func TestParseRule_And(t *testing.T) {
	node, err := ParseRule("Host(`example.com`) && Method(`POST`)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !node.Match(req(t, "POST", "example.com", "/")) {
		t.Error("expected Host+Method match to succeed")
	}
	if node.Match(req(t, "GET", "example.com", "/")) {
		t.Error("expected mismatched method to fail the AND")
	}
}

// This is the most important test in this file. It proves "&&" binds
// tighter than "||", the same as in Go, instead of just reading left to
// right.
//
// Rule: Method(`GET`) || Method(`POST`) && Host(`nope.com`)
//
// Right way:  GET || (POST && Host(nope))
// Wrong way:  (GET || POST) && Host(nope)
//
// A GET request shows the difference. With the right grouping, GET
// matches right away, no matter the host. With the wrong grouping, GET
// would also need Host(`nope.com`) to be true - which it is not here. So
// a bug in precedence would flip this test's result.
func TestParseRule_AndBindsTighterThanOr(t *testing.T) {
	node, err := ParseRule("Method(`GET`) || Method(`POST`) && Host(`nope.com`)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !node.Match(req(t, "GET", "real.com", "/")) {
		t.Error("expected GET to match via the left OR branch regardless of host")
	}
	if node.Match(req(t, "POST", "real.com", "/")) {
		t.Error("expected POST to real.com NOT to match: right branch requires Host(`nope.com`)")
	}
	if !node.Match(req(t, "POST", "nope.com", "/")) {
		t.Error("expected POST to nope.com to match via the right AND branch")
	}
}

func TestParseRule_ParensOverridePrecedence(t *testing.T) {
	// Same tokens as above, but explicit grouping now requires Host to
	// match no matter which method is used.
	node, err := ParseRule("(Method(`GET`) || Method(`POST`)) && Host(`real.com`)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !node.Match(req(t, "GET", "real.com", "/")) {
		t.Error("expected GET to real.com to match")
	}
	if node.Match(req(t, "GET", "other.com", "/")) {
		t.Error("expected GET to other.com NOT to match: grouped OR still needs Host(`real.com`)")
	}
}

func TestParseRule_Not(t *testing.T) {
	node, err := ParseRule("Host(`example.com`) && !Method(`DELETE`)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !node.Match(req(t, "GET", "example.com", "/")) {
		t.Error("expected non-DELETE request to example.com to match")
	}
	if node.Match(req(t, "DELETE", "example.com", "/")) {
		t.Error("expected DELETE request to example.com NOT to match")
	}
}

func TestParseRule_DeeplyNestedFromDesignDoc(t *testing.T) {
	node, err := ParseRule("Host(`example.com`) && (PathPrefix(`/v2`) || !Method(`DELETE`))")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Wrong host: never matches, regardless of the right-hand side.
	if node.Match(req(t, "GET", "other.com", "/v2/x")) {
		t.Error("expected wrong host NOT to match")
	}
	// Right host, path under /v2: matches via PathPrefix.
	if !node.Match(req(t, "DELETE", "example.com", "/v2/x")) {
		t.Error("expected /v2 path to match via PathPrefix even on DELETE")
	}
	// Right host, path outside /v2, non-DELETE: matches via !Method(DELETE).
	if !node.Match(req(t, "GET", "example.com", "/v1/x")) {
		t.Error("expected non-/v2 GET to match via the NOT branch")
	}
	// Right host, path outside /v2, DELETE: both OR branches fail.
	if node.Match(req(t, "DELETE", "example.com", "/v1/x")) {
		t.Error("expected non-/v2 DELETE NOT to match either OR branch")
	}
}

func TestParseRule_Errors(t *testing.T) {
	cases := []string{
		"Host(`a.com`) &&",            // dangling operator
		"(Host(`a.com`)",              // unclosed group
		"Host(`a.com`) Host(`b.com`)", // missing operator between two exprs
		"Frobnicate(`x`)",             // unknown matcher
		"",                            // empty rule
	}
	for _, rule := range cases {
		if _, err := ParseRule(rule); err == nil {
			t.Errorf("ParseRule(%q): expected an error, got nil", rule)
		}
	}
}
