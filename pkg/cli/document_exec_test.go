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

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	stdlog "log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pdfcpuLog "github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type validationErrorWriter struct {
	err error
}

func (w validationErrorWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func validationReportTestPDF(objects []string) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n")
	offsets := make([]int, len(objects))
	for i, object := range objects {
		offsets[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}
	xrefOffset := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xrefOffset)
	return b.Bytes()
}

func validationNoticeTestPDF() []byte {
	return validationReportTestPDF([]string{
		"<<@/Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Resources <<>> >>",
	})
}

func type1RequirednessNoticeTestPDF() []byte {
	return validationReportTestPDF([]string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Resources << /Font << /F1 4 0 R >> >> >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /FirstChar 65 >>",
	})
}

func remoteDestinationNoticeTestPDF() []byte {
	return validationReportTestPDF([]string{
		"<< /Type /Catalog /Pages 2 0 R /OpenAction 4 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Resources <<>> >>",
		"<< /Type /Action /S /GoToR /F (remote.pdf) /D [-1 /Fit] >>",
	})
}

func annotationUnitIntervalNoticeTestPDF() []byte {
	return validationReportTestPDF([]string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Resources <<>> /Annots [4 0 R] >>",
		"<< /Type /Annot /Subtype /Text /Rect [0 0 10 10] /C [2 -1 0] >>",
	})
}

func stitchingFunctionNoticesTestPDF() []byte {
	return validationReportTestPDF([]string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Resources << /Shading << /S 5 0 R >> >> >>",
		"<< /FunctionType 3 /Domain [0 1] /Range 6 0 R " +
			"/Functions [<< /FunctionType 2 /Domain [0 1] /N 1 >> << /FunctionType 2 /Domain [0 1] /N 1 >>] " +
			"/Bounds 7 0 R /Encode [0 1 0 1] >>",
		"<< /ShadingType 2 /ColorSpace /DeviceGray /Coords [0 0 1 1] /Function 4 0 R >>",
		"[]",
		"[]",
	})
}

func simpleFontWidthsNoticesTestPDF() []byte {
	return validationReportTestPDF([]string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] " +
			"/Resources << /Font << /F1 4 0 R /F2 5 0 R /F3 6 0 R >> >> >>",
		"<< /Type /Font /Subtype /TrueType /BaseFont /MissingWidth /FirstChar 0 /LastChar 1 /Widths [500] >>",
		"<< /Type /Font /Subtype /TrueType /BaseFont /ExtraWidth /FirstChar 0 /LastChar 0 /Widths [500 600] >>",
		"<< /Type /Font /Subtype /TrueType /BaseFont /EmptySentinel /FirstChar 65535 /LastChar 0 /Widths [] >>",
	})
}

func indexedImageMaskNoticeTestPDF() []byte {
	return validationReportTestPDF([]string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] " +
			"/Resources << /XObject << /Im1 4 0 R >> >> >>",
		"<< /Type /XObject /Subtype /Image /Width 1 /Height 1 " +
			"/ColorSpace [/Indexed /DeviceRGB 1 <000000FFFFFF>] /BitsPerComponent 8 /Mask 5 0 R /Length 1 >>\n" +
			"stream\n0\nendstream",
		"[182 182 151 151 158 158]",
	})
}

func step24aClosureNoticesTestPDF() []byte {
	return validationReportTestPDF([]string{
		"<<@/Type /Catalog /Pages 2 0 R /OpenAction 4 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Resources << " +
			"/Font << /F1 5 0 R /F2 6 0 R >> /Shading << /S1 8 0 R >> /XObject << /Im1 11 0 R >> >> " +
			"/Annots [13 0 R] >>",
		"<< /Type /Action /S /GoToR /F (remote.pdf) /D [-1 /Fit] >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /FirstChar 65 >>",
		"<< /Type /Font /Subtype /TrueType /BaseFont /MissingWidth /FirstChar 0 /LastChar 1 /Widths [500] >>",
		"<< /FunctionType 3 /Domain [0 1] /Range 9 0 R " +
			"/Functions [<< /FunctionType 2 /Domain [0 1] /N 1 >> << /FunctionType 2 /Domain [0 1] /N 1 >>] " +
			"/Bounds 10 0 R /Encode [0 1 0 1] >>",
		"<< /ShadingType 2 /ColorSpace /DeviceGray /Coords [0 0 1 1] /Function 7 0 R >>",
		"[]",
		"[]",
		"<< /Type /XObject /Subtype /Image /Width 1 /Height 1 " +
			"/ColorSpace [/Indexed /DeviceRGB 1 <000000FFFFFF>] /BitsPerComponent 8 /Mask 12 0 R /Length 1 >>\n" +
			"stream\n0\nendstream",
		"[182 182 151 151 158 158]",
		"<< /Type /Annot /Subtype /Text /Rect [0 0 10 10] /C [2 -1 0] >>",
	})
}

func validationNoticeTestFileWithContent(t *testing.T, content []byte) string {
	t.Helper()
	inFile := filepath.Join(t.TempDir(), "validation-notice.pdf")
	if err := os.WriteFile(inFile, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return inFile
}

func validationNoticeTestFile(t *testing.T) string {
	t.Helper()
	return validationNoticeTestFileWithContent(t, validationNoticeTestPDF())
}

func TestValidateRendersReturnedNoticeExactlyOnce(t *testing.T) {
	var cliOutput bytes.Buffer
	pdfcpuLog.SetCLILogger(stdlog.New(&cliOutput, "", 0))
	defer pdfcpuLog.SetCLILogger(nil)

	conf := model.NewStatelessConfiguration()
	conf.ValidationMode = model.ValidationRelaxed
	cmd := ValidateCommand([]string{validationNoticeTestFile(t)}, conf)
	var noticeOutput bytes.Buffer
	cmd.NoticeOutput = &noticeOutput

	if _, err := validateCommand(t.Context(), cmd); err != nil {
		t.Fatal(err)
	}
	want := "pdfcpu digested: object dictionary contains a non-name key token " +
		"(obj#:1): parse: corrupt dictionary key: parse: corrupt name object\n"
	if got := noticeOutput.String(); got != want {
		t.Fatalf("notice output: got %q, want %q", got, want)
	}
	if strings.Contains(cliOutput.String(), "pdfcpu digested:") {
		t.Fatalf("notice was also written through the global CLI logger: %q", cliOutput.String())
	}
}

func TestValidateRendersType1RequirednessNotice(t *testing.T) {
	pdfcpuLog.SetCLILogger(nil)
	conf := model.NewStatelessConfiguration()
	conf.ValidationMode = model.ValidationRelaxed
	inFile := validationNoticeTestFileWithContent(t, type1RequirednessNoticeTestPDF())
	cmd := ValidateCommand([]string{inFile}, conf)
	var noticeOutput bytes.Buffer
	cmd.NoticeOutput = &noticeOutput

	if _, err := validateCommand(t.Context(), cmd); err != nil {
		t.Fatal(err)
	}
	want := "pdfcpu digested: Type1 font Helvetica: missing required entries LastChar, Widths, FontDescriptor " +
		"(obj#:4): dict=type1FontDict required entry=LastChar missing\n"
	if got := noticeOutput.String(); got != want {
		t.Fatalf("notice output: got %q, want %q", got, want)
	}
}

func TestValidateRendersRemoteDestinationNotice(t *testing.T) {
	pdfcpuLog.SetCLILogger(nil)
	conf := model.NewStatelessConfiguration()
	conf.ValidationMode = model.ValidationRelaxed
	inFile := validationNoticeTestFileWithContent(t, remoteDestinationNoticeTestPDF())
	cmd := ValidateCommand([]string{inFile}, conf)
	var noticeOutput bytes.Buffer
	cmd.NoticeOutput = &noticeOutput

	if _, err := validateCommand(t.Context(), cmd); err != nil {
		t.Fatal(err)
	}
	want := "pdfcpu digested: remote destination array[0]: expected non-negative page number, " +
		"got -1 (types.Integer) (obj#:4)\n"
	if got := noticeOutput.String(); got != want {
		t.Fatalf("notice output: got %q, want %q", got, want)
	}
}

func TestValidateRendersAnnotationUnitIntervalNotice(t *testing.T) {
	pdfcpuLog.SetCLILogger(nil)
	conf := model.NewStatelessConfiguration()
	conf.ValidationMode = model.ValidationRelaxed
	inFile := validationNoticeTestFileWithContent(t, annotationUnitIntervalNoticeTestPDF())
	cmd := ValidateCommand([]string{inFile}, conf)
	var noticeOutput bytes.Buffer
	cmd.NoticeOutput = &noticeOutput

	if _, err := validateCommand(t.Context(), cmd); err != nil {
		t.Fatal(err)
	}
	want := "pdfcpu digested: annotDict.C[0]: invalid value 2, expected 0 through 1 (obj#:4)\n"
	if got := noticeOutput.String(); got != want {
		t.Fatalf("notice output: got %q, want %q", got, want)
	}
}

func TestValidateRendersStitchingFunctionNotices(t *testing.T) {
	pdfcpuLog.SetCLILogger(nil)
	conf := model.NewStatelessConfiguration()
	conf.ValidationMode = model.ValidationRelaxed
	inFile := validationNoticeTestFileWithContent(t, stitchingFunctionNoticesTestPDF())
	cmd := ValidateCommand([]string{inFile}, conf)
	var noticeOutput bytes.Buffer
	cmd.NoticeOutput = &noticeOutput

	if _, err := validateCommand(t.Context(), cmd); err != nil {
		t.Fatal(err)
	}
	want := "pdfcpu digested: stitchingFunctionDict.Range: invalid array length 0, expected complete pairs, " +
		"minimum pair count 1 (obj#:6)\n" +
		"pdfcpu digested: stitchingFunctionDict.Bounds: invalid array length 0, expected 1, " +
		"one fewer than Functions (obj#:7)\n"
	if got := noticeOutput.String(); got != want {
		t.Fatalf("notice output: got %q, want %q", got, want)
	}
}

func TestValidateRendersSimpleFontWidthsNotices(t *testing.T) {
	pdfcpuLog.SetCLILogger(nil)
	conf := model.NewStatelessConfiguration()
	conf.ValidationMode = model.ValidationRelaxed
	inFile := validationNoticeTestFileWithContent(t, simpleFontWidthsNoticesTestPDF())
	cmd := ValidateCommand([]string{inFile}, conf)
	var noticeOutput bytes.Buffer
	cmd.NoticeOutput = &noticeOutput

	if _, err := validateCommand(t.Context(), cmd); err != nil {
		t.Fatal(err)
	}
	want := "pdfcpu digested: trueTypeFontDict.Widths: invalid array length 1, expected 2 " +
		"to match FirstChar 0 and LastChar 1 (obj#:4)\n" +
		"pdfcpu digested: trueTypeFontDict.Widths: invalid array length 2, expected 1 " +
		"to match FirstChar 0 and LastChar 0 (obj#:5)\n" +
		"pdfcpu digested: trueTypeFontDict.LastChar: invalid value 0, expected at least FirstChar 65535 (obj#:6)\n"
	if got := noticeOutput.String(); got != want {
		t.Fatalf("notice output: got %q, want %q", got, want)
	}
}

func TestValidateRendersIndexedImageMaskNotice(t *testing.T) {
	pdfcpuLog.SetCLILogger(nil)
	conf := model.NewStatelessConfiguration()
	conf.ValidationMode = model.ValidationRelaxed
	inFile := validationNoticeTestFileWithContent(t, indexedImageMaskNoticeTestPDF())
	cmd := ValidateCommand([]string{inFile}, conf)
	var noticeOutput bytes.Buffer
	cmd.NoticeOutput = &noticeOutput

	if _, err := validateCommand(t.Context(), cmd); err != nil {
		t.Fatal(err)
	}
	want := "pdfcpu digested: imageStreamDict.Mask: invalid array length 6, " +
		"expected two values per colour component (1 components) (obj#:5)\n"
	if got := noticeOutput.String(); got != want {
		t.Fatalf("notice output: got %q, want %q", got, want)
	}
}

func TestValidateRendersStep24aNoticesInCollectionOrder(t *testing.T) {
	pdfcpuLog.SetCLILogger(nil)
	conf := model.NewStatelessConfiguration()
	conf.ValidationMode = model.ValidationRelaxed
	inFile := validationNoticeTestFileWithContent(t, step24aClosureNoticesTestPDF())
	cmd := ValidateCommand([]string{inFile}, conf)
	var noticeOutput bytes.Buffer
	cmd.NoticeOutput = &noticeOutput

	if _, err := validateCommand(t.Context(), cmd); err != nil {
		t.Fatal(err)
	}
	want := "pdfcpu digested: object dictionary contains a non-name key token (obj#:1): " +
		"parse: corrupt dictionary key: parse: corrupt name object\n" +
		"pdfcpu digested: Type1 font Helvetica: missing required entries LastChar, Widths, FontDescriptor (obj#:5): " +
		"dict=type1FontDict required entry=LastChar missing\n" +
		"pdfcpu digested: trueTypeFontDict.Widths: invalid array length 1, expected 2 " +
		"to match FirstChar 0 and LastChar 1 (obj#:6)\n" +
		"pdfcpu digested: imageStreamDict.Mask: invalid array length 6, " +
		"expected two values per colour component (1 components) (obj#:12)\n" +
		"pdfcpu digested: stitchingFunctionDict.Range: invalid array length 0, expected complete pairs, " +
		"minimum pair count 1 (obj#:9)\n" +
		"pdfcpu digested: stitchingFunctionDict.Bounds: invalid array length 0, expected 1, " +
		"one fewer than Functions (obj#:10)\n" +
		"pdfcpu digested: remote destination array[0]: expected non-negative page number, got -1 (types.Integer) (obj#:4)\n" +
		"pdfcpu digested: annotDict.C[0]: invalid value 2, expected 0 through 1 (obj#:13)\n"
	if got := noticeOutput.String(); got != want {
		t.Fatalf("notice output: got %q, want %q", got, want)
	}
}

func TestValidateSuppressesNoticeWithoutNoticeOutput(t *testing.T) {
	pdfcpuLog.SetCLILogger(nil)
	conf := model.NewStatelessConfiguration()
	conf.ValidationMode = model.ValidationRelaxed
	cmd := ValidateCommand([]string{validationNoticeTestFile(t)}, conf)
	var errorOutput bytes.Buffer
	cmd.ErrorOutput = &errorOutput

	if _, err := validateCommand(t.Context(), cmd); err != nil {
		t.Fatal(err)
	}
	if got := errorOutput.String(); got != "" {
		t.Fatalf("suppressed notice wrote to error output: %q", got)
	}
}

func TestValidateNoticeWriterFailurePreservesCause(t *testing.T) {
	wantErr := errors.New("notice writer failed")
	conf := model.NewStatelessConfiguration()
	conf.ValidationMode = model.ValidationRelaxed
	cmd := ValidateCommand([]string{validationNoticeTestFile(t)}, conf)
	cmd.NoticeOutput = validationErrorWriter{err: wantErr}

	_, err := validateCommand(t.Context(), cmd)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "write validation notice 1") {
		t.Fatalf("expected notice write context, got %q", err)
	}
}

func TestReportValidationNoticesPreservesCollectionOrder(t *testing.T) {
	ctx := &model.Context{}
	ctx.AddValidationNotice(model.NewValidationNotice(
		model.NoticePhaseParse,
		model.NoticeSkipped,
		"first divergence",
		nil,
	))
	ctx.AddValidationNotice(model.NewValidationNotice(
		model.NoticePhaseValidate,
		model.NoticeRepaired,
		"second divergence",
		nil,
	))

	var output bytes.Buffer
	if err := reportValidationNotices(&output, ctx.ValidationReport()); err != nil {
		t.Fatal(err)
	}
	want := "pdfcpu skipped: first divergence\npdfcpu repaired: second divergence\n"
	if got := output.String(); got != want {
		t.Fatalf("notice order: got %q, want %q", got, want)
	}
}

func TestValidateSingleValidFile(t *testing.T) {
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")

	if _, err := validateCommand(t.Context(), ValidateCommand([]string{inFile}, nil)); err != nil {
		t.Fatal(err)
	}
}

func TestValidateSingleMissingFilePreservesNotExist(t *testing.T) {
	_, err := validateCommand(t.Context(), ValidateCommand([]string{"missing.pdf"}, nil))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
}

func TestValidateMultipleFilesReturnsJoinedErrorsWithoutReporter(t *testing.T) {
	inFiles := []string{"missing1.pdf", "missing2.pdf"}

	_, err := validateCommand(t.Context(), ValidateCommand(inFiles, nil))
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %v, got %v", os.ErrNotExist, err)
	}
	for _, fn := range inFiles {
		if !bytes.Contains([]byte(err.Error()), []byte(fn)) {
			t.Fatalf("expected %q in error, got %q", fn, err.Error())
		}
	}
}

func TestValidateMultipleFilesStreamsFailuresAndContinues(t *testing.T) {
	validFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	inFiles := []string{"missing1.pdf", validFile, "missing2.pdf"}
	cmd := ValidateCommand(inFiles, nil)
	var errorOutput bytes.Buffer
	cmd.ErrorOutput = &errorOutput

	_, err := validateCommand(t.Context(), cmd)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "validation failed: 2 of 3 files invalid" {
		t.Fatalf("got %q", err.Error())
	}

	got := errorOutput.String()
	first := strings.Index(got, "missing1.pdf")
	second := strings.Index(got, "missing2.pdf")
	if first < 0 || second < 0 {
		t.Fatalf("expected both failures in error output, got %q", got)
	}
	if first >= second {
		t.Fatalf("expected input order in error output, got %q", got)
	}
	for _, fn := range []string{"missing1.pdf", "missing2.pdf"} {
		if strings.Count(got, fn) != 2 {
			t.Fatalf("expected no additional multi-file wrapping for %q, got %q", fn, got)
		}
	}
}

func TestValidateProgressPrecedesEachInput(t *testing.T) {
	validFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	inFiles := []string{"missing.pdf", validFile}
	cmd := ValidateCommand(inFiles, nil)
	var errorOutput bytes.Buffer
	cmd.BoolVal1 = true
	cmd.ErrorOutput = &errorOutput

	_, err := validateCommand(t.Context(), cmd)
	if err == nil {
		t.Fatal("expected error")
	}

	got := errorOutput.String()
	missingProgress := strings.Index(got, "validating(mode=relaxed) missing.pdf ...")
	missingFailure := strings.Index(got, "validate: open missing.pdf")
	validProgress := strings.Index(got, "validating(mode=relaxed) "+validFile+" ...")
	if missingProgress < 0 || missingFailure < 0 || validProgress < 0 {
		t.Fatalf("expected progress and failure output, got %q", got)
	}
	if missingProgress >= missingFailure || missingFailure >= validProgress {
		t.Fatalf("expected progress and failure in input order, got %q", got)
	}
}

func TestValidateProgressLabelsStandardInput(t *testing.T) {
	withStdinFile(t, "not a pdf")
	cmd := ValidateCommand([]string{"-"}, nil)
	var errorOutput bytes.Buffer
	cmd.BoolVal1 = true
	cmd.ErrorOutput = &errorOutput

	if _, err := validateCommand(t.Context(), cmd); err == nil {
		t.Fatal("expected error")
	}
	if got, want := errorOutput.String(), "validating(mode=relaxed) stdin ...\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestValidateProgressWriterFailurePreservesCause(t *testing.T) {
	wantErr := errors.New("progress writer failed")
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	cmd := ValidateCommand([]string{inFile}, nil)
	cmd.BoolVal1 = true
	cmd.ErrorOutput = validationErrorWriter{err: wantErr}

	_, err := validateCommand(t.Context(), cmd)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "write validation progress") {
		t.Fatalf("expected progress write context, got %q", err.Error())
	}
}

func TestDumpMissingFilePreservesNotExist(t *testing.T) {
	cmd := DumpCommand("missing.pdf", []int{0, 0}, nil)

	_, err := dump(t.Context(), cmd)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist error, got %v", err)
	}
}

func TestInfoReportsRelaxedValidationWithoutChangingJSON(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationStrict
	var notice bytes.Buffer
	input := extractTestPDF(t)
	cmd := InfoCommand([]string{input, input}, nil, false, true, conf)
	cmd.ErrorOutput = &notice

	out, err := listInfoCommand(t.Context(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := notice.String(), "info: using relaxed validation\n"; got != want {
		t.Fatalf("validation notice: got %q, want %q", got, want)
	}
	if len(out) != 1 || !json.Valid([]byte(out[0])) {
		t.Fatalf("expected pure JSON output, got %q", out)
	}
	if conf.ValidationMode != model.ValidationStrict {
		t.Fatalf("caller validation mode: got %d, want %d", conf.ValidationMode, model.ValidationStrict)
	}
}

func TestDumpReportsRelaxedValidationWithoutChangingConfiguration(t *testing.T) {
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationStrict
	var notice bytes.Buffer
	cmd := DumpCommand("missing.pdf", []int{0, 0}, conf)
	cmd.ErrorOutput = &notice

	if _, err := dump(t.Context(), cmd); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist error, got %v", err)
	}
	if got, want := notice.String(), "dump: using relaxed validation\n"; got != want {
		t.Fatalf("validation notice: got %q, want %q", got, want)
	}
	if conf.ValidationMode != model.ValidationStrict {
		t.Fatalf("caller validation mode: got %d, want %d", conf.ValidationMode, model.ValidationStrict)
	}
}

func TestOptimizeCLIPlumbingErrorsIncludePhaseContext(t *testing.T) {
	tests := []struct {
		name    string
		fn      func(t *testing.T) error
		wantErr error
		want    string
	}{
		{
			name: "read stdin",
			fn: func(t *testing.T) error {
				withStdinFile(t, "")
				_, err := optimize(t.Context(), OptimizeCommand("-", "-", nil))
				return err
			},
			want: "optimize: read stdin",
		},
		{
			name: "open input",
			fn: func(t *testing.T) error {
				_, err := optimize(t.Context(), OptimizeCommand("missing.pdf", "-", nil))
				return err
			},
			wantErr: os.ErrNotExist,
			want:    "optimize: open input missing.pdf",
		},
		{
			name: "create output",
			fn: func(t *testing.T) error {
				withStdinFile(t, "not a pdf")
				outFile := filepath.Join(t.TempDir(), "missing-dir", "out.pdf")
				_, err := optimize(t.Context(), OptimizeCommand("-", outFile, nil))
				return err
			},
			wantErr: os.ErrNotExist,
			want:    "optimize: create output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(t)
			if err == nil {
				t.Fatal("expected error")
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got %q", tt.want, err.Error())
			}
		})
	}
}

func TestTrimStreamSetupErrorsIncludePhaseContext(t *testing.T) {
	withStdinFile(t, "")

	_, err := trim(t.Context(), TrimCommand("-", "-", []string{"1"}, nil))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "trim: prepare input/output") {
		t.Fatalf("expected trim input/output context, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "stdin is empty") {
		t.Fatalf("expected stdin setup error, got %q", err.Error())
	}
}

func withStdinFile(t *testing.T, content string) {
	t.Helper()

	stdin := os.Stdin
	f, err := os.CreateTemp(t.TempDir(), "stdin-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Stdin = stdin
		_ = f.Close()
	})
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	os.Stdin = f
}

func TestMergeCreateRawRemovesOutputOnFailure(t *testing.T) {
	withStdinFile(t, "not a pdf")

	outFile := filepath.Join(t.TempDir(), "out.pdf")
	inFile := filepath.Join("..", "samples", "create", "primitives", "textAndAlignment.pdf")
	cmd := MergeCreateCommand([]string{"-", inFile}, outFile, false, nil)

	_, err := mergeCreate(t.Context(), cmd)
	if err == nil {
		t.Fatal("expected merge failure")
	}
	if _, statErr := os.Stat(outFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected output cleanup, got stat error %v", statErr)
	}
}

// TestListInfoFileJSONClosesEachInputBeforeNext verifies multi-file processing
// never accumulates deferred file descriptors.
func TestListInfoFileJSONClosesEachInputBeforeNext(t *testing.T) {
	const fileCount = 128
	dir := t.TempDir()
	files := make([]string, fileCount)
	for i := range files {
		files[i] = filepath.Join(dir, fmt.Sprintf("input-%03d.pdf", i))
		if err := os.WriteFile(files[i], nil, 0600); err != nil {
			t.Fatal(err)
		}
	}

	var previous *os.File
	process := func(
		c context.Context,
		rs io.ReadSeeker,
		_ string,
		_ []string,
		_ bool,
		_ *model.Configuration,
	) (*pdfcpu.PDFInfo, error) {
		if previous != nil {
			if got, want := previous.Fd(), ^uintptr(0); got != want {
				t.Fatalf("previous input descriptor: got %d, want %d", got, want)
			}
		}
		previous = rs.(*os.File)
		return &pdfcpu.PDFInfo{}, nil
	}

	for _, fileName := range files {
		if _, err := listInfoFileJSON(t.Context(), fileName, nil, false, nil, process); err != nil {
			t.Fatal(err)
		}
	}
	if got, want := previous.Fd(), ^uintptr(0); got != want {
		t.Fatalf("last input descriptor: got %d, want %d", got, want)
	}
}

// TestListInfoFileJSONJoinsProcessAndCloseFailures verifies neither lifecycle
// failure is lost.
func TestListInfoFileJSONJoinsProcessAndCloseFailures(t *testing.T) {
	fileName := filepath.Join(t.TempDir(), "input.pdf")
	if err := os.WriteFile(fileName, nil, 0600); err != nil {
		t.Fatal(err)
	}
	processErr := errors.New("process input")
	process := func(
		c context.Context,
		rs io.ReadSeeker,
		_ string,
		_ []string,
		_ bool,
		_ *model.Configuration,
	) (*pdfcpu.PDFInfo, error) {
		if err := rs.(*os.File).Close(); err != nil {
			t.Fatal(err)
		}
		return nil, processErr
	}

	_, err := listInfoFileJSON(t.Context(), fileName, nil, false, nil, process)
	if !errors.Is(err, processErr) || !errors.Is(err, os.ErrClosed) {
		t.Fatalf("expected process and close failures, got %v", err)
	}
}
