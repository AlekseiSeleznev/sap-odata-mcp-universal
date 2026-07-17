# One-shot GPI Invoice WSDL bundle fetch

`sap_wsdl_bundle_fetch_once` is a narrow read-only MCP action for one approved
GPI Employee Shop Invoice WSDL read. It is registered by the dashboard runtime
and is visible in `tools/list` even when no SAP system is active. Merely listing
the action never reads configuration, consumes a permit, resolves DNS, or opens
a network connection.

The caller can provide only these four values:

- `system_id=gpi_100`;
- `contract_id=employee-shop-invoice-wsdl`;
- the lowercase SHA-256 of the sealed request manifest;
- an administrator-issued lowercase UUIDv4 permit id.

Raw URLs, SAP clients, credentials, headers, authentication modes, retry flags,
and evidence paths are not accepted by the MCP schema.

Before configuration is read, the handler checks the successfully activated
dashboard runtime identity. Only exact `gpi_100` is accepted; an empty identity,
GPD, or any other system returns `IDENTITY_MISMATCH` without permit or network
I/O. A failed dashboard activation does not leave the identity set.

## Sealed server-side configuration

Set `SAP_WSDL_BUNDLE_MANIFEST_FILE` to a private mode-`0600` JSON file. The
manifest itself is never returned or logged. It has this shape (placeholder
values only):

```json
{
  "schema_version": 1,
  "system_id": "gpi_100",
  "contract_id": "employee-shop-invoice-wsdl",
  "root_url": "<owner-supplied exact GPI binding WSDL URL including sap-client>",
  "sap_client": "<sealed client>",
  "allowed_origin": "<exact origin of root_url>",
  "expected_service_qname": "<expanded QName>",
  "expected_port_qname": "<expanded QName>",
  "expected_binding_qname": "<expanded QName>",
  "expected_operation": "<operation name>",
  "expected_soap_action": "<sealed exact SOAPAction>",
  "credential_file": "<private mode-0600 credential JSON path>",
  "permit_dir": "<private mode-0700 permit ledger directory>",
  "evidence_dir": "<private mode-0700 sanitized evidence directory>",
  "evidence_hmac_key_file": "<private mode-0600 file containing at least 32 random bytes>",
  "limits": {
    "connect_timeout_ms": 5000,
    "tls_handshake_timeout_ms": 5000,
    "response_header_timeout_ms": 10000,
    "per_document_timeout_ms": 20000,
    "whole_action_timeout_ms": 90000,
    "max_depth": 12,
    "max_documents": 64,
    "max_references": 256,
    "max_document_bytes": 4194304,
    "max_total_bytes": 33554432,
    "max_xml_tokens": 1000000,
    "max_xml_nesting": 256,
    "max_attributes": 128,
    "max_attribute_bytes": 65536,
    "max_evidence_bytes": 4194304
  }
}
```

The credential JSON contains only `username` and `password`. The production
loader rejects unknown fields, unsafe file modes, non-GPI identities, a root
whose query does not contain exactly the sealed SAP client, a non-matching
origin, and any changed limit.

## Permit ledger

The action never creates permits. An external approval step writes a private
`<permit-id>.json` file into `permit_dir`:

```json
{
  "schema_version": 1,
  "permit_id": "<lowercase UUIDv4>",
  "purpose": "WSDL_BUNDLE_READ",
  "system_id": "gpi_100",
  "contract_id": "employee-shop-invoice-wsdl",
  "request_manifest_sha256": "<sealed manifest SHA-256>",
  "binary_sha256": "<approved installed executable SHA-256>",
  "not_before": "<RFC3339 timestamp>",
  "expires_at": "<RFC3339 timestamp>"
}
```

After all local checks pass, the handler atomically creates
`<permit-id>.consumed` with `O_EXCL` before the first `RoundTrip`. Replay and
concurrent races therefore stop with zero HTTP requests. A consumed marker is
not removed after any later network, XML, contract, sanitization, or evidence
failure.

## Fetch and evidence semantics

The action performs deterministic serial closure over WSDL 1.1 imports, XSD
1.0 import/include/redefine, and WS-Policy 1.2/1.5 references. It uses one pinned
DNS result, the sealed origin only, `Accept-Encoding: identity`, no proxy,
redirect, retry, keep-alive, HTTP/2, fallback, parameter rewrite, activation, or
SOAP POST. Each unique normalized URI can reach `RoundTrip` at most once.

Imported WSDL documents are resolved as one semantic symbol table: service,
port, binding, portType, operation, input/output/fault messages, and message
parts may live in separate documents. Duplicate WSDL or XSD definitions must be
identical; conflicts hard-stop. XSD evidence retains parent, sequence order,
cardinality, type, and facets.

Only a complete sanitized bundle is atomically renamed into `evidence_dir`.
Private origins are represented by keyed HMAC identifiers; raw XML, endpoints,
credentials, headers, the SAP client, and raw SOAPAction are absent. SOAPAction
is represented by SHA-256 plus a match flag. Every failure returns a sanitized
`HARD_STOP` envelope and never publishes a partial bundle.

`bundle_sha256` is exactly SHA-256 over the RFC 8785/JCS representation of
`{root_document_id, sorted documents, sorted edges}`. Contract evidence is
published alongside it but is deliberately not added to that fixed digest
projection.

## Local pre-call gate

Before a separately approved live call:

1. verify the installed executable SHA-256 and source commit;
2. use only `tools/list` to check there is exactly one action with the reviewed
   schemas;
3. require the test suite, race test, vet, leak scan, and build to pass;
4. compare the canonical schemas against the reviewed digests;
5. create a permit only in the separate approval phase.

Reviewed canonical schema digests:

- input: `94dd1a4f23157cd0076a685b8104d2cddec090fec99b6fc1a624cbc334007ea2`;
- output: `ecc0286eb79b8dba9efb8e1e776fba5bc0004b5d21a33deb3d083cd5a0c8c853`.
