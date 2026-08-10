/*
Copyright 2025 The pdfcpu Authors.

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
	"fmt"
	"strings"
	"time"
)

const (
	Unknown = iota
	False   // aka invalid, not ok
	True    // aka  valid, ok
)

// Preferred cert revocation checking mechanism values
const (
	CRL = iota
	OCSP
)

const (
	CertifiedSigPermNone = iota
	CertifiedSigPermNoChangesAllowed
	CertifiedSigPermFillingAndSigningOK
	CertifiedSigPermFillingAnnotatingAndSigningOK
)

const (
	SigTypeForm = iota
	SigTypePage
	SigTypeUR
	SigTypeDTS
)

// SignTSFormat is the timestamp layout used in signature validation output.
const SignTSFormat = "2006-01-02 15:04:05 -0700"

const signatureOutputMaxWidth = 120

// RevocationDetails contains observed CRL and OCSP evidence together with a
// local revocation assessment. Its Status and Reason fields are compatibility
// representations, not legal, regulatory, enterprise-policy or policy-based
// trust decisions.
type RevocationDetails struct {
	Status int
	Reason string
	CRL    *CRLEvidence
	CRLs   []*CRLEvidence
	OCSP   *OCSPEvidence
	OCSPs  []*OCSPEvidence
}

// RevocationEvidenceSource identifies where revocation evidence originated.
type RevocationEvidenceSource uint8

const (
	// RevocationEvidenceSourceUnspecified identifies unavailable provenance.
	RevocationEvidenceSourceUnspecified RevocationEvidenceSource = iota

	// RevocationEvidenceSourceArchived identifies evidence embedded in the signed document.
	RevocationEvidenceSourceArchived

	// RevocationEvidenceSourceOnline identifies evidence obtained from a configured endpoint.
	RevocationEvidenceSourceOnline
)

// CRLRevocationEntry records an observed CRL entry without implying authenticity.
type CRLRevocationEntry struct {
	SerialNumber   string
	RevocationTime time.Time
	ReasonCode     int
}

// CRLEvidence records observed CRL material and separate issuer, signature and
// applicability checks. Archived evidence may remain in an unknown state.
type CRLEvidence struct {
	AssessmentScope AssessmentScope
	Source          RevocationEvidenceSource
	Index           int
	Location        string
	Error           string
	IssuerMatched   int
	SignatureValid  int
	Applicable      int
	Entries         []CRLRevocationEntry
}

// OCSPResponder identifies the certificate authenticating an OCSP response.
type OCSPResponder uint8

const (
	// OCSPResponderUnspecified identifies unavailable responder evidence.
	OCSPResponderUnspecified OCSPResponder = iota

	// OCSPResponderIssuer identifies a response authenticated directly by the issuer.
	OCSPResponderIssuer

	// OCSPResponderDelegated identifies an authorized delegated responder.
	OCSPResponderDelegated
)

// OCSPEvidence records observed OCSP provenance and separate response-signature,
// responder-certificate, authentication and applicability checks. Archived
// evidence may remain in an unknown state.
type OCSPEvidence struct {
	AssessmentScope                      AssessmentScope
	Source                               RevocationEvidenceSource
	Index                                int
	Location                             string
	Error                                string
	ProducedAt                           time.Time
	ThisUpdate                           time.Time
	NextUpdate                           time.Time
	RevokedAt                            time.Time
	Applicable                           int
	Responder                            OCSPResponder
	Authenticated                        int
	ResponseSignatureValid               int
	ResponderCertificateIssuedByIssuer   int
	ResponderCertificateOCSPSigningValid int
	CertificateStatus                    int
	ResponderRevocation                  int
}

// String returns the string value of rd.
func (rd RevocationDetails) String() string {
	ss := []string{}
	ss = append(ss, fmt.Sprintf(" Local:  %s", validString(rd.Status)))
	if len(rd.Reason) > 0 {
		ss = appendWrappedSignatureText(ss, "                                         Reason: ", rd.Reason)
	}
	return strings.Join(ss, "\n")
}

func appendWrappedSignatureText(ss []string, prefix, text string) []string {
	continuation := strings.Repeat(" ", len(prefix))
	first := true
	for _, paragraph := range strings.Split(text, "\n") {
		linePrefix := continuation
		if first {
			linePrefix = prefix
			first = false
		}
		lines := wrapSignatureText(paragraph, signatureOutputMaxWidth-len(linePrefix))
		if len(lines) == 0 {
			ss = append(ss, linePrefix)
			continue
		}
		for _, line := range lines {
			ss = append(ss, linePrefix+line)
			linePrefix = continuation
		}
	}
	return ss
}

func wrapSignatureText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	var lines []string
	var line []rune
	for _, word := range strings.Fields(text) {
		runes := []rune(word)
		if len(line) > 0 && len(line)+1+len(runes) > width {
			lines = append(lines, string(line))
			line = nil
		}
		for len(runes) > width {
			head, tail := splitSignatureWord(runes, width)
			lines = append(lines, string(head))
			runes = tail
		}
		if len(runes) == 0 {
			continue
		}
		if len(line) > 0 {
			line = append(line, ' ')
		}
		line = append(line, runes...)
	}
	if len(line) > 0 {
		lines = append(lines, string(line))
	}
	return lines
}

func splitSignatureWord(word []rune, width int) ([]rune, []rune) {
	split := width
	for i := width; i > 0; i-- {
		if strings.ContainsRune("/:?&=;", word[i-1]) {
			split = i
			break
		}
	}
	return word[:split], word[split:]
}

// TrustDetails is the legacy compatibility representation of the
// local certificate-path assessment. It is not a legal, regulatory,
// enterprise-policy or policy-based trust decision.
type TrustDetails struct {
	Status                                int
	Reason                                string
	SourceObtainedFrom                    string
	AllowSignDocuments                    bool
	AllowCertifyDocuments                 bool
	AllowExecuteDynamicContent            bool
	AllowExecuteJavaScript                bool
	AllowExecutePrivilegedSystemOperation bool
}

// AssessmentScope identifies the scope used to assess observed signature,
// certificate, timestamp and revocation evidence.
type AssessmentScope uint8

const (
	// AssessmentScopeLocal represents an assessment using the configured local
	// certificate and revocation sources. It does not imply a policy-based trust
	// decision.
	AssessmentScopeLocal AssessmentScope = iota
)

// ValidationTimeSource identifies where a certificate-validation time originated.
type ValidationTimeSource uint8

const (
	// ValidationTimeSourceUnspecified identifies wall-clock or unavailable provenance.
	ValidationTimeSourceUnspecified ValidationTimeSource = iota

	// ValidationTimeSourceClaimedSigningTime is retained for compatibility.
	// Claimed CMS signing time is display-only and does not select the local
	// certificate-assessment time.
	ValidationTimeSourceClaimedSigningTime

	// ValidationTimeSourceSignatureTimestamp identifies an embedded signature timestamp.
	ValidationTimeSourceSignatureTimestamp

	// ValidationTimeSourceDocumentTimestamp identifies an ETSI.RFC3161 document timestamp.
	ValidationTimeSourceDocumentTimestamp
)

// ValidationTimeEvidence records an observed candidate validation time and its
// provenance. Its presence does not mean the local assessment used that time
// or that a policy-based trust decision accepted it.
type ValidationTimeEvidence struct {
	Time            time.Time
	Source          ValidationTimeSource
	AssessmentScope AssessmentScope
}

// CertificatePathMethod identifies how a certificate-path conclusion was reached.
type CertificatePathMethod uint8

const (
	// CertificatePathMethodUnspecified identifies an unavailable path-assessment method.
	CertificatePathMethodUnspecified CertificatePathMethod = iota

	// CertificatePathMethodLocalTrustStore identifies local X.509 trust-store resolution.
	CertificatePathMethodLocalTrustStore

	// CertificatePathMethodSelfSignature identifies legacy self-signature
	// inspection evidence. Self-signature alone does not resolve a path.
	CertificatePathMethodSelfSignature

	// CertificatePathMethodCertificateAuthority identifies legacy CA evidence.
	// IsCA alone does not resolve a path.
	CertificatePathMethodCertificateAuthority

	// CertificatePathMethodValidity identifies a certificate-validity conclusion.
	CertificatePathMethodValidity

	// CertificatePathMethodMissingCertificate identifies an incomplete certificate path.
	CertificatePathMethodMissingCertificate

	// CertificatePathMethodPublicKey identifies a public-key inspection failure.
	CertificatePathMethodPublicKey
)

// CertificatePathEvidence records local certificate-path assessment evidence.
// Status is True only when local X.509 path verification succeeds. Certificate
// authority and self-signature observations are recorded independently on
// CertificateDetails.
type CertificatePathEvidence struct {
	AssessmentScope AssessmentScope
	Method          CertificatePathMethod
	Status          int
	Reason          string
}

// String returns the string value of td.
func (td TrustDetails) String() string {
	ss := []string{}
	ss = append(ss, fmt.Sprintf(" Status: %s", validString(td.Status)))
	if len(td.Reason) > 0 {
		ss = appendWrappedSignatureText(ss, "                                         Reason: ", td.Reason)
	}
	// if td.Status == True {
	// 	ss = append(ss, fmt.Sprintf("                                         SourceObtainedFrom:                    %s", td.SourceObtainedFrom))
	// 	ss = append(ss, fmt.Sprintf("                                         AllowSignDocuments:                    %t", td.AllowSignDocuments))
	// 	ss = append(ss, fmt.Sprintf("                                         AllowCertifyDocuments:                 %t", td.AllowCertifyDocuments))
	// 	ss = append(ss, fmt.Sprintf("                                         AllowExecuteDynamicContent:            %t", td.AllowExecuteDynamicContent))
	// 	ss = append(ss, fmt.Sprintf("                                         AllowExecuteJavaScript:                %t", td.AllowExecuteJavaScript))
	// 	ss = append(ss, fmt.Sprintf("                                         AllowExecutePrivilegedSystemOperation: %t", td.AllowExecutePrivilegedSystemOperation))
	// }
	return strings.Join(ss, "\n")
}

// CertificateDetails contains observed certificate, path, validation-time and
// revocation evidence together with a local assessment. It is not a legal,
// regulatory, enterprise-policy or policy-based trust decision.
type CertificateDetails struct {
	Leaf         bool
	SelfSigned   bool
	Subject      string
	Issuer       string
	SerialNumber string
	ValidFrom    time.Time
	ValidThru    time.Time
	Expired      bool
	// Qualified records recognized certificate-policy evidence, not a legal or regulatory conclusion.
	Qualified         bool
	CA                bool
	Usage             string
	Version           int
	SignAlg           string
	KeySize           int
	Revocation        RevocationDetails
	Trust             TrustDetails
	PathEvidence      CertificatePathEvidence
	ValidationTime    ValidationTimeEvidence
	IssuerCertificate *CertificateDetails
}

// String returns the string value of cd.
func (cd CertificateDetails) String() string {
	return cd.string()
}

func (cd CertificateDetails) string() string {
	ss := []string{}
	ss = append(ss, fmt.Sprintf("                             Subject:    %s", cd.Subject))
	ss = append(ss, fmt.Sprintf("                             Issuer:     %s", cd.Issuer))
	ss = append(ss, fmt.Sprintf("                             SerialNr:   %s", cd.SerialNumber))
	ss = append(ss, fmt.Sprintf("                             Valid From: %s", cd.ValidFrom.Format(SignTSFormat)))
	ss = append(ss, fmt.Sprintf("                             Valid Thru: %s", cd.ValidThru.Format(SignTSFormat)))
	ss = append(ss, fmt.Sprintf("                             Expired:    %t", cd.Expired))
	ss = append(ss, fmt.Sprintf("                             QC Policy:  %t", cd.Qualified))
	ss = append(ss, fmt.Sprintf("                             CA:         %t", cd.CA))
	ss = append(ss, fmt.Sprintf("                             Usage:      %s", cd.Usage))
	ss = append(ss, fmt.Sprintf("                             Version:    %d", cd.Version))
	ss = append(ss, fmt.Sprintf("                             SignAlg:    %s", cd.SignAlg))
	ss = append(ss, fmt.Sprintf("                             Key Size:   %d bits", cd.KeySize))
	ss = append(ss, fmt.Sprintf("                             SelfSigned: %t", cd.SelfSigned))
	ss = append(ss, fmt.Sprintf("                             Local Path:%s", cd.Trust))
	if cd.Leaf && !cd.SelfSigned {
		ss = append(ss, fmt.Sprintf("                             Revocation:%s", cd.Revocation))
	}

	if cd.IssuerCertificate != nil {
		s := "             Intermediate"
		if cd.IssuerCertificate.IssuerCertificate == nil {
			s = "             Root"
		}
		if cd.IssuerCertificate.CA {
			s += "CA"
		}
		ss = append(ss, s+":")
		ss = append(ss, cd.IssuerCertificate.string())
	}
	return strings.Join(ss, "\n")
}

// Signature represents a digital signature.
type Signature struct {
	Type          int
	Certified     bool
	Authoritative bool
	Visible       bool
	Signed        bool
	ObjNr         int
	PageNr        int
}

// String returns a string representation.
func (sig Signature) String(status SignatureStatus) string {
	s := ""
	if sig.Type == SigTypeForm {
		s = "form signature ("
	} else if sig.Type == SigTypePage {
		s = "page signature ("
	} else if sig.Type == SigTypeUR {
		s = "usage rights signature ("
	} else {
		s = "document timestamp ("
	}

	if sig.Type != SigTypeDTS {
		if sig.Certified {
			s += "certified, "
		} else if sig.Authoritative {
			s += "authoritative, "
		}
	}

	if sig.Type == SigTypeDTS {
		s1 := "locally validated, "
		if status != SignatureStatusValid {
			s1 = "not " + s1
		}
		s += s1
	}

	if sig.Visible {
		s += "visible, "
	} else {
		s += "invisible, "
	}

	if sig.Signed {
		s += "signed)"
	} else {
		s += "unsigned)"
	}

	if sig.Visible {
		s += fmt.Sprintf(" on page %d", sig.PageNr)
	}

	//s += fmt.Sprintf(" objNr%d", sig.ObjNr)

	return s
}

// SignatureStats represents signature stats for a file.
type SignatureStats struct {
	FormSigned          int
	FormSignedVisible   int
	FormUnsigned        int
	FormUnsignedVisible int
	PageSigned          int
	PageSignedVisible   int
	PageUnsigned        int
	PageUnsignedVisible int
	URSigned            int
	URSignedVisible     int
	URUnsigned          int
	URUnsignedVisible   int
	DTSSigned           int
	DTSSignedVisible    int
	DTSUnsigned         int
	DTSUnsignedVisible  int

	Total int
}

// Counter returns counters for detected signatures.
func (sigStats *SignatureStats) Counter(svr *SignatureValidationResult) (*int, *int, *int, *int) {
	switch svr.Type {
	case SigTypeForm:
		return &sigStats.FormSigned, &sigStats.FormSignedVisible, &sigStats.FormUnsigned, &sigStats.FormUnsignedVisible
	case SigTypePage:
		return &sigStats.PageSigned, &sigStats.PageSignedVisible, &sigStats.PageUnsigned, &sigStats.PageUnsignedVisible
	case SigTypeUR:
		return &sigStats.URSigned, &sigStats.URSignedVisible, &sigStats.URUnsigned, &sigStats.URUnsignedVisible
	case SigTypeDTS:
		return &sigStats.DTSSigned, &sigStats.DTSSignedVisible, &sigStats.DTSUnsigned, &sigStats.DTSUnsignedVisible
	}
	return nil, nil, nil, nil
}

// SignatureStatus represents the compatibility status produced by the
// local assessment of observed evidence. It is not a legal, regulatory,
// enterprise-policy or policy-based trust decision.
type SignatureStatus int

const (
	SignatureStatusUnknown SignatureStatus = 1 << iota

	// SignatureStatusValid indicates that the supported cryptographic and local
	// validation checks completed successfully.
	SignatureStatusValid

	SignatureStatusInvalid
)

// SignatureStatusStrings manages string representations for signature statuses.
var SignatureStatusStrings = map[SignatureStatus]string{
	SignatureStatusUnknown: "validity of the signature is unknown",
	SignatureStatusValid:   "signature is valid",
	SignatureStatusInvalid: "signature is invalid",
}

// String returns the string value of st.
func (st SignatureStatus) String() string {
	return SignatureStatusStrings[st]
}

// SignatureReason identifies the reported reason associated with a signature status.
type SignatureReason int

const (
	SignatureReasonUnknown SignatureReason = 1 << iota
	SignatureReasonDocNotModified
	SignatureReasonDocModified
	SignatureReasonSignatureForged
	SignatureReasonSigningTimeInvalid
	SignatureReasonTimestampTokenInvalid
	SignatureReasonCertInvalid
	SignatureReasonCertNotTrusted
	SignatureReasonCertExpired
	SignatureReasonCertRevoked
	SignatureReasonInternal
	SignatureReasonSelfSignedCertErr

	// SignatureReasonCertRevocationUnknown indicates that the available
	// revocation sources did not establish a certificate status.
	SignatureReasonCertRevocationUnknown

	// SignatureReasonMalformed indicates malformed signature data.
	SignatureReasonMalformed

	// SignatureReasonUnsupported indicates an unsupported signature profile or algorithm.
	SignatureReasonUnsupported
)

// SignatureReasonStrings manages string representations for signature reasons.
var SignatureReasonStrings = map[SignatureReason]string{
	SignatureReasonUnknown:               "no reason",
	SignatureReasonDocNotModified:        "document has not been modified",
	SignatureReasonDocModified:           "document has been modified",
	SignatureReasonSignatureForged:       "signer's signature is not authentic",
	SignatureReasonTimestampTokenInvalid: "timestamp token is invalid",
	SignatureReasonCertInvalid:           "signer's certificate is invalid",
	SignatureReasonCertNotTrusted:        "signer's certificate path was not resolved using the configured local certificate store",
	SignatureReasonCertExpired:           "signer's certificate or one of its parent certificates has expired",
	SignatureReasonCertRevoked:           "signer's certificate has been revoked",
	SignatureReasonInternal:              "internal error",
	SignatureReasonSelfSignedCertErr:     "signer's self-signed certificate was not accepted by the configured local certificate assessment",
	SignatureReasonCertRevocationUnknown: "signer's certificate revocation status is unknown",
	SignatureReasonMalformed:             "signature data is malformed",
	SignatureReasonUnsupported:           "signature profile or algorithm is unsupported",
}

// String returns the string value of sr.
func (sr SignatureReason) String() string {
	return SignatureReasonStrings[sr]
}

// Signer contains certificate, timestamp, permission and problem details for a
// signature signer.
type Signer struct {
	Certificate           *CertificateDetails
	CertificatePathStatus int       // overall local path status; CA entries do not override it
	HasTimestamp          bool      // timestamp token presence; does not imply authentication
	Timestamp             time.Time // observed TSTInfo genTime or document timestamp
	LTVEnabled            bool      // retained for API compatibility; signature validation does not set it
	PAdES                 string    // supported baseline conclusion: B-B; timestamp and DSS evidence do not promote it
	Certified             bool      // indicated by DocMDP entry
	Authoritative         bool      // true if certified or first (youngest) signature
	Permissions           int       // see table 257
	Problems              []string
}

// AddProblem adds problem to signer.
func (signer *Signer) AddProblem(s string) {
	signer.Problems = append(signer.Problems, s)
}

func permString(i int) string {
	switch i {
	case CertifiedSigPermNoChangesAllowed:
		return "no changes allowed"
	case CertifiedSigPermFillingAndSigningOK:
		return "filling forms, signing"
	case CertifiedSigPermFillingAnnotatingAndSigningOK:
		return "filling forms, annotating, signing"
	}
	return ""
}

// String returns a string representation.
func (signer Signer) String(dts bool) string {
	ss := []string{}
	s := "false"
	if signer.HasTimestamp {
		if signer.Timestamp.IsZero() {
			s = "invalid"
		} else {
			s = signer.Timestamp.Format(SignTSFormat)
		}
	}

	ss = append(ss, fmt.Sprintf("             Timestamp:      %s", s))
	if !dts {
		if signer.PAdES != "" {
			ss = append(ss, fmt.Sprintf("             PAdES:          %s", signer.PAdES))
		}
		ss = append(ss, fmt.Sprintf("             Certified:      %t", signer.Certified))
		ss = append(ss, fmt.Sprintf("             Authoritative:  %t", signer.Authoritative))
		if signer.Certified && signer.Permissions > 0 {
			ss = append(ss, fmt.Sprintf("             Permissions:    %s", permString(signer.Permissions)))
		}
	}
	if signer.Certificate != nil {
		s := "             Certificate"
		if signer.Certificate.CA {
			s += "(CA)"
		}
		ss = append(ss, s+":")
		ss = append(ss, signer.Certificate.String())
	}

	for i, s := range signer.Problems {
		prefix := "                             "
		if i == 0 {
			prefix = "             Problems:       "
		}
		ss = appendWrappedSignatureText(ss, prefix, s)
	}

	return strings.Join(ss, "\n")
}

// SignatureDetails contains PDF signature dictionary metadata and signer details.
type SignatureDetails struct {
	SubFilter      string    // Signature Dict SubFilter
	SignerIdentity string    // extracted from signature
	SignerName     string    // Signature Dict Name
	ContactInfo    string    // Signature Dict ContactInfo
	Location       string    // Signature Dict Location
	Reason         string    // Signature Dict
	SigningTime    time.Time // Signature Dict M
	FieldName      string    // Signature Field T
	Signers        []*Signer
}

// AddSigner adds signer.
func (sd *SignatureDetails) AddSigner(s *Signer) {
	sd.Signers = append(sd.Signers, s)
}

// IsETSI_CAdES_detached reports whether ETSI c ad es detached.
func (sd *SignatureDetails) IsETSI_CAdES_detached() bool {
	return sd.SubFilter == "ETSI.CAdES.detached"
}

// IsETSI_RFC3161 reports whether ETSI rfc3161.
func (sd *SignatureDetails) IsETSI_RFC3161() bool {
	return sd.SubFilter == "ETSI.RFC3161"
}

// Permissions returns permissions of sd.
func (sd *SignatureDetails) Permissions() int {
	for _, signer := range sd.Signers {
		if signer.Certified {
			return signer.Permissions
		}
	}
	return CertifiedSigPermNone
}

// String returns the string value of sd.
func (sd SignatureDetails) String() string {
	ss := []string{}
	ss = append(ss, fmt.Sprintf("             SubFilter:      %s", sd.SubFilter))
	ss = append(ss, fmt.Sprintf("             SignerIdentity: %s", sd.SignerIdentity))
	ss = append(ss, fmt.Sprintf("             SignerName:     %s", sd.SignerName))
	if !sd.IsETSI_RFC3161() {
		ss = append(ss, fmt.Sprintf("             ContactInfo:    %s", sd.ContactInfo))
		ss = append(ss, fmt.Sprintf("             Location:       %s", sd.Location))
		ss = append(ss, fmt.Sprintf("             Reason:         %s", sd.Reason))
	}
	ss = append(ss, fmt.Sprintf("             SigningTime:    %s", sd.SigningTime.Format(SignTSFormat)))
	ss = append(ss, fmt.Sprintf("             Field:          %s", sd.FieldName))

	if len(sd.Signers) == 1 {
		ss = append(ss, "     Signer:")
		ss = append(ss, sd.Signers[0].String(sd.IsETSI_RFC3161()))
	} else {
		for i, signer := range sd.Signers {
			ss = append(ss, fmt.Sprintf("   Signer %d:", i+1))
			ss = append(ss, signer.String(sd.IsETSI_RFC3161()))
		}
	}

	return strings.Join(ss, "\n")
}

// SignatureValidationResult contains observed signature, certificate,
// timestamp and revocation evidence together with a local assessment. It is not
// a legal, regulatory, enterprise-policy or policy-based trust decision.
type SignatureValidationResult struct {
	Signature
	Status      SignatureStatus
	Reason      SignatureReason
	Details     SignatureDetails
	DocModified int
	Problems    []string
}

// AddProblem adds problem.
func (svr *SignatureValidationResult) AddProblem(s string) {
	svr.Problems = append(svr.Problems, s)
}

// Certified certified.
func (svr *SignatureValidationResult) Certified() bool {
	return svr.Signature.Certified
}

// Permissions permissions.
func (svr *SignatureValidationResult) Permissions() int {
	return svr.Details.Permissions()
}

// SigningTime signing time.
func (svr *SignatureValidationResult) SigningTime() string {
	if !svr.Details.SigningTime.IsZero() {
		return svr.Details.SigningTime.Format(SignTSFormat)
	}
	return "not available"
}

// String returns the string value of svr.
func (svr SignatureValidationResult) String() string {
	ss := []string{}

	ss = append(ss, fmt.Sprintf("       Type: %s", svr.Signature.String(svr.Status)))
	if !svr.Signed {
		return strings.Join(ss, "\n")
	}

	ss = append(ss, fmt.Sprintf("     Status: %s", svr.Status.String()))
	ss = append(ss, fmt.Sprintf("     Reason: %s", svr.Reason.String()))
	ss = append(ss, fmt.Sprintf("     Signed: %s", svr.SigningTime()))
	ss = append(ss, fmt.Sprintf("DocModified: %s", statusString(svr.DocModified)))
	ss = append(ss, fmt.Sprintf("    Details:\n%s", svr.Details))

	for i, s := range svr.Problems {
		prefix := "             "
		if i == 0 {
			prefix = "   Problems: "
		}
		ss = appendWrappedSignatureText(ss, prefix, s)
	}

	return strings.Join(ss, "\n")
}

func statusString(status int) string {
	switch status {
	case False:
		return "false"
	case True:
		return "true"
	}
	return "unknown"
}

func validString(status int) string {
	switch status {
	case False:
		return "not ok"
	case True:
		return "ok"
	}
	return "unknown"
}
