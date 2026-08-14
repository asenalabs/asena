package balancer

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/asenalabs/asena/internal/config"
)

type RoundRobin struct {
	mu      sync.RWMutex
	servers []*config.ServerCfg
	counter uint64
}

func NewRoundRobin(servers []*config.ServerCfg) *RoundRobin {
	return &RoundRobin{
		servers: servers,
	}
}

// Next ignores r: Round Robin doesn't care which client is asking,
// it just rotates throught the server list
func (rr *RoundRobin) Next(r *http.Request) *config.ServerCfg {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	l := len(rr.servers)
	if l == 0 {
		return nil
	}

	pos := atomic.AddUint64(&rr.counter, 1)
	return rr.servers[pos%uint64(l)]
}

// Done is a no-op: Round Robin doesn't track active connections or response
// time, so there's nothing to update when a request finishes.
func (rr *RoundRobin) Done(server *config.ServerCfg, drtn time.Duration, err error) {}
