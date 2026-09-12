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
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/cli"
)

type annotationListJSON struct {
	Header struct {
		Version string `json:"version"`
	} `json:"header"`
	Annotations map[string][]struct {
		Type    string     `json:"type"`
		ObjNr   int        `json:"objNr"`
		Rect    [4]float64 `json:"rect"`
		Content *string    `json:"content"`
	} `json:"annotations"`
}

// TestListAndRemoveAnnotations verifies list and remove annotations.
func TestListAndRemoveAnnotations(t *testing.T) {
	msg := "TestListAndRemoveAnnotations"

	fn := "adobe_errata.pdf"
	copyFile(t, filepath.Join(inDir, fn), filepath.Join(outDir, fn))
	inFile := filepath.Join(outDir, fn)

	// See also api/annotations_test.go for page annotation manipulation
	// including adding annotations.

	cmd := cli.ListAnnotationsCommand(inFile, nil, conf)
	if _, err := cli.Dispatch(t.Context(), cmd); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	cmd = cli.ListAnnotationsJSONCommand(inFile, nil, conf)
	out, err := cli.Dispatch(t.Context(), cmd)
	if err != nil {
		t.Fatalf("%s json: %v\n", msg, err)
	}
	if len(out) != 1 {
		t.Fatalf("%s json: want 1 output string, got %d\n", msg, len(out))
	}
	var annots annotationListJSON
	if err := json.Unmarshal([]byte(out[0]), &annots); err != nil {
		t.Fatalf("%s json: %v\n", msg, err)
	}
	if annots.Header.Version == "" {
		t.Fatalf("%s json: missing header version\n", msg)
	}
	if len(annots.Annotations) == 0 {
		t.Fatalf("%s json: missing annotations\n", msg)
	}

	// Remove page annotation using obj# 34
	cmd = cli.RemoveAnnotationsCommand(inFile, "", nil, nil, []int{34}, conf)
	if _, err := cli.Dispatch(t.Context(), cmd); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	// Remove all page annotations from page 14
	cmd = cli.RemoveAnnotationsCommand(inFile, "", []string{"14"}, nil, nil, conf)
	if _, err := cli.Dispatch(t.Context(), cmd); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

	// Remove all page annotations
	cmd = cli.RemoveAnnotationsCommand(inFile, "", nil, nil, nil, conf)
	if _, err := cli.Dispatch(t.Context(), cmd); err != nil {
		t.Fatalf("%s: %v\n", msg, err)
	}

}
