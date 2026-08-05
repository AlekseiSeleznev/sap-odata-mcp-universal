package test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestREADMEManualCrossCompilationProtectsDarwinArtifacts(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test file")
	}
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(testFile)))
	readme, err := os.ReadFile(filepath.Join(repoRoot, "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}

	sectionStart := strings.Index(string(readme), "Кросс-компиляция вручную:")
	if sectionStart < 0 {
		t.Fatal("README is missing the manual cross-compilation section")
	}
	section := string(readme)[sectionStart:]
	if next := strings.Index(section, "\n### "); next >= 0 {
		section = section[:next]
	}

	if !strings.Contains(section, "make build-macos") {
		t.Fatal("README Darwin guidance must use guarded `make build-macos`")
	}
	if strings.Contains(section, "GOOS=darwin") && strings.Contains(section, "go build") && !strings.Contains(section, "Go 1.24") {
		t.Fatal("README contains an unguarded Darwin go build without a Go 1.24 prerequisite")
	}
	for _, required := range []string{"GOOS=linux", "GOOS=windows"} {
		if !strings.Contains(section, required) {
			t.Fatalf("README lost documented %s cross-compilation guidance", required)
		}
	}
}
