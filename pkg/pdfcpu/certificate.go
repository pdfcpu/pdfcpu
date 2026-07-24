/*
Copyright 2025 The pdfcpu Authors.

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
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/pkcs7"
)

// ErrUnsupportedCertificateFile signals an unsupported certificate file type.
var ErrUnsupportedCertificateFile = errors.New("unsupported file type")

// ErrUnknownFileType signals an unsupported certificate file type.
// Deprecated: Use ErrUnsupportedCertificateFile.
var ErrUnknownFileType = ErrUnsupportedCertificateFile

// ErrNoCertificates signals that a certificate file contains no usable certificates.
var ErrNoCertificates = errors.New("no certificates found")

// ErrMissingCertificate signals a missing required certificate.
var ErrMissingCertificate = errors.New("missing certificate")

type certificatePoolCache struct {
	sync.RWMutex
	dir           string
	loaded        bool
	pool          *x509.CertPool
	storeRevision uint64
}

var trustedCertificatePool certificatePoolCache

func addCertificateFileToPool(path string, certPool *x509.CertPool) error {
	certs, err := LoadCertificatesFile(path)
	if err != nil {
		if errors.Is(err, ErrUnsupportedCertificateFile) {
			return nil
		}
		return fmt.Errorf("load certificate file %q: %w", path, err)
	}
	for _, cert := range certs {
		certPool.AddCert(cert)
	}
	return nil
}

func buildCertificatePool(dir string) (*x509.CertPool, error) {
	certPool := x509.NewCertPool()
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("access %q: %w", path, err)
		}
		if d.IsDir() {
			return nil
		}
		return addCertificateFileToPool(path, certPool)
	})
	if err != nil {
		return nil, fmt.Errorf("walk trusted certificate directory: %w", err)
	}
	return certPool, nil
}

func buildCurrentCertificatePool(dir string) (*x509.CertPool, uint64, error) {
	for {
		storeRevision := model.CertificateStoreRevision()
		certPool, err := buildCertificatePool(dir)
		if err != nil {
			return nil, 0, err
		}
		if storeRevision == model.CertificateStoreRevision() {
			return certPool, storeRevision, nil
		}
	}
}

// LoadCertificates loads and caches certificates from the configured local
// certificate store for signature validation.
// Failed loads are not cached and may be retried.
func LoadCertificates() error {
	trustedCertificatePool.Lock()
	defer trustedCertificatePool.Unlock()

	dir := model.TrustedCertDir
	storeRevision := model.CertificateStoreRevision()
	if trustedCertificatePool.loaded &&
		trustedCertificatePool.dir == dir &&
		trustedCertificatePool.storeRevision == storeRevision {
		return nil
	}

	certPool, storeRevision, err := buildCurrentCertificatePool(dir)
	if err != nil {
		return err
	}
	trustedCertificatePool.dir = dir
	trustedCertificatePool.loaded = true
	trustedCertificatePool.pool = certPool
	trustedCertificatePool.storeRevision = storeRevision
	model.UserCertPool = certPool
	return nil
}

// InvalidateCertificatePool marks the cached trusted certificate pool for reloading.
// The last successfully loaded pool remains available to in-flight API calls.
func InvalidateCertificatePool() {
	trustedCertificatePool.Lock()
	trustedCertificatePool.loaded = false
	trustedCertificatePool.Unlock()
}

func userCertificatePool() *x509.CertPool {
	trustedCertificatePool.RLock()
	defer trustedCertificatePool.RUnlock()
	return trustedCertificatePool.pool
}

func loadSingleCertFile(filename string) (*x509.Certificate, error) {
	bb, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read certificate: %w", err)
	}
	if len(bb) == 0 {
		return nil, ErrNoCertificates
	}

	block, _ := pem.Decode(bb)
	if block != nil {
		if block.Type != "CERTIFICATE" {
			return nil, ErrNoCertificates
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse PEM certificate: %w", err)
		}
		return cert, nil
	}

	// DER
	cert, err := x509.ParseCertificate(bb)
	if err != nil {
		return nil, fmt.Errorf("parse DER certificate: %w", err)
	}
	return cert, nil
}

func loadCertsFromPEM(filename string) ([]*x509.Certificate, error) {
	bb, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read PEM certificates: %w", err)
	}

	var certs []*x509.Certificate

	for len(bb) > 0 {
		var block *pem.Block
		block, bb = pem.Decode(bb)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse PEM certificate %d: %w", len(certs)+1, err)
		}
		certs = append(certs, cert)
	}
	if len(certs) == 0 {
		return nil, ErrNoCertificates
	}
	return certs, nil
}

const (
	pkcs7PEMType   = "PKCS7"
	pkcs7PEMPrefix = "-----BEGIN PKCS7-----"
	pkcs7PEMSuffix = "-----END PKCS7-----"
)

func decodePKCS7PEM(bb []byte) ([]byte, error) {
	trimmed := bytes.TrimSpace(bb)
	block, rest := pem.Decode(trimmed)
	if block == nil {
		if bytes.HasPrefix(trimmed, []byte(pkcs7PEMPrefix)) {
			return nil, errors.New("invalid PEM encoding")
		}
		return bb, nil
	}
	if block.Type != pkcs7PEMType {
		return nil, fmt.Errorf("unexpected PEM block type %q", block.Type)
	}
	if len(block.Headers) != 0 {
		return nil, errors.New("PEM headers are not supported")
	}
	if len(bytes.TrimSpace(rest)) != 0 {
		return nil, errors.New("trailing data after PEM block")
	}
	return block.Bytes, nil
}

func loadCertsFromP7C(filename string) ([]*x509.Certificate, error) {
	bb, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read PKCS#7 certificates: %w", err)
	}
	if len(bb) == 0 {
		return nil, ErrNoCertificates
	}

	bb, err = decodePKCS7PEM(bb)
	if err != nil {
		return nil, fmt.Errorf("decode PEM PKCS#7: %w", err)
	}

	p7, err := pkcs7.Parse(bb)
	if err != nil {
		return nil, fmt.Errorf("parse PKCS#7: %w", err)
	}
	if p7 == nil || len(p7.Certificates) == 0 {
		return nil, ErrNoCertificates
	}
	for i, cert := range p7.Certificates {
		if cert == nil {
			return nil, fmt.Errorf("certificate %d: %w", i+1, ErrMissingCertificate)
		}
	}
	return p7.Certificates, nil
}

// LoadCertificatesFile loads certificates from filename.
func LoadCertificatesFile(filename string) ([]*x509.Certificate, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".crt", ".cer":
		cert, err := loadSingleCertFile(filename)
		if err != nil {
			return nil, err
		}
		return []*x509.Certificate{cert}, nil
	case ".p7c":
		return loadCertsFromP7C(filename)
	case ".pem":
		return loadCertsFromPEM(filename)
	default:
		return nil, ErrUnsupportedCertificateFile
	}
}

func saveCertsAsPEM(certs []*x509.Certificate, filename string, overwrite bool) (bool, error) {
	if len(certs) == 0 {
		return false, ErrNoCertificates
	}

	var buf bytes.Buffer
	for i, cert := range certs {
		block := &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		}
		if err := pem.Encode(&buf, block); err != nil {
			return false, fmt.Errorf("encode PEM certificate %d: %w", i+1, err)
		}
	}
	ok, err := Write(&buf, filename, overwrite)
	if err != nil {
		return false, fmt.Errorf("write PEM certificates: %w", err)
	}
	return ok, nil
}

func saveCertsAsP7C(certs []*x509.Certificate, filename string, overwrite bool) (bool, error) {
	// TODO encodeBase64 bool (PEM)

	if len(certs) == 0 {
		return false, ErrNoCertificates
	}

	p7, err := pkcs7.NewSignedData()
	if err != nil {
		return false, fmt.Errorf("create PKCS#7 container: %w", err)
	}

	for i, cert := range certs {
		if err := p7.AddCertificate(cert); err != nil {
			return false, fmt.Errorf("add PKCS#7 certificate %d: %w", i+1, err)
		}
	}

	bb, err := p7.Finish()
	if err != nil {
		return false, fmt.Errorf("encode PKCS#7 certificates: %w", err)
	}

	ok, err := Write(bytes.NewReader(bb), filename, overwrite)
	if err != nil {
		return false, fmt.Errorf("write PKCS#7 certificates: %w", err)
	}
	return ok, nil
}

// SaveCertificates saves certificates as a PKCS#7 container, atomically replacing outFile.
func SaveCertificates(certs []*x509.Certificate, outFile string) error {
	for i, cert := range certs {
		if cert == nil {
			return fmt.Errorf("certificate %d: %w", i+1, ErrMissingCertificate)
		}
	}
	_, err := saveCertsAsP7C(certs, outFile, true)
	if err != nil {
		return fmt.Errorf("save certificates: %w", err)
	}
	return nil
}

// InspectCertificate returns a string representation of cert.
func InspectCertificate(cert *x509.Certificate) (string, error) {
	if cert == nil {
		return "", ErrMissingCertificate
	}
	return model.CertString(cert), nil
}
