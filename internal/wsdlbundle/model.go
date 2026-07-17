package wsdlbundle

type Limits struct {
	ConnectTimeoutMS        int   `json:"connect_timeout_ms"`
	TLSHandshakeTimeoutMS   int   `json:"tls_handshake_timeout_ms"`
	ResponseHeaderTimeoutMS int   `json:"response_header_timeout_ms"`
	PerDocumentTimeoutMS    int   `json:"per_document_timeout_ms"`
	WholeActionTimeoutMS    int   `json:"whole_action_timeout_ms"`
	MaxDepth                int   `json:"max_depth"`
	MaxDocuments            int   `json:"max_documents"`
	MaxReferences           int   `json:"max_references"`
	MaxDocumentBytes        int64 `json:"max_document_bytes"`
	MaxTotalBytes           int64 `json:"max_total_bytes"`
	MaxXMLTokens            int   `json:"max_xml_tokens"`
	MaxXMLNesting           int   `json:"max_xml_nesting"`
	MaxAttributes           int   `json:"max_attributes"`
	MaxAttributeBytes       int   `json:"max_attribute_bytes"`
	MaxEvidenceBytes        int64 `json:"max_evidence_bytes"`
}

func ProductionLimits() Limits {
	return Limits{
		ConnectTimeoutMS:        5000,
		TLSHandshakeTimeoutMS:   5000,
		ResponseHeaderTimeoutMS: 10000,
		PerDocumentTimeoutMS:    20000,
		WholeActionTimeoutMS:    90000,
		MaxDepth:                12,
		MaxDocuments:            64,
		MaxReferences:           256,
		MaxDocumentBytes:        4 * 1024 * 1024,
		MaxTotalBytes:           32 * 1024 * 1024,
		MaxXMLTokens:            1_000_000,
		MaxXMLNesting:           256,
		MaxAttributes:           128,
		MaxAttributeBytes:       64 * 1024,
		MaxEvidenceBytes:        4 * 1024 * 1024,
	}
}

type Manifest struct {
	SchemaVersion        int    `json:"schema_version"`
	SystemID             string `json:"system_id"`
	ContractID           string `json:"contract_id"`
	RootURL              string `json:"root_url"`
	SAPClient            string `json:"sap_client"`
	AllowedOrigin        string `json:"allowed_origin"`
	ExpectedServiceQName string `json:"expected_service_qname"`
	ExpectedPortQName    string `json:"expected_port_qname"`
	ExpectedBindingQName string `json:"expected_binding_qname"`
	ExpectedOperation    string `json:"expected_operation"`
	ExpectedSOAPAction   string `json:"expected_soap_action"`
	CredentialFile       string `json:"credential_file"`
	PermitDir            string `json:"permit_dir"`
	EvidenceDir          string `json:"evidence_dir"`
	EvidenceHMACKeyFile  string `json:"evidence_hmac_key_file"`
	Limits               Limits `json:"limits"`
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Result struct {
	SchemaVersion          string    `json:"schema_version"`
	Outcome                string    `json:"outcome"`
	AttemptID              string    `json:"attempt_id"`
	SystemID               string    `json:"system_id"`
	ContractID             string    `json:"contract_id"`
	RequestManifestSHA256  string    `json:"request_manifest_sha256"`
	PermitConsumed         bool      `json:"permit_consumed"`
	NetworkGetsStarted     int       `json:"network_gets_started"`
	Identity               Identity  `json:"identity"`
	Bundle                 *Bundle   `json:"bundle"`
	HardStop               *HardStop `json:"hard_stop"`
	EvidenceManifestSHA256 *string   `json:"evidence_manifest_sha256"`
}

type Identity struct {
	MatchesSealedManifest       bool    `json:"matches_sealed_manifest"`
	ConnectionBindingHMACSHA256 string  `json:"connection_binding_hmac_sha256"`
	TLSPeerSHA256               *string `json:"tls_peer_sha256"`
}

type HardStop struct {
	Phase   string `json:"phase"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Bundle struct {
	Complete       bool               `json:"complete"`
	RootDocumentID string             `json:"root_document_id"`
	BundleSHA256   string             `json:"bundle_sha256"`
	Documents      []DocumentEvidence `json:"documents"`
	Edges          []Edge             `json:"edges"`
	Contract       ContractSummary    `json:"contract"`
}

type DocumentEvidence struct {
	DocumentID      string `json:"document_id"`
	Kind            string `json:"kind"`
	ByteCount       int    `json:"byte_count"`
	RawSHA256       string `json:"raw_sha256"`
	SanitizedSHA256 string `json:"sanitized_sha256"`
	MediaType       string `json:"media_type"`
}

type Edge struct {
	FromDocumentID string `json:"from_document_id"`
	ToDocumentID   string `json:"to_document_id"`
	Relation       string `json:"relation"`
}

type ContractSummary struct {
	WSDLVersion                     string           `json:"wsdl_version"`
	TargetNamespaces                []string         `json:"target_namespaces"`
	ServiceQName                    string           `json:"service_qname"`
	PortQName                       string           `json:"port_qname"`
	BindingQName                    string           `json:"binding_qname"`
	SOAPVersion                     string           `json:"soap_version"`
	Operation                       string           `json:"operation"`
	InputMessageQName               string           `json:"input_message_qname"`
	OutputMessageQName              *string          `json:"output_message_qname"`
	FaultMessageQNames              []string         `json:"fault_message_qnames"`
	MessageExchange                 string           `json:"message_exchange"`
	SOAPActionSHA256                string           `json:"soap_action_sha256"`
	SOAPActionMatchesSealedExpected bool             `json:"soap_action_matches_sealed_expected"`
	Messages                        []MessageSummary `json:"messages"`
	XSDComponents                   []XSDComponent   `json:"xsd_components"`
	PolicyAssertionQNames           []string         `json:"policy_assertion_qnames"`
}

type MessageSummary struct {
	QName string        `json:"qname"`
	Parts []MessagePart `json:"parts"`
}

type MessagePart struct {
	Name         string `json:"name"`
	ElementQName string `json:"element_qname"`
	TypeQName    string `json:"type_qname"`
}

type XSDComponent struct {
	Namespace             string     `json:"namespace"`
	ParentQName           string     `json:"parent_qname"`
	Name                  string     `json:"name"`
	Kind                  string     `json:"kind"`
	Order                 int        `json:"order"`
	Type                  string     `json:"type"`
	RefQName              string     `json:"ref_qname"`
	MinOccurs             string     `json:"min_occurs"`
	MaxOccurs             string     `json:"max_occurs"`
	Facets                []XSDFacet `json:"facets"`
	Redefines             bool       `json:"-"`
	TypeReferences        []string   `json:"-"`
	RedefinitionRootQName string     `json:"-"`
}

type XSDFacet struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
