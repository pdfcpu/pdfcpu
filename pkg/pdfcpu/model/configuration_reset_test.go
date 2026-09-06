//go:build !js

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

package model

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/font"
)

func replaceResetConfigurationSetting(t *testing.T, path, old, replacement string) {
	t.Helper()
	bb, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	updated := bytes.Replace(bb, []byte(old), []byte(replacement), 1)
	if bytes.Equal(updated, bb) {
		t.Fatalf("configuration does not contain %q", old)
	}
	if err := os.WriteFile(path, updated, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestResetConfigurationUsesSelectedRootWithoutChangingGlobals(t *testing.T) {
	preserveConfigurationGlobals(t)

	cached := &Configuration{Path: "cached.yml"}
	loadedDefaultConfig = cached
	ConfigPath = "cached-root"
	font.UserFontDir = "cached-fonts"
	TrustedCertDir = "cached-certs"

	root := t.TempDir()
	conf, err := LoadConfiguration(root)
	if err != nil {
		t.Fatalf("initialize configuration: %v", err)
	}
	replaceResetConfigurationSetting(t, conf.Path, "offline: false", "offline: true")
	fontMarker := filepath.Join(root, "pdfcpu", "fonts", "keep.font")
	certificateMarker := filepath.Join(root, "pdfcpu", "certs", "keep.pem")
	for _, path := range []string{fontMarker, certificateMarker} {
		if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	reset, err := ResetConfiguration(root)
	if err != nil {
		t.Fatalf("reset configuration: %v", err)
	}
	if reset.Path != conf.Path || reset.SchemaVersion != ConfigurationSchemaVersionCurrent || reset.Offline {
		t.Fatalf("unexpected reset configuration: %+v", reset)
	}
	for _, path := range []string{fontMarker, certificateMarker} {
		bb, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read preserved resource %q: %v", path, err)
		}
		if string(bb) != "keep" {
			t.Fatalf("resource %q changed: %q", path, bb)
		}
	}
	requireUnchangedConfigurationGlobals(t, cached)
}

func TestResetConfigurationRequiresRoot(t *testing.T) {
	if _, err := ResetConfiguration(""); err == nil {
		t.Fatal("reset configuration unexpectedly accepted an empty root")
	}
}
