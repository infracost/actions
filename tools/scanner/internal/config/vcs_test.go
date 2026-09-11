package config

import (
	"strconv"
	"testing"

	"github.com/infracost/cli/pkg/config/process"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVCS_EnvironmentContract(t *testing.T) {
	tests := []struct {
		name  string
		env   string
		value string
		got   func(*Config) string
	}{
		{"provider", "INFRACOST_VCS_PROVIDER", "gitlab", func(c *Config) string { return c.VCSProvider }},
		{"repository URL", "INFRACOST_VCS_REPOSITORY_URL", "https://gitlab.com/infracost/actions", func(c *Config) string { return c.VCS.RepositoryURL }},
		{"pull request URL", "INFRACOST_VCS_PULL_REQUEST_URL", "https://gitlab.com/infracost/actions/-/merge_requests/42", func(c *Config) string { return c.VCS.PullRequestURL }},
		{"pull request ID", "INFRACOST_VCS_PULL_REQUEST_ID", "42", func(c *Config) string { return strconv.Itoa(c.VCS.PullRequestID) }},
		{"pull request title", "INFRACOST_VCS_PULL_REQUEST_TITLE", "Update scanner", func(c *Config) string { return c.VCS.PullRequestTitle }},
		{"pull request author", "INFRACOST_VCS_PULL_REQUEST_AUTHOR", "owen", func(c *Config) string { return c.VCS.PullRequestAuthor }},
		{"pull request labels", "INFRACOST_VCS_PULL_REQUEST_LABELS", "bug,ci", func(c *Config) string { return c.VCS.PullRequestLabels }},
		{"pull request status", "INFRACOST_VCS_PULL_REQUEST_STATUS", "OPEN", func(c *Config) string { return c.VCS.PullRequestStatus }},
		{"branch", "INFRACOST_VCS_BRANCH", "feature/vcs", func(c *Config) string { return c.VCS.Branch }},
		{"base branch", "INFRACOST_VCS_BASE_BRANCH", "main", func(c *Config) string { return c.VCS.BaseBranch }},
		{"commit SHA", "INFRACOST_VCS_COMMIT_SHA", "abc123", func(c *Config) string { return c.VCS.CommitSHA }},
		{"commit message", "INFRACOST_VCS_COMMIT_MESSAGE", "Fix scanner", func(c *Config) string { return c.VCS.CommitMessage }},
		{"commit author name", "INFRACOST_VCS_COMMIT_AUTHOR_NAME", "Owen", func(c *Config) string { return c.VCS.CommitAuthorName }},
		{"commit author email", "INFRACOST_VCS_COMMIT_AUTHOR_EMAIL", "owen@example.com", func(c *Config) string { return c.VCS.CommitAuthorEmail }},
		{"commit timestamp", "INFRACOST_VCS_COMMIT_TIMESTAMP", "1700000000", func(c *Config) string { return c.VCS.CommitTimestamp }},
		{"pipeline run ID", "INFRACOST_VCS_PIPELINE_RUN_ID", "1234", func(c *Config) string { return c.VCS.PipelineRunID }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.env, tt.value)
			var cfg Config
			diags := process.PreProcess(&cfg, pflag.NewFlagSet("", pflag.ContinueOnError))
			require.Zero(t, diags.Len(), "PreProcess reported %s", diags)
			assert.Equal(t, tt.value, tt.got(&cfg))
		})
	}
}

func TestVCS_MalformedPullRequestIDReportsDiagnostic(t *testing.T) {
	t.Setenv("INFRACOST_VCS_PULL_REQUEST_ID", "abc")

	var cfg Config
	diags := process.PreProcess(&cfg, pflag.NewFlagSet("", pflag.ContinueOnError))
	require.NotZero(t, diags.Critical().Len())
	assert.Contains(t, diags.String(), "INFRACOST_VCS_PULL_REQUEST_ID")
}

func TestVCS_Labels(t *testing.T) {
	assert.Equal(t, []string{"bug", "ci"}, (VCS{PullRequestLabels: " bug, ,ci, "}).Labels())
}
