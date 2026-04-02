package git

import (
	"os/exec"
	"strings"
)

type CommitInfo struct {
	Message     string
	AuthorName  string
	AuthorEmail string
	Timestamp   string
}

func GetCommitInfo(dir, sha string) CommitInfo {
	if sha == "" {
		return CommitInfo{}
	}
	return CommitInfo{
		Message:     Log(dir, sha, "%s"),
		AuthorName:  Log(dir, sha, "%aN"),
		AuthorEmail: Log(dir, sha, "%aE"),
		Timestamp:   Log(dir, sha, "%aI"),
	}
}

func RevParse(dir string, args ...string) string {
	cmd := exec.Command("git", append([]string{"rev-parse"}, args...)...) // #nosec G204 -- args are controlled by caller
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// HasChanges reports whether any tracked files differ between the same
// relative path in two separate checkout directories. This works with
// shallow clones since it compares working trees rather than commit history.
func HasChanges(baseDir, headDir, path string) bool {
	basePath := baseDir
	headPath := headDir
	if path != "" && path != "." {
		basePath = basePath + "/" + path
		headPath = headPath + "/" + path
	}
	cmd := exec.Command("git", "diff", "--no-index", "--quiet", basePath, headPath) // #nosec G204 -- paths are controlled by caller
	return cmd.Run() != nil
}

func Log(dir, sha, format string) string {
	cmd := exec.Command("git", "log", "-1", "--format="+format, sha) // #nosec G204 -- args are controlled by caller
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}