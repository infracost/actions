package commands

import (
	"testing"

	"github.com/infracost/actions/tools/scanner/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// diff is always a pull request run, so it must refuse to scan rather than
// upload a run the dashboard would file as a branch build.
func TestResolveDiffContext_RequiresBuildablePullRequestURL(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		repoURL  string
		prURL    string
		prNumber int
		wantErr  string
	}{
		{
			name:     "missing repo url",
			provider: "github",
			prNumber: 42,
			wantErr:  "cannot determine the repository URL: set INFRACOST_VCS_REPOSITORY_URL",
		},
		{
			name:     "no pull request at all",
			provider: "github",
			repoURL:  "https://github.com/infracost/actions",
			wantErr:  "cannot determine the pull request: set INFRACOST_VCS_PULL_REQUEST_ID or INFRACOST_VCS_PULL_REQUEST_URL",
		},
		{
			name:     "ssh clone url",
			provider: "gitlab",
			repoURL:  "git@gitlab.com:infracost/actions.git",
			prNumber: 42,
			wantErr:  `repo URL "git@gitlab.com:infracost/actions.git" must be an http(s) web URL of the repository, not a clone URL`,
		},
		{
			name:     "azure project url without _git",
			provider: "azure_repos",
			repoURL:  "https://dev.azure.com/infracost/actions",
			prNumber: 42,
			wantErr:  `repo URL "https://dev.azure.com/infracost/actions" must be an Azure Repos repository URL containing /_git/`,
		},
		{
			name:     "url and number disagree",
			provider: "github",
			repoURL:  "https://github.com/infracost/actions",
			prURL:    "https://github.com/infracost/actions/pull/7",
			prNumber: 42,
			wantErr:  `pull request URL "https://github.com/infracost/actions/pull/7" does not match INFRACOST_VCS_REPOSITORY_URL and INFRACOST_VCS_PULL_REQUEST_ID, which give "https://github.com/infracost/actions/pull/42"`,
		},
		{
			name:     "url names another repository",
			provider: "github",
			repoURL:  "https://github.com/infracost/actions",
			prURL:    "https://github.com/attacker/actions/pull/42",
			wantErr:  `pull request URL "https://github.com/attacker/actions/pull/42" does not match INFRACOST_VCS_REPOSITORY_URL and INFRACOST_VCS_PULL_REQUEST_ID, which give "https://github.com/infracost/actions/pull/42"`,
		},
		{
			name:     "url is not a pull request url",
			provider: "github",
			repoURL:  "https://github.com/infracost/actions",
			prURL:    "https://github.com/infracost/actions/issues/42",
			wantErr:  `pull request URL "https://github.com/infracost/actions/issues/42" is not a github pull request URL: expected /pull/<number>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{VCSProvider: tt.provider}
			// prURL is a flag bound to INFRACOST_VCS_PULL_REQUEST_URL.
			args := &diffArgs{repoURL: tt.repoURL, prURL: tt.prURL, prNumber: tt.prNumber}

			_, err := resolveDiffContext(cfg, args)

			require.Error(t, err)
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}
