package balancer

import (
	"hash/fnv"
	"net/http"
	"sort"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/asenalabs/asena/internal/config"
)

// consistentHashReplicas is how many points on the ring each server gets. More points = fairer
// load distribution across servers, at the cost of a slightly bigger ring to build and search.
//
// 100 is the commonly-cited default in blog posts (matching libketama), but testing it here
// against small server counts (2-4, realistic for a project this size) showed real
// imbalance - one server could end up with significantly more of the ring than the others
// just by chance of where its points landed. 150 measured more reliably balanced across several
// different server URL sets. This is a known, documented property of vanilla consistent hashing:
// variance shrinks as the number of real servers grows, so very small clusters will always show
// some visible (not catastrophic) unevenness no matter how many virtual points you add - more
// replicas helps, but doesn't erase it. Worth knowing rather than assuming the commonly-cited number
// is automatically right for any scale.
const consistentHashReplicas = 150

// ringPoint is one point on the hash ring. Everything between this point
// and the previous one (going clockwise) belongs to server.
type ringPoint struct {
	hash   uint32
	server *config.ServerCfg
}

// ConsistentHash routes each client to the same server, same goal as IPHash, but built to survive
// server list changes gracefully. IPHash's hash % server-count reshuffles almost every client when
// the server count changes; ConsistentHash's ring means adding or removing one server only moves
// the roughly 1/N of clients whose ring position was "owned" by that server - everyone else keeps
// landing on the same server they always did.
type ConsistentHash struct {
	ring    []ringPoint // sorted by hash, ascending
	servers []*config.ServerCfg
	// fallback mirrors IPHash: a plain round-robin counter, used only when
	// we can't determine a client IP at all.
	fallback uint64
}

// NewConsistentHash builds the ring once at construction. Like IPHash, the ring and server list never
// change afterwards, so Next() doesn't need a mutex - only the fallback counter is mutated, and that's
// done atomically.
func NewConsistentHash(servers []*config.ServerCfg) *ConsistentHash {
	ch := &ConsistentHash{servers: servers}

	for _, s := range servers {
		if s.URL == nil {
			continue
		}
		for i := 0; i < consistentHashReplicas; i++ {
			ch.ring = append(ch.ring, ringPoint{
				hash:   hashRingKey(*s.URL + "#" + strconv.Itoa(i)),
				server: s,
			})
		}
	}

	sort.Slice(ch.ring, func(i, j int) bool {
		return ch.ring[i].hash < ch.ring[j].hash
	})

	return ch
}

// Next hashes the client's IP (same extraction logic as IPHash - see clientIP in iphash.go) onto the ring,
// then finds the nearest server point clockwise from it via binary search.
func (ch *ConsistentHash) Next(r *http.Request) *config.ServerCfg {
	if len(ch.ring) == 0 {
		return nil
	}

	ip := clientIP(r)
	if ip == "" {
		l := uint64(len(ch.servers))
		if l == 0 {
			return nil
		}
		pos := atomic.AddUint64(&ch.fallback, 1)
		return ch.servers[pos%l]
	}

	keyHash := hashRingKey(ip)

	// "Walk clockwise from the key" = find the first ring point whose hash
	// is >= the key's hash. sort.Search does a binary search for exactly
	// this: the first index where the condition holds.
	idx := sort.Search(len(ch.ring), func(i int) bool {
		return ch.ring[i].hash >= keyHash
	})
	if idx == len(ch.ring) {
		// Walked past the highest point on the ring - wrap back around to
		// the first one, same idea as a clock going from 11 back to 12.
		idx = 0
	}

	return ch.ring[idx].server
}

// Done is a no-op: like IPHash, ConsistentHash deterministically maps clients to servers and doesn't
// track connections or response time.
func (ch *ConsistentHash) Done(server *config.ServerCfg, duration time.Duration, err error) {}

func hashRingKey(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}
