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
	"log"
	"strings"
	"testing"

	pdfcpuLog "github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// TestGoToRDestinations checks remote page numbers, named destinations, and invalid destinations.
func TestGoToRDestinations(t *testing.T) {
	for _, tc := range []struct {
		name string
		dest types.Object
		want string
	}{
		{"firstPage", types.Array{types.Integer(0), types.Name("Fit")}, ""},
		{"laterPage", types.Array{types.Integer(42), types.Name("Fit")}, ""},
		{"indirectPageNumber", types.Array{*types.NewIndirectRef(2, 0), types.Name("Fit")}, ""},
		{"indirectArray", *types.NewIndirectRef(3, 0), ""},
		{"named", types.Name("chapter"), ""},
		{"string", types.StringLiteral("chapter"), ""},
		{"negativePage", types.Array{types.Integer(-1), types.Name("Fit")}, "non-negative page number"},
		{"realPage", types.Array{types.Float(0), types.Name("Fit")}, "non-negative page number"},
		{"localPage", types.Array{*types.NewIndirectRef(1, 0), types.Name("Fit")}, "non-negative page number"},
		{"nullPage", types.Array{nil, types.Name("Fit")}, "non-negative page number"},
		{"empty", types.Array{}, "invalid length"},
		{"missingMode", types.Array{types.Integer(0)}, "invalid length"},
		{"invalidMode", types.Array{types.Integer(0), types.Name("Invalid")}, "invalid mode"},
		{"modeType", types.Array{types.Integer(0), types.Integer(0)}, "expected name"},
		{"dict", types.Dict{"D": types.Array{types.Integer(0), types.Name("Fit")}}, "cannot be dict"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x := sharedActionXRefTable(100, map[int]types.Dict{1: {"Type": types.Name("Page")}})
			x.Table[2] = model.NewXRefTableEntryGen0(types.Integer(0))
			x.Table[3] = model.NewXRefTableEntryGen0(types.Array{types.Integer(0), types.Name("Fit")})
			d := types.Dict{"S": types.Name("GoToR"), "F": types.StringLiteral("go.pdf"), "D": tc.dest}
			err := validateGoToRActionDict(x, d, "GoToR")
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
		})
	}
}

// TestGoToStillRejectsPageNumbers preserves the local destination rule.
func TestGoToStillRejectsPageNumbers(t *testing.T) {
	x := sharedActionXRefTable(100, map[int]types.Dict{1: {"Type": types.Name("Page")}})
	d := types.Dict{"D": types.Array{types.Integer(0), types.Name("Fit")}}
	if err := validateGoToActionDict(x, d, "GoTo"); err == nil {
		t.Fatal("local destination accepted a page number")
	}
	d["D"] = types.Array{*types.NewIndirectRef(1, 0), types.Name("Fit")}
	if err := validateGoToActionDict(x, d, "GoTo"); err != nil {
		t.Fatal(err)
	}
}

// TestGoToRRelaxedDiagnostics reports invalid remote pages without warning about valid page numbers.
func TestGoToRRelaxedDiagnostics(t *testing.T) {
	var buf bytes.Buffer
	pdfcpuLog.SetCLILogger(log.New(&buf, "", 0))
	defer pdfcpuLog.SetCLILogger(nil)
	x := sharedActionXRefTable(100, nil)
	x.ValidationMode = model.ValidationRelaxed
	d := types.Dict{"F": types.StringLiteral("go.pdf"), "D": types.Array{types.Integer(0), types.Name("Fit")}}
	if err := validateGoToRActionDict(x, d, "GoToR"); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Fatalf("valid remote destination produced a diagnostic: %s", buf.String())
	}
	d["D"] = types.Array{types.Integer(-1), types.Name("Fit")}
	if err := validateGoToRActionDict(x, d, "GoToR"); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); !strings.Contains(got, "digested:") || !strings.Contains(got, "non-negative page number") {
		t.Fatalf("missing digested diagnostic: %s", got)
	}
}

// TestGoToRDestinationErrorObject preserves the indirect destination's object number.
func TestGoToRDestinationErrorObject(t *testing.T) {
	x := sharedActionXRefTable(100, nil)
	x.Table[49] = model.NewXRefTableEntryGen0(types.Array{types.Integer(-1), types.Name("Fit")})
	d := types.Dict{"F": types.StringLiteral("go.pdf"), "D": *types.NewIndirectRef(49, 0)}
	err := validateGoToRActionDict(x, d, "GoToR")
	requireActionValidationObject(t, err, 49)
}

// TestGoToEDestinations checks embedded destinations and the relaxed legacy Dest fallback.
func TestGoToEDestinations(t *testing.T) {
	for _, tc := range []struct {
		name      string
		page      types.Object
		entry     string
		mode      int
		wantError bool
	}{
		{"firstPage", types.Integer(0), "D", model.ValidationStrict, false},
		{"laterPage", types.Integer(42), "D", model.ValidationStrict, false},
		{"negativePage", types.Integer(-1), "D", model.ValidationStrict, true},
		{"localPage", *types.NewIndirectRef(1, 0), "D", model.ValidationStrict, true},
		{"legacyDestRelaxed", types.Integer(0), "Dest", model.ValidationRelaxed, false},
		{"legacyDestStrict", types.Integer(0), "Dest", model.ValidationStrict, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x := sharedActionXRefTable(100, map[int]types.Dict{1: {"Type": types.Name("Page")}})
			x.ValidationMode = tc.mode
			d := types.Dict{
				"S":      types.Name("GoToE"),
				"T":      types.Dict{"R": types.Name("C"), "N": types.StringLiteral("go.pdf")},
				tc.entry: types.Array{tc.page, types.Name("Fit")},
			}
			err := validateGoToEActionDict(t.Context(), x, d, "GoToE")
			if (err != nil) != tc.wantError {
				t.Fatalf("got %v, want error %t", err, tc.wantError)
			}
			if tc.entry == "Dest" && !tc.wantError {
				if d["D"] == nil || d["Dest"] != nil {
					t.Fatal("legacy Dest entry was not repaired to D")
				}
			}
		})
	}
}
