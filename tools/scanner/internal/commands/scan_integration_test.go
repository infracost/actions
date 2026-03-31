package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/infracost/actions/tools/scanner/internal/api/dashboard"
	"github.com/infracost/actions/tools/scanner/internal/config"
	testingconfig "github.com/infracost/actions/tools/scanner/internal/config/testing"
	"github.com/stretchr/testify/mock"
)

func runScan(t *testing.T, cfg *config.Config, path string) error {
	t.Helper()
	return scan(cfg, &scanArgs{path: path})
}

func TestScan_BasicUpload(t *testing.T) {
	cfg, m := testingconfig.Config(t)
	processPlugins(&cfg)

	m.Dashboard.EXPECT().
		RunParameters(mock.Anything, mock.Anything, mock.Anything).
		Return(emptyRunParams(), nil)

	m.Dashboard.EXPECT().
		AddRun(mock.Anything, mock.Anything).
		Run(func(_ context.Context, input dashboard.RunInput) {
			// Verify this is an upload (not a comment) run.
			metadata, ok := input.Metadata["command"]
			if !ok || metadata != "upload" {
				t.Errorf("expected command metadata to be 'upload', got %v", metadata)
			}

			// Should have project results but no past breakdowns.
			if len(input.ProjectResults) == 0 {
				t.Error("expected at least one project result")
			}
			for _, pr := range input.ProjectResults {
				if pr.PastBreakdown != nil {
					t.Errorf("expected no past breakdown for scan upload, got one for project %q", pr.ProjectName)
				}
				if pr.Diff != nil {
					t.Errorf("expected no diff for scan upload, got one for project %q", pr.ProjectName)
				}
			}
		}).
		Return(dashboard.AddRunResult{
			ID:       "test-run-id",
			CloudURL: "https://dashboard.infracost.io/org/test-org/runs/test-run-id",
		}, nil)

	err := runScan(t, &cfg, filepath.Join(testdataDir(), "basic", "head"))
	if err != nil {
		t.Fatalf("scan() returned error: %v", err)
	}
}

func TestScan_DashboardDisabled(t *testing.T) {
	cfg, m := testingconfig.Config(t)
	cfg.EnableDashboard = false
	processPlugins(&cfg)

	m.Dashboard.EXPECT().
		RunParameters(mock.Anything, mock.Anything, mock.Anything).
		Return(emptyRunParams(), nil)

	// AddRun should NOT be called when dashboard is disabled.

	err := runScan(t, &cfg, filepath.Join(testdataDir(), "basic", "head"))
	if err != nil {
		t.Fatalf("scan() returned error: %v", err)
	}
}

func TestScan_DashboardError(t *testing.T) {
	cfg, m := testingconfig.Config(t)
	processPlugins(&cfg)

	m.Dashboard.EXPECT().
		RunParameters(mock.Anything, mock.Anything, mock.Anything).
		Return(emptyRunParams(), nil)

	m.Dashboard.EXPECT().
		AddRun(mock.Anything, mock.Anything).
		Return(dashboard.AddRunResult{}, fmt.Errorf("dashboard unavailable"))

	err := runScan(t, &cfg, filepath.Join(testdataDir(), "basic", "head"))
	if err == nil {
		t.Fatal("expected error from scan() when dashboard fails")
	}
}