package config_test

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

func TestScan_BasicCostDiff(t *testing.T) {
	cfg, m := testingconfig.Config(t)
	cfg.BasePath = filepath.Join(testdataDir(), "basic", "base")
	cfg.HeadPath = filepath.Join(testdataDir(), "basic", "head")
	processPlugins(&cfg)

	m.Dashboard.EXPECT().
		RunParameters(mock.Anything, mock.Anything, mock.Anything).
		Return(emptyRunParams(), nil)

	setupDashboardAddRun(m)
	data := setupVCSMocks(m)

	if _, err := cfg.Scan(); err != nil {
		t.Fatalf("Scan() returned error: %v", err)
	}

	if len(data.Projects) == 0 {
		t.Fatal("expected at least one project in comment data")
	}

	project := data.Projects[0]

	// Head has two instances (web + api), base has one (web).
	if project.PastTotalMonthlyCost == nil || project.PastTotalMonthlyCost.IsZero() {
		t.Error("expected non-zero past total monthly cost")
	}
	if project.TotalMonthlyCost == nil || project.TotalMonthlyCost.IsZero() {
		t.Error("expected non-zero total monthly cost")
	}

	// The diff breakdown should show a cost increase (new api instance).
	if project.DiffBreakdown == nil || project.DiffBreakdown.TotalMonthlyCost == nil || project.DiffBreakdown.TotalMonthlyCost.IsZero() {
		t.Error("expected non-zero diff breakdown cost")
	}
}

func TestScan_NoChanges(t *testing.T) {
	cfg, m := testingconfig.Config(t)
	cfg.BasePath = filepath.Join(testdataDir(), "no-changes", "base")
	cfg.HeadPath = filepath.Join(testdataDir(), "no-changes", "head")
	processPlugins(&cfg)

	m.Dashboard.EXPECT().
		RunParameters(mock.Anything, mock.Anything, mock.Anything).
		Return(emptyRunParams(), nil)

	setupDashboardAddRun(m)
	data := setupVCSMocks(m)

	if _, err := cfg.Scan(); err != nil {
		t.Fatalf("Scan() returned error: %v", err)
	}

	if len(data.Projects) == 0 {
		t.Fatal("expected at least one project in comment data")
	}

	project := data.Projects[0]

	// Same base and head — diff breakdown should be nil or zero.
	if project.DiffBreakdown != nil && project.DiffBreakdown.TotalMonthlyCost != nil && !project.DiffBreakdown.TotalMonthlyCost.IsZero() {
		t.Errorf("expected zero diff, got %s", project.DiffBreakdown.TotalMonthlyCost.String())
	}
}

func TestScan_DashboardError(t *testing.T) {
	cfg, m := testingconfig.Config(t)
	cfg.BasePath = filepath.Join(testdataDir(), "basic", "base")
	cfg.HeadPath = filepath.Join(testdataDir(), "basic", "head")
	processPlugins(&cfg)

	m.Dashboard.EXPECT().
		RunParameters(mock.Anything, mock.Anything, mock.Anything).
		Return(dashboard.RunParameters{}, errors.New("dashboard unavailable"))

	_, err := cfg.Scan()
	if err == nil {
		t.Fatal("expected error from Scan() when dashboard fails")
	}
}

func TestScan_GuardrailTriggered(t *testing.T) {
	cfg, m := testingconfig.Config(t)
	cfg.BasePath = filepath.Join(testdataDir(), "basic", "base")
	cfg.HeadPath = filepath.Join(testdataDir(), "basic", "head")
	processPlugins(&cfg)

	// Guardrail with a $1 increase threshold — the new instance will trigger it.
	guardrail := mustProtoJSON(t, &event.Guardrail{
		Id:                 "gr-1",
		Name:               "Cost increase limit",
		Scope:              event.Guardrail_REPO,
		IncreaseThreshold:  ratProto(1),
		PrComment:          true,
		BlockPr:            true,
		Message:            "Cost increase exceeds threshold",
	})

	params := emptyRunParams()
	params.Guardrails = []json.RawMessage{guardrail}

	m.Dashboard.EXPECT().
		RunParameters(mock.Anything, mock.Anything, mock.Anything).
		Return(params, nil)

	setupDashboardAddRun(m)
	data := setupVCSMocks(m)

	result, err := cfg.Scan()
	if err != nil {
		t.Fatalf("Scan() returned error: %v", err)
	}

	// At least one guardrail result should exist and be triggered.
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

	// The blocking guardrail should cause the scan to block the PR.
	if !result.BlockPR {
		t.Error("expected BlockPR to be true")
	}
	if len(result.Reasons) == 0 {
		t.Error("expected at least one blocking reason")
	}
}

func TestScan_GuardrailSuppressed(t *testing.T) {
	cfg, m := testingconfig.Config(t)
	// Use no-changes fixture: base and head are identical, so any guardrail
	// triggered in base should also appear in PreviousGuardrailResults.
	cfg.BasePath = filepath.Join(testdataDir(), "no-changes", "base")
	cfg.HeadPath = filepath.Join(testdataDir(), "no-changes", "head")
	processPlugins(&cfg)

	// Guardrail with $0 total threshold — both base and head exceed it.
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

	result, err := cfg.Scan()
	if err != nil {
		t.Fatalf("Scan() returned error: %v", err)
	}

	// The guardrail should appear in PreviousGuardrailResults (already triggered in base).
	if len(data.PreviousGuardrailResults) == 0 {
		t.Error("expected guardrail in PreviousGuardrailResults (suppressed)")
	}

	// Suppressed guardrails should not block the PR.
	if result.BlockPR {
		t.Errorf("expected BlockPR to be false for suppressed guardrails, got reasons: %v", result.Reasons)
	}
}

func TestScan_FinOpsPolicy(t *testing.T) {
	cfg, m := testingconfig.Config(t)
	cfg.BasePath = filepath.Join(testdataDir(), "basic", "base")
	cfg.HeadPath = filepath.Join(testdataDir(), "basic", "head")
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

	if _, err := cfg.Scan(); err != nil {
		t.Fatalf("Scan() returned error: %v", err)
	}

	// Verify the scan completed with policies configured and projects populated.
	if len(data.Projects) == 0 {
		t.Error("expected at least one project")
	}
}

func TestScan_UsageDefaults(t *testing.T) {
	cfg, m := testingconfig.Config(t)
	cfg.BasePath = filepath.Join(testdataDir(), "basic", "base")
	cfg.HeadPath = filepath.Join(testdataDir(), "basic", "head")
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

	if _, err := cfg.Scan(); err != nil {
		t.Fatalf("Scan() returned error: %v", err)
	}

	if !data.UsageAPIEnabled {
		t.Error("expected UsageAPIEnabled to be true when usage defaults are provided")
	}
}

func TestScan_SingleProjectFilter(t *testing.T) {
	cfg, m := testingconfig.Config(t)
	cfg.BasePath = filepath.Join(testdataDir(), "multi-project", "base")
	cfg.HeadPath = filepath.Join(testdataDir(), "multi-project", "head")
	cfg.Project = "web"
	processPlugins(&cfg)

	m.Dashboard.EXPECT().
		RunParameters(mock.Anything, mock.Anything, mock.Anything).
		Return(emptyRunParams(), nil)

	setupDashboardAddRun(m)
	data := setupVCSMocks(m)

	if _, err := cfg.Scan(); err != nil {
		t.Fatalf("Scan() returned error: %v", err)
	}

	if len(data.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(data.Projects))
	}
	if data.Projects[0].Name != "web" {
		t.Errorf("expected project name 'web', got %q", data.Projects[0].Name)
	}
}
