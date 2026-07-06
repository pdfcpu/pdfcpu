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

import (
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestWatermarkConstructorsRejectEmptyUserStrings(t *testing.T) {
	for _, tt := range []struct {
		name string
		fn   func() error
		want string
	}{
		{
			name: "text watermark",
			fn: func() error {
				_, err := TextWatermark("", "", false, false, types.POINTS)
				return err
			},
			want: "watermark text must not be empty",
		},
		{
			name: "image stamp",
			fn: func() error {
				_, err := ImageWatermark(" \t", "", true, false, types.POINTS)
				return err
			},
			want: "stamp image filename must not be empty",
		},
		{
			name: "PDF watermark",
			fn: func() error {
				_, err := PDFWatermark("", "", false, false, types.POINTS)
				return err
			},
			want: "watermark PDF filename must not be empty",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got %q", tt.want, err.Error())
			}
		})
	}
}

func TestWatermarkReaderConstructorsRejectNil(t *testing.T) {
	for _, tt := range []struct {
		name string
		fn   func() error
		want string
	}{
		{
			name: "image reader",
			fn: func() error {
				_, err := ImageWatermarkForReader(nil, "", false, false, types.POINTS)
				return err
			},
			want: "missing image reader",
		},
		{
			name: "PDF read seeker",
			fn: func() error {
				_, err := PDFWatermarkForReadSeeker(nil, 1, "", false, false, types.POINTS)
				return err
			},
			want: "missing PDF read seeker",
		},
		{
			name: "PDF multi read seeker",
			fn: func() error {
				_, err := PDFMultiWatermarkForReadSeeker(nil, 1, 1, "", false, false, types.POINTS)
				return err
			},
			want: "missing PDF read seeker",
		},
		{
			name: "PDF read seeker file helper",
			fn: func() error {
				return AddPDFWatermarksForReadSeekerFile("", "", nil, false, nil, 1, "", nil)
			},
			want: "missing PDF read seeker",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Fatal("expected error")
			}
			if err.Error() != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, err.Error())
			}
		})
	}
}
