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

package sign

import (
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/pkcs7"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type timestampApplication struct {
	signer               *model.Signer
	result               *model.SignatureValidationResult
	setResultSigningTime bool
	problemPrefix        string
}

const embeddedTimestampNotAuthenticated = "embedded timestamp token observed but not fully authenticated"

func applyTimestampEvidence(evidence timestampEvidence, application timestampApplication) {
	if application.signer == nil {
		return
	}
	if evidence.Err != nil {
		application.signer.HasTimestamp = evidence.Present
		application.signer.AddProblem(fmt.Sprintf("%s: %v", application.problemPrefix, evidence.Err))
		if application.result != nil {
			setResultReason(application.result, model.SignatureReasonTimestampTokenInvalid)
		}
		return
	}
	if !evidence.Present || evidence.SigningTime.IsZero() {
		return
	}
	application.signer.HasTimestamp = true
	application.signer.Timestamp = evidence.SigningTime
	if evidence.Kind == timestampKindSignature {
		application.signer.AddProblem(fmt.Sprintf("%s: %s", application.problemPrefix, embeddedTimestampNotAuthenticated))
		return
	}
	if !isLocallyValidatedDocumentTimestampEvidence(evidence) {
		return
	}
	if application.setResultSigningTime && application.result != nil {
		application.result.Details.SigningTime = evidence.SigningTime
	}
}

func embeddedSignatureTimestampEvidence(p7Signer pkcs7.SignerInfo) timestampEvidence {
	// Collect locally available RFC 3161 timestamp evidence.
	evidence := timestampEvidence{
		Kind:            timestampKindSignature,
		AssessmentScope: model.AssessmentScopeLocal,
		SourceSigner:    p7Signer,
	}
	bb, err := locateTimestampToken(p7Signer)
	if err != nil {
		evidence.Present = true
		evidence.Err = fmt.Errorf("timestamp token: locate: %w", err)
		return evidence
	}
	if len(bb) == 0 {
		return evidence
	}
	evidence.Present = true
	evidence.RawToken = append([]byte(nil), bb...)
	evidence.CMS, err = pkcs7.Parse(bb)
	if err != nil {
		evidence.Err = fmt.Errorf("timestamp token: parse: %w", err)
		return evidence
	}
	if len(evidence.CMS.Signers) != 1 {
		evidence.Err = fmt.Errorf("timestamp token: expected one signer, got %d", len(evidence.CMS.Signers))
		return evidence
	}
	evidence.Err = populateEmbeddedTimestampInfo(&evidence)
	return evidence
}

func populateEmbeddedTimestampInfo(evidence *timestampEvidence) error {
	if evidence.CMS == nil || !evidence.CMS.ContentType.Equal(oidTSTInfo) {
		return errors.New("timestamp token: missing timestamp info")
	}
	tstInfo, err := parseTSTInfo(evidence.CMS.Content)
	evidence.TokenInfoErr = err
	if err != nil {
		return fmt.Errorf("timestamp token: parse timestamp info: %w", err)
	}
	evidence.TokenInfo = structuredTimestampTokenInfo(tstInfo)
	evidence.SigningTime = tstInfo.GenTime
	return nil
}

func documentTimestampEvidence(signingTime time.Time) timestampEvidence {
	return timestampEvidence{
		Kind:            timestampKindDocument,
		SigningTime:     signingTime,
		Present:         true,
		AssessmentScope: model.AssessmentScopeLocal,
	}
}

func preparedDocumentTimestampEvidence(
	tstInfo *TSTInfo,
	rawToken []byte,
	cms *pkcs7.PKCS7,
	sourceSigner pkcs7.SignerInfo,
	signedData []byte,
) timestampEvidence {
	evidence := documentTimestampEvidence(tstInfo.GenTime)
	evidence.RawToken = append([]byte(nil), rawToken...)
	evidence.CMS = cms
	evidence.SourceSigner = sourceSigner
	evidence.TokenInfo = structuredTimestampTokenInfo(tstInfo)
	evidence.SignedData = signedData
	evidence.DigestVerified = true
	evidence.SignatureVerified = true
	return evidence
}

func structuredTimestampTokenInfo(tstInfo *TSTInfo) *timestampTokenInfo {
	if tstInfo == nil {
		return nil
	}
	return &timestampTokenInfo{
		Version:                 tstInfo.Version,
		Policy:                  tstInfo.Policy,
		MessageImprintAlgorithm: tstInfo.MessageImprint.HashAlgorithm.Algorithm,
		MessageImprint:          append([]byte(nil), tstInfo.MessageImprint.HashedMessage...),
		SerialNumber:            tstInfo.SerialNumber,
		GeneratedAt:             tstInfo.GenTime,
		Accuracy:                tstInfo.Accuracy,
		Ordering:                tstInfo.Ordering,
		Nonce:                   tstInfo.Nonce,
		TSA:                     tstInfo.TSA,
		Extensions:              tstInfo.Extensions,
	}
}

// ValidatePKCS7Signatures reports observed signature, certificate, timestamp
// and revocation evidence together with a local assessment for supported
// PKCS#7 SubFilters.
func ValidatePKCS7Signatures(
	ra io.ReaderAt,
	sigDict types.Dict,
	certified bool,
	authoritative bool,
	validateAll bool,
	perms int,
	rootCerts *x509.CertPool,
	result *model.SignatureValidationResult,
	ctx *model.Context,
) error {
	return validatePKCS7Signatures(
		ra,
		sigDict,
		certified,
		authoritative,
		validateAll,
		perms,
		rootCerts,
		result,
		ctx,
	)
}

func validatePKCS7Signatures(
	ra io.ReaderAt,
	sigDict types.Dict,
	certified bool,
	authoritative bool,
	validateAll bool,
	perms int,
	rootCerts *x509.CertPool,
	result *model.SignatureValidationResult,
	ctx *model.Context,
) error {
	if ctx.Configuration.Offline {
		result.AddProblem("pdfcpu is offline, unable to perform certificate revocation checking")
	}

	p7 := validateP7(sigDict, result)
	if p7 == nil {
		return nil
	}

	data, err := signedData(ra, sigDict)
	if err != nil {
		if !errors.Is(err, errMalformedByteRange) {
			return fmt.Errorf("read signed data: %w", err)
		}
		result.Reason = model.SignatureReasonMalformed
		result.AddProblem(fmt.Sprintf("read signed data: %v", err))
		return nil
	}

	detached := len(p7.Content) == 0
	if detached {
		p7.Content = data
	}

	var localAssessment localSignatureAssessment
	for i, p7Signer := range p7.Signers {
		signerAssessment := localSignatureAssessment{SignersProcessed: 1}
		if err := verifyP7SignerWithContentType(
			p7Signer,
			p7.Certificates,
			rootCerts,
			p7.Content,
			data,
			detached,
			certified,
			authoritative,
			perms,
			i,
			result,
			ctx,
			p7.ContentType,
			&signerAssessment,
		); err != nil {
			return fmt.Errorf("signer %d: %w", i+1, err)
		}
		localAssessment.merge(signerAssessment)
		if (certified || authoritative) && !validateAll {
			break
		}
	}

	finalizePKCS7Result(result, localAssessment)

	return nil
}

func finalizePKCS7Result(
	result *model.SignatureValidationResult,
	assessment localSignatureAssessment,
) {
	if assessment.SignersProcessed > 0 &&
		finalizeLocalSignatureResult(result, assessment) {
		return
	}

	// Show PAdES basic evidence levels for valid signatures only.
	for _, signer := range result.Details.Signers {
		signer.PAdES = ""
	}
}

func p7(sigDict types.Dict) (*pkcs7.PKCS7, error) {
	signature, err := signatureContents(sigDict)
	if err != nil {
		return nil, err
	}

	p7, err := pkcs7.Parse(signature)
	if err != nil {
		if errors.Is(err, pkcs7.ErrCertificateParse) {
			return nil, fmt.Errorf("PKCS#7 embedded certificates: parse certificate: %w", err)
		}
		return nil, fmt.Errorf("parse PKCS#7: %w", err)
	}

	return p7, nil
}

func signatureContents(sigDict types.Dict) ([]byte, error) {
	hl := sigDict.HexLiteralEntry("Contents")
	if hl == nil {
		return nil, errors.New("signature dict entry Contents: missing")
	}

	signature, err := hl.Bytes()
	if err != nil {
		return nil, fmt.Errorf("signature dict entry Contents: decode: %w", err)
	}
	return signature, nil
}

func verifyP7Signer(
	p7Signer pkcs7.SignerInfo,
	p7Certs []*x509.Certificate,
	rootCerts *x509.CertPool,
	p7Content []byte,
	data []byte,
	detached bool,
	certified bool,
	authoritative bool,
	perms, i int,
	result *model.SignatureValidationResult,
	ctx *model.Context,
) error {
	return verifyP7SignerWithContentType(
		p7Signer,
		p7Certs,
		rootCerts,
		p7Content,
		data,
		detached,
		certified,
		authoritative,
		perms,
		i,
		result,
		ctx,
		pkcs7.OIDData,
		nil,
	)
}

func verifyP7SignerWithContentType(
	p7Signer pkcs7.SignerInfo,
	p7Certs []*x509.Certificate,
	rootCerts *x509.CertPool,
	p7Content []byte,
	data []byte,
	detached bool,
	certified bool,
	authoritative bool,
	perms, i int,
	result *model.SignatureValidationResult,
	ctx *model.Context,
	contentType asn1.ObjectIdentifier,
	localAssessment *localSignatureAssessment,
) error {
	if localAssessment == nil {
		localAssessment = &localSignatureAssessment{}
	}
	conf := ctx.Configuration

	signer := &model.Signer{}
	result.Details.AddSigner(signer)

	signer.Certified = certified
	signer.Authoritative = signer.Certified || authoritative
	signer.Permissions = perms

	checkPerms(signer, result)
	digestReason, digestErr := verifyP7Digest(p7Signer, p7Content, data, detached)

	signerCert, err := pkcs7.GetCertFromCertsByIssuerAndSerial(p7Certs, p7Signer.IssuerAndSerialNumber)
	if err != nil {
		markCertificateInvalidEvidence(result)
		signer.AddProblem(fmt.Sprintf("pkcs7: signer %d identifier: %v", i+1, err))
		return nil
	}
	if signerCert == nil {
		markCertificateInvalidEvidence(result)
		signer.AddProblem(fmt.Sprintf("pkcs7: missing certificate for signer %d", i+1))
		return nil
	}
	localAssessment.CertificateIdentified = true

	if err := verifyP7Signature(p7Signer, signerCert, p7Content, contentType); err != nil {
		reportP7SignatureError(err, signer, result)
		return nil
	}
	localAssessment.SignatureAuthenticated = true

	// The signature verifies with the public key in the identified certificate.
	if !applyP7DigestEvidence(digestReason, digestErr, signer, result) {
		return nil
	}
	localAssessment.DigestVerified = true
	markDocumentUnmodified(result)

	applyP7ProfileAssessment(
		result.Details.SubFilter,
		detached,
		contentType,
		p7Signer,
		signerCert,
		signer,
		result,
		localAssessment,
	)

	// Process optional DSS and DTS for embedded revocation and timestamp evidence.

	// Record the claimed signing time for compatible presentation only.
	// Locally validated document timestamp evidence does not select an
	// archival validation time.

	// TODO Handle oidArchiveTimestamp

	handleClaimedSigningTime(p7Signer, signer, result)

	if !ctx.DTS.IsZero() {
		if result.Details.SigningTime.After(ctx.DTS) {
			signer.AddProblem(fmt.Sprintf("Claimed signing time: %s is not before document timestamp: %s",
				result.Details.SigningTime.Format(conf.TimestampFormat),
				ctx.DTS.Format(conf.TimestampFormat)))
		}
	}

	checkTimestampToken(
		p7Signer,
		ctx,
		signer,
		result,
	)

	// Look for embedded revocation info.
	crls, ocsps := handleArchivedRevocationInfo(p7Signer, signer)

	certs := p7Certs

	handleDSS(&certs, &crls, &ocsps, ctx, signer, result, detached)

	// Collect certificate-path evidence using the configured local certificate sources.
	chains := buildP7CertChains(i == 0, signerCert, certs, rootCerts, signer, result)
	pathResolved := len(chains) > 0
	if len(chains) == 0 {
		chains = [][]*x509.Certificate{certChain(signerCert, certs)}
	}

	assessment, err := assessCertificateEvidence(
		chains,
		pathResolved,
		rootCerts,
		crls,
		ocsps,
		result.Reason,
		ctx.Configuration,
	)
	if err != nil {
		return fmt.Errorf("pkcs7: assess certificate evidence: %w", err)
	}
	applyCertificateAssessment(assessment, signer, result)
	localAssessment.applyCertificateAssessment(assessment)
	return nil
}

func applyP7ProfileAssessment(
	subFilter string,
	detached bool,
	contentType asn1.ObjectIdentifier,
	p7Signer pkcs7.SignerInfo,
	signerCert *x509.Certificate,
	signer *model.Signer,
	result *model.SignatureValidationResult,
	assessment *localSignatureAssessment,
) {
	switch subFilter {
	case "adbe.pkcs7.detached", "adbe.pkcs7.sha1":
		if err := validateAdobePKCS7Profile(subFilter, detached, contentType); err != nil {
			markMalformedEvidence(result)
			signer.AddProblem(fmt.Sprintf("SubFilter %s: validate profile: %v", subFilter, err))
			return
		}
		assessment.ProfileValidated = true
	case "ETSI.CAdES.detached":
		if err := validateCAdESBaselineBProfile(detached, contentType, p7Signer, signerCert); err != nil {
			reportCAdESBaselineBProfileError(err, signer, result)
			return
		}
		assessment.ProfileValidated = true
		signer.PAdES = "B-B"
	}
}

func validateAdobePKCS7Profile(
	subFilter string,
	detached bool,
	contentType asn1.ObjectIdentifier,
) error {
	if !contentType.Equal(oidData) {
		return fmt.Errorf("%w: content type %s", errMalformedAdobePKCS7Profile, contentType)
	}
	if subFilter == "adbe.pkcs7.detached" && !detached {
		return fmt.Errorf("%w: signed content is encapsulated", errMalformedAdobePKCS7Profile)
	}
	if subFilter == "adbe.pkcs7.sha1" && detached {
		return fmt.Errorf("%w: signed content is detached", errMalformedAdobePKCS7Profile)
	}
	return nil
}

func validateCAdESBaselineBProfile(
	detached bool,
	contentType asn1.ObjectIdentifier,
	p7Signer pkcs7.SignerInfo,
	signerCert *x509.Certificate,
) error {
	if !detached {
		return fmt.Errorf("%w: signed content is encapsulated", errMalformedCAdESBaselineBProfile)
	}
	if !contentType.Equal(oidData) {
		return fmt.Errorf(
			"%w: content type %s",
			errUnsupportedCAdESBaselineBProfile,
			contentType,
		)
	}
	if err := validateESSCertificateBinding(p7Signer, signerCert); err != nil {
		return classifyCAdESBaselineBProfileError(err)
	}
	return nil
}

func classifyCAdESBaselineBProfileError(err error) error {
	switch {
	case errors.Is(err, errESSCertificateMismatch):
		return fmt.Errorf("%w: signing-certificate binding: %w", errCAdESCertificateBindingMismatch, err)
	case errors.Is(err, pkcs7.ErrUnsupportedAlgorithm),
		errors.Is(err, errUnsupportedESSCertificateProfile):
		return fmt.Errorf("%w: signing-certificate binding: %w", errUnsupportedCAdESBaselineBProfile, err)
	default:
		return fmt.Errorf("%w: signing-certificate binding: %w", errMalformedCAdESBaselineBProfile, err)
	}
}

func reportCAdESBaselineBProfileError(
	err error,
	signer *model.Signer,
	result *model.SignatureValidationResult,
) {
	switch {
	case errors.Is(err, errCAdESCertificateBindingMismatch):
		markInvalidEvidence(result, model.SignatureReasonCertInvalid, model.Unknown)
	case errors.Is(err, errUnsupportedCAdESBaselineBProfile):
		markUnsupportedEvidence(result)
	default:
		markMalformedEvidence(result)
	}
	signer.AddProblem(fmt.Sprintf("SubFilter ETSI.CAdES.detached: validate baseline B profile: %v", err))
}

func checkPerms(signer *model.Signer, result *model.SignatureValidationResult) {
	if signer.Certified && signer.Permissions != model.CertifiedSigPermNoChangesAllowed {
		// TODO Check for violation of perm 2 and 3
		signer.AddProblem(CertifiedSigPermsNotSupported)
		markUnsupportedEvidence(result)
	}
}

func applyP7DigestEvidence(
	reason model.SignatureReason,
	err error,
	signer *model.Signer,
	result *model.SignatureValidationResult,
) bool {
	if err == nil {
		return true
	}

	switch reason {
	case model.SignatureReasonDocModified:
		markInvalidEvidence(result, model.SignatureReasonDocModified, model.True)
	case model.SignatureReasonUnsupported:
		markUnsupportedEvidence(result)
	default:
		markMalformedEvidence(result)
	}

	signer.AddProblem(fmt.Sprintf("%v", err))
	return false
}

func verifyP7Digest(p7Signer pkcs7.SignerInfo, p7Content []byte, data []byte, detached bool) (model.SignatureReason, error) {
	// Verify Message Digest
	// Calculate fingerprint and compare with p7.Digest (content hash comparison).
	// Ensures integrity of the document content itself and ensures that the document has not been tampered with since it was signed.

	if detached {

		if len(p7Signer.AuthenticatedAttributes) == 0 {
			return model.SignatureReasonMalformed, errors.New("pkcs7: missing authenticated attributes")
		}

		if err := pkcs7.VerifyMessageDigestDetached(p7Signer, p7Content); err != nil {
			var mdErr *pkcs7.MessageDigestMismatchError
			if errors.As(err, &mdErr) {
				return model.SignatureReasonDocModified, fmt.Errorf("pkcs7: verify message digest: mismatch: %w", err)
			}
			if isUnsupportedP7SignatureError(err) {
				return model.SignatureReasonUnsupported, fmt.Errorf("pkcs7: verify message digest: %w", err)
			}
			return model.SignatureReasonMalformed, fmt.Errorf("pkcs7: verify message digest: %w", err)
		}

	} else {

		if err := pkcs7.VerifyMessageDigestEmbedded(p7Content, data); err != nil {
			return model.SignatureReasonDocModified, fmt.Errorf("pkcs7: verify message digest: mismatch: %w", err)
		}

	}

	return model.SignatureReasonDocNotModified, nil
}

func checkTimestampToken(
	p7Signer pkcs7.SignerInfo,
	_ *model.Context,
	signer *model.Signer,
	result *model.SignatureValidationResult,
) {
	evidence := embeddedSignatureTimestampEvidence(p7Signer)
	applyTimestampEvidence(evidence, timestampApplication{
		signer:        signer,
		result:        result,
		problemPrefix: "pkcs7",
	})
}

func handleDSS(
	certs *[]*x509.Certificate,
	crls *[][]byte,
	ocsps *[][]byte,
	ctx *model.Context,
	signer *model.Signer,
	result *model.SignatureValidationResult,
	_ bool,
) {
	if len(ctx.DSS) == 0 {
		return
	}
	evidence := processDSS(ctx, signer)
	*certs = mergeCerts(*certs, evidence.Certificates)
	*crls = append(*crls, evidence.CRLs...)
	*ocsps = append(*ocsps, evidence.OCSPResponses...)
	if !evidence.Supported {
		markUnsupportedEvidence(result)
	}
}

func verifyP7Signature(
	p7Signer pkcs7.SignerInfo,
	cert *x509.Certificate,
	p7Content []byte,
	contentType ...asn1.ObjectIdentifier,
) error {
	// Verify signature against expected hash using the public key.
	// The signature verifies with the public key in the identified certificate.

	expectedContentType := pkcs7.OIDData
	if len(contentType) > 0 {
		expectedContentType = contentType[0]
	}
	return pkcs7.CheckSignatureWithContentType(cert, p7Signer, p7Content, expectedContentType)
}

func reportP7SignatureError(
	err error,
	signer *model.Signer,
	result *model.SignatureValidationResult,
) {
	reportSignatureVerificationError(
		"pkcs7: verify signature",
		err,
		model.SignatureReasonDocModified,
		signer,
		result,
	)
}

func reportSignatureVerificationError(
	phase string,
	err error,
	contentMismatchReason model.SignatureReason,
	signer *model.Signer,
	result *model.SignatureValidationResult,
) {
	if isMalformedP7SignatureError(err) {
		markMalformedEvidence(result)
		signer.AddProblem(fmt.Sprintf("%s malformed: %v", phase, err))
		return
	}
	if isUnsupportedP7SignatureError(err) {
		markUnsupportedEvidence(result)
		signer.AddProblem(fmt.Sprintf("%s unsupported: %v", phase, err))
		return
	}
	var digestMismatchErr *pkcs7.MessageDigestMismatchError
	if errors.As(err, &digestMismatchErr) {
		docModified := model.Unknown
		if contentMismatchReason == model.SignatureReasonDocModified {
			docModified = model.True
		}
		markInvalidEvidence(result, contentMismatchReason, docModified)
		signer.AddProblem(fmt.Sprintf("%s content mismatch: %v", phase, err))
		return
	}
	if errors.Is(err, pkcs7.ErrSignatureMismatch) {
		markInvalidEvidence(result, model.SignatureReasonSignatureForged, model.Unknown)
		signer.AddProblem(fmt.Sprintf("%s failure: %v", phase, err))
		return
	}
	markMalformedEvidence(result)
	signer.AddProblem(fmt.Sprintf("%s malformed: %v", phase, err))
}

func isUnsupportedP7SignatureError(err error) bool {
	if errors.Is(err, pkcs7.ErrUnsupportedAlgorithm) ||
		errors.Is(err, pkcs7.ErrAlgorithmMismatch) ||
		errors.Is(err, x509.ErrUnsupportedAlgorithm) {
		return true
	}
	var insecureAlgorithmErr x509.InsecureAlgorithmError
	if errors.As(err, &insecureAlgorithmErr) {
		return true
	}
	return false
}

func isMalformedP7SignatureError(err error) bool {
	if errors.Is(err, pkcs7.ErrInvalidPSSParameters) ||
		errors.Is(err, pkcs7.ErrMalformedAttribute) {
		return true
	}
	var syntaxErr asn1.SyntaxError
	if errors.As(err, &syntaxErr) {
		return true
	}
	var structuralErr asn1.StructuralError
	if errors.As(err, &structuralErr) {
		return true
	}
	return false
}

func handleClaimedSigningTime(signerInfo pkcs7.SignerInfo, signer *model.Signer, result *model.SignatureValidationResult) {
	signingTime, err := parseClaimedSigningTime(signerInfo)
	if err != nil {
		signer.AddProblem(fmt.Sprintf("%v", err))
		setResultReason(result, model.SignatureReasonSigningTimeInvalid)
		return
	}
	if signingTime != nil && result.Details.SigningTime.IsZero() {
		result.Details.SigningTime = *signingTime
	}
}

func parseClaimedSigningTime(signerInfo pkcs7.SignerInfo) (*time.Time, error) {
	var (
		signingTime *time.Time
		firstIndex  int
	)
	for i, attr := range signerInfo.AuthenticatedAttributes {
		if !attr.Type.Equal(oidSigningTime) {
			continue
		}
		if signingTime != nil {
			return nil, fmt.Errorf(
				"pkcs7: signing time attributes %d and %d: duplicate",
				firstIndex,
				i+1,
			)
		}
		var t time.Time
		rest, err := asn1.Unmarshal(attr.Value.Bytes, &t)
		if err != nil {
			return nil, fmt.Errorf("pkcs7: signing time attribute %d: unmarshal: %w", i+1, err)
		}
		if len(rest) > 0 {
			err := asn1.SyntaxError{Msg: "trailing data"}
			return nil, fmt.Errorf("pkcs7: signing time attribute %d: %w", i+1, err)
		}
		signingTime = &t
		firstIndex = i + 1
	}
	return signingTime, nil
}

func locateTimestampToken(signerInfo pkcs7.SignerInfo) ([]byte, error) {
	var (
		token      []byte
		firstIndex int
	)
	for i, attr := range signerInfo.UnauthenticatedAttributes {
		if attr.Type.Equal(oidTimestampToken) {
			if firstIndex != 0 {
				return nil, fmt.Errorf(
					"timestamp token attributes %d and %d: duplicate",
					firstIndex,
					i+1,
				)
			}
			token = attr.Value.Bytes
			firstIndex = i + 1
		}
	}
	return token, nil
}

func extractTimestampTokenTime(data []byte) (time.Time, error) {
	var defTime time.Time
	p7, err := pkcs7.Parse(data)
	if err != nil {
		return defTime, fmt.Errorf("timestamp token: parse: %w", err)
	}

	if len(p7.Signers) != 1 {
		return defTime, fmt.Errorf("timestamp token: expected one signer, got %d", len(p7.Signers))
	}
	signer := p7.Signers[0]
	return timestampTokenSigningTime(signer)
}

func timestampTokenSigningTime(signer pkcs7.SignerInfo) (time.Time, error) {
	var defTime time.Time
	var (
		signingTime *time.Time
		firstIndex  int
	)
	for i, attr := range signer.AuthenticatedAttributes {
		if !attr.Type.Equal(oidSigningTime) {
			continue
		}
		if signingTime != nil {
			return defTime, fmt.Errorf(
				"timestamp token: signing time attributes %d and %d: duplicate",
				firstIndex,
				i+1,
			)
		}
		t, err := parseTimestampTokenSigningTime(attr.Value.Bytes)
		if err != nil {
			return defTime, fmt.Errorf("timestamp token: signing time attribute %d: %w", i+1, err)
		}
		signingTime = &t
		firstIndex = i + 1
	}

	if signingTime != nil {
		return *signingTime, nil
	}
	return defTime, errors.New("timestamp token: signing time unavailable")
}

func parseTimestampTokenSigningTime(bb []byte) (time.Time, error) {
	var rawValue asn1.RawValue
	rest, err := asn1.Unmarshal(bb, &rawValue)
	if err != nil {
		return time.Time{}, fmt.Errorf("timestamp token signing time: unmarshal: %w", err)
	}
	if len(rest) > 0 {
		err := asn1.SyntaxError{Msg: "trailing data"}
		return time.Time{}, fmt.Errorf("timestamp token signing time: %w", err)
	}
	t, err := parseTimestampSigningTime(rawValue)
	if err != nil {
		return time.Time{}, fmt.Errorf("timestamp token signing time: %w", err)
	}
	return t, nil
}

func parseTimestampSigningTime(rawValue asn1.RawValue) (time.Time, error) {
	var (
		layout string
		phase  string
	)
	switch rawValue.Tag {
	case asn1.TagUTCTime:
		layout = "060102150405Z"
		phase = "parse UTC signing time"
	case asn1.TagGeneralizedTime:
		layout = "20060102150405Z"
		phase = "parse generalized signing time"
	default:
		return time.Time{}, fmt.Errorf("unexpected tag for signing time: %d", rawValue.Tag)
	}
	t, err := time.Parse(layout, string(rawValue.Bytes))
	if err != nil {
		return time.Time{}, fmt.Errorf("%s: %w", phase, err)
	}
	return t, nil
}

func handleArchivedRevocationInfo(p7Signer pkcs7.SignerInfo, signer *model.Signer) (crls [][]byte, ocsps [][]byte) {
	ria, err := revocationInfoArchival(p7Signer)
	if err != nil {
		signer.AddProblem(fmt.Sprintf("%v", err))
		return nil, nil
	}
	if ria == nil {
		return nil, nil
	}

	for _, raw := range ria.CRLs {
		crls = append(crls, raw.FullBytes)
	}

	for _, raw := range ria.OCSPs {
		ocsps = append(ocsps, raw.FullBytes)
	}

	return
}

func buildP7CertChains(
	first bool,
	cert *x509.Certificate,
	certs []*x509.Certificate,
	rootCerts *x509.CertPool,
	signer *model.Signer,
	result *model.SignatureValidationResult) [][]*x509.Certificate {
	intermediates := collectIntermediates(cert, certs)
	chains, err := pkcs7.VerifyCertChain(cert, intermediates, rootCerts, time.Now())
	if err != nil {
		handleCertVerifyErr(err, cert, signer, result)
		return nil
	}
	if first {
		result.Details.SignerIdentity = cert.Subject.CommonName
	}
	return chains
}

func handleTimestampToken(
	p7Signer pkcs7.SignerInfo,
	signer *model.Signer,
	result *model.SignatureValidationResult,
) {
	evidence := embeddedSignatureTimestampEvidence(p7Signer)
	applyTimestampEvidence(evidence, timestampApplication{
		signer:        signer,
		result:        result,
		problemPrefix: "pkcs7",
	})
}

func revocationInfoArchival(p7Signer pkcs7.SignerInfo) (*RevocationInfoArchival, error) {
	var (
		ria        *RevocationInfoArchival
		firstIndex int
	)
	for i, attr := range p7Signer.AuthenticatedAttributes {
		if !attr.Type.Equal(oidRevocationInfoArchival) {
			continue
		}
		if ria != nil {
			return nil, fmt.Errorf(
				"pkcs7: revocation info archival attributes %d and %d: duplicate",
				firstIndex,
				i+1,
			)
		}
		var value RevocationInfoArchival
		rest, err := asn1.Unmarshal(attr.Value.Bytes, &value)
		if err != nil {
			return nil, fmt.Errorf("pkcs7: revocation info archival attribute %d: unmarshal: %w", i+1, err)
		}
		if len(rest) > 0 {
			err := asn1.SyntaxError{Msg: "trailing data"}
			return nil, fmt.Errorf("pkcs7: revocation info archival attribute %d: %w", i+1, err)
		}
		ria = &value
		firstIndex = i + 1
	}
	return ria, nil
}
