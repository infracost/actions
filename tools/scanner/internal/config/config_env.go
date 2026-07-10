package config

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	pkgscanner "github.com/infracost/cli/pkg/scanner"
	repoconfig "github.com/infracost/config"
	"gopkg.in/yaml.v3"
)

// ValidateHeadConfigEnv rejects environment references that are not trusted by the base branch.
func ValidateHeadConfigEnv(ctx context.Context, baseDir, headDir, repoName string) error {
	basePath, baseTemplate := activeConfigPath(baseDir)
	headPath, headTemplate := activeConfigPath(headDir)
	if headPath == "" {
		return nil
	}

	if headTemplate {
		if basePath == "" || !baseTemplate {
			return fmt.Errorf("%s cannot be introduced in a pull request", pkgscanner.RepoConfigTemplateFilename)
		}
		base, err := os.ReadFile(basePath) // #nosec G304 -- fixed filename under the trusted checkout
		if err != nil {
			return fmt.Errorf("read base repository config template: %w", err)
		}
		head, err := os.ReadFile(headPath) // #nosec G304 -- fixed filename under the untrusted checkout
		if err != nil {
			return fmt.Errorf("read head repository config template: %w", err)
		}
		if !bytes.Equal(base, head) {
			return fmt.Errorf("%s cannot be changed in a pull request", pkgscanner.RepoConfigTemplateFilename)
		}

		baseScalars, err := templateConfigScalars(ctx, baseDir, base, repoName)
		if err != nil {
			return fmt.Errorf("render base repository config template: %w", err)
		}
		headScalars, err := templateConfigScalars(ctx, headDir, head, repoName)
		if err != nil {
			return fmt.Errorf("render head repository config template: %w", err)
		}
		return validateConfigScalars(baseScalars, headScalars)
	}

	headScalars, err := fileConfigScalars(headDir, headPath)
	if err != nil {
		return fmt.Errorf("read head repository config: %w", err)
	}
	baseScalars := map[string]configScalar{}
	if basePath != "" && !baseTemplate {
		baseScalars, err = fileConfigScalars(baseDir, basePath)
		if err != nil {
			return fmt.Errorf("read base repository config: %w", err)
		}
	}

	return validateConfigScalars(baseScalars, headScalars)
}

func templateConfigScalars(ctx context.Context, dir string, template []byte, repoName string) (map[string]configScalar, error) {
	opts := []repoconfig.GenerationOption{
		repoconfig.WithTemplate(string(template)),
		repoconfig.WithIgnorePermissionErrors(true),
		repoconfig.WithIgnoreHiddenDirs(true),
		repoconfig.WithSkipCDK(true),
	}
	if repoName != "" {
		opts = append(opts, repoconfig.WithRepoName(repoName))
	}
	cfg, err := repoconfig.Generate(ctx, dir, opts...)
	if err != nil {
		return nil, err
	}
	content, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	return configScalarsFromContent(content)
}

func fileConfigScalars(dir, path string) (map[string]configScalar, error) {
	cfg, err := repoconfig.LoadConfigFile(path, dir, nil)
	if err != nil {
		return nil, err
	}
	content, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	return configScalarsFromContent(content)
}

func validateConfigScalars(baseScalars, headScalars map[string]configScalar) error {
	for path, scalar := range headScalars {
		base := baseScalars[path]
		if hasEnvReference(scalar.value) && (base.value != scalar.value || base.context != scalar.context) {
			return fmt.Errorf("environment reference at %s must already exist unchanged on the base branch", scalar.displayPath)
		}
	}
	return nil
}

func activeConfigPath(dir string) (string, bool) {
	configPath := filepath.Join(dir, pkgscanner.RepoConfigFilename)
	if fileExistsAt(configPath) {
		return configPath, false
	}
	templatePath := filepath.Join(dir, pkgscanner.RepoConfigTemplateFilename)
	if fileExistsAt(templatePath) {
		return templatePath, true
	}
	return "", false
}

type configScalar struct {
	value       string
	displayPath string
	context     string
}

func configScalarsFromContent(content []byte) (map[string]configScalar, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(content, &root); err != nil {
		return nil, err
	}
	out := make(map[string]configScalar)
	collectScalars(&root, "", "", "", out, make(map[*yaml.Node]bool))
	return out, nil
}

func collectScalars(node *yaml.Node, path, displayPath, context string, out map[string]configScalar, stack map[*yaml.Node]bool) {
	if node == nil || stack[node] {
		return
	}
	stack[node] = true
	defer delete(stack, node)

	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			collectScalars(child, path, displayPath, context, out, stack)
		}
	case yaml.MappingNode:
		context = configNodeFingerprint(node)
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i]
			value := node.Content[i+1]
			segment := strconv.Itoa(i / 2)
			if key.Kind == yaml.ScalarNode {
				segment = escapeConfigPath(key.Value)
				out[path+"/key:"+segment] = configScalar{key.Value, displayPath + "/@key/" + segment, context}
			}
			collectScalars(value, path+"/map:"+segment, displayPath+"/"+segment, context, out, stack)
		}
	case yaml.SequenceNode:
		for i, child := range node.Content {
			segment := strconv.Itoa(i)
			collectScalars(child, path+"/sequence:"+segment, displayPath+"/"+segment, context, out, stack)
		}
	case yaml.AliasNode:
		collectScalars(node.Alias, path, displayPath, context, out, stack)
	case yaml.ScalarNode:
		out[path] = configScalar{node.Value, displayPath, context}
	}
}

func configNodeFingerprint(node *yaml.Node) string {
	content, _ := yaml.Marshal(node)
	return string(content)
}

func escapeConfigPath(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "~", "~0"), "/", "~1")
}

func hasEnvReference(value string) bool {
	found := false
	_ = os.Expand(value, func(name string) string {
		found = found || isEnvName(name)
		return ""
	})
	return found
}

func isEnvName(name string) bool {
	for i := range len(name) {
		c := name[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || (i > 0 && c >= '0' && c <= '9')) {
			return false
		}
	}
	return name != ""
}
