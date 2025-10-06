package balancer

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/asenalabs/asena/internal/config"
	"github.com/stretchr/testify/require"
)

func strPtr(s string) *string {
	return &s
}

func TestRoundRobin_Empty(t *testing.T) {
	rr := NewRoundRobin(nil)
	require.Nil(t, rr.Next())
}

func TestRoundRobin_Sequence(t *testing.T) {
	servers := []*config.ServerCfg{
		{URL: strPtr("s1")}, {URL: strPtr("s2")}, {URL: strPtr("s3")},
	}
	rr := NewRoundRobin(servers)

	want := []string{"s2", "s3", "s1", "s2", "s3", "s1"}
	for i, w := range want {
		got := rr.Next()
		require.NotNil(t, got)
		require.Equal(t, w, *got.URL, "step %d", i)
	}
}

func TestRoundRobin_CounterWrap(t *testing.T) {
	servers := []*config.ServerCfg{{URL: strPtr("only")}}
	rr := NewRoundRobin(servers)

	atomic.AddUint64(&rr.counter, ^uint64(0)-1)
	require.Equal(t, "only", *rr.Next().URL)
}

func TestRoundRobin_Concurrent(t *testing.T) {
	servers := []*config.ServerCfg{
		{URL: strPtr("s1")}, {URL: strPtr("s2")}, {URL: strPtr("s3")},
	}

	rr := NewRoundRobin(servers)

	const workers = 50
	const iterations = 200

	results := make(chan string, workers*iterations)

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if srv := rr.Next(); srv != nil {
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
