package commands

import (
	"context"

	"github.com/infracost/actions/tools/scanner/internal/api/events"
	"github.com/infracost/actions/tools/scanner/internal/config"
	pkgscanner "github.com/infracost/cli/pkg/scanner"
	goprotoevent "github.com/infracost/go-proto/pkg/event"
	"github.com/infracost/proto/gen/go/infracost/provider"
)

// trackRun fires an "infracost-run" event with resource counts, usage info,
// and (when a previous result is available) diff counts.
func trackRun(ctx context.Context, client events.Client, result *config.DirectoryResult, prev *config.DirectoryResult, runSeconds float64, outputFormat string) {
	var totalResources int
	var totalSupported int
	var totalNoPrice int
	var totalUnsupported int

	supportedCounts := make(map[string]int)
	unsupportedCounts := make(map[string]int)

	var supportedList []string
	var unsupportedList []string

	for _, p := range result.Projects {
		for _, r := range p.Resources {
			totalResources++

			switch {
			case !r.IsSupported:
				totalUnsupported++
				unsupportedCounts[r.Type]++
				unsupportedList = append(unsupportedList, r.Type)
			case r.IsFree:
				totalNoPrice++
			default:
				totalSupported++
				supportedCounts[r.Type]++
				supportedList = append(supportedList, r.Type)
			}
		}
	}

	extra := []interface{}{
		"runSeconds", runSeconds,
		"outputFormat", outputFormat,
		"totalResources", totalResources,
		"totalSupportedResources", totalSupported,
		"totalNoPriceResources", totalNoPrice,
		"totalUnsupportedResources", totalUnsupported,
		"supportedResourceCounts", supportedCounts,
		"supportedResourcesList", supportedList,
		"unsupportedResourceCounts", unsupportedCounts,
		"unsupportedResourcesList", unsupportedList,
	}

	hasUsageFile := result.EstimatedUsageCounts != nil
	extra = append(extra, "hasUsageFile", hasUsageFile)
	if hasUsageFile {
		var totalEstimated int
		var totalUnestimated int
		var estimatedList []string
		var unestimatedList []string

		for key, count := range result.EstimatedUsageCounts {
			totalEstimated += count
			for range count {
				estimatedList = append(estimatedList, key)
			}
		}
		for key, count := range result.UnestimatedUsageCounts {
			totalUnestimated += count
			for range count {
				unestimatedList = append(unestimatedList, key)
			}
		}

		extra = append(extra,
			"totalEstimatedUsages", totalEstimated,
			"totalUnestimatedUsages", totalUnestimated,
			"estimatedUsageCounts", result.EstimatedUsageCounts,
			"estimatedUsageList", estimatedList,
			"unestimatedUsageCounts", result.UnestimatedUsageCounts,
			"unestimatedUsageList", unestimatedList,
		)
	}

	if prev != nil {
		diff := computeRunDiff(result, prev)
		extra = append(extra,
			"newResourceCount", diff.newResources,
			"changedResourceCount", diff.changedResources,
			"newIssueCount", diff.newIssues,
			"existingIssueCount", diff.existingIssues,
		)
	}

	client.Push(ctx, "infracost-run", extra...)
}

// trackDiff compares head results against base results and fires a
// "cloud-issue-fixed" event for every policy violation that was present in
// base but is no longer present in head. Projects are matched by name.
func trackDiff(ctx context.Context, client events.Client, head *config.DirectoryResult, base *config.DirectoryResult) {
	if base == nil {
		return
	}

	prev := make(map[string]*pkgscanner.ProjectResult, len(base.Projects))
	for i := range base.Projects {
		prev[base.Projects[i].Name] = &base.Projects[i]
	}

	for i := range head.Projects {
		p := &head.Projects[i]
		old, ok := prev[p.Name]
		if !ok {
			continue
		}
		trackFinopsDiff(ctx, client, p, old.FinopsResults)
		trackTaggingDiff(ctx, client, p, old.TagPolicyResults)
	}
}

type runDiffCounts struct {
	newResources     int
	changedResources int
	newIssues        int
	existingIssues   int
}

func computeRunDiff(current, previous *config.DirectoryResult) runDiffCounts {
	var counts runDiffCounts

	prev := make(map[string]pkgscanner.ProjectResult, len(previous.Projects))
	for _, p := range previous.Projects {
		prev[p.Name] = p
	}

	for _, p := range current.Projects {
		old, ok := prev[p.Name]
		if !ok {
			counts.newResources += len(p.Resources)
			counts.newIssues += countIssues(p)
			continue
		}

		newRes, changedRes := countResourceDiff(p, old)
		counts.newResources += newRes
		counts.changedResources += changedRes

		newIss, existingIss := countIssueDiff(p, old)
		counts.newIssues += newIss
		counts.existingIssues += existingIss
	}

	return counts
}

func countResourceDiff(current, previous pkgscanner.ProjectResult) (newResources, changedResources int) {
	prevChecksums := make(map[string]string, len(previous.Resources))
	for _, r := range previous.Resources {
		prevChecksums[r.Name] = r.Metadata.GetDeepChecksum()
	}

	for _, r := range current.Resources {
		oldChecksum, ok := prevChecksums[r.Name]
		if !ok {
			newResources++
			continue
		}
		if r.Metadata.GetDeepChecksum() != "" && oldChecksum != "" && r.Metadata.GetDeepChecksum() != oldChecksum {
			changedResources++
		}
	}

	return newResources, changedResources
}

func countIssueDiff(current, previous pkgscanner.ProjectResult) (newIssues, existingIssues int) {
	prevIssues := make(map[string]struct{})

	for _, r := range previous.FinopsResults {
		for _, fr := range r.FailingResources {
			prevIssues[r.PolicySlug+"\x00"+fr.CauseAddress] = struct{}{}
		}
	}
	for _, r := range previous.TagPolicyResults {
		for _, fr := range r.FailingResources {
			prevIssues[r.TagPolicyID+"\x00"+fr.Address] = struct{}{}
		}
	}

	for _, r := range current.FinopsResults {
		for _, fr := range r.FailingResources {
			key := r.PolicySlug + "\x00" + fr.CauseAddress
			if _, ok := prevIssues[key]; ok {
				existingIssues++
			} else {
				newIssues++
			}
		}
	}
	for _, r := range current.TagPolicyResults {
		for _, fr := range r.FailingResources {
			key := r.TagPolicyID + "\x00" + fr.Address
			if _, ok := prevIssues[key]; ok {
				existingIssues++
			} else {
				newIssues++
			}
		}
	}

	return newIssues, existingIssues
}

func countIssues(p pkgscanner.ProjectResult) int {
	var total int
	for _, r := range p.FinopsResults {
		total += len(r.FailingResources)
	}
	for _, r := range p.TagPolicyResults {
		total += len(r.FailingResources)
	}
	return total
}

func trackFinopsDiff(ctx context.Context, client events.Client, p *pkgscanner.ProjectResult, other []*provider.FinopsPolicyResult) {
	current := make(map[string]map[string]struct{})
	for _, r := range p.FinopsResults {
		addrs := make(map[string]struct{}, len(r.FailingResources))
		for _, fr := range r.FailingResources {
			addrs[fr.CauseAddress] = struct{}{}
		}
		current[r.PolicySlug] = addrs
	}

	for _, prev := range other {
		for _, fr := range prev.FailingResources {
			if cur, ok := current[prev.PolicySlug]; ok {
				if _, still := cur[fr.CauseAddress]; still {
					continue
				}
			}
			client.Push(ctx, "cloud-issue-fixed",
				"policyId", prev.PolicyId,
				"policySlug", prev.PolicySlug,
				"type", "finops-policy",
				"projectName", p.Name,
				"resourceAddress", fr.CauseAddress,
				"pullRequestId", "",
				"autoFixPullRequest", false,
			)
		}
	}
}

func trackTaggingDiff(ctx context.Context, client events.Client, p *pkgscanner.ProjectResult, other []goprotoevent.TaggingPolicyResult) {
	current := make(map[string]map[string]struct{})
	for _, r := range p.TagPolicyResults {
		addrs := make(map[string]struct{}, len(r.FailingResources))
		for _, fr := range r.FailingResources {
			addrs[fr.Address] = struct{}{}
		}
		current[r.TagPolicyID] = addrs
	}

	for _, prev := range other {
		for _, fr := range prev.FailingResources {
			if cur, ok := current[prev.TagPolicyID]; ok {
				if _, still := cur[fr.Address]; still {
					continue
				}
			}
			client.Push(ctx, "cloud-issue-fixed",
				"policyId", prev.TagPolicyID,
				"type", "tag-policy",
				"projectName", p.Name,
				"resourceAddress", fr.Address,
				"pullRequestId", "",
				"autoFixPullRequest", false,
			)
		}
	}
}