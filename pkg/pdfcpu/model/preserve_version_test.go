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

import "testing"

func TestPreserveVersionDefaultsToDisabled(t *testing.T) {
	if NewStatelessConfiguration().PreserveVersion {
		t.Fatal("PreserveVersion defaults to true")
	}
}

func TestEnsureVersionForWritingRespectsPreserveVersion(t *testing.T) {
	header := V14
	conf := NewStatelessConfiguration()
	xRefTable := &XRefTable{HeaderVersion: &header, Conf: conf}

	xRefTable.EnsureVersionForWriting()
	if got := xRefTable.Version(); got != V17 {
		t.Fatalf("version: got %s, want 1.7", got)
	}

	header = V14
	conf.PreserveVersion = true
	xRefTable = &XRefTable{HeaderVersion: &header, Conf: conf}
	xRefTable.EnsureVersionForWriting()
	if got := xRefTable.Version(); got != V14 {
		t.Fatalf("preserved version: got %s, want 1.4", got)
	}

	xRefTable.EnsureMinimumVersion(V15)
	if got := xRefTable.Version(); got != V15 {
		t.Fatalf("raised version: got %s, want 1.5", got)
	}
	xRefTable.EnsureMinimumVersion(V14)
	if got := xRefTable.Version(); got != V15 {
		t.Fatalf("version decreased: got %s, want 1.5", got)
	}
}

func TestAnnotationRequiredVersionUsesStrictSpecification(t *testing.T) {
	tests := []struct {
		annotation AnnotationType
		version    Version
	}{
		{AnnText, V10},
		{AnnPolygon, V15},
		{AnnRedact, V17},
		{AnnCustom, V10},
	}
	for _, tt := range tests {
		if got := tt.annotation.RequiredVersion(); got != tt.version {
			t.Fatalf("%s: got %s, want %s", AnnotTypeStrings[tt.annotation], got, tt.version)
		}
	}
}
