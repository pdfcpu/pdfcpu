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
	"crypto/x509"
	"encoding/asn1"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/pkcs7"
)

type timestampKind uint8

const (
	timestampKindSignature timestampKind = iota
	timestampKindDocument
)

type timestampTokenInfo struct {
	Version                 int
	Policy                  asn1.ObjectIdentifier
	MessageImprintAlgorithm asn1.ObjectIdentifier
	MessageImprint          []byte
	SerialNumber            asn1.RawValue
	GeneratedAt             time.Time
	Accuracy                asn1.RawValue
	Ordering                bool
	Nonce                   asn1.RawValue
	TSA                     asn1.RawValue
	Extensions              asn1.RawValue
}

type timestampEvidence struct {
	Kind                  timestampKind
	SigningTime           time.Time
	Present               bool
	AssessmentScope       model.AssessmentScope
	RawToken              []byte
	CMS                   *pkcs7.PKCS7
	SourceSigner          pkcs7.SignerInfo
	TokenInfo             *timestampTokenInfo
	TokenInfoErr          error
	SignedData            []byte
	DigestVerified        bool
	SignatureVerified     bool
	CorrectProfile        bool
	LocalTSAPathValidated bool // configured-local TSA certificate path validation succeeded
	Err                   error
}

// isCryptographicallyAuthenticatedDocumentTimestampEvidence reports whether
// the RFC 3161 token and its binding to the signed document were authenticated.
func isCryptographicallyAuthenticatedDocumentTimestampEvidence(evidence timestampEvidence) bool {
	return evidence.Kind == timestampKindDocument &&
		evidence.Present &&
		!evidence.SigningTime.IsZero() &&
		evidence.Err == nil &&
		evidence.DigestVerified &&
		evidence.SignatureVerified &&
		evidence.CorrectProfile
}

// isLocallyValidatedDocumentTimestampEvidence additionally requires successful
// TSA certificate-path validation using the configured local sources.
func isLocallyValidatedDocumentTimestampEvidence(evidence timestampEvidence) bool {
	return isCryptographicallyAuthenticatedDocumentTimestampEvidence(evidence) &&
		evidence.LocalTSAPathValidated
}

type certificateAssessment struct {
	Certificate           *model.CertificateDetails
	CertificatePathStatus int
	Problems              []string
	Reason                model.SignatureReason
}

type localSignatureAssessment struct {
	SignersProcessed       int
	SignatureAuthenticated bool
	DigestVerified         bool
	ProfileValidated       bool
	CertificateIdentified  bool
	PathValidated          bool
	RevocationGood         bool
}

func (a localSignatureAssessment) complete() bool {
	return a.SignatureAuthenticated &&
		a.DigestVerified &&
		a.ProfileValidated &&
		a.CertificateIdentified &&
		a.PathValidated &&
		a.RevocationGood
}

func (a *localSignatureAssessment) applyCertificateAssessment(assessment certificateAssessment) {
	a.CertificateIdentified = assessment.Certificate != nil
	a.PathValidated = assessment.CertificatePathStatus == model.True
	a.RevocationGood = assessment.Certificate != nil &&
		assessment.Certificate.Revocation.Status == model.True
}

func (a *localSignatureAssessment) merge(other localSignatureAssessment) {
	if other.SignersProcessed == 0 {
		return
	}
	if a.SignersProcessed == 0 {
		*a = other
		return
	}
	a.SignersProcessed += other.SignersProcessed
	a.SignatureAuthenticated = a.SignatureAuthenticated && other.SignatureAuthenticated
	a.DigestVerified = a.DigestVerified && other.DigestVerified
	a.ProfileValidated = a.ProfileValidated && other.ProfileValidated
	a.CertificateIdentified = a.CertificateIdentified && other.CertificateIdentified
	a.PathValidated = a.PathValidated && other.PathValidated
	a.RevocationGood = a.RevocationGood && other.RevocationGood
}

func completedLocalSignatureAssessment() localSignatureAssessment {
	return localSignatureAssessment{
		SignersProcessed:       1,
		SignatureAuthenticated: true,
		DigestVerified:         true,
		ProfileValidated:       true,
		CertificateIdentified:  true,
		PathValidated:          true,
		RevocationGood:         true,
	}
}

type dssEvidence struct {
	Certificates  []*x509.Certificate
	CRLs          [][]byte
	OCSPResponses [][]byte
	Supported     bool
}
