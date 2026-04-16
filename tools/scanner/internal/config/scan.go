package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/infracost/actions/tools/scanner/internal/api/dashboard"
	"github.com/infracost/actions/tools/scanner/internal/trace"
	pkgscanner "github.com/infracost/cli/pkg/scanner"
	repoconfig "github.com/infracost/config"
	gorat "github.com/infracost/go-proto/pkg/rat"
	"github.com/infracost/proto/gen/go/infracost/parser/event"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	pj = protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
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

	// Config metadata — surfaced in the addRun mutation metadata.
	HasConfigFile          bool
	UsageFilePath          string
	ConfigFileHasUsageFile bool
}

// ParsedRunParameters holds the unmarshalled run parameters from the dashboard API.
type ParsedRunParameters struct {
	OrganizationID   string
	OrganizationSlug string
	RepositoryID     string
	RepositoryName   string

	UsageDefaults     *event.UsageDefaults
	ProductionFilters []*event.ProductionFilter
	TagPolicies       []*event.TagPolicy
	FinopsPolicies    []*event.FinopsPolicySettings
	Guardrails        []*event.Guardrail
	Budgets           []*event.Budget
}

func ParseRunParameters(raw dashboard.RunParameters) (*ParsedRunParameters, error) {
	parsed := &ParsedRunParameters{
		OrganizationID:   raw.OrganizationID,
		OrganizationSlug: raw.OrganizationSlug,
		RepositoryID:     raw.RepositoryID,
		RepositoryName:   raw.RepositoryName,
	}

	parsed.UsageDefaults = new(event.UsageDefaults)
	if len(raw.UsageDefaults) > 0 {
		if err := pj.Unmarshal(raw.UsageDefaults, parsed.UsageDefaults); err != nil {
			return nil, fmt.Errorf("failed to unmarshal usage defaults: %w", err)
		}
	}

	for _, f := range raw.ProductionFilters {
		filter := new(event.ProductionFilter)
		if err := pj.Unmarshal(f, filter); err != nil {
			return nil, fmt.Errorf("failed to unmarshal production filter: %w", err)
		}
		parsed.ProductionFilters = append(parsed.ProductionFilters, filter)
	}

	for _, p := range raw.TagPolicies {
		policy := new(event.TagPolicy)
		if err := pj.Unmarshal(p, policy); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tag policy: %w", err)
		}
		parsed.TagPolicies = append(parsed.TagPolicies, policy)
	}

	for _, p := range raw.FinopsPolicies {
		policy := new(event.FinopsPolicySettings)
		if err := pj.Unmarshal(p, policy); err != nil {
			return nil, fmt.Errorf("failed to unmarshal FinOps policy: %w", err)
		}
		parsed.FinopsPolicies = append(parsed.FinopsPolicies, policy)
	}

	for _, g := range raw.Guardrails {
		guardrail := new(event.Guardrail)
		if err := pj.Unmarshal(g, guardrail); err != nil {
			return nil, fmt.Errorf("failed to unmarshal guardrail: %w", err)
		}
		parsed.Guardrails = append(parsed.Guardrails, guardrail)
	}

	for _, b := range raw.Budgets {
		budget := new(event.Budget)
		if err := pj.Unmarshal(b, budget); err != nil {
			return nil, fmt.Errorf("failed to unmarshal budget: %w", err)
		}
		parsed.Budgets = append(parsed.Budgets, budget)
	}

	return parsed, nil
}

func (config *Config) ScanDirectory(ctx context.Context, dir string, accessToken string, runParams *ParsedRunParameters, previousAddresses map[string][]string, projectFilter string, branch string) (*DirectoryResult, error) {
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

	cacheDir := filepath.Join(os.TempDir(), ".infracost", "cache")
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	var projects []pkgscanner.ProjectResult
	for _, project := range repoConfig.Projects {
		if projectFilter != "" && project.Name != projectFilter {
			continue
		}

		result, err := pkgscanner.ScanProject(ctx, &pkgscanner.ScanProjectOptions{
			RootDir:                   absoluteDir,
			CacheDir:                  cacheDir,
			RepoConfig:                repoConfig,
			Project:                   project,
			AccessToken:               accessToken,
			BranchName:                branch,
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

	// Detect whether the user has a config file (infracost.yml or template).
	hasConfigFile := fileExistsAt(filepath.Join(absoluteDir, pkgscanner.RepoConfigFilename)) ||
		fileExistsAt(filepath.Join(absoluteDir, pkgscanner.RepoConfigTemplateFilename))

	return &DirectoryResult{
		Projects:               projects,
		TotalMonthlyCost:       totalMonthlyCost,
		Currency:               repoConfig.Currency,
		EstimatedUsageCounts:   estimatedUsageCounts,
		UnestimatedUsageCounts: unestimatedUsageCounts,
		HasConfigFile:          hasConfigFile,
		UsageFilePath:          repoConfig.UsageFilePath,
		ConfigFileHasUsageFile: repoConfig.UsageFilePath != "",
	}, nil
}

func fileExistsAt(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !stat.IsDir()
}
