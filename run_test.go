package gometagen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// // // // // // // // // //

func TestRunWithContentHashAndFormats(t *testing.T) {
	rootPath := t.TempDir()
	runPath := filepath.Join(rootPath, "_run")
	dataPath := filepath.Join(rootPath, "data")

	if err := os.MkdirAll(runPath, 0755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}

	if err := os.WriteFile(filepath.Join(runPath, "values.yml"), []byte("name: demo-project\nver: v1.2.3\n"), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataPath, "a.txt"), []byte("alpha\n"), 0644); err != nil {
		t.Fatalf("write data file: %v", err)
	}

	nowValue := time.Date(2026, time.May, 26, 15, 4, 5, 0, time.UTC)

	goResultObj, err := Run(ConfigObj{
		Source:        rootPath,
		Format:        "go",
		Stdout:        true,
		HashSourceArr: []string{dataPath},
		Now:           nowValue,
	})
	if err != nil {
		t.Fatalf("run go: %v", err)
	}

	goOutput := string(goResultObj.Data)
	if !strings.Contains(goOutput, `Name`) || !strings.Contains(goOutput, `"demo-project"`) {
		t.Fatalf("unexpected go output: %s", goOutput)
	}
	if !strings.Contains(goOutput, `VersionMajor`) || !strings.Contains(goOutput, `"v1"`) {
		t.Fatalf("unexpected go version major: %s", goOutput)
	}
	if !strings.Contains(goOutput, `// Hash point: `+dataPath) {
		t.Fatalf("missing go hash point comment: %s", goOutput)
	}
	if !strings.Contains(goOutput, `// Generated at: 2026-05-26T15:04:05Z`) {
		t.Fatalf("missing go generated at comment: %s", goOutput)
	}

	jsonResultObj, err := Run(ConfigObj{
		Source:        filepath.Join(runPath, "values.yml"),
		Format:        "json",
		Stdout:        true,
		HashSourceArr: []string{dataPath},
		Now:           nowValue,
	})
	if err != nil {
		t.Fatalf("run json: %v", err)
	}

	jsonOutput := string(jsonResultObj.Data)
	if !strings.Contains(jsonOutput, `"date_update": "2026-05-26"`) {
		t.Fatalf("unexpected json output: %s", jsonOutput)
	}
	if strings.Contains(jsonOutput, `"generated_at"`) || strings.Contains(jsonOutput, `"hash_point"`) {
		t.Fatalf("json output must not expose generated_at/hash_point: %s", jsonOutput)
	}

	templatePath := filepath.Join(rootPath, "custom.tmpl")
	if err = os.WriteFile(templatePath, []byte("{{ .Meta.Name }}|{{ .Meta.Version }}|{{ .HashMode }}\n"), 0644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	templateResultObj, err := Run(ConfigObj{
		Source:        rootPath,
		TemplateFile:  templatePath,
		Stdout:        true,
		HashSourceArr: []string{dataPath},
		Now:           nowValue,
	})
	if err != nil {
		t.Fatalf("run custom template: %v", err)
	}

	if string(templateResultObj.Data) != "demo-project|v1.2.3|content\n" {
		t.Fatalf("unexpected template output: %q", string(templateResultObj.Data))
	}
}

func TestRunWithGitHash(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	rootPath := t.TempDir()
	runPath := filepath.Join(rootPath, "_run")

	if err := os.MkdirAll(runPath, 0755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}

	if err := os.WriteFile(filepath.Join(runPath, "values.yml"), []byte("name: demo\nver: 1.2.3\n"), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	runGit(t, rootPath, "init")
	runGit(t, rootPath, "config", "user.email", "tests@example.com")
	runGit(t, rootPath, "config", "user.name", "Tests")
	runGit(t, rootPath, "add", ".")
	runGit(t, rootPath, "commit", "-m", "init")

	resultObj, err := Run(ConfigObj{
		Source: rootPath,
		Format: "yaml",
		Stdout: true,
	})
	if err != nil {
		t.Fatalf("run yaml: %v", err)
	}

	if !strings.Contains(string(resultObj.Data), "hash: '") {
		t.Fatalf("unexpected yaml output: %s", string(resultObj.Data))
	}
	if !strings.Contains(string(resultObj.Data), "# Hash point: "+rootPath) {
		t.Fatalf("missing yaml hash point comment: %s", string(resultObj.Data))
	}
	if !strings.Contains(string(resultObj.Data), "# Generated at: ") {
		t.Fatalf("missing yaml generated at comment: %s", string(resultObj.Data))
	}
}

func TestRunTemplateRequiresExplicitOutputFile(t *testing.T) {
	rootPath := t.TempDir()
	runPath := filepath.Join(rootPath, "_run")

	if err := os.MkdirAll(runPath, 0755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}

	if err := os.WriteFile(filepath.Join(runPath, "values.yml"), []byte("name: demo-project\nver: v1.2.3\n"), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	templatePath := filepath.Join(rootPath, "custom.tmpl")
	if err := os.WriteFile(templatePath, []byte("{{ .Meta.Name }}\n"), 0644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	_, err := Run(ConfigObj{
		Source:       rootPath,
		TemplateFile: templatePath,
	})
	if err == nil || !strings.Contains(err.Error(), "output file is required for custom template") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunTemplateRejectsDirectoryOutput(t *testing.T) {
	rootPath := t.TempDir()
	runPath := filepath.Join(rootPath, "_run")
	outputPath := filepath.Join(rootPath, "output")

	if err := os.MkdirAll(runPath, 0755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		t.Fatalf("mkdir output: %v", err)
	}

	if err := os.WriteFile(filepath.Join(runPath, "values.yml"), []byte("name: demo-project\nver: v1.2.3\n"), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	templatePath := filepath.Join(rootPath, "custom.tmpl")
	if err := os.WriteFile(templatePath, []byte("{{ .Meta.Name }}\n"), 0644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	_, err := Run(ConfigObj{
		Source:       rootPath,
		TemplateFile: templatePath,
		OutputFile:   outputPath,
	})
	if err == nil || !strings.Contains(err.Error(), "custom template output must be an explicit file path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunRejectsFormatTemplateConflict(t *testing.T) {
	rootPath := t.TempDir()
	runPath := filepath.Join(rootPath, "_run")

	if err := os.MkdirAll(runPath, 0755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}

	if err := os.WriteFile(filepath.Join(runPath, "values.yml"), []byte("name: demo-project\nver: v1.2.3\n"), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	templatePath := filepath.Join(rootPath, "custom.tmpl")
	if err := os.WriteFile(templatePath, []byte("{{ .Meta.Name }}\n"), 0644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	_, err := Run(ConfigObj{
		Source:       rootPath,
		Format:       "go",
		TemplateFile: templatePath,
		Stdout:       true,
	})
	if err == nil || !strings.Contains(err.Error(), "conflicts with -template flag") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunRejectsNameLongerThanFortyRunes(t *testing.T) {
	rootPath := t.TempDir()
	runPath := filepath.Join(rootPath, "_run")

	if err := os.MkdirAll(runPath, 0755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(runPath, "values.yml"),
		[]byte("name: 12345678901234567890123456789012345678901\nver: v1.2.3\n"),
		0644,
	); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, err := Run(ConfigObj{
		Source: rootPath,
		Format: "json",
		Stdout: true,
	})
	if err == nil || !strings.Contains(err.Error(), "manifest name exceeds 40 characters") {
		t.Fatalf("unexpected error: %v", err)
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
