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

// GHSA-fjh6-rrhv-4g63: traditional xref tables must enforce the configured MaxObjectCount and MaxXRefEntries limits.
// https://github.com/pdfcpu/pdfcpu/security/advisories/GHSA-fjh6-rrhv-4g63

import (
	"bufio"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func TestGHSA_fjh6_rrhv_4g63_ParseXRefTableSubSectionRejectsLimits(t *testing.T) {
	tests := []struct {
		name   string
		fields []string
		limits model.ResourceLimits
	}{
		{"negative start", []string{"-1", "1"}, model.ResourceLimits{MaxObjectCount: 2, MaxXRefEntries: 2}},
		{"negative count", []string{"0", "-1"}, model.ResourceLimits{MaxObjectCount: 2, MaxXRefEntries: 2}},
		{"object range", []string{"1", "2"}, model.ResourceLimits{MaxObjectCount: 2, MaxXRefEntries: 2}},
		{"entry count", []string{"0", "2"}, model.ResourceLimits{MaxObjectCount: 3, MaxXRefEntries: 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xRefTable := &model.XRefTable{Table: map[int]*model.XRefTableEntry{}}
			s := bufio.NewScanner(strings.NewReader("trailer\n"))
			if _, err := parseXRefTableSubSection(xRefTable, tt.limits, s, tt.fields, 0, 0); err == nil {
				t.Fatal("expected xref subsection limit error")
			}
		})
	}
}

func TestGHSA_fjh6_rrhv_4g63_ParseXRefTableSubSectionBoundsExtraEntries(t *testing.T) {
	xRefTable := &model.XRefTable{Table: map[int]*model.XRefTableEntry{}}
	s := bufio.NewScanner(strings.NewReader("0000000000 65535 f\n0000000000 65535 f\ntrailer\n"))
	limits := model.ResourceLimits{MaxObjectCount: 2, MaxXRefEntries: 1}

	_, err := parseXRefTableSubSection(xRefTable, limits, s, []string{"0", "1"}, 0, 0)
	if err == nil || !strings.Contains(err.Error(), "xref entry count exceeds limit 1") {
		t.Fatalf("got %v, want xref entry count limit error", err)
	}
	if len(xRefTable.Table) != 1 {
		t.Fatalf("parsed %d xref entries, want 1", len(xRefTable.Table))
	}
}

func TestGHSA_fjh6_rrhv_4g63_ParseXRefTableSubSectionRepairsBoundedExtraEntry(t *testing.T) {
	xRefTable := &model.XRefTable{Table: map[int]*model.XRefTableEntry{}}
	s := bufio.NewScanner(strings.NewReader("0000000000 65535 f\n0000000000 65535 f\ntrailer\n"))
	limits := model.ResourceLimits{MaxObjectCount: 2, MaxXRefEntries: 2}

	line, err := parseXRefTableSubSection(xRefTable, limits, s, []string{"0", "1"}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if line != "trailer" || len(xRefTable.Table) != 2 {
		t.Fatalf("got line %q and %d entries, want trailer and 2 entries", line, len(xRefTable.Table))
	}
}

func TestGHSA_fjh6_rrhv_4g63_ParseXRefSectionUsesConfiguredLimits(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	conf.Limits.MaxObjectCount = 1
	ctx := &model.Context{
		Configuration: conf,
		XRefTable:     &model.XRefTable{Table: map[int]*model.XRefTableEntry{}},
	}
	s := bufio.NewScanner(strings.NewReader(""))
	ssCount := 0
	offset := int64(0)

	_, err := parseXRefSection(t.Context(), ctx, s, []string{"0", "2"}, &ssCount, &offset, 0, 0)
	if err == nil || !strings.Contains(err.Error(), "xref subsection object range exceeds limit 1") {
		t.Fatalf("got %v, want configured object range limit error", err)
	}
}
