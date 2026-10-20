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

package cli

// GHSA-44h9-p34h-2w4p: preserve local-image CLI workflows without changing the caller's API configuration.
// https://github.com/pdfcpu/pdfcpu/security/advisories/GHSA-44h9-p34h-2w4p

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func ghsaCLIWrite(t *testing.T, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(name, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func ghsaCLIImage(t *testing.T, dir string) {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 2, 3))); err != nil {
		t.Fatal(err)
	}
	ghsaCLIWrite(t, filepath.Join(dir, "image.bin"), buf.Bytes())
}

func ghsaCLIInputs(t *testing.T) {
	t.Helper()
	conf := model.NewStatelessConfiguration()
	template := `{"pages":{"1":{"content":{"textfield":[{"id":"name","pos":[10,10],"width":100,"font":{"name":"Helvetica","size":12}}]}}}}`
	var out bytes.Buffer
	if err := api.Create(t.Context(), nil, strings.NewReader(template), &out, conf); err != nil {
		t.Fatal(err)
	}
	ghsaCLIWrite(t, "template.pdf", out.Bytes())
	ghsaCLIWrite(t, "create.json", []byte(`{"pages":{"1":{"content":{"image":[{"src":"image.bin","width":20,"height":30}]}}}}`))
	ghsaCLIWrite(t, "fill.json", []byte(`{"forms":[{"textfield":[{"name":"name","value":"filled"}],"pages":{"1":{"image":[{"src":"image.bin","width":20,"height":30}]}}}]}`))
	ghsaCLIWrite(t, "fill.csv", []byte("name,\"@img(page:1, w:20, h:30)\"\nfilled,image.bin\n"))
}

func ghsaCLIAssertImage(t *testing.T, name string) {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	pages, err := api.ExtractImagesRaw(t.Context(), bytes.NewReader(data), nil, model.NewStatelessConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, page := range pages {
		count += len(page)
	}
	if count != 1 {
		t.Fatalf("image count=%d, want 1", count)
	}
}

func TestGHSA_44h9_p34h_2w4p_CLIDefaultAndExplicitRoot(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		t.Run(map[bool]string{false: "workingDirectory", true: "explicitRoot"}[explicit], func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			root := ""
			assetDir := dir
			if explicit {
				assetDir = t.TempDir()
				root = assetDir
			}
			ghsaCLIImage(t, assetDir)
			ghsaCLIInputs(t)
			conf := model.NewStatelessConfiguration()
			conf.Offline = true
			conf.LocalImageRoot = root
			commands := []struct {
				cmd    *Command
				output string
			}{
				{CreateCommand("", "create.json", "created.pdf", conf), "created.pdf"},
				{FillFormCommand("template.pdf", "fill.json", "filled.pdf", conf), "filled.pdf"},
				{MultiFillFormCommand("template.pdf", "fill.json", ".", "json.pdf", false, conf), "json_01.pdf"},
				{MultiFillFormCommand("template.pdf", "fill.csv", ".", "csv.pdf", false, conf), "csv_01.pdf"},
			}
			for _, entry := range commands {
				if _, err := Dispatch(t.Context(), entry.cmd); err != nil {
					t.Fatal(err)
				}
				ghsaCLIAssertImage(t, entry.output)
			}
			if conf.LocalImageRoot != root {
				t.Fatal("CLI mutated caller configuration")
			}
		})
	}
}
