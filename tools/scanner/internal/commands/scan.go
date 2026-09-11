package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/infracost/actions/tools/scanner/internal/api"
	"github.com/infracost/actions/tools/scanner/internal/config"
	"github.com/infracost/actions/tools/scanner/internal/git"
	"github.com/infracost/actions/tools/scanner/internal/vcsurl"
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
	cmd, _ := scanCommand(cfg)
	return cmd
}

// scanCommand also returns the args it binds, so tests can drive the real
// flag registration and read what parsing produced.
func scanCommand(cfg *config.Config) (*cobra.Command, *scanArgs) {
	var args scanArgs

	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan a directory and upload baseline results to the Infracost dashboard",
		RunE: func(_ *cobra.Command, _ []string) error {
			return scan(cfg, &args)
		},
	}

	scanCmd.Flags().StringVar(&args.path, "path", "", "Path to the directory to scan")
	scanCmd.Flags().StringVar(&args.repoURL, "repo-url", cfg.VCS.RepositoryURL, "Repository URL for metadata")
	scanCmd.Flags().StringVar(&args.project, "project", "", "Filter scanning to a single project")
	scanCmd.Flags().StringVar(&args.pipelineRunID, "pipeline-run-id", cfg.VCS.PipelineRunID, "CI pipeline run ID (e.g. GitHub Actions run ID)")

	_ = scanCmd.MarkFlagRequired("path")

	return scanCmd, &args
}

func scan(cfg *config.Config, args *scanArgs) error {
	ctx := context.Background()
	startTime := time.Now()

	warnIgnoredPullRequestEnv()

	if args.repoURL == "" {
		return fmt.Errorf("cannot determine the repository URL: set INFRACOST_VCS_REPOSITORY_URL")
	}

	// A bad value fails here; an absent one is only fatal for the upload, so
	// the error is kept rather than returned.
	provider, providerErr := resolveVCSProvider(cfg)
	if providerErr != nil && cfg.VCSProvider != "" {
		return providerErr
	}

	// Nothing else here validates it, and it is uploaded as run metadata: a
	// copy-pasted clone URL would store its token. The provider only widens
	// the check, so an unresolved one stays strict.
	if err := vcsurl.CheckRepoURL(provider, args.repoURL); err != nil {
		return err
	}

	if len(cfg.Auth.AuthenticationToken) == 0 {
		return fmt.Errorf("authentication token is required: set INFRACOST_CLI_AUTHENTICATION_TOKEN")
	}

	tokenSource, err := cfg.Auth.Token(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve access token: %w", err)
	}
	httpClient := api.Client(ctx, tokenSource, cfg.OrgID)

	dashboardClient := cfg.Dashboard.Client(httpClient)
	branch := firstNonEmpty(cfg.VCS.Branch, git.RevParse(args.path, "--abbrev-ref", "HEAD"))
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

	uploadEnabled := !cfg.DisableDashboard && runParams.CloudEnabled

	// Only the upload carries the provider, so an unset one must not fail a
	// local run with the dashboard disabled.
	if uploadEnabled && providerErr != nil {
		return providerErr
	}

	headSHA := git.RevParse(args.path, "HEAD")
	commit := git.GetCommitInfo(args.path, headSHA)
	timestamp, err := normaliseTimestamp("INFRACOST_VCS_COMMIT_TIMESTAMP", firstNonEmpty(cfg.VCS.CommitTimestamp, commit.Timestamp))
	if err != nil {
		return err
	}

	runOpts := config.RunInputOptions{
		CommentPosted:     false,
		Command:           "upload",
		CIPlatform:        ciPlatform(),
		VCSProvider:       provider,
		RepoURL:           args.repoURL,
		RepoID:            runParams.RepositoryID,
		RepoName:          runParams.RepositoryName,
		CommitSHA:         firstNonEmpty(cfg.VCS.CommitSHA, headSHA),
		CommitMessage:     firstNonEmpty(cfg.VCS.CommitMessage, commit.Message),
		CommitAuthorName:  firstNonEmpty(cfg.VCS.CommitAuthorName, commit.AuthorName),
		CommitAuthorEmail: firstNonEmpty(cfg.VCS.CommitAuthorEmail, commit.AuthorEmail),
		CommitTimestamp:   timestamp,
		Branch:            branch,
		PipelineRunID:     args.pipelineRunID,
		UsageAPIEnabled:   runParams.UsageDefaults != nil && len(runParams.UsageDefaults.Resources) > 0,
	}

	result, err := cfg.ScanDirectory(ctx, args.path, token.AccessToken, runParams, nil, args.project, branch)
	if err != nil {
		if uploadEnabled {
			errInput := config.BuildErrorRunInput(runOpts, diagnostic.ErrorCodeCLIBreakdownError, "Failed to scan", err.Error())
			_, _ = dashboardClient.AddRun(ctx, errInput)
		}
		return fmt.Errorf("failed to scan path: %w", err)
	}

	if uploadEnabled {
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

	eventsClient := cfg.Events.Client(httpClient)
	trackRun(ctx, eventsClient, result, nil, time.Since(startTime).Seconds(), "upload")

	return nil
}
