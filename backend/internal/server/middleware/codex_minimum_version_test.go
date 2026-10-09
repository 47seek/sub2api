//go:build unit

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCodexMinimumVersionBeforeForwarding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settings := service.NewSettingService(fakeSettingRepo{values: map[string]string{
		service.SettingKeyMinCodexVersion: "0.162.0",
		service.SettingKeyMaxCodexVersion: "",
	}}, &config.Config{})
	for _, tc := range []struct {
		version string
		allowed bool
	}{
		{"0.161.0", false},
		{"0.162.0-rc.1", false},
		{"0.162.0", true},
		{"0.163.0", true},
		{"0.163.0-rc.1", true},
		{"0.162", false},
		{"invalid", false},
		{"", false},
	} {
		t.Run(tc.version, func(t *testing.T) {
			forwarded := false
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set(string(ContextKeyAPIKey), &service.APIKey{Group: &service.Group{Platform: service.PlatformOpenAI}})
			})
			router.Use(CodexMinimumVersion(settings))
			router.POST("/v1/responses", func(c *gin.Context) {
				forwarded = true
				c.Status(http.StatusOK)
			})
			request := httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
			request.Header.Set("User-Agent", "codex-tui/"+tc.version)
			request.Header.Set("Originator", "codex-tui")
			request.Header.Set("Version", "0.999.0")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			require.Equal(t, tc.allowed, forwarded, "rejected versions must not reach forwarding")
			if tc.allowed {
				require.Equal(t, http.StatusOK, response.Code)
			} else {
				require.Equal(t, http.StatusBadRequest, response.Code)
				require.Contains(t, response.Body.String(), "minimum required version 0.162.0")
			}
		})
	}
}
