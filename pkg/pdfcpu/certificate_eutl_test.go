//go:build pdfcpu_eutl
// +build pdfcpu_eutl

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
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// TestLoadCertificatesReloadsBundledEUTL verifies embedded initialization replaces an empty cached pool.
func TestLoadCertificatesReloadsBundledEUTL(t *testing.T) {
	preserveCertificatePoolState(t)
	configRoot := t.TempDir()
	trustedDir := filepath.Join(configRoot, "pdfcpu", "certs")
	if err := os.MkdirAll(trustedDir, 0755); err != nil {
		t.Fatal(err)
	}
	model.TrustedCertDir = trustedDir

	if err := LoadCertificates(); err != nil {
		t.Fatal(err)
	}
	oldPool := userCertificatePool()
	if oldPool == nil || len(oldPool.Subjects()) != 0 {
		t.Fatal("expected an initially empty certificate pool")
	}

	if err := model.EnsureDefaultConfigAt(configRoot, false); err != nil {
		t.Fatal(err)
	}
	if err := LoadCertificates(); err != nil {
		t.Fatal(err)
	}
	if pool := userCertificatePool(); pool == nil || pool == oldPool || len(pool.Subjects()) == 0 {
		t.Fatal("embedded EUTL initialization did not replace the cached pool")
	}
}
