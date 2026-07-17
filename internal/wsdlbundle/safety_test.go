package wsdlbundle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestServiceStopsBeforePermitAndTransportOnManifestMismatch(t *testing.T) {
	ledger := &fixtureLedger{}
	var transportFactories atomic.Int32
	service, input := fixtureService(t, ledger, func(context.Context, Manifest) (http.RoundTripper, error) {
		transportFactories.Add(1)
		return roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, fmt.Errorf("must not run") }), nil
	}, nil)
	args := inputArgs(input)
	args["request_manifest_sha256"] = strings.Repeat("0", 64)
	result, err := service.Fetch(context.Background(), args)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	assertHardStop(t, result, "MANIFEST_MISMATCH", false, 0)
	if ledger.calls.Load() != 0 || transportFactories.Load() != 0 {
		t.Fatalf("pre-manifest rejection touched permit or transport: ledger=%d transport=%d", ledger.calls.Load(), transportFactories.Load())
	}
}

func TestServiceStopsBeforeTransportOnPermitMismatch(t *testing.T) {
	ledger := &fixtureLedger{err: permitError("PERMIT_MISMATCH")}
	var transportFactories atomic.Int32
	service, input := fixtureService(t, ledger, func(context.Context, Manifest) (http.RoundTripper, error) {
		transportFactories.Add(1)
		return nil, fmt.Errorf("must not run")
	}, nil)
	result, err := service.Fetch(context.Background(), inputArgs(input))
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	assertHardStop(t, result, "PERMIT_MISMATCH", false, 0)
	if ledger.calls.Load() != 1 || transportFactories.Load() != 0 {
		t.Fatalf("permit rejection touched transport: ledger=%d transport=%d", ledger.calls.Load(), transportFactories.Load())
	}
}

func TestServiceNeverRetriesRedirectAuthMediaOrTransportFailures(t *testing.T) {
	tests := []struct {
		name      string
		code      string
		roundTrip func(*http.Request) (*http.Response, error)
	}{
		{"redirect", "HTTP_REDIRECT", responseRoundTrip(http.StatusFound, "application/xml", `<x/>`)},
		{"unauthorized", "AUTH_FAILURE", responseRoundTrip(http.StatusUnauthorized, "application/xml", `<x/>`)},
		{"forbidden", "AUTH_FAILURE", responseRoundTrip(http.StatusForbidden, "application/xml", `<x/>`)},
		{"login html", "MEDIA_TYPE_INVALID", responseRoundTrip(http.StatusOK, "text/html", `<html>login</html>`)},
		{"connection reset", "NETWORK_ERROR", func(*http.Request) (*http.Response, error) { return nil, fmt.Errorf("fixture reset") }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			service, input := fixtureService(t, &fixtureLedger{}, func(context.Context, Manifest) (http.RoundTripper, error) {
				return roundTripFunc(func(req *http.Request) (*http.Response, error) { calls.Add(1); return tc.roundTrip(req) }), nil
			}, nil)
			result, err := service.Fetch(context.Background(), inputArgs(input))
			if err != nil {
				t.Fatalf("fetch: %v", err)
			}
			assertHardStop(t, result, tc.code, true, 1)
			if calls.Load() != 1 {
				t.Fatalf("transport attempted %d times, want exactly one", calls.Load())
			}
		})
	}
}

func TestServiceRejectsUnsafeXMLDialectsAndCrossOriginReferences(t *testing.T) {
	tests := []struct{ name, code, body string }{
		{"doctype", "XML_UNSAFE", `<!DOCTYPE definitions [<!ENTITY x "boom">]><wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/"/>`},
		{"xinclude", "XML_UNSAFE", `<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/" xmlns:xi="http://www.w3.org/2001/XInclude"><xi:include href="local.xml"/></wsdl:definitions>`},
		{"wsdl 2", "UNSUPPORTED_DIALECT", `<description xmlns="http://www.w3.org/ns/wsdl"/>`},
		{"xsd 1.1", "UNSUPPORTED_DIALECT", `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema"><xsd:assert test="true()"/></xsd:schema>`},
		{"unknown policy dialect", "UNSUPPORTED_DIALECT", `<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/" xmlns:bad="urn:vendor:policy"><bad:Policy/></wsdl:definitions>`},
		{"cross origin", "URI_POLICY_VIOLATION", `<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/"><wsdl:import namespace="urn:x" location="https://other.invalid/private.wsdl"/></wsdl:definitions>`},
		{"dot traversal", "URI_POLICY_VIOLATION", `<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/"><wsdl:import namespace="urn:x" location="../private.wsdl?sap-client=100"/></wsdl:definitions>`},
		{"decoded control", "URI_POLICY_VIOLATION", `<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/"><wsdl:import namespace="urn:x" location="/%0aprivate.wsdl?sap-client=100"/></wsdl:definitions>`},
		{"missing import", "REFERENCE_MISSING", `<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/"><wsdl:import namespace="urn:x"/></wsdl:definitions>`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			service, input := fixtureService(t, &fixtureLedger{}, func(context.Context, Manifest) (http.RoundTripper, error) {
				return roundTripFunc(func(req *http.Request) (*http.Response, error) {
					calls.Add(1)
					return responseRoundTrip(http.StatusOK, "application/xml", tc.body)(req)
				}), nil
			}, nil)
			result, err := service.Fetch(context.Background(), inputArgs(input))
			if err != nil {
				t.Fatalf("fetch: %v", err)
			}
			assertHardStop(t, result, tc.code, true, 1)
			if calls.Load() != 1 {
				t.Fatalf("transport attempted %d times", calls.Load())
			}
		})
	}
}

func TestServiceEnforcesXMLAndClosureLimitsWithoutPartialEvidence(t *testing.T) {
	tests := []struct {
		name   string
		code   string
		adjust func(*Limits)
		body   string
	}{
		{"document bytes", "DOCUMENT_SIZE_LIMIT", func(l *Limits) { l.MaxDocumentBytes = 32; l.MaxTotalBytes = 64 }, strings.Repeat("x", 33)},
		{"xml tokens", "XML_TOKEN_LIMIT", func(l *Limits) { l.MaxXMLTokens = 2 }, `<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/"><wsdl:message name="x"/></wsdl:definitions>`},
		{"xml nesting", "XML_NESTING_LIMIT", func(l *Limits) { l.MaxXMLNesting = 2 }, `<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/"><wsdl:types><wsdl:documentation/></wsdl:types></wsdl:definitions>`},
		{"attribute count", "XML_ATTRIBUTE_LIMIT", func(l *Limits) { l.MaxAttributes = 1 }, `<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/" name="x" targetNamespace="urn:x"/>`},
		{"attribute bytes", "XML_ATTRIBUTE_SIZE_LIMIT", func(l *Limits) { l.MaxAttributeBytes = 3 }, `<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/" name="long"/>`},
		{"reference count", "REFERENCE_LIMIT", func(l *Limits) { l.MaxReferences = 1 }, `<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/"><wsdl:import namespace="urn:a" location="/a.wsdl?sap-client=100"/><wsdl:import namespace="urn:b" location="/b.wsdl?sap-client=100"/></wsdl:definitions>`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			limits := ProductionLimits()
			tc.adjust(&limits)
			service, input := fixtureService(t, &fixtureLedger{}, func(context.Context, Manifest) (http.RoundTripper, error) {
				return roundTripFunc(responseRoundTrip(http.StatusOK, "application/xml", tc.body)), nil
			}, &limits)
			result, err := service.Fetch(context.Background(), inputArgs(input))
			if err != nil {
				t.Fatalf("fetch: %v", err)
			}
			assertHardStop(t, result, tc.code, true, 1)
			if result.Bundle != nil || result.EvidenceManifestSHA256 != nil {
				t.Fatal("limit failure published partial evidence")
			}
		})
	}
}

func TestServiceEnforcesDocumentDepthTotalAndEvidenceLimits(t *testing.T) {
	tests := []struct {
		name, code string
		adjust     func(*Limits)
	}{
		{"documents", "DOCUMENT_LIMIT", func(l *Limits) { l.MaxDocuments = 1 }},
		{"depth", "DEPTH_LIMIT", func(l *Limits) { l.MaxDepth = 1 }},
		{"total bytes", "TOTAL_SIZE_LIMIT", func(l *Limits) { l.MaxDocumentBytes = 600; l.MaxTotalBytes = 700 }},
		{"evidence bytes", "ATOMIC_PUBLISH_FAILED", func(l *Limits) { l.MaxEvidenceBytes = 1 }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			limits := ProductionLimits()
			tc.adjust(&limits)
			var calls atomic.Int32
			service, input := fixtureService(t, &fixtureLedger{}, func(context.Context, Manifest) (http.RoundTripper, error) {
				return roundTripFunc(func(req *http.Request) (*http.Response, error) {
					calls.Add(1)
					body := minimalWSDL()
					switch tc.name {
					case "documents":
						body = `<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/"><wsdl:import namespace="urn:a" location="/a.wsdl?sap-client=100"/></wsdl:definitions>`
					case "depth":
						if strings.HasSuffix(req.URL.Path, "invoice.wsdl") {
							body = `<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/"><wsdl:import namespace="urn:a" location="/a.wsdl?sap-client=100"/></wsdl:definitions>`
						} else {
							body = `<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/"><wsdl:import namespace="urn:b" location="/b.wsdl?sap-client=100"/></wsdl:definitions>`
						}
					case "total bytes":
						if strings.HasSuffix(req.URL.Path, "invoice.wsdl") {
							body = `<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/"><wsdl:import namespace="urn:a" location="/a.wsdl?sap-client=100"/>` + strings.Repeat(" ", 350) + `</wsdl:definitions>`
						} else {
							body = `<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/">` + strings.Repeat(" ", 350) + `</wsdl:definitions>`
						}
					}
					return responseRoundTrip(http.StatusOK, "application/xml", body)(req)
				}), nil
			}, &limits)
			result, err := service.Fetch(context.Background(), inputArgs(input))
			if err != nil {
				t.Fatalf("fetch: %v", err)
			}
			assertHardStop(t, result, tc.code, true, int(calls.Load()))
			if result.Bundle != nil || result.EvidenceManifestSHA256 != nil {
				t.Fatal("failure published partial evidence")
			}
		})
	}
}

func TestServiceEnforcesPerDocumentAndWholeActionTimeouts(t *testing.T) {
	t.Run("per document", func(t *testing.T) {
		limits := ProductionLimits()
		limits.PerDocumentTimeoutMS = 5
		limits.WholeActionTimeoutMS = 100
		service, input := fixtureService(t, &fixtureLedger{}, func(context.Context, Manifest) (http.RoundTripper, error) {
			return roundTripFunc(func(req *http.Request) (*http.Response, error) {
				<-req.Context().Done()
				return nil, req.Context().Err()
			}), nil
		}, &limits)
		result, err := service.Fetch(context.Background(), inputArgs(input))
		if err != nil {
			t.Fatalf("fetch: %v", err)
		}
		assertHardStop(t, result, "DOCUMENT_TIMEOUT", true, 1)
	})
	t.Run("whole action", func(t *testing.T) {
		service, input := fixtureService(t, &fixtureLedger{}, func(context.Context, Manifest) (http.RoundTripper, error) {
			return nil, fmt.Errorf("must not initialize")
		}, nil)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		result, err := service.Fetch(ctx, inputArgs(input))
		if err != nil {
			t.Fatalf("fetch: %v", err)
		}
		assertHardStop(t, result, "WHOLE_ACTION_TIMEOUT", true, 0)
	})
}

func TestServiceHardStopsOnConflictingXSDComponents(t *testing.T) {
	root := strings.Replace(minimalWSDL(), "</wsdl:definitions>", `<wsdl:types><xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema"><xsd:import namespace="urn:types" schemaLocation="/a.xsd?sap-client=100"/><xsd:import namespace="urn:types" schemaLocation="/b.xsd?sap-client=100"/></xsd:schema></wsdl:types></wsdl:definitions>`, 1)
	var calls atomic.Int32
	service, input := fixtureService(t, &fixtureLedger{}, func(context.Context, Manifest) (http.RoundTripper, error) {
		return roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls.Add(1)
			body := root
			if req.URL.Path == "/a.xsd" {
				body = `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:types"><xsd:complexType name="InvoiceType"><xsd:sequence><xsd:element name="Amount" type="xsd:string"/></xsd:sequence></xsd:complexType></xsd:schema>`
			}
			if req.URL.Path == "/b.xsd" {
				body = `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:types"><xsd:complexType name="InvoiceType"><xsd:sequence><xsd:element name="Amount" type="xsd:decimal"/></xsd:sequence></xsd:complexType></xsd:schema>`
			}
			return responseRoundTrip(http.StatusOK, "application/xml", body)(req)
		}), nil
	}, nil)
	result, err := service.Fetch(context.Background(), inputArgs(input))
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	assertHardStop(t, result, "CONTRACT_CONFLICT", true, 3)
	if calls.Load() != 3 {
		t.Fatalf("unexpected request count: %d", calls.Load())
	}
}

func TestProductionLimitsRemainExact(t *testing.T) {
	got := ProductionLimits()
	if got.ConnectTimeoutMS != 5000 || got.TLSHandshakeTimeoutMS != 5000 || got.ResponseHeaderTimeoutMS != 10000 || got.PerDocumentTimeoutMS != 20000 || got.WholeActionTimeoutMS != 90000 || got.MaxDepth != 12 || got.MaxDocuments != 64 || got.MaxReferences != 256 || got.MaxDocumentBytes != 4*1024*1024 || got.MaxTotalBytes != 32*1024*1024 || got.MaxXMLTokens != 1_000_000 || got.MaxXMLNesting != 256 || got.MaxAttributes != 128 || got.MaxAttributeBytes != 64*1024 || got.MaxEvidenceBytes != 4*1024*1024 {
		t.Fatalf("production limits drifted: %#v", got)
	}
}

func TestStrictTransportPinsOneDNSResolutionAndIgnoresProxyEnvironment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, `<ok/>`)
	}))
	defer server.Close()
	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	rootURL := "http://localhost:" + parsed.Port() + "/invoice.wsdl?sap-client=100"
	manifest := productionFixtureManifest(rootURL, t.TempDir())
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	var resolutions atomic.Int32
	transport, err := newStrictTransportWithResolver(context.Background(), manifest, func(context.Context, string) ([]net.IPAddr, error) {
		resolutions.Add(1)
		return []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}}, nil
	})
	if err != nil {
		t.Fatalf("create strict transport: %v", err)
	}
	for _, path := range []string{"/one", "/two"} {
		req, _ := http.NewRequest(http.MethodGet, "http://localhost:"+parsed.Port()+path, nil)
		resp, err := transport.RoundTrip(req)
		if err != nil {
			t.Fatalf("round trip %s: %v", path, err)
		}
		_ = resp.Body.Close()
	}
	if resolutions.Load() != 1 {
		t.Fatalf("DNS resolved %d times, want exactly once", resolutions.Load())
	}
}

func TestStrictTransportRejectsPlaintextLocalhostResolvedOutsideLoopback(t *testing.T) {
	manifest := productionFixtureManifest("http://localhost:18080/invoice.wsdl?sap-client=100", t.TempDir())
	_, err := newStrictTransportWithResolver(context.Background(), manifest, func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("192.0.2.10")}}, nil
	})
	if err == nil {
		t.Fatal("plaintext localhost resolved outside loopback was accepted")
	}
}

func TestParseXMLDocumentHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, stopped := parseXMLDocument(ctx, []byte(minimalWSDL()), ProductionLimits())
	if stopped == nil || stopped.Code != "DOCUMENT_TIMEOUT" {
		t.Fatalf("parse stop = %+v, want DOCUMENT_TIMEOUT", stopped)
	}
}

func TestPublishEvidenceHonorsCanceledContextWithoutPublishing(t *testing.T) {
	evidenceDir := filepath.Join(t.TempDir(), "evidence")
	manifest := productionFixtureManifest("https://gpi.invalid/invoice.wsdl?sap-client=100", evidenceDir)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := publishEvidence(ctx, manifest, "6ba7b810-9dad-41d1-80b4-00c04fd430c8", &Bundle{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("publish error = %v, want context cancellation", err)
	}
	if entries, err := os.ReadDir(evidenceDir); err == nil && len(entries) != 0 {
		t.Fatalf("canceled publish left %d evidence files", len(entries))
	} else if err != nil && !os.IsNotExist(err) {
		t.Fatalf("inspect evidence dir: %v", err)
	}
}

func TestResolvedFetchURIPreservesQueryAndDeduplicatesEscapedUnreservedPath(t *testing.T) {
	manifest := productionFixtureManifest("https://gpi.invalid/invoice.wsdl?sap-client=100", t.TempDir())
	plain, err := resolveFetchURI("https://gpi.invalid/invoice.wsdl?sap-client=100", nil, manifest)
	if err != nil {
		t.Fatalf("normalize plain URI: %v", err)
	}
	escaped, err := resolveFetchURI("https://gpi.invalid/%69nvoice.wsdl?sap-client=%31%30%30", nil, manifest)
	if err != nil {
		t.Fatalf("normalize escaped URI: %v", err)
	}
	if plain.RequestURI != "https://gpi.invalid/invoice.wsdl?sap-client=100" || escaped.RequestURI != "https://gpi.invalid/%69nvoice.wsdl?sap-client=%31%30%30" {
		t.Fatalf("request URI was rewritten: plain=%q escaped=%q", plain.RequestURI, escaped.RequestURI)
	}
	if plain.NormalizedKey != escaped.NormalizedKey {
		t.Fatalf("equivalent URIs did not deduplicate: plain=%q escaped=%q", plain, escaped)
	}
}

func TestResolveFetchURIRejectsClientSwitchAndUnsafeDecodedQuery(t *testing.T) {
	manifest := productionFixtureManifest("https://gpi.invalid/invoice.wsdl?sap-client=100", t.TempDir())
	for _, raw := range []string{
		"https://gpi.invalid/invoice.wsdl",
		"https://gpi.invalid/invoice.wsdl?sap-client=200",
		"https://gpi.invalid/invoice.wsdl?sap-client=100&sap-client=100",
		"https://gpi.invalid/invoice.wsdl?sap-client=100&x=%0A",
		"https://gpi.invalid/invoice.wsdl?sap-client=100&x=%5C",
	} {
		if _, err := resolveFetchURI(raw, nil, manifest); err == nil {
			t.Fatalf("unsafe or switched query accepted: %q", raw)
		}
	}
}

func TestResolveFetchURIKeyNormalizesEmptyAuthorityPath(t *testing.T) {
	manifest := productionFixtureManifest("https://gpi.invalid/invoice.wsdl?sap-client=100", t.TempDir())
	emptyPath, err := resolveFetchURI("https://gpi.invalid?sap-client=100", nil, manifest)
	if err != nil {
		t.Fatalf("normalize empty path: %v", err)
	}
	slashPath, err := resolveFetchURI("https://gpi.invalid/?sap-client=100", nil, manifest)
	if err != nil {
		t.Fatalf("normalize slash path: %v", err)
	}
	if emptyPath.NormalizedKey != slashPath.NormalizedKey {
		t.Fatalf("empty and slash paths did not deduplicate: empty=%q slash=%q", emptyPath.NormalizedKey, slashPath.NormalizedKey)
	}
}

func TestServiceHardStopsWhenWSDLMessagePartDoesNotResolveInXSDClosure(t *testing.T) {
	wsdl := strings.Replace(minimalWSDL(), `<wsdl:message name="InvoiceRequest"/>`, `<wsdl:message name="InvoiceRequest"><wsdl:part name="parameters" element="tns:MissingElement"/></wsdl:message>`, 1)
	service, input := fixtureService(t, &fixtureLedger{}, func(context.Context, Manifest) (http.RoundTripper, error) {
		return roundTripFunc(responseRoundTrip(http.StatusOK, "application/xml", wsdl)), nil
	}, nil)
	result, err := service.Fetch(context.Background(), inputArgs(input))
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	assertHardStop(t, result, "CONTRACT_MISMATCH", true, 1)
}

func TestServiceHardStopsWhenReferencedXSDElementTypeDoesNotResolve(t *testing.T) {
	wsdl := strings.Replace(minimalWSDL(), `<wsdl:message name="InvoiceRequest"/>`, `<wsdl:types><xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:employee-shop:invoice"><xsd:element name="InvoiceRequest" type="tns:MissingType"/></xsd:schema></wsdl:types><wsdl:message name="InvoiceRequest"><wsdl:part name="parameters" element="tns:InvoiceRequest"/></wsdl:message>`, 1)
	service, input := fixtureService(t, &fixtureLedger{}, func(context.Context, Manifest) (http.RoundTripper, error) {
		return roundTripFunc(responseRoundTrip(http.StatusOK, "application/xml", wsdl)), nil
	}, nil)
	result, err := service.Fetch(context.Background(), inputArgs(input))
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	assertHardStop(t, result, "CONTRACT_MISMATCH", true, 1)
}

func TestServiceHardStopsOnEveryUnresolvedXSDReferenceForm(t *testing.T) {
	tests := []struct {
		name     string
		fragment string
	}{
		{"extension base", `<xsd:complexType name="T"><xsd:complexContent><xsd:extension base="tns:MissingType"/></xsd:complexContent></xsd:complexType>`},
		{"list item type", `<xsd:simpleType name="T"><xsd:list itemType="tns:MissingType"/></xsd:simpleType>`},
		{"union member type", `<xsd:simpleType name="T"><xsd:union memberTypes="xsd:string tns:MissingType"/></xsd:simpleType>`},
		{"element ref", `<xsd:complexType name="T"><xsd:sequence><xsd:element ref="tns:MissingElement"/></xsd:sequence></xsd:complexType>`},
		{"group ref", `<xsd:complexType name="T"><xsd:sequence><xsd:group ref="tns:MissingGroup"/></xsd:sequence></xsd:complexType>`},
		{"unknown builtin", `<xsd:element name="E" type="xsd:MissingType"/>`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			schema := `<wsdl:types><xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:employee-shop:invoice">` + tc.fragment + `</xsd:schema></wsdl:types>`
			wsdl := strings.Replace(minimalWSDL(), `<wsdl:message name="InvoiceRequest"/>`, schema+`<wsdl:message name="InvoiceRequest"/>`, 1)
			service, input := fixtureService(t, &fixtureLedger{}, func(context.Context, Manifest) (http.RoundTripper, error) {
				return roundTripFunc(responseRoundTrip(http.StatusOK, "application/xml", wsdl)), nil
			}, nil)
			result, err := service.Fetch(context.Background(), inputArgs(input))
			if err != nil {
				t.Fatalf("fetch: %v", err)
			}
			assertHardStop(t, result, "CONTRACT_MISMATCH", true, 1)
		})
	}
}

func TestParseXMLRejectsAmbiguousXSDDeclarations(t *testing.T) {
	tests := []struct {
		name string
		xml  string
		code string
	}{
		{"element name and ref", `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:tns="urn:t" targetNamespace="urn:t"><xsd:element name="E" type="xsd:string"/><xsd:element name="Other" ref="tns:E"/></xsd:schema>`, "XSD_DECLARATION_INVALID"},
		{"element ref and type", `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:tns="urn:t" targetNamespace="urn:t"><xsd:element name="E" type="xsd:string"/><xsd:element ref="tns:E" type="xsd:string"/></xsd:schema>`, "XSD_DECLARATION_INVALID"},
		{"group name and ref", `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:tns="urn:t" targetNamespace="urn:t"><xsd:group name="G"/><xsd:group name="Other" ref="tns:G"/></xsd:schema>`, "XSD_DECLARATION_INVALID"},
		{"attribute group name and ref", `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:tns="urn:t" targetNamespace="urn:t"><xsd:attributeGroup name="G"/><xsd:attributeGroup name="Other" ref="tns:G"/></xsd:schema>`, "XSD_DECLARATION_INVALID"},
		{"attribute name and ref", `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:tns="urn:t" targetNamespace="urn:t"><xsd:attribute name="A" type="xsd:string"/><xsd:attribute name="Other" ref="tns:A"/></xsd:schema>`, "XSD_DECLARATION_INVALID"},
		{"complex type with type", `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t"><xsd:complexType name="T" type="xsd:string"/></xsd:schema>`, "XSD_DECLARATION_INVALID"},
		{"simple type with type", `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t"><xsd:simpleType name="T" type="xsd:string"/></xsd:schema>`, "XSD_DECLARATION_INVALID"},
		{"group with type", `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t"><xsd:group name="G" type="xsd:string"/></xsd:schema>`, "XSD_DECLARATION_INVALID"},
		{"attribute group with type", `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t"><xsd:attributeGroup name="G" type="xsd:string"/></xsd:schema>`, "XSD_DECLARATION_INVALID"},
		{"anonymous complex type", `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t"><xsd:element name="E"><xsd:complexType/></xsd:element></xsd:schema>`, "UNSUPPORTED_DIALECT"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, stopped := parseXMLDocument(context.Background(), []byte(tc.xml), ProductionLimits())
			if stopped == nil || stopped.Code != tc.code {
				t.Fatalf("parse stop = %+v, want %s", stopped, tc.code)
			}
		})
	}
}

func TestMergeSimpleTypeRedefinitionRequiresExactSelfRestriction(t *testing.T) {
	original := XSDComponent{Namespace: "urn:t", Name: "Code", Kind: "simpleType", Type: qname(xsdNamespace, "string"), TypeReferences: []string{qname(xsdNamespace, "string")}, Facets: []XSDFacet{{Name: "length", Value: "3"}}}
	selfQName := qname("urn:t", "Code")
	redefinition := XSDComponent{Namespace: "urn:t", Name: "Code", Kind: "simpleType", Type: selfQName, TypeReferences: []string{selfQName}, Facets: []XSDFacet{{Name: "pattern", Value: "[A-Z]{3}"}}, Redefines: true, RedefinitionRootQName: selfQName, Derivation: "restriction"}
	merged, err := mergeSimpleTypeRedefinition(original, redefinition)
	if err != nil {
		t.Fatalf("merge valid self restriction: %v", err)
	}
	if merged.Type != qname(xsdNamespace, "string") || len(merged.Facets) != 2 || merged.Facets[0].Name != "length" || merged.Facets[1].Name != "pattern" {
		t.Fatalf("effective simpleType evidence lost inherited constraints: %#v", merged)
	}

	invalid := []XSDComponent{redefinition, redefinition, redefinition}
	invalid[0].Derivation = "list"
	invalid[1].Type = qname(xsdNamespace, "string")
	invalid[2].TypeReferences = append(invalid[2].TypeReferences, qname(xsdNamespace, "string"))
	for index, candidate := range invalid {
		if _, err := mergeSimpleTypeRedefinition(original, candidate); err == nil {
			t.Fatalf("invalid redefine case %d was accepted: %#v", index, candidate)
		}
	}
}

func TestServiceFailsClosedOnComplexTypeRedefine(t *testing.T) {
	redefine := `<wsdl:types><xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:tns="urn:employee-shop:invoice" targetNamespace="urn:employee-shop:invoice"><xsd:redefine schemaLocation="/base.xsd?sap-client=100"><xsd:complexType name="T"><xsd:complexContent><xsd:restriction base="tns:T"><xsd:sequence><xsd:element name="A" type="xsd:string"/></xsd:sequence></xsd:restriction></xsd:complexContent></xsd:complexType></xsd:redefine></xsd:schema></wsdl:types>`
	root := strings.Replace(minimalWSDL(), `<wsdl:message name="InvoiceRequest"/>`, redefine+`<wsdl:message name="InvoiceRequest"/>`, 1)
	var calls atomic.Int32
	service, input := fixtureService(t, &fixtureLedger{}, func(context.Context, Manifest) (http.RoundTripper, error) {
		return roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls.Add(1)
			body := root
			if req.URL.Path == "/base.xsd" {
				body = `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:employee-shop:invoice"><xsd:complexType name="T"><xsd:sequence><xsd:element name="A" type="xsd:string"/><xsd:element name="B" type="xsd:string"/></xsd:sequence></xsd:complexType></xsd:schema>`
			}
			return responseRoundTrip(http.StatusOK, "application/xml", body)(req)
		}), nil
	}, nil)
	result, err := service.Fetch(context.Background(), inputArgs(input))
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	assertHardStop(t, result, "UNSUPPORTED_DIALECT", true, 2)
	if calls.Load() != 2 {
		t.Fatalf("transport attempts = %d, want 2", calls.Load())
	}
}

func TestStrictTransportConfiguresEveryNetworkTimeoutAndDisablesAutomaticBehavior(t *testing.T) {
	manifest := productionFixtureManifest("https://gpi.invalid/invoice.wsdl?sap-client=100", t.TempDir())
	transport, err := newStrictTransportWithResolver(context.Background(), manifest, func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}}, nil
	})
	if err != nil {
		t.Fatalf("create strict transport: %v", err)
	}
	strict, ok := transport.(*strictRoundTripper)
	if !ok {
		t.Fatalf("unexpected transport type %T", transport)
	}
	configured := strict.transport
	if strict.connectTimeout != 5*time.Second || configured.TLSHandshakeTimeout != 5*time.Second || configured.ResponseHeaderTimeout != 10*time.Second {
		t.Fatalf("network timeouts drifted: connect=%s tls=%s headers=%s", strict.connectTimeout, configured.TLSHandshakeTimeout, configured.ResponseHeaderTimeout)
	}
	if configured.Proxy != nil || !configured.DisableKeepAlives || !configured.DisableCompression || configured.ForceAttemptHTTP2 || len(configured.TLSNextProto) != 0 {
		t.Fatalf("automatic transport behavior is not disabled: %#v", configured)
	}
}

func TestBundleDigestsAreDeterministicAcrossIndependentAttempts(t *testing.T) {
	service, input := fixtureService(t, &fixtureLedger{}, func(context.Context, Manifest) (http.RoundTripper, error) {
		return roundTripFunc(responseRoundTrip(http.StatusOK, "application/xml", minimalWSDL())), nil
	}, nil)
	first, err := service.Fetch(context.Background(), inputArgs(input))
	if err != nil || first.Bundle == nil {
		t.Fatalf("first attempt: result=%#v err=%v", first, err)
	}
	second, err := service.Fetch(context.Background(), inputArgs(input))
	if err != nil || second.Bundle == nil {
		t.Fatalf("second attempt: result=%#v err=%v", second, err)
	}
	if first.AttemptID == second.AttemptID {
		t.Fatal("attempt IDs unexpectedly match")
	}
	if first.Bundle.BundleSHA256 != second.Bundle.BundleSHA256 {
		t.Fatalf("bundle digest drifted: first=%s second=%s", first.Bundle.BundleSHA256, second.Bundle.BundleSHA256)
	}
	if len(first.Bundle.Documents) != 1 || len(second.Bundle.Documents) != 1 || first.Bundle.Documents[0] != second.Bundle.Documents[0] {
		t.Fatalf("document evidence drifted: first=%#v second=%#v", first.Bundle.Documents, second.Bundle.Documents)
	}
}

type fixtureLedger struct {
	calls atomic.Int32
	err   error
}

func (l *fixtureLedger) Consume(Input) error { l.calls.Add(1); return l.err }

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func fixtureService(t *testing.T, ledger PermitLedger, factory func(context.Context, Manifest) (http.RoundTripper, error), limits *Limits) (*Service, Input) {
	t.Helper()
	manifest := productionFixtureManifest("https://gpi.invalid/invoice.wsdl?sap-client=100", t.TempDir())
	manifest.ExpectedBindingQName = "{urn:employee-shop:invoice}InvoiceBinding"
	if limits != nil {
		manifest.Limits = *limits
	}
	digest, err := ManifestSHA256(manifest)
	if err != nil {
		t.Fatalf("manifest digest: %v", err)
	}
	service, err := NewService(ServiceConfig{ActiveSystemID: SystemID, Manifest: manifest, ManifestSHA256: digest, Ledger: ledger, Credentials: Credentials{Username: "fixture-user", Password: "fixture-password"}, EvidenceKey: []byte("0123456789abcdef0123456789abcdef"), TransportFactory: factory})
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	return service, Input{SystemID: SystemID, ContractID: ContractID, RequestManifestSHA256: digest, PermitID: "6ba7b810-9dad-41d1-80b4-00c04fd430c8"}
}

func inputArgs(input Input) map[string]interface{} {
	return map[string]interface{}{"system_id": input.SystemID, "contract_id": input.ContractID, "request_manifest_sha256": input.RequestManifestSHA256, "permit_id": input.PermitID}
}
func assertHardStop(t *testing.T, result Result, code string, consumed bool, gets int) {
	t.Helper()
	if result.Outcome != "HARD_STOP" || result.HardStop == nil || result.HardStop.Code != code || result.PermitConsumed != consumed || result.NetworkGetsStarted != gets || result.Bundle != nil || result.EvidenceManifestSHA256 != nil {
		t.Fatalf("unexpected hard stop: result=%#v hard_stop=%+v want_code=%s", result, result.HardStop, code)
	}
}
func responseRoundTrip(status int, contentType, body string) func(*http.Request) (*http.Response, error) {
	return func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	}
}
func minimalWSDL() string {
	return `<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/" xmlns:tns="urn:employee-shop:invoice" xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/" targetNamespace="urn:employee-shop:invoice"><wsdl:message name="InvoiceRequest"/><wsdl:message name="InvoiceResponse"/><wsdl:portType name="InvoicePortType"><wsdl:operation name="SubmitInvoice"><wsdl:input message="tns:InvoiceRequest"/><wsdl:output message="tns:InvoiceResponse"/></wsdl:operation></wsdl:portType><wsdl:binding name="InvoiceBinding" type="tns:InvoicePortType"><soap:binding/><wsdl:operation name="SubmitInvoice"><soap:operation soapAction="urn:private:employee-shop:invoice"/></wsdl:operation></wsdl:binding><wsdl:service name="InvoiceService"><wsdl:port name="InvoicePort" binding="tns:InvoiceBinding"><soap:address location="https://gpi.invalid/private"/></wsdl:port></wsdl:service></wsdl:definitions>`
}
