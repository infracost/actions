package config

import (
	pkgscanner "github.com/infracost/cli/pkg/scanner"
	goprotoevent "github.com/infracost/go-proto/pkg/event"
	"github.com/infracost/go-proto/pkg/rat"
	"github.com/infracost/proto/gen/go/infracost/parser/event"
	"github.com/infracost/proto/gen/go/infracost/provider"
	"github.com/infracost/vcs/pkg/vcs/comment"
)

// CommentDataOptions contains the parameters needed to build a PR comment.
type CommentDataOptions struct {
	BaseResult               *DirectoryResult
	HeadResult               *DirectoryResult
	GuardrailResults         []goprotoevent.GuardrailResult
	PreviousGuardrailResults []goprotoevent.GuardrailResult
	BudgetResults            []goprotoevent.BudgetResult
	FinopsPolicySettings     []*event.FinopsPolicySettings
	UsageAPIEnabled          bool
	Currency                 string
	RepoURL                  string
	CommitSHA                string
	Branch                   string
	OrgSlug                  string
	RepoID                   string
	RepoName                 string
}

// BuildCommentData converts base and head scan results into the comment.Data
// format required by the VCS library to generate a PR comment.
func BuildCommentData(opts CommentDataOptions) comment.Data {
	baseResult := opts.BaseResult
	headResult := opts.HeadResult
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
	policyGroupBySlug := make(map[string]event.FinopsPolicySettings_Group, len(opts.FinopsPolicySettings))
	for _, s := range opts.FinopsPolicySettings {
		policyGroupBySlug[s.Slug] = s.Group
	}

	finopsPolicies, securityPolicies := splitPolicyResults(allFinops, policyGroupBySlug)
	prevFinopsPolicies, prevSecurityPolicies := splitPolicyResults(allPrevFinops, policyGroupBySlug)

	return comment.Data{
		EnableEnvironmentalMetrics:      true,
		UsedUsageFile:                   headResult.EstimatedUsageCounts != nil,
		UsageAPIEnabled:                 opts.UsageAPIEnabled,
		OrgSlug:                         opts.OrgSlug,
		RepoID:                          opts.RepoID,
		BaseBranchName:                  opts.Branch,
		Currency:                        opts.Currency,
		TotalMonthlyCost:                totalMonthlyCost,
		PastTotalMonthlyCost:            pastTotalMonthlyCost,
		DiffTotalMonthlyCarbonGramsCo2e: diffCarbon,
		CloudURL:                        "", // TODO: We need to implement addRun first.
		RepoURL:                         opts.RepoURL,
		CommitSHA:                       opts.CommitSHA,
		Summary:                         summary,
		Projects:                        projects,
		FinOpsPolicyResults:             finopsPolicies,
		PreviousFinOpsPolicyResults:     prevFinopsPolicies,
		SecurityPolicyResults:           securityPolicies,
		PreviousSecurityPolicyResults:   prevSecurityPolicies,
		TaggingPolicyResults:            allTagging,
		PreviousTaggingPolicyResults:    allPrevTagging,
		GuardrailResults:                opts.GuardrailResults,
		PreviousGuardrailResults:        opts.PreviousGuardrailResults,
		BudgetResults:                   convertBudgetResults(opts.BudgetResults),
		RepoName:                        opts.RepoName,
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

// EvaluateBudgets pre-computes resource cost info from head projects and
// evaluates budgets using go-proto.
func EvaluateBudgets(budgets []*event.Budget, projects []pkgscanner.ProjectResult) []goprotoevent.BudgetResult {
	if len(budgets) == 0 {
		return nil
	}

	var costInfos []goprotoevent.ResourceCostInfo
	for _, p := range projects {
		costInfos = append(costInfos, pkgscanner.ResourceCostInfos(p.Resources)...)
	}

	return goprotoevent.Budgets(budgets).Evaluate(costInfos)
}

func convertBudgetResults(results []goprotoevent.BudgetResult) []comment.BudgetResult {
	out := make([]comment.BudgetResult, 0, len(results))
	for _, r := range results {
		tags := make([]comment.BudgetTag, 0, len(r.Tags))
		for _, t := range r.Tags {
			tags = append(tags, comment.BudgetTag{Key: t.Key, Value: t.Value})
		}
		out = append(out, comment.BudgetResult{
			Tags:                 tags,
			StartDate:            r.StartDate,
			EndDate:              r.EndDate,
			Amount:               r.Amount,
			CurrentCost:          r.CurrentCost,
			CustomOverrunMessage: r.CustomOverrunMessage,
		})
	}
	return out
}

func convertResource(r *provider.Resource) comment.BreakdownResource {
	br := comment.BreakdownResource{
		Name:        r.Name,
		MonthlyCost: rat.Zero,
	}

	if r.Tagging != nil && len(r.Tagging.Tags) > 0 {
		br.Tags = make(map[string]string, len(r.Tagging.Tags))
		for _, t := range r.Tagging.Tags {
			br.Tags[t.Key] = t.Value
		}
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
// It mirrors the runner's pricing/schema CalculateDiff: a resource is included
// in the diff when any cost component, sub-resource, or resource-level cost
// changed (including resources that were added or removed). A resource whose
// total monthly cost happens to be unchanged but whose components shifted is
// still considered changed, matching the dashboard's hasDiff criterion.
func diffCostBreakdown(past, current *comment.CostBreakdown) *comment.CostBreakdown {
	if past == nil {
		return current
	}
	if current == nil {
		return nil
	}

	diff := &comment.CostBreakdown{
		TotalMonthlyCost: orZero(current.TotalMonthlyCost).Sub(orZero(past.TotalMonthlyCost)),
	}

	pastByName := indexBreakdownResources(past.Resources)
	currentByName := indexBreakdownResources(current.Resources)

	seen := make(map[string]bool, len(past.Resources)+len(current.Resources))
	// Walk past first to keep removed/changed resources in past order, then
	// append any current-only (added) resources in current order.
	for _, r := range past.Resources {
		if seen[r.Name] {
			continue
		}
		seen[r.Name] = true
		if changed, dr := diffResource(pastByName[r.Name], currentByName[r.Name]); changed {
			diff.Resources = append(diff.Resources, dr)
		}
	}
	for _, r := range current.Resources {
		if seen[r.Name] {
			continue
		}
		seen[r.Name] = true
		if changed, dr := diffResource(pastByName[r.Name], currentByName[r.Name]); changed {
			diff.Resources = append(diff.Resources, dr)
		}
	}

	return diff
}

func indexBreakdownResources(resources []comment.BreakdownResource) map[string]*comment.BreakdownResource {
	m := make(map[string]*comment.BreakdownResource, len(resources))
	for i := range resources {
		m[resources[i].Name] = &resources[i]
	}
	return m
}

// diffResource recursively compares a past and current breakdown resource and
// returns whether they differ along with a synthesised "diff" resource whose
// fields hold the deltas (old=0 for adds, current=0 for removes).
func diffResource(past, current *comment.BreakdownResource) (bool, comment.BreakdownResource) {
	pastSafe := past
	if pastSafe == nil {
		pastSafe = &comment.BreakdownResource{}
	}
	currentSafe := current
	if currentSafe == nil {
		currentSafe = &comment.BreakdownResource{}
	}

	name := currentSafe.Name
	if name == "" {
		name = pastSafe.Name
	}

	monthlyCostDiff := orZero(currentSafe.MonthlyCost).Sub(orZero(pastSafe.MonthlyCost))
	out := comment.BreakdownResource{
		Name:        name,
		MonthlyCost: monthlyCostDiff,
	}

	changed := past == nil || current == nil || !monthlyCostDiff.IsZero()

	pastSubs := indexBreakdownResources(pastSafe.SubResources)
	currentSubs := indexBreakdownResources(currentSafe.SubResources)
	seenSub := make(map[string]bool, len(pastSafe.SubResources)+len(currentSafe.SubResources))
	for _, sr := range pastSafe.SubResources {
		if seenSub[sr.Name] {
			continue
		}
		seenSub[sr.Name] = true
		if subChanged, sd := diffResource(pastSubs[sr.Name], currentSubs[sr.Name]); subChanged {
			out.SubResources = append(out.SubResources, sd)
			changed = true
		}
	}
	for _, sr := range currentSafe.SubResources {
		if seenSub[sr.Name] {
			continue
		}
		seenSub[sr.Name] = true
		if subChanged, sd := diffResource(pastSubs[sr.Name], currentSubs[sr.Name]); subChanged {
			out.SubResources = append(out.SubResources, sd)
			changed = true
		}
	}

	if ccChanged, ccDiff := diffCostComponents(pastSafe.CostComponents, currentSafe.CostComponents); ccChanged {
		out.CostComponents = ccDiff
		changed = true
	}

	return changed, out
}

func diffCostComponents(past, current []comment.BreakdownCostComponent) (bool, []comment.BreakdownCostComponent) {
	currentByName := make(map[string]*comment.BreakdownCostComponent, len(current))
	for i := range current {
		currentByName[current[i].Name] = &current[i]
	}

	var out []comment.BreakdownCostComponent
	changed := false
	seen := make(map[string]bool, len(past))
	for i := range past {
		seen[past[i].Name] = true
		if cChanged, cd := diffCostComponent(&past[i], currentByName[past[i].Name]); cChanged {
			out = append(out, cd)
			changed = true
		}
	}
	for i := range current {
		if seen[current[i].Name] {
			continue
		}
		if cChanged, cd := diffCostComponent(nil, &current[i]); cChanged {
			out = append(out, cd)
			changed = true
		}
	}
	return changed, out
}

func diffCostComponent(past, current *comment.BreakdownCostComponent) (bool, comment.BreakdownCostComponent) {
	pastSafe := past
	if pastSafe == nil {
		pastSafe = &comment.BreakdownCostComponent{}
	}
	currentSafe := current
	if currentSafe == nil {
		currentSafe = &comment.BreakdownCostComponent{}
	}

	base := current
	if base == nil {
		base = past
	}

	priceDiff := orZero(currentSafe.Price).Sub(orZero(pastSafe.Price))
	qtyDiff := orZero(currentSafe.MonthlyQuantity).Sub(orZero(pastSafe.MonthlyQuantity))
	costDiff := orZero(currentSafe.MonthlyCost).Sub(orZero(pastSafe.MonthlyCost))

	out := comment.BreakdownCostComponent{
		Name:            base.Name,
		Unit:            base.Unit,
		UsageBased:      base.UsageBased,
		Price:           priceDiff,
		MonthlyQuantity: qtyDiff,
		MonthlyCost:     costDiff,
	}

	changed := !priceDiff.IsZero() || !qtyDiff.IsZero() || !costDiff.IsZero()
	return changed, out
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
