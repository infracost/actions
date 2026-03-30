package config

import (
	"context"
	"fmt"
	"os"

	"github.com/infracost/actions/tools/scanner/internal/api/dashboard"
	"github.com/infracost/actions/tools/scanner/internal/api/events"
	"github.com/infracost/cli/pkg/auth"
	"github.com/infracost/cli/pkg/environment"
	"github.com/infracost/cli/pkg/logging"
	"github.com/infracost/cli/pkg/plugins"
	"github.com/infracost/vcs/pkg/vcs"
	"github.com/infracost/vcs/pkg/vcs/github"
)

type Config struct {
	// Environment is the environment to target for operations / authentication (development or production). Defaults to
	// production.
	Environment environment.Environment `flag:"environment;hidden" usage:"The environment to use for authentication" default:"prod"`

	// PricingEndpoint is the endpoint to use for prices. Defaults to https://pricing.api.infracost.io.
	PricingEndpoint string `env:"INFRACOST_CI_PRICING_ENDPOINT" flag:"pricing-endpoint;hidden" usage:"The pricing endpoint to use for prices" default:"https://pricing.api.infracost.io"`

	// OrgID is the organization ID to use for authentication. Defaults to the value of the INFRACOST_ORG_ID environment variable.
	OrgID string `env:"INFRACOST_CI_ORG_ID" flag:"org-id;hidden" usage:"The organization ID to use for authentication"`

	// BasePath is the path to the base branch checkout to diff against.
	BasePath string `flag:"base-path" usage:"Path to the base branch checkout"`

	// HeadPath is the path to the PR branch checkout.
	HeadPath string `flag:"head-path" usage:"Path to the head (PR) branch checkout"`

	// Project filters scanning to a single project when specified.
	Project string `flag:"project" usage:"Filter scanning to a single project"`

	// Branch is the branch name used for policy filtering. Applied to both base and head scans.
	Branch string `flag:"branch" usage:"Branch name used for policy filtering" default:"main"`

	// VCS provider configuration.
	VCSProvider string `flag:"vcs-provider" usage:"VCS provider to use for posting comments (github)" default:"github"`

	// Common VCS fields.
	CommitSHA string `env:"GITHUB_SHA" flag:"commit-sha" usage:"Head commit SHA"`
	RepoURL   string `flag:"repo-url" usage:"Repository URL for source links in comments"`
	PRNumber  int32  `flag:"pr-number" usage:"Pull request number to comment on"`

	// GitHub-specific fields.
	GitHubToken string `env:"GITHUB_TOKEN" flag:"github-token" usage:"GitHub API token for posting comments"`
	GitHubOwner string `flag:"github-owner" usage:"GitHub repository owner"`
	GitHubRepo  string `flag:"github-repo" usage:"GitHub repository name"`

	// VCSClientFn overrides the default VCS client construction. Used in tests.
	VCSClientFn func(ctx context.Context) (vcs.VCS, error)

	// Logging contains the configuration for logging.
	// keep logging above other structs, so it gets processed first and others can log in their process functions.
	Logging logging.Config

	// Dashboard contains the configuration for the dashboard API.
	Dashboard dashboard.Config

	// Events contains the configuration for the events API.
	Events events.Config

	// Auth contains the configuration for authenticating with Infracost.
	Auth auth.Config

	// Plugins contains the configuration for plugins.
	Plugins plugins.Config
}

// VCSClient constructs the appropriate VCS provider based on the configured
// VCSProvider flag. If VCSClientFn is set, it is used instead.
func (config *Config) VCSClient(ctx context.Context) (vcs.VCS, error) {
	if config.VCSClientFn != nil {
		return config.VCSClientFn(ctx)
	}

	switch config.VCSProvider {
	case "github":
		return github.New(ctx, config.GitHubOwner, config.GitHubRepo, config.GitHubToken, config.PRNumber, github.Options{})
	default:
		return nil, fmt.Errorf("unsupported VCS provider: %q", config.VCSProvider)
	}
}

func (config *Config) Process() {
	events.RegisterMetadata("cloudEnabled", os.Getenv("INFRACOST_ENABLE_CLOUD") == "true")
	events.RegisterMetadata("dashboardEnabled", os.Getenv("INFRACOST_ENABLE_DASHBOARD") == "true")
	events.RegisterMetadata("environment", config.Environment.String())
	events.RegisterMetadata("isDefaultPricingApiEndpoint", config.PricingEndpoint == "https://pricing.api.infracost.io")
}
