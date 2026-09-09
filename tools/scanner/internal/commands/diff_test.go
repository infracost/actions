package commands

import (
	"testing"

	"github.com/infracost/actions/tools/scanner/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// diff is always a pull request run, so it must refuse to scan rather than
// upload a run the dashboard would file as a branch build.
func TestDiff_RequiresBuildablePullRequestURL(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		repoURL  string
		prNumber int
		wantErr  string
	}{
		{
			name:     "missing repo url",
			provider: "github",
			prNumber: 42,
			wantErr:  "cannot determine the pull request URL: --repo-url and --pr-number are required",
		},
		{
			name:     "zero pr number",
			provider: "github",
			repoURL:  "https://github.com/infracost/actions",
			wantErr:  "cannot determine the pull request URL: --repo-url and --pr-number are required",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{VCSProvider: tt.provider}
			args := &diffArgs{repoURL: tt.repoURL, prNumber: tt.prNumber}

			err := diff(cfg, args, nil, &ScanResult{})

			require.Error(t, err)
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}
