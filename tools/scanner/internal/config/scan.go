package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/infracost/actions/tools/scanner/internal/api"
	"github.com/infracost/actions/tools/scanner/internal/api/dashboard"
	"github.com/infracost/actions/tools/scanner/internal/trace"
	pkgscanner "github.com/infracost/cli/pkg/scanner"
	repoconfig "github.com/infracost/config"
	gorat "github.com/infracost/go-proto/pkg/rat"
	"github.com/infracost/proto/gen/go/infracost/parser/event"
	"github.com/infracost/vcs/pkg/vcs"
	"google.golang.org/protobuf/encoding/protojson"
)

// DirectoryResult holds the outputs for scanning an entire directory.
type DirectoryResult struct {
	Projects         []pkgscanner.ProjectResult
	TotalMonthlyCost *gorat.Rat
	Currency         string

	// EstimatedUsageCounts tracks usage parameters with non-zero values.
	// A nil map means no usage file was loaded.
	EstimatedUsageCounts map[string]int
	// UnestimatedUsageCounts tracks usage parameters with zero/empty values.
	UnestimatedUsageCounts map[string]int
}

// parsedRunParameters holds the unmarshalled run parameters from the dashboard API.
type parsedRunParameters struct {
	OrganizationID   string
	OrganizationSlug string
	RepositoryID     string
	RepositoryName   string

	UsageDefaults     *event.UsageDefaults
	ProductionFilters []*event.ProductionFilter
	TagPolicies       []*event.TagPolicy
	FinopsPolicies    []*event.FinopsPolicySettings
	Guardrails        []*event.Guardrail
}

func parseRunParameters(raw dashboard.RunParameters) (*parsedRunParameters, error) {
	parsed := &parsedRunParameters{
		OrganizationID:   raw.OrganizationID,
		OrganizationSlug: raw.OrganizationSlug,
		RepositoryID:     raw.RepositoryID,
		RepositoryName:   raw.RepositoryName,
	}

	parsed.UsageDefaults = new(event.UsageDefaults)
	if len(raw.UsageDefaults) > 0 {
		if err := protojson.Unmarshal(raw.UsageDefaults, parsed.UsageDefaults); err != nil {
			return nil, fmt.Errorf("failed to unmarshal usage defaults: %w", err)
		}
	}

	for _, f := range raw.ProductionFilters {
		filter := new(event.ProductionFilter)
		if err := protojson.Unmarshal(f, filter); err != nil {
			return nil, fmt.Errorf("failed to unmarshal production filter: %w", err)
		}
		parsed.ProductionFilters = append(parsed.ProductionFilters, filter)
	}

	for _, p := range raw.TagPolicies {
		policy := new(event.TagPolicy)
		if err := protojson.Unmarshal(p, policy); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tag policy: %w", err)
		}
		parsed.TagPolicies = append(parsed.TagPolicies, policy)
	}

	for _, p := range raw.FinopsPolicies {
		policy := new(event.FinopsPolicySettings)
		if err := protojson.Unmarshal(p, policy); err != nil {
			return nil, fmt.Errorf("failed to unmarshal FinOps policy: %w", err)
		}
		parsed.FinopsPolicies = append(parsed.FinopsPolicies, policy)
	}

	for _, g := range raw.Guardrails {
		guardrail := new(event.Guardrail)
		if err := protojson.Unmarshal(g, guardrail); err != nil {
			return nil, fmt.Errorf("failed to unmarshal guardrail: %w", err)
		}
		parsed.Guardrails = append(parsed.Guardrails, guardrail)
	}

	return parsed, nil
}

func (config *Config) Scan() error {
	ctx := context.Background()

	tokenSource := config.Auth.TokenFromCache(ctx)
	httpClient := api.Client(ctx, tokenSource, config.OrgID)

	dashboardClient := config.Dashboard.Client(httpClient)
	rawRunParams, err := dashboardClient.RunParameters(ctx, "", "")
	if err != nil {
		return fmt.Errorf("failed to fetch run parameters: %w", err)
	}

	runParams, err := parseRunParameters(rawRunParams)
	if err != nil {
		return fmt.Errorf("failed to parse run parameters: %w", err)
	}

	baseResult, err := config.scanDirectory(ctx, config.BasePath, runParams, nil)
	if err != nil {
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

	headResult, err := config.scanDirectory(ctx, config.HeadPath, runParams, previousAddresses)
	if err != nil {
		return fmt.Errorf("failed to scan head path: %w", err)
	}

	guardrailResults := pkgscanner.EvaluateGuardrails(runParams.Guardrails, baseResult.Projects, headResult.Projects)

	// Evaluate guardrails against the base branch to determine which were
	// already triggered before this PR — these are suppressed in the comment.
	previousGuardrailResults := pkgscanner.EvaluateGuardrails(runParams.Guardrails, nil, baseResult.Projects)

	usageAPIEnabled := runParams.UsageDefaults != nil && len(runParams.UsageDefaults.Resources) > 0

	data := buildCommentData(baseResult, headResult, guardrailResults, previousGuardrailResults, runParams.FinopsPolicies, usageAPIEnabled, headResult.Currency, config.RepoURL, config.CommitSHA, config.Branch, runParams.OrganizationSlug, runParams.RepositoryID)

	vcsClient, err := config.VCSClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create VCS client: %w", err)
	}

	body, err := vcsClient.GenerateComment(data)
	if err != nil {
		return fmt.Errorf("failed to generate comment: %w", err)
	}

	if _, err := vcsClient.PostComment(ctx, body, vcs.BehaviorUpdate); err != nil {
		return fmt.Errorf("failed to post comment: %w", err)
	}

	return nil
}

func (config *Config) scanDirectory(ctx context.Context, dir string, runParams *parsedRunParameters, previousAddresses map[string][]string) (*DirectoryResult, error) {
	absoluteDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path for %q: %w", dir, err)
	}

	var repoConfigOpts []repoconfig.GenerationOption
	if runParams.RepositoryName != "" {
		repoConfigOpts = append(repoConfigOpts, repoconfig.WithRepoName(runParams.RepositoryName))
	}

	repoConfig, err := pkgscanner.LoadOrGenerateRepositoryConfig(absoluteDir, repoConfigOpts...)
	if err != nil {
		return nil, fmt.Errorf("repository configuration error: %w", err)
	}

	if repoConfig.Currency == "" {
		repoConfig.Currency = "USD"
	}

	// Load repo-level usage defaults, then overlay the usage file if present.
	repoUsage := pkgscanner.LoadUsageDefaults(runParams.UsageDefaults, "")
	if repoConfig.UsageFilePath != "" {
		usagePath := filepath.Join(absoluteDir, repoConfig.UsageFilePath)
		if stat, err := os.Stat(usagePath); err == nil && !stat.IsDir() {
			f, err := os.Open(usagePath) // #nosec G304 -- user-specified usage file in their repo
			if err != nil {
				return nil, fmt.Errorf("failed to open usage file %q: %w", usagePath, err)
			}
			u, err := pkgscanner.LoadUsageData(f, repoUsage)
			_ = f.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to load usage data from %q: %w", usagePath, err)
			}
			repoUsage = u
		}
	}

	estimatedUsageCounts, unestimatedUsageCounts := pkgscanner.CountUsage(repoUsage)

	tokenSource := config.Auth.TokenFromCache(ctx)
	token, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve access token: %w", err)
	}

	cacheDir := filepath.Join(os.TempDir(), ".infracost", "cache")
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	var projects []pkgscanner.ProjectResult
	for _, project := range repoConfig.Projects {
		if config.Project != "" && project.Name != config.Project {
			continue
		}

		result, err := pkgscanner.ScanProject(ctx, &pkgscanner.ScanProjectOptions{
			RootDir:                   absoluteDir,
			CacheDir:                  cacheDir,
			RepoConfig:                repoConfig,
			Project:                   project,
			AccessToken:               token.AccessToken,
			BranchName:                config.Branch,
			RepositoryName:            runParams.RepositoryName,
			OrgID:                     runParams.OrganizationID,
			PricingEndpoint:           config.PricingEndpoint,
			Currency:                  repoConfig.Currency,
			TraceID:                   trace.ID,
			ProductionFilters:         runParams.ProductionFilters,
			FinopsPolicies:            runParams.FinopsPolicies,
			TagPolicies:               runParams.TagPolicies,
			UsageDefaults:             runParams.UsageDefaults,
			RepoUsage:                 repoUsage,
			PreviousResourceAddresses: previousAddresses[project.Name],
			Plugins:                   &config.Plugins,
			Logging:                   config.Logging,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to scan project %q: %w", project.Name, err)
		}
		projects = append(projects, *result)
	}

	totalMonthlyCost := gorat.Zero
	for _, p := range projects {
		if p.TotalMonthlyCost != nil {
			totalMonthlyCost = totalMonthlyCost.Add(p.TotalMonthlyCost)
		}
	}

	return &DirectoryResult{
		Projects:               projects,
		TotalMonthlyCost:       totalMonthlyCost,
		Currency:               repoConfig.Currency,
		EstimatedUsageCounts:   estimatedUsageCounts,
		UnestimatedUsageCounts: unestimatedUsageCounts,
	}, nil
}