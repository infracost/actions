package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildRunInput_Metadata_CIPlatformAndVCSProvider(t *testing.T) {
	tests := []struct {
		name        string
		opts        RunInputOptions
		ciPlatform  string
		vcsProvider string
	}{
		{
			name:        "github actions",
			opts:        RunInputOptions{CIPlatform: "github_actions", VCSProvider: "github"},
			ciPlatform:  "github_actions",
			vcsProvider: "github",
		},
		{
			name:        "namespaces are independent",
			opts:        RunInputOptions{CIPlatform: "gitlab_ci", VCSProvider: "gitlab"},
			ciPlatform:  "gitlab_ci",
			vcsProvider: "gitlab",
		},
		{
			name:        "unrecognised platform",
			opts:        RunInputOptions{CIPlatform: "unknown", VCSProvider: "bitbucket"},
			ciPlatform:  "unknown",
			vcsProvider: "bitbucket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := BuildRunInput(tt.opts)

			assert.Equal(t, tt.ciPlatform, input.Metadata["ciPlatform"])
			assert.Equal(t, tt.vcsProvider, input.Metadata["vcsProvider"])
		})
	}
}

func TestBuildErrorRunInput_Metadata_CIPlatformAndVCSProvider(t *testing.T) {
	input := BuildErrorRunInput(
		RunInputOptions{CIPlatform: "gitlab_ci", VCSProvider: "gitlab"},
		0, "Failed to scan", "boom",
	)

	assert.Equal(t, "gitlab_ci", input.Metadata["ciPlatform"])
	assert.Equal(t, "gitlab", input.Metadata["vcsProvider"])
}
