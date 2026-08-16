package balancer

import (
	"net/http"
	"sync"
	"time"

	"github.com/asenalabs/asena/internal/config"
)

// connServer pairs a server with how many requests are currently in
// flight to it.
type connServer struct {
	server *config.ServerCfg
	active int64
}

// LeastConnections sends each request to whichever server currently has the
// fewest in-flight requests. Unlike Round Robin or Weighted Round Robin, this
// needs to know when a request *finishes*, not just when it starts - that's what
// Done() is for.
type LeastConnections struct {
	mu      sync.Mutex
	servers []*connServer
	// byServer gives Done() an O(1) way to find the right counter, instead
	// of scanning the slice every time a request finishes.
	byServer map[*config.ServerCfg]*connServer
}

func NewLeastConnections(servers []*config.ServerCfg) *LeastConnections {
	lc := &LeastConnections{
		servers:  make([]*connServer, 0, len(servers)),
		byServer: make(map[*config.ServerCfg]*connServer, len(servers)),
	}

	for _, s := range servers {
		cs := &connServer{server: s}
		lc.servers = append(lc.servers, cs)
		lc.byServer[s] = cs
	}

	return lc
}

// Next ignores r: Least Connections doesn't care who's asking, only how busy each server
//
//	currently is.
//
// It scans for the lowest active count (ties go to whichever server comes first in the list,
// same tie-breaking style as WeightedRoundRobin), then immediately increments that server's
// count - the count has to go up here, at pick time, not later, or two requests arriving at
// once could both see the same 'least busy" server and both get sent to it.
func (lc *LeastConnections) Next(r *http.Request) *config.ServerCfg {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if len(lc.servers) == 0 {
		return nil
	}

	picked := lc.servers[0]
	for _, cs := range lc.servers[1:] {
		if cs.active < picked.active {
			picked = cs
		}
	}

	picked.active++
	return picked.server
}

// Done decrements the finished server's active count, freeing up its slot for the next comparison
// in Next(). This runs whether the request succeeded or failed (err is ignored) - either way, the
// connection to that server is no longer in flight. drtn (duration) is ignored too: Least Connections
// only cares about *how many* requests are active, not how long they take (that's Least Time's job).
// If server isn't one we know about (for example, a stale pointer from a balancer that's since been
// replaced by a config reload), this is a no-op rather than a panic.
func (lc *LeastConnections) Done(server *config.ServerCfg, drtn time.Duration, err error) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	cs, ok := lc.byServer[server]
	if !ok || cs.active <= 0 {
		return
	}
	cs.active--
}
