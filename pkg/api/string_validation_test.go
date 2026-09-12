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
)

func TestAPIUserStringGuardsRejectEmptyStrings(t *testing.T) {
	for _, tt := range []struct {
		name string
		fn   func() error
		want string
	}{
		{
			name: "add keywords",
			fn: func() error {
				return AddKeywordsFile(t.Context(), "missing.pdf", "", []string{""}, nil)
			},
			want: "keyword must not be empty",
		},
		{
			name: "remove keywords",
			fn: func() error {
				return RemoveKeywordsFile(t.Context(), "missing.pdf", "", []string{""}, nil)
			},
			want: "keyword must not be empty",
		},
		{
			name: "add properties name",
			fn: func() error {
				return AddPropertiesFile(t.Context(), "missing.pdf", "", map[string]string{"": "value"}, nil)
			},
			want: "property name must not be empty",
		},
		{
			name: "add properties value",
			fn: func() error {
				return AddPropertiesFile(t.Context(), "missing.pdf", "", map[string]string{"subject": ""}, nil)
			},
			want: "property value must not be empty",
		},
		{
			name: "remove properties",
			fn: func() error {
				return RemovePropertiesFile(t.Context(), "missing.pdf", "", []string{" "}, nil)
			},
			want: "property name must not be empty",
		},
		{
			name: "add attachments",
			fn: func() error {
				return AddAttachmentsFile(t.Context(), "missing.pdf", "", []string{",desc"}, false, nil)
			},
			want: "attachment filename must not be empty",
		},
		{
			name: "remove attachments",
			fn: func() error {
				return RemoveAttachmentsFile(t.Context(), "missing.pdf", "", []string{""}, nil)
			},
			want: "attachment filename must not be empty",
		},
		{
			name: "extract attachments",
			fn: func() error {
				return ExtractAttachmentsFile(t.Context(), "missing.pdf", "out", []string{""}, nil)
			},
			want: "attachment filename must not be empty",
		},
		{
			name: "remove form fields",
			fn: func() error {
				return RemoveFormFieldsFile(t.Context(), "missing.pdf", "", []string{""}, nil)
			},
			want: "form field ID or name must not be empty",
		},
		{
			name: "lock form fields",
			fn: func() error {
				return LockFormFieldsFile(t.Context(), "missing.pdf", "", []string{""}, nil)
			},
			want: "form field ID or name must not be empty",
		},
		{
			name: "unlock form fields",
			fn: func() error {
				return UnlockFormFieldsFile(t.Context(), "missing.pdf", "", []string{""}, nil)
			},
			want: "form field ID or name must not be empty",
		},
		{
			name: "reset form fields",
			fn: func() error {
				return ResetFormFieldsFile(t.Context(), "missing.pdf", "", []string{""}, nil)
			},
			want: "form field ID or name must not be empty",
		},
		{
			name: "remove annotations",
			fn: func() error {
				return RemoveAnnotationsFile(t.Context(), "missing.pdf", "", nil, []string{""}, nil, nil, false)
			},
			want: "annotation ID or type must not be empty",
		},
		{
			name: "font cheatsheet",
			fn: func() error {
				return CreateCheatSheetsUserFonts(t.Context(), []string{""})
			},
			want: "font name must not be empty",
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
