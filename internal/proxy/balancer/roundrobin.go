package balancer

import (
	"sync"
	"sync/atomic"

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

func (rr *RoundRobin) Next() *config.ServerCfg {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	l := len(rr.servers)
	if l == 0 {
		return nil
	}

	pos := atomic.AddUint64(&rr.counter, 1)
	return rr.servers[pos%uint64(l)]
}
