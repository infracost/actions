package config

import (
	"context"
	"fmt"
	"os"

	"github.com/infracost/actions/tools/scanner/internal/api"
	"github.com/infracost/actions/tools/scanner/internal/api/dashboard"
	"github.com/infracost/actions/tools/scanner/internal/api/events"
	"github.com/infracost/cli/pkg/auth"
	"github.com/infracost/cli/pkg/environment"
	"github.com/infracost/cli/pkg/logging"
	"github.com/infracost/cli/pkg/plugins"
)

// Config holds the shared configuration used by all subcommands.
type Config struct {
	// Environment is the environment to target for operations / authentication (development or production). Defaults to
	// production.
	Environment environment.Environment `flag:"environment;hidden" usage:"The environment to use for authentication" default:"prod"`

	// PricingEndpoint is the endpoint to use for prices. Defaults to https://pricing.api.infracost.io.
	PricingEndpoint string `env:"INFRACOST_CI_PRICING_ENDPOINT" flag:"pricing-endpoint;hidden" usage:"The pricing endpoint to use for prices" default:"https://pricing.api.infracost.io"`

	// OrgID is the organization ID to use for authentication. Defaults to the value of the INFRACOST_ORG_ID environment variable.
	OrgID string `env:"INFRACOST_CI_ORG_ID" flag:"org-id;hidden" usage:"The organization ID to use for authentication"`

	// Project filters scanning to a single project when specified.
	Project string `flag:"project" usage:"Filter scanning to a single project"`

	// Branch is the branch name used for policy filtering.
	Branch string `flag:"branch" usage:"Branch name used for policy filtering" default:"main"`

	// EnableDashboard controls whether scan results are uploaded to the
	// Infracost dashboard via the addRun mutation. Defaults to true.
	EnableDashboard bool `flag:"enable-dashboard" usage:"Upload scan results to the Infracost dashboard" default:"true"`

	// RepoURL is the repository URL used for source links and PR URL construction.
	RepoURL string `flag:"repo-url" usage:"Repository URL for source links in comments"`

	// PRNumber is the pull request number, used for PR URL construction and comment posting.
	PRNumber int `flag:"pr-number" usage:"Pull request number to comment on"`

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

// PullRequestURL constructs the full pull request URL from RepoURL and PRNumber.
func (config *Config) PullRequestURL() string {
	if config.RepoURL == "" || config.PRNumber == 0 {
		return ""
	}
	return fmt.Sprintf("%s/pull/%d", config.RepoURL, config.PRNumber)
}

// UpdatePullRequestStatus updates the pull request status in the dashboard.
func (config *Config) UpdatePullRequestStatus(status dashboard.PullRequestStatus) error {
	ctx := context.Background()

	if len(config.Auth.AuthenticationToken) == 0 {
		return fmt.Errorf("authentication token is required: set INFRACOST_CLI_AUTHENTICATION_TOKEN")
	}

	prURL := config.PullRequestURL()
	if prURL == "" {
		return fmt.Errorf("cannot determine pull request URL: repo-url and pr-number are required")
	}

	tokenSource, err := config.Auth.Token(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve access token: %w", err)
	}

	httpClient := api.Client(ctx, tokenSource, config.OrgID)
	dashboardClient := config.Dashboard.Client(httpClient)
	return dashboardClient.UpdatePullRequestStatus(ctx, prURL, status)
}

func (config *Config) Process() {
	events.RegisterMetadata("cloudEnabled", os.Getenv("INFRACOST_ENABLE_CLOUD") == "true")
	events.RegisterMetadata("dashboardEnabled", config.EnableDashboard)
	events.RegisterMetadata("environment", config.Environment.String())
	events.RegisterMetadata("isDefaultPricingApiEndpoint", config.PricingEndpoint == "https://pricing.api.infracost.io")
}