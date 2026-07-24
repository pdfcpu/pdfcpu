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
	"bytes"
	"crypto"
	"crypto/subtle"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"fmt"
	"io"
	"math/big"
	"slices"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/pkcs7"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// AlgorithmIdentifier represents an RFC 3161 message-imprint algorithm.
type AlgorithmIdentifier struct {
	Algorithm  asn1.ObjectIdentifier
	Parameters asn1.RawValue `asn1:"tag:0,optional"`
}

// TSTInfo represents the RFC 3161 timestamp token information.
type TSTInfo struct {
	Version        int
	Policy         asn1.ObjectIdentifier
	MessageImprint struct {
		HashAlgorithm AlgorithmIdentifier
		HashedMessage []byte
	}
	SerialNumber asn1.RawValue
	GenTime      time.Time
	Accuracy     asn1.RawValue `asn1:"optional"`
	Ordering     bool          `asn1:"optional"`
	Nonce        asn1.RawValue `asn1:"optional"`
	TSA          asn1.RawValue `asn1:"optional"`
	Extensions   asn1.RawValue `asn1:"optional"`
}

type essIssuerSerial struct {
	Issuer       []asn1.RawValue
	SerialNumber *big.Int
}

type essCertID struct {
	CertHash     []byte
	IssuerSerial asn1.RawValue `asn1:"optional"`
}

type essCertIDv2 struct {
	HashAlgorithm asn1.RawValue `asn1:"optional"`
	CertHash      []byte
	IssuerSerial  asn1.RawValue `asn1:"optional"`
}

type signingCertificate struct {
	Certs    []essCertID
	Policies asn1.RawValue `asn1:"optional"`
}

type signingCertificateV2 struct {
	Certs    []essCertIDv2
	Policies asn1.RawValue `asn1:"optional"`
}

type rawSigningCertificateV2 struct {
	Certs    []asn1.RawValue
	Policies asn1.RawValue `asn1:"optional"`
}

// ValidateDTS reports observed signature, certificate, timestamp and
// revocation evidence together with a local assessment for an ETSI.RFC3161
// document timestamp.
func ValidateDTS(
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
	return validateDTS(
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

func validateDTS(
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
	localAssessment := localSignatureAssessment{}
	ctx.DTS = time.Time{}

	// The last increment contains the DocTimeStamp only.

	// TODO if DocMDP ignore DTS.

	// Note: perms are disregarded for ETSI.RFC3161.

	if ctx.Configuration.Offline {
		result.AddProblem("pdfcpu is offline, unable to perform certificate revocation checking")
	}

	p7 := validateP7(sigDict, result)
	if p7 == nil {
		return nil
	}

	if len(p7.Signers) != 1 {
		result.Reason = model.SignatureReasonTimestampTokenInvalid
		result.AddProblem("SubFilter ETSI.RFC3161: requires a single signer")
		return nil
	}

	signer := &model.Signer{}
	result.Details.AddSigner(signer)

	certs, crls, ocsps := dtsValidationMaterial(ctx, signer, result, p7.Certificates)

	if !p7.ContentType.Equal(oidTSTInfo) {
		result.Reason = model.SignatureReasonTimestampTokenInvalid
		signer.AddProblem("SubFilter ETSI.RFC3161: missing timestamp info")
		return nil
	}

	tstInfo, err := parseTSTInfo(p7.Content)
	if err != nil {
		result.Reason = model.SignatureReasonTimestampTokenInvalid
		signer.AddProblem(fmt.Sprintf("SubFilter ETSI.RFC3161: invalid timestamp info: %v", err))
		return nil
	}
	if !checkTSTInfoProfile(tstInfo, signer, result) {
		return nil
	}

	// ByteRange shall cover the entire document, including the Document Time-stamp dictionary
	// but excluding the TimeStampToken itself (the entry with key Contents).
	data, ok, err := readDTSSignedData(ra, sigDict, result)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	digestErr := verifyDTSDigest(tstInfo, data)

	p7Signer := p7.Signers[0]

	signerCert := dtsSignerCertificate(certs, p7Signer, signer, result)
	if signerCert == nil {
		return nil
	}
	localAssessment.CertificateIdentified = true

	if !authenticateDTSEvidence(p7, p7Signer, signerCert, digestErr, signer, result) {
		return nil
	}
	localAssessment.SignatureAuthenticated = true
	localAssessment.DigestVerified = true
	localAssessment.ProfileValidated = true

	return evaluateDTSTimestamp(
		sigDict,
		tstInfo,
		p7,
		p7Signer,
		data,
		signerCert,
		certs,
		rootCerts,
		crls,
		ocsps,
		signer,
		result,
		ctx,
		localAssessment,
	)
}

func authenticateDTSEvidence(
	p7 *pkcs7.PKCS7,
	p7Signer pkcs7.SignerInfo,
	signerCert *x509.Certificate,
	digestErr error,
	signer *model.Signer,
	result *model.SignatureValidationResult,
) bool {
	if !checkDTSSignedAttributes(p7Signer, signer, result) {
		return false
	}
	if err := pkcs7.CheckSignatureWithContentType(signerCert, p7Signer, p7.Content, p7.ContentType); err != nil {
		reportDTSSignatureError(err, signer, result)
		return false
	}
	if !applyDTSDigestEvidence(digestErr, signer, result) ||
		!checkTimestampingEKU(signerCert, signer, result) ||
		!checkTimestampSigningCertificate(p7Signer, signerCert, signer, result) {
		return false
	}
	markDocumentUnmodified(result)
	return true
}

func evaluateDTSTimestamp(
	sigDict types.Dict,
	tstInfo *TSTInfo,
	p7 *pkcs7.PKCS7,
	p7Signer pkcs7.SignerInfo,
	data []byte,
	signerCert *x509.Certificate,
	certs []*x509.Certificate,
	rootCerts *x509.CertPool,
	crls, ocsps [][]byte,
	signer *model.Signer,
	result *model.SignatureValidationResult,
	ctx *model.Context,
	localAssessment localSignatureAssessment,
) error {
	rawToken, err := signatureContents(sigDict)
	if err != nil {
		return fmt.Errorf("SubFilter ETSI.RFC3161: read timestamp token: %w", err)
	}

	preparedEvidence := preparedDocumentTimestampEvidence(
		tstInfo,
		rawToken,
		p7,
		p7Signer,
		data,
	)

	preparedEvidence.CorrectProfile = true

	// Assess the TSA certificate using the configured local certificate sources.
	pathValidated, err := validateDTSCert(
		signerCert,
		certs,
		rootCerts,
		crls,
		ocsps,
		signer,
		result,
		ctx,
	)

	if err != nil {
		return fmt.Errorf("timestamp certificate validation: %w", err)
	}

	preparedEvidence.LocalTSAPathValidated = pathValidated
	localAssessment.PathValidated = pathValidated
	localAssessment.RevocationGood = signer.Certificate != nil && signer.Certificate.Revocation.Status == model.True

	applyTimestampEvidence(
		preparedEvidence,
		timestampApplication{
			signer:               signer,
			result:               result,
			setResultSigningTime: true,
			problemPrefix:        "SubFilter ETSI.RFC3161: evaluate timestamp",
		},
	)

	finalizeDTSValidationResult(result, ctx, preparedEvidence, localAssessment)

	return nil
}

func dtsValidationMaterial(
	ctx *model.Context,
	signer *model.Signer,
	result *model.SignatureValidationResult,
	certs []*x509.Certificate,
) ([]*x509.Certificate, [][]byte, [][]byte) {
	if len(ctx.DSS) == 0 {
		return certs, nil, nil
	}
	evidence := processDSS(ctx, signer)
	if !evidence.Supported {
		markUnsupportedEvidence(result)
	}
	return mergeCerts(certs, evidence.Certificates), evidence.CRLs, evidence.OCSPResponses
}

func readDTSSignedData(
	ra io.ReaderAt,
	sigDict types.Dict,
	result *model.SignatureValidationResult,
) ([]byte, bool, error) {
	data, err := signedData(ra, sigDict)
	if err == nil {
		return data, true, nil
	}
	if !errors.Is(err, errMalformedByteRange) {
		return nil, false, fmt.Errorf("SubFilter ETSI.RFC3161: read signed data: %w", err)
	}
	markMalformedEvidence(result)
	result.AddProblem(fmt.Sprintf("SubFilter ETSI.RFC3161: read signed data: %v", err))
	return nil, false, nil
}

func dtsSignerCertificate(
	certs []*x509.Certificate,
	p7Signer pkcs7.SignerInfo,
	signer *model.Signer,
	result *model.SignatureValidationResult,
) *x509.Certificate {
	cert, err := pkcs7.GetCertFromCertsByIssuerAndSerial(certs, p7Signer.IssuerAndSerialNumber)
	if err != nil {
		markCertificateInvalidEvidence(result)
		signer.AddProblem(fmt.Sprintf("SubFilter ETSI.RFC3161: signer identifier: %v", err))
		return nil
	}
	if cert == nil {
		markCertificateInvalidEvidence(result)
		signer.AddProblem("SubFilter ETSI.RFC3161: missing certificate for signer")
	}
	return cert
}

func reportDTSSignatureError(err error, signer *model.Signer, result *model.SignatureValidationResult) {
	reportSignatureVerificationError(
		"SubFilter ETSI.RFC3161: verify signature",
		err,
		model.SignatureReasonTimestampTokenInvalid,
		signer,
		result,
	)
}

func checkTSTInfoProfile(tstInfo *TSTInfo, signer *model.Signer, result *model.SignatureValidationResult) bool {
	if tstInfo == nil {
		result.Reason = model.SignatureReasonTimestampTokenInvalid
		signer.AddProblem("SubFilter ETSI.RFC3161: timestamp info missing")
		return false
	}
	if tstInfo.Version != 1 {
		markUnsupportedEvidence(result)
		signer.AddProblem(fmt.Sprintf(
			"SubFilter ETSI.RFC3161: unsupported timestamp info version %d",
			tstInfo.Version,
		))
		return false
	}
	if tstInfo.GenTime.IsZero() {
		result.Reason = model.SignatureReasonTimestampTokenInvalid
		signer.AddProblem("SubFilter ETSI.RFC3161: timestamp info genTime missing")
		return false
	}
	return true
}

func checkDTSSignedAttributes(p7Signer pkcs7.SignerInfo, signer *model.Signer, result *model.SignatureValidationResult) bool {
	if len(p7Signer.AuthenticatedAttributes) > 0 {
		return true
	}
	markUnsupportedEvidence(result)
	signer.AddProblem("SubFilter ETSI.RFC3161: unsupported CMS profile: signed attributes missing")
	return false
}

func checkTimestampingEKU(cert *x509.Certificate, signer *model.Signer, result *model.SignatureValidationResult) bool {
	if err := validateTimestampingEKU(cert); err != nil {
		markUnsupportedEvidence(result)
		signer.AddProblem(fmt.Sprintf("SubFilter ETSI.RFC3161: unsupported TSA certificate profile: %v", err))
		return false
	}
	return true
}

func validateTimestampingEKU(cert *x509.Certificate) error {
	if cert == nil {
		return errors.New("certificate missing")
	}
	if !slices.Contains(cert.ExtKeyUsage, x509.ExtKeyUsageTimeStamping) {
		return errors.New("timestamping EKU missing")
	}
	if len(cert.ExtKeyUsage) != 1 || len(cert.UnknownExtKeyUsage) != 0 {
		return errors.New("timestamping EKU must be exclusive")
	}
	for _, extension := range cert.Extensions {
		if extension.Id.Equal(oidExtensionExtendedKeyUsage) {
			if !extension.Critical {
				return errors.New("timestamping EKU extension must be critical")
			}
			return nil
		}
	}
	return errors.New("timestamping EKU extension missing")
}

func checkTimestampSigningCertificate(
	p7Signer pkcs7.SignerInfo,
	cert *x509.Certificate,
	signer *model.Signer,
	result *model.SignatureValidationResult,
) bool {
	err := validateESSCertificateBinding(p7Signer, cert)
	if err == nil {
		return true
	}
	switch {
	case errors.Is(err, errESSCertificateMismatch):
		markInvalidEvidence(result, model.SignatureReasonTimestampTokenInvalid, model.Unknown)
	case errors.Is(err, pkcs7.ErrUnsupportedAlgorithm),
		errors.Is(err, errUnsupportedESSCertificateProfile):
		markUnsupportedEvidence(result)
	default:
		markMalformedEvidence(result)
	}
	signer.AddProblem(fmt.Sprintf("SubFilter ETSI.RFC3161: authenticate TSA certificate: %v", err))
	return false
}

func validateESSCertificateBinding(p7Signer pkcs7.SignerInfo, cert *x509.Certificate) error {
	oid, value, err := essSigningCertificateAttribute(p7Signer)
	if err != nil {
		return err
	}
	hash, certHash, issuerSerial, err := decodeESSCertificateBinding(oid, value)
	if err != nil {
		return err
	}
	if err := matchESSCertificateHash(hash, certHash, cert); err != nil {
		return err
	}
	if err := matchESSCertificateIssuerSerial(issuerSerial, cert); err != nil {
		return err
	}
	return validateESSCertificateKeyUsage(cert)
}

func essSigningCertificateAttribute(p7Signer pkcs7.SignerInfo) (asn1.ObjectIdentifier, asn1.RawValue, error) {
	var (
		oid   asn1.ObjectIdentifier
		value asn1.RawValue
		count int
	)
	for _, attr := range p7Signer.AuthenticatedAttributes {
		if !attr.Type.Equal(oidSigningCertificate) &&
			!attr.Type.Equal(oidSigningCertificateV2) {
			continue
		}
		count++
		oid = attr.Type
		value = attr.Value
	}
	if count != 1 {
		return nil, asn1.RawValue{}, fmt.Errorf(
			"%w: expected exactly one SigningCertificate or SigningCertificateV2 attribute, got %d",
			pkcs7.ErrMalformedAttribute,
			count,
		)
	}
	return oid, value, nil
}

func decodeESSCertificateBinding(oid asn1.ObjectIdentifier, value asn1.RawValue) (crypto.Hash, []byte, asn1.RawValue, error) {
	if oid.Equal(oidSigningCertificate) {
		var attribute signingCertificate
		if err := unmarshalESSCertificateAttribute(value, oid, &attribute); err != nil {
			return 0, nil, asn1.RawValue{}, err
		}
		if len(attribute.Certs) == 0 {
			return 0, nil, asn1.RawValue{}, missingESSCertificateID(oid)
		}
		return crypto.SHA1, attribute.Certs[0].CertHash, attribute.Certs[0].IssuerSerial, nil
	}

	var attribute rawSigningCertificateV2
	if err := unmarshalESSCertificateAttribute(value, oid, &attribute); err != nil {
		return 0, nil, asn1.RawValue{}, err
	}
	if len(attribute.Certs) == 0 {
		return 0, nil, asn1.RawValue{}, missingESSCertificateID(oid)
	}
	certID, err := parseESSCertIDv2(attribute.Certs[0])
	if err != nil {
		return 0, nil, asn1.RawValue{}, err
	}
	hash, err := essCertIDv2Hash(certID.HashAlgorithm)
	if err != nil {
		return 0, nil, asn1.RawValue{}, err
	}
	return hash, certID.CertHash, certID.IssuerSerial, nil
}

func parseESSCertIDv2(raw asn1.RawValue) (essCertIDv2, error) {
	if raw.Class != asn1.ClassUniversal || raw.Tag != asn1.TagSequence || !raw.IsCompound {
		return essCertIDv2{}, malformedESSCertIDv2("certificate identifier is not a sequence")
	}
	first, rest, err := unmarshalRawValue(raw.Bytes)
	if err != nil {
		return essCertIDv2{}, err
	}
	certID := essCertIDv2{}
	if first.Class == asn1.ClassUniversal && first.Tag == asn1.TagSequence {
		certID.HashAlgorithm = first
		first, rest, err = unmarshalRawValue(rest)
		if err != nil {
			return essCertIDv2{}, err
		}
	}
	if first.Class != asn1.ClassUniversal || first.Tag != asn1.TagOctetString {
		return essCertIDv2{}, malformedESSCertIDv2("certificate hash is not an octet string")
	}
	certID.CertHash = append([]byte(nil), first.Bytes...)
	if len(rest) == 0 {
		return certID, nil
	}
	certID.IssuerSerial, rest, err = unmarshalRawValue(rest)
	if err != nil {
		return essCertIDv2{}, err
	}
	if len(rest) > 0 {
		return essCertIDv2{}, malformedESSCertIDv2("trailing certificate identifier data")
	}
	return certID, nil
}

func unmarshalRawValue(bb []byte) (asn1.RawValue, []byte, error) {
	var raw asn1.RawValue
	rest, err := asn1.Unmarshal(bb, &raw)
	if err != nil {
		return asn1.RawValue{}, nil, fmt.Errorf(
			"%w, OID %s: decode certificate identifier: %w",
			pkcs7.ErrMalformedAttribute,
			oidSigningCertificateV2,
			err,
		)
	}
	return raw, rest, nil
}

func malformedESSCertIDv2(detail string) error {
	return fmt.Errorf("%w, OID %s: %s", pkcs7.ErrMalformedAttribute, oidSigningCertificateV2, detail)
}

func unmarshalESSCertificateAttribute(value asn1.RawValue, oid asn1.ObjectIdentifier, out any) error {
	rest, err := asn1.Unmarshal(value.Bytes, out)
	if err != nil {
		return fmt.Errorf("%w, OID %s: decode ASN.1: %w", pkcs7.ErrMalformedAttribute, oid, err)
	}
	if len(rest) > 0 {
		err := asn1.SyntaxError{Msg: "trailing data after attribute value"}
		return fmt.Errorf("%w, OID %s: decode ASN.1: %w", pkcs7.ErrMalformedAttribute, oid, err)
	}
	return nil
}

func missingESSCertificateID(oid asn1.ObjectIdentifier) error {
	return fmt.Errorf("%w, OID %s: certificate identifier missing", pkcs7.ErrMalformedAttribute, oid)
}

func essCertIDv2Hash(raw asn1.RawValue) (crypto.Hash, error) {
	if len(raw.FullBytes) == 0 {
		return crypto.SHA256, nil
	}
	var algorithm pkix.AlgorithmIdentifier
	rest, err := asn1.Unmarshal(raw.FullBytes, &algorithm)
	if err != nil {
		return 0, fmt.Errorf(
			"%w, OID %s: decode hash algorithm: %w",
			pkcs7.ErrMalformedAttribute,
			oidSigningCertificateV2,
			err,
		)
	}
	if len(rest) > 0 || len(algorithm.Algorithm) == 0 {
		err := asn1.SyntaxError{Msg: "invalid ESSCertIDv2 hash algorithm"}
		return 0, fmt.Errorf("%w, OID %s: %w", pkcs7.ErrMalformedAttribute, oidSigningCertificateV2, err)
	}
	hash, err := pkcs7.HashForOID(algorithm.Algorithm)
	if err != nil {
		return 0, fmt.Errorf("ESSCertIDv2 hash algorithm: %w", err)
	}
	return hash, nil
}

func matchESSCertificateHash(hash crypto.Hash, want []byte, cert *x509.Certificate) error {
	if cert == nil || len(cert.Raw) == 0 {
		return fmt.Errorf("%w: signer certificate DER unavailable", pkcs7.ErrMalformedAttribute)
	}
	if len(want) == 0 {
		return fmt.Errorf("%w: ESS certificate hash missing", pkcs7.ErrMalformedAttribute)
	}
	if !hash.Available() {
		return fmt.Errorf("%w: digest %s unavailable", pkcs7.ErrUnsupportedAlgorithm, hash)
	}
	digest := hash.New()
	if _, err := digest.Write(cert.Raw); err != nil {
		return fmt.Errorf("ESS: hash signer certificate: %w", err)
	}
	if subtle.ConstantTimeCompare(digest.Sum(nil), want) != 1 {
		return fmt.Errorf("%w: certificate hash does not match signer certificate", errESSCertificateMismatch)
	}
	return nil
}

func matchESSCertificateIssuerSerial(raw asn1.RawValue, cert *x509.Certificate) error {
	if len(raw.FullBytes) == 0 {
		return nil
	}
	var issuerSerial essIssuerSerial
	rest, err := asn1.Unmarshal(raw.FullBytes, &issuerSerial)
	if err != nil {
		return fmt.Errorf("%w: decode ESS issuer and serial: %w", pkcs7.ErrMalformedAttribute, err)
	}
	if len(rest) > 0 || issuerSerial.SerialNumber == nil || len(issuerSerial.Issuer) == 0 {
		err := asn1.SyntaxError{Msg: "incomplete ESS issuer and serial"}
		return fmt.Errorf("%w: %w", pkcs7.ErrMalformedAttribute, err)
	}
	if cert == nil || cert.SerialNumber == nil ||
		issuerSerial.SerialNumber.Cmp(cert.SerialNumber) != 0 {
		return fmt.Errorf("%w: certificate serial number does not match signer certificate", errESSCertificateMismatch)
	}
	for _, name := range issuerSerial.Issuer {
		if name.Class == asn1.ClassContextSpecific &&
			name.Tag == 4 &&
			bytes.Equal(name.Bytes, cert.RawIssuer) {
			return nil
		}
	}
	return fmt.Errorf("%w: certificate issuer does not match signer certificate", errESSCertificateMismatch)
}

func validateESSCertificateKeyUsage(cert *x509.Certificate) error {
	if cert == nil {
		return fmt.Errorf("%w: certificate missing", pkcs7.ErrMalformedAttribute)
	}
	for _, extension := range cert.Extensions {
		if !extension.Id.Equal(oidExtensionKeyUsage) {
			continue
		}
		if cert.KeyUsage&(x509.KeyUsageDigitalSignature|x509.KeyUsageContentCommitment) == 0 {
			return fmt.Errorf(
				"%w: key usage does not permit signature creation",
				errUnsupportedESSCertificateProfile,
			)
		}
		return nil
	}
	return nil
}

func parseTSTInfo(bb []byte) (*TSTInfo, error) {
	var tstInfo TSTInfo
	rest, err := asn1.Unmarshal(bb, &tstInfo)
	if err != nil {
		return nil, fmt.Errorf("timestamp info: unmarshal ASN.1: %w", err)
	}
	if len(rest) > 0 {
		return nil, errors.New("timestamp info: trailing data")
	}
	return &tstInfo, nil
}

func validateDTSCert(
	signerCert *x509.Certificate,
	certs []*x509.Certificate,
	rootCerts *x509.CertPool,
	crls, ocsps [][]byte,
	signer *model.Signer,
	result *model.SignatureValidationResult,
	ctx *model.Context,
) (bool, error) {
	// Collect certificate-path evidence using the configured local certificate sources.
	chains := buildP7CertChains(true, signerCert, certs, rootCerts, signer, result)
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
		return false, fmt.Errorf("DTS: assess certificate evidence: %w", err)
	}
	applyCertificateAssessment(assessment, signer, result)
	return isLocallyValidatedTSAPathAssessment(assessment), nil
}

func isLocallyValidatedTSAPathAssessment(assessment certificateAssessment) bool {
	return assessment.Reason == model.SignatureReasonUnknown &&
		assessment.Certificate != nil &&
		assessment.Certificate.Trust.Status == model.True
}

func verifyDTSDigest(tstInfo *TSTInfo, data []byte) error {
	return pkcs7.VerifyMessageDigestTSToken(
		tstInfo.MessageImprint.HashAlgorithm.Algorithm,
		tstInfo.MessageImprint.HashedMessage,
		data,
	)
}

func applyDTSDigestEvidence(err error, signer *model.Signer, result *model.SignatureValidationResult) bool {
	if err != nil {
		var mdErr *pkcs7.MessageDigestMismatchError
		if errors.As(err, &mdErr) {
			markInvalidEvidence(result, model.SignatureReasonDocModified, model.True)
			signer.AddProblem(fmt.Sprintf("SubFilter ETSI.RFC3161: message digest mismatch: %v", err))
			return false
		}
		markUnsupportedEvidence(result)
		signer.AddProblem(fmt.Sprintf("SubFilter ETSI.RFC3161: verify message digest: %v", err))
		return false
	}
	return true
}

func collectIntermediates(signerCert *x509.Certificate, certs []*x509.Certificate) []*x509.Certificate {
	var intermediates []*x509.Certificate
	for _, cert := range certs {
		if cert == nil {
			intermediates = append(intermediates, nil)
			continue
		}
		if !cert.Equal(signerCert) {
			intermediates = append(intermediates, cert)
		}
	}
	return intermediates
}

// finalizeDTSValidationResult concludes DTS validation only for
// cryptographically authenticated and locally validated evidence.
func finalizeDTSValidationResult(result *model.SignatureValidationResult, ctx *model.Context, evidence timestampEvidence, assessment localSignatureAssessment) {
	if isLocallyValidatedDocumentTimestampEvidence(evidence) &&
		finalizeLocalSignatureResult(result, assessment) {
		ctx.DTS = evidence.SigningTime
	} else {
		ctx.DTS = time.Time{}
	}
}
