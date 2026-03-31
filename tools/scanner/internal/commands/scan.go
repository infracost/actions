package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/infracost/actions/tools/scanner/internal/api"
	"github.com/infracost/actions/tools/scanner/internal/config"
	"github.com/spf13/cobra"
)

type scanArgs struct {
	path      string
	commitSHA string
}

func Scan(cfg *config.Config) *cobra.Command {
	var args scanArgs

	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan a directory and upload baseline results to the Infracost dashboard",
		RunE: func(_ *cobra.Command, _ []string) error {
			return scan(cfg, &args)
		},
	}

	scanCmd.Flags().StringVar(&args.path, "path", "", "Path to the directory to scan")
	scanCmd.Flags().StringVar(&args.commitSHA, "commit-sha", os.Getenv("GITHUB_SHA"), "Commit SHA for metadata")

	return scanCmd
}

func scan(cfg *config.Config, args *scanArgs) error {
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
	rawRunParams, err := dashboardClient.RunParameters(ctx, cfg.RepoURL, cfg.Branch)
	if err != nil {
		return fmt.Errorf("failed to fetch run parameters: %w", err)
	}

	runParams, err := config.ParseRunParameters(rawRunParams)
	if err != nil {
		return fmt.Errorf("failed to parse run parameters: %w", err)
	}

	token, err := tokenSource.Token()
	if err != nil {
		return fmt.Errorf("failed to retrieve access token: %w", err)
	}

	result, err := cfg.ScanDirectory(ctx, args.path, token.AccessToken, runParams, nil)
	if err != nil {
		return fmt.Errorf("failed to scan path: %w", err)
	}

	if cfg.EnableDashboard {
		runInput := config.BuildRunInput(config.RunInputOptions{
			HeadResult:    result,
			CommentPosted: false,
			Currency:      result.Currency,
			Command:       "upload",
			RepoURL:       cfg.RepoURL,
			CommitSHA:     args.commitSHA,
			Branch:        cfg.Branch,
		})
		_, err := dashboardClient.AddRun(ctx, runInput)
		if err != nil {
			return fmt.Errorf("failed to upload run to dashboard: %w", err)
		}
	}

	return nil
}