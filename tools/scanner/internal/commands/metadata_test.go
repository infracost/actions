package commands

import (
	"testing"

	"github.com/infracost/actions/tools/scanner/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeCIPlatform(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected string
	}{
		{name: "nothing detected", raw: "", expected: "unknown"},
		{name: "azure devops without repository provider", raw: "azure_devops_", expected: "unknown"},
		{name: "CI=true", raw: "true", expected: "unknown"},
		{name: "CI=1", raw: "1", expected: "unknown"},
		{name: "CI=yes", raw: "yes", expected: "unknown"},
		{name: "CI=false", raw: "false", expected: "unknown"},
		{name: "CI=TRUE", raw: "TRUE", expected: "unknown"},
		{name: "CI=on", raw: "on", expected: "unknown"},
		{name: "github actions", raw: "github_actions", expected: "github_actions"},
		{name: "gitlab ci", raw: "gitlab_ci", expected: "gitlab_ci"},
		{name: "CI names a platform", raw: "woodpecker", expected: "woodpecker"},
		{name: "azure devops with repository provider", raw: "azure_devops_github", expected: "azure_devops_github"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeCIPlatform(tt.raw))
		})
	}
}

func TestCIPlatform_ExplicitOverride(t *testing.T) {
	t.Setenv("INFRACOST_CI_PLATFORM", "on")
	assert.Equal(t, "on", ciPlatform())
}

func TestResolveVCSProvider(t *testing.T) {
	tests := []struct {
		name          string
		configured    string
		githubActions string
		expected      string
		expectErr     bool
	}{
		{name: "configured value wins", configured: "gitlab", githubActions: "true", expected: "gitlab"},
		{name: "configured without github actions", configured: "bitbucket", expected: "bitbucket"},
		{name: "case and whitespace folded", configured: " Github ", expected: "github"},
		{name: "typo rejected", configured: "githbu", githubActions: "true", expectErr: true},
		{name: "vcs module package name rejected", configured: "azure", expectErr: true},
		{name: "ci platform name rejected", configured: "gitlab_ci", expectErr: true},
		{name: "unset falls back on github actions", githubActions: "true", expected: "github"},
		{name: "unset elsewhere errors", expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_ACTIONS", tt.githubActions)

			provider, err := resolveVCSProvider(&config.Config{VCSProvider: tt.configured})
			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "INFRACOST_VCS_PROVIDER")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, provider)
		})
	}
}
