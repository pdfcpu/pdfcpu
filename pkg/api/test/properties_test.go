/*
Copyright 2020 The pdf Authors.

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

func listProperties(t *testing.T, msg, fileName string, want []string) []string {
	t.Helper()

	got, err := api.ListPropertiesFile(fileName, nil)
	if err != nil {
		t.Fatalf("%s list properties: %v\n", msg, err)
	}

	// # of keywords must be want
	if len(got) != len(want) {
		t.Fatalf("%s: list properties %s: want %d got %d\n", msg, fileName, len(want), len(got))
	}
	for i, v := range got {
		if v != want[i] {
			t.Fatalf("%s: list properties %s: want %v got %v\n", msg, fileName, want, got)
		}
	}
	return got
}

func TestProperties(t *testing.T) {
	msg := "TestProperties"

	fileName := filepath.Join(outDir, "go.pdf")
	if err := copyFile(t, filepath.Join(inDir, "go.pdf"), fileName); err != nil {
		t.Fatalf("%s: copyFile: %v\n", msg, err)
	}

	// # of properties must be 0
	listProperties(t, msg, fileName, nil)

	properties := map[string]string{"name1": "value1", "nameÖ": "valueö"}
	if err := api.AddPropertiesFile(fileName, "", properties, nil); err != nil {
		t.Fatalf("%s add properties: %v\n", msg, err)
	}

	listProperties(t, msg, fileName, []string{"name1 = value1", "nameÖ = valueö"})

	if err := api.RemovePropertiesFile(fileName, "", []string{"nameÖ"}, nil); err != nil {
		t.Fatalf("%s remove 1 property: %v\n", msg, err)
	}

	listProperties(t, msg, fileName, []string{"name1 = value1"})

	if err := api.RemovePropertiesFile(fileName, "", nil, nil); err != nil {
		t.Fatalf("%s remove all properties: %v\n", msg, err)
	}

	// # of properties must be 0
	listProperties(t, msg, fileName, nil)
}
