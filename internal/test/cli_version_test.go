package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestCLIVersionIsStandaloneAndDeterministic(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test file")
	}
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(testFile)))
	cmd := exec.Command("go", "run", "./cmd/sap-odata-mcp-universal", "--version")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "ODATA_SERVICE_URL=http://127.0.0.1:1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--version failed: %v; output=%q", err, output)
	}
	if !regexp.MustCompile(`^[^\n]+ \(commit [^\n]+, built [^\n]+\)$`).MatchString(strings.TrimSpace(string(output))) {
		t.Fatalf("unexpected version output: %q", output)
	}
}
