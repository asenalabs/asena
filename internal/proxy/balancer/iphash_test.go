package balancer

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/asenalabs/asena/internal/config"
	"github.com/stretchr/testify/require"
)

func reqFromIP(remoteAddr string) *http.Request {
	r := httptest.NewRequest("GET", "http://example.com/", nil)
	r.RemoteAddr = remoteAddr
	return r
}

func TestIPHash_Empty(t *testing.T) {
	ih := NewIPHash(nil)
	require.Nil(t, ih.Next(reqFromIP("203.0.113.10:5555")))
}

func TestIPHash_SameIPAlwaysPicksSameServer(t *testing.T) {
	servers := []*config.ServerCfg{
		{URL: strPtr("s1")}, {URL: strPtr("s2")}, {URL: strPtr("s3")},
	}
	ih := NewIPHash(servers)

	r := reqFromIP("203.0.113.10:5555")
	first := ih.Next(r)
	require.NotNil(t, first)

	for i := 0; i < 10; i++ {
		got := ih.Next(r)
		require.Equal(t, *first.URL, *got.URL, "call %d", i)
	}
}

func TestIPHash_KnownIPsMapToVerifiedServers(t *testing.T) {
	// These bucket numbers were computed by actually running FNV-1a on
	// these IPs against a 3-server list, not guessed by hand - see the
	// PR description for how.
	servers := []*config.ServerCfg{
		{URL: strPtr("s1")}, {URL: strPtr("s2")}, {URL: strPtr("s3")},
	}
	ih := NewIPHash(servers)

	cases := []struct {
		ip   string
		want string
	}{
		{"203.0.113.10:1", "s1"},
		{"198.51.100.5:1", "s2"},
		{"203.0.113.11:1", "s3"},
	}

	for _, c := range cases {
		got := ih.Next(reqFromIP(c.ip))
		require.NotNil(t, got)
		require.Equal(t, c.want, *got.URL, "ip %s", c.ip)
	}
}

func TestIPHash_PortIsIgnored(t *testing.T) {
	// Same IP, different port -> same server. Only the IP should matter.
	servers := []*config.ServerCfg{
		{URL: strPtr("s1")}, {URL: strPtr("s2")}, {URL: strPtr("s3")},
	}
	ih := NewIPHash(servers)

	a := ih.Next(reqFromIP("203.0.113.10:1111"))
	b := ih.Next(reqFromIP("203.0.113.10:9999"))
	require.Equal(t, *a.URL, *b.URL)
}

func TestIPHash_NilRequestFallsBackAcrossAllServers(t *testing.T) {
	servers := []*config.ServerCfg{
		{URL: strPtr("s1")}, {URL: strPtr("s2")}, {URL: strPtr("s3")},
	}
	ih := NewIPHash(servers)

	seen := map[string]bool{}
	for i := 0; i < 9; i++ {
		got := ih.Next(nil)
		require.NotNil(t, got)
		seen[*got.URL] = true
	}

	for _, s := range servers {
		require.True(t, seen[*s.URL], "server %s should be reachable via the fallback path", *s.URL)
	}
}

func TestIPHash_UnparseableRemoteAddrStillDeterministic(t *testing.T) {
	servers := []*config.ServerCfg{
		{URL: strPtr("s1")}, {URL: strPtr("s2")},
	}
	ih := NewIPHash(servers)

	r := reqFromIP("not-a-valid-host-port") // no port to split off

	first := ih.Next(r)
	second := ih.Next(r)
	require.NotNil(t, first)
	require.Equal(t, *first.URL, *second.URL)
}
