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
	RepositoryID     string `json:"repositoryId"`
	RepositoryName   string `json:"repositoryName"`

	UsageDefaults     json.RawMessage   `json:"usageDefaults"`
	ProductionFilters []json.RawMessage `json:"productionFilters"`
	TagPolicies       []json.RawMessage `json:"tagPolicies"`
	FinopsPolicies    []json.RawMessage `json:"finopsPolicies"`
	Guardrails        []json.RawMessage `json:"guardrails"`
}

// RunInput is the input to the addRun mutation.
type RunInput struct {
	ProjectResults          []ProjectResultInput   `json:"projectResults"`
	Currency                string                 `json:"currency,omitempty"`
	TimeGenerated           string                 `json:"timeGenerated,omitempty"`
	Metadata                map[string]interface{} `json:"metadata,omitempty"`
	GuardrailResults        []GuardrailResultInput `json:"guardrailResults,omitempty"`
	PoliciesAlreadyEvaluated bool                  `json:"policiesAlreadyEvaluated,omitempty"`
	ClientPostedComment     *bool                  `json:"clientPostedComment,omitempty"`
}

// ProjectResultInput represents a single project's cost data for addRun.
type ProjectResultInput struct {
	ProjectName        string                           `json:"projectName"`
	Breakdown          BreakdownInput                   `json:"breakdown"`
	PastBreakdown      *BreakdownInput                  `json:"pastBreakdown,omitempty"`
	Diff               *BreakdownInput                  `json:"diff,omitempty"`
	Metadata           map[string]interface{}            `json:"metadata,omitempty"`
	TagPolicyResults   []map[string]interface{}          `json:"tagPolicyResults,omitempty"`
	FinopsPolicyResults []map[string]interface{}         `json:"finopsPolicyResults,omitempty"`
}

// BreakdownInput represents a cost breakdown for a project.
type BreakdownInput struct {
	TotalHourlyCost             string                   `json:"totalHourlyCost"`
	TotalMonthlyCost            string                   `json:"totalMonthlyCost"`
	TotalMonthlyUsageCost       string                   `json:"totalMonthlyUsageCost,omitempty"`
	TotalMonthlyCarbonGramsCo2e string                   `json:"totalMonthlyCarbonGramsCo2e,omitempty"`
	Resources                   []map[string]interface{} `json:"resources,omitempty"`
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
    repositoryId
    repositoryName
    usageDefaults
    productionFilters
    tagPolicies
    finopsPolicies
    guardrails
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
