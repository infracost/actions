package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/infracost/actions/tools/scanner/internal/api"
	"github.com/infracost/actions/tools/scanner/internal/config"
	"github.com/infracost/actions/tools/scanner/internal/git"
	pkgscanner "github.com/infracost/cli/pkg/scanner"
	"github.com/infracost/go-proto/pkg/diagnostic"
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
	prURL         string
	prNumber      int
	prTitle       string
	prAuthor      string
	prLabels      []string
	repoURL       string
	project       string
	pipelineRunID string
	githubToken   string
	githubOwner   string
	githubRepo    string
}

// diffContext is the VCS metadata for one diff run, resolved environment then
// flag then git, once, so the policy lookup and the metadata cannot disagree.
type diffContext struct {
	provider string
	repoURL  string
	prURL    string
	prNumber int
	prTitle  string
	prAuthor string
	prLabels []string
	branch   string
	// baseBranch selects the policy set: RunParameters takes it and returns the
	// guardrails, budgets and policies this run is judged against.
	baseBranch        string
	commitSHA         string
	baseCommitSHA     string
	commitMessage     string
	commitAuthorName  string
	commitAuthorEmail string
	commitTimestamp   string
	pipelineRunID     string
}

// ScanResult holds the outcome of a scan, including whether policies or
// guardrails require the PR to be blocked.
type ScanResult struct {
	BlockPR bool
	Reasons []string
}

func Diff(cfg *config.Config, results *ScanResult) *cobra.Command {
	cmd, _ := diffCommand(cfg, results)
	return cmd
}

// diffCommand also returns the args it binds, so tests can drive the real
// flag registration and read what parsing produced.
func diffCommand(cfg *config.Config, results *ScanResult) (*cobra.Command, *diffArgs) {
	var args diffArgs

	diffCmd := &cobra.Command{
		Use:   "diff",
		Short: "Scan base and head branches, compute cost diff, and post a PR comment",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx := context.Background()
			vcsCtx, err := resolveDiffContext(cfg, &args)
			if err != nil {
				return err
			}
			client, err := newVCSClient(ctx, &args, vcsCtx)
			if err != nil {
				return fmt.Errorf("failed to create VCS client: %w", err)
			}
			return diff(cfg, &args, vcsCtx, client, results)
		},
	}

	// The VCS flags default to the hydrated INFRACOST_VCS_* value, so an
	// explicit flag overrides the environment. main must PreProcess first.
	diffCmd.Flags().StringVar(&args.basePath, "base-path", "", "Path to the base branch checkout")
	diffCmd.Flags().StringVar(&args.headPath, "head-path", "", "Path to the head (PR) branch checkout")
	diffCmd.Flags().StringVar(&args.prURL, "pr-url", cfg.VCS.PullRequestURL, "Pull request URL, the key the dashboard matches on")
	diffCmd.Flags().IntVar(&args.prNumber, "pr-number", cfg.VCS.PullRequestID, "Pull request number to comment on")
	diffCmd.Flags().StringVar(&args.prTitle, "pr-title", cfg.VCS.PullRequestTitle, "Pull request title")
	diffCmd.Flags().StringVar(&args.prAuthor, "pr-author", cfg.VCS.PullRequestAuthor, "Pull request author")
	diffCmd.Flags().StringSliceVar(&args.prLabels, "pr-labels", cfg.VCS.Labels(), "Pull request labels")
	diffCmd.Flags().StringVar(&args.repoURL, "repo-url", cfg.VCS.RepositoryURL, "Repository URL for source links in comments")
	diffCmd.Flags().StringVar(&args.pipelineRunID, "pipeline-run-id", cfg.VCS.PipelineRunID, "CI pipeline run ID (e.g. GitHub Actions run ID)")
	diffCmd.Flags().StringVar(&args.project, "project", "", "Filter scanning to a single project")
	diffCmd.Flags().StringVar(&args.githubToken, "github-token", os.Getenv("GITHUB_TOKEN"), "API token for posting comments")
	diffCmd.Flags().StringVar(&args.githubOwner, "github-owner", "", "GitHub repository owner (derived from the repo URL when unset)")
	diffCmd.Flags().StringVar(&args.githubRepo, "github-repo", "", "GitHub repository name (derived from the repo URL when unset)")

	// pr-number, github-owner and github-repo were MarkFlagRequired, which asks
	// only about the command line; resolveDiffContext names the variable.
	_ = diffCmd.MarkFlagRequired("base-path")
	_ = diffCmd.MarkFlagRequired("head-path")

	return diffCmd, &args
}

// resolveDiffContext collapses environment, flags and git into the single set
// of values the run is uploaded and judged with.
func resolveDiffContext(cfg *config.Config, args *diffArgs) (diffContext, error) {
	provider, err := resolveVCSProvider(cfg)
	if err != nil {
		return diffContext{}, err
	}

	if args.repoURL == "" {
		return diffContext{}, fmt.Errorf("cannot determine the repository URL: set INFRACOST_VCS_REPOSITORY_URL")
	}

	// Fail before scanning: diff is always a pull request run, so a missing or
	// unbuildable PR URL would upload as a branch run and lose the PR.
	prURL, prNumber, err := resolvePullRequest(provider, args.repoURL, args.prURL, args.prNumber)
	if err != nil {
		return diffContext{}, err
	}

	headSHA := git.RevParse(args.headPath, "HEAD")
	commit := git.GetCommitInfo(args.headPath, headSHA)
	timestamp, err := normaliseTimestamp("INFRACOST_VCS_COMMIT_TIMESTAMP", firstNonEmpty(cfg.VCS.CommitTimestamp, commit.Timestamp))
	if err != nil {
		return diffContext{}, err
	}

	return diffContext{
		provider:   provider,
		repoURL:    args.repoURL,
		prURL:      prURL,
		prNumber:   prNumber,
		prTitle:    args.prTitle,
		prAuthor:   args.prAuthor,
		prLabels:   args.prLabels,
		branch:     firstNonEmpty(cfg.VCS.Branch, git.RevParse(args.headPath, "--abbrev-ref", "HEAD")),
		baseBranch: firstNonEmpty(cfg.VCS.BaseBranch, git.RevParse(args.basePath, "--abbrev-ref", "HEAD")),
		commitSHA:  firstNonEmpty(cfg.VCS.CommitSHA, headSHA),
		// No v0.1 name: the base commit is only knowable from the checkout.
		baseCommitSHA:     git.RevParse(args.basePath, "HEAD"),
		commitMessage:     firstNonEmpty(cfg.VCS.CommitMessage, commit.Message),
		commitAuthorName:  firstNonEmpty(cfg.VCS.CommitAuthorName, commit.AuthorName),
		commitAuthorEmail: firstNonEmpty(cfg.VCS.CommitAuthorEmail, commit.AuthorEmail),
		commitTimestamp:   timestamp,
		pipelineRunID:     args.pipelineRunID,
	}, nil
}

func newVCSClient(ctx context.Context, args *diffArgs, vcsCtx diffContext) (vcs.VCS, error) {
	owner, repo, err := resolveOwnerRepo(vcsCtx.provider, vcsCtx.repoURL, args.githubOwner, args.githubRepo)
	if err != nil {
		return nil, err
	}
	return github.New(ctx, owner, repo, args.githubToken, int32(vcsCtx.prNumber), github.Options{}) //nolint:gosec // PR numbers won't overflow int32
}

func diff(cfg *config.Config, args *diffArgs, vcsCtx diffContext, vcsClient vcs.VCS, results *ScanResult) error {
	ctx := context.Background()
	startTime := time.Now()

	if len(cfg.Auth.AuthenticationToken) == 0 {
		return fmt.Errorf("authentication token is required: set INFRACOST_CLI_AUTHENTICATION_TOKEN")
	}

	tokenSource, err := cfg.Auth.Token(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve access token: %w", err)
	}
	httpClient := api.Client(ctx, tokenSource, cfg.OrgID)

	dashboardClient := cfg.Dashboard.Client(httpClient)
	rawRunParams, err := dashboardClient.RunParameters(ctx, vcsCtx.repoURL, vcsCtx.baseBranch)
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

	runOpts := config.RunInputOptions{
		CIPlatform:        ciPlatform(),
		VCSProvider:       vcsCtx.provider,
		RepoURL:           vcsCtx.repoURL,
		RepoID:            runParams.RepositoryID,
		RepoName:          runParams.RepositoryName,
		PRURL:             vcsCtx.prURL,
		PRNumber:          vcsCtx.prNumber,
		PRTitle:           vcsCtx.prTitle,
		PRAuthor:          vcsCtx.prAuthor,
		PRLabels:          vcsCtx.prLabels,
		CommitSHA:         vcsCtx.commitSHA,
		CommitMessage:     vcsCtx.commitMessage,
		CommitAuthorName:  vcsCtx.commitAuthorName,
		CommitAuthorEmail: vcsCtx.commitAuthorEmail,
		CommitTimestamp:   vcsCtx.commitTimestamp,
		Branch:            vcsCtx.branch,
		BaseBranch:        vcsCtx.baseBranch,
		BaseCommitSHA:     vcsCtx.baseCommitSHA,
		PipelineRunID:     vcsCtx.pipelineRunID,
	}

	uploadEnabled := !cfg.DisableDashboard && runParams.CloudEnabled

	baseResult, err := cfg.ScanDirectory(ctx, args.basePath, token.AccessToken, runParams, nil, args.project, vcsCtx.baseBranch)
	if err != nil {
		if uploadEnabled {
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

	headResult, err := cfg.ScanDirectory(ctx, args.headPath, token.AccessToken, runParams, previousAddresses, args.project, vcsCtx.baseBranch)
	if err != nil {
		if uploadEnabled {
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
		RepoURL:                  vcsCtx.repoURL,
		CommitSHA:                vcsCtx.commitSHA,
		Branch:                   vcsCtx.baseBranch,
		OrgSlug:                  runParams.OrganizationSlug,
		RepoID:                   runParams.RepositoryID,
		RepoName:                 runParams.RepositoryName,
	})

	// Upload run results to the dashboard and set the cloud URL in the comment.
	// TODO: on failure, post the comment without the cloud URL and include a
	// message explaining that this run could not be uploaded to the dashboard.
	if uploadEnabled {
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
		// The VCS library builds the dashboard run link itself from OrgSlug,
		// RepoID and RunID (see comment.Data.runURL), so pass the run ID and
		// mark cloud as enabled rather than a pre-built URL.
		data.CloudEnabled = true
		data.RunID = addRunResult.ID
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
