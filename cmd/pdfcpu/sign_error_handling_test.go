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

package main

import (
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// TestSignaturesValidateCommandRequiresOneInput verifies the command parser owns argument cardinality.
func TestSignaturesValidateCommandRequiresOneInput(t *testing.T) {
	cmd := signaturesValidateCmd()
	tests := []struct {
		name string
		args []string
		ok   bool
	}{
		{name: "Missing", args: nil},
		{name: "OneInput", args: []string{"in.pdf"}, ok: true},
		{name: "ExtraInput", args: []string{"in.pdf", "other.pdf"}},
	}

	for _, tt := range tests {
		err := cmd.Args(cmd, tt.args)
		if tt.ok && err != nil {
			t.Errorf("%s: expected accepted arguments, got %v", tt.name, err)
		}
		if !tt.ok && err == nil {
			t.Errorf("%s: expected argument error", tt.name)
		}
	}
}

// TestSignaturesRemoveCommandRequiresOneOrTwoInputs verifies the command parser owns argument cardinality.
func TestSignaturesRemoveCommandRequiresOneOrTwoInputs(t *testing.T) {
	cmd := signaturesRemoveCmd()
	tests := []struct {
		name string
		args []string
		ok   bool
	}{
		{name: "Missing"},
		{name: "Input", args: []string{"in.pdf"}, ok: true},
		{name: "InputAndOutput", args: []string{"in.pdf", "out.pdf"}, ok: true},
		{name: "ExtraInput", args: []string{"in.pdf", "out.pdf", "other.pdf"}},
	}

	for _, tt := range tests {
		err := cmd.Args(cmd, tt.args)
		if tt.ok && err != nil {
			t.Errorf("%s: expected accepted arguments, got %v", tt.name, err)
		}
		if !tt.ok && err == nil {
			t.Errorf("%s: expected argument error", tt.name)
		}
	}
}

// TestHandleValidateSignaturesCommandRetainsInputPhase verifies filename checks remain at the command boundary.
func TestHandleValidateSignaturesCommandRetainsInputPhase(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	conf.CheckFileNameExt = true

	err := handleValidateSignaturesCommand(conf, []string{"input.txt"}, &signaturesValidateOptions{})
	if err == nil {
		t.Fatal("expected input extension failure")
	}
	if want := `input.txt needs extension ".pdf"`; !strings.Contains(err.Error(), want) {
		t.Fatalf("expected %q, got %q", want, err)
	}
}
