package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"golang.org/x/mod/semver"
)

func CodexMinimumVersion(settings *service.SettingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		key, ok := GetAPIKeyFromContext(c)
		if !ok || key.Group == nil || settings == nil || (key.Group.Platform != service.PlatformOpenAI && key.Group.Platform != service.PlatformComposite) {
			return
		}
		ua := c.GetHeader("User-Agent")
		if !openai.IsCodexOfficialClientRequestStrict(ua) && !openai.IsCodexOfficialClientOriginator(c.GetHeader("Originator")) {
			return
		}
		minimum := settings.GetCodexRestrictionPolicy(c.Request.Context()).MinCodexVersion
		if minimum == "" {
			return
		}
		version := openai.CodexUserAgentVersion(ua)
		core, _, _ := strings.Cut(version, "-")
		core, _, _ = strings.Cut(core, "+")
		valid := strings.Count(core, ".") == 2 && semver.IsValid("v"+version)
		comparison := service.CompareVersions(core, minimum)
		if !valid || comparison < 0 || (comparison == 0 && semver.Prerelease("v"+version) != "" && version != "0.162.0-alpha.17.2") {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": fmt.Sprintf("Codex version %q does not meet minimum required version %s", version, minimum)}})
		}
	}
}
