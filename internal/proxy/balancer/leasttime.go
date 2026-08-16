package balancer

import (
	"net/http"
	"sync"
	"time"

	"github.com/asenalabs/asena/internal/config"
)

// defaultLeastTimeAlpha controls how fast the moving average reacts to a new sample.
// Smaller = smoother/slower to change, larger = jumpier. 0.2 is a common starting
// point: a spike moves the average by 20% of the gap between old and new, not all the way.
const defaultLeastTimeAlpha = 0.2

// errorPenaltyMultiplier makes a failed request look "slow" instead of "fast". Without
// this, a server that's erroring out instantly (connection refused, timeout) would
// look like the best possible choice, since a fast failure is still a low duration.
const errorPenaltyMultiplier = 5.0

// timeServer tracks everything LeastTime needs about one server: how many requests are
// in flight to it right now, and how fast it's historically responded.
type timeServer struct {
	server    *config.ServerCfg
	active    int64
	avgMillis float64
	hasSample bool // whether Done() has ever reported back for this server
}

// score is what Next() compares between servers - lower wins. Combining active connections
// with average latency (instead of using latency alone) is what prevents every request
// from piling onto the currently-fastest server before its average has a chance to catch
// up. See the package-level comment on LeastTime for the full explanation.
func (ts *timeServer) score() float64 {
	return float64(ts.active) + ts.avgMillis
}

// LeastTime sends each request to whichever server currently has the lowest combined score
// of (requests in flight + average response time in milliseconds). A server with no reported
// samples yet starts at an average of 0, so it's tried before any "known" server is trusted
// purely on reputation - but active connections still count against it, so a burst of
// concurrent requests spreads across all untested servers instead of piling onto just the first one.
type LeastTime struct {
	mu       sync.Mutex
	servers  []*timeServer
	byServer map[*config.ServerCfg]*timeServer
	alpha    float64
}

func NewLeastTime(servers []*config.ServerCfg) *LeastTime {
	lt := &LeastTime{
		servers:  make([]*timeServer, 0, len(servers)),
		byServer: make(map[*config.ServerCfg]*timeServer, len(servers)),
		alpha:    defaultLeastTimeAlpha,
	}

	for _, s := range servers {
		ts := &timeServer{server: s}
		lt.servers = append(lt.servers, ts)
		lt.byServer[s] = ts
	}

	return lt
}

// Next ignores r, same reason as Least Connections: this algorithm judges
// servers by their own behavior, not by anything about the client.
func (lt *LeastTime) Next(r *http.Request) *config.ServerCfg {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	if len(lt.servers) == 0 {
		return nil
	}

	picked := lt.servers[0]
	pickedScore := picked.score()
	for _, ts := range lt.servers[1:] {
		s := ts.score()
		if s < pickedScore {
			picked = ts
			pickedScore = s
		}
	}

	picked.active++
	return picked.server
}

// Done releases the server's active-connection count and folds the response time into
// its moving average. Runs on both success and failure - a failing server still needs
// its slot freed, and its speed at failing still tells us something (see errorPenaltyMultiplier).
func (lt *LeastTime) Done(server *config.ServerCfg, duration time.Duration, err error) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	ts, ok := lt.byServer[server]
	if !ok {
		return
	}

	if ts.active > 0 {
		ts.active--
	}

	ms := float64(duration) / float64(time.Millisecond)
	if err != nil {
		ms *= errorPenaltyMultiplier
	}

	if !ts.hasSample {
		// First real data point for this server - use it directly rather than blending with
		// the zero-value average, which would make a slow server look artificially fast for
		// several requests while the average "catches up".
		ts.avgMillis = ms
		ts.hasSample = true
		return
	}

	ts.avgMillis = lt.alpha*ms + (1-lt.alpha)*ts.avgMillis
}
