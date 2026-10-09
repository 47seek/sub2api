package repository

import (
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/imroc/req/v3"
)

// req still owns form/JSON encoding and response decoding. The shared upstream
// owns sockets, redirects and the local account's allowed infrastructure cookies.
func NewOpenAIPrivacyClientFactory(upstream service.HTTPUpstream) service.PrivacyClientFactory {
	return func(proxyURL string) (*req.Client, error) {
		client := req.C().SetTimeout(120 * time.Second).SetCookieJar(nil)
		ua, _ := service.CodexCanonicalAuthIdentity()
		client.SetCommonHeader("User-Agent", ua)
		client.GetClient().CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		client.GetClient().Transport = req.HttpRoundTripFunc(func(request *http.Request) (*http.Response, error) {
			ctx := service.WithHTTPUpstreamProfile(request.Context(), service.HTTPUpstreamProfileOpenAI)
			return upstream.DoWithTLS(request.WithContext(ctx), proxyURL, service.OpenAIHTTPAccountID(ctx), 0, tlsfingerprint.Codex0162)
		})
		return client, nil
	}
}
