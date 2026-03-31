package commands

import (
	"encoding/json"
	"errors"
	"math/big"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/infracost/actions/tools/scanner/internal/api/dashboard"
	"github.com/infracost/actions/tools/scanner/internal/config"
	testingconfig "github.com/infracost/actions/tools/scanner/internal/config/testing"
	"github.com/infracost/proto/gen/go/infracost/parser/event"
	"github.com/infracost/proto/gen/go/infracost/rational"
	"github.com/infracost/vcs/pkg/vcs"
	"github.com/infracost/vcs/pkg/vcs/comment"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func testdataDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata")
}

// processPlugins initialises the parser and provider Load functions.
// Must be called after the config is in its final location (pointer-stable)
// so that the closures capture the correct *parser.Config / *providers.Config.
func processPlugins(cfg *config.Config) {
	cfg.Plugins.Process()
	cfg.Plugins.Parser.Process()
	cfg.Plugins.Providers.Process()
}

// emptyRunParams returns dashboard.RunParameters with minimal required fields.
func emptyRunParams() dashboard.RunParameters {
	return dashboard.RunParameters{
		OrganizationID:   "test-org-id",
		OrganizationSlug: "test-org",
		RepositoryID:     "test-repo-id",
		RepositoryName:   "test-repo",
	}
}

// setupDashboardAddRun configures the dashboard mock to accept AddRun and return a test URL.
func setupDashboardAddRun(m *testingconfig.Mocks) {
	m.Dashboard.EXPECT().
		AddRun(mock.Anything, mock.Anything).
		Return(dashboard.AddRunResult{
			ID:       "test-run-id",
			CloudURL: "https://dashboard.infracost.io/org/test-org/repos/test-repo-id/runs/test-run-id",
		}, nil)
}

// setupVCSMocks configures the VCS mock to capture comment.Data and accept PostComment.
func setupVCSMocks(m *testingconfig.Mocks) *comment.Data {
	var captured comment.Data
	m.VCS.EXPECT().
		GenerateComment(mock.Anything).
		Run(func(data comment.Data) {
			captured = data
		}).
		Return("comment body", nil)
	m.VCS.EXPECT().
		PostComment(mock.Anything, "comment body", vcs.BehaviorUpdate).
		Return(vcs.PostResult{}, nil)
	return &captured
}

// mustProtoJSON marshals a proto message to json.RawMessage using protojson.
func mustProtoJSON(t *testing.T, msg proto.Message) json.RawMessage {
	t.Helper()
	b, err := protojson.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal proto to JSON: %v", err)
	}
	return b
}

// ratProto creates a rational.Rat from a numerator integer (denominator = 1).
func ratProto(num int64) *rational.Rat {
	n := big.NewInt(num)
	negative := num < 0
	if negative {
		n = n.Abs(n)
	}
	return &rational.Rat{
		Numerator:   n.Bytes(),
		Denominator: big.NewInt(1).Bytes(),
		Negative:    negative,
	}
}

func runDiff(t *testing.T, cfg *config.Config, m *testingconfig.Mocks, basePath, headPath string) (*ScanResult, error) {
	t.Helper()
	var results ScanResult
	err := diff(cfg, &diffArgs{
		basePath: basePath,
		headPath: headPath,
	}, m.VCS, &results)
	return &results, err
}

func TestDiff_BasicCostDiff(t *testing.T) {
	cfg, m := testingconfig.Config(t)
	processPlugins(&cfg)

	m.Dashboard.EXPECT().
		RunParameters(mock.Anything, mock.Anything, mock.Anything).
		Return(emptyRunParams(), nil)

	setupDashboardAddRun(m)
	data := setupVCSMocks(m)

	_, err := runDiff(t, &cfg, m, filepath.Join(testdataDir(), "basic", "base"), filepath.Join(testdataDir(), "basic", "head"))
	if err != nil {
		t.Fatalf("diff() returned error: %v", err)
	}

	if len(data.Projects) == 0 {
		t.Fatal("expected at least one project in comment data")
	}

	project := data.Projects[0]

	if project.PastTotalMonthlyCost == nil || project.PastTotalMonthlyCost.IsZero() {
		t.Error("expected non-zero past total monthly cost")
	}
	if project.TotalMonthlyCost == nil || project.TotalMonthlyCost.IsZero() {
		t.Error("expected non-zero total monthly cost")
	}

	if project.DiffBreakdown == nil || project.DiffBreakdown.TotalMonthlyCost == nil || project.DiffBreakdown.TotalMonthlyCost.IsZero() {
		t.Error("expected non-zero diff breakdown cost")
	}
}

func TestDiff_NoChanges(t *testing.T) {
	cfg, m := testingconfig.Config(t)
	processPlugins(&cfg)

	m.Dashboard.EXPECT().
		RunParameters(mock.Anything, mock.Anything, mock.Anything).
		Return(emptyRunParams(), nil)

	setupDashboardAddRun(m)
	data := setupVCSMocks(m)

	_, err := runDiff(t, &cfg, m, filepath.Join(testdataDir(), "no-changes", "base"), filepath.Join(testdataDir(), "no-changes", "head"))
	if err != nil {
		t.Fatalf("diff() returned error: %v", err)
	}

	if len(data.Projects) == 0 {
		t.Fatal("expected at least one project in comment data")
	}

	project := data.Projects[0]

	if project.DiffBreakdown != nil && project.DiffBreakdown.TotalMonthlyCost != nil && !project.DiffBreakdown.TotalMonthlyCost.IsZero() {
		t.Errorf("expected zero diff, got %s", project.DiffBreakdown.TotalMonthlyCost.String())
	}
}

func TestDiff_DashboardError(t *testing.T) {
	cfg, m := testingconfig.Config(t)
	processPlugins(&cfg)

	m.Dashboard.EXPECT().
		RunParameters(mock.Anything, mock.Anything, mock.Anything).
		Return(dashboard.RunParameters{}, errors.New("dashboard unavailable"))

	_, err := runDiff(t, &cfg, m, filepath.Join(testdataDir(), "basic", "base"), filepath.Join(testdataDir(), "basic", "head"))
	if err == nil {
		t.Fatal("expected error from diff() when dashboard fails")
	}
}

func TestDiff_GuardrailTriggered(t *testing.T) {
	cfg, m := testingconfig.Config(t)
	processPlugins(&cfg)

	guardrail := mustProtoJSON(t, &event.Guardrail{
		Id:                "gr-1",
		Name:              "Cost increase limit",
		Scope:             event.Guardrail_REPO,
		IncreaseThreshold: ratProto(1),
		PrComment:         true,
		BlockPr:           true,
		Message:           "Cost increase exceeds threshold",
	})

	params := emptyRunParams()
	params.Guardrails = []json.RawMessage{guardrail}

	m.Dashboard.EXPECT().
		RunParameters(mock.Anything, mock.Anything, mock.Anything).
		Return(params, nil)

	setupDashboardAddRun(m)
	data := setupVCSMocks(m)

	result, err := runDiff(t, &cfg, m, filepath.Join(testdataDir(), "basic", "base"), filepath.Join(testdataDir(), "basic", "head"))
	if err != nil {
		t.Fatalf("diff() returned error: %v", err)
	}

	triggered := false
	for _, gr := range data.GuardrailResults {
		if gr.Triggered {
			triggered = true
			break
		}
	}
	if !triggered {
		t.Error("expected at least one triggered guardrail result")
	}

	if !result.BlockPR {
		t.Error("expected BlockPR to be true")
	}
	if len(result.Reasons) == 0 {
		t.Error("expected at least one blocking reason")
	}
}

func TestDiff_GuardrailSuppressed(t *testing.T) {
	cfg, m := testingconfig.Config(t)
	processPlugins(&cfg)

	guardrail := mustProtoJSON(t, &event.Guardrail{
		Id:             "gr-suppress",
		Name:           "Total cost limit",
		Scope:          event.Guardrail_REPO,
		TotalThreshold: ratProto(0),
		PrComment:      true,
		Message:        "Total cost exceeds threshold",
	})

	params := emptyRunParams()
	params.Guardrails = []json.RawMessage{guardrail}

	m.Dashboard.EXPECT().
		RunParameters(mock.Anything, mock.Anything, mock.Anything).
		Return(params, nil)

	setupDashboardAddRun(m)
	data := setupVCSMocks(m)

	result, err := runDiff(t, &cfg, m, filepath.Join(testdataDir(), "no-changes", "base"), filepath.Join(testdataDir(), "no-changes", "head"))
	if err != nil {
		t.Fatalf("diff() returned error: %v", err)
	}

	if len(data.PreviousGuardrailResults) == 0 {
		t.Error("expected guardrail in PreviousGuardrailResults (suppressed)")
	}

	if result.BlockPR {
		t.Errorf("expected BlockPR to be false for suppressed guardrails, got reasons: %v", result.Reasons)
	}
}

func TestDiff_FinOpsPolicy(t *testing.T) {
	cfg, m := testingconfig.Config(t)
	processPlugins(&cfg)

	finopsPolicy := mustProtoJSON(t, &event.FinopsPolicySettings{
		Id:        "fp-1",
		Slug:      "aws-gp2-volumes",
		Name:      "Use gp3 instead of gp2",
		Message:   "gp3 volumes are cheaper",
		PrComment: true,
		Group:     event.FinopsPolicySettings_FINOPS,
	})

	securityPolicy := mustProtoJSON(t, &event.FinopsPolicySettings{
		Id:        "sp-1",
		Slug:      "aws-unencrypted-volumes",
		Name:      "Encrypt EBS volumes",
		Message:   "EBS volumes should be encrypted",
		PrComment: true,
		Group:     event.FinopsPolicySettings_CLOUD_SECURITY,
	})

	params := emptyRunParams()
	params.FinopsPolicies = []json.RawMessage{finopsPolicy, securityPolicy}

	m.Dashboard.EXPECT().
		RunParameters(mock.Anything, mock.Anything, mock.Anything).
		Return(params, nil)

	setupDashboardAddRun(m)
	data := setupVCSMocks(m)

	_, err := runDiff(t, &cfg, m, filepath.Join(testdataDir(), "basic", "base"), filepath.Join(testdataDir(), "basic", "head"))
	if err != nil {
		t.Fatalf("diff() returned error: %v", err)
	}

	if len(data.Projects) == 0 {
		t.Error("expected at least one project")
	}
}

func TestDiff_UsageDefaults(t *testing.T) {
	cfg, m := testingconfig.Config(t)
	processPlugins(&cfg)

	usageDefaults := mustProtoJSON(t, &event.UsageDefaults{
		Resources: map[string]*event.UsageResourceMap{
			"aws_instance": {
				Usages: map[string]*event.UsageDefaultList{
					"monthly_hrs": {
						List: []*event.UsageDefault{
							{Quantity: "730"},
						},
					},
				},
			},
		},
	})

	params := emptyRunParams()
	params.UsageDefaults = usageDefaults

	m.Dashboard.EXPECT().
		RunParameters(mock.Anything, mock.Anything, mock.Anything).
		Return(params, nil)

	setupDashboardAddRun(m)
	data := setupVCSMocks(m)

	_, err := runDiff(t, &cfg, m, filepath.Join(testdataDir(), "basic", "base"), filepath.Join(testdataDir(), "basic", "head"))
	if err != nil {
		t.Fatalf("diff() returned error: %v", err)
	}

	if !data.UsageAPIEnabled {
		t.Error("expected UsageAPIEnabled to be true when usage defaults are provided")
	}
}

func TestDiff_SingleProjectFilter(t *testing.T) {
	cfg, m := testingconfig.Config(t)
	cfg.Project = "web"
	processPlugins(&cfg)

	m.Dashboard.EXPECT().
		RunParameters(mock.Anything, mock.Anything, mock.Anything).
		Return(emptyRunParams(), nil)

	setupDashboardAddRun(m)
	data := setupVCSMocks(m)

	_, err := runDiff(t, &cfg, m, filepath.Join(testdataDir(), "multi-project", "base"), filepath.Join(testdataDir(), "multi-project", "head"))
	if err != nil {
		t.Fatalf("diff() returned error: %v", err)
	}

	if len(data.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(data.Projects))
	}
	if data.Projects[0].Name != "web" {
		t.Errorf("expected project name 'web', got %q", data.Projects[0].Name)
	}
}