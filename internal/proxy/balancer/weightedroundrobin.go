package balancer

import (
	"net/http"
	"sync"
	"time"

	"github.com/asenalabs/asena/internal/config"
)

// weightedServer pairs a server with the mutable state Smooth Weighted
// Round Robin needs to track between picks.
type weightedServer struct {
	server        *config.ServerCfg
	weight        int // fixed, comes from config and never changes
	currentWeight int //mutable, changes on every Next() call
}

// WeightedRoundRobin distributes requests proportionallly to each server's
// configured weight, using the "smooth weighted round robin" algorithm.
// A server with weight 5 gets roughtly 5x the traffic of a server with weight 1,
// but the extra requests are spread evenly through the sequence instead of arriving 5-in-a-row.
type WeightedRoundRobin struct {
	mu      sync.Mutex
	servers []*weightedServer
	total   int
}

// NewWeightedRoundRobin builds a WeightedRoundRobin from the configured servers.
// A server with no weight set (Weight == nil) defaults to 1, same as a plain Round Robin server.
// A weight of 0 or less is also treated as 1, a server with zero weight would never be picked,
// which almost always means a misconfiguration rather than  an international "disable this server".
func NewWeightedRoundRobin(servers []*config.ServerCfg) *WeightedRoundRobin {
	wrr := &WeightedRoundRobin{
		servers: make([]*weightedServer, 0, len(servers)),
	}

	for _, s := range servers {
		w := 1
		if s.Weight != nil && int(*s.Weight) > 0 {
			w = int(*s.Weight)
		}

		wrr.servers = append(wrr.servers, &weightedServer{
			server: s,
			weight: w,
		})
		wrr.total += w
	}

	return wrr
}

// Next ignores r, same as Round Robin: the weight comes from config, notfrom anything about the client asking.
//
// Every server's currentWeight grows by its own weight. Whoever ends up with the highest currentWeight is picked,
// then gets docked by the total weight of all servers. Run this enough times and you get a sequence where each
// server appears in proportion to its weight, spread out evenly.
func (wrr *WeightedRoundRobin) Next(r *http.Request) *config.ServerCfg {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	if len(wrr.servers) == 0 {
		return nil
	}

	var picked *weightedServer
	for _, s := range wrr.servers {
		s.currentWeight += s.weight
		if picked == nil || s.currentWeight > picked.currentWeight {
			picked = s
		}
	}

	picked.currentWeight -= wrr.total
	return picked.server
}

// Done is a no-op: like Round Robin, Weighted Round Robin doesn't track active connections or response time,
// so there's nothing to update when a request finishes.
func (wrr *WeightedRoundRobin) Done(server *config.ServerCfg, duration time.Duration, err error) {}
