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
	"errors"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestInfoNilGuards(t *testing.T) {
	tests := []struct {
		name    string
		fn      func() error
		wantErr error
	}{
		{
			name: "info missing context",
			fn: func() error {
				_, err := Info(nil, "", nil, false)
				return err
			},
			wantErr: ErrMissingPDFContext,
		},
		{
			name: "list info missing info",
			fn: func() error {
				_, err := ListInfo(nil, nil, false)
				return err
			},
			wantErr: ErrMissingPDFInfo,
		},
		{
			name:    "detect watermarks missing context",
			fn:      func() error { return DetectWatermarks(nil) },
			wantErr: ErrMissingPDFContext,
		},
		{
			name:    "detect page tree watermarks missing context",
			fn:      func() error { return DetectPageTreeWatermarks(nil) },
			wantErr: ErrMissingPDFContext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestInfoPageBoundaryErrorsIncludePageTreeContext(t *testing.T) {
	ctx, err := CreateContextWithXRefTable(nil, types.PaperSize["A4"])
	if err != nil {
		t.Fatal(err)
	}
	ctx.RootDict["Pages"] = *types.NewIndirectRef(999, 0)

	_, err = Info(ctx, "broken.pdf", nil, false)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"page boundaries", "page tree"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in %q", want, err.Error())
		}
	}
}

func TestDetectWatermarksErrorsIncludeOptionalContentAndPageTreeContext(t *testing.T) {
	t.Run("optional content", func(t *testing.T) {
		ctx, err := CreateContextWithXRefTable(nil, types.PaperSize["A4"])
		if err != nil {
			t.Fatal(err)
		}
		ctx.RootDict["OCProperties"] = *types.NewIndirectRef(999, 0)

		err = DetectWatermarks(ctx)
		if err == nil {
			t.Fatal("expected error")
		}
		for _, want := range []string{"optional content groups", "optional content properties"} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q in %q", want, err.Error())
			}
		}
	})

	t.Run("page tree", func(t *testing.T) {
		ctx, err := model.NewContext(strings.NewReader(""), model.NewDefaultConfiguration())
		if err != nil {
			t.Fatal(err)
		}
		ctx.RootDict = types.Dict{}

		err = DetectPageTreeWatermarks(ctx)
		if err == nil {
			t.Fatal("expected error")
		}
		for _, want := range []string{"page tree watermarks", "page tree"} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("expected %q in %q", want, err.Error())
			}
		}
	})
}
