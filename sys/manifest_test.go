package sys

import (
	"os"
	"path/filepath"
	"testing"
)

// // // // // // // // // //

func TestNewResolvesManifestFromRootDir(t *testing.T) {
	rootPath := t.TempDir()
	runPath := filepath.Join(rootPath, "_run")
	if err := os.MkdirAll(runPath, 0755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}

	manifestPath := filepath.Join(runPath, cManifestYML)
	if err := os.WriteFile(manifestPath, []byte("name: demo\nver: v1.2.3\n"), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	sysObj, err := New(rootPath)
	if err != nil {
		t.Fatalf("new sys obj: %v", err)
	}

	if sysObj.ManifestPath() != manifestPath {
		t.Fatalf("unexpected manifest path: %s", sysObj.ManifestPath())
	}
}

func TestNewResolvesExplicitManifestFile(t *testing.T) {
	rootPath := t.TempDir()
	manifestPath := filepath.Join(rootPath, "values.json")
	if err := os.WriteFile(manifestPath, []byte("{\"name\":\"demo\",\"ver\":\"1.2.3\"}\n"), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	sysObj, err := New(manifestPath)
	if err != nil {
		t.Fatalf("new sys obj: %v", err)
	}

	if sysObj.ManifestPath() != manifestPath {
		t.Fatalf("unexpected manifest path: %s", sysObj.ManifestPath())
	}
}

func TestCreateManifestUnderProjectRoot(t *testing.T) {
	rootPath := t.TempDir()

	manifestPath, err := Create(rootPath, "json", "demo", true, true)
	if err != nil {
		t.Fatalf("create manifest: %v", err)
	}

	expectedPath := filepath.Join(rootPath, "_run", cManifestJSON)
	if manifestPath != expectedPath {
		t.Fatalf("unexpected manifest path: %s", manifestPath)
	}

	dataArr, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	if string(dataArr) != "{\n  \"name\": \"demo\",\n  \"ver\": \"v0.0.1\"\n}\n" {
		t.Fatalf("unexpected manifest content: %q", string(dataArr))
	}
}
