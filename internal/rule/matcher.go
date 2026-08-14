package rule

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

// buildLeaf turns one matcher call, like "Host(`example.com`)", into a real matcher. This is the
// one place that turns a name into a type, so it is also the best place to give a clear error for
// a bad name or a wrong number of arguments.
func buildLeaf(raw string) (Node, error) {
	name, args, err := parseFunc(raw)
	if err != nil {
		return nil, err
	}

	switch name {
	case "Host":
		if len(args) != 1 {
			return nil, fmt.Errorf("rule: Host expects exactly 1 argument, got %d in %q", len(args), raw)
		}
		// Lowercase it once, here, so Match never has to think about case.
		return &HostNode{host: strings.ToLower(args[0])}, nil

	case "PathPrefix":
		if len(args) != 1 {
			return nil, fmt.Errorf("rule: PathPrefix expects exactly 1 argument, got %d in %q", len(args), raw)
		}
		return &PathPrefixNode{prefix: args[0]}, nil

	case "Path":
		if len(args) != 1 {
			return nil, fmt.Errorf("rule: Path expects exactly 1 argument, got %d in %q", len(args), raw)
		}
		return &PathNode{path: args[0]}, nil

	case "Method":
		if len(args) != 1 {
			return nil, fmt.Errorf("rule: Method expects exactly 1 argument, got %d in %q", len(args), raw)
		}
		return &MethodNode{method: strings.ToUpper(args[0])}, nil

	case "Header":
		if len(args) != 2 {
			return nil, fmt.Errorf("rule: Header expects exactly 2 arguments (key, value), got %d in %q", len(args), raw)
		}
		return &HeaderNode{key: args[0], val: args[1]}, nil

	case "ClientIP":
		if len(args) != 1 {
			return nil, fmt.Errorf("rule: ClientIP expects exactly 1 argument, got %d in %q", len(args), raw)
		}
		return newClientIPNode(args[0])

	default:
		return nil, fmt.Errorf("rule: unknown matcher %q in %q (supported: Host, PathPrefix, Method, Header)", name, raw)
	}
}

// HostNode matches the request's Host header. It ignores the port and the letter case,
// since hostnames don't care about case, and the port is often present even when
// the operator only means the name.
type HostNode struct{ host string }

func (n *HostNode) Match(r *http.Request) bool {
	h := strings.ToLower(r.Host)
	if i := strings.IndexByte(h, ':'); i != -1 {
		h = h[:i]
	}
	return h == n.host
}

// Specificity, a Host match only narrows down which site. It says nothing about which
// path, method, or header - so it scores lower than the others below.
func (n *HostNode) Specificity() int { return 15 }

// PathPrefixNode matches when the request path starts with prefix.
type PathPrefixNode struct{ prefix string }

func (n *PathPrefixNode) Match(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, n.prefix)
}

// Specificity grows with the length of the prefix. "/api/v2/users" is a
// much narrower claim than "/", even though both use PathPrefix. If we
// gave every PathPrefix the same flat score, a short, broad prefix and
// a long, exact one would look equally specific, which is wrong.
func (n *PathPrefixNode) Specificity() int { return 20 + len(n.prefix) }

// PathNode matches the request path is exactly equal to path. Unlike PathPrefixNode, nothing
// may come after it, "/health" matches only "/health", not "/health/live".
type PathNode struct {
	path string
}

func (n *PathNode) Match(r *http.Request) bool {
	return r.URL.Path == n.path
}

// Specificity uses the same length-based formula as PathPrefixNode, plus a flat +10 bonus.
func (n *PathNode) Specificity() int {
	return 30 + len(n.path)
}

// MethodNode matches the HTTP method (GET, POST, ...).
type MethodNode struct{ method string }

func (n *MethodNode) Match(r *http.Request) bool { return r.Method == n.method }

// Specificity, there are only a few HTTP methods in real use, so a Method match rules out more
// traffic than a Host match usually does, most setups have far more hostnames than methods.
func (n *MethodNode) Specificity() int { return 25 }

// HeaderNode matches when the named header has exactly this value.
type HeaderNode struct{ key, val string }

func (n *HeaderNode) Match(r *http.Request) bool {
	// Header.Get already ignores letter case on the key.
	return r.Header.Get(n.key) == n.val
}

// Specificity is the highest of the four. A header match needs the caller to know and send an
// exact value, so it's the narrowest and most deliberate check a request can carry.
func (n *HeaderNode) Specificity() int { return 30 }

// ClientIPNode matches when the request's source IP falls inside a range,
// or equals a single address exactly.
//
// Exactly one of the two fields below is set, decided once, when the rule is first read, in newClientIPNode.
// Match and Specificity only ever look at whichever one is non-nil, they never re-parse the original text.
type ClientIPNode struct {
	single net.IP     // set when the rule was one plain address, e.g. "203.0.113.5"
	ipNet  *net.IPNet // set when the rule was a CIDR range, e.g. "10.0.0.0/24"
}

// newClientIPNode decides, once, whether raw is a single address or a CIDR range, and parses it accordingly.
// A "/" in the text is the only single we need, IP addresses never contain one, CIDR ranges aslways do.
func newClientIPNode(raw string) (*ClientIPNode, error) {
	if strings.Contains(raw, "/") {
		_, ipNet, err := net.ParseCIDR(raw)
		if err != nil {
			return nil, fmt.Errorf("rule: invalid CIDR %q in ClientIP: %w", raw, err)
		}
		return &ClientIPNode{ipNet: ipNet}, nil
	}

	ip := net.ParseIP(raw)
	if ip == nil {
		return nil, fmt.Errorf("rule: invalid IP address %q in ClientIP", raw)
	}

	return &ClientIPNode{single: ip}, nil
}

// Match reads the client's address straight from the TCP connection (r.RemoteAddr), never from a header like X-Forwarded-For.
// A header is just text the client sent, anyone can put any value in it. RemoteAddr is not, it isthe actual socket Go accepted
// the connection on, so it cannot be spoofed the way a header can.
func (n *ClientIPNode) Match(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	if n.ipNet != nil {
		return n.ipNet.Contains(ip)
	}

	return ip.Equal(n.single)
}

// Specificity counts how many bits of the address are actually pinned down, the same as PathPrefixNode counting characters.
// A /24 range fixes 24 bits and leaves 8 free, covering 256 addresses; a/8 fixes only 8 bits and covers 16 million. The more bits fixed,
// the narrower the range, the higher the score, exactly like a longer PathPrefix.
//
// A single exact address (no "/" in the rule) fixes every bit there is, so it is treated the same as the narrowest possible range : /32 for
// an IPv4 address, /128 for IPv6.
func (n *ClientIPNode) Specificity() int {
	if n.ipNet != nil {
		ones, _ := n.ipNet.Mask.Size()
		return ones
	}

	if n.single.To4() != nil {
		return 32
	}

	return 128
}

// parseFunc splits a matcher call, like "Host(`example.com`)", into its
// name ("Host") and its arguments (["example.com"]).
func parseFunc(raw string) (name string, args []string, err error) {
	open := strings.IndexByte(raw, '(')
	if open == -1 || raw[len(raw)-1] != ')' {
		// This should never happen, the lexer already checked the shape.
		return "", nil, fmt.Errorf("rule: malformed matcher call %q", raw)
	}
	name = raw[:open]
	inner := raw[open+1 : len(raw)-1]

	args, err = parseArgs(inner)
	if err != nil {
		return "", nil, fmt.Errorf("rule: %w in %q", err, raw)
	}
	return name, args, nil
}

// parseArgs splits the text inside a matcher's parentheses into separate, unquoted arguments.
//
// A plain split on "," would break the day a value has a comma inside it, like Header(`X-Roles`, `admin,editor`).
// So we only treat, "," as a separator when we are not inside backticks.
func parseArgs(inner string) ([]string, error) {
	var args []string
	var current strings.Builder
	inBacktick := false

	for i := 0; i < len(inner); i++ {
		c := inner[i]
		switch {
		case c == '`':
			inBacktick = !inBacktick
			current.WriteByte(c)
		case c == ',' && !inBacktick:
			args = append(args, strings.TrimSpace(current.String()))
			current.Reset()
		default:
			current.WriteByte(c)
		}
	}
	if inBacktick {
		return nil, fmt.Errorf("unterminated backtick in argument list")
	}
	args = append(args, strings.TrimSpace(current.String()))

	unquoted := make([]string, len(args))
	for i, a := range args {
		if len(a) < 2 || a[0] != '`' || a[len(a)-1] != '`' {
			return nil, fmt.Errorf("argument %q is not backtick-quoted", a)
		}
		unquoted[i] = a[1 : len(a)-1]
	}
	return unquoted, nil
}
