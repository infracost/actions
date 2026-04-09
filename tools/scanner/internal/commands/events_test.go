package commands

import (
	"context"
	"testing"

	"github.com/infracost/actions/tools/scanner/internal/api/events/mocks"
	"github.com/infracost/actions/tools/scanner/internal/config"
	pkgscanner "github.com/infracost/cli/pkg/scanner"
	goprotoevent "github.com/infracost/go-proto/pkg/event"
	"github.com/infracost/proto/gen/go/infracost/provider"
	"github.com/stretchr/testify/mock"
)

func TestTrackRun_BasicResourceCounts(t *testing.T) {
	client := mocks.NewMockClient(t)

	result := &config.DirectoryResult{
		Projects: []pkgscanner.ProjectResult{
			{
				Name: "project-a",
				Resources: []*provider.Resource{
					{Name: "aws_instance.web", Type: "aws_instance", IsSupported: true},
					{Name: "aws_instance.api", Type: "aws_instance", IsSupported: true},
					{Name: "aws_s3_bucket.logs", Type: "aws_s3_bucket", IsSupported: true, IsFree: true},
					{Name: "aws_custom.thing", Type: "aws_custom", IsSupported: false},
				},
			},
		},
	}

	var captured []interface{}
	client.EXPECT().
		Push(mock.Anything, "infracost-run", mock.Anything).
		Run(func(_ context.Context, _ string, extra ...interface{}) {
			captured = extra
		}).
		Return()

	trackRun(context.Background(), client, result, nil, 1.5, "upload")

	env := toMap(t, captured)
	assertEq(t, env, "totalResources", 4)
	assertEq(t, env, "totalSupportedResources", 2)
	assertEq(t, env, "totalNoPriceResources", 1)
	assertEq(t, env, "totalUnsupportedResources", 1)
	assertEq(t, env, "runSeconds", 1.5)
	assertEq(t, env, "outputFormat", "upload")
	assertEq(t, env, "hasUsageFile", false)

	// No diff fields when prev is nil.
	if _, ok := env["newResourceCount"]; ok {
		t.Error("expected no newResourceCount when prev is nil")
	}
}

func TestTrackRun_WithUsageFile(t *testing.T) {
	client := mocks.NewMockClient(t)

	result := &config.DirectoryResult{
		Projects:               []pkgscanner.ProjectResult{{Name: "p"}},
		EstimatedUsageCounts:   map[string]int{"aws_instance.monthly_hrs": 2},
		UnestimatedUsageCounts: map[string]int{"aws_lambda.requests": 1},
	}

	var captured []interface{}
	client.EXPECT().
		Push(mock.Anything, "infracost-run", mock.Anything).
		Run(func(_ context.Context, _ string, extra ...interface{}) {
			captured = extra
		}).
		Return()

	trackRun(context.Background(), client, result, nil, 0.5, "upload")

	env := toMap(t, captured)
	assertEq(t, env, "hasUsageFile", true)
	assertEq(t, env, "totalEstimatedUsages", 2)
	assertEq(t, env, "totalUnestimatedUsages", 1)
}

func TestTrackRun_WithDiff(t *testing.T) {
	client := mocks.NewMockClient(t)

	prev := &config.DirectoryResult{
		Projects: []pkgscanner.ProjectResult{
			{
				Name: "p",
				Resources: []*provider.Resource{
					{Name: "aws_instance.web", Type: "aws_instance", IsSupported: true, Metadata: &provider.ResourceMetadata{DeepChecksum: "aaa"}},
				},
			},
		},
	}
	current := &config.DirectoryResult{
		Projects: []pkgscanner.ProjectResult{
			{
				Name: "p",
				Resources: []*provider.Resource{
					{Name: "aws_instance.web", Type: "aws_instance", IsSupported: true, Metadata: &provider.ResourceMetadata{DeepChecksum: "bbb"}},
					{Name: "aws_instance.api", Type: "aws_instance", IsSupported: true, Metadata: &provider.ResourceMetadata{DeepChecksum: "ccc"}},
				},
			},
		},
	}

	var captured []interface{}
	client.EXPECT().
		Push(mock.Anything, "infracost-run", mock.Anything).
		Run(func(_ context.Context, _ string, extra ...interface{}) {
			captured = extra
		}).
		Return()

	trackRun(context.Background(), client, current, prev, 2.0, "comment")

	env := toMap(t, captured)
	assertEq(t, env, "newResourceCount", 1)
	assertEq(t, env, "changedResourceCount", 1)
}

func TestTrackDiff_FinopsFixed(t *testing.T) {
	client := mocks.NewMockClient(t)

	base := &config.DirectoryResult{
		Projects: []pkgscanner.ProjectResult{
			{
				Name: "p",
				FinopsResults: []*provider.FinopsPolicyResult{
					{
						PolicyId:   "fp-1",
						PolicySlug: "aws-gp2-volumes",
						FailingResources: []*provider.FinopsPolicyFailingResource{
							{CauseAddress: "aws_ebs_volume.old"},
							{CauseAddress: "aws_ebs_volume.still_bad"},
						},
					},
				},
			},
		},
	}

	head := &config.DirectoryResult{
		Projects: []pkgscanner.ProjectResult{
			{
				Name: "p",
				FinopsResults: []*provider.FinopsPolicyResult{
					{
						PolicyId:   "fp-1",
						PolicySlug: "aws-gp2-volumes",
						FailingResources: []*provider.FinopsPolicyFailingResource{
							{CauseAddress: "aws_ebs_volume.still_bad"},
						},
					},
				},
			},
		},
	}

	// Expect one cloud-issue-fixed for the resolved violation.
	client.EXPECT().
		Push(mock.Anything, "cloud-issue-fixed", mock.Anything).
		Run(func(_ context.Context, _ string, extra ...interface{}) {
			env := toMap(t, extra)
			assertEq(t, env, "policyId", "fp-1")
			assertEq(t, env, "policySlug", "aws-gp2-volumes")
			assertEq(t, env, "type", "finops-policy")
			assertEq(t, env, "resourceAddress", "aws_ebs_volume.old")
		}).
		Return().
		Once()

	trackDiff(context.Background(), client, head, base)
}

func TestTrackDiff_TaggingFixed(t *testing.T) {
	client := mocks.NewMockClient(t)

	base := &config.DirectoryResult{
		Projects: []pkgscanner.ProjectResult{
			{
				Name: "p",
				TagPolicyResults: []goprotoevent.TaggingPolicyResult{
					{
						TagPolicyID: "tp-1",
						FailingResources: []goprotoevent.TagPolicyResultResource{
							{Address: "aws_instance.untagged"},
						},
					},
				},
			},
		},
	}

	head := &config.DirectoryResult{
		Projects: []pkgscanner.ProjectResult{
			{
				Name:             "p",
				TagPolicyResults: []goprotoevent.TaggingPolicyResult{},
			},
		},
	}

	client.EXPECT().
		Push(mock.Anything, "cloud-issue-fixed", mock.Anything).
		Run(func(_ context.Context, _ string, extra ...interface{}) {
			env := toMap(t, extra)
			assertEq(t, env, "policyId", "tp-1")
			assertEq(t, env, "type", "tag-policy")
			assertEq(t, env, "resourceAddress", "aws_instance.untagged")
		}).
		Return().
		Once()

	trackDiff(context.Background(), client, head, base)
}

func TestTrackDiff_NilBase(t *testing.T) {
	client := mocks.NewMockClient(t)
	// No Push calls expected.
	trackDiff(context.Background(), client, &config.DirectoryResult{}, nil)
}

func TestTrackDiff_NoMatchingProject(t *testing.T) {
	client := mocks.NewMockClient(t)

	base := &config.DirectoryResult{
		Projects: []pkgscanner.ProjectResult{{Name: "old-project"}},
	}
	head := &config.DirectoryResult{
		Projects: []pkgscanner.ProjectResult{{Name: "new-project"}},
	}

	// No Push calls expected — projects don't match.
	trackDiff(context.Background(), client, head, base)
}

func TestTrackDiff_NothingFixed(t *testing.T) {
	client := mocks.NewMockClient(t)

	results := &config.DirectoryResult{
		Projects: []pkgscanner.ProjectResult{
			{
				Name: "p",
				FinopsResults: []*provider.FinopsPolicyResult{
					{
						PolicySlug: "pol",
						FailingResources: []*provider.FinopsPolicyFailingResource{
							{CauseAddress: "r1"},
						},
					},
				},
			},
		},
	}

	// Same violations in base and head — no events.
	trackDiff(context.Background(), client, results, results)
}

func TestComputeRunDiff_NewProject(t *testing.T) {
	prev := &config.DirectoryResult{}
	current := &config.DirectoryResult{
		Projects: []pkgscanner.ProjectResult{
			{
				Name: "new-project",
				Resources: []*provider.Resource{
					{Name: "r1"},
					{Name: "r2"},
				},
				FinopsResults: []*provider.FinopsPolicyResult{
					{FailingResources: []*provider.FinopsPolicyFailingResource{{CauseAddress: "r1"}}},
				},
			},
		},
	}

	diff := computeRunDiff(current, prev)
	if diff.newResources != 2 {
		t.Errorf("expected 2 new resources, got %d", diff.newResources)
	}
	if diff.newIssues != 1 {
		t.Errorf("expected 1 new issue, got %d", diff.newIssues)
	}
	if diff.changedResources != 0 {
		t.Errorf("expected 0 changed resources, got %d", diff.changedResources)
	}
	if diff.existingIssues != 0 {
		t.Errorf("expected 0 existing issues, got %d", diff.existingIssues)
	}
}

func TestComputeRunDiff_ExistingIssues(t *testing.T) {
	prev := &config.DirectoryResult{
		Projects: []pkgscanner.ProjectResult{
			{
				Name: "p",
				Resources: []*provider.Resource{
					{Name: "r1", Metadata: &provider.ResourceMetadata{DeepChecksum: "aaa"}},
				},
				FinopsResults: []*provider.FinopsPolicyResult{
					{PolicySlug: "pol", FailingResources: []*provider.FinopsPolicyFailingResource{{CauseAddress: "r1"}}},
				},
				TagPolicyResults: []goprotoevent.TaggingPolicyResult{
					{TagPolicyID: "tp", FailingResources: []goprotoevent.TagPolicyResultResource{{Address: "r1"}}},
				},
			},
		},
	}
	current := &config.DirectoryResult{
		Projects: []pkgscanner.ProjectResult{
			{
				Name: "p",
				Resources: []*provider.Resource{
					{Name: "r1", Metadata: &provider.ResourceMetadata{DeepChecksum: "aaa"}},
				},
				FinopsResults: []*provider.FinopsPolicyResult{
					{PolicySlug: "pol", FailingResources: []*provider.FinopsPolicyFailingResource{{CauseAddress: "r1"}}},
				},
				TagPolicyResults: []goprotoevent.TaggingPolicyResult{
					{TagPolicyID: "tp", FailingResources: []goprotoevent.TagPolicyResultResource{{Address: "r1"}}},
				},
			},
		},
	}

	diff := computeRunDiff(current, prev)
	if diff.existingIssues != 2 {
		t.Errorf("expected 2 existing issues, got %d", diff.existingIssues)
	}
	if diff.newIssues != 0 {
		t.Errorf("expected 0 new issues, got %d", diff.newIssues)
	}
	if diff.changedResources != 0 {
		t.Errorf("expected 0 changed resources, got %d", diff.changedResources)
	}
}

// toMap converts the variadic key-value pairs into a map for easier assertions.
func toMap(t *testing.T, kvs []interface{}) map[string]interface{} {
	t.Helper()
	if len(kvs)%2 != 0 {
		t.Fatalf("expected even number of key-value pairs, got %d", len(kvs))
	}
	m := make(map[string]interface{}, len(kvs)/2)
	for i := 0; i < len(kvs); i += 2 {
		key, ok := kvs[i].(string)
		if !ok {
			t.Fatalf("expected string key at index %d, got %T", i, kvs[i])
		}
		m[key] = kvs[i+1]
	}
	return m
}

// assertEq checks that m[key] equals want.
func assertEq(t *testing.T, m map[string]interface{}, key string, want interface{}) {
	t.Helper()
	got, ok := m[key]
	if !ok {
		t.Errorf("key %q not found in event metadata", key)
		return
	}
	if got != want {
		t.Errorf("key %q: got %v (%T), want %v (%T)", key, got, got, want, want)
	}
}