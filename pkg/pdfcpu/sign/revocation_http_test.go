/*
Copyright 2026 The pdfcpu Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sign

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type revocationTestResolver struct {
	ips []net.IPAddr
	err error
}

func (r revocationTestResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	return r.ips, r.err
}

func revocationTestIPs(values ...string) []net.IPAddr {
	ips := make([]net.IPAddr, 0, len(values))
	for _, value := range values {
		ips = append(ips, net.IPAddr{IP: net.ParseIP(value)})
	}
	return ips
}

func revocationTestHost(t *testing.T, rawURL string) string {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	return u.Hostname()
}

// TestValidateRevocationURL verifies the revocation client accepts only credential-free HTTP URLs.
func TestValidateRevocationURL(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		wantErr bool
	}{
		{"HTTP", "http://crl.example.com/list", false},
		{"HTTPS", "https://ocsp.example.com", false},
		{"Credentials", "https://user:secret@ocsp.example.com", true},
		{"File", "file:///tmp/list.crl", true},
		{"MissingHost", "https:///list.crl", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.rawURL)
			if err != nil {
				t.Fatal(err)
			}
			err = validateRevocationURL(u)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validate URL: got %v, want error %t", err, tt.wantErr)
			}
		})
	}
}

// TestValidateRevocationIPs verifies local address classes are blocked unless their exact host is trusted.
func TestValidateRevocationIPs(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		ips     []net.IPAddr
		allowed []string
		wantErr bool
	}{
		{"Public", "crl.example.com", revocationTestIPs("8.8.8.8"), nil, false},
		{"LoopbackIPv4", "localhost", revocationTestIPs("127.0.0.1"), nil, true},
		{"LoopbackIPv6", "localhost", revocationTestIPs("::1"), nil, true},
		{"PrivateIPv4", "pki.example.corp", revocationTestIPs("10.0.0.1"), nil, true},
		{"PrivateIPv6", "pki.example.corp", revocationTestIPs("fd00::1"), nil, true},
		{"LinkLocal", "pki.example.corp", revocationTestIPs("169.254.1.1"), nil, true},
		{"Mixed", "crl.example.com", revocationTestIPs("8.8.8.8", "127.0.0.1"), nil, true},
		{"AllowedPrivate", "pki.example.corp", revocationTestIPs("10.0.0.1"), []string{"PKI.EXAMPLE.CORP."}, false},
		{"Unresolved", "crl.example.com", nil, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRevocationIPs(tt.host, tt.ips, allowedRevocationHostSet(tt.allowed))
			if (err != nil) != tt.wantErr {
				t.Fatalf("validate IPs: got %v, want error %t", err, tt.wantErr)
			}
		})
	}
}

// TestRevocationDialContextDialsValidatedIP verifies DNS is not repeated after policy validation.
func TestRevocationDialContextDialsValidatedIP(t *testing.T) {
	var target string
	dial := func(_ context.Context, _, addr string) (net.Conn, error) {
		target = addr
		client, server := net.Pipe()
		t.Cleanup(func() {
			client.Close()
			server.Close()
		})
		return client, nil
	}
	dialContext := revocationDialContext(
		revocationTestResolver{ips: revocationTestIPs("8.8.8.8")},
		dial,
		nil,
	)

	conn, err := dialContext(context.Background(), "tcp", "crl.example.com:443")
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
	if target != "8.8.8.8:443" {
		t.Fatalf("dial target: got %q, want %q", target, "8.8.8.8:443")
	}
}

// TestRevocationDialContextBlocksPrivateDNS verifies address policy runs before any connection attempt.
func TestRevocationDialContextBlocksPrivateDNS(t *testing.T) {
	called := false
	dial := func(context.Context, string, string) (net.Conn, error) {
		called = true
		return nil, errors.New("unexpected dial")
	}
	dialContext := revocationDialContext(
		revocationTestResolver{ips: revocationTestIPs("10.0.0.1")},
		dial,
		nil,
	)

	_, err := dialContext(context.Background(), "tcp", "pki.example.corp:80")
	if err == nil || !strings.Contains(err.Error(), "disallowed address") {
		t.Fatalf("expected disallowed address error, got %v", err)
	}
	if called {
		t.Fatal("dial called for a blocked address")
	}
}

// TestRevocationRedirect verifies redirects retain URL validation and a finite hop limit.
func TestRevocationRedirect(t *testing.T) {
	valid, err := http.NewRequest(http.MethodGet, "https://crl.example.com/list", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := revocationRedirect(valid, nil); err != nil {
		t.Fatal(err)
	}

	credentials, err := http.NewRequest(http.MethodGet, "https://user:secret@crl.example.com/list", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := revocationRedirect(credentials, nil); err == nil {
		t.Fatal("expected credential-bearing redirect rejection")
	}

	via := make([]*http.Request, maxRevocationRedirects)
	if err := revocationRedirect(valid, via); err == nil {
		t.Fatal("expected redirect limit rejection")
	}
}

// TestRevocationHTTPClientUsesSafeDefaults verifies proxy inheritance is disabled and missing timeouts are repaired.
func TestRevocationHTTPClientUsesSafeDefaults(t *testing.T) {
	client := revocationHTTPClient(0, nil)
	if client.Timeout != defaultRevocationHTTPTimeout {
		t.Fatalf("client timeout: got %v, want %v", client.Timeout, defaultRevocationHTTPTimeout)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type: %T", client.Transport)
	}
	if transport.Proxy != nil {
		t.Fatal("revocation transport must not inherit proxy settings")
	}
	if transport.ResponseHeaderTimeout != defaultRevocationHTTPTimeout {
		t.Fatalf("response header timeout: got %v, want %v", transport.ResponseHeaderTimeout, defaultRevocationHTTPTimeout)
	}

	configured := revocationHTTPClient(3*time.Second, nil)
	if configured.Timeout != 3*time.Second {
		t.Fatalf("configured timeout: got %v, want %v", configured.Timeout, 3*time.Second)
	}
}
