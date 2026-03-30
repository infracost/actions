package events

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/Masterminds/semver/v3"

	"github.com/infracost/cli/version"
)

var metadata map[string]interface{}

func init() {
	metadata = map[string]interface{}{
		"caller":      getCaller(),
		"ciPlatform":  getCIPlatform(),
		"cliPlatform": os.Getenv("INFRACOST_CLI_PLATFORM"),
		"version":     stripVersion(version.Version),
		"fullVersion": version.Version,
		"isDev":       version.Version == "dev",
		"isTest":      strings.HasSuffix(os.Args[0], ".test"),
		"os":          runtime.GOOS,
		"arch":        runtime.GOARCH,
	}
}

// RegisterMetadata adds or updates entries in the global event metadata.
// Other packages should call this during initialization to attach metadata
// that will be included with every event.
func RegisterMetadata(key string, value interface{}) {
	metadata[key] = value
}

// GetMetadata retrieves the value for the specified metadata, and false if it doesn't
// exist or the type is wrong.
func GetMetadata[V any](key string) (V, bool) {
	value, ok := metadata[key]
	if !ok {
		var v V
		return v, false
	}
	v, ok := value.(V)
	return v, ok
}

func stripVersion(v string) string {
	parsed, err := semver.NewVersion(v)
	if err != nil {
		return v
	}
	return fmt.Sprintf("%d.%d.%d", parsed.Major(), parsed.Minor(), parsed.Patch())
}

func getCaller() string {
	if caller, ok := os.LookupEnv("INFRACOST_CLI_CALLER"); ok {
		// if this has been explicitly set, then use that.
		return caller
	}

	// Check for known AI agent environment variables.
	for env, caller := range map[string]string{
		"CLAUDE_CODE":            "claude-code",
		"CLAUDE_CODE_ENTRYPOINT": "claude-code",
		"CLAUDECODE":             "claude-code",
		"CODEX_CLI_INVOKED_BY":   "codex",
		"GEMINI_CLI":             "gemini-cli",
		"AIDER":                  "aider",
		"CONTINUE_GLOBAL_DIR":    "continue",
		"CLINE_TASK_ID":          "cline",
		"CURSOR_TRACE_ID":        "cursor",
	} {
		if _, ok := os.LookupEnv(env); ok {
			return caller
		}
	}

	// Check TERM_PROGRAM for AI-powered IDEs. This isn't a perfect signal since
	// the user could be typing manually in the IDE's terminal, but it's a
	// reasonable heuristic.
	switch os.Getenv("TERM_PROGRAM") {
	case "cursor":
		return "cursor"
	case "windsurf":
		return "windsurf"
	}

	return ""
}

func getCIPlatform() string {
	if ciPlatform, ok := os.LookupEnv("INFRACOST_CI_PLATFORM"); ok {
		return ciPlatform
	}

	for env, platform := range map[string]string{
		"GITHUB_ACTIONS":      "github_actions",
		"GITLAB_CI":           "gitlab_ci",
		"CIRCLECI":            "circleci",
		"JENKINS_HOME":        "jenkins",
		"BUILDKITE":           "buildkite",
		"TFC_RUN_ID":          "tfc",
		"ENV0_ENVIRONMENT_ID": "env0",
		"SCALR_RUN_ID":        "scalr",
		"CF_BUILD_ID":         "codefresh",
		"TRAVIS":              "travis",
		"CODEBUILD_CI":        "codebuild",
		"TEAMCITY_VERSION":    "teamcity",
		"BUDDYBUILD_BRANCH":   "buddybuild",
		"BITRISE_IO":          "bitrise",
		"SEMAPHORE":           "semaphoreci",
		"APPVEYOR":            "appveyor",
		"WERCKER_GIT_BRANCH":  "wercker",
		"MAGNUM":              "magnumci",
		"SHIPPABLE":           "shippable",
		"TDDIUM":              "tddium",
		"GREENHOUSE":          "greenhouse",
		"CIRRUS_CI":           "cirrusci",
		"TS_ENV":              "terraspace",
	} {
		if _, ok := os.LookupEnv(env); ok {
			return platform
		}
	}

	// Azure DevOps uses a dynamic platform name based on the repository provider.
	if _, ok := os.LookupEnv("SYSTEM_COLLECTIONURI"); ok {
		return fmt.Sprintf("azure_devops_%s", os.Getenv("BUILD_REPOSITORY_PROVIDER"))
	}

	for prefix, platform := range map[string]string{
		"ATLANTIS_":       "atlantis",
		"BITBUCKET_":      "bitbucket",
		"CONCOURSE_":      "concourse",
		"SPACELIFT_":      "spacelift",
		"HARNESS_":        "harness",
		"TERRATEAM_":      "terrateam",
		"KEPTN_":          "keptn",
		"CLOUDCONCIERGE_": "cloudconcierge",
	} {
		for _, k := range os.Environ() {
			if strings.HasPrefix(k, prefix) {
				return platform
			}
		}
	}

	if ciPlatform, ok := os.LookupEnv("CI"); ok {
		return ciPlatform
	}

	return ""
}
