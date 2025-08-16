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

type AlgorithmIdentifier struct {
	Algorithm  asn1.ObjectIdentifier
	Parameters asn1.RawValue `asn1:"tag:0,optional"`
}

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

// ValidateDTS validates an ETSI.RFC3161 digital timestamp.
func ValidateDTS(
	ra io.ReaderAt,
	sigDict types.Dict,
	certified bool,
	authoritative bool,
	validateAll bool,
	perms int,
	rootCerts *x509.CertPool,
	result *model.SignatureValidationResult,
	ctx *model.Context) error {

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

	signer := &model.Signer{}
	result.Details.AddSigner(signer)

	certs := p7.Certificates

	var (
		dssCerts []*x509.Certificate
		crls     [][]byte
		ocsps    [][]byte
		ok       bool
	)

	if len(ctx.DSS) > 0 {
		if dssCerts, crls, ocsps, ok = processDSS(ctx, signer); ok {
			certs = mergeCerts(certs, dssCerts)
		}
	}

	if !p7.ContentType.Equal(oidTSTInfo) {
		signer.AddProblem("\"ETSI.RFC3161\": missing timestamp info")
		return nil
	}

	var tstInfo TSTInfo
	if _, err := asn1.Unmarshal(p7.Content, &tstInfo); err != nil {
		signer.AddProblem("\"ETSI.RFC3161\": invalid timestamp info")
		return nil
	}

	// TODO Check
	// ByteRange shall cover the entire document, including the Document Time-stamp dictionary
	// but excluding the TimeStampToken itself (the entry with key Contents).
	data, err := signedData(ra, sigDict)
	if err != nil {
		result.Reason = model.SignatureReasonInternal
		result.AddProblem(fmt.Sprintf("\"ETSI.RFC3161\": unmarshal asn1 content: %v", err))
		return nil
	}

	if ok := checkDTSDigest(&tstInfo, data, signer); !ok {
		return nil
	}

	if result.Status == model.SignatureStatusUnknown {
		if result.DocModified == model.Unknown {
			result.DocModified = model.False
		}
	}

	p7Signer := p7.Signers[0]

	signerCert := pkcs7.GetCertFromCertsByIssuerAndSerial(certs, p7Signer.IssuerAndSerialNumber)
	if signerCert == nil {
		signer.AddProblem("\"ETSI.RFC3161\": missing certificate for signer")
		return nil
	}

	if err := pkcs7.CheckSignature(signerCert, p7Signer, nil); err != nil {
		signer.AddProblem(fmt.Sprintf("\"ETSI.RFC3161\": signature verification failure: %v", err))
		return nil
	}

	signingTime := tstInfo.GenTime
	signer.Timestamp = signingTime
	signer.HasTimestamp = true
	result.Details.SigningTime = signingTime

	// Ensure issueing TSA is trusted.
	validateDTSCert(signingTime, signerCert, certs, rootCerts, crls, ocsps, signer, result, ctx)

	return nil
}

func validateDTSCert(signingTime time.Time,
	signerCert *x509.Certificate,
	certs []*x509.Certificate,
	rootCerts *x509.CertPool,
	crls, ocsps [][]byte,
	signer *model.Signer,
	result *model.SignatureValidationResult,
	ctx *model.Context) {

	if signingTime.After(signerCert.NotAfter) || signingTime.Before(signerCert.NotBefore) {
		signer.AddProblem(fmt.Sprintf("\"ETSI.RFC3161\": signing time %q is outside of certificate validity %q to %q",
			signingTime.Format(time.RFC3339),
			signerCert.NotBefore.Format(time.RFC3339),
			signerCert.NotAfter.Format(time.RFC3339)))
		return
	}

	// Does signerCert chain up to a trusted Root CA?
	chains := buildP7CertChains(true, signerCert, certs, rootCerts, signer, &signingTime, result)
	if len(chains) == 0 {
		chains = [][]*x509.Certificate{certChain(signerCert, certs)}
	}

	validateCertChains(chains, rootCerts, signer, &signingTime, crls, ocsps, result, ctx.Configuration)

	finalizeDTSResult(result, ctx, signingTime)
}

func checkDTSDigest(tstInfo *TSTInfo, data []byte, signer *model.Signer) bool {

	oidHashAlg := tstInfo.MessageImprint.HashAlgorithm.Algorithm
	digest := tstInfo.MessageImprint.HashedMessage

	if err := pkcs7.VerifyMessageDigestTSToken(oidHashAlg, digest, data); err != nil {
		var mdErr *pkcs7.MessageDigestMismatchError
		if errors.As(err, &mdErr) {
			signer.AddProblem(fmt.Sprintf("\"ETSI.RFC3161\": message digest verification failure: %v", err))
			return false
		}
		signer.AddProblem(fmt.Sprintf("\"ETSI.RFC3161\": message digest verification: %v", err))
		return false
	}

	return true
}

func collectIntermediates(signerCert *x509.Certificate, certs []*x509.Certificate) []*x509.Certificate {
	var intermediates []*x509.Certificate
	for _, cert := range certs {
		if !cert.Equal(signerCert) {
			intermediates = append(intermediates, cert)
		}
	}
	return intermediates
}

func finalizeDTSResult(result *model.SignatureValidationResult, ctx *model.Context, signingTime time.Time) {
	if result.Status == model.SignatureStatusUnknown && result.Reason == model.SignatureReasonUnknown {
		result.Status = model.SignatureStatusValid
		result.Reason = model.SignatureReasonDocNotModified
		ctx.DTS = signingTime
	} else {
		ctx.DTS = time.Time{}
	}
}
