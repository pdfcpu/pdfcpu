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

package validate

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"testing"

	pdfcpuLog "github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func type1FontTestObjects(t *testing.T, version model.Version, fontName string) (*model.Context, types.Dict) {
	t.Helper()
	conf := model.NewStatelessConfiguration()
	ctx, err := model.NewContext(bytes.NewReader(nil), conf)
	if err != nil {
		t.Fatal(err)
	}
	fd := types.Dict{
		"Type":        types.Name("FontDescriptor"),
		"FontName":    types.Name(fontName),
		"Flags":       types.Integer(33),
		"FontBBox":    types.NewIntegerArray(-23, -250, 715, 805),
		"ItalicAngle": types.Integer(0),
		"Ascent":      types.Integer(629),
		"Descent":     types.Integer(-157),
		"CapHeight":   types.Integer(562),
		"StemV":       types.Integer(51),
	}
	ctx.XRefTable.Table = map[int]*model.XRefTableEntry{1: model.NewXRefTableEntryGen0(fd)}
	ctx.XRefTable.HeaderVersion = &version
	ctx.XRefTable.ValidationMode = model.ValidationStrict
	d := types.Dict{
		"Type":           types.Name("Font"),
		"Subtype":        types.Name("Type1"),
		"BaseFont":       types.Name(fontName),
		"FirstChar":      types.Integer(65),
		"LastChar":       types.Integer(65),
		"Widths":         types.NewIntegerArray(600),
		"FontDescriptor": *types.NewIndirectRef(1, 0),
	}
	return ctx, d
}

func checkType1FontValidation(t *testing.T, ctx *model.Context, d types.Dict, wantMissing string) {
	t.Helper()
	fontName, err := validateType1FontDict(ctx.XRefTable, d, 2)
	if wantMissing != "" {
		if err == nil || !strings.Contains(err.Error(), "required entry="+wantMissing+" missing") {
			t.Fatalf("got %v, want missing %s", err, wantMissing)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if want := d["BaseFont"].(types.Name).Value(); fontName != want {
		t.Fatalf("font name: got %q, want %q", fontName, want)
	}
}

// TestType1FontMetricsVersions verifies every metrics combination across the PDF 2.0 boundary.
func TestType1FontMetricsVersions(t *testing.T) {
	keys := []string{"FirstChar", "LastChar", "Widths", "FontDescriptor"}
	for _, version := range []model.Version{model.V17, model.V20} {
		for _, fontName := range []string{"Courier", "CustomFont"} {
			for mask := 0; mask < 16; mask++ {
				t.Run(fmt.Sprintf("%s/%s/metrics_%04b", version, fontName, mask), func(t *testing.T) {
					ctx, d := type1FontTestObjects(t, version, fontName)
					missing := ""
					for i, key := range keys {
						if mask&(1<<i) == 0 {
							delete(d, key)
							if missing == "" {
								missing = key
							}
						}
					}
					if version == model.V17 && fontName == "Courier" && mask == 0 {
						missing = ""
					}
					checkType1FontValidation(t, ctx, d, missing)
					if !ctx.ValidationReport().Empty() {
						t.Fatalf("strict validation reported accepted divergences: %+v", ctx.ValidationReport().Notices())
					}
				})
			}
		}
	}
}

// TestType1FontMetricsRelaxed preserves acceptance of omitted metrics in relaxed mode.
func TestType1FontMetricsRelaxed(t *testing.T) {
	for _, version := range []model.Version{model.V17, model.V20} {
		t.Run(version.String(), func(t *testing.T) {
			ctx, d := type1FontTestObjects(t, version, "Courier")
			ctx.XRefTable.ValidationMode = model.ValidationRelaxed
			for _, key := range []string{"FirstChar", "LastChar", "Widths", "FontDescriptor"} {
				delete(d, key)
			}
			checkType1FontValidation(t, ctx, d, "")
		})
	}
}

// TestType1FontMetricsNull verifies that direct and indirect null entries count as absent.
func TestType1FontMetricsNull(t *testing.T) {
	for _, indirect := range []bool{false, true} {
		t.Run(fmt.Sprintf("indirect_%t", indirect), func(t *testing.T) {
			ctx, d := type1FontTestObjects(t, model.V17, "Courier")
			ctx.XRefTable.Table[2] = model.NewXRefTableEntryGen0(nil)
			var null types.Object
			if indirect {
				null = *types.NewIndirectRef(2, 0)
			}
			for _, key := range []string{"FirstChar", "LastChar", "Widths", "FontDescriptor"} {
				d[key] = null
			}
			checkType1FontValidation(t, ctx, d, "")
			d["LastChar"] = types.Integer(65)
			if _, err := validateType1FontDict(ctx.XRefTable, d, 2); err == nil || !strings.Contains(err.Error(), "required entry=FirstChar") {
				t.Fatalf("got %v, want required FirstChar error", err)
			}
		})
	}
}

// TestType1FontMetricsRelaxedDiagnostics reports only omissions that violate the strict rules.
func TestType1FontMetricsRelaxedDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name     string
		version  model.Version
		fontName string
		remove   []string
		want     string
	}{
		{"standard17Absent", model.V17, "Courier", []string{"FirstChar", "LastChar", "Widths", "FontDescriptor"}, ""},
		{"standard17Partial", model.V17, "Courier", []string{"FirstChar"}, "FirstChar"},
		{"standard20Absent", model.V20, "Courier", []string{"FirstChar", "LastChar", "Widths", "FontDescriptor"}, "FirstChar, LastChar, Widths, FontDescriptor"},
		{"nonstandard17Partial", model.V17, "CustomFont", []string{"Widths"}, "Widths"},
		{"standard20Complete", model.V20, "Courier", nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			pdfcpuLog.SetCLILogger(log.New(&buf, "", 0))
			defer pdfcpuLog.SetCLILogger(nil)
			ctx, d := type1FontTestObjects(t, tc.version, tc.fontName)
			ctx.XRefTable.ValidationMode = model.ValidationRelaxed
			for _, key := range tc.remove {
				delete(d, key)
			}
			checkType1FontValidation(t, ctx, d, "")
			if got := buf.String(); got != "" {
				t.Fatalf("global diagnostic: got %q, want none", got)
			}
			notices := ctx.ValidationReport().Notices()
			if tc.want == "" {
				if len(notices) != 0 {
					t.Fatalf("notices: got %+v, want none", notices)
				}
				return
			}
			if len(notices) != 1 {
				t.Fatalf("notice count: got %d, want 1", len(notices))
			}
			want := fmt.Sprintf("Type1 font %s: missing required entries %s", tc.fontName, tc.want)
			notice := notices[0]
			wantCause := "dict=type1FontDict required entry=" + strings.Split(tc.want, ",")[0] + " missing"
			if notice.Message != want || notice.Cause == nil || notice.Cause.Error() != wantCause {
				t.Fatalf("notice: got %+v, want %q", notice, want)
			}
			if notice.ObjectNumber != 2 || notice.Phase != model.NoticePhaseValidate || notice.Disposition != model.NoticeDigested {
				t.Fatalf("notice classification: got %+v", notice)
			}
		})
	}
}
