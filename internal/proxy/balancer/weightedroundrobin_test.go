package balancer

import (
	"sync"
	"testing"

	"github.com/asenalabs/asena/internal/config"
	"github.com/stretchr/testify/require"
)

func uintPtr(u uint) *uint {
	return &u
}

func TestWeightedRoundRobin_Empty(t *testing.T) {
	wrr := NewWeightedRoundRobin(nil)
	require.Nil(t, wrr.Next(nil))
}

func TestWeightedRoundRobin_DefaultWeightIsOne(t *testing.T) {
	// No Weight set at all, behaves exactly like plain Round Robin.
	servers := []*config.ServerCfg{
		{URL: strPtr("s1")}, {URL: strPtr("s2")},
	}
	wrr := NewWeightedRoundRobin(servers)

	want := []string{"s1", "s2", "s1", "s2"}
	for i, w := range want {
		got := wrr.Next(nil)
		require.NotNil(t, got)
		require.Equal(t, w, *got.URL, "step %d", i)
	}
}

func TestWeightedRoundRobin_ZeroWeightTreatedAsOne(t *testing.T) {
	servers := []*config.ServerCfg{
		{URL: strPtr("s1"), Weight: uintPtr(0)},
		{URL: strPtr("s2"), Weight: uintPtr(1)},
	}
	wrr := NewWeightedRoundRobin(servers)

	require.Equal(t, 1, wrr.servers[0].weight, "zero weight should fall back to 1")
}

func TestWeightedRoundRobin_SmoothSequence(t *testing.T) {
	// Weights 5:1:1. Traced by hand: over one full cycle (7 picks, since 5+1+1=7)
	// each server's currentWeight returns to 0, and the picks come out spread evenly
	// rather than "5 in a row".
	servers := []*config.ServerCfg{
		{URL: strPtr("a"), Weight: uintPtr(5)},
		{URL: strPtr("b"), Weight: uintPtr(1)},
		{URL: strPtr("c"), Weight: uintPtr(1)},
	}
	wrr := NewWeightedRoundRobin(servers)

	want := []string{"a", "a", "b", "a", "c", "a", "a"}
	for i, w := range want {
		got := wrr.Next(nil)
		require.NotNil(t, got)
		require.Equal(t, w, *got.URL, "step %d", i)
	}

	// The cycle repeats, one more full lap should give the same sequence.
	for i, w := range want {
		got := wrr.Next(nil)
		require.NotNil(t, got)
		require.Equal(t, w, *got.URL, "second lap, step %d", i)
	}
}

func TestWeightedRoundRobin_Concurrent(t *testing.T) {
	servers := []*config.ServerCfg{
		{URL: strPtr("s1"), Weight: uintPtr(5)},
		{URL: strPtr("s2"), Weight: uintPtr(1)},
		{URL: strPtr("s3"), Weight: uintPtr(1)},
	}
	wrr := NewWeightedRoundRobin(servers)

	const workers = 50
	const iterations = 200

	results := make(chan string, workers*iterations)

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if srv := wrr.Next(nil); srv != nil {
					results <- *srv.URL
				}
			}
		}()
	}

	wg.Wait()
	close(results)

	seen := map[string]bool{}
	for url := range results {
		seen[url] = true
	}
	for _, srv := range servers {
		require.True(t, seen[*srv.URL], "server %s should be selected", *srv.URL)
	}
}
