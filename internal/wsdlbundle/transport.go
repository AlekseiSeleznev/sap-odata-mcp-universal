package wsdlbundle

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type strictRoundTripper struct {
	transport      *http.Transport
	connectTimeout time.Duration
}

func (s *strictRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return s.transport.RoundTrip(request)
}

func newStrictTransport(ctx context.Context, manifest Manifest) (http.RoundTripper, error) {
	return newStrictTransportWithResolver(ctx, manifest, net.DefaultResolver.LookupIPAddr)
}

func newStrictTransportWithResolver(ctx context.Context, manifest Manifest, resolve func(context.Context, string) ([]net.IPAddr, error)) (http.RoundTripper, error) {
	root, err := url.Parse(manifest.RootURL)
	if err != nil {
		return nil, fmt.Errorf("invalid sealed root")
	}
	host := root.Hostname()
	port := root.Port()
	if port == "" {
		if root.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	var pinned net.IP
	if parsed := net.ParseIP(host); parsed != nil {
		pinned = parsed
	} else {
		addresses, err := resolve(ctx, host)
		if err != nil || len(addresses) == 0 {
			return nil, fmt.Errorf("sealed host resolution failed")
		}
		pinned = addresses[0].IP
	}
	if root.Scheme == "http" && !pinned.IsLoopback() {
		return nil, fmt.Errorf("plaintext tunnel host did not resolve to loopback")
	}
	dialer := &net.Dialer{Timeout: time.Duration(manifest.Limits.ConnectTimeoutMS) * time.Millisecond}
	dialContext := func(dialCtx context.Context, network, address string) (net.Conn, error) {
		requestedHost, requestedPort, err := net.SplitHostPort(address)
		if err != nil || !strings.EqualFold(requestedHost, host) || requestedPort != port {
			return nil, fmt.Errorf("dial target differs from sealed origin")
		}
		return dialer.DialContext(dialCtx, network, net.JoinHostPort(pinned.String(), port))
	}
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12, ServerName: host}
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           dialContext,
		ForceAttemptHTTP2:     false,
		TLSNextProto:          map[string]func(string, *tls.Conn) http.RoundTripper{},
		TLSClientConfig:       tlsConfig,
		TLSHandshakeTimeout:   time.Duration(manifest.Limits.TLSHandshakeTimeoutMS) * time.Millisecond,
		ResponseHeaderTimeout: time.Duration(manifest.Limits.ResponseHeaderTimeoutMS) * time.Millisecond,
		DisableCompression:    true,
		DisableKeepAlives:     true,
		MaxIdleConns:          0,
	}
	return &strictRoundTripper{transport: transport, connectTimeout: dialer.Timeout}, nil
}
