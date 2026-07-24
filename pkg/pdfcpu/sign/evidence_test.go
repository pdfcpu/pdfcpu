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

package sign

import (
	"errors"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// TestAssessmentScopeZeroValueIsLocal locks down backwards-compatible local
// assessment for zero-value timestamp evidence.
func TestAssessmentScopeZeroValueIsLocal(t *testing.T) {
	var evidence timestampEvidence
	if evidence.AssessmentScope != model.AssessmentScopeLocal {
		t.Fatalf(
			"got zero-value assessment scope %d, want local scope %d",
			evidence.AssessmentScope,
			model.AssessmentScopeLocal,
		)
	}
}

// TestDocumentTimestampAuthenticationAndLocalValidationPredicates verifies
// cryptographic authentication remains distinct from local TSA path validation.
func TestDocumentTimestampAuthenticationAndLocalValidationPredicates(t *testing.T) {
	authenticated := timestampEvidence{
		Kind:              timestampKindDocument,
		SigningTime:       time.Now(),
		Present:           true,
		DigestVerified:    true,
		SignatureVerified: true,
		CorrectProfile:    true,
	}
	if !isCryptographicallyAuthenticatedDocumentTimestampEvidence(authenticated) {
		t.Fatalf("complete document timestamp evidence was not cryptographically authenticated: %+v", authenticated)
	}
	if isLocallyValidatedDocumentTimestampEvidence(authenticated) {
		t.Fatalf("cryptographic authentication implied local path validation: %+v", authenticated)
	}

	tests := []struct {
		name   string
		mutate func(*timestampEvidence)
	}{
		{"NotObserved", func(e *timestampEvidence) { e.Present = false }},
		{"MissingTime", func(e *timestampEvidence) { e.SigningTime = time.Time{} }},
		{"DigestNotVerified", func(e *timestampEvidence) { e.DigestVerified = false }},
		{"SignatureNotVerified", func(e *timestampEvidence) { e.SignatureVerified = false }},
		{"WrongProfile", func(e *timestampEvidence) { e.CorrectProfile = false }},
		{"EvidenceError", func(e *timestampEvidence) { e.Err = errors.New("timestamp evidence") }},
		{"EmbeddedTimestamp", func(e *timestampEvidence) { e.Kind = timestampKindSignature }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evidence := authenticated
			tt.mutate(&evidence)
			if isCryptographicallyAuthenticatedDocumentTimestampEvidence(evidence) ||
				isLocallyValidatedDocumentTimestampEvidence(evidence) {
				t.Fatalf("incomplete evidence crossed timestamp predicates: %+v", evidence)
			}
		})
	}

	authenticated.LocalTSAPathValidated = true
	if !isLocallyValidatedDocumentTimestampEvidence(authenticated) {
		t.Fatalf("authenticated evidence with a validated local TSA path was not locally validated: %+v", authenticated)
	}
}

// TestOnlyLocallyValidatedDocumentTimestampPopulatesContext verifies local TSA
// path validation gates ctx.DTS and the timestamp result.
func TestOnlyLocallyValidatedDocumentTimestampPopulatesContext(t *testing.T) {
	signingTime := time.Now()
	locallyValidated := timestampEvidence{
		Kind:                  timestampKindDocument,
		SigningTime:           signingTime,
		Present:               true,
		DigestVerified:        true,
		SignatureVerified:     true,
		CorrectProfile:        true,
		LocalTSAPathValidated: true,
	}

	result := unknownSignatureResult()
	ctx := &model.Context{XRefTable: &model.XRefTable{}}
	cryptographicallyAuthenticated := locallyValidated
	cryptographicallyAuthenticated.LocalTSAPathValidated = false
	finalizeDTSValidationResult(
		result,
		ctx,
		cryptographicallyAuthenticated,
		completedLocalSignatureAssessment(),
	)
	if !ctx.DTS.IsZero() ||
		result.Status != model.SignatureStatusUnknown ||
		result.Reason != model.SignatureReasonUnknown {
		t.Fatalf("authentication without local validation produced DTS conclusion: ctx=%+v result=%+v", ctx, result)
	}

	finalizeDTSValidationResult(result, ctx, locallyValidated, completedLocalSignatureAssessment())
	if !ctx.DTS.Equal(signingTime) ||
		result.Status != model.SignatureStatusValid ||
		result.Reason != model.SignatureReasonDocNotModified {
		t.Fatalf("locally validated evidence was not applied: ctx=%+v result=%+v", ctx, result)
	}
}

// TestTimestampKindZeroValueIsSignatureTimestamp locks down zero-value request
// routing without relying on evidence presence.
func TestTimestampKindZeroValueIsSignatureTimestamp(t *testing.T) {
	var evidence timestampEvidence
	if evidence.Kind != timestampKindSignature {
		t.Fatalf("got zero-value evidence kind %d, want signature timestamp", evidence.Kind)
	}
}
