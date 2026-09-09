package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/infracost/actions/tools/scanner/internal/api/events"
	"github.com/infracost/actions/tools/scanner/internal/config"
	"github.com/infracost/actions/tools/scanner/internal/vcsurl"
)

// ciPlatform passes an explicit INFRACOST_CI_PLATFORM through verbatim;
// normalising it would discard a boolean-ish name chosen on purpose.
func ciPlatform() string {
	if platform, ok := os.LookupEnv("INFRACOST_CI_PLATFORM"); ok && platform != "" {
		return platform
	}
	return normalizeCIPlatform(events.CIPlatform())
}

// normalizeCIPlatform reports a platform we cannot name as unknown, so the
// dashboard can tell "somewhere unrecognised" from "field absent".
func normalizeCIPlatform(raw string) string {
	if raw == "" || raw == "azure_devops_" {
		return "unknown"
	}

	// The CI fallback returns that variable's raw value, which is a real name on
	// some systems (CI=woodpecker) and boolean-ish on most.
	if _, err := strconv.ParseBool(raw); err == nil {
		return "unknown"
	}
	switch strings.ToLower(raw) {
	case "yes", "no", "on", "off":
		return "unknown"
	}

	return raw
}

// resolveVCSProvider falls back to github when GITHUB_ACTIONS is set, so that
// pinned action versions running old YAML still report a provider.
func resolveVCSProvider(cfg *config.Config) (string, error) {
	if cfg.VCSProvider != "" {
		provider := strings.ToLower(strings.TrimSpace(cfg.VCSProvider))
		if !vcsurl.Valid(provider) {
			return "", fmt.Errorf("unrecognised VCS provider %q: set INFRACOST_VCS_PROVIDER to %s", cfg.VCSProvider, vcsurl.ProviderList())
		}
		return provider, nil
	}
	if os.Getenv("GITHUB_ACTIONS") != "" {
		return vcsurl.ProviderGitHub, nil
	}
	return "", fmt.Errorf("cannot determine the VCS provider: set INFRACOST_VCS_PROVIDER to %s", vcsurl.ProviderList())
}
