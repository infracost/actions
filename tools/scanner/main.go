package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/infracost/actions/tools/scanner/internal/api"
	"github.com/infracost/actions/tools/scanner/internal/api/events"
	"github.com/infracost/actions/tools/scanner/internal/commands"
	"github.com/infracost/actions/tools/scanner/internal/config"
	"github.com/infracost/actions/tools/scanner/internal/version"
	"github.com/infracost/cli/pkg/config/process"
	"github.com/infracost/cli/pkg/stacktrace"
	"github.com/infracost/go-proto/pkg/diagnostic"
	parserpb "github.com/infracost/proto/gen/go/infracost/parser"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func main() {
	os.Exit(run())
}

func run() int {
	var diags *diagnostic.Diagnostics
	cfg := new(config.Config)
	defer func() {
		if r := recover(); r != nil {
			client := cfg.Events.Client(api.Client(context.Background(), cfg.Auth.TokenFromCache(context.Background()), cfg.OrgID))
			client.Push(context.Background(), "infracost-error", "error", r, "stacktrace", stacktrace.Sanitize(debug.Stack(), "github.com/infracost/cli/", "github.com/infracost/actions/"))
			_, _ = fmt.Fprintf(os.Stderr, "An unexpected error occurred. This is a bug in Infracost, please report it at https://github.com/infracost/infracost/issues\n\n")
			_, _ = fmt.Fprintf(os.Stderr, "panic: %v\n\n%s\n", r, debug.Stack())
			os.Exit(1)
		}
	}()

	cmd := &cobra.Command{
		Use:     "scanner",
		Version: version.Version,
		Short:   "Cloud cost estimates for IaC in your CI pipeline",
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			events.RegisterMetadata("command", cmd.Name())
			events.RegisterMetadata("flags", func() []string {
				var flags []string
				cmd.Flags().Visit(func(flag *pflag.Flag) {
					flags = append(flags, flag.Name)
				})
				return flags
			}())

			process.Process(cfg)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	var results commands.ScanResult
	cmd.AddCommand(commands.Diff(cfg, &results))
	cmd.AddCommand(commands.Status(cfg))

	diags.Merge(process.PreProcess(cfg, cmd.PersistentFlags()))
	if diags.Critical().Len() > 0 {
		for _, diag := range diags.Critical().Unwrap() {
			_, _ = fmt.Fprintf(os.Stderr, "%s\n", diag.FormatMessage())
		}
		client := cfg.Events.Client(api.Client(context.Background(), cfg.Auth.TokenFromCache(context.Background()), cfg.OrgID))
		for _, diag := range diags.Critical().Unwrap() {
			client.Push(context.Background(), "infracost-error", "error", diag.String())
		}
		return 1
	}

	if err := cmd.Execute(); err != nil {
		diags = diags.Add(diagnostic.FromError(parserpb.DiagnosticType_DIAGNOSTIC_TYPE_UNSPECIFIED, err))
	}
	if diags.Critical().Len() > 0 {
		for _, diag := range diags.Critical().Unwrap() {
			_, _ = fmt.Fprintf(os.Stderr, "%s\n", diag.FormatMessage())
		}
		client := cfg.Events.Client(api.Client(context.Background(), cfg.Auth.TokenFromCache(context.Background()), cfg.OrgID))
		for _, diag := range diags.Critical().Unwrap() {
			client.Push(context.Background(), "infracost-error", "error", diag.String())
		}
		return 1
	}

	if results.BlockPR {
		for _, reason := range results.Reasons {
			_, _ = fmt.Fprintf(os.Stderr, "Blocking: %s\n", reason)
		}
		return 1
	}

	return 0
}
