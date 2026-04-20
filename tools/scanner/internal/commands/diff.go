package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/infracost/actions/tools/scanner/internal/api"
	"github.com/infracost/actions/tools/scanner/internal/config"
	"github.com/infracost/actions/tools/scanner/internal/git"
	"github.com/infracost/go-proto/pkg/diagnostic"
	pkgscanner "github.com/infracost/cli/pkg/scanner"
	"github.com/infracost/proto/gen/go/infracost/parser/event"
	"github.com/infracost/proto/gen/go/infracost/provider"
	"github.com/infracost/vcs/pkg/vcs"
	"github.com/infracost/vcs/pkg/vcs/comment"
	"github.com/infracost/vcs/pkg/vcs/github"
	"github.com/spf13/cobra"
)

type diffArgs struct {
	basePath      string
	headPath      string
	prNumber      int
	prTitle       string
	prAuthor      string
	prLabels      []string
	repoURL       string
	project       string
	pipelineRunID string
	vcsProvider   string
	githubToken     string
	githubOwner     string
	githubRepo      string
}

// ScanResult holds the outcome of a scan, including whether policies or
// guardrails require the PR to be blocked.
type ScanResult struct {
	BlockPR bool
	Reasons []string
}

func Diff(cfg *config.Config, results *ScanResult) *cobra.Command {
	var args diffArgs

	diffCmd := &cobra.Command{
		Use:   "diff",
		Short: "Scan base and head branches, compute cost diff, and post a PR comment",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx := context.Background()
			client, err := newVCSClient(ctx, &args)
			if err != nil {
				return fmt.Errorf("failed to create VCS client: %w", err)
			}
			return diff(cfg, &args, client, results)
		},
	}

	diffCmd.Flags().StringVar(&args.basePath, "base-path", "", "Path to the base branch checkout")
	diffCmd.Flags().StringVar(&args.headPath, "head-path", "", "Path to the head (PR) branch checkout")
	diffCmd.Flags().IntVar(&args.prNumber, "pr-number", 0, "Pull request number to comment on")
	diffCmd.Flags().StringVar(&args.prTitle, "pr-title", "", "Pull request title")
	diffCmd.Flags().StringVar(&args.prAuthor, "pr-author", "", "Pull request author")
	diffCmd.Flags().StringSliceVar(&args.prLabels, "pr-labels", nil, "Pull request labels")
	diffCmd.Flags().StringVar(&args.repoURL, "repo-url", "", "Repository URL for source links in comments")
	diffCmd.Flags().StringVar(&args.pipelineRunID, "pipeline-run-id", "", "CI pipeline run ID (e.g. GitHub Actions run ID)")
	diffCmd.Flags().StringVar(&args.project, "project", "", "Filter scanning to a single project")
	diffCmd.Flags().StringVar(&args.vcsProvider, "vcs-provider", "github", "VCS provider to use for posting comments")
	diffCmd.Flags().StringVar(&args.githubToken, "github-token", os.Getenv("GITHUB_TOKEN"), "API token for posting comments")
	diffCmd.Flags().StringVar(&args.githubOwner, "github-owner", "", "GitHub repository owner")
	diffCmd.Flags().StringVar(&args.githubRepo, "github-repo", "", "GitHub repository name")

	_ = diffCmd.MarkFlagRequired("base-path")
	_ = diffCmd.MarkFlagRequired("head-path")
	_ = diffCmd.MarkFlagRequired("pr-number")
	_ = diffCmd.MarkFlagRequired("github-owner")
	_ = diffCmd.MarkFlagRequired("github-repo")

	return diffCmd
}

func newVCSClient(ctx context.Context, args *diffArgs) (vcs.VCS, error) {
	switch args.vcsProvider {
	case "github":
		return github.New(ctx, args.githubOwner, args.githubRepo, args.githubToken, int32(args.prNumber), github.Options{}) //nolint:gosec // PR numbers won't overflow int32
	default:
		return nil, fmt.Errorf("unsupported VCS provider: %q", args.vcsProvider)
	}
}

func diff(cfg *config.Config, args *diffArgs, vcsClient vcs.VCS, results *ScanResult) error {
	ctx := context.Background()
	startTime := time.Now()

	headCommitSHA := git.RevParse(args.headPath, "HEAD")
	headBranch := git.RevParse(args.headPath, "--abbrev-ref", "HEAD")
	baseBranch := git.RevParse(args.basePath, "--abbrev-ref", "HEAD")

	if len(cfg.Auth.AuthenticationToken) == 0 {
		return fmt.Errorf("authentication token is required: set INFRACOST_CLI_AUTHENTICATION_TOKEN")
	}

	tokenSource, err := cfg.Auth.Token(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve access token: %w", err)
	}
	httpClient := api.Client(ctx, tokenSource, cfg.OrgID)

	dashboardClient := cfg.Dashboard.Client(httpClient)
	rawRunParams, err := dashboardClient.RunParameters(ctx, args.repoURL, baseBranch)
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

	commit := git.GetCommitInfo(args.headPath, headCommitSHA)
	runOpts := config.RunInputOptions{
		RepoURL:           args.repoURL,
		RepoID:            runParams.RepositoryID,
		RepoName:          runParams.RepositoryName,
		PRNumber:          args.prNumber,
		PRTitle:           args.prTitle,
		PRAuthor:          args.prAuthor,
		PRLabels:          args.prLabels,
		CommitSHA:         headCommitSHA,
		CommitMessage:     commit.Message,
		CommitAuthorName:  commit.AuthorName,
		CommitAuthorEmail: commit.AuthorEmail,
		CommitTimestamp:   commit.Timestamp,
		Branch:            headBranch,
		BaseBranch:        baseBranch,
		BaseCommitSHA:     git.RevParse(args.basePath, "HEAD"),
		PipelineRunID:     args.pipelineRunID,
	}

	baseResult, err := cfg.ScanDirectory(ctx, args.basePath, token.AccessToken, runParams, nil, args.project, baseBranch)
	if err != nil {
		if !cfg.DisableDashboard {
			errInput := config.BuildErrorRunInput(runOpts, diagnostic.ErrorCodeCLIBreakdownError, "Failed to scan base branch", err.Error())
			_, _ = dashboardClient.AddRun(ctx, errInput)
		}
		return fmt.Errorf("failed to scan base path: %w", err)
	}

	// Build previous resource addresses from base results so the head scan
	// can determine which resources are new/modified/deleted.
	previousAddresses := make(map[string][]string)
	for _, p := range baseResult.Projects {
		var addrs []string
		for _, r := range p.Resources {
			addrs = append(addrs, r.Name)
		}
		previousAddresses[p.Name] = addrs
	}

	headResult, err := cfg.ScanDirectory(ctx, args.headPath, token.AccessToken, runParams, previousAddresses, args.project, baseBranch)
	if err != nil {
		if !cfg.DisableDashboard {
			errInput := config.BuildErrorRunInput(runOpts, diagnostic.ErrorCodeCLIBreakdownError, "Failed to scan head branch", err.Error())
			_, _ = dashboardClient.AddRun(ctx, errInput)
		}
		return fmt.Errorf("failed to scan head path: %w", err)
	}

	guardrailResults := pkgscanner.EvaluateGuardrails(runParams.Guardrails, baseResult.Projects, headResult.Projects)

	// Evaluate guardrails against the base branch to determine which were
	// already triggered before this PR — these are suppressed in the comment.
	previousGuardrailResults := pkgscanner.EvaluateGuardrails(runParams.Guardrails, nil, baseResult.Projects)

	// Evaluate budgets against all head resources.
	budgetResults := config.EvaluateBudgets(runParams.Budgets, headResult.Projects)

	usageAPIEnabled := runParams.UsageDefaults != nil && len(runParams.UsageDefaults.Resources) > 0

	data := config.BuildCommentData(config.CommentDataOptions{
		BaseResult:               baseResult,
		HeadResult:               headResult,
		GuardrailResults:         guardrailResults,
		PreviousGuardrailResults: previousGuardrailResults,
		BudgetResults:            budgetResults,
		FinopsPolicySettings:     runParams.FinopsPolicies,
		UsageAPIEnabled:          usageAPIEnabled,
		Currency:                 headResult.Currency,
		RepoURL:                  args.repoURL,
		CommitSHA:                headCommitSHA,
		Branch:                   baseBranch,
		OrgSlug:                  runParams.OrganizationSlug,
		RepoID:                   runParams.RepositoryID,
		RepoName:                 runParams.RepositoryName,
	})

	// Upload run results to the dashboard and set the cloud URL in the comment.
	// TODO: on failure, post the comment without the cloud URL and include a
	// message explaining that this run could not be uploaded to the dashboard.
	if !cfg.DisableDashboard {
		runOpts.BaseResult = baseResult
		runOpts.HeadResult = headResult
		runOpts.GuardrailResults = guardrailResults
		runOpts.BudgetResults = budgetResults
		runOpts.CommentPosted = true
		runOpts.Currency = headResult.Currency
		runOpts.Command = "comment"
		runOpts.UsageAPIEnabled = usageAPIEnabled
		runOpts.UsageFilePath = headResult.UsageFilePath
		runOpts.HasConfigFile = headResult.HasConfigFile
		runOpts.ConfigFileHasUsageFile = headResult.ConfigFileHasUsageFile
		runInput := config.BuildRunInput(runOpts)
		addRunResult, err := dashboardClient.AddRun(ctx, runInput)
		if err != nil {
			return fmt.Errorf("failed to upload run to dashboard: %w", err)
		}
		data.CloudURL = addRunResult.CloudURL
	}

	body, err := vcsClient.GenerateComment(data)
	if err != nil {
		return fmt.Errorf("failed to generate comment: %w", err)
	}

	if _, err := vcsClient.PostComment(ctx, body, vcs.BehaviorUpdate); err != nil {
		return fmt.Errorf("failed to post comment: %w", err)
	}

	eventsClient := cfg.Events.Client(httpClient)
	trackRun(ctx, eventsClient, headResult, baseResult, time.Since(startTime).Seconds(), "comment")
	trackDiff(ctx, eventsClient, headResult, baseResult)

	checkBlockingViolations(data, runParams.Guardrails, results)
	return nil
}

// checkBlockingViolations inspects the comment data for new guardrail or
// policy violations that should block the PR.
func checkBlockingViolations(data comment.Data, guardrails []*event.Guardrail, results *ScanResult) {
	// Build a set of guardrail IDs that only use total thresholds (no increase
	// thresholds) and were already triggered in the base branch. Only these
	// are eligible for suppression — increase thresholds measure the delta
	// between base and head, so "already triggered in base" is not meaningful.
	totalOnly := make(map[string]bool, len(guardrails))
	for _, g := range guardrails {
		if g.TotalThreshold != nil && g.IncreaseThreshold == nil && g.IncreasePercentThreshold == nil {
			totalOnly[g.Id] = true
		}
	}

	previouslyTriggered := make(map[string]bool, len(data.PreviousGuardrailResults))
	for _, gr := range data.PreviousGuardrailResults {
		if gr.Triggered && totalOnly[gr.GuardrailID] {
			previouslyTriggered[gr.GuardrailID] = true
		}
	}

	for _, gr := range data.GuardrailResults {
		if gr.Triggered && gr.BlockPR && !previouslyTriggered[gr.GuardrailID] {
			results.BlockPR = true
			results.Reasons = append(results.Reasons, fmt.Sprintf("guardrail %q triggered", gr.GuardrailName))
		}
	}

	// Build a set of policy slugs that were already failing in the base branch.
	previouslyFailing := make(map[string]bool)
	for _, policies := range [][]*provider.FinopsPolicyResult{data.PreviousFinOpsPolicyResults, data.PreviousSecurityPolicyResults} {
		for _, p := range policies {
			if len(p.FailingResources) > 0 {
				previouslyFailing[p.PolicySlug] = true
			}
		}
	}

	// Check policies (FinOps + Security) for new blocking failures.
	for _, policies := range [][]*provider.FinopsPolicyResult{data.FinOpsPolicyResults, data.SecurityPolicyResults} {
		for _, p := range policies {
			if p.BlockPullRequest && len(p.FailingResources) > 0 && !previouslyFailing[p.PolicySlug] {
				results.BlockPR = true
				results.Reasons = append(results.Reasons, fmt.Sprintf("policy %q has failing resources", p.PolicyName))
			}
		}
	}
}
