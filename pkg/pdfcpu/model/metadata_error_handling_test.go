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

package model

import (
	"errors"
	"testing"
)

// TestRemoveKeywordsRejectsMissingMetadata verifies the public compatibility wrapper preserves the sentinel.
func TestRemoveKeywordsRejectsMissingMetadata(t *testing.T) {
	if err := RemoveKeywords(nil); !errors.Is(err, ErrMissingMetadata) {
		t.Fatalf("expected %v, got %v", ErrMissingMetadata, err)
	}
}

// TestRemoveKeywordsReportsContentChanges verifies the internal helper distinguishes changes from no-ops.
func TestRemoveKeywordsReportsContentChanges(t *testing.T) {
	tests := []struct {
		name        string
		metadata    string
		want        string
		wantChanged bool
	}{
		{
			name:     "no keywords",
			metadata: "<x:xmpmeta>unchanged</x:xmpmeta>",
			want:     "<x:xmpmeta>unchanged</x:xmpmeta>",
		},
		{
			name:        "keywords without subject",
			metadata:    "before<pdf:Keywords>alpha</pdf:Keywords>after",
			want:        "beforeafter",
			wantChanged: true,
		},
		{
			name: "keywords and subject",
			metadata: "before<pdf:Keywords>alpha</pdf:Keywords>middle" +
				"<dc:subject><rdf:Bag><rdf:li>alpha</rdf:li></rdf:Bag></dc:subject>after",
			want:        "beforemiddleafter",
			wantChanged: true,
		},
		{
			name:     "malformed keywords",
			metadata: "before<pdf:Keywords>alpha after",
			want:     "before<pdf:Keywords>alpha after",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := []byte(tt.metadata)
			changed, err := removeKeywords(&metadata)
			if err != nil {
				t.Fatal(err)
			}
			if changed != tt.wantChanged {
				t.Fatalf("changed: got %t, want %t", changed, tt.wantChanged)
			}
			if got := string(metadata); got != tt.want {
				t.Fatalf("metadata: got %q, want %q", got, tt.want)
			}
		})
	}
}
