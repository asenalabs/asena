package balancer

import (
	"sync"
	"testing"
	"time"

	"github.com/asenalabs/asena/internal/config"
	"github.com/stretchr/testify/require"
)

func TestLeastConnections_Empty(t *testing.T) {
	lc := NewLeastConnections(nil)
	require.Nil(t, lc.Next(nil))
}

func TestLeastConnections_PicksLeastLoaded(t *testing.T) {
	servers := []*config.ServerCfg{
		{URL: strPtr("s1")}, {URL: strPtr("s2")}, {URL: strPtr("s3")},
	}
	lc := NewLeastConnections(servers)

	// All start at 0 active connections. Each Next() call bumps the picked
	// server's count, so with nothing calling Done() yet, four calls
	// should visit s1, s2, s3, then wrap back to s1 (now the only one
	// tied for least-loaded again... actually all are at 1, so it's the
	// first in the tie).
	want := []string{"s1", "s2", "s3", "s1"}
	for i, w := range want {
		got := lc.Next(nil)
		require.NotNil(t, got)
		require.Equal(t, w, *got.URL, "step %d", i)
	}
}

func TestLeastConnections_DoneFreesUpASlot(t *testing.T) {
	servers := []*config.ServerCfg{
		{URL: strPtr("s1")}, {URL: strPtr("s2")}, {URL: strPtr("s3")},
	}
	lc := NewLeastConnections(servers)

	first := lc.Next(nil) // s1, active=1
	_ = lc.Next(nil)      // s2, active=1
	_ = lc.Next(nil)      // s3, active=1
	require.Equal(t, "s1", *first.URL)

	// Free up s1. It should immediately become eligible again, ahead of
	// s2/s3 which are still "busy".
	lc.Done(first, 10*time.Millisecond, nil)

	got := lc.Next(nil)
	require.Equal(t, "s1", *got.URL)
}

func TestLeastConnections_DoneOnUnknownServerIsNoop(t *testing.T) {
	servers := []*config.ServerCfg{{URL: strPtr("s1")}}
	lc := NewLeastConnections(servers)

	stranger := &config.ServerCfg{URL: strPtr("not-in-the-list")}

	require.NotPanics(t, func() {
		lc.Done(stranger, time.Millisecond, nil)
	})
}

func TestLeastConnections_DoneNeverGoesNegative(t *testing.T) {
	servers := []*config.ServerCfg{{URL: strPtr("s1")}}
	lc := NewLeastConnections(servers)

	// Calling Done() more times than Next() shouldn't be possible in
	// normal operation, but the balancer shouldn't misbehave if it
	// somehow happens - active should floor at 0, not go negative.
	got := lc.Next(nil)
	lc.Done(got, time.Millisecond, nil)
	lc.Done(got, time.Millisecond, nil)

	require.Equal(t, int64(0), lc.byServer[servers[0]].active)
}

func TestLeastConnections_ErrDoesNotChangeBehavior(t *testing.T) {
	// A failed request still frees the slot - Least Connections tracks
	// "in flight", not "succeeded".
	servers := []*config.ServerCfg{{URL: strPtr("s1")}, {URL: strPtr("s2")}}
	lc := NewLeastConnections(servers)

	picked := lc.Next(nil) // s1
	lc.Done(picked, 0, assertError)

	got := lc.Next(nil)
	require.Equal(t, "s1", *got.URL, "s1 should be eligible again even though its last request errored")
}

var assertError = &testError{"backend unreachable"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }

func TestLeastConnections_Concurrent(t *testing.T) {
	servers := []*config.ServerCfg{
		{URL: strPtr("s1")}, {URL: strPtr("s2")}, {URL: strPtr("s3")},
	}
	lc := NewLeastConnections(servers)

	const workers = 50
	const iterations = 200

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				srv := lc.Next(nil)
				if srv != nil {
					lc.Done(srv, time.Microsecond, nil)
				}
			}
		}()
	}

	wg.Wait()

	// Every Next() was paired with a Done(), so every counter should be
	// back to exactly 0.
	for _, cs := range lc.servers {
		require.Equal(t, int64(0), cs.active, "server %s", *cs.server.URL)
	}
}
