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
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hhrutter/pkcs7"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pkg/errors"
)

var ErrUnknownFileType = errors.New("pdfcpu: unsupported file type")

func loadSingleCertFile(filename string) (*x509.Certificate, error) {
	bb, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(bb)
	if block != nil && block.Type == "CERTIFICATE" {
		return x509.ParseCertificate(block.Bytes)
	}

	// DER
	return x509.ParseCertificate(bb)
}

func loadCertsFromPEM(filename string) ([]*x509.Certificate, error) {
	bb, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
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
			return nil, err
		}
		certs = append(certs, cert)
	}

	return certs, nil
}

const PKCS7_PREFIX = "-----BEGIN PKCS7-----"
const PKCS7_SUFFIX = "-----END PKCS7-----"

func isPEMEncoded(s string) bool {
	s = strings.TrimRight(s, " \t\r\n")
	return strings.HasPrefix(s, PKCS7_PREFIX) && strings.HasSuffix(s, PKCS7_SUFFIX)
}

func decodePKCS7Block(s string) ([]byte, error) {
	start := strings.Index(s, PKCS7_PREFIX)
	end := strings.Index(s, PKCS7_SUFFIX)

	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("decodePKCS7Block: PEM block not found")
	}

	s = s[start+len(PKCS7_PREFIX) : end]
	s = strings.TrimSpace(s)

	return base64.StdEncoding.DecodeString(s)
}

func loadCertsFromP7C(filename string) ([]*x509.Certificate, error) {
	bb, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	s := string(bb)
	if isPEMEncoded(s) {
		bb, err = decodePKCS7Block(s)
		if err != nil {
			return nil, err
		}
	} // else DER (binary)

	p7, err := pkcs7.Parse(bb)
	if err != nil {
		return nil, err
	}

	return p7.Certificates, nil
}

func LoadCertificates(filename string) ([]*x509.Certificate, error) {
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
		return nil, ErrUnknownFileType
	}
}

func loadCertificatesToCertPool(path string, certPool *x509.CertPool, n *int) error {
	certs, err := LoadCertificates(path)
	if err != nil {
		if err == ErrUnknownFileType {
			return nil
		}
		return err
	}
	for _, cert := range certs {
		certPool.AddCert(cert)
	}
	*n += len(certs)
	return nil
}

func LoadCertificatesToCertPool(dir string, certPool *x509.CertPool) (int, error) {
	n := 0
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		return loadCertificatesToCertPool(path, certPool, &n)
	})
	return n, err
}

func saveCertsAsPEM(certs []*x509.Certificate, filename string, overwrite bool) (bool, error) {
	if len(certs) == 0 {
		return false, errors.New("no certificates to save")
	}

	if !overwrite {
		if _, err := os.Stat(filename); err == nil {
			return false, nil
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		return false, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	for _, cert := range certs {
		block := &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		}
		if err := pem.Encode(file, block); err != nil {
			return false, err
		}
	}

	return true, nil
}

func saveCertsAsP7C(certs []*x509.Certificate, filename string, overwrite bool) (bool, error) {
	// TODO encodeBase64 bool (PEM)

	if len(certs) == 0 {
		return false, errors.New("no certificates to save")
	}

	p7, err := pkcs7.NewSignedData(nil)
	if err != nil {
		return false, err
	}

	for _, cert := range certs {
		p7.AddCertificate(cert)
	}

	bb, err := p7.Finish()
	if err != nil {
		return false, err
	}

	return Write(bytes.NewReader(bb), filename, overwrite)
}

func ImportCertificate(inFile string, overwrite bool) (int, bool, error) {
	certs, err := LoadCertificates(inFile)
	if err != nil {
		return 0, false, err
	}

	// We have validated the incoming cert info.

	enforceP7C := true // takes less disk space

	base := filepath.Base(inFile)
	outFileNoExt := base[:len(base)-len(filepath.Ext(base))]
	outFile := outFileNoExt + ".p7c"
	outFile = filepath.Join(model.CertDir, outFile)

	if enforceP7C {
		// Write certs as .p7c to certDir.
		ok, err := saveCertsAsP7C(certs, outFile, overwrite)
		if err != nil {
			return 0, false, err
		}
		return len(certs), ok, nil
	}

	// Copy inFile to certDir (may be .pem or p7c)
	ok, err := CopyFile(inFile, outFile, overwrite)
	if err != nil {
		return 0, false, err
	}
	return len(certs), ok, nil
}

func InspectCertificate(cert *x509.Certificate) (string, error) {
	return model.CertString(cert), nil
}
