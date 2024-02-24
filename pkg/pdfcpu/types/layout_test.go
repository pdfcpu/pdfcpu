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

func TestParsePageFormat(t *testing.T) {
	dim, _, err := ParsePageFormat("A3L")
	if err != nil {
		t.Error(err)
	}
	if (dim.Width != 1191) || (dim.Height != 842) {
		t.Errorf("expected 1191x842. got %s", dim)
	}
	// the original dim should be unmodified
	dimOrig := PaperSize["A3"]
	if (dimOrig.Width != 842) || (dimOrig.Height != 1191) {
		t.Errorf("expected origDim=842x1191x842. got %s", dimOrig)
	}
}
