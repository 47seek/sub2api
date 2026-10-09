package repository

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"golang.org/x/net/publicsuffix"
)

type openAIInfrastructureCookieJar struct {
	http.CookieJar
}

// Called with the upstream pool mutex held. Pre-import OAuth calls have no row
// and must not share a synthetic account-zero cookie jar.
func (s *httpUpstreamService) openAIJarLocked(accountID int64) (http.CookieJar, error) {
	if accountID <= 0 {
		return nil, nil
	}
	if jar := s.openAIJars[accountID]; jar != nil {
		return jar, nil
	}
	jar, err := newOpenAIInfrastructureCookieJar()
	if err != nil {
		return nil, err
	}
	if s.openAIJars == nil {
		s.openAIJars = make(map[int64]http.CookieJar)
	}
	s.openAIJars[accountID] = jar
	return jar, nil
}

func newOpenAIInfrastructureCookieJar() (http.CookieJar, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}
	return &openAIInfrastructureCookieJar{CookieJar: jar}, nil
}

func openAICookieURL(u *url.URL) bool {
	host := u.Hostname()
	return u.Scheme == "https" && (host == "chatgpt.com" || host == "chat.openai.com" || host == "chatgpt-staging.com" ||
		strings.HasSuffix(host, ".chatgpt.com") || strings.HasSuffix(host, ".chatgpt-staging.com"))
}

func (j *openAIInfrastructureCookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	if !openAICookieURL(u) {
		return
	}
	allowed := make([]*http.Cookie, 0, len(cookies))
	for _, cookie := range cookies {
		switch cookie.Name {
		case "__cf_bm", "__cflb", "__cfruid", "__cfseq", "__cfwaitingroom", "__oailb", "_cfuvid", "cf_clearance", "cf_ob_info", "cf_use_ob":
			allowed = append(allowed, cookie)
		default:
			if strings.HasPrefix(cookie.Name, "cf_chl_") {
				allowed = append(allowed, cookie)
			}
		}
	}
	j.CookieJar.SetCookies(u, allowed)
}

func (j *openAIInfrastructureCookieJar) Cookies(u *url.URL) []*http.Cookie {
	if !openAICookieURL(u) {
		return nil
	}
	return j.CookieJar.Cookies(u)
}
