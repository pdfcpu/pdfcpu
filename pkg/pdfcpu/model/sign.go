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

// Three-state result values.
const (
	Unknown = iota
	False   // aka invalid, not ok
	True    // aka  valid, ok
)

// Preferred certificate revocation checking mechanisms.
const (
	CRL = iota
	OCSP
)

// Certified signature permission levels.
const (
	CertifiedSigPermNone = iota
	CertifiedSigPermNoChangesAllowed
	CertifiedSigPermFillingAndSigningOK
	CertifiedSigPermFillingAnnotatingAndSigningOK
)

// Supported PDF signature field types.
const (
	SigTypeForm = iota
	SigTypePage
	SigTypeUR
	SigTypeDTS
)

// SignTSFormat is the timestamp layout used in signature validation output.
const SignTSFormat = "2006-01-02 15:04:05 -0700"

const signatureOutputMaxWidth = 120

type signatureOutputField struct {
	label string
	value string
}

func signatureOutputFields(indent string, fields []signatureOutputField) []string {
	labelWidth := 0
	for _, field := range fields {
		if len(field.label) > labelWidth {
			labelWidth = len(field.label)
		}
	}

	var ss []string
	for _, field := range fields {
		prefix := indent + field.label + ":" + strings.Repeat(" ", labelWidth-len(field.label)+1)
		ss = appendWrappedSignatureText(ss, prefix, field.value)
	}
	return ss
}

func signatureOutputProblems(ss []string, headingIndent, valueIndent string, problems []string) []string {
	if len(problems) == 0 {
		return ss
	}
	ss = append(ss, "", headingIndent+"Problems:")
	for _, problem := range problems {
		ss = appendWrappedSignatureText(ss, valueIndent, problem)
	}
	return ss
}

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

// Revocation evidence source values.
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
	fields := []signatureOutputField{{label: "Status", value: validString(rd.Status)}}
	if len(rd.Reason) > 0 {
		fields = append(fields, signatureOutputField{label: "Reason", value: rd.Reason})
	}
	return strings.Join(signatureOutputFields("", fields), "\n")
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

// TimestampKind identifies the role of observed timestamp evidence.
type TimestampKind uint8

const (
	// TimestampKindUnspecified identifies absent or unclassified timestamp evidence.
	TimestampKindUnspecified TimestampKind = iota

	// TimestampKindSignature identifies an embedded timestamp over a signature value.
	TimestampKindSignature

	// TimestampKindDocument identifies an ETSI.RFC3161 document timestamp.
	TimestampKindDocument
)

// TimestampValidationEvidence records independent timestamp checks. Check
// fields use Unknown, False and True; the zero value establishes no conclusion.
type TimestampValidationEvidence struct {
	Kind                     TimestampKind
	Present                  bool
	Time                     time.Time
	DigestVerified           int
	SignatureAuthenticated   int
	ProfileValidated         int
	CertificatePathValidated int
}

// SignerValidationEvidence records independent signer checks. Check fields use
// Unknown, False and True; the zero value establishes no conclusion.
type SignerValidationEvidence struct {
	SignatureAuthenticated int
	DigestVerified         int
	ProfileValidated       int
	CertificateIdentified  int
	Timestamp              TimestampValidationEvidence
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
	fields := []signatureOutputField{{label: "Status", value: validString(td.Status)}}
	if len(td.Reason) > 0 {
		fields = append(fields, signatureOutputField{label: "Reason", value: td.Reason})
	}
	// if td.Status == True {
	// 	ss = append(ss, fmt.Sprintf("                                         SourceObtainedFrom:                    %s", td.SourceObtainedFrom))
	// 	ss = append(ss, fmt.Sprintf("                                         AllowSignDocuments:                    %t", td.AllowSignDocuments))
	// 	ss = append(ss, fmt.Sprintf("                                         AllowCertifyDocuments:                 %t", td.AllowCertifyDocuments))
	// 	ss = append(ss, fmt.Sprintf("                                         AllowExecuteDynamicContent:            %t", td.AllowExecuteDynamicContent))
	// 	ss = append(ss, fmt.Sprintf("                                         AllowExecuteJavaScript:                %t", td.AllowExecuteJavaScript))
	// 	ss = append(ss, fmt.Sprintf("                                         AllowExecutePrivilegedSystemOperation: %t", td.AllowExecutePrivilegedSystemOperation))
	// }
	return strings.Join(signatureOutputFields("", fields), "\n")
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
	return cd.string("")
}

func (cd CertificateDetails) string(indent string) string {
	ss := cd.output(indent)
	for issuer := cd.IssuerCertificate; issuer != nil; issuer = issuer.IssuerCertificate {
		s := "Intermediate"
		if issuer.IssuerCertificate == nil {
			s = "Root"
		}
		if issuer.CA {
			s += " CA"
		}
		ss = append(ss, "", indent+s+":")
		ss = append(ss, issuer.output(indent+"  ")...)
	}
	return strings.Join(ss, "\n")
}

func (cd CertificateDetails) chain(indent string) []string {
	heading := "Certificate"
	if cd.CA {
		heading += " (CA)"
	}
	ss := []string{indent + heading + ":"}
	ss = append(ss, cd.output(indent+"  ")...)
	for issuer := cd.IssuerCertificate; issuer != nil; issuer = issuer.IssuerCertificate {
		heading = "Intermediate"
		if issuer.IssuerCertificate == nil {
			heading = "Root"
		}
		if issuer.CA {
			heading += " CA"
		}
		ss = append(ss, "", indent+heading+":")
		ss = append(ss, issuer.output(indent+"  ")...)
	}
	return ss
}

func (cd CertificateDetails) output(indent string) []string {
	var fields []signatureOutputField
	if cd.Subject != "" {
		fields = append(fields, signatureOutputField{label: "Subject", value: cd.Subject})
	}
	if cd.Issuer != "" {
		fields = append(fields, signatureOutputField{label: "Issuer", value: cd.Issuer})
	}
	if cd.SerialNumber != "" {
		fields = append(fields, signatureOutputField{label: "Serial number", value: cd.SerialNumber})
	}
	if !cd.ValidFrom.IsZero() {
		fields = append(fields, signatureOutputField{label: "Valid from", value: cd.ValidFrom.Format(SignTSFormat)})
	}
	if !cd.ValidThru.IsZero() {
		fields = append(fields, signatureOutputField{label: "Valid thru", value: cd.ValidThru.Format(SignTSFormat)})
	}
	fields = append(fields,
		signatureOutputField{label: "Expired", value: fmt.Sprintf("%t", cd.Expired)},
		signatureOutputField{label: "QC policy", value: fmt.Sprintf("%t", cd.Qualified)},
		signatureOutputField{label: "CA", value: fmt.Sprintf("%t", cd.CA)},
	)
	if cd.Usage != "" {
		fields = append(fields, signatureOutputField{label: "Usage", value: cd.Usage})
	}
	if cd.Version > 0 {
		fields = append(fields, signatureOutputField{label: "Version", value: fmt.Sprintf("%d", cd.Version)})
	}
	if cd.SignAlg != "" {
		fields = append(fields, signatureOutputField{label: "Signing algorithm", value: cd.SignAlg})
	}
	if cd.KeySize > 0 {
		fields = append(fields, signatureOutputField{label: "Key size", value: fmt.Sprintf("%d bits", cd.KeySize)})
	}
	fields = append(fields, signatureOutputField{label: "Self-signed", value: fmt.Sprintf("%t", cd.SelfSigned)})
	ss := signatureOutputFields(indent, fields)

	ss = append(ss, "", indent+"Local path:")
	ss = append(ss, signatureOutputFields(indent+"  ", trustOutputFields(cd.Trust))...)
	if cd.Leaf && !cd.SelfSigned {
		ss = append(ss, "", indent+"Revocation:")
		ss = append(ss, signatureOutputFields(indent+"  ", revocationOutputFields(cd.Revocation))...)
	}
	return ss
}

func trustOutputFields(td TrustDetails) []signatureOutputField {
	fields := []signatureOutputField{{label: "Status", value: validString(td.Status)}}
	if td.Reason != "" {
		fields = append(fields, signatureOutputField{label: "Reason", value: td.Reason})
	}
	return fields
}

func revocationOutputFields(rd RevocationDetails) []signatureOutputField {
	fields := []signatureOutputField{{label: "Status", value: validString(rd.Status)}}
	if rd.Reason != "" {
		fields = append(fields, signatureOutputField{label: "Reason", value: rd.Reason})
	}
	return fields
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

// Signature validation status values.
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

// Signature validation reason values.
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
	Evidence              SignerValidationEvidence
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
	return signer.string(dts, "")
}

func (signer Signer) string(dts bool, indent string) string {
	var fields []signatureOutputField
	if signer.HasTimestamp {
		s := "invalid"
		if !signer.Timestamp.IsZero() {
			s = signer.Timestamp.Format(SignTSFormat)
		}
		fields = append(fields, signatureOutputField{label: "Timestamp", value: s})
	}

	if !dts {
		if signer.PAdES != "" {
			fields = append(fields, signatureOutputField{label: "PAdES", value: signer.PAdES})
		}
		fields = append(fields,
			signatureOutputField{label: "Authoritative", value: fmt.Sprintf("%t", signer.Authoritative)},
			signatureOutputField{label: "Certified", value: fmt.Sprintf("%t", signer.Certified)},
		)
		if signer.Certified && signer.Permissions > 0 {
			fields = append(fields, signatureOutputField{label: "Permissions", value: permString(signer.Permissions)})
		}
	}
	ss := signatureOutputFields(indent, fields)
	if signer.Certificate != nil {
		if len(ss) > 0 {
			ss = append(ss, "")
		}
		ss = append(ss, signer.Certificate.chain(indent)...)
	}

	ss = signatureOutputProblems(ss, indent, indent+"  ", signer.Problems)

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
	ss := signatureOutputFields("", sd.outputFields())
	ss = append(ss, sd.signerOutput()...)
	return strings.Join(ss, "\n")
}

func (sd SignatureDetails) outputFields() []signatureOutputField {
	var fields []signatureOutputField
	if sd.SubFilter != "" {
		fields = append(fields, signatureOutputField{label: "SubFilter", value: sd.SubFilter})
	}
	if sd.SignerIdentity != "" {
		identity := sd.SignerIdentity
		if strings.EqualFold(identity, "unknown") {
			identity = "unknown"
		}
		fields = append(fields, signatureOutputField{label: "Signer identity", value: identity})
	}
	if sd.SignerName != "" {
		fields = append(fields, signatureOutputField{label: "Signer name", value: sd.SignerName})
	}
	if !sd.IsETSI_RFC3161() {
		if sd.ContactInfo != "" {
			fields = append(fields, signatureOutputField{label: "Contact info", value: sd.ContactInfo})
		}
		if sd.Location != "" {
			fields = append(fields, signatureOutputField{label: "Location", value: sd.Location})
		}
		if sd.Reason != "" {
			fields = append(fields, signatureOutputField{label: "Reason", value: sd.Reason})
		}
	}
	if sd.FieldName != "" {
		fields = append(fields, signatureOutputField{label: "Field", value: sd.FieldName})
	}
	return fields
}

func (sd SignatureDetails) signerOutput() []string {
	var ss []string
	for i, signer := range sd.Signers {
		heading := "Signer:"
		if len(sd.Signers) > 1 {
			heading = fmt.Sprintf("Signer %d:", i+1)
		}
		if len(ss) > 0 || len(sd.outputFields()) > 0 {
			ss = append(ss, "")
		}
		ss = append(ss, heading)
		if signer != nil {
			ss = append(ss, signer.string(sd.IsETSI_RFC3161(), "  "))
		}
	}
	return ss
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
	ss := []string{svr.Signature.String(svr.Status)}
	if !svr.Signed {
		return strings.Join(ss, "\n")
	}

	ss = append(ss, "", "Assessment:")
	ss = append(ss, signatureOutputFields("  ", []signatureOutputField{
		{label: "Status", value: svr.Status.String()},
		{label: "Reason", value: svr.Reason.String()},
		{label: "Signed", value: svr.SigningTime()},
		{label: "Document modified", value: statusString(svr.DocModified)},
	})...)

	details := signatureOutputFields("  ", svr.Details.outputFields())
	if len(details) > 0 {
		ss = append(ss, "", "Details:")
		ss = append(ss, details...)
	}

	for i, signer := range svr.Details.Signers {
		heading := "Signer:"
		if len(svr.Details.Signers) > 1 {
			heading = fmt.Sprintf("Signer %d:", i+1)
		}
		ss = append(ss, "", heading)
		if signer != nil {
			ss = append(ss, signer.string(svr.Details.IsETSI_RFC3161(), "  "))
		}
	}

	ss = signatureOutputProblems(ss, "", "  ", svr.Problems)

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
