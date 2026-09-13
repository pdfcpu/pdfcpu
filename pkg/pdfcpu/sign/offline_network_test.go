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
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// TestOfflineRevocationSkipsDNSAndHTTP verifies both checkers and their fallback stay offline.
func TestOfflineRevocationSkipsDNSAndHTTP(t *testing.T) {
	var lookups, requests atomic.Int32
	old := net.DefaultResolver
	net.DefaultResolver = &net.Resolver{PreferGo: true, Dial: func(context.Context, string, string) (net.Conn, error) {
		lookups.Add(1)
		return nil, errors.New("test DNS blocked")
	}}
	t.Cleanup(func() { net.DefaultResolver = old })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	issuer, _, cert := testCurrentCRLChain(t, "Offline Network Issuer")
	cert.CRLDistributionPoints = []string{"http://pdfcpu-offline-crl.invalid/list", server.URL + "/crl"}
	cert.OCSPServer = []string{"http://pdfcpu-offline-ocsp.invalid/status", server.URL + "/ocsp"}
	conf := model.NewStatelessConfiguration()
	conf.AllowedRevocationHosts = []string{u.Hostname()}
	conf.Offline = true
	archived := [][]byte{{0}}
	for _, checker := range []int{model.CRL, model.OCSP} {
		conf.PreferredCertRevocationChecker = checker
		_, err := checkCertificateRevocation(t.Context(), cert, issuer, x509.NewCertPool(), &model.Signer{}, archived, archived, conf)
		if err == nil || !strings.Contains(err.Error(), "CRL: offline") || !strings.Contains(err.Error(), "OCSP: offline") {
			t.Fatalf("checker %v: expected offline fallback errors, got %v", checker, err)
		}
	}
	if lookups.Load() != 0 || requests.Load() != 0 {
		t.Fatalf("offline DNS=%d, HTTP=%d", lookups.Load(), requests.Load())
	}
	conf.Offline = false
	_, _ = checkCertificateRevocation(t.Context(), cert, issuer, x509.NewCertPool(), &model.Signer{}, nil, nil, conf)
	if lookups.Load() == 0 || requests.Load() == 0 {
		t.Fatalf("online controls DNS=%d, HTTP=%d", lookups.Load(), requests.Load())
	}
}
