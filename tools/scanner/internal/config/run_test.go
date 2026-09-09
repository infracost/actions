package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRunInput_Metadata_CIPlatformAndVCSProvider(t *testing.T) {
	tests := []struct {
		name        string
		opts        RunInputOptions
		ciPlatform  string
		vcsProvider string
	}{
		{
			name:        "github actions",
			opts:        RunInputOptions{CIPlatform: "github_actions", VCSProvider: "github"},
			ciPlatform:  "github_actions",
			vcsProvider: "github",
		},
		{
			name:        "namespaces are independent",
			opts:        RunInputOptions{CIPlatform: "gitlab_ci", VCSProvider: "gitlab"},
			ciPlatform:  "gitlab_ci",
			vcsProvider: "gitlab",
		},
		{
			name:        "unrecognised platform",
			opts:        RunInputOptions{CIPlatform: "unknown", VCSProvider: "bitbucket"},
			ciPlatform:  "unknown",
			vcsProvider: "bitbucket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := BuildRunInput(tt.opts)

			assert.Equal(t, tt.ciPlatform, input.Metadata["ciPlatform"])
			assert.Equal(t, tt.vcsProvider, input.Metadata["vcsProvider"])
		})
	}
}

func TestBuildErrorRunInput_Metadata_CIPlatformAndVCSProvider(t *testing.T) {
	input := BuildErrorRunInput(
		RunInputOptions{CIPlatform: "gitlab_ci", VCSProvider: "gitlab"},
		0, "Failed to scan", "boom",
	)

	assert.Equal(t, "gitlab_ci", input.Metadata["ciPlatform"])
	assert.Equal(t, "gitlab", input.Metadata["vcsProvider"])
}

func TestBuildRunInput_PullRequestURL(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		repoURL  string
		prNumber int
		wantURL  string
		wantID   string
	}{
		{
			name:     "github",
			provider: "github",
			repoURL:  "https://github.com/infracost/actions",
			prNumber: 42,
			wantURL:  "https://github.com/infracost/actions/pull/42",
			wantID:   "42",
		},
		{
			name:     "gitlab",
			provider: "gitlab",
			repoURL:  "https://gitlab.com/infracost/actions",
			prNumber: 42,
			wantURL:  "https://gitlab.com/infracost/actions/-/merge_requests/42",
			wantID:   "42",
		},
		{
			name:     "azure repos",
			provider: "azure_repos",
			repoURL:  "https://dev.azure.com/infracost/actions/_git/actions",
			prNumber: 42,
			wantURL:  "https://dev.azure.com/infracost/actions/_git/actions/pullrequest/42",
			wantID:   "42",
		},
		{
			name:     "bitbucket",
			provider: "bitbucket",
			repoURL:  "https://bitbucket.org/infracost/actions",
			prNumber: 42,
			wantURL:  "https://bitbucket.org/infracost/actions/pull-requests/42",
			wantID:   "42",
		},
		{
			name:     "branch run has no pull request",
			provider: "gitlab",
			repoURL:  "https://gitlab.com/infracost/actions",
			prNumber: 0,
		},
		{
			name:     "unbuildable url yields no pull request",
			provider: "gitlab",
			repoURL:  "git@gitlab.com:infracost/actions.git",
			prNumber: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := BuildRunInput(RunInputOptions{
				Command:     "comment",
				VCSProvider: tt.provider,
				RepoURL:     tt.repoURL,
				PRNumber:    tt.prNumber,
			})

			assertMetadata(t, input.Metadata, "vcsPullRequestUrl", tt.wantURL)
			assertMetadata(t, input.Metadata, "vcsPullRequestId", tt.wantID)

			repoMeta, ok := input.Metadata["repoMetadata"].(map[string]any)
			require.True(t, ok, "repoMetadata should be an object")
			assertMetadata(t, repoMeta, "vcsPullRequestUrl", tt.wantURL)
		})
	}
}

// assertMetadata treats an absent key as empty: the PR fields are omitempty.
func assertMetadata(t *testing.T, metadata map[string]any, key, want string) {
	t.Helper()

	if want == "" {
		assert.Empty(t, metadata[key], key)
		return
	}
	assert.Equal(t, want, metadata[key], key)
}
