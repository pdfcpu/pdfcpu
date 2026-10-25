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

// GHSA-vwmc-mhq7-fqhc: an object shared by several parents in a resource, pattern, property list or visibility
// expression graph must be validated once per depth, not once per path, so a few KB cannot cause 2^n work.
// https://github.com/pdfcpu/pdfcpu/security/advisories/GHSA-vwmc-mhq7-fqhc

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

const ghsaVwmcEmptyStream = "<< /Length 0 >>\nstream\n\nendstream"

func ghsaVwmcPDF(objs map[int]string) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n")
	n := slices.Max(slices.Collect(maps.Keys(objs)))
	offsets := make([]int, n+1)
	for i := 1; i <= n; i++ {
		if body, ok := objs[i]; ok {
			offsets[i] = b.Len()
			fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i, body)
		}
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", n+1)
	for i := 1; i <= n; i++ {
		if offsets[i] == 0 {
			b.WriteString("0000000000 65535 f \n")
			continue
		}
		fmt.Fprintf(&b, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", n+1, xref)
	return b.Bytes()
}

func ghsaVwmcObjects(page string) map[int]string {
	return map[int]string{
		1: "<< /Type /Catalog /Pages 2 0 R >>",
		2: "<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		3: "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Contents 4 0 R " + page + " >>",
		4: ghsaVwmcEmptyStream,
	}
}

func ghsaVwmcRefs(key string, objNr, count int) string {
	refs := make([]string, count)
	for i := range refs {
		refs[i] = fmt.Sprintf("/%s%d %d 0 R", key, i, objNr)
	}
	return strings.Join(refs, " ")
}

func ghsaVwmcTilingPattern(resources string) string {
	return "<< /Type /Pattern /PatternType 1 /PaintType 1 /TilingType 1 /BBox [0 0 1 1] /XStep 1 /YStep 1 " +
		"/Resources " + resources + " /Length 0 >>\nstream\n\nendstream"
}

// ghsaVwmcVisibilityExpression chains levels arrays, each referring twice to the next one.
func ghsaVwmcVisibilityExpression(levels int) map[int]string {
	objs := ghsaVwmcObjects("/Resources << >> /Annots [5 0 R]")
	objs[5] = "<< /Type /Annot /Subtype /Text /Rect [0 0 10 10] /OC 7 0 R >>"
	objs[6] = "<< /Type /OCG /Name (x) >>"
	objs[7] = "<< /Type /OCMD /OCGs 6 0 R /VE 8 0 R >>"
	for i := range levels {
		objs[8+i] = fmt.Sprintf("[/And %d 0 R %d 0 R]", 9+i, 9+i)
	}
	objs[8+levels] = "[/And 6 0 R 6 0 R]"
	return objs
}

// ghsaVwmcPatternResources chains resource dictionaries whose patterns all refer to the next resource dictionary.
func ghsaVwmcPatternResources(levels int) map[int]string {
	objs := ghsaVwmcObjects("/Resources 10 0 R")
	res := func(i int) int { return 10 + 2*i }
	for i := range levels {
		objs[res(i)] = "<< /Pattern << " + ghsaVwmcRefs("P", res(i)+1, 2) + " >> >>"
		objs[res(i)+1] = ghsaVwmcTilingPattern(fmt.Sprintf("%d 0 R", res(i+1)))
	}
	objs[res(levels)] = "<< >>"
	return objs
}

// ghsaVwmcProperties chains resource dictionaries whose property lists all refer to the next property list.
func ghsaVwmcProperties(levels int) map[int]string {
	objs := ghsaVwmcObjects("/Resources 10 0 R")
	res := func(i int) int { return 10 + 2*i }
	for i := range levels {
		objs[res(i)] = "<< /Properties << " + ghsaVwmcRefs("P", res(i)+1, 2) + " >> >>"
		objs[res(i)+1] = fmt.Sprintf("<< /Resources %d 0 R >>", res(i+1))
	}
	objs[res(levels)] = "<< >>"
	return objs
}

// ghsaVwmcTiling chains tiling patterns with direct resources, so only patterns have object identities.
func ghsaVwmcTiling(levels int) map[int]string {
	resources := func(next int) string { return "<< /Pattern << " + ghsaVwmcRefs("P", next, 2) + " >> >>" }
	objs := ghsaVwmcObjects("/Resources " + resources(10))
	for i := range levels {
		r := "<< >>"
		if i < levels-1 {
			r = resources(11 + i)
		}
		objs[10+i] = ghsaVwmcTilingPattern(r)
	}
	return objs
}

func ghsaVwmcValidate(t *testing.T, objs map[int]string, mode, maxDepth int) error {
	t.Helper()
	conf := model.NewStatelessConfiguration()
	conf.ValidationMode = mode
	conf.Limits.MaxRecursionDepth = maxDepth
	ctx, err := pdfcpu.Read(t.Context(), bytes.NewReader(ghsaVwmcPDF(objs)), conf)
	if err != nil {
		t.Fatal(err)
	}
	return XRefTable(t.Context(), ctx)
}

// ghsaVwmcValidateWithin fails if validation does not finish within the budget. The visibility expression
// traversal does not poll the context, so a goroutine bounds the wait instead of a context deadline.
func ghsaVwmcValidateWithin(t *testing.T, objs map[int]string, mode int) {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		conf := model.NewStatelessConfiguration()
		conf.ValidationMode = mode
		ctx, err := pdfcpu.Read(t.Context(), bytes.NewReader(ghsaVwmcPDF(objs)), conf)
		if err == nil {
			err = XRefTable(t.Context(), ctx)
		}
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("validation of a shared graph did not finish: work grows per path")
	}
}

// TestGHSA_vwmc_mhq7_fqhc_SharedGraphsValidateOnce verifies 48 levels of doubly shared children validate quickly.
func TestGHSA_vwmc_mhq7_fqhc_SharedGraphsValidateOnce(t *testing.T) {
	const levels = 48
	for _, tc := range []struct {
		name string
		objs map[int]string
	}{
		{"visibility expression", ghsaVwmcVisibilityExpression(levels)},
		{"pattern resources", ghsaVwmcPatternResources(levels)},
		{"properties", ghsaVwmcProperties(levels)},
		{"tiling patterns", ghsaVwmcTiling(levels)},
	} {
		for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
			t.Run(fmt.Sprintf("%s/mode %d", tc.name, mode), func(t *testing.T) {
				ghsaVwmcValidateWithin(t, tc.objs, mode)
			})
		}
	}
}

// TestGHSA_vwmc_mhq7_fqhc_VisibilityExpressionChecksDeeperPath verifies a shared expression validated on a short
// path is revalidated when a longer path reaches it, so the depth limit still applies to its descendants.
func TestGHSA_vwmc_mhq7_fqhc_VisibilityExpressionChecksDeeperPath(t *testing.T) {
	objs := ghsaVwmcObjects("/Resources << >> /Annots [5 0 R]")
	objs[5] = "<< /Type /Annot /Subtype /Text /Rect [0 0 10 10] /OC 7 0 R >>"
	objs[6] = "<< /Type /OCG /Name (x) >>"
	objs[7] = "<< /Type /OCMD /OCGs 6 0 R /VE 8 0 R >>"
	// Shared expression 10–12 is reached at depth 1 and, through 20–22, at depth 4; its leaf at depth 3 and 6.
	objs[8] = "[/And 10 0 R 20 0 R]"
	objs[10] = "[/And 11 0 R 6 0 R]"
	objs[11] = "[/And 12 0 R 6 0 R]"
	objs[12] = "[/And 6 0 R 6 0 R]"
	objs[20] = "[/And 21 0 R 6 0 R]"
	objs[21] = "[/And 22 0 R 6 0 R]"
	objs[22] = "[/And 10 0 R 6 0 R]"
	for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
		if err := ghsaVwmcValidate(t, objs, mode, 5); !errors.Is(err, model.ErrMaxRecursionDepthExceeded) {
			t.Fatalf("mode %d: got %v, want depth error", mode, err)
		}
		if err := ghsaVwmcValidate(t, objs, mode, 6); err != nil {
			t.Fatalf("mode %d: %v", mode, err)
		}
	}
}

// TestGHSA_vwmc_mhq7_fqhc_PropertiesCheckDeeperPath verifies a shared property list validated on a short path is
// revalidated when a longer path reaches it, so the resource graph limit still applies to its descendants.
func TestGHSA_vwmc_mhq7_fqhc_PropertiesCheckDeeperPath(t *testing.T) {
	const shared, detour = 5, 8
	objs := ghsaVwmcObjects("/Resources 10 0 R")
	objs[10] = "<< /Properties << /A 100 0 R /B 200 0 R >> >>"
	for i := range shared {
		p, r := 100+2*i, 101+2*i
		objs[p] = fmt.Sprintf("<< /Resources %d 0 R >>", r)
		objs[r] = fmt.Sprintf("<< /Properties << /A %d 0 R >> >>", p+2)
	}
	objs[100+2*shared] = "<< >>"
	for i := range detour {
		p, r := 200+2*i, 201+2*i
		next := p + 2
		if i == detour-1 {
			next = 100
		}
		objs[p] = fmt.Sprintf("<< /Resources %d 0 R >>", r)
		objs[r] = fmt.Sprintf("<< /Properties << /B %d 0 R >> >>", next)
	}
	for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
		if err := ghsaVwmcValidate(t, objs, mode, 10); !errors.Is(err, model.ErrMaxRecursionDepthExceeded) {
			t.Fatalf("mode %d: got %v, want depth error", mode, err)
		}
		if err := ghsaVwmcValidate(t, objs, mode, shared+detour); err != nil {
			t.Fatalf("mode %d: %v", mode, err)
		}
	}
}

// TestGHSA_vwmc_mhq7_fqhc_PatternsCheckDeeperPath verifies a shared tiling pattern validated on a short path is
// revalidated when a longer path reaches it, so the pattern graph limit still applies to its descendants.
func TestGHSA_vwmc_mhq7_fqhc_PatternsCheckDeeperPath(t *testing.T) {
	const shared, detour = 5, 8
	ref := func(next int) string { return fmt.Sprintf("<< /Pattern << /A %d 0 R >> >>", next) }
	objs := ghsaVwmcObjects("/Resources << /Pattern << /A 100 0 R /B 200 0 R >> >>")
	for i := range shared {
		r := "<< >>"
		if i < shared-1 {
			r = ref(101 + i)
		}
		objs[100+i] = ghsaVwmcTilingPattern(r)
	}
	for i := range detour {
		next := 201 + i
		if i == detour-1 {
			next = 100
		}
		objs[200+i] = ghsaVwmcTilingPattern(ref(next))
	}
	for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
		if err := ghsaVwmcValidate(t, objs, mode, 10); !errors.Is(err, model.ErrMaxRecursionDepthExceeded) {
			t.Fatalf("mode %d: got %v, want depth error", mode, err)
		}
		if err := ghsaVwmcValidate(t, objs, mode, shared+detour+1); err != nil {
			t.Fatalf("mode %d: %v", mode, err)
		}
	}
}
