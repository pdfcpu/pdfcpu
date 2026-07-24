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
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// ValidateX509RSASHA1Signature reports observed signature, certificate,
// timestamp and revocation evidence together with a local assessment for
// SubFilter adbe.x509.rsa_sha1.
func ValidateX509RSASHA1Signature(
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
	if ctx.Configuration.Offline {
		result.AddProblem("pdfcpu is offline, unable to perform certificate revocation checking")
	}

	signer := &model.Signer{}
	result.Details.AddSigner(signer)

	signer.Certified = certified
	signer.Authoritative = signer.Certified || authoritative
	signer.Permissions = perms

	if signer.Certified && signer.Permissions != model.CertifiedSigPermNoChangesAllowed {
		// TODO Check for violation of perm 2 and 3
		result.AddProblem(CertifiedSigPermsNotSupported)
		markUnsupportedEvidence(result)
	}

	p1Certs, err := parseP1Certificates(sigDict)
	if err != nil {
		markCertificateInvalidEvidence(result)
		if errors.Is(err, errCertificateParse) {
			handleCertParseErr(err, result)
		}
		result.AddProblem(fmt.Sprintf("legacy certificate: %v", err))
		result.AddProblem("skipped certificate revocation check")
		return nil
	}

	cert, rsaPubKey, err := p1SigningCertificate(p1Certs)
	if err != nil {
		result.Reason = model.SignatureReasonCertInvalid
		result.AddProblem(err.Error())
		result.AddProblem("skipped certificate revocation check")
		return nil
	}
	localAssessment.CertificateIdentified = true

	reason, err := verifyRSASHA1Signature(ra, sigDict, rsaPubKey)
	if err != nil {
		return handleP1VerificationError(reason, err, result)
	}
	localAssessment.SignatureAuthenticated = true

	if reason == model.SignatureReasonDocNotModified {
		localAssessment.DigestVerified = true
		markDocumentUnmodified(result)
	}
	localAssessment.ProfileValidated = true

	// The signature verifies with the public key in the identified certificate.
	// Document has not been modified since time of signing.

	// Collect certificate-path evidence using the configured local certificate sources.
	chains := buildP1CertChains(cert, p1Certs[1:], rootCerts, signer, result)
	pathResolved := len(chains) > 0

	if len(chains) == 0 {
		chains = [][]*x509.Certificate{certChain(cert, p1Certs)}
	}

	assessment, err := assessCertificateEvidence(
		chains,
		pathResolved,
		rootCerts,
		nil,
		nil,
		result.Reason,
		ctx.Configuration,
	)
	if err != nil {
		return fmt.Errorf("PKCS#1: assess certificate evidence: %w", err)
	}
	applyCertificateAssessment(assessment, signer, result)
	localAssessment.applyCertificateAssessment(assessment)

	finalizeLocalSignatureResult(result, localAssessment)

	return nil
}

func handleP1VerificationError(
	reason model.SignatureReason,
	err error,
	result *model.SignatureValidationResult,
) error {
	if isFatalByteRangeRead(err) {
		return err
	}
	if reason == model.SignatureReasonDocModified {
		// Signature is invalid and document has been modified.
		result.Status = model.SignatureStatusInvalid
		result.Reason = model.SignatureReasonSignatureForged
		result.DocModified = model.True
	}
	if reason == model.SignatureReasonMalformed ||
		reason == model.SignatureReasonUnsupported {
		setResultReason(result, reason)
	}
	result.AddProblem(fmt.Sprintf("%v", err))
	return nil
}

func isFatalByteRangeRead(err error) bool {
	var readErr *byteRangeReadError
	return errors.As(err, &readErr)
}

func p1SigningCertificate(certs []*x509.Certificate) (*x509.Certificate, *rsa.PublicKey, error) {
	if len(certs) == 0 {
		return nil, nil, errors.New("legacy certificate: signature dict entry Cert: empty array")
	}
	cert := certs[0]
	if cert == nil {
		return nil, nil, errors.New("legacy certificate: signature dict entry Cert, array index 1: missing certificate")
	}
	rsaPubKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, nil, fmt.Errorf(
			"legacy certificate: signature dict entry Cert, array index 1: SubFilter adbe.x509.rsa_sha1 requires an RSA public key, got %T",
			cert.PublicKey,
		)
	}
	if err := validateP1RSAPublicKey(rsaPubKey); err != nil {
		return nil, nil, fmt.Errorf(
			"legacy certificate: signature dict entry Cert, array index 1: invalid RSA public key: %w",
			err,
		)
	}
	if err := validateP1SigningKeyUsage(cert); err != nil {
		return nil, nil, fmt.Errorf(
			"legacy certificate: signature dict entry Cert, array index 1: invalid signing KeyUsage: %w",
			err,
		)
	}
	return cert, rsaPubKey, nil
}

func validateP1SigningKeyUsage(cert *x509.Certificate) error {
	for _, extension := range cert.Extensions {
		if !extension.Id.Equal(oidExtensionKeyUsage) {
			continue
		}
		if cert.KeyUsage&(x509.KeyUsageDigitalSignature|x509.KeyUsageContentCommitment) == 0 {
			return errors.New("does not permit signature creation")
		}
		return nil
	}
	return nil
}

func validateP1RSAPublicKey(key *rsa.PublicKey) error {
	if key == nil {
		return errors.New("missing key")
	}
	if key.N == nil {
		return errors.New("missing modulus")
	}
	if key.N.Sign() <= 0 {
		return errors.New("modulus must be positive")
	}
	if key.N.Bit(0) == 0 {
		return errors.New("modulus must be odd")
	}
	if key.E < 3 || key.E > 1<<31-1 || key.E%2 == 0 {
		return fmt.Errorf("invalid public exponent %d", key.E)
	}
	return nil
}

func parseP1Certificates(sigDict types.Dict) ([]*x509.Certificate, error) {
	obj, ok := sigDict.Find("Cert")
	if !ok {
		//  TODO Find certificate by other means.
		return nil, errors.New("signature dict entry Cert: missing")
	}

	var chain []*x509.Certificate

	switch obj := obj.(type) {
	case types.Array:
		for i, v := range obj {
			cert, err := certFromObj(v)
			if err != nil {
				return nil, fmt.Errorf("signature dict entry Cert, array index %d: parse certificate: %w", i+1, err)
			}
			chain = append(chain, cert)
		}

	case types.StringLiteral:
		cert, err := certFromStringLiteral(obj)
		if err != nil {
			return nil, fmt.Errorf("signature dict entry Cert: parse certificate: %w", err)
		}
		chain = append(chain, cert)

	case types.HexLiteral:
		cert, err := certFromHexLiteral(obj)
		if err != nil {
			return nil, fmt.Errorf("signature dict entry Cert: parse certificate: %w", err)
		}
		chain = append(chain, cert)

	default:
		return nil, fmt.Errorf("signature dict entry Cert: unsupported type %T", obj)
	}

	return chain, nil
}

func certFromObj(obj types.Object) (*x509.Certificate, error) {
	switch obj := obj.(type) {
	case types.StringLiteral:
		return certFromStringLiteral(obj)
	case types.HexLiteral:
		return certFromHexLiteral(obj)
	}
	return nil, fmt.Errorf("unsupported certificate object type %T", obj)
}

func certFromStringLiteral(obj types.StringLiteral) (*x509.Certificate, error) {
	bb, err := types.Unescape(obj.Value())
	if err != nil {
		return nil, fmt.Errorf("decode string literal: %w", err)
	}
	return parseCertificate(bb)
}

func certFromHexLiteral(obj types.HexLiteral) (*x509.Certificate, error) {
	bb, err := obj.Bytes()
	if err != nil {
		return nil, fmt.Errorf("decode hex literal: %w", err)
	}
	return parseCertificate(bb)
}

func verifyRSASHA1Signature(ra io.ReaderAt, sigDict types.Dict, rsaPubKey *rsa.PublicKey) (model.SignatureReason, error) {
	// Use public key from the signer's certificate to verify the RSA signature.
	// The signature itself is an RSA-encrypted SHA-1 hash of the signed data.
	hl := sigDict.HexLiteralEntry("Contents")
	if hl == nil {
		return model.SignatureReasonMalformed, errors.New("signature dict entry Contents: missing")
	}

	contents, err := hl.Bytes()
	if err != nil {
		return model.SignatureReasonMalformed, fmt.Errorf("signature dict entry Contents: decode: %w", err)
	}

	var bb []byte
	rest, err := asn1.Unmarshal(contents, &bb)
	if err != nil {
		return model.SignatureReasonMalformed, fmt.Errorf("signature dict entry Contents: unmarshal ASN.1: %w", err)
	}
	if len(rest) > 0 {
		err := asn1.SyntaxError{Msg: "trailing data"}
		return model.SignatureReasonMalformed, fmt.Errorf("signature dict entry Contents: unmarshal ASN.1: %w", err)
	}

	data, err := signedData(ra, sigDict)
	if err != nil {
		return model.SignatureReasonMalformed, fmt.Errorf("read signed data: %w", err)
	}

	// Combine hash calculation and signature verification.

	// Hash signed data (extracted using ByteRange) using SHA-1, 160 Bits = 20 bytes
	hashed := sha1.Sum(data)

	// Confirm that the signature was created using the private key corresponding to the public key from the certificate.
	if err := rsa.VerifyPKCS1v15(rsaPubKey, crypto.SHA1, hashed[:], bb); err != nil {
		return model.SignatureReasonDocModified, fmt.Errorf("SubFilter adbe.x509.rsa_sha1: verify signature: %w", err)
	}

	return model.SignatureReasonDocNotModified, nil
}

func buildP1CertChains(
	cert *x509.Certificate,
	certs []*x509.Certificate,
	rootCerts *x509.CertPool,
	signer *model.Signer,
	result *model.SignatureValidationResult) [][]*x509.Certificate {
	intermediates, err := p1IntermediatePool(certs)
	if err != nil {
		markCertificateInvalidEvidence(result)
		signer.AddProblem(err.Error())
		return nil
	}
	chains, err := cert.Verify(x509.VerifyOptions{
		Roots:         rootCerts,
		Intermediates: intermediates,
		CurrentTime:   time.Now(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})
	if err != nil {
		handleCertVerifyErr(err, cert, signer, result)
		return nil
	}

	result.Details.SignerIdentity = cert.Subject.CommonName

	return chains
}

func p1IntermediatePool(certs []*x509.Certificate) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	for i, cert := range certs {
		if cert == nil {
			return nil, fmt.Errorf(
				"legacy certificate: signature dict entry Cert, array index %d: missing certificate",
				i+2,
			)
		}
		pool.AddCert(cert)
	}
	return pool, nil
}
