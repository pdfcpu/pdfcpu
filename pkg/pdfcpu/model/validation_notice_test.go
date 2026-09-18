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
	"bytes"
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestXRefTableAddsNoticeToOwningContext(t *testing.T) {
	ctx, err := NewContext(bytes.NewReader(nil), NewStatelessConfiguration())
	if err != nil {
		t.Fatal(err)
	}
	ctx.XRefTable.AddValidationNotice(NewValidationNotice(NoticePhaseValidate, NoticeDigested, "semantic", nil))

	notices := ctx.ValidationReport().Notices()
	if len(notices) != 1 || notices[0].Message != "semantic" {
		t.Fatalf("context report: got %+v", notices)
	}
}

func TestValidationReportPreservesNoticeOrder(t *testing.T) {
	ctx := &Context{}
	ctx.AddValidationNotice(NewValidationNotice(NoticePhaseParse, NoticeDigested, "first", nil))
	ctx.AddValidationNotice(NewValidationNotice(NoticePhaseRead, NoticeRepaired, "second", nil))

	report := ctx.ValidationReport()
	ctx.AddValidationNotice(NewValidationNotice(NoticePhaseValidate, NoticeSkipped, "third", nil))

	notices := report.Notices()
	if len(notices) != 2 {
		t.Fatalf("notice count: got %d, want 2", len(notices))
	}
	if notices[0].Message != "first" || notices[1].Message != "second" {
		t.Fatalf("notice order: got %q, %q", notices[0].Message, notices[1].Message)
	}

	notices[0].Message = "changed"
	if got := report.Notices()[0].Message; got != "first" {
		t.Fatalf("report snapshot changed through returned slice: got %q, want first", got)
	}
}

func TestNewValidationNoticeExtractsObjectAttribution(t *testing.T) {
	cause := fmt.Errorf("parse object: %w", WithValidationErrorObject(errors.New("corrupt name object"), 17))
	notice := NewValidationNotice(NoticePhaseParse, NoticeDigested, "relaxed dictionary parse", cause)
	if notice.Severity != NoticeSeverityWarning || notice.Phase != NoticePhaseParse || notice.Disposition != NoticeDigested {
		t.Fatalf("classification: got %q/%q/%q", notice.Severity, notice.Phase, notice.Disposition)
	}
	if notice.ObjectNumber != 17 {
		t.Fatalf("object number: got %d, want 17", notice.ObjectNumber)
	}
	if !errors.Is(notice.Cause, cause) {
		t.Fatalf("cause: got %v, want %v", notice.Cause, cause)
	}

	notice = NewValidationNotice(NoticePhaseValidate, NoticeDigested, "unknown object", errors.New("invalid value"))
	if notice.ObjectNumber != 0 {
		t.Fatalf("unknown object number: got %d, want 0", notice.ObjectNumber)
	}
}

func TestValidationReportsIsolateConcurrentContexts(t *testing.T) {
	const contextCount = 16

	reports := make([]ValidationReport, contextCount)
	var wg sync.WaitGroup
	for i := range contextCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := &Context{}
			message := fmt.Sprintf("context %d", i)
			ctx.AddValidationNotice(NewValidationNotice(NoticePhaseValidate, NoticeDigested, message, nil))
			reports[i] = ctx.ValidationReport()
		}()
	}
	wg.Wait()

	for i, report := range reports {
		notices := report.Notices()
		if len(notices) != 1 {
			t.Fatalf("context %d notice count: got %d, want 1", i, len(notices))
		}
		want := fmt.Sprintf("context %d", i)
		if notices[0].Message != want {
			t.Fatalf("context %d notice: got %q, want %q", i, notices[0].Message, want)
		}
	}
}

func TestEmptyValidationReport(t *testing.T) {
	if !((&Context{}).ValidationReport().Empty()) {
		t.Fatal("new context report is not empty")
	}
}
