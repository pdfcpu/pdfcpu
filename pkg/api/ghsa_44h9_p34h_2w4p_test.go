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

package api

// GHSA-44h9-p34h-2w4p: untrusted create/form input must not read images outside the configured root.
// https://github.com/pdfcpu/pdfcpu/security/advisories/GHSA-44h9-p34h-2w4p

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/form"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

const ghsaImageTemplate = `{"pages":{"1":{"content":{"textfield":[{"id":"name","pos":[10,10],"width":100,"font":{"name":"Helvetica","size":12}}]}}}}`

func ghsaImageAssets(t *testing.T) (string, []byte) {
	t.Helper()
	root := t.TempDir()
	var img bytes.Buffer
	if err := png.Encode(&img, image.NewRGBA(image.Rect(0, 0, 2, 3))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "image.bin"), img.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	var template bytes.Buffer
	if err := Create(t.Context(), nil, strings.NewReader(ghsaImageTemplate), &template, model.NewStatelessConfiguration()); err != nil {
		t.Fatal(err)
	}
	return root, template.Bytes()
}

func ghsaCreateImageJSON(src string) string {
	name, _ := json.Marshal(src)
	return `{"pages":{"1":{"content":{"image":[{"src":` + string(name) + `,"width":20,"height":30}]}}}}`
}

func ghsaFillImageJSON(src string) string {
	name, _ := json.Marshal(src)
	return `{"forms":[{"textfield":[{"name":"name","value":"filled"}],"pages":{"1":{"image":[{"src":` + string(name) + `,"width":20,"height":30}]}}}]}`
}

func ghsaCheckImageOutput(t *testing.T, data []byte) {
	t.Helper()
	pages, err := ExtractImagesRaw(t.Context(), bytes.NewReader(data), nil, model.NewStatelessConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, page := range pages {
		count += len(page)
	}
	if count != 1 {
		t.Fatalf("embedded image count = %d, want 1", count)
	}
}

func TestGHSA44h9P34h2w4pCreateAndFill(t *testing.T) {
	root, template := ghsaImageAssets(t)
	for _, fill := range []bool{false, true} {
		for _, tc := range []struct {
			name, root, src string
			deny            bool
		}{
			{"disabled", "", "image.bin", true},
			{"relative", root, "image.bin", false},
			{"absolute", root, filepath.Join(root, "image.bin"), true},
			{"parent", root, "sub/../image.bin", true},
			{"windowsParent", root, `sub\..\image.bin`, true},
			{"windowsDrive", root, `C:\image.bin`, true},
			{"directory", root, ".", true},
		} {
			t.Run(fmt.Sprintf("fill=%t/%s", fill, tc.name), func(t *testing.T) {
				conf := model.NewStatelessConfiguration()
				conf.Offline = true
				conf.LocalImageRoot = tc.root
				var out bytes.Buffer
				var err error
				if fill {
					err = FillForm(t.Context(), bytes.NewReader(template), strings.NewReader(ghsaFillImageJSON(tc.src)), &out, conf)
				} else {
					err = Create(t.Context(), nil, strings.NewReader(ghsaCreateImageJSON(tc.src)), &out, conf)
				}
				if tc.deny {
					if !errors.Is(err, os.ErrPermission) {
						t.Fatalf("want access denial, got %v", err)
					}
					if out.Len() != 0 {
						t.Fatal("denial wrote output")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				ghsaCheckImageOutput(t, out.Bytes())
			})
		}
	}
}

func TestGHSA44h9P34h2w4pMultiFill(t *testing.T) {
	root, template := ghsaImageAssets(t)
	input := filepath.Join(t.TempDir(), "template.pdf")
	if err := os.WriteFile(input, template, 0600); err != nil {
		t.Fatal(err)
	}
	for _, format := range []form.DataFormat{form.JSON, form.CSV} {
		for _, allow := range []bool{false, true} {
			t.Run(fmt.Sprintf("format=%d/allow=%t", format, allow), func(t *testing.T) {
				conf := model.NewStatelessConfiguration()
				conf.Offline = true
				if allow {
					conf.LocalImageRoot = root
				}
				data := ghsaFillImageJSON("image.bin")
				if format == form.CSV {
					data = "name,\"@img(page:1, w:20, h:30)\"\nfilled,image.bin\n"
				}
				out := t.TempDir()
				err := MultiFillForm(t.Context(), input, strings.NewReader(data), out, "result", format, false, conf)
				if !allow {
					if !errors.Is(err, os.ErrPermission) {
						t.Fatalf("want denial, got %v", err)
					}
					entries, readErr := os.ReadDir(out)
					if readErr != nil || len(entries) != 0 {
						t.Fatalf("denied operation left output: %v %v", entries, readErr)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				result, err := os.ReadFile(filepath.Join(out, "result_01.pdf"))
				if err != nil {
					t.Fatal(err)
				}
				ghsaCheckImageOutput(t, result)
			})
		}
	}
}

func TestGHSA44h9P34h2w4pAliasesAndPolicyInjection(t *testing.T) {
	root, _ := ghsaImageAssets(t)
	for _, tc := range []struct {
		name, dirs, files, image string
		deny                     bool
	}{
		{"global", `{"assets":"."}`, `{"logo":"$assets/image.bin"}`, `{"src":"$logo"}`, false},
		{"emptyDirectory", `{"assets":""}`, `{"logo":"$assets/image.bin"}`, `{"src":"$logo"}`, false},
		{"aliasParent", `{"assets":"."}`, `{"logo":"$assets/sub/../image.bin"}`, `{"src":"$logo"}`, true},
		{"directoryParent", `{"assets":"sub/.."}`, `{"logo":"$assets/image.bin"}`, `{"src":"$logo"}`, true},
	} {
		for _, page := range []bool{false, true} {
			t.Run(fmt.Sprintf("page=%t/%s", page, tc.name), func(t *testing.T) {
				conf := model.NewStatelessConfiguration()
				conf.LocalImageRoot = root
				data := fmt.Sprintf(`{"dirs":%s,"files":%s,"images":{"logo":%s},"pages":{"1":{"content":{"image":[{"name":"$logo","width":20,"height":30}]}}}}`, tc.dirs, tc.files, tc.image)
				if page {
					data = fmt.Sprintf(`{"dirs":%s,"pages":{"1":{"files":%s,"content":{"image":[{"src":"$logo","width":20,"height":30}]}}}}`, tc.dirs, tc.files)
				}
				var out bytes.Buffer
				err := Create(t.Context(), nil, strings.NewReader(data), &out, conf)
				if tc.deny {
					if !errors.Is(err, os.ErrPermission) {
						t.Fatalf("want denial, got %v", err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				ghsaCheckImageOutput(t, out.Bytes())
			})
		}
	}
	data := strings.Replace(ghsaCreateImageJSON("image.bin"), "{", `{"conf":{"LocalImageRoot":`+fmt.Sprintf("%q", root)+`},"LocalImageRoot":`+fmt.Sprintf("%q", root)+",", 1)
	var out bytes.Buffer
	err := Create(t.Context(), nil, strings.NewReader(data), &out, model.NewStatelessConfiguration())
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("JSON relaxed caller policy: %v", err)
	}
}
