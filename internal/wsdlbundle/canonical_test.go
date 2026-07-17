package wsdlbundle

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestBundleSHA256UsesExactJCSProjectionWithoutContract(t *testing.T) {
	documents := []DocumentEvidence{{
		DocumentID: "doc-00000000000000000000000000000000", Kind: "WSDL11", ByteCount: 7,
		RawSHA256:       "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		SanitizedSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		MediaType:       "application/xml",
	}}
	edges := []Edge{}
	got, err := bundleSHA256("doc-00000000000000000000000000000000", documents, edges)
	if err != nil {
		t.Fatalf("bundle digest: %v", err)
	}
	canonicalLiteral := `{"documents":[{"byte_count":7,"document_id":"doc-00000000000000000000000000000000","kind":"WSDL11","media_type":"application/xml","raw_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","sanitized_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}],"edges":[],"root_document_id":"doc-00000000000000000000000000000000"}`
	wantBytes := sha256.Sum256([]byte(canonicalLiteral))
	want := fmt.Sprintf("%x", wantBytes)
	if got != want {
		t.Fatalf("bundle digest mismatch: got=%s want=%s", got, want)
	}
}

func TestCanonicalJSONUsesUTF16KeyOrderAndDoesNotHTMLEscape(t *testing.T) {
	got, err := canonicalJSON(map[string]interface{}{"z": 1, "a": "<safe>"})
	if err != nil {
		t.Fatalf("canonical JSON: %v", err)
	}
	if string(got) != `{"a":"<safe>","z":1}` {
		t.Fatalf("unexpected canonical JSON: %s", got)
	}
}
