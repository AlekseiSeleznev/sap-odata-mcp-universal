package wsdlbundle

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const permitPurpose = "WSDL_BUNDLE_READ"

type Permit struct {
	SchemaVersion         int       `json:"schema_version"`
	PermitID              string    `json:"permit_id"`
	Purpose               string    `json:"purpose"`
	SystemID              string    `json:"system_id"`
	ContractID            string    `json:"contract_id"`
	RequestManifestSHA256 string    `json:"request_manifest_sha256"`
	BinarySHA256          string    `json:"binary_sha256"`
	NotBefore             time.Time `json:"not_before"`
	ExpiresAt             time.Time `json:"expires_at"`
}

type FilePermitLedger struct {
	Dir          string
	Now          func() time.Time
	BinarySHA256 string
}

func (l *FilePermitLedger) Consume(input Input) error {
	if l == nil || l.Dir == "" || !lowerSHA256Pattern.MatchString(l.BinarySHA256) {
		return fmt.Errorf("PERMIT_CONFIG_INVALID")
	}
	now := time.Now().UTC()
	if l.Now != nil {
		now = l.Now().UTC()
	}
	permit, err := readPermit(filepath.Join(l.Dir, input.PermitID+".json"))
	if err != nil {
		return err
	}
	if permit.SchemaVersion != 1 ||
		permit.PermitID != input.PermitID ||
		permit.Purpose != permitPurpose ||
		permit.SystemID != input.SystemID ||
		permit.ContractID != input.ContractID ||
		permit.RequestManifestSHA256 != input.RequestManifestSHA256 ||
		permit.BinarySHA256 != l.BinarySHA256 ||
		now.Before(permit.NotBefore) || !now.Before(permit.ExpiresAt) {
		return fmt.Errorf("PERMIT_MISMATCH")
	}

	markerPath := filepath.Join(l.Dir, input.PermitID+".consumed")
	marker, err := os.OpenFile(markerPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("PERMIT_REPLAY")
		}
		return fmt.Errorf("PERMIT_CONSUME_FAILED")
	}
	consumed := struct {
		SchemaVersion int       `json:"schema_version"`
		PermitID      string    `json:"permit_id"`
		ConsumedAt    time.Time `json:"consumed_at"`
	}{1, input.PermitID, now}
	if err := json.NewEncoder(marker).Encode(consumed); err != nil {
		_ = marker.Close()
		return fmt.Errorf("PERMIT_CONSUME_FAILED")
	}
	if err := marker.Sync(); err != nil {
		_ = marker.Close()
		return fmt.Errorf("PERMIT_CONSUME_FAILED")
	}
	if err := marker.Close(); err != nil {
		return fmt.Errorf("PERMIT_CONSUME_FAILED")
	}
	return nil
}

func readPermit(path string) (Permit, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return Permit{}, fmt.Errorf("PERMIT_UNAVAILABLE")
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return Permit{}, fmt.Errorf("PERMIT_FILE_UNSAFE")
	}
	file, err := os.Open(path)
	if err != nil {
		return Permit{}, fmt.Errorf("PERMIT_UNAVAILABLE")
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 16*1024))
	decoder.DisallowUnknownFields()
	var permit Permit
	if err := decoder.Decode(&permit); err != nil {
		return Permit{}, fmt.Errorf("PERMIT_INVALID")
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		return Permit{}, fmt.Errorf("PERMIT_INVALID")
	}
	return permit, nil
}
