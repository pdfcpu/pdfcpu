/*
Copyright 2021 The pdfcpu Authors.

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

package test

import (
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

func TestCreateForm(t *testing.T) {
	msg := "TestCreateForm"

	outDir := filepath.Join("..", "..", "samples", "forms")
	outFile := filepath.Join(outDir, "test.pdf")

	if err := api.CreateForm("in.json", outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	if err := api.ValidateFile(outFile, nil); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	// if err := api.ExtractForm(outFile, "out.json", nil); err != nil {
	// 	t.Fatalf("%s: %v\n", msg, err)
	// }

	// in.json should == out.sign
}
