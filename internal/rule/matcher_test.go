package rule

import (
	"net/http"
	"testing"
)

func TestHostNode_MatchIgnoresPortAndCase(t *testing.T) {
	node := &HostNode{host: "example.com"}

	cases := []struct {
		host string
		want bool
	}{
		{"example.com", true},
		{"EXAMPLE.COM", true},
		{"example.com:8080", true},
		{"other.com", false},
	}
	for _, c := range cases {
		r := &http.Request{Host: c.host}
		if got := node.Match(r); got != c.want {
			t.Errorf("Host=%q: Match() = %v, want %v", c.host, got, c.want)
		}
	}
}

func TestPathPrefixNode_Match(t *testing.T) {
	node := &PathPrefixNode{prefix: "/api/v2"}

	r, _ := http.NewRequest("GET", "http://x/api/v2/users", nil)
	if !node.Match(r) {
		t.Error("expected /api/v2/users to match prefix /api/v2")
	}

	r2, _ := http.NewRequest("GET", "http://x/api/v1/users", nil)
	if node.Match(r2) {
		t.Error("expected /api/v1/users NOT to match prefix /api/v2")
	}
}

func TestPathPrefixNode_SpecificityGrowsWithLength(t *testing.T) {
	short := &PathPrefixNode{prefix: "/api"}
	long := &PathPrefixNode{prefix: "/api/v2/users"}
	if long.Specificity() <= short.Specificity() {
		t.Errorf("expected longer prefix to score higher: short=%d long=%d",
			short.Specificity(), long.Specificity())
	}
}

func TestMethodNode_Match(t *testing.T) {
	node := &MethodNode{method: "POST"}
	r, _ := http.NewRequest("POST", "http://x/", nil)
	if !node.Match(r) {
		t.Error("expected POST request to match Method(`POST`)")
	}
	r2, _ := http.NewRequest("GET", "http://x/", nil)
	if node.Match(r2) {
		t.Error("expected GET request NOT to match Method(`POST`)")
	}
}

func TestHeaderNode_MatchIsKeyCaseInsensitive(t *testing.T) {
	node := &HeaderNode{key: "X-Api-Key", val: "secret123"}
	r, _ := http.NewRequest("GET", "http://x/", nil)
	r.Header.Set("x-api-key", "secret123") // client sent lowercase

	if !node.Match(r) {
		t.Error("expected header match to be case-insensitive on the key")
	}

	r.Header.Set("x-api-key", "wrong")
	if node.Match(r) {
		t.Error("expected mismatched header value NOT to match")
	}
}

func TestBuildLeaf_UnknownMatcher(t *testing.T) {
	_, err := buildLeaf("Frobnicate(`x`)")
	if err == nil {
		t.Fatal("expected an error for an unknown matcher name")
	}
}

func TestBuildLeaf_WrongArgCount(t *testing.T) {
	_, err := buildLeaf("Host(`a.com`, `b.com`)")
	if err == nil {
		t.Fatal("expected an error: Host takes exactly 1 argument")
	}
}

func TestParseArgs_CommaInsideBacktickIsPreserved(t *testing.T) {
	// The exact case that a naive strings.Split(s, ",") would break.
	args, err := parseArgs("`X-Roles`, `admin,editor`")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(args) != 2 || args[0] != "X-Roles" || args[1] != "admin,editor" {
		t.Errorf("unexpected args: %+v", args)
	}
}

func TestParseArgs_UnterminatedBacktick(t *testing.T) {
	_, err := parseArgs("`unterminated")
	if err == nil {
		t.Fatal("expected an error for an unterminated backtick")
	}
}

func TestParseArgs_NotBacktickQuoted(t *testing.T) {
	_, err := parseArgs("bareword")
	if err == nil {
		t.Fatal("expected an error for an argument with no backticks")
	}
}
