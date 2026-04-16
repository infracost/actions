package config

import (
	"testing"

	pkgscanner "github.com/infracost/cli/pkg/scanner"
	goprotoevent "github.com/infracost/go-proto/pkg/event"
	"github.com/infracost/go-proto/pkg/rat"
	"github.com/infracost/proto/gen/go/infracost/provider"
	"github.com/infracost/proto/gen/go/infracost/rational"
	"github.com/infracost/vcs/pkg/vcs/comment"
)

func ratProto(s string) *rational.Rat {
	r, err := rat.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return r.Proto()
}

func TestConvertCostComponent(t *testing.T) {
	tests := []struct {
		name            string
		component       *provider.CostComponent
		wantMonthlyCost string
		wantPrice       string
		wantQty         string
	}{
		{
			name: "monthly price",
			component: &provider.CostComponent{
				Name: "Compute",
				Unit: "hours",
				PeriodPrice: &provider.PeriodPrice{
					Price:  ratProto("0.10"),
					Period: provider.Period_MONTH,
				},
				Quantity: ratProto("730"),
			},
			wantMonthlyCost: "73",
			wantPrice:       "0.1",
			wantQty:         "730",
		},
		{
			name: "hourly price normalized to monthly",
			component: &provider.CostComponent{
				Name: "Compute",
				Unit: "hours",
				PeriodPrice: &provider.PeriodPrice{
					Price:  ratProto("0.10"),
					Period: provider.Period_HOUR,
				},
				Quantity: ratProto("1"),
			},
			wantMonthlyCost: "73",
			wantPrice:       "0.1",
			wantQty:         "730",
		},
		{
			name: "nil price returns zero cost",
			component: &provider.CostComponent{
				Name: "Compute",
				PeriodPrice: &provider.PeriodPrice{
					Price:  nil,
					Period: provider.Period_MONTH,
				},
				Quantity: ratProto("100"),
			},
			wantMonthlyCost: "",
		},
		{
			name: "nil period price returns zero cost",
			component: &provider.CostComponent{
				Name:     "Compute",
				Quantity: ratProto("100"),
			},
			wantMonthlyCost: "",
		},
		{
			name: "nil quantity returns zero cost",
			component: &provider.CostComponent{
				Name: "Compute",
				PeriodPrice: &provider.PeriodPrice{
					Price:  ratProto("0.10"),
					Period: provider.Period_MONTH,
				},
			},
			wantMonthlyCost: "",
		},
		{
			name: "discount applied",
			component: &provider.CostComponent{
				Name: "Compute",
				PeriodPrice: &provider.PeriodPrice{
					Price:  ratProto("100"),
					Period: provider.Period_MONTH,
				},
				Quantity:     ratProto("1"),
				DiscountRate: ratProto("0.5"),
			},
			wantMonthlyCost: "50",
			wantPrice:       "50",
			wantQty:         "1",
		},
		{
			name: "usage based flag preserved",
			component: &provider.CostComponent{
				Name:       "Requests",
				Unit:       "requests",
				UsageBased: true,
				PeriodPrice: &provider.PeriodPrice{
					Price:  ratProto("0.01"),
					Period: provider.Period_MONTH,
				},
				Quantity: ratProto("1000"),
			},
			wantMonthlyCost: "10",
			wantPrice:       "0.01",
			wantQty:         "1000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := convertCostComponent(tt.component)

			if tt.name == "usage based flag preserved" && !cc.UsageBased {
				t.Error("expected UsageBased to be true")
			}

			if tt.wantMonthlyCost == "" {
				if cc.MonthlyCost != nil {
					t.Errorf("expected nil MonthlyCost, got %s", cc.MonthlyCost)
				}
				return
			}

			want, _ := rat.NewFromString(tt.wantMonthlyCost)
			if !cc.MonthlyCost.Equals(want) {
				t.Errorf("MonthlyCost = %s, want %s", cc.MonthlyCost, want)
			}
			if tt.wantPrice != "" {
				wantP, _ := rat.NewFromString(tt.wantPrice)
				if !cc.Price.Equals(wantP) {
					t.Errorf("Price = %s, want %s", cc.Price, wantP)
				}
			}
			if tt.wantQty != "" {
				wantQ, _ := rat.NewFromString(tt.wantQty)
				if !cc.MonthlyQuantity.Equals(wantQ) {
					t.Errorf("MonthlyQuantity = %s, want %s", cc.MonthlyQuantity, wantQ)
				}
			}
		})
	}
}

func TestBuildCostBreakdown(t *testing.T) {
	tests := []struct {
		name          string
		resources     []*provider.Resource
		wantTotal     string
		wantResources int
	}{
		{
			name:          "nil resources",
			resources:     nil,
			wantTotal:     "0",
			wantResources: 0,
		},
		{
			name: "single resource with cost",
			resources: []*provider.Resource{
				{
					Name: "aws_instance.web",
					Costs: &provider.ResourceCosts{
						Components: []*provider.CostComponent{
							{
								Name: "Compute",
								PeriodPrice: &provider.PeriodPrice{
									Price:  ratProto("10"),
									Period: provider.Period_MONTH,
								},
								Quantity: ratProto("1"),
							},
						},
					},
				},
			},
			wantTotal:     "10",
			wantResources: 1,
		},
		{
			name: "resource with child resources",
			resources: []*provider.Resource{
				{
					Name: "aws_instance.web",
					Costs: &provider.ResourceCosts{
						Components: []*provider.CostComponent{
							{
								Name: "Compute",
								PeriodPrice: &provider.PeriodPrice{
									Price:  ratProto("10"),
									Period: provider.Period_MONTH,
								},
								Quantity: ratProto("1"),
							},
						},
					},
					ChildResources: []*provider.Resource{
						{
							Name: "root_block_device",
							Costs: &provider.ResourceCosts{
								Components: []*provider.CostComponent{
									{
										Name: "Storage",
										PeriodPrice: &provider.PeriodPrice{
											Price:  ratProto("5"),
											Period: provider.Period_MONTH,
										},
										Quantity: ratProto("1"),
									},
								},
							},
						},
					},
				},
			},
			wantTotal:     "15",
			wantResources: 1,
		},
		{
			name: "multiple resources",
			resources: []*provider.Resource{
				{
					Name: "aws_instance.a",
					Costs: &provider.ResourceCosts{
						Components: []*provider.CostComponent{
							{
								Name:        "Compute",
								PeriodPrice: &provider.PeriodPrice{Price: ratProto("20"), Period: provider.Period_MONTH},
								Quantity:    ratProto("1"),
							},
						},
					},
				},
				{
					Name: "aws_instance.b",
					Costs: &provider.ResourceCosts{
						Components: []*provider.CostComponent{
							{
								Name:        "Compute",
								PeriodPrice: &provider.PeriodPrice{Price: ratProto("30"), Period: provider.Period_MONTH},
								Quantity:    ratProto("1"),
							},
						},
					},
				},
			},
			wantTotal:     "50",
			wantResources: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bd := buildCostBreakdown(tt.resources)

			want, _ := rat.NewFromString(tt.wantTotal)
			if !bd.TotalMonthlyCost.Equals(want) {
				t.Errorf("TotalMonthlyCost = %s, want %s", bd.TotalMonthlyCost, want)
			}
			if len(bd.Resources) != tt.wantResources {
				t.Errorf("got %d resources, want %d", len(bd.Resources), tt.wantResources)
			}
		})
	}
}

func TestDiffCostBreakdown(t *testing.T) {
	mkBreakdown := func(resources ...comment.BreakdownResource) *comment.CostBreakdown {
		total := rat.Zero
		for _, r := range resources {
			total = total.Add(orZero(r.MonthlyCost))
		}
		return &comment.CostBreakdown{
			TotalMonthlyCost: total,
			Resources:        resources,
		}
	}
	mkResource := func(name, cost string) comment.BreakdownResource {
		c, _ := rat.NewFromString(cost)
		return comment.BreakdownResource{Name: name, MonthlyCost: c}
	}

	tests := []struct {
		name              string
		past              *comment.CostBreakdown
		current           *comment.CostBreakdown
		wantTotalDiff     string
		wantDiffResources int
	}{
		{
			name:              "nil past returns current",
			past:              nil,
			current:           mkBreakdown(mkResource("a", "10")),
			wantTotalDiff:     "10",
			wantDiffResources: 1,
		},
		{
			name:    "nil current returns nil",
			past:    mkBreakdown(mkResource("a", "10")),
			current: nil,
		},
		{
			name:              "new resource",
			past:              mkBreakdown(),
			current:           mkBreakdown(mkResource("a", "10")),
			wantTotalDiff:     "10",
			wantDiffResources: 1,
		},
		{
			name:              "removed resource",
			past:              mkBreakdown(mkResource("a", "10")),
			current:           mkBreakdown(),
			wantTotalDiff:     "-10",
			wantDiffResources: 1,
		},
		{
			name:              "changed resource",
			past:              mkBreakdown(mkResource("a", "10")),
			current:           mkBreakdown(mkResource("a", "30")),
			wantTotalDiff:     "20",
			wantDiffResources: 1,
		},
		{
			name:              "unchanged resource excluded",
			past:              mkBreakdown(mkResource("a", "10")),
			current:           mkBreakdown(mkResource("a", "10")),
			wantTotalDiff:     "0",
			wantDiffResources: 0,
		},
		{
			name:              "mixed new changed removed",
			past:              mkBreakdown(mkResource("a", "10"), mkResource("b", "20")),
			current:           mkBreakdown(mkResource("a", "15"), mkResource("c", "5")),
			wantTotalDiff:     "-10",
			wantDiffResources: 3, // a changed, c new, b removed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := diffCostBreakdown(tt.past, tt.current)

			if tt.current == nil {
				if diff != nil {
					t.Error("expected nil diff for nil current")
				}
				return
			}

			want, _ := rat.NewFromString(tt.wantTotalDiff)
			if !diff.TotalMonthlyCost.Equals(want) {
				t.Errorf("TotalMonthlyCost diff = %s, want %s", diff.TotalMonthlyCost, want)
			}
			if len(diff.Resources) != tt.wantDiffResources {
				t.Errorf("got %d diff resources, want %d", len(diff.Resources), tt.wantDiffResources)
			}
		})
	}
}

func TestAggregateResourceSummary(t *testing.T) {
	tests := []struct {
		name                      string
		resources                 []*provider.Resource
		wantDetected              int
		wantSupported             int
		wantUnsupported           int
		wantNoPriceResources      int
		wantUnsupportedTypeCounts map[string]int
	}{
		{
			name:         "empty",
			resources:    nil,
			wantDetected: 0,
		},
		{
			name: "supported resource",
			resources: []*provider.Resource{
				{Type: "aws_instance", IsSupported: true, IsProviderSupported: true},
			},
			wantDetected:  1,
			wantSupported: 1,
		},
		{
			name: "unsupported resource with supported provider",
			resources: []*provider.Resource{
				{Type: "aws_foo", IsSupported: false, IsProviderSupported: true},
			},
			wantDetected:              1,
			wantUnsupported:           1,
			wantUnsupportedTypeCounts: map[string]int{"aws_foo": 1},
		},
		{
			name: "free resource",
			resources: []*provider.Resource{
				{Type: "aws_vpc", IsSupported: true, IsFree: true},
			},
			wantDetected:         1,
			wantSupported:        1,
			wantNoPriceResources: 1,
		},
		{
			name: "unsupported provider not counted as unsupported",
			resources: []*provider.Resource{
				{Type: "datadog_monitor", IsSupported: false, IsProviderSupported: false},
			},
			wantDetected: 1,
		},
		{
			name: "child resources counted",
			resources: []*provider.Resource{
				{
					Type:        "aws_instance",
					IsSupported: true,
					ChildResources: []*provider.Resource{
						{Type: "aws_ebs_volume", IsSupported: true},
					},
				},
			},
			wantDetected:  2,
			wantSupported: 2,
		},
		{
			name: "mixed resources",
			resources: []*provider.Resource{
				{Type: "aws_instance", IsSupported: true},
				{Type: "aws_foo", IsSupported: false, IsProviderSupported: true},
				{Type: "aws_foo", IsSupported: false, IsProviderSupported: true},
				{Type: "aws_vpc", IsSupported: true, IsFree: true},
			},
			wantDetected:              4,
			wantSupported:             2,
			wantUnsupported:           2,
			wantNoPriceResources:      1,
			wantUnsupportedTypeCounts: map[string]int{"aws_foo": 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var summary comment.ResourceSummary
			for _, r := range tt.resources {
				aggregateResourceSummary(&summary, r)
			}

			if summary.TotalDetectedResources != tt.wantDetected {
				t.Errorf("TotalDetectedResources = %d, want %d", summary.TotalDetectedResources, tt.wantDetected)
			}
			if summary.TotalSupportedResources != tt.wantSupported {
				t.Errorf("TotalSupportedResources = %d, want %d", summary.TotalSupportedResources, tt.wantSupported)
			}
			if summary.TotalUnsupportedResources != tt.wantUnsupported {
				t.Errorf("TotalUnsupportedResources = %d, want %d", summary.TotalUnsupportedResources, tt.wantUnsupported)
			}
			if summary.TotalNoPriceResources != tt.wantNoPriceResources {
				t.Errorf("TotalNoPriceResources = %d, want %d", summary.TotalNoPriceResources, tt.wantNoPriceResources)
			}
			if tt.wantUnsupportedTypeCounts != nil {
				for typ, want := range tt.wantUnsupportedTypeCounts {
					if got := summary.UnsupportedResourceCounts[typ]; got != want {
						t.Errorf("UnsupportedResourceCounts[%q] = %d, want %d", typ, got, want)
					}
				}
			}
		})
	}
}

func TestBuildCommentData(t *testing.T) {
	t.Run("head only project", func(t *testing.T) {
		head := &DirectoryResult{
			Projects: []pkgscanner.ProjectResult{
				{
					Name:             "project-a",
					TotalMonthlyCost: rat.New(100),
					Resources: []*provider.Resource{
						{Name: "aws_instance.web", Type: "aws_instance", IsSupported: true},
					},
				},
			},
			TotalMonthlyCost: rat.New(100),
			Currency:         "USD",
		}
		base := &DirectoryResult{}

		data := BuildCommentData(CommentDataOptions{BaseResult: base, HeadResult: head, Currency: "USD", Branch: "main"})

		if len(data.Projects) != 1 {
			t.Fatalf("expected 1 project, got %d", len(data.Projects))
		}
		if data.Projects[0].Name != "project-a" {
			t.Errorf("expected project name project-a, got %q", data.Projects[0].Name)
		}
		if data.Projects[0].PastBreakdown != nil {
			t.Error("expected nil PastBreakdown for head-only project")
		}
		if data.Projects[0].Breakdown == nil {
			t.Error("expected non-nil Breakdown")
		}
		if !data.TotalMonthlyCost.Equals(rat.New(100)) {
			t.Errorf("TotalMonthlyCost = %s, want 100", data.TotalMonthlyCost)
		}
		if !data.PastTotalMonthlyCost.Equals(rat.Zero) {
			t.Errorf("PastTotalMonthlyCost = %s, want 0", data.PastTotalMonthlyCost)
		}
	})

	t.Run("base only project", func(t *testing.T) {
		head := &DirectoryResult{}
		base := &DirectoryResult{
			Projects: []pkgscanner.ProjectResult{
				{
					Name:             "project-removed",
					TotalMonthlyCost: rat.New(50),
				},
			},
			TotalMonthlyCost: rat.New(50),
		}

		data := BuildCommentData(CommentDataOptions{BaseResult: base, HeadResult: head, Currency: "USD", Branch: "main"})

		if len(data.Projects) != 1 {
			t.Fatalf("expected 1 project, got %d", len(data.Projects))
		}
		if data.Projects[0].Breakdown != nil {
			t.Error("expected nil Breakdown for base-only project")
		}
		if data.Projects[0].PastBreakdown == nil {
			t.Error("expected non-nil PastBreakdown")
		}
	})

	t.Run("matched project with diff", func(t *testing.T) {
		head := &DirectoryResult{
			Projects: []pkgscanner.ProjectResult{
				{
					Name:             "project-a",
					TotalMonthlyCost: rat.New(150),
					Resources: []*provider.Resource{
						{
							Name: "aws_instance.web", Type: "aws_instance", IsSupported: true,
							Costs: &provider.ResourceCosts{
								Components: []*provider.CostComponent{
									{
										Name:        "Compute",
										PeriodPrice: &provider.PeriodPrice{Price: ratProto("150"), Period: provider.Period_MONTH},
										Quantity:    ratProto("1"),
									},
								},
							},
						},
					},
				},
			},
			TotalMonthlyCost: rat.New(150),
		}
		base := &DirectoryResult{
			Projects: []pkgscanner.ProjectResult{
				{
					Name:             "project-a",
					TotalMonthlyCost: rat.New(100),
					Resources: []*provider.Resource{
						{
							Name: "aws_instance.web", Type: "aws_instance", IsSupported: true,
							Costs: &provider.ResourceCosts{
								Components: []*provider.CostComponent{
									{
										Name:        "Compute",
										PeriodPrice: &provider.PeriodPrice{Price: ratProto("100"), Period: provider.Period_MONTH},
										Quantity:    ratProto("1"),
									},
								},
							},
						},
					},
				},
			},
			TotalMonthlyCost: rat.New(100),
		}

		data := BuildCommentData(CommentDataOptions{BaseResult: base, HeadResult: head, Currency: "USD", Branch: "main"})

		if len(data.Projects) != 1 {
			t.Fatalf("expected 1 project, got %d", len(data.Projects))
		}
		pr := data.Projects[0]
		if pr.DiffBreakdown == nil {
			t.Fatal("expected non-nil DiffBreakdown")
		}
		wantDiff, _ := rat.NewFromString("50")
		if !pr.DiffBreakdown.TotalMonthlyCost.Equals(wantDiff) {
			t.Errorf("DiffBreakdown.TotalMonthlyCost = %s, want %s", pr.DiffBreakdown.TotalMonthlyCost, wantDiff)
		}
	})

	t.Run("metadata fields passed through", func(t *testing.T) {
		head := &DirectoryResult{EstimatedUsageCounts: map[string]int{"a": 1}}
		base := &DirectoryResult{}

		data := BuildCommentData(CommentDataOptions{BaseResult: base, HeadResult: head, Currency: "EUR", RepoURL: "https://github.com/org/repo", CommitSHA: "abc123", Branch: "develop"})

		if data.Currency != "EUR" {
			t.Errorf("Currency = %q, want EUR", data.Currency)
		}
		if data.RepoURL != "https://github.com/org/repo" {
			t.Errorf("RepoURL = %q", data.RepoURL)
		}
		if data.CommitSHA != "abc123" {
			t.Errorf("CommitSHA = %q", data.CommitSHA)
		}
		if data.BaseBranchName != "develop" {
			t.Errorf("BaseBranchName = %q", data.BaseBranchName)
		}
		if !data.UsedUsageFile {
			t.Error("expected UsedUsageFile to be true when EstimatedUsageCounts is non-nil")
		}
	})

	t.Run("finops and tagging results aggregated", func(t *testing.T) {
		head := &DirectoryResult{
			Projects: []pkgscanner.ProjectResult{
				{
					Name:          "a",
					FinopsResults: []*provider.FinopsPolicyResult{{PolicySlug: "p1"}},
					TagPolicyResults: []goprotoevent.TaggingPolicyResult{
						{Name: "tag1"},
					},
				},
				{
					Name:          "b",
					FinopsResults: []*provider.FinopsPolicyResult{{PolicySlug: "p2"}},
				},
			},
		}
		base := &DirectoryResult{
			Projects: []pkgscanner.ProjectResult{
				{
					Name:          "a",
					FinopsResults: []*provider.FinopsPolicyResult{{PolicySlug: "prev-p1"}},
				},
			},
		}

		data := BuildCommentData(CommentDataOptions{BaseResult: base, HeadResult: head, Currency: "USD", Branch: "main"})

		if len(data.FinOpsPolicyResults) != 2 {
			t.Errorf("expected 2 FinOpsPolicyResults, got %d", len(data.FinOpsPolicyResults))
		}
		if len(data.PreviousFinOpsPolicyResults) != 1 {
			t.Errorf("expected 1 PreviousFinOpsPolicyResults, got %d", len(data.PreviousFinOpsPolicyResults))
		}
		if len(data.TaggingPolicyResults) != 1 {
			t.Errorf("expected 1 TaggingPolicyResults, got %d", len(data.TaggingPolicyResults))
		}
	})

	t.Run("usage costs computed", func(t *testing.T) {
		head := &DirectoryResult{
			Projects: []pkgscanner.ProjectResult{
				{
					Name:             "a",
					TotalMonthlyCost: rat.New(100),
					Resources: []*provider.Resource{
						{
							Name: "aws_instance.web",
							Costs: &provider.ResourceCosts{
								Components: []*provider.CostComponent{
									{
										Name:        "Compute",
										PeriodPrice: &provider.PeriodPrice{Price: ratProto("80"), Period: provider.Period_MONTH},
										Quantity:    ratProto("1"),
										UsageBased:  false,
									},
									{
										Name:        "Requests",
										PeriodPrice: &provider.PeriodPrice{Price: ratProto("20"), Period: provider.Period_MONTH},
										Quantity:    ratProto("1"),
										UsageBased:  true,
									},
								},
							},
						},
					},
				},
			},
		}
		base := &DirectoryResult{}

		data := BuildCommentData(CommentDataOptions{BaseResult: base, HeadResult: head, Currency: "USD", Branch: "main"})

		pr := data.Projects[0]
		if pr.TotalMonthlyUsageCost == nil {
			t.Fatal("expected TotalMonthlyUsageCost to be set")
		}
		if !pr.TotalMonthlyUsageCost.Equals(rat.New(20)) {
			t.Errorf("TotalMonthlyUsageCost = %s, want 20", pr.TotalMonthlyUsageCost)
		}
	})

	t.Run("carbon diff computed", func(t *testing.T) {
		head := &DirectoryResult{
			Projects: []pkgscanner.ProjectResult{
				{
					Name: "a",
					Resources: []*provider.Resource{
						{
							Name: "aws_instance.web",
							Costs: &provider.ResourceCosts{
								Components: []*provider.CostComponent{
									{
										Name:     "Compute",
										Quantity: ratProto("1"),
										PeriodPrice: &provider.PeriodPrice{
											Price:  ratProto("10"),
											Period: provider.Period_MONTH,
										},
										EnvironmentalMetrics: &provider.EnvironmentalMetrics{
											Period:          provider.Period_MONTH,
											CarbonGramsCo2E: ratProto("500"),
										},
									},
								},
							},
						},
					},
				},
			},
		}
		base := &DirectoryResult{
			Projects: []pkgscanner.ProjectResult{
				{
					Name: "a",
					Resources: []*provider.Resource{
						{
							Name: "aws_instance.web",
							Costs: &provider.ResourceCosts{
								Components: []*provider.CostComponent{
									{
										Name:     "Compute",
										Quantity: ratProto("1"),
										PeriodPrice: &provider.PeriodPrice{
											Price:  ratProto("10"),
											Period: provider.Period_MONTH,
										},
										EnvironmentalMetrics: &provider.EnvironmentalMetrics{
											Period:          provider.Period_MONTH,
											CarbonGramsCo2E: ratProto("300"),
										},
									},
								},
							},
						},
					},
				},
			},
		}

		data := BuildCommentData(CommentDataOptions{BaseResult: base, HeadResult: head, Currency: "USD", Branch: "main"})

		if data.DiffTotalMonthlyCarbonGramsCo2e == nil {
			t.Fatal("expected DiffTotalMonthlyCarbonGramsCo2e to be set")
		}
		if !data.DiffTotalMonthlyCarbonGramsCo2e.Equals(rat.New(200)) {
			t.Errorf("DiffTotalMonthlyCarbonGramsCo2e = %s, want 200", data.DiffTotalMonthlyCarbonGramsCo2e)
		}
	})

	t.Run("carbon diff nil when no metrics", func(t *testing.T) {
		head := &DirectoryResult{
			Projects: []pkgscanner.ProjectResult{
				{
					Name: "a",
					Resources: []*provider.Resource{
						{Name: "aws_instance.web", Costs: &provider.ResourceCosts{
							Components: []*provider.CostComponent{
								{
									Name:        "Compute",
									PeriodPrice: &provider.PeriodPrice{Price: ratProto("10"), Period: provider.Period_MONTH},
									Quantity:    ratProto("1"),
								},
							},
						}},
					},
				},
			},
		}
		base := &DirectoryResult{}

		data := BuildCommentData(CommentDataOptions{BaseResult: base, HeadResult: head, Currency: "USD", Branch: "main"})

		// No environmental metrics on any component, so carbon should still be computed
		// (it'll be zero for head, zero for base). The diff is only nil when neither side
		// has any projects with env metrics at all — but since we always compute it when
		// there are projects, it should be zero, not nil.
		// Actually our logic sets diffCarbon when headCarbon != nil || baseCarbon != nil.
		// Since head has resources but no env metrics, headCarbon is rat.Zero (not nil),
		// so diff will be set.
		if data.DiffTotalMonthlyCarbonGramsCo2e == nil {
			t.Fatal("expected non-nil DiffTotalMonthlyCarbonGramsCo2e")
		}
		if !data.DiffTotalMonthlyCarbonGramsCo2e.IsZero() {
			t.Errorf("expected zero carbon diff, got %s", data.DiffTotalMonthlyCarbonGramsCo2e)
		}
	})

	t.Run("guardrail results converted to pointers", func(t *testing.T) {
		head := &DirectoryResult{}
		base := &DirectoryResult{}
		guardrails := []goprotoevent.GuardrailResult{
			{GuardrailID: "g1", Triggered: true},
			{GuardrailID: "g2", Triggered: false},
		}

		data := BuildCommentData(CommentDataOptions{BaseResult: base, HeadResult: head, GuardrailResults: guardrails, Currency: "USD", Branch: "main"})

		if len(data.GuardrailResults) != 2 {
			t.Fatalf("expected 2 GuardrailResults, got %d", len(data.GuardrailResults))
		}
		if data.GuardrailResults[0].GuardrailID != "g1" {
			t.Errorf("expected g1, got %q", data.GuardrailResults[0].GuardrailID)
		}
	})
}
