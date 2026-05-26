package sys

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// // // // // // // // // //

func TestValidateGoVersion(t *testing.T) {
	if err := ValidateGoVersion("v1.2.3"); err != nil {
		t.Fatalf("valid version rejected: %v", err)
	}
	if err := ValidateGoVersion("v1.2.3-beta.1"); err != nil {
		t.Fatalf("valid prerelease rejected: %v", err)
	}
	if err := ValidateGoVersion("1.2.3"); err == nil {
		t.Fatalf("invalid version accepted")
	}
	if err := ValidateGoVersion("v1.02.3"); err == nil {
		t.Fatalf("invalid Go semver accepted")
	}
}

func TestValidateVersion(t *testing.T) {
	validArr := []string{
		"1.2.3",
		"v1.2.3",
		"release.2.3",
		"release.2.3-alpha.1+build-9",
	}

	for _, item := range validArr {
		if err := ValidateVersion(item); err != nil {
			t.Fatalf("valid version rejected %q: %v", item, err)
		}
	}

	invalidArr := []string{
		"",
		"1.2",
		"1.two.3",
		"bad major.1.2",
		"1.2.3-",
		"1.2.3+",
		"1.2.3-alpha.!",
	}

	for _, item := range invalidArr {
		if err := ValidateVersion(item); err == nil {
			t.Fatalf("invalid version accepted %q", item)
		}
	}
}

func TestPrePatch(t *testing.T) {
	rootPath := t.TempDir()
	runPath := filepath.Join(rootPath, "_run")
	if err := os.MkdirAll(runPath, 0755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}

	manifestPath := filepath.Join(runPath, cManifestJSON)
	if err := os.WriteFile(manifestPath, []byte("{\"name\":\"demo\",\"ver\":\"release.1.2\"}\n"), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	sysObj, err := New(rootPath)
	if err != nil {
		t.Fatalf("new sys obj: %v", err)
	}

	versionValue, err := sysObj.PrePatch("")
	if err != nil {
		t.Fatalf("prepatch first: %v", err)
	}
	if versionValue != "release.1.3-0" {
		t.Fatalf("unexpected first prepatch version: %s", versionValue)
	}

	versionValue, err = sysObj.PrePatch("")
	if err != nil {
		t.Fatalf("prepatch second: %v", err)
	}
	if versionValue != "release.1.4-0" {
		t.Fatalf("unexpected second prepatch version: %s", versionValue)
	}
}

func TestCreateRejectsNameLongerThanFortyRunes(t *testing.T) {
	rootPath := t.TempDir()

	_, err := Create(rootPath, "json", "12345678901234567890123456789012345678901", true, true)
	if err == nil {
		t.Fatalf("expected error for long name")
	}
	if !strings.Contains(err.Error(), "manifest name exceeds 40 characters") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVersionIncrementSemantics(t *testing.T) {
	testCaseArr := []struct {
		name        string
		raw         string
		runFunc     func(versionObj *VersionObj) error
		expected    string
		expectedErr string
	}{
		{
			name: "major stable",
			raw:  "v1.2.3",
			runFunc: func(versionObj *VersionObj) error {
				return versionObj.IncrementMajor()
			},
			expected: "v2.0.0",
		},
		{
			name: "major prerelease same major",
			raw:  "v1.0.0-beta.1",
			runFunc: func(versionObj *VersionObj) error {
				return versionObj.IncrementMajor()
			},
			expected: "v1.0.0",
		},
		{
			name: "minor prerelease same minor",
			raw:  "v1.2.0-beta.1",
			runFunc: func(versionObj *VersionObj) error {
				return versionObj.IncrementMinor()
			},
			expected: "v1.2.0",
		},
		{
			name: "patch prerelease same patch",
			raw:  "v1.2.3-beta.1",
			runFunc: func(versionObj *VersionObj) error {
				return versionObj.IncrementPatch()
			},
			expected: "v1.2.3",
		},
		{
			name: "premajor stable",
			raw:  "v1.2.3",
			runFunc: func(versionObj *VersionObj) error {
				return versionObj.PreMajor("")
			},
			expected: "v2.0.0-0",
		},
		{
			name: "preminor stable",
			raw:  "v1.2.3",
			runFunc: func(versionObj *VersionObj) error {
				return versionObj.PreMinor("")
			},
			expected: "v1.3.0-0",
		},
		{
			name: "prepatch stable",
			raw:  "v1.2.3",
			runFunc: func(versionObj *VersionObj) error {
				return versionObj.PrePatch("")
			},
			expected: "v1.2.4-0",
		},
		{
			name: "prepatch prerelease bumps patch",
			raw:  "v1.2.4-beta.1",
			runFunc: func(versionObj *VersionObj) error {
				return versionObj.PrePatch("")
			},
			expected: "v1.2.5-0",
		},
		{
			name: "prerelease stable",
			raw:  "v1.2.3",
			runFunc: func(versionObj *VersionObj) error {
				return versionObj.IncrementPrerelease("")
			},
			expected: "v1.2.4-0",
		},
		{
			name: "prerelease existing numeric tail",
			raw:  "v1.2.3-beta.1",
			runFunc: func(versionObj *VersionObj) error {
				return versionObj.IncrementPrerelease("")
			},
			expected: "v1.2.3-beta.2",
		},
		{
			name: "prerelease existing text tail",
			raw:  "v1.2.3-beta.foo",
			runFunc: func(versionObj *VersionObj) error {
				return versionObj.IncrementPrerelease("")
			},
			expected: "v1.2.3-beta.foo.0",
		},
		{
			name: "prerelease stable with preid",
			raw:  "v1.2.3",
			runFunc: func(versionObj *VersionObj) error {
				return versionObj.IncrementPrerelease("beta")
			},
			expected: "v1.2.4-beta.0",
		},
		{
			name: "prerelease same preid increments",
			raw:  "v1.2.3-beta.1",
			runFunc: func(versionObj *VersionObj) error {
				return versionObj.IncrementPrerelease("beta")
			},
			expected: "v1.2.3-beta.2",
		},
		{
			name: "prerelease different preid resets",
			raw:  "v1.2.3-beta.1",
			runFunc: func(versionObj *VersionObj) error {
				return versionObj.IncrementPrerelease("alpha")
			},
			expected: "v1.2.3-alpha.0",
		},
		{
			name: "major non incrementable",
			raw:  "release.2.3",
			runFunc: func(versionObj *VersionObj) error {
				return versionObj.IncrementMajor()
			},
			expectedErr: "version major is not incrementable",
		},
		{
			name: "premajor non incrementable",
			raw:  "release.2.3",
			runFunc: func(versionObj *VersionObj) error {
				return versionObj.PreMajor("beta")
			},
			expectedErr: "version major is not incrementable",
		},
	}

	for _, testCaseObj := range testCaseArr {
		t.Run(testCaseObj.name, func(t *testing.T) {
			versionObj, err := ParseVersion(testCaseObj.raw)
			if err != nil {
				t.Fatalf("parse version: %v", err)
			}

			err = testCaseObj.runFunc(versionObj)
			if testCaseObj.expectedErr != "" {
				if err == nil {
					t.Fatalf("expected error")
				}
				if !strings.Contains(err.Error(), testCaseObj.expectedErr) {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("run increment: %v", err)
			}
			if versionObj.String() != testCaseObj.expected {
				t.Fatalf("unexpected version: %s", versionObj.String())
			}
		})
	}
}
