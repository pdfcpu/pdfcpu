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
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestEqualObjectsRejectsUnloadedSecondStream(t *testing.T) {
	sd1 := types.StreamDict{Dict: types.Dict{}, Raw: []byte("loaded")}
	sd2 := types.StreamDict{Dict: types.Dict{}}

	_, err := EqualObjects(t.Context(), sd1, sd2, newXRefTable(NewDefaultConfiguration()), nil)
	if err == nil || err.Error() != "stream dict not loaded" {
		t.Fatalf("got %v, want stream dict not loaded", err)
	}
}
