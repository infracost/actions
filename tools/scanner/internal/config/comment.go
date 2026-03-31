package config

import (
	pkgscanner "github.com/infracost/cli/pkg/scanner"
	goprotoevent "github.com/infracost/go-proto/pkg/event"
	"github.com/infracost/go-proto/pkg/rat"
	"github.com/infracost/proto/gen/go/infracost/parser/event"
	"github.com/infracost/proto/gen/go/infracost/provider"
	"github.com/infracost/vcs/pkg/vcs/comment"
)

// buildCommentData converts base and head scan results into the comment.Data
// format required by the VCS library to generate a PR comment.
func buildCommentData(
	baseResult *DirectoryResult,
	headResult *DirectoryResult,
	guardrailResults []goprotoevent.GuardrailResult,
	previousGuardrailResults []goprotoevent.GuardrailResult,
	finopsPolicySettings []*event.FinopsPolicySettings,
	usageAPIEnabled bool,
	currency string,
	repoURL string,
	commitSHA string,
	branch string,
	orgSlug string,
	repoID string,
) comment.Data {
	// Match projects by name between base and head.
	baseByName := make(map[string]*pkgscanner.ProjectResult, len(baseResult.Projects))
	for i := range baseResult.Projects {
		baseByName[baseResult.Projects[i].Name] = &baseResult.Projects[i]
	}

	headByName := make(map[string]*pkgscanner.ProjectResult, len(headResult.Projects))
	for i := range headResult.Projects {
		headByName[headResult.Projects[i].Name] = &headResult.Projects[i]
	}

	// Collect all project names preserving head order, then appending base-only.
	seen := make(map[string]bool)
	var projectNames []string
	for _, p := range headResult.Projects {
		projectNames = append(projectNames, p.Name)
		seen[p.Name] = true
	}
	for _, p := range baseResult.Projects {
		if !seen[p.Name] {
			projectNames = append(projectNames, p.Name)
		}
	}

	projects := make([]comment.ProjectResult, 0, len(projectNames))
	var summary comment.ResourceSummary
	var allFinops, allPrevFinops []*provider.FinopsPolicyResult
	var allTagging, allPrevTagging []goprotoevent.TaggingPolicyResult
	totalMonthlyCost := rat.Zero
	pastTotalMonthlyCost := rat.Zero

	for _, name := range projectNames {
		head := headByName[name]
		base := baseByName[name]

		pr := comment.ProjectResult{
			Name: name,
		}

		if head != nil {
			pr.Breakdown = buildCostBreakdown(head.Resources)
			pr.TotalMonthlyCost = head.TotalMonthlyCost
			pr.TotalMonthlyUsageCost = totalMonthlyUsageCostFromResources(head.Resources)
			pr.Resources = head.Resources
			pr.Diagnostics = head.Diagnostics

			if head.Config != nil {
				pr.Workspace = head.Config.Terraform.Workspace
			}

			totalMonthlyCost = totalMonthlyCost.Add(orZero(head.TotalMonthlyCost))

			// Aggregate resource summary from head resources.
			for _, r := range head.Resources {
				aggregateResourceSummary(&summary, r)
			}

			allFinops = append(allFinops, head.FinopsResults...)
			allTagging = append(allTagging, head.TagPolicyResults...)
		}

		if base != nil {
			pr.PastBreakdown = buildCostBreakdown(base.Resources)
			pr.PastTotalMonthlyCost = base.TotalMonthlyCost
			pr.PastTotalMonthlyUsageCost = totalMonthlyUsageCostFromResources(base.Resources)
			pr.PastDiagnostics = base.Diagnostics

			pastTotalMonthlyCost = pastTotalMonthlyCost.Add(orZero(base.TotalMonthlyCost))

			allPrevFinops = append(allPrevFinops, base.FinopsResults...)
			allPrevTagging = append(allPrevTagging, base.TagPolicyResults...)
		}

		// Compute diff breakdown.
		if pr.Breakdown != nil && pr.PastBreakdown != nil {
			pr.DiffBreakdown = diffCostBreakdown(pr.PastBreakdown, pr.Breakdown)
		} else if pr.Breakdown != nil {
			pr.DiffBreakdown = pr.Breakdown
		}

		projects = append(projects, pr)
	}

	// Compute carbon diff across all projects.
	headCarbon := totalMonthlyCarbonFromProjects(headResult.Projects)
	baseCarbon := totalMonthlyCarbonFromProjects(baseResult.Projects)
	var diffCarbon *rat.Rat
	if headCarbon != nil || baseCarbon != nil {
		diffCarbon = orZero(headCarbon).Sub(orZero(baseCarbon))
	}

	// Build slug-to-group map from policy settings to split results.
	policyGroupBySlug := make(map[string]event.FinopsPolicySettings_Group, len(finopsPolicySettings))
	for _, s := range finopsPolicySettings {
		policyGroupBySlug[s.Slug] = s.Group
	}

	finopsPolicies, securityPolicies := splitPolicyResults(allFinops, policyGroupBySlug)
	prevFinopsPolicies, prevSecurityPolicies := splitPolicyResults(allPrevFinops, policyGroupBySlug)

	return comment.Data{
		EnableEnvironmentalMetrics:      true,
		UsedUsageFile:                   headResult.EstimatedUsageCounts != nil,
		UsageAPIEnabled:                 usageAPIEnabled,
		OrgSlug:                         orgSlug,
		RepoID:                          repoID,
		BaseBranchName:                  branch,
		Currency:                        currency,
		TotalMonthlyCost:                totalMonthlyCost,
		PastTotalMonthlyCost:            pastTotalMonthlyCost,
		DiffTotalMonthlyCarbonGramsCo2e: diffCarbon,
		CloudURL:                        "", // TODO: We need to implement addRun first.
		RepoURL:                         repoURL,
		CommitSHA:                       commitSHA,
		Summary:                         summary,
		Projects:                        projects,
		FinOpsPolicyResults:             finopsPolicies,
		PreviousFinOpsPolicyResults:     prevFinopsPolicies,
		SecurityPolicyResults:           securityPolicies,
		PreviousSecurityPolicyResults:   prevSecurityPolicies,
		TaggingPolicyResults:            allTagging,
		PreviousTaggingPolicyResults:    allPrevTagging,
		GuardrailResults:                guardrailResults,
		PreviousGuardrailResults:        previousGuardrailResults,
	}
}

func splitPolicyResults(finops []*provider.FinopsPolicyResult, slugs map[string]event.FinopsPolicySettings_Group) ([]*provider.FinopsPolicyResult, []*provider.FinopsPolicyResult) {
	var securityPolicies []*provider.FinopsPolicyResult
	var finopsPolicies []*provider.FinopsPolicyResult
	for _, finop := range finops {
		switch slugs[finop.PolicySlug] {
		case event.FinopsPolicySettings_CLOUD_SECURITY:
			securityPolicies = append(securityPolicies, finop)
		default:
			// default to finops policies if we get something unspecified
			finopsPolicies = append(finopsPolicies, finop)
		}
	}
	return finopsPolicies, securityPolicies
}

func orZero(r *rat.Rat) *rat.Rat {
	if r == nil {
		return rat.Zero
	}
	return r
}

// buildCostBreakdown converts proto resources into a CostBreakdown.
func buildCostBreakdown(resources []*provider.Resource) *comment.CostBreakdown {
	breakdown := &comment.CostBreakdown{
		TotalMonthlyCost: rat.Zero,
	}
	for _, r := range resources {
		br := convertResource(r)
		breakdown.Resources = append(breakdown.Resources, br)
		breakdown.TotalMonthlyCost = breakdown.TotalMonthlyCost.Add(orZero(br.MonthlyCost))
	}
	return breakdown
}

func convertResource(r *provider.Resource) comment.BreakdownResource {
	br := comment.BreakdownResource{
		Name:        r.Name,
		MonthlyCost: rat.Zero,
	}

	if r.Costs != nil {
		for _, c := range r.Costs.Components {
			cc := convertCostComponent(c)
			br.CostComponents = append(br.CostComponents, cc)
			br.MonthlyCost = br.MonthlyCost.Add(orZero(cc.MonthlyCost))
		}
	}

	for _, child := range r.ChildResources {
		sub := convertResource(child)
		br.SubResources = append(br.SubResources, sub)
		br.MonthlyCost = br.MonthlyCost.Add(orZero(sub.MonthlyCost))
	}

	return br
}

func convertCostComponent(c *provider.CostComponent) comment.BreakdownCostComponent {
	cc := comment.BreakdownCostComponent{
		Name:       c.Name,
		Unit:       c.Unit,
		UsageBased: c.UsageBased,
	}

	if c.PeriodPrice != nil && c.PeriodPrice.Price != nil && c.Quantity != nil {
		price := pkgscanner.ApplyDiscount(rat.FromProto(c.PeriodPrice.Price), rat.FromProto(c.DiscountRate))
		_, monthlyQty := pkgscanner.ConvertQuantityByPeriod(rat.FromProto(c.Quantity), c.PeriodPrice.Period)

		cc.Price = price
		cc.MonthlyQuantity = monthlyQty
		cc.MonthlyCost = monthlyQty.Mul(price)
	}

	return cc
}

// diffCostBreakdown computes the difference between two cost breakdowns.
func diffCostBreakdown(past, current *comment.CostBreakdown) *comment.CostBreakdown {
	if past == nil {
		return current
	}
	if current == nil {
		return nil
	}

	diff := &comment.CostBreakdown{
		TotalMonthlyCost: current.TotalMonthlyCost.Sub(past.TotalMonthlyCost),
	}

	// Build map of past resources by name for matching.
	pastByName := make(map[string]*comment.BreakdownResource, len(past.Resources))
	for i := range past.Resources {
		pastByName[past.Resources[i].Name] = &past.Resources[i]
	}

	for _, r := range current.Resources {
		if pastR, ok := pastByName[r.Name]; ok {
			costDiff := orZero(r.MonthlyCost).Sub(orZero(pastR.MonthlyCost))
			if !costDiff.IsZero() {
				diff.Resources = append(diff.Resources, comment.BreakdownResource{
					Name:        r.Name,
					MonthlyCost: costDiff,
				})
			}
			delete(pastByName, r.Name)
		} else {
			// New resource.
			diff.Resources = append(diff.Resources, r)
		}
	}

	// Remaining past resources were removed.
	for _, r := range pastByName {
		diff.Resources = append(diff.Resources, comment.BreakdownResource{
			Name:        r.Name,
			MonthlyCost: orZero(r.MonthlyCost).Mul(rat.New(-1)),
		})
	}

	return diff
}

// totalMonthlyUsageCostFromResources sums only usage-based cost components
// across all resources (including children).
func totalMonthlyUsageCostFromResources(resources []*provider.Resource) *rat.Rat {
	total := rat.Zero
	for _, r := range resources {
		total = total.Add(resourceMonthlyUsageCost(r))
	}
	return total
}

func resourceMonthlyUsageCost(r *provider.Resource) *rat.Rat {
	cost := rat.Zero
	if r.Costs != nil {
		for _, c := range r.Costs.Components {
			if c.UsageBased {
				cc := convertCostComponent(c)
				if cc.MonthlyCost != nil && cc.MonthlyCost.GreaterThan(rat.Zero) {
					cost = cost.Add(cc.MonthlyCost)
				}
			}
		}
	}
	for _, child := range r.ChildResources {
		cost = cost.Add(resourceMonthlyUsageCost(child))
	}
	return cost
}

// totalMonthlyCarbonFromProjects computes the total monthly carbon emissions
// across all projects by summing environmental metrics from cost components.
func totalMonthlyCarbonFromProjects(projects []pkgscanner.ProjectResult) *rat.Rat {
	total := rat.Zero
	for _, p := range projects {
		for _, r := range p.Resources {
			total = total.Add(resourceMonthlyCarbonGramsCo2e(r))
		}
	}
	return total
}

func resourceMonthlyCarbonGramsCo2e(r *provider.Resource) *rat.Rat {
	total := rat.Zero
	if r.Costs != nil {
		for _, c := range r.Costs.Components {
			total = total.Add(componentMonthlyCarbonGramsCo2e(c))
		}
	}
	for _, child := range r.ChildResources {
		total = total.Add(resourceMonthlyCarbonGramsCo2e(child))
	}
	return total
}

func componentMonthlyCarbonGramsCo2e(c *provider.CostComponent) *rat.Rat {
	if c.EnvironmentalMetrics == nil || c.Quantity == nil || c.EnvironmentalMetrics.CarbonGramsCo2E == nil {
		return rat.Zero
	}
	_, monthlyQty := pkgscanner.ConvertQuantityByPeriod(rat.FromProto(c.Quantity), c.EnvironmentalMetrics.Period)
	return monthlyQty.Mul(rat.FromProto(c.EnvironmentalMetrics.CarbonGramsCo2E))
}

// aggregateResourceSummary counts resources into the summary.
func aggregateResourceSummary(summary *comment.ResourceSummary, r *provider.Resource) {
	summary.TotalDetectedResources++
	if r.IsSupported {
		summary.TotalSupportedResources++
	} else if r.IsProviderSupported {
		summary.TotalUnsupportedResources++
		if summary.UnsupportedResourceCounts == nil {
			summary.UnsupportedResourceCounts = make(map[string]int)
		}
		summary.UnsupportedResourceCounts[r.Type]++
	}
	if r.IsFree {
		summary.TotalNoPriceResources++
	}

	for _, child := range r.ChildResources {
		aggregateResourceSummary(summary, child)
	}
}
