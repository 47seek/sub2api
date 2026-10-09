package repository

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/imroc/req/v3"
	"github.com/stretchr/testify/require"
)

func TestOpenAIDeviceCookieLifecycle(t *testing.T) {
	cfg := &config.Config{}
	cfg.Gateway.MaxUpstreamClients = 1
	s := NewHTTPUpstream(cfg).(*httpUpstreamService)
	u, err := url.Parse("https://chatgpt.com/backend-api/codex/models")
	require.NoError(t, err)
	client := func(row int64, concurrency int) *http.Client {
		entry, err := s.getClientEntryWithTLS("", row, concurrency, tlsfingerprint.Codex0162, service.HTTPUpstreamProfileOpenAI, false, true)
		require.NoError(t, err)
		return entry.client
	}
	requestCookie := func(client *http.Client) string {
		clone := *client
		cookie := ""
		clone.Transport = req.HttpRoundTripFunc(func(r *http.Request) (*http.Response, error) {
			cookie = r.Header.Get("Cookie")
			return &http.Response{StatusCode: http.StatusNoContent, Header: http.Header{}, Body: http.NoBody, Request: r}, nil
		})
		r, err := http.NewRequest(http.MethodGet, u.String(), nil)
		require.NoError(t, err)
		r.Header.Set("Authorization", "Bearer refreshed-token")
		resp, err := clone.Do(r)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		return cookie
	}
	a := client(1, 3)
	a.Jar.SetCookies(u, []*http.Cookie{
		{Name: "__oailb", Value: "row-A", Path: "/", Secure: true},
		{Name: "session", Value: "forbidden", Path: "/", Secure: true},
		{Name: "__cf_bm", Value: "expired", Path: "/", Expires: time.Now().Add(-time.Hour)},
	})
	require.Equal(t, "__oailb=row-A", requestCookie(client(1, 0)), "pool rebuild and token refresh retain the row jar")
	require.Empty(t, requestCookie(client(2, 3)), "another row must not inherit A's cookies")
	require.Equal(t, "__oailb=row-A", requestCookie(client(1, 3)), "LRU eviction must not discard device state")
	s.mu.Lock()
	s.evictIdleLocked(time.Now().Add(24 * time.Hour))
	s.mu.Unlock()
	require.Equal(t, "__oailb=row-A", requestCookie(client(1, 3)), "idle client expiry must not expire live cookies")
	a.Jar.SetCookies(u, []*http.Cookie{{Name: "__oailb", MaxAge: -1, Path: "/"}})
	require.Empty(t, requestCookie(client(1, 3)), "server cookie deletion must be honored")
	require.Nil(t, client(0, 0).Jar, "pre-import calls have no shared row-zero jar")
}

func TestOpenAIDeviceTLSRejectsProxyFallbackAndRedirectDowngrade(t *testing.T) {
	s := NewHTTPUpstream(nil).(*httpUpstreamService)
	_, err := s.getClientEntryWithTLS("https://proxy.example:443", 1, 1, tlsfingerprint.Codex0162, service.HTTPUpstreamProfileOpenAI, false, true)
	require.ErrorContains(t, err, "HTTPS proxies")
	entry, err := s.getClientEntryWithTLS("http://proxy.example:8080", 1, 1, tlsfingerprint.Codex0162, service.HTTPUpstreamProfileOpenAI, false, true)
	require.NoError(t, err)
	for _, target := range []string{"http://chatgpt.com/plaintext", "https://127.0.0.1/private"} {
		calls := 0
		client := *entry.client
		client.Transport = req.HttpRoundTripFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": {target}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
		})
		r, err := http.NewRequestWithContext(service.WithHTTPUpstreamPublicHostsOnly(t.Context()), http.MethodGet, "https://chatgpt.com/start", nil)
		require.NoError(t, err)
		resp, err := client.Do(r)
		require.Error(t, err)
		if resp != nil {
			_ = resp.Body.Close()
		}
		require.Equal(t, 1, calls, "the rejected redirect must not reach any transport")
	}
}
