package commands

import (
	"fmt"

	"github.com/infracost/actions/tools/scanner/internal/api/dashboard"
	"github.com/infracost/actions/tools/scanner/internal/config"
	"github.com/spf13/cobra"
)

func Status(cfg *config.Config) *cobra.Command {
	var status string

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Update pull request status in the Infracost dashboard",
		RunE: func(_ *cobra.Command, _ []string) error {
			status := dashboard.PullRequestStatus(status)
			switch status {
			case dashboard.PullRequestStatusOpen, dashboard.PullRequestStatusMerged, dashboard.PullRequestStatusClosed:
			default:
				return fmt.Errorf("invalid status %q: must be OPEN, MERGED, or CLOSED", status)
			}
			return cfg.UpdatePullRequestStatus(status)
		},
	}

	statusCmd.Flags().StringVar(&status, "status", "", "Pull request status (OPEN, MERGED, CLOSED)")
	_ = statusCmd.MarkFlagRequired("status")

	return statusCmd
}
