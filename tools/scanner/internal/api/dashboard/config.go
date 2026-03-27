package dashboard

import (
	"net/http"

	"github.com/infracost/cli/pkg/config/process"
	"github.com/infracost/cli/pkg/environment"
)

var (
	_ process.Processor = (*Config)(nil)

	defaultValues = map[string]map[string]string{
		environment.Production: {
			"endpoint": "https://dashboard.api.infracost.io",
		},
		environment.Development: {
			"endpoint": "https://dashboard.api.dev.infracost.io",
		},
		environment.Local: {
			"endpoint": "http://localhost:5000",
		},
	}
)

type Config struct {
	Environment string `flagvalue:"environment"`
	Endpoint    string `env:"INFRACOST_CLI_DASHBOARD_ENDPOINT" flag:"dashboard-endpoint;hidden" usage:"The endpoint for the Infracost dashboard"`

	// Can override this in tests.
	Client func(httpClient *http.Client) Client
}

func (c *Config) Process() {
	if c.Endpoint == "" {
		c.Endpoint = defaultValues[c.Environment]["endpoint"]
	}

	c.Client = func(httpClient *http.Client) Client {
		return &client{
			client: httpClient,
			config: c,
		}
	}
}
