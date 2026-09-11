package commands

import (
	"path/filepath"
	"testing"

	"github.com/infracost/actions/tools/scanner/internal/config"
	"github.com/infracost/cli/pkg/config/process"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// The integration tests call diff() directly, so nothing in them exercises
// the env-then-flag ordering the contract is. These helpers run the real
// thing: PreProcess hydrates the config, then the command registers its flags
// against it, then cobra parses and executes.

func newTestRoot(t *testing.T, cfg *config.Config) *cobra.Command {
	t.Helper()

	root := &cobra.Command{Use: "scanner", SilenceUsage: true, SilenceErrors: true}
	diags := process.PreProcess(cfg, root.PersistentFlags())
	require.Zero(t, diags.Len(), "PreProcess reported %s", diags)
	return root
}

// execDiff parses argv and resolves the VCS context, stopping short of the
// scan. The resolution is what the contract specifies; the scan is not.
func execDiff(t *testing.T, argv ...string) (diffContext, error) {
	t.Helper()

	cfg := new(config.Config)
	root := newTestRoot(t, cfg)

	cmd, args := diffCommand(cfg, &ScanResult{})
	var captured diffContext
	cmd.RunE = func(*cobra.Command, []string) error {
		var err error
		captured, err = resolveDiffContext(cfg, args)
		return err
	}
	root.AddCommand(cmd)

	base := filepath.Join(testdataDir(), "basic", "base")
	head := filepath.Join(testdataDir(), "basic", "head")
	root.SetArgs(append([]string{"diff", "--base-path", base, "--head-path", head}, argv...))
	return captured, root.Execute()
}

// execStatus parses argv and resolves the pull request URL status keys on.
func execStatus(t *testing.T, argv ...string) (string, error) {
	t.Helper()

	cfg := new(config.Config)
	root := newTestRoot(t, cfg)

	cmd, args := statusCommand(cfg)
	var captured string
	cmd.RunE = func(*cobra.Command, []string) error {
		var err error
		captured, err = resolveStatusPullRequest(cfg, args)
		return err
	}
	root.AddCommand(cmd)

	root.SetArgs(append([]string{"status"}, argv...))
	return captured, root.Execute()
}

// execScanArgs parses argv and returns the bound args without scanning.
func execScanArgs(t *testing.T, argv ...string) (*config.Config, *scanArgs, error) {
	t.Helper()

	cfg := new(config.Config)
	root := newTestRoot(t, cfg)

	cmd, args := scanCommand(cfg)
	cmd.RunE = func(*cobra.Command, []string) error { return nil }
	root.AddCommand(cmd)

	root.SetArgs(append([]string{"scan", "--path", testdataDir()}, argv...))
	return cfg, args, root.Execute()
}
