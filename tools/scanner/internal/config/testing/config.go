package testing

import (
	"net/http"
	"os"
	"testing"

	"github.com/infracost/actions/tools/scanner/internal/api/dashboard"
	dashboardMock "github.com/infracost/actions/tools/scanner/internal/api/dashboard/mocks"
	"github.com/infracost/actions/tools/scanner/internal/api/events"
	eventsMock "github.com/infracost/actions/tools/scanner/internal/api/events/mocks"
	"github.com/infracost/actions/tools/scanner/internal/config"
	vcsMock "github.com/infracost/actions/tools/scanner/internal/mocks/vcs"
	"github.com/infracost/cli/pkg/auth"
	"github.com/infracost/cli/pkg/environment"
	"github.com/infracost/cli/pkg/logging"
	"github.com/infracost/cli/pkg/plugins"
	"github.com/rs/zerolog"
)

// Mocks holds all the mock clients created by Config so callers can set
// expectations before invoking Scan.
type Mocks struct {
	Dashboard *dashboardMock.MockClient
	Events    *eventsMock.MockClient
	VCS       *vcsMock.MockVCS
}

// Config returns a Config pre-wired with mock clients for testing.
// Requires INFRACOST_CLI_AUTHENTICATION_TOKEN to be set; skips the test otherwise.
func Config(t *testing.T) (config.Config, *Mocks) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	token := auth.AuthenticationToken(os.Getenv("INFRACOST_CLI_AUTHENTICATION_TOKEN"))
	if len(token) == 0 {
		t.Skip("INFRACOST_CLI_AUTHENTICATION_TOKEN not set, skipping integration test")
	}

	m := &Mocks{
		Dashboard: dashboardMock.NewMockClient(t),
		Events:    eventsMock.NewMockClient(t),
		VCS:       vcsMock.NewMockVCS(t),
	}

	cfg := config.Config{
		Environment: environment.Environment{
			Value: environment.Local,
		},
		OrgID:           "testing-organization",
		PricingEndpoint: "https://pricing.api.infracost.io",
		Plugins: plugins.Config{
			ManifestURL: "https://releases.infracost.io/plugins/manifest.json",
			AutoUpdate:  false,
		},
		Dashboard: dashboard.Config{
			Environment: environment.Local,
			Client: func(*http.Client) dashboard.Client {
				return m.Dashboard
			},
		},
		Events: events.Config{
			ClientFn: func(*http.Client) events.Client {
				return m.Events
			},
		},
		Auth: auth.Config{
			ExternalConfig: auth.ExternalConfig{
				AuthenticationToken: token,
			},
			Environment: environment.Local,
		},
		Logging: logging.Config{
			WriteLevel: zerolog.TraceLevel.String(),
		},
	}
	cfg.Logging.ForTest(t)
	return cfg, m
}