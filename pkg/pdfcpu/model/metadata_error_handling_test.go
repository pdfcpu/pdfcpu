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
	"encoding/xml"
	"errors"
	"testing"
	"time"
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

// TestUserDateUnmarshalXML verifies XMP date forms and preserves explicit offsets and fractional seconds.
func TestUserDateUnmarshalXML(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  string
	}{
		{"2020", "2020-01-01T00:00:00Z"},
		{"2020-02", "2020-02-01T00:00:00Z"},
		{"2020-02-29", "2020-02-29T00:00:00Z"},
		{"2020-02-29T12:34", "2020-02-29T12:34:00Z"},
		{"2020-02-29T12:34Z", "2020-02-29T12:34:00Z"},
		{"2020-02-29T12:34+05:30", "2020-02-29T12:34:00+05:30"},
		{"2020-02-29T12:34-03:30", "2020-02-29T12:34:00-03:30"},
		{"2020-02-29T12:34:56", "2020-02-29T12:34:56Z"},
		{"2020-02-29T12:34:56Z", "2020-02-29T12:34:56Z"},
		{"2020-02-29T12:34:56+05:30", "2020-02-29T12:34:56+05:30"},
		{"2020-02-29T12:34:56-03:30", "2020-02-29T12:34:56-03:30"},
		{"2020-02-29T12:34:56.123456789", "2020-02-29T12:34:56.123456789Z"},
		{"2020-02-29T12:34:56.123456789Z", "2020-02-29T12:34:56.123456789Z"},
		{"2020-02-29T12:34:56.125+05:30", "2020-02-29T12:34:56.125+05:30"},
		{"2020-02-29T12:34:56.125-03:30", "2020-02-29T12:34:56.125-03:30"},
		{" 2020-02-29T12:34:56Z ", "2020-02-29T12:34:56Z"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			var date UserDate
			if err := xml.Unmarshal([]byte("<date>"+tt.input+"</date>"), &date); err != nil {
				t.Fatal(err)
			}
			if got := time.Time(date).Format(time.RFC3339Nano); got != tt.want {
				t.Fatalf("date = %s, want %s", got, tt.want)
			}
		})
	}
}

// TestUserDateUnmarshalXMLRejectsInvalid retains the receiver when date or XML parsing fails.
func TestUserDateUnmarshalXMLRejectsInvalid(t *testing.T) {
	for _, input := range []string{
		"", "not a date", "20", "2020-13", "2019-02-29", "2020-02-30",
		"2020-02-29T24:00Z", "2020-02-29T12:60Z", "2020-02-29T12:34:60Z",
		"2020-02-29T12:34:56+99:00", "2020-02-29junk", "2020-02-29T12:34:56Zjunk",
		"2020-02-29</wrong>",
	} {
		t.Run(input, func(t *testing.T) {
			before := time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC)
			date := UserDate(before)
			if err := xml.Unmarshal([]byte("<date>"+input+"</date>"), &date); err == nil {
				t.Fatal("expected date or XML parse error")
			}
			if time.Time(date) != before {
				t.Fatal("failed decode changed the receiver")
			}
		})
	}
}
