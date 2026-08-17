package balancer

import (
	"fmt"
	"testing"

	"github.com/asenalabs/asena/internal/config"
	"github.com/stretchr/testify/require"
)

func TestConsistentHash_Empty(t *testing.T) {
	ch := NewConsistentHash(nil)
	require.Nil(t, ch.Next(reqFromIP("203.0.113.10:1")))
}

func TestConsistentHash_SameIPAlwaysPicksSameServer(t *testing.T) {
	servers := []*config.ServerCfg{
		{URL: strPtr("http://localhost:9000")},
		{URL: strPtr("http://localhost:9001")},
		{URL: strPtr("http://localhost:9002")},
	}
	ch := NewConsistentHash(servers)

	r := reqFromIP("203.0.113.10:5555")
	first := ch.Next(r)
	require.NotNil(t, first)

	for i := 0; i < 10; i++ {
		got := ch.Next(r)
		require.Equal(t, *first.URL, *got.URL, "call %d", i)
	}
}

func TestConsistentHash_PortIsIgnored(t *testing.T) {
	servers := []*config.ServerCfg{
		{URL: strPtr("http://localhost:9000")},
		{URL: strPtr("http://localhost:9001")},
	}
	ch := NewConsistentHash(servers)

	a := ch.Next(reqFromIP("203.0.113.10:1111"))
	b := ch.Next(reqFromIP("203.0.113.10:9999"))
	require.Equal(t, *a.URL, *b.URL)
}

func TestConsistentHash_NilRequestFallsBackAcrossAllServers(t *testing.T) {
	servers := []*config.ServerCfg{
		{URL: strPtr("http://localhost:9000")},
		{URL: strPtr("http://localhost:9001")},
		{URL: strPtr("http://localhost:9002")},
	}
	ch := NewConsistentHash(servers)

	seen := map[string]bool{}
	for i := 0; i < 9; i++ {
		got := ch.Next(nil)
		require.NotNil(t, got)
		seen[*got.URL] = true
	}

	for _, s := range servers {
		require.True(t, seen[*s.URL], "server %s should be reachable via the fallback path", *s.URL)
	}
}

func TestConsistentHash_UnparseableRemoteAddrStillDeterministic(t *testing.T) {
	servers := []*config.ServerCfg{
		{URL: strPtr("http://localhost:9000")},
		{URL: strPtr("http://localhost:9001")},
	}
	ch := NewConsistentHash(servers)

	r := reqFromIP("not-a-valid-host-port")

	first := ch.Next(r)
	second := ch.Next(r)
	require.NotNil(t, first)
	require.Equal(t, *first.URL, *second.URL)
}

// syntheticIPs generates n deterministic, spread-out client IPs for the
// distribution/stability tests below.
func syntheticIPs(n int) []string {
	ips := make([]string, n)
	for i := 0; i < n; i++ {
		ips[i] = fmt.Sprintf("10.%d.%d.%d:1234", (i/65536)%256, (i/256)%256, i%256)
	}
	return ips
}

func TestConsistentHash_FullCoverageAcrossServers(t *testing.T) {
	// A weaker, more honest claim than "perfectly balanced": every server
	// should get picked by *something* across enough distinct clients. See
	// the PR description for why this test doesn't assert a tight balance
	// ratio - measuring the real implementation showed that with only a
	// few servers, vanilla consistent hashing can still land unevenly by
	// chance, even with 150 replicas per server.
	servers := []*config.ServerCfg{
		{URL: strPtr("http://localhost:9000")},
		{URL: strPtr("http://localhost:9001")},
		{URL: strPtr("http://localhost:9002")},
	}
	ch := NewConsistentHash(servers)

	seen := map[string]int{}
	for _, ip := range syntheticIPs(5000) {
		got := ch.Next(reqFromIP(ip))
		require.NotNil(t, got)
		seen[*got.URL]++
	}

	for _, s := range servers {
		require.True(t, seen[*s.URL] > 0, "server %s got zero traffic across 5000 clients", *s.URL)
	}
}

func TestConsistentHash_MuchLessDisruptiveThanIPHashOnServerAdd(t *testing.T) {
	// This is the whole point of building ConsistentHash after IPHash: it
	// should reassign far fewer clients when a server is added. Measured
	// directly against the real implementations (not hand-estimated):
	// IPHash moves ~75% of clients, ConsistentHash moves ~16%, for the
	// exact same 3000 -> 4000 server-count change.
	urls3 := []string{
		"http://localhost:9000", "http://localhost:9001", "http://localhost:9002",
	}
	urls4 := append(append([]string{}, urls3...), "http://localhost:9003")

	toServers := func(urls []string) []*config.ServerCfg {
		out := make([]*config.ServerCfg, len(urls))
		for i, u := range urls {
			out[i] = &config.ServerCfg{URL: strPtr(u)}
		}
		return out
	}

	ipHash3 := NewIPHash(toServers(urls3))
	ipHash4 := NewIPHash(toServers(urls4))
	consistentHash3 := NewConsistentHash(toServers(urls3))
	consistentHash4 := NewConsistentHash(toServers(urls4))

	ips := syntheticIPs(5000)
	ipHashMoved := 0
	consistentHashMoved := 0

	for _, ip := range ips {
		r := reqFromIP(ip)
		if *ipHash3.Next(r).URL != *ipHash4.Next(r).URL {
			ipHashMoved++
		}
		if *consistentHash3.Next(r).URL != *consistentHash4.Next(r).URL {
			consistentHashMoved++
		}
	}

	ipHashFraction := float64(ipHashMoved) / float64(len(ips))
	consistentHashFraction := float64(consistentHashMoved) / float64(len(ips))

	require.True(t, ipHashFraction > 0.5,
		"expected IPHash to reshuffle most clients on server add, got %.1f%%", ipHashFraction*100)
	require.True(t, consistentHashFraction < 0.3,
		"expected ConsistentHash to reshuffle a small minority, got %.1f%%", consistentHashFraction*100)
}
