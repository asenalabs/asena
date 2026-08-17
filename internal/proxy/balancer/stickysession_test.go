package balancer

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/asenalabs/asena/internal/config"
	"github.com/stretchr/testify/require"
)

func reqWithCookie(name, value string) *http.Request {
	r := httptest.NewRequest("GET", "http://example.com/", nil)
	if value != "" {
		r.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	return r
}

func TestStickySession_Empty(t *testing.T) {
	ss := NewStickySession(nil)
	require.Nil(t, ss.Next(nil))
}

func TestStickySession_NoCookieFallsBackToRoundRobin(t *testing.T) {
	servers := []*config.ServerCfg{
		{URL: strPtr("s1")}, {URL: strPtr("s2")}, {URL: strPtr("s3")},
	}
	ss := NewStickySession(servers)

	// No cookie on any of these requests, so this should behave exactly
	// like plain Round Robin. Round Robin increments its counter before
	// indexing, so the first pick is index 1 ("s2"), not index 0 - see
	// RoundRobin.Next() and its own tests for the same behavior.
	want := []string{"s2", "s3", "s1", "s2"}
	for i, w := range want {
		got := ss.Next(httptest.NewRequest("GET", "http://example.com/", nil))
		require.NotNil(t, got)
		require.Equal(t, w, *got.URL, "step %d", i)
	}
}

func TestStickySession_ExistingCookieIsHonored(t *testing.T) {
	// These hash values were computed by actually running the same
	// hashServerURL function against these URLs - see the PR description.
	servers := []*config.ServerCfg{
		{URL: strPtr("http://localhost:9000")},
		{URL: strPtr("http://localhost:9001")},
		{URL: strPtr("http://localhost:9002")},
	}
	ss := NewStickySession(servers)

	r := reqWithCookie(defaultStickyCookieName, "8e53ceca9206b6d0") // hash of :9001

	for i := 0; i < 5; i++ {
		got := ss.Next(r)
		require.NotNil(t, got)
		require.Equal(t, "http://localhost:9001", *got.URL, "call %d", i)
	}
}

func TestStickySession_UnknownCookieValueFallsBack(t *testing.T) {
	servers := []*config.ServerCfg{
		{URL: strPtr("s1")}, {URL: strPtr("s2")},
	}
	ss := NewStickySession(servers)

	// A value that doesn't match any server's hash - maybe the config
	// changed since this cookie was issued, maybe it's just garbage.
	r := reqWithCookie(defaultStickyCookieName, "not-a-real-hash")

	got := ss.Next(r)
	require.NotNil(t, got)
	require.Equal(t, "s2", *got.URL, "should fall back to round robin's real first pick (index 1, see RoundRobin.Next)")
}

func TestStickySession_SetStickyCookie_WritesExpectedCookie(t *testing.T) {
	servers := []*config.ServerCfg{{URL: strPtr("http://localhost:9000")}}
	ss := NewStickySession(servers)

	header := http.Header{}
	ss.SetStickyCookie(header, servers[0])

	resp := http.Response{Header: header}
	cookies := resp.Cookies()
	require.Equal(t, 1, len(cookies))
	require.Equal(t, defaultStickyCookieName, cookies[0].Name)
	require.Equal(t, "8e53cfca9206b883", cookies[0].Value) // hash of :9000
}

func TestStickySession_SetStickyCookie_NilServerIsNoop(t *testing.T) {
	ss := NewStickySession([]*config.ServerCfg{{URL: strPtr("s1")}})

	header := http.Header{}
	ss.SetStickyCookie(header, nil)

	require.Equal(t, 0, len(header.Values("Set-Cookie")))
}

func TestStickySession_RoundTrip_CookieFromResponseIsHonoredNextTime(t *testing.T) {
	// Simulates the real flow: first request has no cookie, gets a pick
	// and a Set-Cookie header; a follow-up request carrying that exact
	// cookie should land on the same server, no matter how many other
	// picks happen in between.
	servers := []*config.ServerCfg{
		{URL: strPtr("http://localhost:9000")},
		{URL: strPtr("http://localhost:9001")},
	}
	ss := NewStickySession(servers)

	first := ss.Next(httptest.NewRequest("GET", "http://example.com/", nil))
	require.NotNil(t, first)

	header := http.Header{}
	ss.SetStickyCookie(header, first)
	resp := http.Response{Header: header}
	cookies := resp.Cookies()
	require.Equal(t, 1, len(cookies))

	// Some unrelated traffic happens in between.
	_ = ss.Next(httptest.NewRequest("GET", "http://example.com/", nil))
	_ = ss.Next(httptest.NewRequest("GET", "http://example.com/", nil))

	followUp := reqWithCookie(cookies[0].Name, cookies[0].Value)
	got := ss.Next(followUp)
	require.Equal(t, *first.URL, *got.URL)
}
