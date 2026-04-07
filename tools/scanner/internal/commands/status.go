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
	prNumber int
}

func Status(cfg *config.Config) *cobra.Command {
	var args statusArgs

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Update pull request status in the Infracost dashboard",
		RunE: func(_ *cobra.Command, _ []string) error {
			status := dashboard.PullRequestStatus(args.status)
			switch status {
			case dashboard.PullRequestStatusOpen, dashboard.PullRequestStatusMerged, dashboard.PullRequestStatusClosed:
			default:
				return fmt.Errorf("invalid status %q: must be OPEN, MERGED, or CLOSED", status)
			}
			return updatePullRequestStatus(cfg, args.repoURL, args.prNumber, status)
		},
	}

	statusCmd.Flags().StringVar(&args.status, "status", "", "Pull request status (OPEN, MERGED, CLOSED)")
	statusCmd.Flags().StringVar(&args.repoURL, "repo-url", "", "Repository URL")
	statusCmd.Flags().IntVar(&args.prNumber, "pr-number", 0, "Pull request number")
	_ = statusCmd.MarkFlagRequired("status")
	_ = statusCmd.MarkFlagRequired("repo-url")
	_ = statusCmd.MarkFlagRequired("pr-number")

	return statusCmd
}

func updatePullRequestStatus(cfg *config.Config, repoURL string, prNumber int, status dashboard.PullRequestStatus) error {
	if repoURL == "" || prNumber == 0 {
		return fmt.Errorf("cannot determine pull request URL: repo-url and pr-number are required")
	}
	prURL := fmt.Sprintf("%s/pull/%d", repoURL, prNumber)

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
	return dashboardClient.UpdatePullRequestStatus(ctx, prURL, status)
}
