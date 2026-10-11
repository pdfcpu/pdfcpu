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

package pdfcpu

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// TestReadAliasedXRefEntriesHaveBudget checks that repeated parsing of one object exhausts the read budget.
func TestReadAliasedXRefEntriesHaveBudget(t *testing.T) {
	const aliases = 12000
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	off1 := b.Len()
	b.WriteString("1 0 obj\n<</Type/Catalog/Pages 2 0 R>>\nendobj\n")
	off2 := b.Len()
	b.WriteString("2 0 obj\n<</Type/Pages/Kids[]/Count 0>>\nendobj\n")
	off3 := b.Len()
	b.WriteString("3 0 obj\n(" + strings.Repeat("x", 256<<10) + ")\nendobj\n")
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n%010d 00000 n \n%010d 00000 n \n", aliases+3, off1, off2)
	for i := 0; i < aliases; i++ {
		fmt.Fprintf(&b, "%010d 00000 n \n", off3)
	}
	fmt.Fprintf(&b, "trailer\n<</Size %d/Root 1 0 R>>\nstartxref\n%d\n%%%%EOF\n", aliases+3, xref)

	for _, mode := range []int{model.ValidationStrict, model.ValidationRelaxed} {
		t.Run(fmt.Sprintf("mode=%d", mode), func(t *testing.T) {
			conf := model.NewDefaultConfiguration()
			conf.ValidationMode = mode
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			_, err := Read(ctx, bytes.NewReader(b.Bytes()), conf)
			if !errors.Is(err, errObjectReadBudget) {
				t.Fatalf("expected object read budget error, got %v", err)
			}
		})
	}
}
