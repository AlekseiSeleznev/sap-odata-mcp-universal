package wsdlbundle

import (
	"fmt"
	"regexp"
)

const (
	SystemID   = "gpi_100"
	ContractID = "employee-shop-invoice-wsdl"
)

var (
	lowerSHA256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
	uuidV4Pattern      = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	documentIDPattern  = regexp.MustCompile(`^doc-[0-9a-f]{32}$`)
	stopCodePattern    = regexp.MustCompile(`^[A-Z0-9_]+$`)
)

type Input struct {
	SystemID              string `json:"system_id"`
	ContractID            string `json:"contract_id"`
	RequestManifestSHA256 string `json:"request_manifest_sha256"`
	PermitID              string `json:"permit_id"`
}

func ParseInput(args map[string]interface{}) (Input, error) {
	if len(args) != 4 {
		return Input{}, fmt.Errorf("INVALID_INPUT: exactly four properties are required")
	}
	allowed := map[string]bool{
		"system_id":               true,
		"contract_id":             true,
		"request_manifest_sha256": true,
		"permit_id":               true,
	}
	for key := range args {
		if !allowed[key] {
			return Input{}, fmt.Errorf("INVALID_INPUT: additional properties are not allowed")
		}
	}
	readString := func(name string) (string, error) {
		value, ok := args[name].(string)
		if !ok || value == "" {
			return "", fmt.Errorf("INVALID_INPUT: %s must be a non-empty string", name)
		}
		return value, nil
	}
	systemID, err := readString("system_id")
	if err != nil {
		return Input{}, err
	}
	contractID, err := readString("contract_id")
	if err != nil {
		return Input{}, err
	}
	manifestSHA, err := readString("request_manifest_sha256")
	if err != nil {
		return Input{}, err
	}
	permitID, err := readString("permit_id")
	if err != nil {
		return Input{}, err
	}
	if systemID != SystemID {
		return Input{}, fmt.Errorf("INVALID_INPUT: system_id does not match the sealed contract")
	}
	if contractID != ContractID {
		return Input{}, fmt.Errorf("INVALID_INPUT: contract_id does not match the sealed contract")
	}
	if !lowerSHA256Pattern.MatchString(manifestSHA) {
		return Input{}, fmt.Errorf("INVALID_INPUT: request_manifest_sha256 must be lowercase SHA-256")
	}
	if !uuidV4Pattern.MatchString(permitID) {
		return Input{}, fmt.Errorf("INVALID_INPUT: permit_id must be a lowercase UUIDv4")
	}
	return Input{
		SystemID:              systemID,
		ContractID:            contractID,
		RequestManifestSHA256: manifestSHA,
		PermitID:              permitID,
	}, nil
}
