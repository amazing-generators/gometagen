package git

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// // // // // // // // // //

func TestGitOperationsAndHooks(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			t.Skip("git is not installed")
		}
		t.Fatalf("lookup git: %v", err)
	}

	rootPath := t.TempDir()

	runGit(t, rootPath, "init")
	runGit(t, rootPath, "config", "user.email", "tests@example.com")
	runGit(t, rootPath, "config", "user.name", "Tests")

	filePath := filepath.Join(rootPath, "README.md")
	if err := os.WriteFile(filePath, []byte("test\n"), 0644); err != nil {
		t.Fatalf("write readme: %v", err)
	}

	runGit(t, rootPath, "add", "README.md")
	runGit(t, rootPath, "commit", "-m", "init")
	runGit(t, rootPath, "tag", "v0.1.0")

	gitObj, err := New(rootPath)
	if err != nil {
		t.Fatalf("new git obj: %v", err)
	}

	branchValue, err := gitObj.Branch()
	if err != nil {
		t.Fatalf("read branch: %v", err)
	}
	if branchValue == "" {
		t.Fatalf("branch must not be empty")
	}

	hashValue, err := gitObj.Hash()
	if err != nil {
		t.Fatalf("read hash: %v", err)
	}
	if len(hashValue) != 40 {
		t.Fatalf("unexpected hash length: %d", len(hashValue))
	}

	tagValue, err := gitObj.Tag()
	if err != nil {
		t.Fatalf("read tag: %v", err)
	}
	if tagValue != "v0.1.0" {
		t.Fatalf("unexpected tag: %s", tagValue)
	}

	if err = gitObj.AddCommitHook(); err != nil {
		t.Fatalf("add commit hook: %v", err)
	}
	if err = gitObj.AddPushHook(); err != nil {
		t.Fatalf("add push hook: %v", err)
	}

	hooksPath, err := gitObj.HooksPath()
	if err != nil {
		t.Fatalf("hooks path: %v", err)
	}

	commitHookArr, err := os.ReadFile(filepath.Join(hooksPath, "commit-msg"))
	if err != nil {
		t.Fatalf("read commit hook: %v", err)
	}
	if string(commitHookArr) != string(CommitMsgHookArr) {
		t.Fatalf("unexpected commit hook content")
	}

	pushHookArr, err := os.ReadFile(filepath.Join(hooksPath, "pre-push"))
	if err != nil {
		t.Fatalf("read push hook: %v", err)
	}
	if string(pushHookArr) != string(PrePushHookArr) {
		t.Fatalf("unexpected push hook content")
	}

	if err = gitObj.DelCommitHook(); err != nil {
		t.Fatalf("delete commit hook: %v", err)
	}
	if err = gitObj.DelPushHook(); err != nil {
		t.Fatalf("delete push hook: %v", err)
	}
}

func runGit(t *testing.T, rootPath string, argsArr ...string) {
	t.Helper()

	commandObj := exec.Command("git", argsArr...)
	commandObj.Dir = rootPath
	dataArr, err := commandObj.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v: %s", argsArr, err, string(dataArr))
	}
}
