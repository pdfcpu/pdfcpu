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

package pdfcpu

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/pkcs7"
)

func writeCertificateErrorTestFile(t *testing.T, name string, bb []byte) string {
	t.Helper()
	fileName := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(fileName, bb, 0600); err != nil {
		t.Fatal(err)
	}
	return fileName
}

func preserveCertificatePoolState(t *testing.T) {
	t.Helper()
	oldDir := model.TrustedCertDir
	trustedCertificatePool.Lock()
	oldCacheDir := trustedCertificatePool.dir
	oldLoaded := trustedCertificatePool.loaded
	oldPool := trustedCertificatePool.pool
	oldModelPool := model.UserCertPool
	trustedCertificatePool.dir = ""
	trustedCertificatePool.loaded = false
	trustedCertificatePool.pool = nil
	trustedCertificatePool.storeRevision = 0
	model.UserCertPool = nil
	trustedCertificatePool.Unlock()
	t.Cleanup(func() {
		model.TrustedCertDir = oldDir
		trustedCertificatePool.Lock()
		trustedCertificatePool.dir = oldCacheDir
		trustedCertificatePool.loaded = oldLoaded
		trustedCertificatePool.pool = oldPool
		trustedCertificatePool.storeRevision = model.CertificateStoreRevision()
		model.UserCertPool = oldModelPool
		trustedCertificatePool.Unlock()
	})
}

func installCertificatePoolTestFile(t *testing.T, dir string) {
	t.Helper()
	certs, err := LoadCertificatesFile(filepath.Join("model", "resources", "certs", "uk.p7c"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := SaveCertificates(certs, filepath.Join(dir, "uk.p7c")); err != nil {
		t.Fatal(err)
	}
}

// TestLoadCertificatesRetriesAfterFailure verifies failed loads do not poison the cache.
func TestLoadCertificatesRetriesAfterFailure(t *testing.T) {
	preserveCertificatePoolState(t)
	trustedDir := filepath.Join(t.TempDir(), "trusted")
	model.TrustedCertDir = trustedDir

	if err := LoadCertificates(); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}

	installCertificatePoolTestFile(t, trustedDir)
	if err := LoadCertificates(); err != nil {
		t.Fatalf("retry load: %v", err)
	}
	if pool := userCertificatePool(); pool == nil || len(pool.Subjects()) == 0 {
		t.Fatal("expected successfully loaded certificate pool")
	}
}

// TestLoadCertificatesPreservesSuccessfulPool verifies failed replacements do not alter active trust.
func TestLoadCertificatesPreservesSuccessfulPool(t *testing.T) {
	preserveCertificatePoolState(t)
	trustedDir := t.TempDir()
	model.TrustedCertDir = trustedDir
	installCertificatePoolTestFile(t, trustedDir)

	if err := LoadCertificates(); err != nil {
		t.Fatal(err)
	}
	oldPool := userCertificatePool()
	if oldPool == nil {
		t.Fatal("expected loaded certificate pool")
	}

	invalidFile := filepath.Join(trustedDir, "invalid.pem")
	if err := os.WriteFile(invalidFile, nil, 0600); err != nil {
		t.Fatal(err)
	}
	InvalidateCertificatePool()
	if err := LoadCertificates(); !errors.Is(err, ErrNoCertificates) {
		t.Fatalf("expected %v, got %v", ErrNoCertificates, err)
	}
	if pool := userCertificatePool(); pool != oldPool {
		t.Fatal("failed load replaced the last successful certificate pool")
	}

	if err := os.Remove(invalidFile); err != nil {
		t.Fatal(err)
	}
	if err := LoadCertificates(); err != nil {
		t.Fatalf("retry load: %v", err)
	}
	if pool := userCertificatePool(); pool == nil || pool == oldPool {
		t.Fatal("successful retry did not replace the certificate pool")
	}
}

// TestLoadCertificatesReloadsAfterReset verifies store replacement invalidates a successful cache.
func TestLoadCertificatesReloadsAfterReset(t *testing.T) {
	preserveCertificatePoolState(t)
	trustedDir := t.TempDir()
	model.TrustedCertDir = trustedDir
	installCertificatePoolTestFile(t, trustedDir)

	if err := LoadCertificates(); err != nil {
		t.Fatal(err)
	}
	oldPool := userCertificatePool()
	oldRevision := model.CertificateStoreRevision()

	if err := model.ResetCertificates(); err != nil {
		t.Fatal(err)
	}
	if revision := model.CertificateStoreRevision(); revision <= oldRevision {
		t.Fatalf("certificate store revision did not advance: got %d, previous %d", revision, oldRevision)
	}
	if err := LoadCertificates(); err != nil {
		t.Fatal(err)
	}
	if pool := userCertificatePool(); pool == nil || pool == oldPool {
		t.Fatal("certificate reset did not replace the cached pool")
	}
}

// TestLoadCertificatesConcurrent verifies concurrent first loads share one successful pool.
func TestLoadCertificatesConcurrent(t *testing.T) {
	preserveCertificatePoolState(t)
	trustedDir := t.TempDir()
	model.TrustedCertDir = trustedDir
	installCertificatePoolTestFile(t, trustedDir)

	const loadCount = 16
	start := make(chan struct{})
	errs := make(chan error, loadCount)
	for range loadCount {
		go func() {
			<-start
			errs <- LoadCertificates()
		}()
	}
	close(start)
	for range loadCount {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	if pool := userCertificatePool(); pool == nil || pool != model.UserCertPool {
		t.Fatal("concurrent loads did not publish one shared certificate pool")
	}
}

// TestLoadCertificatesFileReadContext verifies format-specific read failures preserve their cause.
func TestLoadCertificatesFileReadContext(t *testing.T) {
	tests := []struct {
		name string
		ext  string
		want string
	}{
		{name: "certificate", ext: ".crt", want: "read certificate"},
		{name: "PEM", ext: ".pem", want: "read PEM certificates"},
		{name: "PKCS7", ext: ".p7c", want: "read PKCS#7 certificates"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileName := filepath.Join(t.TempDir(), "missing"+tt.ext)
			_, err := LoadCertificatesFile(fileName)
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %q", tt.want, err)
			}
		})
	}
}

// TestLoadCertificatesFileRejectsCertificateFreeContainers verifies empty inputs share a sentinel.
func TestLoadCertificatesFileRejectsCertificateFreeContainers(t *testing.T) {
	noCertificatePEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte{0}})
	tests := []struct {
		name string
		file string
		data []byte
	}{
		{name: "empty certificate", file: "empty.crt"},
		{name: "empty PEM", file: "empty.pem"},
		{name: "empty PKCS7", file: "empty.p7c"},
		{name: "PEM without certificate", file: "private-key.pem", data: noCertificatePEM},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadCertificatesFile(writeCertificateErrorTestFile(t, tt.file, tt.data))
			if !errors.Is(err, ErrNoCertificates) {
				t.Fatalf("expected %v, got %v", ErrNoCertificates, err)
			}
		})
	}
}

// TestLoadCertificatesFileParseContext verifies certificate format boundaries add parse context.
func TestLoadCertificatesFileParseContext(t *testing.T) {
	invalidPEMCertificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte{0}})
	invalidPKCS7PEM := []byte(pkcs7PEMPrefix + "\n%%%\n" + pkcs7PEMSuffix + "\n")
	tests := []struct {
		name string
		file string
		data []byte
		want string
	}{
		{name: "DER certificate", file: "invalid.crt", data: []byte("invalid"), want: "parse DER certificate"},
		{name: "PEM certificate", file: "invalid.crt", data: invalidPEMCertificate, want: "parse PEM certificate"},
		{name: "PEM collection", file: "invalid.pem", data: invalidPEMCertificate, want: "parse PEM certificate 1"},
		{name: "PEM PKCS7", file: "invalid.p7c", data: invalidPKCS7PEM, want: "decode PEM PKCS#7"},
		{name: "DER PKCS7", file: "invalid.p7c", data: []byte("invalid"), want: "parse PKCS#7"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadCertificatesFile(writeCertificateErrorTestFile(t, tt.file, tt.data))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

// TestLoadCertificatesFilePKCS7PEM verifies strict PEM decoding around valid PKCS#7 data.
func TestLoadCertificatesFilePKCS7PEM(t *testing.T) {
	der, err := os.ReadFile(filepath.Join("model", "resources", "certs", "uk.p7c"))
	if err != nil {
		t.Fatal(err)
	}
	validPEM := pem.EncodeToMemory(&pem.Block{Type: pkcs7PEMType, Bytes: der})
	tests := []struct {
		name    string
		data    []byte
		wantErr string
	}{
		{name: "leading whitespace", data: append([]byte(" \t\r\n"), validPEM...)},
		{
			name:    "wrong block type",
			data:    pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
			wantErr: `unexpected PEM block type "CERTIFICATE"`,
		},
		{
			name:    "malformed base64",
			data:    []byte(pkcs7PEMPrefix + "\n%%%\n" + pkcs7PEMSuffix + "\n"),
			wantErr: "invalid PEM encoding",
		},
		{name: "trailing data", data: append(validPEM, []byte("trailing")...), wantErr: "trailing data after PEM block"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			certs, err := LoadCertificatesFile(writeCertificateErrorTestFile(t, "certificates.p7c", tt.data))
			if tt.wantErr == "" {
				if err != nil {
					t.Fatal(err)
				}
				if len(certs) == 0 {
					t.Fatal("expected certificates")
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "decode PEM PKCS#7: "+tt.wantErr) {
				t.Fatalf("expected %q, got %v", tt.wantErr, err)
			}
		})
	}
}

// TestCertificateSentinels verifies unsupported types and nil certificates remain discoverable.
func TestCertificateSentinels(t *testing.T) {
	if ErrUnknownFileType != ErrUnsupportedCertificateFile {
		t.Fatal("unsupported certificate compatibility sentinel has different identity")
	}
	if _, err := LoadCertificatesFile("certificate.txt"); !errors.Is(err, ErrUnsupportedCertificateFile) {
		t.Fatalf("expected %v, got %v", ErrUnsupportedCertificateFile, err)
	}
	if _, err := InspectCertificate(nil); !errors.Is(err, ErrMissingCertificate) {
		t.Fatalf("expected %v, got %v", ErrMissingCertificate, err)
	}
	if _, err := saveCertsAsPEM(nil, "unused.pem", true); !errors.Is(err, ErrNoCertificates) {
		t.Fatalf("expected %v, got %v", ErrNoCertificates, err)
	}
	if _, err := saveCertsAsP7C(nil, "unused.p7c", true); !errors.Is(err, ErrNoCertificates) {
		t.Fatalf("expected %v, got %v", ErrNoCertificates, err)
	}
	if err := SaveCertificates([]*x509.Certificate{nil}, "unused.p7c"); !errors.Is(err, ErrMissingCertificate) {
		t.Fatalf("expected %v, got %v", ErrMissingCertificate, err)
	}
}

// TestSaveCertificatesWriteContext verifies save failures preserve filesystem causes.
func TestSaveCertificatesWriteContext(t *testing.T) {
	certs, err := LoadCertificatesFile(filepath.Join("model", "resources", "certs", "uk.p7c"))
	if err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(t.TempDir(), "missing")
	tests := []struct {
		name string
		save func(string) error
		want string
	}{
		{
			name: "PEM",
			save: func(fileName string) error {
				_, err := saveCertsAsPEM(certs, fileName, true)
				return err
			},
			want: "write PEM certificates",
		},
		{
			name: "PKCS7",
			save: func(fileName string) error {
				_, err := saveCertsAsP7C(certs, fileName, true)
				return err
			},
			want: "write PKCS#7 certificates",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.save(filepath.Join(outDir, "out"))
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %q", tt.want, err)
			}
		})
	}
}

// TestSaveCertsAsP7CPreservesCertificateParseClassification verifies malformed
// certificate data retains PKCS#7 construction context and classification.
func TestSaveCertsAsP7CPreservesCertificateParseClassification(t *testing.T) {
	_, err := saveCertsAsP7C(
		[]*x509.Certificate{{Raw: []byte{0x30, 0x01}}},
		filepath.Join(t.TempDir(), "invalid.p7c"),
		true,
	)
	if !errors.Is(err, pkcs7.ErrCertificateParse) {
		t.Fatalf("got %v, want ErrCertificateParse", err)
	}
	for _, want := range []string{"add PKCS#7 certificate 1", "add certificate: parse DER", "x509:"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("got %v, want context %q", err, want)
		}
	}
}
