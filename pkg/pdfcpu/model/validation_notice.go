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

import "errors"

// NoticeSeverity identifies the severity of an accepted validation divergence.
type NoticeSeverity string

const (
	// NoticeSeverityWarning identifies an accepted validation divergence.
	NoticeSeverityWarning NoticeSeverity = "warning"
)

// NoticeDisposition identifies how a validation divergence was handled.
type NoticeDisposition string

const (
	// NoticeDigested identifies an accepted specification violation.
	NoticeDigested NoticeDisposition = "digested"

	// NoticeRepaired identifies an accepted and repaired specification violation.
	NoticeRepaired NoticeDisposition = "repaired"

	// NoticeSkipped identifies an accepted and skipped specification violation.
	NoticeSkipped NoticeDisposition = "skipped"
)

// NoticePhase identifies the processing phase that accepted a validation divergence.
type NoticePhase string

const (
	// NoticePhaseParse identifies a parser notice.
	NoticePhaseParse NoticePhase = "parse"

	// NoticePhaseRead identifies a reader notice.
	NoticePhaseRead NoticePhase = "read"

	// NoticePhaseValidate identifies a semantic validation notice.
	NoticePhaseValidate NoticePhase = "validate"
)

// ValidationNotice describes an accepted validation divergence.
type ValidationNotice struct {
	Severity        NoticeSeverity
	Disposition     NoticeDisposition
	Phase           NoticePhase
	Message         string
	Cause           error
	ObjectNumber    int
	PageNumber      int
	Path            string
	ActualVersion   Version
	RequiredVersion Version
	HasVersions     bool
}

// NewValidationNotice returns a warning for an accepted validation divergence.
func NewValidationNotice(phase NoticePhase, disposition NoticeDisposition, message string, cause error) ValidationNotice {
	return ValidationNotice{
		Severity:     NoticeSeverityWarning,
		Disposition:  disposition,
		Phase:        phase,
		Message:      message,
		Cause:        cause,
		ObjectNumber: validationNoticeObjectNumber(cause),
	}
}

func validationNoticeObjectNumber(err error) int {
	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		return validationErr.ObjectNumber()
	}
	return 0
}

// ValidationReport contains the accepted validation divergences for one PDF context.
type ValidationReport struct {
	notices []ValidationNotice
}

// Empty reports whether the validation report contains no notices.
func (r ValidationReport) Empty() bool {
	return len(r.notices) == 0
}

// Notices returns the validation notices in collection order.
func (r ValidationReport) Notices() []ValidationNotice {
	return append([]ValidationNotice(nil), r.notices...)
}

// AddValidationNotice adds notice to this PDF context's validation report.
func (ctx *Context) AddValidationNotice(notice ValidationNotice) {
	if ctx == nil {
		return
	}
	ctx.validationReport.notices = append(ctx.validationReport.notices, notice)
}

// AddValidationNotice adds notice to the report owned by this cross-reference table's PDF context.
func (xRefTable *XRefTable) AddValidationNotice(notice ValidationNotice) {
	if xRefTable == nil || xRefTable.validationReport == nil {
		return
	}
	xRefTable.validationReport.notices = append(xRefTable.validationReport.notices, notice)
}

// ValidationReport returns a snapshot of this PDF context's validation report.
func (ctx *Context) ValidationReport() ValidationReport {
	if ctx == nil {
		return ValidationReport{}
	}
	return ValidationReport{notices: ctx.validationReport.Notices()}
}
