package config

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/infracost/actions/tools/scanner/internal/api/dashboard"
	"github.com/infracost/actions/tools/scanner/internal/version"
	pkgscanner "github.com/infracost/cli/pkg/scanner"
	goprotoevent "github.com/infracost/go-proto/pkg/event"
	"github.com/infracost/go-proto/pkg/rat"
	"github.com/infracost/proto/gen/go/infracost/provider"
)

// RunInputOptions contains the parameters needed to build an addRun mutation input.
type RunInputOptions struct {
	BaseResult       *DirectoryResult
	HeadResult       *DirectoryResult
	GuardrailResults []goprotoevent.GuardrailResult
	CommentPosted    bool
	Currency         string

	// Command identifies the type of run: "comment" for diff/PR runs,
	// "upload" for baseline scans.
	Command string

	// VCS metadata — used by the dashboard to create repo, branch, and PR records.
	// See the README for the full addRun metadata reference.
	RepoURL    string
	PRNumber   int
	CommitSHA  string
	Branch     string
	BaseBranch string
}

// runMetadata is the top-level metadata sent with the addRun mutation.
// The dashboard reads specific fields from this to create VCS records,
// store commit info, and control run behavior.
type runMetadata struct {
	// Run context
	Command    string `json:"command"`
	Version    string `json:"version"`
	CIPlatform string `json:"ciPlatform"`

	// VCS provider
	VCSProvider string `json:"vcsProvider"`

	// Repository
	VCSRepositoryURL string `json:"vcsRepositoryUrl"`

	// Branch
	VCSBranch     string `json:"vcsBranch"`
	VCSBaseBranch string `json:"vcsBaseBranch,omitempty"`

	// Commit
	VCSCommitSHA string `json:"vcsCommitSha"`

	// Pull request (only for diff/comment runs)
	VCSPullRequestURL string `json:"vcsPullRequestUrl,omitempty"`
	VCSPullRequestID  string `json:"vcsPullRequestId,omitempty"`

	// Dashboard
	DashboardEnabled bool `json:"dashboardEnabled"`

	// Nested repo metadata for backwards compatibility — the dashboard
	// merges this into combinedMetadata alongside top-level fields.
	RepoMetadata repoMetadata `json:"repoMetadata"`
}

// repoMetadata is the nested metadata the dashboard merges with top-level fields.
type repoMetadata struct {
	VCSRepositoryURL  string `json:"vcsRepositoryUrl"`
	VCSBranch         string `json:"vcsBranch"`
	VCSBaseBranch     string `json:"vcsBaseBranch,omitempty"`
	VCSCommitSHA      string `json:"vcsCommitSha"`
	VCSPullRequestURL string `json:"vcsPullRequestUrl,omitempty"`
}

// BuildRunInput converts scan results into the dashboard addRun mutation input.
func BuildRunInput(opts RunInputOptions) dashboard.RunInput {
	projectResults := make([]dashboard.ProjectResultInput, 0, len(opts.HeadResult.Projects))

	var baseByName map[string]*pkgscanner.ProjectResult
	if opts.BaseResult != nil {
		baseByName = make(map[string]*pkgscanner.ProjectResult, len(opts.BaseResult.Projects))
		for i := range opts.BaseResult.Projects {
			baseByName[opts.BaseResult.Projects[i].Name] = &opts.BaseResult.Projects[i]
		}
	}

	for i := range opts.HeadResult.Projects {
		head := &opts.HeadResult.Projects[i]
		base := baseByName[head.Name]

		pr := dashboard.ProjectResultInput{
			ProjectName: head.Name,
			Breakdown:   buildBreakdownInput(head),
		}

		if base != nil {
			pastBreakdown := buildBreakdownInput(base)
			pr.PastBreakdown = &pastBreakdown
			diff := buildDiffBreakdownInput(base, head)
			pr.Diff = &diff
		}

		pr.FinopsPolicyResults = buildFinopsPolicyResults(head.FinopsResults)
		pr.TagPolicyResults = buildTagPolicyResults(head.TagPolicyResults)

		projectResults = append(projectResults, pr)
	}

	var prURL string
	var prID string
	if opts.PRNumber > 0 && opts.RepoURL != "" {
		prURL = fmt.Sprintf("%s/pull/%d", opts.RepoURL, opts.PRNumber)
		prID = fmt.Sprintf("%d", opts.PRNumber)
	}

	metadata := runMetadata{
		Command:          opts.Command,
		Version:          version.Version,
		CIPlatform:       "github_actions",
		VCSProvider:      "github",
		VCSRepositoryURL: opts.RepoURL,
		VCSBranch:        opts.Branch,
		VCSBaseBranch:    opts.BaseBranch,
		VCSCommitSHA:     opts.CommitSHA,
		VCSPullRequestURL: prURL,
		VCSPullRequestID:  prID,
		DashboardEnabled: true,
		RepoMetadata: repoMetadata{
			VCSRepositoryURL:  opts.RepoURL,
			VCSBranch:         opts.Branch,
			VCSBaseBranch:     opts.BaseBranch,
			VCSCommitSHA:      opts.CommitSHA,
			VCSPullRequestURL: prURL,
		},
	}

	posted := opts.CommentPosted
	input := dashboard.RunInput{
		ProjectResults:           projectResults,
		Currency:                 opts.Currency,
		TimeGenerated:            time.Now().UTC().Format(time.RFC3339),
		PoliciesAlreadyEvaluated: true,
		ClientPostedComment:      &posted,
		Metadata:                 structToJSON(metadata),
	}

	if len(opts.GuardrailResults) > 0 {
		input.GuardrailResults = buildGuardrailResults(opts.GuardrailResults)
	}

	return input
}

func buildBreakdownInput(result *pkgscanner.ProjectResult) dashboard.BreakdownInput {
	monthlyCost := orZero(result.TotalMonthlyCost)

	resources := make([]map[string]any, 0, len(result.Resources))
	for _, r := range result.Resources {
		resources = append(resources, resourceToJSON(r))
	}

	return dashboard.BreakdownInput{
		TotalHourlyCost:  monthlyCost.Div(rat.New(730)).String(),
		TotalMonthlyCost: monthlyCost.String(),
		Resources:        resources,
	}
}

func buildDiffBreakdownInput(base, head *pkgscanner.ProjectResult) dashboard.BreakdownInput {
	headCost := orZero(head.TotalMonthlyCost)
	baseCost := orZero(base.TotalMonthlyCost)
	diffCost := headCost.Sub(baseCost)

	return dashboard.BreakdownInput{
		TotalHourlyCost:  diffCost.Div(rat.New(730)).String(),
		TotalMonthlyCost: diffCost.String(),
	}
}

func buildGuardrailResults(results []goprotoevent.GuardrailResult) []dashboard.GuardrailResultInput {
	out := make([]dashboard.GuardrailResultInput, 0, len(results))
	for _, gr := range results {
		out = append(out, dashboard.GuardrailResultInput{
			GuardrailID:            gr.GuardrailID,
			Triggered:              gr.Triggered,
			PRComment:              gr.PRComment,
			BlockPR:                gr.BlockPR,
			TriggeringProjectNames: gr.TriggeringProjectNames,
			Increase:               ratString(gr.Increase),
			PercentIncrease:        ratString(gr.PercentIncrease),
			TotalMonthlyCost:       ratString(gr.TotalMonthlyCost),
		})
	}
	return out
}

func buildFinopsPolicyResults(results []*provider.FinopsPolicyResult) []map[string]any {
	if len(results) == 0 {
		return nil
	}
	var out []map[string]any
	for _, r := range results {
		out = append(out, structToJSON(r))
	}
	return out
}

func buildTagPolicyResults(results []goprotoevent.TaggingPolicyResult) []map[string]any {
	if len(results) == 0 {
		return nil
	}
	var out []map[string]any
	for _, r := range results {
		out = append(out, structToJSON(r))
	}
	return out
}

func resourceToJSON(r *provider.Resource) map[string]any {
	return structToJSON(r)
}

func structToJSON(v any) map[string]any {
	b, _ := json.Marshal(v)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}

func ratString(r *rat.Rat) string {
	if r == nil {
		return "0"
	}
	return r.String()
}