package wsdlbundle

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

const permitPurpose = "WSDL_BUNDLE_READ"

type PermitError struct{ Code string }

func (e *PermitError) Error() string { return e.Code }

func permitError(code string) error { return &PermitError{Code: code} }

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
		return permitError("PERMIT_CONFIG_INVALID")
	}
	dirInfo, err := os.Lstat(l.Dir)
	if err != nil || !dirInfo.IsDir() || dirInfo.Mode().Perm()&0o077 != 0 {
		return permitError("PERMIT_LEDGER_UNSAFE")
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
		return permitError("PERMIT_MISMATCH")
	}

	markerPath := filepath.Join(l.Dir, input.PermitID+".consumed")
	marker, err := os.OpenFile(markerPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return permitError("PERMIT_REPLAY")
		}
		return permitError("PERMIT_CONSUME_FAILED")
	}
	consumed := struct {
		SchemaVersion int       `json:"schema_version"`
		PermitID      string    `json:"permit_id"`
		ConsumedAt    time.Time `json:"consumed_at"`
	}{1, input.PermitID, now}
	if err := json.NewEncoder(marker).Encode(consumed); err != nil {
		_ = marker.Close()
		return permitError("PERMIT_CONSUME_FAILED")
	}
	if err := marker.Sync(); err != nil {
		_ = marker.Close()
		return permitError("PERMIT_CONSUME_FAILED")
	}
	if err := marker.Close(); err != nil {
		return permitError("PERMIT_CONSUME_FAILED")
	}
	directory, err := os.Open(l.Dir)
	if err != nil {
		return permitError("PERMIT_CONSUME_FAILED")
	}
	if err := directory.Sync(); err != nil {
		_ = directory.Close()
		return permitError("PERMIT_CONSUME_FAILED")
	}
	if err := directory.Close(); err != nil {
		return permitError("PERMIT_CONSUME_FAILED")
	}
	return nil
}

func readPermit(path string) (Permit, error) {
	var permit Permit
	if err := decodePrivateJSON(path, &permit, 16*1024); err != nil {
		var privateErr *privateFileError
		if errors.As(err, &privateErr) {
			switch privateErr.kind {
			case "unsafe":
				return Permit{}, permitError("PERMIT_FILE_UNSAFE")
			case "unavailable":
				return Permit{}, permitError("PERMIT_UNAVAILABLE")
			}
		}
		return Permit{}, permitError("PERMIT_INVALID")
	}
	return permit, nil
}
