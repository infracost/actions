package config

import "strings"

// VCS is the v0.1 INFRACOST_VCS_* contract. Names are fixed by it; the
// INFRACOST_CI_* prefix used elsewhere in Config does not apply.
// No flag tags: each command registers only the flags it actually has, bound
// to these fields with the hydrated value as the default. INFRACOST_VCS_PROVIDER
// is the exception and lives on Config, where it is also a persistent flag.
type VCS struct {
	RepositoryURL string `env:"INFRACOST_VCS_REPOSITORY_URL"`

	// PullRequestURL is the pull request identity. PullRequestID is an alias
	// kept because the dashboard sets it and GitHub Actions has the number.
	PullRequestURL    string `env:"INFRACOST_VCS_PULL_REQUEST_URL"`
	PullRequestID     int    `env:"INFRACOST_VCS_PULL_REQUEST_ID"`
	PullRequestTitle  string `env:"INFRACOST_VCS_PULL_REQUEST_TITLE"`
	PullRequestAuthor string `env:"INFRACOST_VCS_PULL_REQUEST_AUTHOR"`

	// PullRequestLabels is the raw comma list, split at the call site.
	// registerFlag and setFieldValue panic on a slice, and PreProcess runs
	// once for the whole binary, so a []string here would break every command.
	PullRequestLabels string `env:"INFRACOST_VCS_PULL_REQUEST_LABELS"`

	// PullRequestStatus is the one name added to the contract rather than
	// adopted from it: OPEN, MERGED or CLOSED.
	PullRequestStatus string `env:"INFRACOST_VCS_PULL_REQUEST_STATUS"`

	Branch     string `env:"INFRACOST_VCS_BRANCH"`
	BaseBranch string `env:"INFRACOST_VCS_BASE_BRANCH"`

	CommitSHA         string `env:"INFRACOST_VCS_COMMIT_SHA"`
	CommitMessage     string `env:"INFRACOST_VCS_COMMIT_MESSAGE"`
	CommitAuthorName  string `env:"INFRACOST_VCS_COMMIT_AUTHOR_NAME"`
	CommitAuthorEmail string `env:"INFRACOST_VCS_COMMIT_AUTHOR_EMAIL"`

	// CommitTimestamp is Unix epoch seconds in v0.1 recipes and RFC3339 from
	// our own git fallback. Both are in the field; normalise, never reject.
	CommitTimestamp string `env:"INFRACOST_VCS_COMMIT_TIMESTAMP"`

	PipelineRunID string `env:"INFRACOST_VCS_PIPELINE_RUN_ID"`
}

// PullRequestEnvNames are the contract names that only mean something on a
// pull request run. scan warns on each rather than failing, so one job-level
// env: block can feed both scan and diff.
var PullRequestEnvNames = []string{
	"INFRACOST_VCS_PULL_REQUEST_URL",
	"INFRACOST_VCS_PULL_REQUEST_ID",
	"INFRACOST_VCS_PULL_REQUEST_TITLE",
	"INFRACOST_VCS_PULL_REQUEST_AUTHOR",
	"INFRACOST_VCS_PULL_REQUEST_LABELS",
	"INFRACOST_VCS_PULL_REQUEST_STATUS",
}

// Labels splits the raw comma list. Blank entries are dropped so a trailing
// comma from a CI template does not become a label.
func (v VCS) Labels() []string {
	var labels []string
	for _, label := range strings.Split(v.PullRequestLabels, ",") {
		if trimmed := strings.TrimSpace(label); trimmed != "" {
			labels = append(labels, trimmed)
		}
	}
	return labels
}
