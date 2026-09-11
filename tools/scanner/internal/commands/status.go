package commands

import (
	"context"
	"fmt"

	"github.com/infracost/actions/tools/scanner/internal/api"
	"github.com/infracost/actions/tools/scanner/internal/api/dashboard"
	"github.com/infracost/actions/tools/scanner/internal/config"
	"github.com/spf13/cobra"
)

type statusArgs struct {
	status   string
	repoURL  string
	prURL    string
	prNumber int
}

func Status(cfg *config.Config) *cobra.Command {
	cmd, _ := statusCommand(cfg)
	return cmd
}

// statusCommand also returns the args it binds, so tests can drive the real
// flag registration and read what parsing produced.
func statusCommand(cfg *config.Config) (*cobra.Command, *statusArgs) {
	var args statusArgs

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Update pull request status in the Infracost dashboard",
		RunE: func(_ *cobra.Command, _ []string) error {
			prURL, err := resolveStatusPullRequest(cfg, &args)
			if err != nil {
				return err
			}
			return updatePullRequestStatus(cfg, prURL, args.repoURL, dashboard.PullRequestStatus(args.status))
		},
	}

	statusCmd.Flags().StringVar(&args.status, "status", cfg.VCS.PullRequestStatus, "Pull request status (OPEN, MERGED, CLOSED)")
	statusCmd.Flags().StringVar(&args.repoURL, "repo-url", cfg.VCS.RepositoryURL, "Repository URL")
	// Bound here as well as on diff: the two must take the same inputs, or the
	// status lands on a different pull request than the run.
	statusCmd.Flags().StringVar(&args.prURL, "pr-url", cfg.VCS.PullRequestURL, "Pull request URL, the key the dashboard matches on")
	statusCmd.Flags().IntVar(&args.prNumber, "pr-number", cfg.VCS.PullRequestID, "Pull request number")

	// All three were MarkFlagRequired, which only asks whether the flag was
	// set on the command line and so rejects an env-supplied value.
	return statusCmd, &args
}

// resolveStatusPullRequest validates the status inputs and settles the PR URL
// the dashboard is keyed on. It must agree byte for byte with the URL diff
// uploads, so it goes through the same resolver.
func resolveStatusPullRequest(cfg *config.Config, args *statusArgs) (string, error) {
	switch dashboard.PullRequestStatus(args.status) {
	case dashboard.PullRequestStatusOpen, dashboard.PullRequestStatusMerged, dashboard.PullRequestStatusClosed:
	case "":
		return "", fmt.Errorf("cannot determine the pull request status: set INFRACOST_VCS_PULL_REQUEST_STATUS to OPEN, MERGED, or CLOSED")
	default:
		return "", fmt.Errorf("invalid status %q: must be OPEN, MERGED, or CLOSED", args.status)
	}

	provider, err := resolveVCSProvider(cfg)
	if err != nil {
		return "", err
	}

	if args.repoURL == "" {
		return "", fmt.Errorf("cannot determine the repository URL: set INFRACOST_VCS_REPOSITORY_URL")
	}

	prURL, _, err := resolvePullRequest(provider, args.repoURL, args.prURL, args.prNumber)
	return prURL, err
}

func updatePullRequestStatus(cfg *config.Config, prURL, repoURL string, status dashboard.PullRequestStatus) error {
	ctx := context.Background()
	if len(cfg.Auth.AuthenticationToken) == 0 {
		return fmt.Errorf("authentication token is required: set INFRACOST_CLI_AUTHENTICATION_TOKEN")
	}

	tokenSource, err := cfg.Auth.Token(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve access token: %w", err)
	}

	httpClient := api.Client(ctx, tokenSource, cfg.OrgID)
	dashboardClient := cfg.Dashboard.Client(httpClient)

	if cfg.DisableDashboard {
		return nil
	}

	runParams, err := dashboardClient.RunParameters(ctx, repoURL, "")
	if err != nil {
		return fmt.Errorf("failed to fetch run parameters: %w", err)
	}
	if !runParams.CloudEnabled {
		return nil
	}

	return dashboardClient.UpdatePullRequestStatus(ctx, prURL, status)
}
