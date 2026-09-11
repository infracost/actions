package commands

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/infracost/actions/tools/scanner/internal/config"
	"github.com/infracost/actions/tools/scanner/internal/vcsurl"
	"github.com/infracost/cli/pkg/logging"
)

// resolvePullRequest settles the pull request identity from the URL, the
// number, or both. The URL is the dashboard's key; the number derives from it.
func resolvePullRequest(provider, repoURL, prURL string, prNumber int) (string, int, error) {
	// Trimmed once so a trailing slash cannot fail the comparison below
	// against a URL PullRequest has already normalised.
	prURL = strings.TrimSuffix(prURL, "/")

	if prURL == "" && prNumber <= 0 {
		return "", 0, fmt.Errorf("cannot determine the pull request: set INFRACOST_VCS_PULL_REQUEST_ID or INFRACOST_VCS_PULL_REQUEST_URL")
	}

	if prURL == "" {
		built, err := vcsurl.PullRequest(provider, repoURL, prNumber)
		if err != nil {
			return "", 0, err
		}
		if built == "" {
			return "", 0, fmt.Errorf("cannot determine the pull request URL: set INFRACOST_VCS_REPOSITORY_URL")
		}
		return built, prNumber, nil
	}

	parsedRepoURL, parsedNumber, err := vcsurl.ParsePullRequest(provider, prURL)
	if err != nil {
		return "", 0, err
	}

	// Rebuilt from its own parts so a clone-style .git suffix normalises the
	// same way PullRequest normalises repoURL below.
	prURL, err = vcsurl.PullRequest(provider, parsedRepoURL, parsedNumber)
	if err != nil {
		return "", 0, err
	}

	// Rebuilt from repoURL and the number and compared, so a pair that
	// disagrees fails here rather than keying the run to the wrong PR.
	// Both callers require repoURL before this, so it is always set.
	number := prNumber
	if number <= 0 {
		number = parsedNumber
	}
	built, err := vcsurl.PullRequest(provider, repoURL, number)
	if err != nil {
		return "", 0, err
	}
	if built != prURL {
		return "", 0, fmt.Errorf("pull request URL %q does not match INFRACOST_VCS_REPOSITORY_URL and INFRACOST_VCS_PULL_REQUEST_ID, which give %q", prURL, built)
	}

	return prURL, parsedNumber, nil
}

// resolveOwnerRepo derives the GitHub owner and repository from the repo URL.
// The flags override the path, not the host.
func resolveOwnerRepo(provider, repoURL, owner, repo string) (string, string, error) {
	// Both checked before the flags: they name a github.com repository, so
	// accepting them elsewhere would send that host's token to api.github.com.
	if provider != vcsurl.ProviderGitHub {
		return "", "", fmt.Errorf("posting comments is only supported on github, not %q", provider)
	}
	if err := vcsurl.CheckGitHubHost(repoURL); err != nil {
		return "", "", err
	}

	if owner != "" && repo != "" {
		return owner, repo, nil
	}

	derivedOwner, derivedRepo, err := vcsurl.OwnerRepo(repoURL)
	if err != nil {
		return "", "", err
	}
	if owner == "" {
		owner = derivedOwner
	}
	if repo == "" {
		repo = derivedRepo
	}
	return owner, repo, nil
}

// normaliseTimestamp accepts both formats in the field: epoch seconds from the
// v0.1 recipes' %ct, and RFC3339 from our git fallback's %aI.
func normaliseTimestamp(name, value string) (string, error) {
	if value == "" {
		return "", nil
	}

	if isEpochSeconds(value) {
		epoch, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			return time.Unix(epoch, 0).UTC().Format(time.RFC3339), nil
		}
	}

	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return "", fmt.Errorf("invalid %s %q: expected Unix epoch seconds or an RFC3339 timestamp", name, value)
	}
	return t.Format(time.RFC3339), nil
}

// warnIgnoredPullRequestEnv logs the pull request variables scan ignores. It
// never rejects: one env: block must be able to feed both scan and diff.
func warnIgnoredPullRequestEnv() {
	for _, name := range config.PullRequestEnvNames {
		if value, ok := os.LookupEnv(name); ok && value != "" {
			logging.Warnf("ignoring %s: scan uploads a branch run, not a pull request run", name)
		}
	}
}

// VCSEnvNames lists the contract variables actually set, for the telemetry
// that Flags().Visit cannot see once a run is driven by the environment.
func VCSEnvNames() []string {
	var names []string
	for _, env := range os.Environ() {
		name, value, _ := strings.Cut(env, "=")
		if strings.HasPrefix(name, "INFRACOST_VCS_") && value != "" {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

// isEpochSeconds bounds the digit form to 9-11 digits, so a millisecond epoch
// or a %Y%m%d date falls through to RFC3339 instead of becoming a year 46000.
func isEpochSeconds(value string) bool {
	if len(value) < 9 || len(value) > 11 {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// firstNonEmpty returns the first set value, in env-then-flag-then-git order
// as each call site lists them.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
