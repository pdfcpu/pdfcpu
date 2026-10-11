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
	"errors"
	"testing"
)

// TestValidateVersionReturnsVersionRequirementError verifies version failures expose structured details.
func TestValidateVersionReturnsVersionRequirementError(t *testing.T) {
	actualVersion := V13
	xRefTable := &XRefTable{HeaderVersion: &actualVersion}

	err := xRefTable.ValidateVersion("dict=Example entry=Feature", V14)
	if !errors.Is(err, ErrVersionTooLow) {
		t.Fatalf("got %v, want %v", err, ErrVersionTooLow)
	}

	var versionErr *VersionRequirementError
	if !errors.As(err, &versionErr) {
		t.Fatalf("got %T, want *VersionRequirementError", err)
	}
	if versionErr.Element != "dict=Example entry=Feature" {
		t.Fatalf("got element %q", versionErr.Element)
	}
	if versionErr.ActualVersion != V13 {
		t.Fatalf("got actual version %s, want %s", versionErr.ActualVersion, V13)
	}
	if versionErr.RequiredVersion != V14 {
		t.Fatalf("got required version %s, want %s", versionErr.RequiredVersion, V14)
	}
}

// TestValidateVersionAcceptsSupportedVersions verifies equal and newer versions satisfy the requirement.
func TestValidateVersionAcceptsSupportedVersions(t *testing.T) {
	for _, version := range []Version{V14, V15} {
		version := version
		xRefTable := &XRefTable{HeaderVersion: &version}
		if err := xRefTable.ValidateVersion("Example", V14); err != nil {
			t.Fatalf("version %s: %v", version, err)
		}
	}
}

// TestValidateVersionUsesRootVersion verifies only a later catalog version overrides the header version.
func TestValidateVersionUsesRootVersion(t *testing.T) {
	for _, tc := range []struct{ header, root, want Version }{
		{V17, V20, V20},
		{V20, V17, V20},
		{V17, V17, V17},
	} {
		x := &XRefTable{HeaderVersion: &tc.header, RootVersion: &tc.root}
		if got := x.Version(); got != tc.want {
			t.Fatalf("header=%s catalog=%s: got %s, want %s", tc.header, tc.root, got, tc.want)
		}
		if err := x.ValidateVersion("Example", tc.want); err != nil {
			t.Fatal(err)
		}
	}
}
