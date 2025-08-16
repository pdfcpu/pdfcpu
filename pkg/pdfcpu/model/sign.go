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

const SignTSFormat = "2006-01-02 15:04:05 -0700"

type RevocationDetails struct {
	Status int
	Reason string
}

func (rd RevocationDetails) String() string {
	ss := []string{}
	ss = append(ss, fmt.Sprintf(" Status: %s", validString(rd.Status)))
	if len(rd.Reason) > 0 {
		ss = append(ss, fmt.Sprintf("                                         Reason: %s", rd.Reason))
	}
	return strings.Join(ss, "\n")
}

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

func (td TrustDetails) String() string {
	ss := []string{}
	ss = append(ss, fmt.Sprintf("      Status: %s", validString(td.Status)))
	if len(td.Reason) > 0 {
		ss = append(ss, fmt.Sprintf("                                         Reason: %s", td.Reason))
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

type CertificateDetails struct {
	Leaf              bool
	SelfSigned        bool
	Subject           string
	Issuer            string
	SerialNumber      string
	ValidFrom         time.Time
	ValidThru         time.Time
	Expired           bool
	Qualified         bool
	CA                bool
	Usage             string
	Version           int
	SignAlg           string
	KeySize           int
	Revocation        RevocationDetails
	Trust             TrustDetails
	IssuerCertificate *CertificateDetails
}

func (cd CertificateDetails) String() string {
	ss := []string{}
	ss = append(ss, fmt.Sprintf("                             Subject:    %s", cd.Subject))
	ss = append(ss, fmt.Sprintf("                             Issuer:     %s", cd.Issuer))
	ss = append(ss, fmt.Sprintf("                             SerialNr:   %s", cd.SerialNumber))
	ss = append(ss, fmt.Sprintf("                             Valid From: %s", cd.ValidFrom.Format(SignTSFormat)))
	ss = append(ss, fmt.Sprintf("                             Valid Thru: %s", cd.ValidThru.Format(SignTSFormat)))
	ss = append(ss, fmt.Sprintf("                             Expired:    %t", cd.Expired))
	ss = append(ss, fmt.Sprintf("                             Qualified:  %t", cd.Qualified))
	ss = append(ss, fmt.Sprintf("                             CA:         %t", cd.CA))
	ss = append(ss, fmt.Sprintf("                             Usage:      %s", cd.Usage))
	ss = append(ss, fmt.Sprintf("                             Version:    %d", cd.Version))
	ss = append(ss, fmt.Sprintf("                             SignAlg:    %s", cd.SignAlg))
	ss = append(ss, fmt.Sprintf("                             Key Size:   %d bits", cd.KeySize))
	ss = append(ss, fmt.Sprintf("                             SelfSigned: %t", cd.SelfSigned))
	ss = append(ss, fmt.Sprintf("                             Trust:%s", cd.Trust))
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
		ss = append(ss, cd.IssuerCertificate.String())
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
		s1 := "trusted, "
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

func (sigStats SignatureStats) Counter(svr *SignatureValidationResult) (*int, *int, *int, *int) {
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

// SignatureStatus represents all possible signature statuses.
type SignatureStatus int

const (
	SignatureStatusUnknown SignatureStatus = 1 << iota
	SignatureStatusValid
	SignatureStatusInvalid
)

// SignatureStatusStrings manages string representations for signature statuses.
var SignatureStatusStrings = map[SignatureStatus]string{
	SignatureStatusUnknown: "validity of the signature is unknown",
	SignatureStatusValid:   "signature is valid",
	SignatureStatusInvalid: "signature is invalid",
}

func (st SignatureStatus) String() string {
	return SignatureStatusStrings[st]
}

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
)

// SignatureReasonStrings manages string representations for signature reasons.
var SignatureReasonStrings = map[SignatureReason]string{
	SignatureReasonUnknown:               "no reason",
	SignatureReasonDocNotModified:        "document has not been modified",
	SignatureReasonDocModified:           "document has been modified",
	SignatureReasonSignatureForged:       "signer's signature is not authentic",
	SignatureReasonTimestampTokenInvalid: "timestamp token is invalid",
	SignatureReasonCertInvalid:           "signer's certificate is invalid",
	SignatureReasonCertNotTrusted:        "signer's certificate chain is not in the trusted list of Root CAs",
	SignatureReasonCertExpired:           "signer's certificate or one of its parent certificates has expired",
	SignatureReasonCertRevoked:           "signer's certificate or one of its parent certificates has been revoked",
	SignatureReasonInternal:              "internal error",
	SignatureReasonSelfSignedCertErr:     "signer's self signed certificate is not trusted",
}

func (sr SignatureReason) String() string {
	return SignatureReasonStrings[sr]
}

type Signer struct {
	Certificate           *CertificateDetails
	CertificatePathStatus int
	HasTimestamp          bool
	Timestamp             time.Time // signature timestamp attribute (which contains a timestamp token)
	LTVEnabled            bool      // needs timestamp token & revocation info
	PAdES                 string    // baseline level: B-B, B-T, B-LT, B-LTA
	Certified             bool      // indicated by DocMDP entry
	Authoritative         bool      // true if certified or first (youngest) signature
	Permissions           int       // see table 257
	Problems              []string
}

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
		ss = append(ss, fmt.Sprintf("             LTVEnabled:     %t", signer.LTVEnabled))
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
		if i == 0 {
			ss = append(ss, fmt.Sprintf("             Problems:       %s", s))
			continue
		}
		ss = append(ss, fmt.Sprintf("                             %s", s))
	}

	return strings.Join(ss, "\n")
}

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

func (sd *SignatureDetails) AddSigner(s *Signer) {
	sd.Signers = append(sd.Signers, s)
}

func (sd *SignatureDetails) IsETSI_CAdES_detached() bool {
	return sd.SubFilter == "ETSI.CAdES.detached"
}

func (sd *SignatureDetails) IsETSI_RFC3161() bool {
	return sd.SubFilter == "ETSI.RFC3161"
}

func (sd *SignatureDetails) Permissions() int {
	for _, signer := range sd.Signers {
		if signer.Certified {
			return signer.Permissions
		}
	}
	return CertifiedSigPermNone
}

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

type SignatureValidationResult struct {
	Signature
	Status      SignatureStatus
	Reason      SignatureReason
	Details     SignatureDetails
	DocModified int
	Problems    []string
}

func (svr *SignatureValidationResult) AddProblem(s string) {
	svr.Problems = append(svr.Problems, s)
}

func (svr *SignatureValidationResult) Certified() bool {
	return svr.Signature.Certified
}

func (svr *SignatureValidationResult) Permissions() int {
	return svr.Details.Permissions()
}

func (svr *SignatureValidationResult) SigningTime() string {
	if !svr.Details.SigningTime.IsZero() {
		return svr.Details.SigningTime.Format(SignTSFormat)
	}
	return "not available"
}

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
		if i == 0 {
			ss = append(ss, fmt.Sprintf("   Problems: %s", s))
			continue
		}
		ss = append(ss, fmt.Sprintf("             %s", s))
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
