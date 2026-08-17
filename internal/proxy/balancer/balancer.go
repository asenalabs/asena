package balancer

import (
	"net/http"
	"time"

	"github.com/asenalabs/asena/internal/config"
)

// Balancer picks which backend server should handle the next request.
//
// Next is called once per request, before it goes to a backend. It receives the original client request
// so algorithms like IP Hash or Sticky Sessions can look at the client IP or a cookie. Algorithms that
// don't need this (Round Robin, Weighted Round Robin) can just ignore r.
//
// Done is called once per request, after the response comes back or the request fails. Algorithms like
// Least Connections or Least Time use it to update their own state. Algorithms that don't need it leave
// the method body empty.
type Balancer interface {
	Next(r *http.Request) *config.ServerCfg
	Done(server *config.ServerCfg, drtn time.Duration, err error)
}

// StickyCookieSetter is an optional capability a Balancer can implement when it needs to write something
// onto the response - specifically, Sticky Sessions needs to set a cookie so the client comes back to the
// same server next time.
//
// This is deliberately NOT part of the Balancer interface itself: only one algorithm needs it, and adding
// an empty method to every other balancer just to satisfy an interface would be noise. Instead, manager.go
// checks for this with a type assertion (bl.(StickyCookieSetter)) and only calls it if the balancer actually
// implements it - the same pattern the standard library uses for http.Flusher and http.Hijacker.
type StickyCookieSetter interface {
	SetStickyCookie(header http.Header, server *config.ServerCfg)
}

func New(algorithm string, servers []*config.ServerCfg) Balancer {
	switch algorithm {
	case config.RoundRobin:
		return NewRoundRobin(servers)
	case config.WeightedRoundRobin:
		return NewWeightedRoundRobin(servers)
	case config.LeastConnections:
		return NewLeastConnections(servers)
	case config.LeastTime:
		return NewLeastTime(servers)
	case config.IPHash:
		return NewIPHash(servers)
	case config.StickySession:
		return NewStickySession(servers)
	default:
		return NewRoundRobin(servers) // default fallback
	}
}
