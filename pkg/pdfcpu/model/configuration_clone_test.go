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
	"reflect"
	"slices"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func containsMutableReferences(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return true
	case reflect.Array:
		return containsMutableReferences(t.Elem())
	case reflect.Struct:
		for i := range t.NumField() {
			if containsMutableReferences(t.Field(i).Type) {
				return true
			}
		}
	}
	return false
}

type configurationScalarValues struct {
	resources                       configurationResources
	Path                            string
	CreationDate                    string
	Version                         string
	SchemaVersion                   int
	CheckFileNameExt                bool
	Reader15                        bool
	DecodeAllStreams                bool
	ValidationMode                  int
	UnsupportedResourcePolicy       UnsupportedResourcePolicy
	PostProcessValidate             bool
	PreserveInfoDict                bool
	ValidateLinks                   bool
	Eol                             string
	WriteObjectStream               bool
	WriteXRefStream                 bool
	StatsFileName                   string
	UserPW                          string
	OwnerPW                         string
	PrivateKeyPW                    string
	EncryptUsingAES                 bool
	EncryptKeyLength                int
	Permissions                     PermissionFlags
	Cmd                             CommandMode
	Unit                            types.DisplayUnit
	TimestampFormat                 string
	DateFormat                      string
	Optimize                        bool
	OptimizeBeforeWriting           bool
	OptimizeResourceDicts           bool
	OptimizeDuplicateContentStreams bool
	CreateBookmarks                 bool
	MergeBookmarkMode               MergeBookmarkMode
	NeedAppearances                 bool
	Offline                         bool
	Timeout                         int
	TimeoutCRL                      int
	TimeoutOCSP                     int
	PreferredCertRevocationChecker  int
	FormFieldListMaxColWidth        int
	Limits                          ResourceLimits
	RemoveEncryption                bool
	RemoveSignatures                bool
}

func scalarValues(c *Configuration) configurationScalarValues {
	return configurationScalarValues{
		resources:                       c.resources,
		Path:                            c.Path,
		CreationDate:                    c.CreationDate,
		Version:                         c.Version,
		SchemaVersion:                   c.SchemaVersion,
		CheckFileNameExt:                c.CheckFileNameExt,
		Reader15:                        c.Reader15,
		DecodeAllStreams:                c.DecodeAllStreams,
		ValidationMode:                  c.ValidationMode,
		UnsupportedResourcePolicy:       c.UnsupportedResourcePolicy,
		PostProcessValidate:             c.PostProcessValidate,
		PreserveInfoDict:                c.PreserveInfoDict,
		ValidateLinks:                   c.ValidateLinks,
		Eol:                             c.Eol,
		WriteObjectStream:               c.WriteObjectStream,
		WriteXRefStream:                 c.WriteXRefStream,
		StatsFileName:                   c.StatsFileName,
		UserPW:                          c.UserPW,
		OwnerPW:                         c.OwnerPW,
		PrivateKeyPW:                    c.PrivateKeyPW,
		EncryptUsingAES:                 c.EncryptUsingAES,
		EncryptKeyLength:                c.EncryptKeyLength,
		Permissions:                     c.Permissions,
		Cmd:                             c.Cmd,
		Unit:                            c.Unit,
		TimestampFormat:                 c.TimestampFormat,
		DateFormat:                      c.DateFormat,
		Optimize:                        c.Optimize,
		OptimizeBeforeWriting:           c.OptimizeBeforeWriting,
		OptimizeResourceDicts:           c.OptimizeResourceDicts,
		OptimizeDuplicateContentStreams: c.OptimizeDuplicateContentStreams,
		CreateBookmarks:                 c.CreateBookmarks,
		MergeBookmarkMode:               c.MergeBookmarkMode,
		NeedAppearances:                 c.NeedAppearances,
		Offline:                         c.Offline,
		Timeout:                         c.Timeout,
		TimeoutCRL:                      c.TimeoutCRL,
		TimeoutOCSP:                     c.TimeoutOCSP,
		PreferredCertRevocationChecker:  c.PreferredCertRevocationChecker,
		FormFieldListMaxColWidth:        c.FormFieldListMaxColWidth,
		Limits:                          c.Limits,
		RemoveEncryption:                c.RemoveEncryption,
		RemoveSignatures:                c.RemoveSignatures,
	}
}

func configurationCloneFixture() *Configuration {
	return &Configuration{
		resources: resourcesForConfigurationDir(configurationResourceModeReadOnly, "test/config"),

		Path:                            "test/path",
		CreationDate:                    "2026-08-31 12:34",
		Version:                         "v0.16.0-test",
		SchemaVersion:                   ConfigurationSchemaVersionCurrent,
		CheckFileNameExt:                true,
		Reader15:                        true,
		DecodeAllStreams:                true,
		ValidationMode:                  ValidationRelaxed,
		UnsupportedResourcePolicy:       UnsupportedResourceFail,
		PostProcessValidate:             true,
		PreserveInfoDict:                true,
		ValidateLinks:                   true,
		Eol:                             types.EolCRLF,
		WriteObjectStream:               true,
		WriteXRefStream:                 true,
		StatsFileName:                   "stats.csv",
		UserPW:                          "user",
		OwnerPW:                         "owner",
		PrivateKeyPW:                    "private-key",
		EncryptUsingAES:                 true,
		EncryptKeyLength:                128,
		Permissions:                     PermissionsPrint,
		Cmd:                             VALIDATESIGNATURES,
		Unit:                            types.MILLIMETRES,
		TimestampFormat:                 "2006-01-02T15:04:05Z07:00",
		DateFormat:                      "2006/01/02",
		Optimize:                        true,
		OptimizeBeforeWriting:           true,
		OptimizeResourceDicts:           true,
		OptimizeDuplicateContentStreams: true,
		CreateBookmarks:                 true,
		MergeBookmarkMode:               MergeBookmarkModePreserve,
		NeedAppearances:                 true,
		Offline:                         true,
		Timeout:                         11,
		TimeoutCRL:                      12,
		TimeoutOCSP:                     13,
		PreferredCertRevocationChecker:  OCSP,
		FormFieldListMaxColWidth:        14,
		Limits: ResourceLimits{
			MaxStreamBytes:       15,
			MaxDecodeBytes:       16,
			MaxImagePixels:       17,
			MaxImageBytes:        18,
			MaxObjectCount:       19,
			MaxObjectStreamCount: 20,
			MaxObjectStreamFirst: 21,
			MaxXRefEntries:       22,
			MaxRecursionDepth:    23,
		},
		RemoveEncryption: true,
		RemoveSignatures: true,
	}
}

func configurationCloneReferenceFixture() *Configuration {
	userPWNew := "new-user"
	ownerPWNew := "new-owner"
	conf := configurationCloneFixture()
	conf.UserPWNew = &userPWNew
	conf.OwnerPWNew = &ownerPWNew
	conf.AllowedRevocationHosts = []string{"ocsp.example.corp", "crl.example.corp"}
	return conf
}

func requireConfigurationReferenceClone(t *testing.T, conf *Configuration) *Configuration {
	t.Helper()

	clone := conf.Clone()
	if clone == nil {
		t.Fatal("clone: got nil")
	}
	if clone.UserPWNew == nil || *clone.UserPWNew != *conf.UserPWNew {
		t.Fatalf("new user password: got %v, want %q", clone.UserPWNew, *conf.UserPWNew)
	}
	if clone.OwnerPWNew == nil || *clone.OwnerPWNew != *conf.OwnerPWNew {
		t.Fatalf("new owner password: got %v, want %q", clone.OwnerPWNew, *conf.OwnerPWNew)
	}
	if len(clone.AllowedRevocationHosts) != len(conf.AllowedRevocationHosts) {
		t.Fatalf("allowed revocation hosts: got %v, want %v", clone.AllowedRevocationHosts, conf.AllowedRevocationHosts)
	}
	for i, host := range conf.AllowedRevocationHosts {
		if clone.AllowedRevocationHosts[i] != host {
			t.Fatalf("allowed revocation host %d: got %q, want %q", i, clone.AllowedRevocationHosts[i], host)
		}
	}
	return clone
}

// TestConfigurationCloneNilReceiver verifies that a nil configuration remains nil when cloned.
func TestConfigurationCloneNilReceiver(t *testing.T) {
	var conf *Configuration
	if clone := conf.Clone(); clone != nil {
		t.Fatalf("clone: got %v, want nil", clone)
	}
}

// TestConfigurationCloneCopiesScalarFields verifies that cloning preserves every scalar configuration field.
func TestConfigurationCloneCopiesScalarFields(t *testing.T) {
	conf := configurationCloneFixture()
	clone := conf.Clone()
	if clone == nil {
		t.Fatal("clone: got nil")
	}
	if clone == conf {
		t.Fatal("clone returned source configuration")
	}
	if got, want := scalarValues(clone), scalarValues(conf); got != want {
		t.Fatalf("scalar values:\ngot:  %+v\nwant: %+v", got, want)
	}
}

// TestConfigurationClonePreservesNilReferenceFields verifies that nil reference-bearing fields remain nil.
func TestConfigurationClonePreservesNilReferenceFields(t *testing.T) {
	clone := configurationCloneFixture().Clone()
	if clone == nil {
		t.Fatal("clone: got nil")
	}
	if clone.UserPWNew != nil {
		t.Fatalf("new user password: got %q, want nil", *clone.UserPWNew)
	}
	if clone.OwnerPWNew != nil {
		t.Fatalf("new owner password: got %q, want nil", *clone.OwnerPWNew)
	}
	if clone.AllowedRevocationHosts != nil {
		t.Fatalf("allowed revocation hosts: got %v, want nil", clone.AllowedRevocationHosts)
	}
}

// TestConfigurationClonePreservesEmptyHostSlice verifies that an empty non-nil host slice remains non-nil.
func TestConfigurationClonePreservesEmptyHostSlice(t *testing.T) {
	conf := configurationCloneFixture()
	conf.AllowedRevocationHosts = []string{}

	clone := conf.Clone()
	if clone == nil {
		t.Fatal("clone: got nil")
	}
	if clone.AllowedRevocationHosts == nil {
		t.Fatal("allowed revocation hosts: got nil, want empty non-nil slice")
	}
	if len(clone.AllowedRevocationHosts) != 0 {
		t.Fatalf("allowed revocation hosts: got %v, want empty slice", clone.AllowedRevocationHosts)
	}
}

// TestConfigurationCloneDoesNotAliasSource verifies that mutating cloned reference fields does not change the source.
func TestConfigurationCloneDoesNotAliasSource(t *testing.T) {
	conf := configurationCloneReferenceFixture()
	clone := requireConfigurationReferenceClone(t, conf)

	if clone.UserPWNew == conf.UserPWNew {
		t.Fatal("new user password pointer aliases source")
	}
	if clone.OwnerPWNew == conf.OwnerPWNew {
		t.Fatal("new owner password pointer aliases source")
	}

	*clone.UserPWNew = "changed-clone-user"
	*clone.OwnerPWNew = "changed-clone-owner"
	clone.AllowedRevocationHosts[0] = "changed-clone.example.corp"

	if got, want := *conf.UserPWNew, "new-user"; got != want {
		t.Fatalf("source new user password: got %q, want %q", got, want)
	}
	if got, want := *conf.OwnerPWNew, "new-owner"; got != want {
		t.Fatalf("source new owner password: got %q, want %q", got, want)
	}
	if got, want := conf.AllowedRevocationHosts[0], "ocsp.example.corp"; got != want {
		t.Fatalf("source allowed revocation host: got %q, want %q", got, want)
	}
}

// TestConfigurationCloneDoesNotObserveSourceMutation verifies source mutations do not change cloned reference fields.
func TestConfigurationCloneDoesNotObserveSourceMutation(t *testing.T) {
	conf := configurationCloneReferenceFixture()
	clone := requireConfigurationReferenceClone(t, conf)

	*conf.UserPWNew = "changed-source-user"
	*conf.OwnerPWNew = "changed-source-owner"
	conf.AllowedRevocationHosts[1] = "changed-source.example.corp"

	if got, want := *clone.UserPWNew, "new-user"; got != want {
		t.Fatalf("cloned new user password: got %q, want %q", got, want)
	}
	if got, want := *clone.OwnerPWNew, "new-owner"; got != want {
		t.Fatalf("cloned new owner password: got %q, want %q", got, want)
	}
	if got, want := clone.AllowedRevocationHosts[1], "crl.example.corp"; got != want {
		t.Fatalf("cloned allowed revocation host: got %q, want %q", got, want)
	}
}

// TestConfigurationCloneReferenceFieldGuard requires clone coverage to be reviewed when mutable fields are added.
func TestConfigurationCloneReferenceFieldGuard(t *testing.T) {
	typ := reflect.TypeFor[Configuration]()
	var got []string
	for i := range typ.NumField() {
		field := typ.Field(i)
		if containsMutableReferences(field.Type) {
			got = append(got, field.Name)
		}
	}

	want := []string{"UserPWNew", "OwnerPWNew", "AllowedRevocationHosts"}
	if !slices.Equal(got, want) {
		t.Fatalf("mutable reference fields: got %v, want %v; update Clone and its tests", got, want)
	}
}
