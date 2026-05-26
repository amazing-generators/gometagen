package gometagen

import (
	"context"
	"crypto/sha1"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// // // // // // // // // //

const cContentHashHexLen = sha1.Size * 2

// //

func TestHashPathChangesWhenContentChanges(t *testing.T) {
	rootPath := t.TempDir()
	filePath := filepath.Join(rootPath, "a.txt")

	if err := os.WriteFile(filePath, []byte("alpha\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	firstHashValue, err := hashPath(rootPath)
	if err != nil {
		t.Fatalf("first hash: %v", err)
	}

	if err = os.WriteFile(filePath, []byte("beta\n"), 0644); err != nil {
		t.Fatalf("rewrite file: %v", err)
	}

	secondHashValue, err := hashPath(rootPath)
	if err != nil {
		t.Fatalf("second hash: %v", err)
	}

	if firstHashValue == secondHashValue {
		t.Fatalf("hash must change when content changes")
	}
}

func TestHashPathsChangesWhenSecondSourceChanges(t *testing.T) {
	firstRootPath := t.TempDir()
	secondRootPath := t.TempDir()

	if err := os.WriteFile(filepath.Join(firstRootPath, "a.txt"), []byte("alpha\n"), 0644); err != nil {
		t.Fatalf("write first file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(secondRootPath, "b.txt"), []byte("beta\n"), 0644); err != nil {
		t.Fatalf("write second file: %v", err)
	}

	firstHashValue, err := hashPaths(context.Background(), []string{firstRootPath, secondRootPath}, nil, false)
	if err != nil {
		t.Fatalf("first hash: %v", err)
	}

	if err = os.WriteFile(filepath.Join(secondRootPath, "b.txt"), []byte("gamma\n"), 0644); err != nil {
		t.Fatalf("rewrite second file: %v", err)
	}

	secondHashValue, err := hashPaths(context.Background(), []string{firstRootPath, secondRootPath}, nil, false)
	if err != nil {
		t.Fatalf("second hash: %v", err)
	}

	if firstHashValue == secondHashValue {
		t.Fatalf("hash must change when one source changes")
	}
}

func TestHashPathKeepsFixedHexLength(t *testing.T) {
	rootPath := t.TempDir()
	filePath := filepath.Join(rootPath, "a.txt")

	if err := os.WriteFile(filePath, []byte("alpha\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	hashValue, err := hashPath(rootPath)
	if err != nil {
		t.Fatalf("hash path: %v", err)
	}

	if len(hashValue) != cContentHashHexLen {
		t.Fatalf("unexpected hash length: %d", len(hashValue))
	}
}

func TestHashPathIgnoresDefaultExcludedDirectories(t *testing.T) {
	rootPath := t.TempDir()

	if err := os.MkdirAll(filepath.Join(rootPath, "tmp"), 0755); err != nil {
		t.Fatalf("mkdir tmp: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootPath, "keep.txt"), []byte("alpha\n"), 0644); err != nil {
		t.Fatalf("write keep file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootPath, "tmp", "ignore.txt"), []byte("beta\n"), 0644); err != nil {
		t.Fatalf("write ignored file: %v", err)
	}

	firstHashValue, err := hashPath(rootPath)
	if err != nil {
		t.Fatalf("first hash: %v", err)
	}

	if err = os.WriteFile(filepath.Join(rootPath, "tmp", "ignore.txt"), []byte("gamma\n"), 0644); err != nil {
		t.Fatalf("rewrite ignored file: %v", err)
	}

	secondHashValue, err := hashPath(rootPath)
	if err != nil {
		t.Fatalf("second hash: %v", err)
	}

	if firstHashValue != secondHashValue {
		t.Fatalf("default excluded directory must not affect hash")
	}
}

func TestHashPathsIgnoresGlobExcludedFiles(t *testing.T) {
	rootPath := t.TempDir()

	if err := os.WriteFile(filepath.Join(rootPath, "keep.txt"), []byte("alpha\n"), 0644); err != nil {
		t.Fatalf("write keep file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootPath, "skip.exe"), []byte("beta\n"), 0644); err != nil {
		t.Fatalf("write excluded file: %v", err)
	}

	firstHashValue, err := hashPaths(context.Background(), []string{rootPath}, []string{"*.exe"}, false)
	if err != nil {
		t.Fatalf("first hash: %v", err)
	}

	if err = os.WriteFile(filepath.Join(rootPath, "skip.exe"), []byte("gamma\n"), 0644); err != nil {
		t.Fatalf("rewrite excluded file: %v", err)
	}

	secondHashValue, err := hashPaths(context.Background(), []string{rootPath}, []string{"*.exe"}, false)
	if err != nil {
		t.Fatalf("second hash: %v", err)
	}

	if firstHashValue != secondHashValue {
		t.Fatalf("excluded glob file must not affect hash")
	}
}

func TestHashPathsStopsOnCanceledContext(t *testing.T) {
	rootPath := t.TempDir()

	if err := os.WriteFile(filepath.Join(rootPath, "a.txt"), []byte("alpha\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	contextObj, cancelFunc := context.WithCancel(context.Background())
	cancelFunc()

	_, err := hashPaths(contextObj, []string{rootPath}, nil, false)
	if err == nil {
		t.Fatalf("expected context error")
	}
}

func TestHashPathsRejectsEmptyTree(t *testing.T) {
	rootPath := t.TempDir()

	_, err := hashPaths(context.Background(), []string{rootPath}, nil, false)
	if err == nil || !strings.Contains(err.Error(), "hash source contains no files") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHashPathsSequentialMatchesAsync(t *testing.T) {
	rootPath := t.TempDir()

	if err := os.WriteFile(filepath.Join(rootPath, "a.txt"), []byte("alpha\n"), 0644); err != nil {
		t.Fatalf("write first file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootPath, "b.txt"), []byte("beta\n"), 0644); err != nil {
		t.Fatalf("write second file: %v", err)
	}

	asyncHashValue, err := hashPaths(context.Background(), []string{rootPath}, nil, false)
	if err != nil {
		t.Fatalf("async hash: %v", err)
	}

	serialHashValue, err := hashPaths(context.Background(), []string{rootPath}, nil, true)
	if err != nil {
		t.Fatalf("serial hash: %v", err)
	}

	if asyncHashValue != serialHashValue {
		t.Fatalf("hash mismatch: async=%s serial=%s", asyncHashValue, serialHashValue)
	}
}
