package config

import (
	"os"

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

	// DisableDashboard disables uploading scan results to the Infracost dashboard.
	DisableDashboard bool `env:"INFRACOST_CI_DISABLE_DASHBOARD"`

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

func (config *Config) Process() {
	events.RegisterMetadata("cloudEnabled", os.Getenv("INFRACOST_ENABLE_CLOUD") == "true")
	events.RegisterMetadata("dashboardEnabled", !config.DisableDashboard)
	events.RegisterMetadata("environment", config.Environment.String())
	events.RegisterMetadata("isDefaultPricingApiEndpoint", config.PricingEndpoint == "https://pricing.api.infracost.io")
}