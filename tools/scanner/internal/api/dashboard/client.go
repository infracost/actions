package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/infracost/actions/tools/scanner/internal/api/dashboard/graphql"
)

type RunParameters struct {
	OrganizationID   string `json:"organizationId"`
	OrganizationSlug string `json:"organizationSlug"`
	CloudEnabled     bool   `json:"cloudEnabled"`
	RepositoryID     string `json:"repositoryId"`
	RepositoryName   string `json:"repositoryName"`

	UsageDefaults     json.RawMessage   `json:"usageDefaults"`
	ProductionFilters []json.RawMessage `json:"productionFilters"`
	TagPolicies       []json.RawMessage `json:"tagPolicies"`
	FinopsPolicies    []json.RawMessage `json:"finopsPolicies"`
	Guardrails        []json.RawMessage `json:"guardrails"`
	Budgets           []json.RawMessage `json:"budgets"`
}

// RunInput is the input to the addRun mutation.
type RunInput struct {
	ProjectResults           []ProjectResultInput   `json:"projectResults"`
	Currency                 string                 `json:"currency,omitempty"`
	TimeGenerated            string                 `json:"timeGenerated,omitempty"`
	Metadata                 map[string]interface{} `json:"metadata,omitempty"`
	GuardrailResults         []GuardrailResultInput `json:"guardrailResults,omitempty"`
	BudgetResults            []BudgetResultInput    `json:"budgetResults,omitempty"`
	PoliciesAlreadyEvaluated bool                   `json:"policiesAlreadyEvaluated,omitempty"`
	ClientPostedComment      *bool                  `json:"clientPostedComment,omitempty"`

	Error *RunError `json:"error,omitempty"`

	// Unused — PullRequestCheckID is an internal dashboard ID created during
	// webhook-initiated flows (e.g. the GitHub App runner). Not applicable
	// when the scanner is invoked directly by CI.
	// PullRequestCheckID string `json:"pullRequestCheckId,omitempty"`

	// Unused — ExistingBreakdownSHAs and ExistingBreakdownIDs are a performance
	// optimization used by the runner to skip re-uploading unchanged breakdown
	// data. The runner maintains a persistent cache of breakdown SHAs across runs
	// and queries the dashboard's breakdownShas endpoint before building the run.
	// The scanner has no persistent state between runs, so it always uploads full
	// breakdowns. The dashboard handles this gracefully by creating new records.
	// ExistingBreakdownSHAs []string `json:"existingBreakdownShas,omitempty"`
	// ExistingBreakdownIDs  []string `json:"existingBreakdownIds,omitempty"`
}

// RunError represents a top-level run failure. This is sent when the scanner
// cannot complete at all (e.g. authentication failure, config error). Per-project
// errors are sent via ProjectMetadataInput.Errors instead.
type RunError struct {
	Code        string `json:"code"`
	Level       string `json:"level"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// ProjectResultInput represents a single project's cost data for addRun.
type ProjectResultInput struct {
	ProjectName         string                    `json:"projectName"`
	Breakdown           BreakdownInput            `json:"breakdown"`
	PastBreakdown       *BreakdownInput           `json:"pastBreakdown,omitempty"`
	Diff                *BreakdownInput           `json:"diff,omitempty"`
	Metadata            *ProjectMetadataInput     `json:"projectMetadata,omitempty"`
	TagPolicyResults    []map[string]interface{}  `json:"tagPolicyResults,omitempty"`
	FinopsPolicyResults []FinopsPolicyResultInput `json:"finopsPolicyResults,omitempty"`

	// Unused — BreakdownResultSha and PastBreakdownResultSha are computed by
	// the runner from the project name, workspace, errors, warnings, summary,
	// policy SHA, breakdown SHA, and resource checksums. They allow the
	// dashboard to deduplicate breakdown records. Requires a BreakdownSHA
	// input (from a persistent per-project cache) which the scanner does not
	// maintain. See calculateBreakdownResultSHA in runner/internal/service/runner/addrun.go.
	// BreakdownResultSha     string `json:"breakdownResultSha,omitempty"`
	// PastBreakdownResultSha string `json:"pastBreakdownResultSha,omitempty"`
}

// ProjectMetadataInput holds per-project metadata for addRun.
type ProjectMetadataInput struct {
	Path                string   `json:"path"`
	Type                string   `json:"type"`
	ConfigSha           string   `json:"configSha,omitempty"`
	TerraformModulePath string   `json:"terraformModulePath,omitempty"`
	TerraformWorkspace  string   `json:"terraformWorkspace,omitempty"`
	VCSSubPath          string   `json:"vcsSubPath,omitempty"`
	VCSCodeChanged      *bool    `json:"vcsCodeChanged,omitempty"`
	RemoteModuleCalls   []string `json:"remoteModuleCalls,omitempty"`

	// Unused — PolicySha and PastPolicySha are managed by the dashboard when
	// storing breakdowns. They are not set by the client (runner or scanner).
	// PolicySha     string `json:"policySha,omitempty"`
	// PastPolicySha string `json:"pastPolicySha,omitempty"`
	Errors   []ProjectDiagnostic `json:"errors,omitempty"`
	Warnings []ProjectDiagnostic `json:"warnings,omitempty"`
}

// ProjectDiagnostic represents a project-level error or warning.
type ProjectDiagnostic struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	IsError bool   `json:"isError"`
}

// FinopsPolicyResultInput represents a finops policy evaluation result for addRun.
type FinopsPolicyResultInput struct {
	Name                     string                             `json:"name"`
	PolicyID                 string                             `json:"policyId"`
	Message                  string                             `json:"message"`
	BlockPR                  bool                               `json:"blockPr"`
	PRComment                bool                               `json:"prComment"`
	OnlyNewResources         bool                               `json:"onlyNewResources"`
	AllCurrentResources      []string                           `json:"allCurrentResources"`
	Resources                []FinopsPolicyResourceInput        `json:"resources"`
	PastResources            []FinopsPolicyResourceInput        `json:"pastResources"`
	PassingResources         []FinopsPolicyPassingResourceInput `json:"passingResources"`
	TotalApplicableResources int                                `json:"totalApplicableResources"`
}

// FinopsPolicyResourceInput represents a failing resource in a finops policy result.
type FinopsPolicyResourceInput struct {
	NewResource          bool                     `json:"newResource"`
	Address              string                   `json:"address"`
	ResourceType         string                   `json:"resourceType"`
	Path                 string                   `json:"path"`
	ProjectName          string                   `json:"projectName"`
	ModulePath           string                   `json:"modulePath"`
	ModuleCallPath       string                   `json:"moduleCallPath"`
	ModuleCallStartLine  int                      `json:"moduleCallStartLine"`
	Issues               []FinopsPolicyIssueInput `json:"issues"`
	StartLine            int                      `json:"startLine"`
	EndLine              int                      `json:"endLine"`
	UnlabeledProjectName string                   `json:"unlabeledProjectName"`
	TerraformModulePath  string                   `json:"terraformModulePath"`
	TerraformWorkspace   string                   `json:"terraformWorkspace"`
	Checksum             string                   `json:"checksum"`
	ParserChecksum       string                   `json:"parserChecksum"`
	PastParserChecksum   string                   `json:"pastParserChecksum"`
}

// FinopsPolicyPassingResourceInput represents a passing resource in a finops policy result.
type FinopsPolicyPassingResourceInput struct {
	Address     string `json:"address"`
	ProjectName string `json:"projectName"`
}

// FinopsPolicyIssueInput represents an issue found by a finops policy.
type FinopsPolicyIssueInput struct {
	FromAddress                   string                       `json:"fromAddress"`
	Attribute                     string                       `json:"attribute"`
	Value                         string                       `json:"value"`
	Description                   string                       `json:"description"`
	SavingsDetails                *string                      `json:"savingsDetails,omitempty"`
	MonthlySavings                *string                      `json:"monthlySavings,omitempty"`
	MonthlyCarbonSavingsGramsCo2e *string                      `json:"monthlyCarbonSavingsGramsCo2e,omitempty"`
	MonthlyWaterSavingsLitres     *string                      `json:"monthlyWaterSavingsLitres,omitempty"`
	CurrentCostBreakdowns         []FinopsPolicyBreakdownInput `json:"currentCostBreakdowns,omitempty"`
	NewCostBreakdowns             []FinopsPolicyBreakdownInput `json:"newCostBreakdowns,omitempty"`
}

// FinopsPolicyBreakdownInput represents a cost breakdown for an issue.
type FinopsPolicyBreakdownInput struct {
	Name           string                             `json:"name"`
	HourlyCost     string                             `json:"hourlyCost"`
	MonthlyCost    string                             `json:"monthlyCost"`
	Metadata       FinopsPolicyBreakdownMetadataInput `json:"metadata"`
	CostComponents []FinopsPolicyCostComponentInput   `json:"costComponents"`
	SubResources   []FinopsPolicyBreakdownInput       `json:"subresources"`
}

// FinopsPolicyBreakdownMetadataInput holds metadata for a cost breakdown.
type FinopsPolicyBreakdownMetadataInput struct {
	Region string `json:"region"`
}

// FinopsPolicyCostComponentInput represents a cost component within a breakdown.
type FinopsPolicyCostComponentInput struct {
	Name            string `json:"name"`
	Unit            string `json:"unit"`
	Price           string `json:"price"`
	HourlyCost      string `json:"hourlyCost"`
	MonthlyCost     string `json:"monthlyCost"`
	HourlyQuantity  string `json:"hourlyQuantity"`
	MonthlyQuantity string `json:"monthlyQuantity"`
	UsageBased      bool   `json:"usageBased"`
}

// BreakdownInput represents a cost breakdown for a project.
type BreakdownInput struct {
	TotalHourlyCost             string                   `json:"totalHourlyCost"`
	TotalMonthlyCost            string                   `json:"totalMonthlyCost"`
	TotalMonthlyUsageCost       string                   `json:"totalMonthlyUsageCost,omitempty"`
	TotalMonthlyCarbonGramsCo2e string                   `json:"totalMonthlyCarbonGramsCo2e,omitempty"`
	Resources                   []map[string]interface{} `json:"resources,omitempty"`
}

// BudgetResultInput represents a budget evaluation result.
type BudgetResultInput struct {
	BudgetID             string              `json:"budgetId"`
	Tags                 []BudgetTagInput    `json:"tags"`
	StartDate            string              `json:"startDate"`
	EndDate              string              `json:"endDate"`
	Amount               string              `json:"amount"`
	CurrentCost          string              `json:"currentCost"`
	CustomOverrunMessage string              `json:"customOverrunMessage,omitempty"`
}

// BudgetTagInput represents a budget tag key-value pair.
type BudgetTagInput struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// GuardrailResultInput represents a guardrail evaluation result.
type GuardrailResultInput struct {
	GuardrailID            string   `json:"guardrailId"`
	Triggered              bool     `json:"triggered"`
	PRComment              bool     `json:"prComment"`
	BlockPR                bool     `json:"blockPr"`
	TriggeringProjectNames []string `json:"triggeringProjectNames"`
	Increase               string   `json:"increase"`
	PercentIncrease        string   `json:"percentIncrease"`
	TotalMonthlyCost       string   `json:"totalMonthlyCost"`
}

// AddRunResult is the response from the addRun mutation.
type AddRunResult struct {
	ID       string `json:"id"`
	CloudURL string `json:"cloudUrl"`
}

// PullRequestStatus represents the status of a pull request.
type PullRequestStatus string

const (
	PullRequestStatusOpen   PullRequestStatus = "OPEN"
	PullRequestStatusMerged PullRequestStatus = "MERGED"
	PullRequestStatusClosed PullRequestStatus = "CLOSED"
)

type Client interface {
	RunParameters(ctx context.Context, repoURL, branchName string) (RunParameters, error)
	AddRun(ctx context.Context, input RunInput) (AddRunResult, error)
	UpdatePullRequestStatus(ctx context.Context, prURL string, status PullRequestStatus) error
}

var (
	_ Client = (*client)(nil)
)

type client struct {
	client *http.Client
	config *Config
}

func (c *client) RunParameters(ctx context.Context, repoURL, branchName string) (RunParameters, error) {
	const query = `query RunParameters($repoUrl: String, $branchName: String) {
  runParameters(repoUrl: $repoUrl, branchName: $branchName) {
    organizationId
    organizationSlug
    cloudEnabled
    repositoryId
    repositoryName
    usageDefaults
    productionFilters
    tagPolicies
    finopsPolicies
    guardrails
    budgets
  }
}`

	type response struct {
		RunParameters RunParameters `json:"runParameters"`
	}

	variables := map[string]interface{}{}
	if repoURL != "" {
		variables["repoUrl"] = repoURL
	}
	if branchName != "" {
		variables["branchName"] = branchName
	}

	r, err := graphql.Query[response](ctx, c.client, fmt.Sprintf("%s/graphql", c.config.Endpoint), query, variables)
	if err != nil {
		return RunParameters{}, err
	}

	if len(r.Errors) > 0 {
		var errs []string
		for _, e := range r.Errors {
			errs = append(errs, e.Message)
		}
		return r.Data.RunParameters, errors.New(strings.Join(errs, ";"))
	}
	return r.Data.RunParameters, nil
}

func (c *client) AddRun(ctx context.Context, input RunInput) (AddRunResult, error) {
	const query = `mutation AddRun($run: RunInput!) {
  addRun(run: $run) {
    id
    cloudUrl
  }
}`

	type response struct {
		AddRun AddRunResult `json:"addRun"`
	}

	variables := map[string]interface{}{
		"run": input,
	}

	r, err := graphql.Query[response](ctx, c.client, fmt.Sprintf("%s/graphql", c.config.Endpoint), query, variables)
	if err != nil {
		return AddRunResult{}, err
	}

	if len(r.Errors) > 0 {
		var errs []string
		for _, e := range r.Errors {
			errs = append(errs, e.Message)
		}
		return r.Data.AddRun, errors.New(strings.Join(errs, "; "))
	}
	return r.Data.AddRun, nil
}

func (c *client) UpdatePullRequestStatus(ctx context.Context, prURL string, status PullRequestStatus) error {
	const query = `mutation UpdatePullRequestStatus($url: String!, $status: PullRequestStatus!) {
  updatePullRequestStatus(url: $url, status: $status)
}`

	type response struct {
		UpdatePullRequestStatus bool `json:"updatePullRequestStatus"`
	}

	variables := map[string]interface{}{
		"url":    prURL,
		"status": status,
	}

	r, err := graphql.Query[response](ctx, c.client, fmt.Sprintf("%s/graphql", c.config.Endpoint), query, variables)
	if err != nil {
		return err
	}

	if len(r.Errors) > 0 {
		var errs []string
		for _, e := range r.Errors {
			errs = append(errs, e.Message)
		}
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}
