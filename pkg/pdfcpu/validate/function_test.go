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
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func TestValidateFunctionReportsMissingObjectContext(t *testing.T) {
	err := validateFunction(testXRef(t, model.ValidationStrict), nil)
	if err == nil {
		t.Fatal("expected missing function object error")
	}
	if got := err.Error(); got != "function: missing object" {
		t.Fatalf("got %q, want function: missing object", got)
	}
}

func TestProcessFunctionReportsUnsupportedObjectContext(t *testing.T) {
	err := processFunction(testXRef(t, model.ValidationStrict), types.Integer(1))
	requireErrContains(t, err, "function object: expected dict or stream dict")
}

func TestProcessFunctionDictReportsFunctionTypeContext(t *testing.T) {
	err := processFunction(testXRef(t, model.ValidationStrict), types.Dict{
		"FunctionType": types.Integer(0),
	})
	requireErrContains(t, err, "function dictionary: FunctionType")
}

func TestProcessFunctionStreamDictReportsFunctionTypeContext(t *testing.T) {
	err := processFunction(testXRef(t, model.ValidationStrict), types.StreamDict{
		Dict: types.Dict{
			"FunctionType": types.Integer(2),
		},
	})
	requireErrContains(t, err, "function stream dictionary: FunctionType")
}

func TestProcessFunctionDictReportsSubtypeEntryContext(t *testing.T) {
	err := processFunction(testXRef(t, model.ValidationStrict), types.Dict{
		"FunctionType": types.Integer(2),
	})
	requireErrChainContains(t, err, "exponential interpolation function", "exponentialInterpolationFunctionDict.Domain")
}

func TestProcessFunctionStreamDictReportsSubtypeEntryContext(t *testing.T) {
	err := processFunction(testXRef(t, model.ValidationStrict), types.StreamDict{
		Dict: types.Dict{
			"FunctionType": types.Integer(0),
		},
	})
	requireErrChainContains(t, err, "sampled function", "sampledFunctionStreamDict.Domain")
}
