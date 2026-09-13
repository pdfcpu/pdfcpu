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
	"errors"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type schemaConfigurationValues struct {
	Version                         string
	SchemaVersion                   int
	CheckFileNameExt                bool
	Reader15                        bool
	DecodeAllStreams                bool
	ValidationMode                  int
	PostProcessValidate             bool
	Eol                             string
	WriteObjectStream               bool
	WriteXRefStream                 bool
	EncryptUsingAES                 bool
	EncryptKeyLength                int
	Permissions                     PermissionFlags
	Unit                            types.DisplayUnit
	TimestampFormat                 string
	DateFormat                      string
	Optimize                        bool
	OptimizeBeforeWriting           bool
	OptimizeResourceDicts           bool
	OptimizeDuplicateContentStreams bool
	CreateBookmarks                 bool
	NeedAppearances                 bool
	Offline                         bool
	Timeout                         int
	TimeoutCRL                      int
	TimeoutOCSP                     int
	PreferredCertRevocationChecker  int
	FormFieldListMaxColWidth        int
	Limits                          ResourceLimits
}

func schemaValues(conf *Configuration) schemaConfigurationValues {
	return schemaConfigurationValues{
		Version:                         conf.Version,
		SchemaVersion:                   conf.SchemaVersion,
		CheckFileNameExt:                conf.CheckFileNameExt,
		Reader15:                        conf.Reader15,
		DecodeAllStreams:                conf.DecodeAllStreams,
		ValidationMode:                  conf.ValidationMode,
		PostProcessValidate:             conf.PostProcessValidate,
		Eol:                             conf.Eol,
		WriteObjectStream:               conf.WriteObjectStream,
		WriteXRefStream:                 conf.WriteXRefStream,
		EncryptUsingAES:                 conf.EncryptUsingAES,
		EncryptKeyLength:                conf.EncryptKeyLength,
		Permissions:                     conf.Permissions,
		Unit:                            conf.Unit,
		TimestampFormat:                 conf.TimestampFormat,
		DateFormat:                      conf.DateFormat,
		Optimize:                        conf.Optimize,
		OptimizeBeforeWriting:           conf.OptimizeBeforeWriting,
		OptimizeResourceDicts:           conf.OptimizeResourceDicts,
		OptimizeDuplicateContentStreams: conf.OptimizeDuplicateContentStreams,
		CreateBookmarks:                 conf.CreateBookmarks,
		NeedAppearances:                 conf.NeedAppearances,
		Offline:                         conf.Offline,
		Timeout:                         conf.Timeout,
		TimeoutCRL:                      conf.TimeoutCRL,
		TimeoutOCSP:                     conf.TimeoutOCSP,
		PreferredCertRevocationChecker:  conf.PreferredCertRevocationChecker,
		FormFieldListMaxColWidth:        conf.FormFieldListMaxColWidth,
		Limits:                          conf.Limits,
	}
}

func configurationWithSchemaLine(t *testing.T, line string) []byte {
	t.Helper()
	current := []byte("schemaVersion: 1")
	if !bytes.Contains(configFileBytes, current) {
		t.Fatal("embedded configuration has no schemaVersion")
	}
	return bytes.Replace(configFileBytes, current, []byte(line), 1)
}

func TestConfigurationSchemaVersionParsing(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantVersion int
		wantErr     bool
	}{
		{"legacy absent", "", ConfigurationSchemaVersionLegacy, false},
		{"current", "schemaVersion: 1", ConfigurationSchemaVersionCurrent, false},
		{"zero", "schemaVersion: 0", 0, true},
		{"negative", "schemaVersion: -1", 0, true},
		{"newer", "schemaVersion: 2", 0, true},
		{"string", "schemaVersion: one", 0, true},
		{"quoted integer", `schemaVersion: "1"`, 0, true},
		{"fractional", "schemaVersion: 1.5", 0, true},
		{"boolean", "schemaVersion: true", 0, true},
		{"null", "schemaVersion: null", 0, true},
		{"overflow", "schemaVersion: 999999999999999999999999", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf, err := readConfiguration(bytes.NewReader(configurationWithSchemaLine(t, tt.line)), "config.yml")
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidConfigurationSchema) {
					t.Fatalf("parse schema: got %v, want %v", err, ErrInvalidConfigurationSchema)
				}
				return
			}
			if err != nil {
				t.Fatalf("parse schema: %v", err)
			}
			if conf.SchemaVersion != tt.wantVersion {
				t.Fatalf("schema version: got %d, want %d", conf.SchemaVersion, tt.wantVersion)
			}
		})
	}
}

func TestLegacyConfigurationAcceptsProductVersionMetadata(t *testing.T) {
	legacy := configurationWithSchemaLine(t, "")
	bb := append([]byte("version: legacy-product-version\n"), legacy...)
	conf, err := readConfiguration(bytes.NewReader(bb), "config.yml")
	if err != nil {
		t.Fatal(err)
	}
	if conf.SchemaVersion != ConfigurationSchemaVersionLegacy {
		t.Fatalf("schema version: got %d, want %d", conf.SchemaVersion, ConfigurationSchemaVersionLegacy)
	}
	if conf.Version != "legacy-product-version" {
		t.Fatalf("product version: got %q", conf.Version)
	}
}

func TestGeneratedConfigurationOmitsProductVersion(t *testing.T) {
	bb := defaultConfigurationFileBytes()
	if bytes.Contains(bb, []byte("\nversion:")) {
		t.Fatalf("generated configuration contains product version:\n%s", bb)
	}
}

func TestBuiltInConfigurationsUseCurrentSchema(t *testing.T) {
	tests := []struct {
		name string
		conf *Configuration
	}{
		{"generated", func() *Configuration {
			conf, err := readConfiguration(bytes.NewReader(defaultConfigurationFileBytes()), "config.yml")
			if err != nil {
				t.Fatal(err)
			}
			return conf
		}()},
		{"stateless", NewStatelessConfiguration()},
	}
	for _, tt := range tests {
		if tt.conf.SchemaVersion != ConfigurationSchemaVersionCurrent {
			t.Fatalf(
				"%s schema version: got %d, want %d",
				tt.name,
				tt.conf.SchemaVersion,
				ConfigurationSchemaVersionCurrent,
			)
		}
	}
}

func TestSchema1OmittedValuesUseBuiltInDefaults(t *testing.T) {
	conf, err := readConfiguration(strings.NewReader("schemaVersion: 1\n"), "minimal.yml")
	if err != nil {
		t.Fatal(err)
	}
	want := NewStatelessConfiguration()
	if got, want := schemaValues(conf), schemaValues(want); got != want {
		t.Fatalf("effective configuration:\n got: %+v\nwant: %+v", got, want)
	}
	if conf.Path != "minimal.yml" {
		t.Fatalf("path: got %q, want %q", conf.Path, "minimal.yml")
	}
	if conf.AllowedRevocationHosts != nil {
		t.Fatalf("allowed revocation hosts: got %v, want nil", conf.AllowedRevocationHosts)
	}
}

func assertSchema1GeneralOverlay(t *testing.T, conf *Configuration) {
	t.Helper()
	if conf.CheckFileNameExt || conf.ValidationMode != ValidationStrict || !conf.PostProcessValidate {
		t.Fatalf("general overlay not applied: %+v", schemaValues(conf))
	}
	if conf.Eol != types.EolCRLF || conf.EncryptUsingAES || conf.EncryptKeyLength != 128 {
		t.Fatalf("output overlay not applied: %+v", schemaValues(conf))
	}
	if conf.Permissions != PermissionsAll || conf.Unit != types.MILLIMETRES {
		t.Fatalf("format overlay not applied: %+v", schemaValues(conf))
	}
}

func assertSchema1RuntimeOverlay(t *testing.T, conf *Configuration) {
	t.Helper()
	if conf.Optimize || conf.OptimizeBeforeWriting || conf.Timeout != 17 {
		t.Fatalf("runtime overlay not applied: %+v", schemaValues(conf))
	}
	if conf.PreferredCertRevocationChecker != OCSP || conf.FormFieldListMaxColWidth != 42 {
		t.Fatalf("certificate overlay not applied: %+v", schemaValues(conf))
	}
	if got, want := conf.Limits.MaxObjectBytes, int64(2<<20); got != want {
		t.Fatalf("max object bytes: got %d, want %d", got, want)
	}
	if got, want := conf.Limits.MaxStreamBytes, int64(64<<20); got != want {
		t.Fatalf("max stream bytes: got %d, want %d", got, want)
	}
}

func assertSchema1OverlayPreservesDefaults(t *testing.T, conf *Configuration) {
	t.Helper()
	if got, want := strings.Join(conf.AllowedRevocationHosts, ","), "ocsp.example.test,crl.example.test"; got != want {
		t.Fatalf("allowed revocation hosts: got %q, want %q", got, want)
	}
	if !conf.Reader15 || !conf.WriteObjectStream || conf.Limits.MaxDecodeBytes != DefaultResourceLimits().MaxDecodeBytes {
		t.Fatalf("omitted defaults not preserved: %+v", schemaValues(conf))
	}
}

func TestSchema1SuppliedValuesOverlayBuiltInDefaults(t *testing.T) {
	const input = `schemaVersion: 1
checkFileNameExt: false
validationMode: ValidationStrict
postProcessValidate: true
eol: EolCRLF
encryptUsingAES: false
encryptKeyLength: 128
permissions: 0xFFFF
unit: mm
optimize: false
optimizeBeforeWriting: false
timeout: 17
allowedRevocationHosts: [ocsp.example.test, crl.example.test]
preferredCertRevocationChecker: ocsp
formFieldListMaxColWidth: 42
maxObjectBytes: 2 MB
maxStreamBytes: 64 MB
`
	conf, err := readConfiguration(strings.NewReader(input), "overlay.yml")
	if err != nil {
		t.Fatal(err)
	}
	assertSchema1GeneralOverlay(t, conf)
	assertSchema1RuntimeOverlay(t, conf)
	assertSchema1OverlayPreservesDefaults(t, conf)
}

func TestSchema1RejectsUnknownAndDuplicateKeys(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			"unknown",
			"schemaVersion: 1\nfutureSetting: true\n",
			`invalid schema 1 configuration: unknown key "futureSetting"`,
		},
		{
			"legacy capitalization",
			"schemaVersion: 1\nFormFieldListMaxColWidth: 10\n",
			`invalid schema 1 configuration: unknown key "FormFieldListMaxColWidth"`,
		},
		{
			"duplicate value",
			"schemaVersion: 1\ntimeout: 5\ntimeout: 10\n",
			`invalid schema 1 configuration: duplicate key "timeout"`,
		},
		{
			"duplicate schema",
			"schemaVersion: 1\nschemaVersion: 1\n",
			`invalid schema 1 configuration: duplicate key "schemaVersion"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := readConfiguration(strings.NewReader(tt.input), "config.yml")
			if err == nil || err.Error() != tt.want {
				t.Fatalf("parse error: got %v, want %q", err, tt.want)
			}
		})
	}
}

func TestSchema0KeepsUnknownKeyCompatibility(t *testing.T) {
	legacy := append(configurationWithSchemaLine(t, ""), []byte("\nfutureSetting: true\n")...)
	conf, err := readConfiguration(bytes.NewReader(legacy), "config.yml")
	if err != nil {
		t.Fatal(err)
	}
	if conf.SchemaVersion != ConfigurationSchemaVersionLegacy {
		t.Fatalf("schema version: got %d, want %d", conf.SchemaVersion, ConfigurationSchemaVersionLegacy)
	}
}

// TestSchema1ObjectBufferLimit rejects invalid limits and preserves defaults for older schema-1 files.
func TestSchema1ObjectBufferLimit(t *testing.T) {
	for _, value := range []string{"0", "-1", "0 MB", "9223372036854775808"} {
		_, err := readConfiguration(strings.NewReader("schemaVersion: 1\nmaxObjectBytes: "+value+"\n"), "limits.yml")
		if err == nil {
			t.Fatalf("accepted invalid maxObjectBytes %q", value)
		}
	}
	conf, err := readConfiguration(strings.NewReader("schemaVersion: 1\n"), "limits.yml")
	if err != nil {
		t.Fatal(err)
	}
	if conf.Limits.MaxObjectBytes != 64<<20 {
		t.Fatalf("default maxObjectBytes: %d", conf.Limits.MaxObjectBytes)
	}
}

// TestSchema1InputLimit verifies zero, quantities, defaulting and invalid input limits.
func TestSchema1InputLimit(t *testing.T) {
	tests := []struct {
		value string
		want  int64
		valid bool
	}{
		{"0", 0, true}, {`"0"`, 0, true}, {"3", 3, true}, {"2 MB", 2 << 20, true},
		{"-1", 0, false}, {"9223372036854775808", 0, false}, {"0 MB", 0, false},
	}
	for _, tt := range tests {
		conf, err := readConfiguration(strings.NewReader("schemaVersion: 1\nmaxInputBytes: "+tt.value+"\n"), "limits.yml")
		if !tt.valid {
			if err == nil {
				t.Fatalf("accepted %q", tt.value)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: %v", tt.value, err)
		}
		if conf.Limits.MaxInputBytes != tt.want {
			t.Fatalf("%s: got %d, want %d", tt.value, conf.Limits.MaxInputBytes, tt.want)
		}
	}
}
