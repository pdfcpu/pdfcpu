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

package sign_test

import (
	"crypto/x509"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/sign"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type exportedValidationFunc func(
	io.ReaderAt,
	types.Dict,
	bool,
	bool,
	bool,
	int,
	*x509.CertPool,
	*model.SignatureValidationResult,
	*model.Context,
) error

type domainValidationFunc func(
	io.ReaderAt,
	*model.Context,
	bool,
) ([]*model.SignatureValidationResult, error)

type apiFileValidationFunc func(
	string,
	bool,
	*model.Configuration,
) ([]*model.SignatureValidationResult, error)

type apiRawValidationFunc func(
	api.ReadSeekerAt,
	bool,
	*model.Configuration,
) ([]*model.SignatureValidationResult, error)

type apiPresentationFunc func(
	string,
	bool,
	bool,
	*model.Configuration,
) ([]string, error)

var (
	_ exportedValidationFunc = sign.ValidatePKCS7Signatures
	_ exportedValidationFunc = sign.ValidateDTS
	_ exportedValidationFunc = sign.ValidateX509RSASHA1Signature
	_ domainValidationFunc   = pdfcpu.ValidateSignatures
	_ apiFileValidationFunc  = api.ValidateSignatures
	_ apiRawValidationFunc   = api.ValidateSignaturesRaw
	_ apiPresentationFunc    = api.ValidateSignaturesFile
)

// TestPublicSignatureAPIRejectsServiceInjection prevents the public API,
// dispatcher, and signature package from exposing service/evaluator injection.
func TestPublicSignatureAPIRejectsServiceInjection(t *testing.T) {
	dirs := signatureAPIDirectories(t)
	for _, dir := range dirs {
		inspectSignatureAPIDirectory(t, dir)
	}
}

func signatureAPIDirectories(t *testing.T) []string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve architecture test location")
	}
	signDir := filepath.Dir(filename)
	pdfcpuDir := filepath.Dir(signDir)
	return []string{
		signDir,
		pdfcpuDir,
		filepath.Join(filepath.Dir(pdfcpuDir), "api"),
	}
}

func inspectSignatureAPIDirectory(t *testing.T, dir string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		inspectSignatureAPIFile(t, filepath.Join(dir, entry.Name()))
	}
}

func inspectSignatureAPIFile(t *testing.T, filename string) {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), filename, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch decl := node.(type) {
		case *ast.FuncDecl:
			inspectExportedSignatureFunction(t, filename, decl)
		case *ast.TypeSpec:
			inspectExportedSignatureType(t, filename, decl)
		}
		return true
	})
}

func inspectExportedSignatureFunction(t *testing.T, filename string, decl *ast.FuncDecl) {
	t.Helper()

	if !decl.Name.IsExported() || !isSignatureAPIName(decl.Name.Name) {
		return
	}
	if hasInjectionName(decl.Name.Name) || fieldListHasInjection(decl.Type.Params) {
		t.Errorf("%s exposes service injection through function %s", filename, decl.Name.Name)
	}
}

func inspectExportedSignatureType(t *testing.T, filename string, decl *ast.TypeSpec) {
	t.Helper()

	if !decl.Name.IsExported() || !isSignatureAPIName(decl.Name.Name) {
		return
	}
	if hasInjectionName(decl.Name.Name) || expressionHasInjection(decl.Type) {
		t.Errorf("%s exposes service injection through type %s", filename, decl.Name.Name)
	}
}

func isSignatureAPIName(name string) bool {
	return strings.Contains(name, "Signatur") || strings.Contains(name, "DTS")
}

func hasInjectionName(name string) bool {
	return strings.Contains(name, "Service") ||
		strings.Contains(name, "Evaluator") ||
		strings.Contains(name, "Validator")
}

func fieldListHasInjection(fields *ast.FieldList) bool {
	if fields == nil {
		return false
	}
	for _, field := range fields.List {
		if expressionHasInjection(field.Type) {
			return true
		}
	}
	return false
}

func expressionHasInjection(expr ast.Expr) bool {
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if ok && hasInjectionName(ident.Name) {
			found = true
			return false
		}
		return !found
	})
	return found
}
