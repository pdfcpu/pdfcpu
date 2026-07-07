/*
Copyright 2024 The pdfcpu Authors.

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

package types

import "testing"

// TestEncodeDict verifies encode dict.
func TestEncodeDict(t *testing.T) {
	dict := Dict{
		"A()": Integer(1),
	}
	expected := `<</A#28#29 1>>`
	s := dict.PDFString()
	if s != expected {
		t.Errorf("expected %s for %v, got %s", expected, dict, s)
	}
}

func TestDictHasEntry(t *testing.T) {
	dict := Dict{
		"A":       Integer(1),
		"B":       nil,
		"C#28#29": Name("encoded"),
	}

	if !dict.HasEntry("A") {
		t.Fatal("expected entry A")
	}
	if dict.HasEntry("B") {
		t.Fatal("expected nil entry B to be absent")
	}
	if dict.HasEntry("D") {
		t.Fatal("expected missing entry D to be absent")
	}
	if !dict.HasEntry("C()") {
		t.Fatal("expected decoded entry C()")
	}
}
