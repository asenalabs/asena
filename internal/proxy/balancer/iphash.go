package balancer

import (
	"hash/fnv"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/asenalabs/asena/internal/config"
)

// IPHash routes each client to the same server every time, based on a hash of the client's IP address.
// Useful when a server holds per-client state in memory (a websocket, an in-memory cache) and you want
// the same client landing on the same server, without relying on cookies.
//
// Honest limitation: this uses hash % number-of-servers, which is simple but has a real weakness - adding
// or removing a server changes the modulo for almost every client, reshuffling most of them to a different
// backend. Consistent Hashing exists specifically to fix that; IP Hash is the simpler version worth
// understanding first.
type IPHash struct {
	servers []*config.ServerCfg
	// fallback is a plain round-robin counter, used only when we can't
	// determine a client IP at all - so "unknown" clients still spread
	// across servers instead of piling onto one.
	fallback uint64
}

// NewIPHash builds an IPHash from the configured servers. The list never changes after construction,
// so - unlike every mutable-state balancer we've built so far - Next() doesn't need a mutex: reading
// a slice concurrently is safe as long as nothing writes to it after setup.
func NewIPHash(servers []*config.ServerCfg) *IPHash {
	return &IPHash{servers: servers}
}

// Next hashes the client's IP and uses it to pick a server.
func (ih *IPHash) Next(r *http.Request) *config.ServerCfg {
	l := len(ih.servers)
	if l == 0 {
		return nil
	}

	ip := clientIP(r)
	if ip == "" {
		pos := atomic.AddUint64(&ih.fallback, 1)
		return ih.servers[pos%uint64(l)]
	}

	h := fnv.New32a()
	h.Write([]byte(ip))
	return ih.servers[h.Sum32()%uint32(l)]
}

// Done is a no-op: IP Hash doesn't track connections or response time, it
// deterministically maps clients to servers based on IP alone.
func (ih *IPHash) Done(server *config.ServerCfg, duration time.Duration, err error) {}

// clientIP extracts just the IP portion (no port) from a request's
// RemoteAddr. Returns "" if r is nil or no address is available at all.
func clientIP(r *http.Request) string {
	if r == nil || r.RemoteAddr == "" {
		return ""
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// No port to strip, for whatever reason - use the raw value
		// rather than failing outright. Still deterministic.
		return r.RemoteAddr
	}
	return host
}
