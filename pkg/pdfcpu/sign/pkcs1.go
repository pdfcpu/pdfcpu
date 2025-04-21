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
	"fmt"
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// ValidateX509RSASHA1Signature validates signatures using subFilter adbe.x509.rsa_sha1.
func ValidateX509RSASHA1Signature(
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

	signer := &model.Signer{}
	result.Details.AddSigner(signer)

	signer.Certified = certified
	signer.Authoritative = signer.Certified || authoritative
	signer.Permissions = perms

	if signer.Certified && signer.Permissions != model.CertifiedSigPermNoChangesAllowed {
		// TODO Check for violation of perm 2 and 3
		result.AddProblem(CertifiedSigPermsNotSupported)
		result.Reason = model.SignatureReasonInternal
	}

	p1Certs, err := parseP1Certificates(sigDict)
	if err != nil {
		result.Reason = model.SignatureReasonCertNotTrusted
		result.AddProblem(fmt.Sprintf("cannot verify certificate %v", err))
		result.AddProblem("skipped certificate revocation check")
		return nil
	}

	cert := p1Certs[0]

	rsaPubKey := cert.PublicKey.(*rsa.PublicKey)
	reason, err := verifyRSASHA1Signature(ra, sigDict, rsaPubKey)
	if err != nil {
		if reason == model.SignatureReasonDocModified {
			// Signature is invalid and document has been modified.
			result.Status = model.SignatureStatusInvalid
			result.Reason = model.SignatureReasonSignatureForged
			result.DocModified = model.True
		}
		if reason == model.SignatureReasonInternal {
			result.Status = model.SignatureStatusInvalid
			result.Reason = model.SignatureReasonInternal
		}
		result.AddProblem(fmt.Sprintf("%v", err))
		return nil
	}

	if result.Reason == model.SignatureReasonDocNotModified {
		result.DocModified = model.False
	}

	// Signature is authenticated and the signer is who they claim to be.
	// Document has not been modified since time of signing.

	// Does cert chain up to a trusted Root CA?
	chains := buildP1CertChains(cert, rootCerts, signer, result)

	if len(chains) == 0 {
		chains = [][]*x509.Certificate{certChain(cert, p1Certs)}
	}

	validateCertChains(chains, rootCerts, signer, nil, nil, nil, result, ctx.Configuration)

	if result.Status == model.SignatureStatusUnknown && result.Reason == model.SignatureReasonUnknown {
		result.Status = model.SignatureStatusValid
		result.Reason = model.SignatureReasonDocNotModified
	}

	return nil
}

func parseP1Certificates(sigDict types.Dict) ([]*x509.Certificate, error) {
	obj, ok := sigDict.Find("Cert")
	if !ok {
		//  TODO Find certificate by other means.
		return nil, errors.New("pdfcpu: missing \"Cert\"")
	}

	var chain []*x509.Certificate

	switch obj := obj.(type) {
	case types.Array:
		for _, v := range obj {
			cert, err := certFromObj(v)
			if err != nil {
				return nil, err
			}
			chain = append(chain, cert)
		}

	case types.StringLiteral:
		cert, err := certFromStringLiteral(obj)
		if err != nil {
			return nil, err
		}
		chain = append(chain, cert)

	case types.HexLiteral:
		cert, err := certFromHexLiteral(obj)
		if err != nil {
			return nil, err
		}
		chain = append(chain, cert)

	default:
		return nil, errors.New("pdfcpu: invalid entry: \"Cert\"")
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
	return nil, errors.Errorf("unable to parse certificate for %T", obj)
}

func certFromStringLiteral(obj types.StringLiteral) (*x509.Certificate, error) {
	bb, err := types.Unescape(obj.Value())
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(bb)
}

func certFromHexLiteral(obj types.HexLiteral) (*x509.Certificate, error) {
	bb, err := obj.Bytes()
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(bb)
}

func verifyRSASHA1Signature(ra io.ReaderAt, sigDict types.Dict, rsaPubKey *rsa.PublicKey) (model.SignatureReason, error) {
	// Use public key from the signer's certificate to verify the RSA signature.
	// The signature itself is an RSA-encrypted SHA-1 hash of the signed data.
	hl := sigDict.HexLiteralEntry("Contents")
	if hl == nil {
		return model.SignatureReasonInternal, errors.New("invalid signature dict - missing \"Contents\"")
	}

	contents, err := hl.Bytes()
	if err != nil {
		return model.SignatureReasonInternal, errors.Errorf("invalid content data: %v", err)
	}

	var bb []byte
	if _, err = asn1.Unmarshal(contents, &bb); err != nil {
		return model.SignatureReasonInternal, errors.Errorf("unmarshal asn1 content: %v", err)
	}

	data, err := signedData(ra, sigDict)
	if err != nil {
		return model.SignatureReasonInternal, errors.Errorf("unmarshal asn1 content: %v", err)
	}

	// Combine hash calculation and signature verification.

	// Hash signed data (extracted using ByteRange) using SHA-1, 160 Bits = 20 bytes
	hashed := sha1.Sum(data)

	// Confirm that the signature was created using the private key corresponding to the public key from the certificate.
	if err := rsa.VerifyPKCS1v15(rsaPubKey, crypto.SHA1, hashed[:], bb); err != nil {
		return model.SignatureReasonDocModified, errors.Errorf("RSA PKCS#1v15 signature verification failure: %v\n", err)
	}

	return model.SignatureReasonDocNotModified, nil
}

func buildP1CertChains(
	cert *x509.Certificate,
	rootCerts *x509.CertPool,
	signer *model.Signer,
	result *model.SignatureValidationResult) [][]*x509.Certificate {

	chains, err := cert.Verify(x509.VerifyOptions{Roots: rootCerts})
	if err != nil {
		handleCertVerifyErr(err, cert, signer, result)
		return nil
	}

	result.Details.SignerIdentity = cert.Subject.CommonName

	return chains
}
