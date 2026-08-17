package balancer

import (
	"hash/fnv"
	"net/http"
	"strconv"
	"time"

	"github.com/asenalabs/asena/internal/config"
)

const (
	defaultStickyCookieName = "asena_sticky"
	defaultStickyCookieTTL  = 1 * time.Hour
)

// StickySession pins a client to the same server across requests using a cookie, instead of
// hashing the client's IP (IP Hash) or anything about the request itself.
//
// Known limitation: Asena doesn't have health checking yet, so if a client is pinned to a
// server that goes down, StickySession has no way to notice and move them elsewhere - they'll
// keep hitting the same broken server until their cookie expires. Worth revisiting once
// health checks exist.
type StickySession struct {
	servers []*config.ServerCfg
	// byHash maps a hashed server URL back to the actual server, so a
	// cookie value can be looked up without exposing the real backend URL
	// to the client.
	byHash map[string]*config.ServerCfg
	// fallback picks a server when there's no valid sticky cookie yet
	// (first visit, expired cookie, or a cookie pointing at a server
	// that's no longer in the config). Round Robin spreads first-time
	// visitors evenly instead of piling them onto one server.
	fallback   *RoundRobin
	cookieName string
	cookieTTL  time.Duration
}

func NewStickySession(servers []*config.ServerCfg) *StickySession {
	ss := &StickySession{
		servers:    servers,
		byHash:     make(map[string]*config.ServerCfg, len(servers)),
		fallback:   NewRoundRobin(servers),
		cookieName: defaultStickyCookieName,
		cookieTTL:  defaultStickyCookieTTL,
	}

	for _, s := range servers {
		if s.URL == nil {
			continue
		}
		ss.byHash[hashServerURL(*s.URL)] = s
	}

	return ss
}

// Next checks for an existing sticky cookie and, if it points to a server we still know about,
// returns that server. Otherwise it falls back to Round Robin - this covers first-time visitors,
// expired cookies, and cookies pointing at a server that's been removed from config since.
//
// byHash and servers are both built once in NewStickySession and never modified afterwards,
// so - like IPHash - this doesn't need a mutex of its own. fallback (RoundRobin) manages its
// own locking internally.
func (ss *StickySession) Next(r *http.Request) *config.ServerCfg {
	if r != nil {
		if cookie, err := r.Cookie(ss.cookieName); err == nil {
			if srv, ok := ss.byHash[cookie.Value]; ok {
				return srv
			}
		}
	}

	return ss.fallback.Next(r)
}

// Done is a no-op for now: StickySession doesn't track connections or response time itself.
// If the fallback strategy is ever swapped from Round Robin to something that does (Least
// Connections, for example), this would need to forward to it.
func (ss *StickySession) Done(server *config.ServerCfg, duration time.Duration, err error) {}

// SetStickyCookie implements the optional StickyCookieSetter interface. manager.go calls this
// after a successful response, so the client's cookie always reflects whichever server actually
// handled the request - including refreshing the cookie's expiry on every visit (a "sliding"
// session, not a fixed one).
func (ss *StickySession) SetStickyCookie(header http.Header, server *config.ServerCfg) {
	if server == nil || server.URL == nil {
		return
	}

	cookie := &http.Cookie{
		Name:     ss.cookieName,
		Value:    hashServerURL(*server.URL),
		Path:     "/",
		MaxAge:   int(ss.cookieTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	header.Add("Set-Cookie", cookie.String())
}

// hashServerURL turns a server URL into an opaque cookie value.
//
// Two reasons not to just use the URL directly: it would expose internal backend addresses (hostnames,
// internal IPs) to whoever inspects their own cookies, and it's a bit longer than it needs to be.
// A malicious client editing their own cookie can only ever "claim" a hash that matches one of our real,
// currently-configured servers (see byHash in Next) - there's no way to use this to reach anything that
// isn't already a valid backend, so this doesn't need to be cryptographically signed.
//
// 64-bit FNV-1a, not the 32-bit version IPHash uses: a collision here would silently bind a client to the
// wrong server, which is a different (worse) failure mode than IP Hash's "two clients land in the same
// bucket", which is normal and expected. The extra bits make that already-tiny risk even smaller.
func hashServerURL(url string) string {
	h := fnv.New64a()
	h.Write([]byte(url))
	return strconv.FormatUint(h.Sum64(), 16)
}
