package wsdlbundle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestFilePermitLedgerConsumesExactlyOnceUnderRace(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatalf("chmod ledger: %v", err)
	}
	now := time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)
	input := Input{
		SystemID:              SystemID,
		ContractID:            ContractID,
		RequestManifestSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		PermitID:              "6ba7b810-9dad-41d1-80b4-00c04fd430c8",
	}
	permit := Permit{
		SchemaVersion:         1,
		PermitID:              input.PermitID,
		Purpose:               "WSDL_BUNDLE_READ",
		SystemID:              input.SystemID,
		ContractID:            input.ContractID,
		RequestManifestSHA256: input.RequestManifestSHA256,
		BinarySHA256:          "abcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd",
		NotBefore:             now.Add(-time.Minute),
		ExpiresAt:             now.Add(time.Minute),
	}
	body, err := json.Marshal(permit)
	if err != nil {
		t.Fatalf("encode permit: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, input.PermitID+".json"), body, 0o600); err != nil {
		t.Fatalf("write permit: %v", err)
	}

	ledger := &FilePermitLedger{Dir: dir, Now: func() time.Time { return now }, BinarySHA256: permit.BinarySHA256}
	var successes atomic.Int32
	var wait sync.WaitGroup
	for i := 0; i < 24; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if err := ledger.Consume(input); err == nil {
				successes.Add(1)
			}
		}()
	}
	wait.Wait()
	if successes.Load() != 1 {
		t.Fatalf("permit succeeded %d times, want exactly one", successes.Load())
	}
	if err := ledger.Consume(input); err == nil {
		t.Fatal("replayed permit unexpectedly succeeded")
	}
	marker := filepath.Join(dir, input.PermitID+".consumed")
	info, err := os.Stat(marker)
	if err != nil {
		t.Fatalf("consumed marker missing: %v", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("consumed marker permissions are too broad: %o", info.Mode().Perm())
	}
}

func TestFilePermitLedgerRejectsMismatchWithoutConsumption(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatalf("chmod ledger: %v", err)
	}
	now := time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)
	input := Input{
		SystemID:              SystemID,
		ContractID:            ContractID,
		RequestManifestSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		PermitID:              "6ba7b810-9dad-41d1-80b4-00c04fd430c8",
	}
	permit := Permit{
		SchemaVersion:         1,
		PermitID:              input.PermitID,
		Purpose:               "OTHER_PURPOSE",
		SystemID:              input.SystemID,
		ContractID:            input.ContractID,
		RequestManifestSHA256: input.RequestManifestSHA256,
		BinarySHA256:          "abcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd",
		NotBefore:             now.Add(-time.Minute),
		ExpiresAt:             now.Add(time.Minute),
	}
	body, _ := json.Marshal(permit)
	if err := os.WriteFile(filepath.Join(dir, input.PermitID+".json"), body, 0o600); err != nil {
		t.Fatalf("write permit: %v", err)
	}
	ledger := &FilePermitLedger{Dir: dir, Now: func() time.Time { return now }, BinarySHA256: permit.BinarySHA256}
	if err := ledger.Consume(input); err == nil {
		t.Fatal("mismatched permit unexpectedly succeeded")
	}
	if _, err := os.Stat(filepath.Join(dir, input.PermitID+".consumed")); !os.IsNotExist(err) {
		t.Fatalf("mismatched permit was consumed: %v", err)
	}
}
