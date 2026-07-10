package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateHeadConfigEnv(t *testing.T) {
	tests := []struct {
		name         string
		baseConfig   string
		headConfig   string
		baseTemplate string
		headTemplate string
		baseFiles    map[string]string
		headFiles    map[string]string
		wantErr      string
	}{
		{name: "no config"},
		{
			name:       "config change without environment reference",
			baseConfig: "version: 0.3\nprojects:\n  - name: api\n    path: .\n",
			headConfig: "version: 0.3\nprojects:\n  - name: web\n    path: .\n",
		},
		{
			name:       "terraform capture group",
			baseConfig: "version: 0.3\nprojects: []\n",
			headConfig: "version: 0.3\nterraform:\n  source_map:\n    - match: example.com/(.*)\n      replace: mirror.example.com/${1}\nprojects: []\n",
		},
		{
			name:       "unchanged environment reference",
			baseConfig: "version: 0.3\nprojects:\n  - name: ${PROJECT_NAME}\n    path: .\n",
			headConfig: "projects:\n  - path: .\n    name: ${PROJECT_NAME}\nversion: 0.3\n",
		},
		{
			name:       "new environment reference",
			baseConfig: "version: 0.3\nprojects:\n  - name: api\n    path: .\n",
			headConfig: "version: 0.3\nprojects:\n  - name: ${GITHUB_TOKEN}\n    path: .\n",
			wantErr:    "environment reference at /projects/0/name must already exist unchanged",
		},
		{
			name:       "modified environment reference",
			baseConfig: "version: 0.3\nprojects:\n  - name: ${PROJECT_NAME}\n    path: .\n",
			headConfig: "version: 0.3\nprojects:\n  - name: prefix-${PROJECT_NAME}\n    path: .\n",
			wantErr:    "environment reference at /projects/0/name must already exist unchanged",
		},
		{
			name:       "moved environment reference",
			baseConfig: "version: 0.3\nprojects:\n  - name: ${PROJECT_NAME}\n    path: .\n",
			headConfig: "version: 0.3\nprojects:\n  - name: api\n    path: ${PROJECT_NAME}\n",
			wantErr:    "environment reference at /projects/0/path must already exist unchanged",
		},
		{
			name: "inherited environment reference with changed endpoint",
			baseConfig: `version: 0.3
terraform:
  defaults:
    cloud:
      host: app.terraform.io
      token: ${TFC_TOKEN}
projects:
  - name: main
    path: .
`,
			headConfig: `version: 0.3
terraform:
  defaults:
    cloud:
      host: app.terraform.io
      token: ${TFC_TOKEN}
projects:
  - name: main
    path: .
    terraform:
      cloud:
        host: attacker.example.com
`,
			wantErr: "environment reference at /projects/0/terraform/cloud/token must already exist unchanged",
		},
		{
			name:         "unchanged template",
			baseTemplate: "version: 0.3\nprojects:\n  - name: ${PROJECT_NAME}\n    path: .\n",
			headTemplate: "version: 0.3\nprojects:\n  - name: ${PROJECT_NAME}\n    path: .\n",
		},
		{
			name:         "new template",
			headTemplate: "version: 0.3\nprojects: {{ .Projects }}\n",
			wantErr:      "infracost.yml.tmpl cannot be introduced",
		},
		{
			name:         "changed template",
			baseTemplate: "version: 0.3\nprojects: []\n",
			headTemplate: "version: 0.3\nprojects: {{ .Projects }}\n",
			wantErr:      "infracost.yml.tmpl cannot be changed",
		},
		{
			name:         "inactive template change",
			baseConfig:   "version: 0.3\nprojects: []\n",
			headConfig:   "version: 0.3\nprojects: []\n",
			baseTemplate: "base",
			headTemplate: "${GITHUB_TOKEN}",
		},
		{
			name: "environment reference generated from template input",
			baseTemplate: `version: 0.3
projects:
{{- range matchPaths "environment/:env/main.tf" }}
  - path: .
    name: {{ .env }}
{{- end }}
`,
			headTemplate: `version: 0.3
projects:
{{- range matchPaths "environment/:env/main.tf" }}
  - path: .
    name: {{ .env }}
{{- end }}
`,
			baseFiles: map[string]string{"environment/dev/main.tf": ""},
			headFiles: map[string]string{"environment/${GITHUB_TOKEN}/main.tf": ""},
			wantErr:   "environment reference at /projects/0/name must already exist unchanged",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()
			headDir := t.TempDir()
			writeConfigTestFile(t, baseDir, "infracost.yml", tt.baseConfig)
			writeConfigTestFile(t, headDir, "infracost.yml", tt.headConfig)
			writeConfigTestFile(t, baseDir, "infracost.yml.tmpl", tt.baseTemplate)
			writeConfigTestFile(t, headDir, "infracost.yml.tmpl", tt.headTemplate)
			for name, content := range tt.baseFiles {
				writeConfigTestFile(t, baseDir, name, content)
			}
			for name, content := range tt.headFiles {
				writeConfigTestFile(t, headDir, name, content)
			}

			err := ValidateHeadConfigEnv(t.Context(), baseDir, headDir, "repo")
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func writeConfigTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if content == "" && (name == "infracost.yml" || name == "infracost.yml.tmpl") {
		return
	}
	require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0600))
}
