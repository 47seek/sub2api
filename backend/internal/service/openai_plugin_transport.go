package service

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/klauspost/compress/zstd"
)

func (s *OpenAIGatewayService) SetPluginManager(manager *PluginManager) {
	s.pluginManager = manager
}

// doOpenAIUpstream 只在 OpenAI OAuth 能力绑定已启用时把真实请求交给插件。
// 插件返回标准 http.Response，响应解析、错误映射、SSE 和计费仍由现有核心链处理。
func (s *OpenAIGatewayService) doOpenAIUpstream(request *http.Request, proxyURL string, account *Account) (*http.Response, error) {
	if account.IsOpenAIOAuthLike() && account.GetCodexFingerprintMode() == codexFingerprintDevice && !s.isAgentIdentityAccount(request.Context(), account) {
		return doOpenAIDeviceUpstream(s.httpUpstream, request, proxyURL, account)
	}
	if s.pluginManager != nil {
		response, handled, err := s.pluginManager.RoundTripOpenAIOAuth(request.Context(), request, proxyURL, account)
		if handled {
			return response, err
		}
	}
	return s.httpUpstream.Do(request, proxyURL, account.ID, account.Concurrency)
}

func doOpenAIDeviceUpstream(upstream HTTPUpstream, request *http.Request, proxyURL string, account *Account) (*http.Response, error) {
	if account.ProxyID != nil && proxyURL == "" {
		return nil, fmt.Errorf("OpenAI account %d has a bound proxy that was not resolved", account.ID)
	}
	if request.Method == http.MethodPost && strings.HasSuffix(request.URL.Path, "/responses") && request.Body != nil && request.Header.Get("Content-Encoding") == "" {
		body, err := io.ReadAll(request.Body)
		_ = request.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read Codex request for compression: %w", err)
		}
		encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(3)), zstd.WithEncoderConcurrency(1))
		if err != nil {
			return nil, err
		}
		encoded := encoder.EncodeAll(body, nil)
		_ = encoder.Close()
		request.Body = io.NopCloser(bytes.NewReader(encoded))
		request.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(encoded)), nil }
		request.ContentLength = int64(len(encoded))
		request.Header.Set("Content-Encoding", "zstd")
	}
	request = request.WithContext(WithHTTPUpstreamProfile(request.Context(), HTTPUpstreamProfileOpenAI))
	return upstream.DoWithTLS(request, proxyURL, account.ID, account.Concurrency, tlsfingerprint.Codex0162)
}

// doOpenAIAccountTestUpstream 让 OpenAI OAuth 账号测试与真实转发使用同一插件路径。
// API Key 和未命中插件的账号保持各自原有的 HTTPUpstream 行为。
func (s *AccountTestService) doOpenAIAccountTestUpstream(
	request *http.Request,
	proxyURL string,
	account *Account,
	credentialAccount *Account,
	useTLSFallback bool,
) (*http.Response, error) {
	if account.IsOpenAIOAuthLike() && account.GetCodexFingerprintMode() == codexFingerprintDevice && !credentialAccount.IsOpenAIAgentIdentity() {
		return doOpenAIDeviceUpstream(s.httpUpstream, request, proxyURL, account)
	}
	if s.pluginManager != nil {
		response, handled, err := s.pluginManager.RoundTripOpenAIOAuth(request.Context(), request, proxyURL, account)
		if handled {
			return response, err
		}
	}
	if useTLSFallback {
		return s.httpUpstream.DoWithTLS(
			request,
			proxyURL,
			account.ID,
			account.Concurrency,
			s.tlsFPProfileService.ResolveTLSProfile(account),
		)
	}
	return s.httpUpstream.Do(request, proxyURL, account.ID, account.Concurrency)
}
