package commands

import (
	"context"
	"fmt"

	"github.com/infracost/actions/tools/scanner/internal/api"
	"github.com/infracost/actions/tools/scanner/internal/api/dashboard"
	"github.com/infracost/actions/tools/scanner/internal/config"
	"github.com/infracost/actions/tools/scanner/internal/vcsurl"
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
			provider, err := resolveVCSProvider(cfg)
			if err != nil {
				return err
			}
			return updatePullRequestStatus(cfg, provider, args.repoURL, args.prNumber, status)
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

func updatePullRequestStatus(cfg *config.Config, provider, repoURL string, prNumber int, status dashboard.PullRequestStatus) error {
	prURL, err := vcsurl.PullRequest(provider, repoURL, prNumber)
	if err != nil {
		return err
	}
	if prURL == "" {
		return fmt.Errorf("cannot determine pull request URL: repo-url and pr-number are required")
	}

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
