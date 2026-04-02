package commands

import (
	"context"
	"fmt"

	"github.com/infracost/actions/tools/scanner/internal/api"
	"github.com/infracost/actions/tools/scanner/internal/config"
	"github.com/infracost/actions/tools/scanner/internal/git"
	"github.com/infracost/go-proto/pkg/diagnostic"
	"github.com/spf13/cobra"
)

type scanArgs struct {
	path          string
	repoURL       string
	project       string
	pipelineRunID string
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
	scanCmd.Flags().StringVar(&args.repoURL, "repo-url", "", "Repository URL for metadata")
	scanCmd.Flags().StringVar(&args.project, "project", "", "Filter scanning to a single project")
	scanCmd.Flags().StringVar(&args.pipelineRunID, "pipeline-run-id", "", "CI pipeline run ID (e.g. GitHub Actions run ID)")

	_ = scanCmd.MarkFlagRequired("path")

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
	branch := git.RevParse(args.path, "--abbrev-ref", "HEAD")
	rawRunParams, err := dashboardClient.RunParameters(ctx, args.repoURL, branch)
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

	commitSHA := git.RevParse(args.path, "HEAD")
	commit := git.GetCommitInfo(args.path, commitSHA)
	runOpts := config.RunInputOptions{
		CommentPosted:   false,
		Command:         "upload",
		RepoURL:         args.repoURL,
		RepoID:          runParams.RepositoryID,
		RepoName:        runParams.RepositoryName,
		CommitSHA:       commitSHA,
		CommitMessage:   commit.Message,
		CommitAuthorName: commit.AuthorName,
		CommitAuthorEmail: commit.AuthorEmail,
		CommitTimestamp: commit.Timestamp,
		Branch:          branch,
		PipelineRunID:   args.pipelineRunID,
		UsageAPIEnabled: runParams.UsageDefaults != nil && len(runParams.UsageDefaults.Resources) > 0,
	}

	result, err := cfg.ScanDirectory(ctx, args.path, token.AccessToken, runParams, nil, args.project, branch)
	if err != nil {
		if !cfg.DisableDashboard {
			errInput := config.BuildErrorRunInput(runOpts, diagnostic.ErrorCodeCLIBreakdownError, "Failed to scan", err.Error())
			_, _ = dashboardClient.AddRun(ctx, errInput)
		}
		return fmt.Errorf("failed to scan path: %w", err)
	}

	if !cfg.DisableDashboard {
		runOpts.HeadResult = result
		runOpts.Currency = result.Currency
		runOpts.UsageFilePath = result.UsageFilePath
		runOpts.HasConfigFile = result.HasConfigFile
		runOpts.ConfigFileHasUsageFile = result.ConfigFileHasUsageFile
		runInput := config.BuildRunInput(runOpts)
		_, err := dashboardClient.AddRun(ctx, runInput)
		if err != nil {
			return fmt.Errorf("failed to upload run to dashboard: %w", err)
		}
	}

	return nil
}