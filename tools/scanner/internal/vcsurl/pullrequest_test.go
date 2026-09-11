package vcsurl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullRequest(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		repoURL  string
		number   int
		want     string
	}{
		{
			name:     "github",
			provider: ProviderGitHub,
			repoURL:  "https://github.com/infracost/actions",
			number:   42,
			want:     "https://github.com/infracost/actions/pull/42",
		},
		{
			name:     "gitlab",
			provider: ProviderGitLab,
			repoURL:  "https://gitlab.com/infracost/actions",
			number:   42,
			want:     "https://gitlab.com/infracost/actions/-/merge_requests/42",
		},
		{
			name:     "azure repos",
			provider: ProviderAzureRepos,
			repoURL:  "https://dev.azure.com/infracost/actions/_git/actions",
			number:   42,
			want:     "https://dev.azure.com/infracost/actions/_git/actions/pullrequest/42",
		},
		{
			name:     "bitbucket",
			provider: ProviderBitbucket,
			repoURL:  "https://bitbucket.org/infracost/actions",
			number:   42,
			want:     "https://bitbucket.org/infracost/actions/pull-requests/42",
		},
		{
			name:     "trims .git suffix",
			provider: ProviderGitHub,
			repoURL:  "https://github.com/infracost/actions.git",
			number:   7,
			want:     "https://github.com/infracost/actions/pull/7",
		},
		{
			name:     "trims trailing slash",
			provider: ProviderGitLab,
			repoURL:  "https://gitlab.com/infracost/actions/",
			number:   7,
			want:     "https://gitlab.com/infracost/actions/-/merge_requests/7",
		},
		{
			name:     "trims trailing slash then .git",
			provider: ProviderGitHub,
			repoURL:  "https://github.com/infracost/actions.git/",
			number:   7,
			want:     "https://github.com/infracost/actions/pull/7",
		},
		{
			name:     "gitlab subgroup",
			provider: ProviderGitLab,
			repoURL:  "https://gitlab.com/infracost/team/platform/actions",
			number:   3,
			want:     "https://gitlab.com/infracost/team/platform/actions/-/merge_requests/3",
		},
		{
			name:     "gitlab self-managed under a relative url root",
			provider: ProviderGitLab,
			repoURL:  "https://git.example.com/gitlab/infracost/actions",
			number:   3,
			want:     "https://git.example.com/gitlab/infracost/actions/-/merge_requests/3",
		},
		{
			name:     "github enterprise server",
			provider: ProviderGitHub,
			repoURL:  "https://github.example.com/infracost/actions",
			number:   3,
			want:     "https://github.example.com/infracost/actions/pull/3",
		},
		{
			name:     "azure visualstudio.com collection",
			provider: ProviderAzureRepos,
			repoURL:  "https://fabrikam.visualstudio.com/DefaultCollection/_git/Fabrikam",
			number:   1,
			want:     "https://fabrikam.visualstudio.com/DefaultCollection/_git/Fabrikam/pullrequest/1",
		},
		{
			// sanitizeUrl strips userinfo server-side, so both scanner sites
			// agree either way. Stripping in one of them is how they drift.
			name:     "azure userinfo is passed through untouched",
			provider: ProviderAzureRepos,
			repoURL:  "https://infracost@dev.azure.com/infracost/actions/_git/actions",
			number:   1,
			want:     "https://infracost@dev.azure.com/infracost/actions/_git/actions/pullrequest/1",
		},
		{
			name:     "http scheme",
			provider: ProviderGitHub,
			repoURL:  "http://github.example.com/infracost/actions",
			number:   9,
			want:     "http://github.example.com/infracost/actions/pull/9",
		},
		{
			name:     "no pr number",
			provider: ProviderGitHub,
			repoURL:  "https://github.com/infracost/actions",
			number:   0,
			want:     "",
		},
		{
			name:     "negative pr number",
			provider: ProviderGitHub,
			repoURL:  "https://github.com/infracost/actions",
			number:   -1,
			want:     "",
		},
		{
			name:     "empty repo url",
			provider: ProviderGitHub,
			repoURL:  "",
			number:   42,
			want:     "",
		},
		{
			name:     "empty repo url wins over an unknown provider",
			provider: "not-a-provider",
			repoURL:  "",
			number:   42,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PullRequest(tt.provider, tt.repoURL, tt.number)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParsePullRequest_RoundTrips(t *testing.T) {
	for _, tt := range []struct {
		provider string
		repoURL  string
		number   int
	}{
		{ProviderGitHub, "https://github.com/infracost/actions", 42},
		{ProviderGitLab, "https://gitlab.com/infracost/team/actions", 7},
		{ProviderAzureRepos, "https://dev.azure.com/infracost/actions/_git/actions", 3},
		{ProviderBitbucket, "https://bitbucket.org/infracost/actions", 9},
	} {
		t.Run(tt.provider, func(t *testing.T) {
			prURL, err := PullRequest(tt.provider, tt.repoURL, tt.number)
			require.NoError(t, err)

			repoURL, number, err := ParsePullRequest(tt.provider, prURL)
			require.NoError(t, err)
			assert.Equal(t, tt.repoURL, repoURL)
			assert.Equal(t, tt.number, number)
		})
	}
}

func TestOwnerRepo(t *testing.T) {
	tests := []struct {
		name    string
		repoURL string
		owner   string
		repo    string
		wantErr string
	}{
		{name: "github", repoURL: "https://github.com/infracost/actions", owner: "infracost", repo: "actions"},
		{name: "trims clone suffix", repoURL: "https://github.com/infracost/actions.git/", owner: "infracost", repo: "actions"},
		{name: "host case is ignored", repoURL: "https://GitHub.com/infracost/actions", owner: "infracost", repo: "actions"},
		{name: "explicit port is ignored", repoURL: "https://github.com:443/infracost/actions", owner: "infracost", repo: "actions"},
		{name: "enterprise host is refused", repoURL: "https://github.example.com/infracost/actions", wantErr: "the repository URL host is not github.com"},
		{name: "credentials do not leak", repoURL: "https://user:secret@github.com/infracost/actions", wantErr: "repo URL must not contain credentials"},
		{name: "token in the userinfo does not leak", repoURL: "https://secret@github.com/infracost/actions", wantErr: "repo URL must not contain credentials"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := OwnerRepo(tt.repoURL)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.NotContains(t, err.Error(), "secret")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.owner, owner)
			assert.Equal(t, tt.repo, repo)
		})
	}
}

func TestPullRequest_Errors(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		repoURL     string
		wantMessage string
	}{
		{
			name:        "unknown provider",
			provider:    "not-a-provider",
			repoURL:     "https://github.com/infracost/actions",
			wantMessage: `cannot build a pull request URL for VCS provider "not-a-provider": must be one of github, gitlab, azure_repos, bitbucket`,
		},
		{
			name:        "empty provider",
			provider:    "",
			repoURL:     "https://github.com/infracost/actions",
			wantMessage: `cannot build a pull request URL for VCS provider "": must be one of github, gitlab, azure_repos, bitbucket`,
		},
		{
			name:        "scp-style ssh remote",
			provider:    ProviderGitHub,
			repoURL:     "git@github.com:infracost/actions.git",
			wantMessage: `repo URL "git@github.com:infracost/actions.git" must be an http(s) web URL of the repository, not a clone URL`,
		},
		{
			// The userinfo is checked first, so the URL is never echoed.
			name:        "ssh scheme remote",
			provider:    ProviderGitHub,
			repoURL:     "ssh://git@github.com/infracost/actions.git",
			wantMessage: "repo URL must not contain credentials: pass the repository's web URL, not a credentialed clone URL",
		},
		{
			name:        "no scheme",
			provider:    ProviderGitHub,
			repoURL:     "github.com/infracost/actions",
			wantMessage: `repo URL "github.com/infracost/actions" must be an http(s) web URL of the repository, not a clone URL`,
		},
		{
			name:        "no host",
			provider:    ProviderGitLab,
			repoURL:     "https:///infracost/actions",
			wantMessage: `repo URL "https:///infracost/actions" must be an http(s) web URL of the repository, not a clone URL`,
		},
		{
			name:        "empty host and path",
			provider:    ProviderGitHub,
			repoURL:     "https://",
			wantMessage: `repo URL "https://" must be an http(s) web URL of the repository, not a clone URL`,
		},
		{
			// The message must not echo the URL, or the token lands in the log.
			name:        "gitlab job token in the userinfo",
			provider:    ProviderGitLab,
			repoURL:     "https://gitlab-ci-token:secret-token@gitlab.com/infracost/actions.git",
			wantMessage: "repo URL must not contain credentials: pass the repository's web URL, not a credentialed clone URL",
		},
		{
			name:        "github token as the whole userinfo",
			provider:    ProviderGitHub,
			repoURL:     "https://secret-token@github.com/infracost/actions.git",
			wantMessage: "repo URL must not contain credentials: pass the repository's web URL, not a credentialed clone URL",
		},
		{
			// The org@ prefix Azure hands out is allowed, a password is not.
			name:        "azure pat in the userinfo",
			provider:    ProviderAzureRepos,
			repoURL:     "https://infracost:secret-token@dev.azure.com/infracost/actions/_git/actions",
			wantMessage: "repo URL must not contain credentials: pass the repository's web URL, not a credentialed clone URL",
		},
		{
			name:        "credentialed url with no host",
			provider:    ProviderGitHub,
			repoURL:     "https://user:secret-token@",
			wantMessage: "repo URL must not contain credentials: pass the repository's web URL, not a credentialed clone URL",
		},
		{
			name:        "fragment",
			provider:    ProviderGitHub,
			repoURL:     "https://github.com/infracost/actions#frag",
			wantMessage: `repo URL must not contain a query or fragment: pass the repository's web URL`,
		},
		{
			name:        "query",
			provider:    ProviderGitHub,
			repoURL:     "https://github.com/infracost/actions?x=1",
			wantMessage: `repo URL must not contain a query or fragment: pass the repository's web URL`,
		},
		{
			name:        "azure project url without a repository",
			provider:    ProviderAzureRepos,
			repoURL:     "https://dev.azure.com/infracost/actions",
			wantMessage: `repo URL "https://dev.azure.com/infracost/actions" must be an Azure Repos repository URL containing /_git/`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PullRequest(tt.provider, tt.repoURL, 42)
			require.Error(t, err)
			assert.Empty(t, got)
			if tt.wantMessage != "" {
				assert.EqualError(t, err, tt.wantMessage)
			}
		})
	}
}
