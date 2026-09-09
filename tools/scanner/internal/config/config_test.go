package config

import (
	"testing"

	"github.com/infracost/cli/pkg/config/process"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func TestConfig_Process(t *testing.T) {
	var cfg Config

	flags := pflag.NewFlagSet("", pflag.ContinueOnError)

	// first, make sure that preprocess doesn't error or panic when no values provided.
	if diags := process.PreProcess(&cfg, flags); diags.Len() != 0 {
		t.Fatal(diags)
	}
	require.NoError(t, flags.Parse(nil)) // we have no required flags yet, so will provide nothing
	process.Process(&cfg)                // make sure doesn't panic
}

// The action's status step passes the provider by environment variable only,
// so the env tag is the whole contract.
func TestConfig_VCSProviderFromEnv(t *testing.T) {
	t.Setenv("INFRACOST_VCS_PROVIDER", "gitlab")

	var cfg Config
	flags := pflag.NewFlagSet("", pflag.ContinueOnError)
	if diags := process.PreProcess(&cfg, flags); diags.Len() != 0 {
		t.Fatal(diags)
	}
	require.NoError(t, flags.Parse(nil))
	process.Process(&cfg)

	require.Equal(t, "gitlab", cfg.VCSProvider)
}

func TestConfig_VCSProviderFlagOverridesEnv(t *testing.T) {
	t.Setenv("INFRACOST_VCS_PROVIDER", "gitlab")

	var cfg Config
	flags := pflag.NewFlagSet("", pflag.ContinueOnError)
	if diags := process.PreProcess(&cfg, flags); diags.Len() != 0 {
		t.Fatal(diags)
	}
	require.NoError(t, flags.Parse([]string{"--vcs-provider", "bitbucket"}))
	process.Process(&cfg)

	require.Equal(t, "bitbucket", cfg.VCSProvider)
}
