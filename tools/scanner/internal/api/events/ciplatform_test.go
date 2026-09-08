package events

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ciEnvVars is every exact variable name getCIPlatform looks up. Its
// prefix scan (ATLANTIS_, BITBUCKET_, ...) cannot be enumerated, so these
// tests assume no such variable is set.
var ciEnvVars = []string{
	"INFRACOST_CI_PLATFORM",
	"GITHUB_ACTIONS", "GITLAB_CI", "CIRCLECI", "JENKINS_HOME", "BUILDKITE",
	"TFC_RUN_ID", "ENV0_ENVIRONMENT_ID", "SCALR_RUN_ID", "CF_BUILD_ID",
	"TRAVIS", "CODEBUILD_CI", "TEAMCITY_VERSION", "BUDDYBUILD_BRANCH",
	"BITRISE_IO", "SEMAPHORE", "APPVEYOR", "WERCKER_GIT_BRANCH", "MAGNUM",
	"SHIPPABLE", "TDDIUM", "GREENHOUSE", "CIRRUS_CI", "TS_ENV",
	"SYSTEM_COLLECTIONURI", "BUILD_REPOSITORY_PROVIDER",
	"CI",
}

// clearCIEnv unsets every CI variable for the duration of the test.
// getCIPlatform ranges over a Go map, so a variable left set by the CI running
// these tests would otherwise win at random. t.Setenv cannot unset, but it does
// register the restore, so pair it with os.Unsetenv.
func clearCIEnv(t *testing.T) {
	t.Helper()
	for _, k := range ciEnvVars {
		if v, ok := os.LookupEnv(k); ok {
			t.Setenv(k, v)
			require.NoError(t, os.Unsetenv(k))
		}
	}
}

func TestCIPlatform(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		expected string
	}{
		{name: "nothing recognised", expected: ""},
		{name: "explicit override", env: map[string]string{"INFRACOST_CI_PLATFORM": "custom"}, expected: "custom"},
		{name: "github actions", env: map[string]string{"GITHUB_ACTIONS": "true"}, expected: "github_actions"},
		{name: "gitlab ci", env: map[string]string{"GITLAB_CI": "true"}, expected: "gitlab_ci"},
		{
			name:     "azure devops without repository provider",
			env:      map[string]string{"SYSTEM_COLLECTIONURI": "https://dev.azure.com/acme/"},
			expected: "azure_devops_",
		},
		{
			name:     "azure devops with repository provider",
			env:      map[string]string{"SYSTEM_COLLECTIONURI": "https://dev.azure.com/acme/", "BUILD_REPOSITORY_PROVIDER": "GitHub"},
			expected: "azure_devops_GitHub",
		},
		{name: "CI passes through a boolean", env: map[string]string{"CI": "true"}, expected: "true"},
		{name: "CI passes through a platform name", env: map[string]string{"CI": "woodpecker"}, expected: "woodpecker"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearCIEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			assert.Equal(t, tt.expected, CIPlatform())
		})
	}
}
