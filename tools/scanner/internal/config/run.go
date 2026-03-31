package config

import (
	"encoding/json"
	"time"

	"github.com/infracost/actions/tools/scanner/internal/api/dashboard"
	"github.com/infracost/actions/tools/scanner/internal/version"
	pkgscanner "github.com/infracost/cli/pkg/scanner"
	goprotoevent "github.com/infracost/go-proto/pkg/event"
	"github.com/infracost/go-proto/pkg/rat"
	"github.com/infracost/proto/gen/go/infracost/provider"
)

// buildRunInput converts scan results into the dashboard addRun mutation input.
func buildRunInput(
	baseResult *DirectoryResult,
	headResult *DirectoryResult,
	guardrailResults []goprotoevent.GuardrailResult,
	commentPosted bool,
	currency string,
	repoURL string,
	prURL string,
	commitSHA string,
	branch string,
) dashboard.RunInput {
	projectResults := make([]dashboard.ProjectResultInput, 0, len(headResult.Projects))

	// Match projects by name between base and head (same logic as buildCommentData).
	baseByName := make(map[string]*pkgscanner.ProjectResult, len(baseResult.Projects))
	for i := range baseResult.Projects {
		baseByName[baseResult.Projects[i].Name] = &baseResult.Projects[i]
	}

	for i := range headResult.Projects {
		head := &headResult.Projects[i]
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

	posted := commentPosted
	input := dashboard.RunInput{
		ProjectResults:           projectResults,
		Currency:                 currency,
		TimeGenerated:            time.Now().UTC().Format(time.RFC3339),
		PoliciesAlreadyEvaluated: true,
		ClientPostedComment:      &posted,
		Metadata: map[string]any{
			"vcsProvider":      "github",
			"infracostCommand": "scan",
			"version":          version.Version,
			"repoMetadata": map[string]any{
				"repoUrl":            repoURL,
				"vcsPullRequestUrl":  prURL,
				"commitSha":          commitSHA,
				"branch":             branch,
			},
		},
	}

	if len(guardrailResults) > 0 {
		input.GuardrailResults = buildGuardrailResults(guardrailResults)
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