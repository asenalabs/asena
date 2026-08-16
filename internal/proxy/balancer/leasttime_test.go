package balancer

import (
	"sync"
	"testing"
	"time"

	"github.com/asenalabs/asena/internal/config"
	"github.com/stretchr/testify/require"
)

func TestLeastTime_Empty(t *testing.T) {
	lt := NewLeastTime(nil)
	require.Nil(t, lt.Next(nil))
}

func TestLeastTime_ColdStartSpreadsAcrossUntestedServers(t *testing.T) {
	// None of these servers have reported back yet, so their scores all start at 0.
	// If Next() only looked at average time, it would send all three of these picks
	// to the same server. Because active connections count too, each pick should
	// move to a different server.
	servers := []*config.ServerCfg{
		{URL: strPtr("s1")}, {URL: strPtr("s2")}, {URL: strPtr("s3")},
	}
	lt := NewLeastTime(servers)

	want := []string{"s1", "s2", "s3"}
	for i, w := range want {
		got := lt.Next(nil)
		require.NotNil(t, got)
		require.Equal(t, w, *got.URL, "step %d", i)
	}
}

func TestLeastTime_PrefersFasterServerAfterSamples(t *testing.T) {
	fast := &config.ServerCfg{URL: strPtr("fast")}
	slow := &config.ServerCfg{URL: strPtr("slow")}
	lt := NewLeastTime([]*config.ServerCfg{fast, slow})

	lt.Done(fast, 10*time.Millisecond, nil)
	lt.Done(slow, 100*time.Millisecond, nil)

	got := lt.Next(nil)
	require.Equal(t, "fast", *got.URL)
}

func TestLeastTime_EWMASmoothsASpike(t *testing.T) {
	srv := &config.ServerCfg{URL: strPtr("s1")}
	lt := NewLeastTime([]*config.ServerCfg{srv})
	ts := lt.byServer[srv]

	lt.Done(srv, 10*time.Millisecond, nil) // first sample: avg = 10
	require.InDelta(t, 10.0, ts.avgMillis, 0.001)

	lt.Done(srv, 10*time.Millisecond, nil) // steady state: avg stays 10
	require.InDelta(t, 10.0, ts.avgMillis, 0.001)

	lt.Done(srv, 1000*time.Millisecond, nil) // a spike
	// alpha=0.2: 0.2*1000 + 0.8*10 = 208, NOT a jump straight to 1000.
	require.InDelta(t, 208.0, ts.avgMillis, 0.001)
}

func TestLeastTime_FailedRequestIsPenalized(t *testing.T) {
	srv := &config.ServerCfg{URL: strPtr("s1")}
	lt := NewLeastTime([]*config.ServerCfg{srv})
	ts := lt.byServer[srv]

	lt.Done(srv, 10*time.Millisecond, assertError)
	// errorPenaltyMultiplier=5: a 10ms failure should read as if it took
	// 50ms, so a server that fails fast doesn't look attractive.
	require.InDelta(t, 50.0, ts.avgMillis, 0.001)
}

func TestLeastTime_DoneOnUnknownServerIsNoop(t *testing.T) {
	lt := NewLeastTime([]*config.ServerCfg{{URL: strPtr("s1")}})
	stranger := &config.ServerCfg{URL: strPtr("not-in-the-list")}

	require.NotPanics(t, func() {
		lt.Done(stranger, time.Millisecond, nil)
	})
}

func TestLeastTime_ActiveNeverGoesNegative(t *testing.T) {
	srv := &config.ServerCfg{URL: strPtr("s1")}
	lt := NewLeastTime([]*config.ServerCfg{srv})

	got := lt.Next(nil)
	lt.Done(got, time.Millisecond, nil)
	lt.Done(got, time.Millisecond, nil) // extra Done(), shouldn't go negative

	require.Equal(t, int64(0), lt.byServer[srv].active)
}

func TestLeastTime_Concurrent(t *testing.T) {
	servers := []*config.ServerCfg{
		{URL: strPtr("s1")}, {URL: strPtr("s2")}, {URL: strPtr("s3")},
	}
	lt := NewLeastTime(servers)

	const workers = 50
	const iterations = 200

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				srv := lt.Next(nil)
				if srv != nil {
					lt.Done(srv, time.Microsecond, nil)
				}
			}
		}()
	}

	wg.Wait()

	for _, ts := range lt.servers {
		require.Equal(t, int64(0), ts.active, "server %s", *ts.server.URL)
	}
}
