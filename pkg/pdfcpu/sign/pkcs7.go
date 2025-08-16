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
	"fmt"
	"io"
	"time"

	"github.com/hhrutter/pkcs7"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// ValidatePKCS7Signature validates contained signatures using subFilter adbe.pkcs7.sha1, adbe.pkcs7.detached and ETSI.CAdES.detached.
func ValidatePKCS7Signatures(
	ra io.ReaderAt,
	sigDict types.Dict,
	certified bool,
	authoritative bool,
	validateAll bool,
	perms int,
	rootCerts *x509.CertPool,
	result *model.SignatureValidationResult,
	ctx *model.Context) error {

	if ctx.Configuration.Offline {
		result.AddProblem("pdfcpu is offline, unable to perform certificate revocation checking")
	}

	p7 := validateP7(sigDict, result)
	if p7 == nil {
		return nil
	}

	data, err := signedData(ra, sigDict)
	if err != nil {
		result.Reason = model.SignatureReasonInternal
		result.AddProblem(fmt.Sprintf("unmarshal asn1 content: %v", err))
		return nil
	}

	detached := len(p7.Content) == 0
	if detached {
		p7.Content = data
	}

	for i, p7Signer := range p7.Signers {
		verifyP7Signer(p7Signer, p7.Certificates, rootCerts, p7.Content, data, detached, certified, authoritative, perms, i, result, ctx)
		if (certified || authoritative) && !validateAll {
			break
		}
	}

	finalizePKCS7Result(result)

	return nil
}

func finalizePKCS7Result(result *model.SignatureValidationResult) {
	if result.Status == model.SignatureStatusUnknown && result.Reason == model.SignatureReasonUnknown {
		result.Status = model.SignatureStatusValid
		result.Reason = model.SignatureReasonDocNotModified
	} else {
		// Show PAdES basic level for valid signatures only.
		if len(result.Details.Signers) > 0 {
			result.Details.Signers[0].PAdES = ""
		}
	}
}

func p7(sigDict types.Dict) (*pkcs7.PKCS7, error) {
	hl := sigDict.HexLiteralEntry("Contents")
	if hl == nil {
		return nil, errors.New("invalid signature dict - missing \"Contents\"")
	}

	signature, err := hl.Bytes()
	if err != nil {
		return nil, errors.Errorf("invalid content data: %v", err)
	}

	p7, err := pkcs7.Parse(signature)
	if err != nil {
		return nil, errors.Errorf("failed to parse PKCS#7: %v", err)
	}

	return p7, nil
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
	ctx *model.Context) {

	conf := ctx.Configuration

	signer := &model.Signer{}
	result.Details.AddSigner(signer)

	signer.Certified = certified
	signer.Authoritative = signer.Certified || authoritative
	signer.Permissions = perms

	checkPerms(signer, result)

	if ok := checkP7Digest(p7Signer, p7Content, data, detached, signer, result); !ok {
		return
	}

	if result.Status == model.SignatureStatusUnknown {
		if result.DocModified == model.Unknown {
			result.DocModified = model.False
		}
	}

	signerCert := pkcs7.GetCertFromCertsByIssuerAndSerial(p7Certs, p7Signer.IssuerAndSerialNumber)
	if signerCert == nil {
		result.Reason = model.SignatureReasonInternal
		signer.AddProblem(fmt.Sprintf("pkcs7: missing certificate for signer %d", i+1))
		return
	}

	if err := verifyP7Signature(p7Signer, signerCert, p7Content, detached); err != nil {
		if result.Status == model.SignatureStatusUnknown {
			result.Status = model.SignatureStatusInvalid
			result.Reason = model.SignatureReasonSignatureForged
		}
		signer.AddProblem(fmt.Sprintf("pkcs7: signature verification failure: %v\n", err))
		return
	}

	// Signature is authenticated and the signer is who they claim to be.

	if detached {
		signer.PAdES = "B-B"
	}

	// Process optional DSS and DTS for embedded revocation info and trusted timestamp.
	// This may upgrade PAdES level to B-T, B-LT, B-LTA respectively.

	// Calculate the signingTime we use for validation.
	// Use either a present timestamp token or document timestamp.
	// Fallback to claimed signingTime and in absence to time.Now().

	// TODO Handle oidArchiveTimestamp

	var signingTime *time.Time

	signingTime = handleClaimedSigningTime(p7Signer, signer, result)

	if !ctx.DTS.IsZero() {
		if result.Details.SigningTime.After(ctx.DTS) {
			signer.AddProblem(fmt.Sprintf("Claimed signing time: %s is not before document timestamp: %s",
				result.Details.SigningTime.Format(conf.TimestampFormat),
				ctx.DTS.Format(conf.TimestampFormat)))
		}
	}

	if ts := checkTimestampToken(detached, p7Signer, rootCerts, ctx, signer, result); ts != nil {
		signingTime = ts
	}

	// Look for embedded revocation info.
	crls, ocsps := handleArchivedRevocationInfo(p7Signer, signer)

	certs := p7Certs

	handleDSS(&certs, &crls, &ocsps, ctx, signer, detached)

	// Does signerCert chain up to a trusted Root CA?
	chains := buildP7CertChains(i == 0, signerCert, certs, rootCerts, signer, signingTime, result)
	if len(chains) == 0 {
		chains = [][]*x509.Certificate{certChain(signerCert, certs)}
	}

	validateCertChains(chains, rootCerts, signer, signingTime, crls, ocsps, result, ctx.Configuration)
}

func checkPerms(signer *model.Signer, result *model.SignatureValidationResult) {
	if signer.Certified && signer.Permissions != model.CertifiedSigPermNoChangesAllowed {
		// TODO Check for violation of perm 2 and 3
		signer.AddProblem(CertifiedSigPermsNotSupported)
		result.Reason = model.SignatureReasonInternal
	}
}

func checkP7Digest(
	p7Signer pkcs7.SignerInfo,
	p7Content,
	data []byte, detached bool,
	signer *model.Signer,
	result *model.SignatureValidationResult) bool {

	reason, err := verifyP7Digest(p7Signer, p7Content, data, detached)
	if err == nil {
		return true
	}

	if result.Status == model.SignatureStatusUnknown {
		if reason == model.SignatureReasonDocModified {
			// Document has been modified since time of signing.
			result.Status = model.SignatureStatusInvalid
			result.Reason = model.SignatureReasonDocModified
			result.DocModified = model.True
		}
		if reason == model.SignatureReasonInternal {
			//result.Status = model.SignatureStatusInvalid
			result.Reason = model.SignatureReasonInternal
		}
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
			return model.SignatureReasonInternal, errors.New("pkcs7: missing authenticated attributes")
		}

		if err := pkcs7.VerifyMessageDigestDetached(p7Signer, p7Content); err != nil {
			var mdErr *pkcs7.MessageDigestMismatchError
			if errors.As(err, &mdErr) {
				return model.SignatureReasonDocModified, errors.Errorf("pkcs7: message digest verification failure: %v\n", err)
			}
			return model.SignatureReasonInternal, errors.Errorf("pkcs7: message digest verification: %v\n", err)
		}

	} else {

		if err := pkcs7.VerifyMessageDigestEmbedded(p7Content, data); err != nil {
			return model.SignatureReasonDocModified, errors.Errorf("pkcs7: message digest verification failure: %v\n", err)
		}

	}

	return model.SignatureReasonDocNotModified, nil
}

func checkTimestampToken(
	detached bool,
	p7Signer pkcs7.SignerInfo,
	rootCerts *x509.CertPool,
	ctx *model.Context,
	signer *model.Signer,
	result *model.SignatureValidationResult) (signingTime *time.Time) {

	token := handleTimestampToken(p7Signer, rootCerts, signer, result)

	if token != nil {
		signingTime = token
		signer.HasTimestamp = true
		signer.Timestamp = *token
		if detached {
			signer.PAdES = "B-T"
		}
	} else if !ctx.DTS.IsZero() {
		signingTime = &ctx.DTS
		signer.HasTimestamp = true
		signer.Timestamp = ctx.DTS
		if detached {
			signer.PAdES = "B-T"
		}
	}

	return signingTime
}

func handleDSS(certs *[]*x509.Certificate, crls *[][]byte, ocsps *[][]byte, ctx *model.Context, signer *model.Signer, detached bool) {
	if len(ctx.DSS) > 0 {
		if dssCerts, dssCRLs, dssOCSPs, ok := processDSS(ctx, signer); ok {
			*certs = mergeCerts(*certs, dssCerts)
			if len(dssCRLs) > 0 {
				*crls = dssCRLs
			}
			if len(dssOCSPs) > 0 {
				*ocsps = dssOCSPs
			}
			if detached && signer.PAdES == "B-T" {
				signer.PAdES = "B-LT"
			}
			signer.LTVEnabled = true
		}
	}

	if signer.PAdES == "B-LT" && !ctx.DTS.IsZero() {
		signer.PAdES = "B-LTA"
	}
}

func verifyP7Signature(p7Signer pkcs7.SignerInfo, cert *x509.Certificate, p7Content []byte, detached bool) error {
	// Verify signature against expected hash using the public key.
	// Ensures integrity and authenticity of the signature itself.
	// Confirms the signer is who they claim to be.

	var content []byte
	if !detached {
		content = p7Content
	}
	return pkcs7.CheckSignature(cert, p7Signer, content)
}

func handleClaimedSigningTime(signerInfo pkcs7.SignerInfo, signer *model.Signer, result *model.SignatureValidationResult) *time.Time {
	var (
		err         error
		signingTime time.Time
	)

	for _, attr := range signerInfo.AuthenticatedAttributes {
		if attr.Type.Equal(oidSigningTime) {
			_, err = asn1.Unmarshal(attr.Value.Bytes, &signingTime)
			break
		}
	}

	if err != nil {
		signer.AddProblem(fmt.Sprintf("invalid signing time: %v", err))
		if result.Status == model.SignatureStatusUnknown {
			result.Reason = model.SignatureReasonSigningTimeInvalid
		}
		return nil
	}

	if !signingTime.IsZero() {
		result.Details.SigningTime = signingTime
		return &signingTime
	}

	return nil
}

func timestampToken(p7Signer pkcs7.SignerInfo, rootCerts *x509.CertPool) (time.Time, error) {
	// A trusted timestamp token aka trusted signing time.
	if bb := locateTimestampToken(p7Signer); len(bb) > 0 {
		return validateTimestampToken(bb, rootCerts)
	}
	return time.Time{}, nil
}

func locateTimestampToken(signerInfo pkcs7.SignerInfo) []byte {
	for _, attr := range signerInfo.UnauthenticatedAttributes {
		if attr.Type.Equal(oidTimestampToken) {
			return attr.Value.Bytes
		}
	}
	return nil
}

func validateTimestampToken(data []byte, rootCAs *x509.CertPool) (time.Time, error) {
	var defTime time.Time
	p7, err := pkcs7.Parse(data)
	if err != nil {
		return defTime, errors.Errorf("failed to parse timestamp token: %v", err)
	}

	if len(p7.Signers) != 1 {
		return defTime, errors.Errorf("malformed timestamp token")
	}
	signer := p7.Signers[0]

	// if err := p7.VerifyWithChain(rootCAs); err != nil {
	// 	return defTime, errors.Errorf("timestamp token signature verification failed: %v", err)
	// }

	for _, attr := range signer.AuthenticatedAttributes {
		if attr.Type.Equal(oidSigningTime) {
			var rawValue asn1.RawValue
			if _, err := asn1.Unmarshal(attr.Value.Bytes, &rawValue); err != nil {
				return defTime, errors.Errorf("failed to unmarshal signing time: %v", err)
			}
			if rawValue.Tag == asn1.TagUTCTime {
				return time.Parse("060102150405Z", string(rawValue.Bytes))
			}
			if rawValue.Tag == asn1.TagGeneralizedTime {
				return time.Parse("20060102150405Z", string(rawValue.Bytes))
			}
			return defTime, errors.Errorf("unexpected tag for signing time: %d", rawValue.Tag)
		}
	}

	return defTime, errors.New("unable to resolve timestamp info")
}

func handleArchivedRevocationInfo(p7Signer pkcs7.SignerInfo, signer *model.Signer) (crls [][]byte, ocsps [][]byte) {
	if !signer.HasTimestamp {
		return nil, nil
	}
	ria, err := revocationInfoArchival(p7Signer)
	if err != nil {
		signer.LTVEnabled = true
		signer.AddProblem(fmt.Sprintf("revocationInfoArchival extraction failed: %v", err))
	}
	if ria == nil {
		return nil, nil
	}

	signer.LTVEnabled = true

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
	signingTime *time.Time,
	result *model.SignatureValidationResult) [][]*x509.Certificate {

	currentTime := time.Now()
	if signingTime != nil {
		currentTime = *signingTime
	}

	intermediates := collectIntermediates(cert, certs)
	chains, err := pkcs7.VerifyCertChain(cert, intermediates, rootCerts, currentTime)
	if err != nil {
		handleCertVerifyErr(err, cert, signer, result)
		return nil
	}
	if first {
		result.Details.SignerIdentity = cert.Subject.CommonName
	}
	return chains
}

func handleTimestampToken(p7Signer pkcs7.SignerInfo, rootCerts *x509.CertPool, signer *model.Signer, result *model.SignatureValidationResult) *time.Time {
	ts, err := timestampToken(p7Signer, rootCerts)
	if err != nil {
		signer.HasTimestamp = true
		signer.AddProblem(fmt.Sprintf("invalid TimestampToken: %v", err))
		if result.Status == model.SignatureStatusUnknown {
			result.Reason = model.SignatureReasonTimestampTokenInvalid
		}
	} else if !ts.IsZero() {
		signer.HasTimestamp = true
		signer.Timestamp = ts
		return &ts
	}
	return nil
}

func revocationInfoArchival(p7Signer pkcs7.SignerInfo) (*RevocationInfoArchival, error) {
	for _, attr := range p7Signer.AuthenticatedAttributes {
		if attr.Type.Equal(oidRevocationInfoArchival) {
			var ria RevocationInfoArchival
			_, err := asn1.Unmarshal(attr.Value.Bytes, &ria)
			return &ria, err
		}
	}
	return nil, nil
}
