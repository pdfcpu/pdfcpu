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
	"errors"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func testXRef(t *testing.T, mode int) *model.XRefTable {
	t.Helper()

	return testXRefVersion(t, mode, model.V17)
}

func testXRefVersion(t *testing.T, mode int, version model.Version) *model.XRefTable {
	t.Helper()

	ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	ctx.XRefTable.ValidationMode = mode
	ctx.XRefTable.HeaderVersion = &version
	return ctx.XRefTable
}

func requireErrContains(t *testing.T, err error, want string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error containing %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("got %v, want %s", err, want)
	}
}

func requireErrChainContains(t *testing.T, err error, wants ...string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error containing %q", wants)
	}

	msg := err.Error()
	offset := 0
	for _, want := range wants {
		i := strings.Index(msg[offset:], want)
		if i < 0 {
			t.Fatalf("got %v, want ordered error context %q", err, wants)
		}
		offset += i + len(want)
	}
}

func requireErrIs(t *testing.T, err, target error) {
	t.Helper()

	if !errors.Is(err, target) {
		t.Fatalf("got %v, want %v", err, target)
	}
}

func requireErrNotContains(t *testing.T, err error, unwanted string) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), unwanted) {
		t.Fatalf("got %v, did not want %s", err, unwanted)
	}
}
