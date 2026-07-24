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
	"crypto/dsa"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/pkcs7"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

const (
	// CertifiedSigPermsNotSupported reports unsupported certified signature permission validation.
	CertifiedSigPermsNotSupported = "Certified signature detected. Permission validation not supported."

	certImportHint = "import missing certificates into pdfcpu's local certificate store with \"pdfcpu certificates import <file>\""
)

func parseCertificate(bb []byte) (*x509.Certificate, error) {
	cert, err := x509.ParseCertificate(bb)
	if err != nil {
		return nil, &certificateParseError{cause: err}
	}
	return cert, nil
}

func setResultReason(result *model.SignatureValidationResult, reason model.SignatureReason) {
	if result.Reason == model.SignatureReasonUnknown {
		result.Reason = reason
	}
}

func markDocumentUnmodified(result *model.SignatureValidationResult) {
	if result.DocModified == model.Unknown {
		result.DocModified = model.False
	}
}

func markInvalidEvidence(
	result *model.SignatureValidationResult,
	reason model.SignatureReason,
	docModified int,
) {
	if result.Status != model.SignatureStatusInvalid {
		result.Status = model.SignatureStatusInvalid
		result.Reason = reason
	}
	if docModified == model.True || result.DocModified == model.Unknown {
		result.DocModified = docModified
	}
}

func markUnsupportedEvidence(result *model.SignatureValidationResult) {
	setResultReason(result, model.SignatureReasonUnsupported)
}

func markMalformedEvidence(result *model.SignatureValidationResult) {
	setResultReason(result, model.SignatureReasonMalformed)
}

func markCertificateInvalidEvidence(result *model.SignatureValidationResult) {
	setResultReason(result, model.SignatureReasonCertInvalid)
}

func finalizeLocalSignatureResult(
	result *model.SignatureValidationResult,
	assessment localSignatureAssessment,
) bool {
	if !assessment.complete() ||
		result.Status != model.SignatureStatusUnknown ||
		result.Reason != model.SignatureReasonUnknown {
		return false
	}
	result.Status = model.SignatureStatusValid
	result.Reason = model.SignatureReasonDocNotModified
	markDocumentUnmodified(result)
	return true
}

func assessCertificateEvidence(
	chains [][]*x509.Certificate,
	pathResolved bool,
	rootCerts *x509.CertPool,
	crls, ocsps [][]byte,
	reason model.SignatureReason,
	conf *model.Configuration,
) (certificateAssessment, error) {
	signer := &model.Signer{}
	result := &model.SignatureValidationResult{Reason: reason}
	if err := validateCertChains(
		chains,
		pathResolved,
		rootCerts,
		signer,
		crls,
		ocsps,
		result,
		conf,
	); err != nil {
		return certificateAssessment{}, err
	}
	return certificateAssessment{
		Certificate:           signer.Certificate,
		CertificatePathStatus: signer.CertificatePathStatus,
		Problems:              append([]string(nil), signer.Problems...),
		Reason:                result.Reason,
	}, nil
}

func applyCertificateAssessment(
	assessment certificateAssessment,
	signer *model.Signer,
	result *model.SignatureValidationResult,
) {
	if assessment.Certificate != nil {
		signer.Certificate = assessment.Certificate
	}
	signer.CertificatePathStatus = assessment.CertificatePathStatus
	for _, problem := range assessment.Problems {
		signer.AddProblem(problem)
	}
	if assessment.Reason != 0 {
		setResultReason(result, assessment.Reason)
	}
}

func validateCertChains(
	chains [][]*x509.Certificate, // All chain paths for cert leading to a root CA.
	pathResolved bool,
	rootCerts *x509.CertPool,
	signer *model.Signer,
	crls [][]byte,
	ocsps [][]byte,
	result *model.SignatureValidationResult,
	conf *model.Configuration,
) error {
	if len(chains) == 0 || len(chains[0]) == 0 {
		signer.AddProblem("certificate chain: missing")
		if result.Reason == model.SignatureReasonUnknown {
			result.Reason = model.SignatureReasonCertInvalid
		}
		return nil
	}

	var cd *model.CertificateDetails

	// TODO Process all chains.
	chain := chains[0]

	for i, cert := range chain {
		certDetails, err := validateCertificateInChain(
			cert,
			certificateIssuer(chain, i),
			i,
			pathResolved,
			rootCerts,
			signer,
			crls,
			ocsps,
			result,
			conf,
		)
		if err != nil {
			return err
		}
		cd = appendCertificateDetails(signer, cd, certDetails)
	}
	if signer.Certificate != nil {
		signer.CertificatePathStatus = signer.Certificate.PathEvidence.Status
	}
	return nil
}

func validateCertificateInChain(
	cert, issuer *x509.Certificate,
	certIndex int,
	pathResolved bool,
	rootCerts *x509.CertPool,
	signer *model.Signer,
	crls, ocsps [][]byte,
	result *model.SignatureValidationResult,
	conf *model.Configuration,
) (*model.CertificateDetails, error) {
	certDetails := &model.CertificateDetails{}
	setLocalCertificatePathStatus(certDetails, pathResolved)
	if cert == nil {
		reportMissingCertificate(certDetails, signer, result, certIndex)
		return certDetails, nil
	}
	ok, err := setupCertDetails(cert, certDetails, signer, result, certIndex)
	if err != nil {
		return nil, err
	}
	if !ok {
		return certDetails, nil
	}
	selfSigned, err := isSelfSigned(cert)
	certDetails.SelfSigned = selfSigned
	if err != nil {
		reportSelfSignedCertificateError(cert, certDetails, signer, result, certIndex, err)
		return certDetails, nil
	}
	if selfSigned || certDetails.CA {
		return certDetails, nil
	}
	if certDetails.Expired && len(crls) == 0 && len(ocsps) == 0 {
		return certDetails, nil
	}
	checkRevocation(cert, issuer, rootCerts, signer, certDetails, crls, ocsps, result, conf)
	return certDetails, nil
}

func reportMissingCertificate(
	certDetails *model.CertificateDetails,
	signer *model.Signer,
	result *model.SignatureValidationResult,
	certIndex int,
) {
	certDetails.Leaf = certIndex == 0
	signer.AddProblem(fmt.Sprintf("certificate chain, certificate index %d: missing certificate", certIndex+1))
	if result.Reason == model.SignatureReasonUnknown {
		result.Reason = model.SignatureReasonCertInvalid
	}
}

func reportSelfSignedCertificateError(
	cert *x509.Certificate,
	certDetails *model.CertificateDetails,
	signer *model.Signer,
	result *model.SignatureValidationResult,
	certIndex int,
	err error,
) {
	signer.AddProblem(fmt.Sprintf(
		"certificate chain, certificate index %d: verify self-signed certificate %s: %v",
		certIndex+1,
		certInfo(cert),
		err,
	))
	if result.Reason == model.SignatureReasonUnknown {
		result.Reason = model.SignatureReasonSelfSignedCertErr
	}
}

func appendCertificateDetails(
	signer *model.Signer,
	previous, current *model.CertificateDetails,
) *model.CertificateDetails {
	if previous == nil {
		signer.Certificate = current
		return current
	}
	previous.IssuerCertificate = current
	return current
}

func certificateIssuer(chain []*x509.Certificate, certIndex int) *x509.Certificate {
	if certIndex < 0 || certIndex+1 >= len(chain) {
		return nil
	}
	return chain[certIndex+1]
}

func setupCertDetails(
	cert *x509.Certificate,
	certDetails *model.CertificateDetails,
	signer *model.Signer,
	result *model.SignatureValidationResult,
	i int,
) (bool, error) {
	certDetails.Leaf = i == 0
	certDetails.Subject = cert.Subject.CommonName
	certDetails.Issuer = cert.Issuer.CommonName
	certDetails.SerialNumber = cert.SerialNumber.Text(16)
	certDetails.Version = cert.Version
	certDetails.ValidFrom = cert.NotBefore
	certDetails.ValidThru = cert.NotAfter

	ts := time.Now()
	certDetails.Expired = ts.Before(cert.NotBefore) || ts.After(cert.NotAfter)

	certDetails.Usage = certUsage(cert)
	certDetails.Qualified = hasRecognizedQualifiedCertificatePolicy(cert)
	certDetails.CA = cert.IsCA

	certDetails.SignAlg = cert.PublicKeyAlgorithm.String()

	keySize, ok, err := getKeySize(cert, signer, certDetails, result, i)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	certDetails.KeySize = keySize

	return true, nil
}

func getKeySize(
	cert *x509.Certificate,
	signer *model.Signer,
	certDetails *model.CertificateDetails,
	result *model.SignatureValidationResult,
	certIndex int,
) (int, bool, error) {
	return getKeySizeWith(cert, signer, certDetails, result, certIndex, publicKeySize)
}

func getKeySizeWith(
	cert *x509.Certificate,
	signer *model.Signer,
	certDetails *model.CertificateDetails,
	result *model.SignatureValidationResult,
	certIndex int,
	inspect func(*x509.Certificate) (int, error),
) (int, bool, error) {
	keySize, err := inspect(cert)
	if err == nil {
		return keySize, true, nil
	}
	context := fmt.Sprintf(
		"certificate chain, certificate index %d: inspect public key for %s: %v",
		certIndex+1,
		certInfo(cert),
		err,
	)
	switch {
	case errors.Is(err, errUnsupportedPublicKey):
		signer.AddProblem(context)
		markUnsupportedEvidence(result)
	case errors.Is(err, errMalformedPublicKey):
		signer.AddProblem(context)
		markCertificateInvalidEvidence(result)
	default:
		return 0, false, fmt.Errorf(
			"certificate chain, certificate index %d: inspect public key for %s: %w",
			certIndex+1,
			certInfo(cert),
			err,
		)
	}
	return 0, false, nil
}

func setLocalCertificatePathStatus(certDetails *model.CertificateDetails, pathResolved bool) {
	if !pathResolved {
		setCertificatePathConclusion(
			certDetails,
			model.Unknown,
			"certificate path was not resolved using the configured local certificate store",
			model.CertificatePathMethodLocalTrustStore,
		)
		return
	}
	setCertificatePathConclusion(
		certDetails,
		model.True,
		"certificate path resolved using the configured local certificate store",
		model.CertificatePathMethodLocalTrustStore,
	)
}

func setCertificatePathConclusion(
	certDetails *model.CertificateDetails,
	status int,
	reason string,
	method model.CertificatePathMethod,
) {
	certDetails.Trust.Status = status
	certDetails.Trust.Reason = reason
	certDetails.PathEvidence = model.CertificatePathEvidence{
		AssessmentScope: model.AssessmentScopeLocal,
		Method:          method,
		Status:          status,
		Reason:          reason,
	}
}

func signedData(ra io.ReaderAt, sigDict types.Dict) ([]byte, error) {
	if ra == nil {
		return nil, fatalByteRangeRead(errors.New("signature dict entry ByteRange: missing reader"))
	}
	arr := sigDict.ArrayEntry("ByteRange")
	if len(arr) != 4 {
		return nil, malformedByteRange(errors.New("signature dict entry ByteRange: missing or invalid length"))
	}
	values, err := byteRangeValues(arr)
	if err != nil {
		return nil, err
	}
	if _, err := validateByteRange(values); err != nil {
		return nil, err
	}
	if err := validateContentsGap(ra, sigDict, values); err != nil {
		return nil, err
	}
	return bytesForByteRange(ra, arr)
}

func validateContentsGap(ra io.ReaderAt, sigDict types.Dict, values [4]int64) error {
	contents := sigDict.HexLiteralEntry("Contents")
	if contents == nil {
		return malformedByteRange(errors.New("signature dict entry ByteRange: excluded gap has no signature dict entry Contents"))
	}

	end1, err := byteRangeEnd(values[0], values[1])
	if err != nil {
		return err
	}
	gapSize := values[2] - end1
	if gapSize < 2 || gapSize > int64(len(contents.Value()))+2+(1<<20) {
		return malformedByteRange(errors.New("signature dict entry ByteRange: excluded gap does not match signature dict entry Contents"))
	}

	var gap bytes.Buffer
	if err := copyByteRange(&gap, ra, end1, gapSize); err != nil {
		return err
	}
	if !contentsGapMatches(gap.Bytes(), contents.Value()) {
		return malformedByteRange(errors.New("signature dict entry ByteRange: excluded gap does not match signature dict entry Contents"))
	}
	return nil
}

func contentsGapMatches(gap []byte, contents string) bool {
	if len(gap) < 2 || gap[0] != '<' || gap[len(gap)-1] != '>' {
		return false
	}
	i := 0
	for _, b := range gap[1 : len(gap)-1] {
		if strings.ContainsRune(" \t\n\f\r", rune(b)) {
			continue
		}
		if i >= len(contents) || toUpperHex(b) != toUpperHex(contents[i]) {
			return false
		}
		i++
	}
	return i == len(contents)
}

func toUpperHex(b byte) byte {
	if b >= 'a' && b <= 'f' {
		return b - ('a' - 'A')
	}
	return b
}

func byteRangeValues(arr types.Array) ([4]int64, error) {
	var values [4]int64
	if len(arr) != len(values) {
		return values, malformedByteRange(errors.New("signature dict entry ByteRange: invalid length"))
	}
	for i, o := range arr {
		v, ok := o.(types.Integer)
		if !ok {
			return values, malformedByteRange(
				fmt.Errorf("signature dict entry ByteRange, array index %d: expected integer", i+1),
			)
		}
		values[i] = int64(v.Value())
		if values[i] < 0 {
			return values, malformedByteRange(
				fmt.Errorf("signature dict entry ByteRange, array index %d: negative value", i+1),
			)
		}
	}
	return values, nil
}

func byteRangeEnd(off, size int64) (int64, error) {
	if off > math.MaxInt64-size {
		return 0, malformedByteRange(errors.New("signature dict entry ByteRange: offset and length overflow"))
	}
	return off + size, nil
}

func validateByteRange(values [4]int64) (int64, error) {
	if values[0] != 0 {
		return 0, malformedByteRange(errors.New("signature dict entry ByteRange: first range must begin at offset 0"))
	}
	end1, err := byteRangeEnd(values[0], values[1])
	if err != nil {
		return 0, err
	}
	if end1 > values[2] {
		return 0, malformedByteRange(errors.New("signature dict entry ByteRange: overlapping ranges"))
	}
	if _, err := byteRangeEnd(values[2], values[3]); err != nil {
		return 0, err
	}
	total, err := byteRangeEnd(values[1], values[3])
	if err != nil || total > int64(math.MaxInt) {
		return 0, malformedByteRange(errors.New("signature dict entry ByteRange: total size overflow"))
	}
	return total, nil
}

func copyByteRange(w io.Writer, ra io.ReaderAt, off, size int64) error {
	n, err := io.CopyN(w, io.NewSectionReader(ra, off, size), size)
	if err != nil {
		readErr := fmt.Errorf("signature dict entry ByteRange, offset %d, length %d: read: %w", off, size, err)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return malformedByteRange(readErr)
		}
		return fatalByteRangeRead(readErr)
	}
	if n != size {
		return malformedByteRange(
			fmt.Errorf("signature dict entry ByteRange, offset %d, length %d: short read", off, size),
		)
	}
	return nil
}

func bytesForByteRange(ra io.ReaderAt, arr types.Array) ([]byte, error) {
	if ra == nil {
		return nil, fatalByteRangeRead(errors.New("signature dict entry ByteRange: missing reader"))
	}
	values, err := byteRangeValues(arr)
	if err != nil {
		return nil, err
	}
	total, err := validateByteRange(values)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if total <= 1<<20 {
		buf.Grow(int(total))
	}
	if err := copyByteRange(&buf, ra, values[0], values[1]); err != nil {
		return nil, err
	}
	if err := copyByteRange(&buf, ra, values[2], values[3]); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// isSelfSigned checks if a given certificate is self-signed.
func isSelfSigned(cert *x509.Certificate) (bool, error) {
	// Check if subject and issuer are the same
	if !comparePKIXName(cert.Subject, cert.Issuer) {
		return false, nil
	}

	// Verify the certificate's signature against its own public key
	err := cert.CheckSignatureFrom(cert)

	return true, err
}

func comparePKIXName(a, b pkix.Name) bool {
	return slices.Equal(a.Country, b.Country) &&
		slices.Equal(a.Organization, b.Organization) &&
		slices.Equal(a.OrganizationalUnit, b.OrganizationalUnit) &&
		slices.Equal(a.Locality, b.Locality) &&
		slices.Equal(a.Province, b.Province) &&
		slices.Equal(a.StreetAddress, b.StreetAddress) &&
		slices.Equal(a.PostalCode, b.PostalCode) &&
		a.CommonName == b.CommonName
}

func certUsage(cert *x509.Certificate) string {
	ss := []string{}
	for _, usage := range cert.ExtKeyUsage {
		switch usage {
		case x509.ExtKeyUsageServerAuth:
			ss = append(ss, "Server Authentication")
		case x509.ExtKeyUsageClientAuth:
			ss = append(ss, "Client Authentication")
		case x509.ExtKeyUsageCodeSigning:
			ss = append(ss, "Code Signing")
		case x509.ExtKeyUsageEmailProtection:
			ss = append(ss, "Email Protection")
		case x509.ExtKeyUsageTimeStamping:
			ss = append(ss, "Time Stamping")
		case x509.ExtKeyUsageOCSPSigning:
			ss = append(ss, "OCSP Signing")
		case x509.ExtKeyUsageIPSECEndSystem:
			ss = append(ss, "IPSEC End System")
		case x509.ExtKeyUsageIPSECTunnel:
			ss = append(ss, "IPSEC Tunnel")
		case x509.ExtKeyUsageIPSECUser:
			ss = append(ss, "IPSEC User")
		case x509.ExtKeyUsageAny:
			ss = append(ss, "Any")
		default:
			ss = append(ss, "Any")
		}
	}
	return strings.Join(ss, ",")
}

func hasRecognizedQualifiedCertificatePolicy(cert *x509.Certificate) bool {
	for _, policy := range cert.PolicyIdentifiers {
		switch {
		case policy.Equal(oidQCESign):
			return true
		case policy.Equal(oidQCESeal):
			return true
		case policy.Equal(oidQWebAuthCert):
			return true
		case policy.Equal(oidETSIQCPublicWithSSCD):
			return true
		}
	}
	return false
}

func certChain(cert *x509.Certificate, certs []*x509.Certificate) []*x509.Certificate {
	if cert == nil {
		return nil
	}
	certMap := make(map[string]*x509.Certificate)
	for _, cert := range certs {
		if cert == nil {
			continue
		}
		certMap[string(cert.RawSubject)] = cert
	}

	current := cert

	var sorted []*x509.Certificate
	visited := map[*x509.Certificate]bool{}

	for current != nil && !visited[current] {
		visited[current] = true
		sorted = append(sorted, current)
		current = certMap[string(current.RawIssuer)]
	}

	return sorted
}

func publicKeySize(cert *x509.Certificate) (int, error) {
	switch pubKey := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		return rsaPublicKeySize(pubKey)
	case *ecdsa.PublicKey:
		return ecdsaPublicKeySize(pubKey)
	case ed25519.PublicKey:
		return ed25519PublicKeySize(pubKey)
	case *dsa.PublicKey:
		return dsaPublicKeySize(pubKey)
	default:
		return 0, fmt.Errorf("%w: type %T", errUnsupportedPublicKey, pubKey)
	}
}

func rsaPublicKeySize(pubKey *rsa.PublicKey) (int, error) {
	if pubKey == nil || pubKey.N == nil || pubKey.N.Sign() <= 0 || pubKey.E <= 0 {
		return 0, fmt.Errorf("%w: invalid RSA public key", errMalformedPublicKey)
	}
	return pubKey.Size() * 8, nil
}

func ecdsaPublicKeySize(pubKey *ecdsa.PublicKey) (int, error) {
	if pubKey == nil || pubKey.Curve == nil || pubKey.Curve.Params() == nil {
		return 0, fmt.Errorf("%w: invalid ECDSA public key", errMalformedPublicKey)
	}
	return pubKey.Curve.Params().BitSize, nil
}

func ed25519PublicKeySize(pubKey ed25519.PublicKey) (int, error) {
	if len(pubKey) != ed25519.PublicKeySize {
		return 0, fmt.Errorf("%w: invalid Ed25519 public key", errMalformedPublicKey)
	}
	return ed25519.PublicKeySize * 8, nil
}

func dsaPublicKeySize(pubKey *dsa.PublicKey) (int, error) {
	if pubKey == nil || pubKey.Y == nil || pubKey.Y.Sign() <= 0 {
		return 0, fmt.Errorf("%w: invalid DSA public key", errMalformedPublicKey)
	}
	return pubKey.Y.BitLen(), nil
}

func handleCertVerifyErr(err error, cert *x509.Certificate, signer *model.Signer, result *model.SignatureValidationResult) {
	var unknownAuthorityErr x509.UnknownAuthorityError
	if errors.As(err, &unknownAuthorityErr) {
		if result.Reason == model.SignatureReasonUnknown {
			result.Reason = model.SignatureReasonCertNotTrusted
		}
		signer.AddProblem(fmt.Sprintf(
			"certificate path was not resolved using the configured local certificate store for %s: %v",
			certInfo(cert),
			err,
		))
		signer.AddProblem(certImportHint)
		return
	}

	var certInvalidErr x509.CertificateInvalidError
	if errors.As(err, &certInvalidErr) && certInvalidErr.Reason == x509.Expired {
		if result.Reason == model.SignatureReasonUnknown {
			result.Reason = model.SignatureReasonCertExpired
		}
		signer.AddProblem(fmt.Sprintf("certificate verification failed for %s: %v", certInfo(cert), err))
		return
	}

	if result.Reason == model.SignatureReasonUnknown {
		result.Reason = model.SignatureReasonCertInvalid
	}
	signer.AddProblem(fmt.Sprintf("certificate verification failed for %s: %v", certInfo(cert), err))
}

func handleCertParseErr(err error, result *model.SignatureValidationResult) {
	if result.Reason == model.SignatureReasonUnknown {
		result.Reason = model.SignatureReasonCertInvalid
	}
	result.AddProblem(fmt.Sprintf("certificate data is malformed: %v", err))
}

func certInfo(cert *x509.Certificate) string {
	return fmt.Sprintf("serial=%q", cert.SerialNumber.Text(16))
}

func processDSS(ctx *model.Context, signer *model.Signer) dssEvidence {
	ok := true
	dssCerts, err := extractCertsFromDSS(ctx)
	if err != nil {
		signer.AddProblem(fmt.Sprintf("%v", err))
		ok = false
	}

	dssCRLs, err := extractCRLsFromDSS(ctx)
	if err != nil {
		signer.AddProblem(fmt.Sprintf("%v", err))
		ok = false
	}

	dssOCSPs, err := extractOCSPsFromDSS(ctx)
	if err != nil {
		signer.AddProblem(fmt.Sprintf("%v", err))
		ok = false
	}

	if _, found := ctx.DSS.Find("VRI"); found {
		signer.AddProblem("DSS dict entry VRI: unsupported")
		ok = false
	}

	return dssEvidence{
		Certificates:  dssCerts,
		CRLs:          dssCRLs,
		OCSPResponses: dssOCSPs,
		Supported:     ok,
	}
}

func extractCertsFromDSS(ctx *model.Context) ([]*x509.Certificate, error) {
	entry, found := ctx.DSS.Find("Certs")
	if !found {
		return nil, nil
	}

	arr, err := ctx.DereferenceArray(entry)
	if err != nil {
		return nil, fmt.Errorf("DSS dict entry Certs: dereference array: %w", err)
	}

	var certs []*x509.Certificate

	for i, obj := range arr {
		sd, _, err := ctx.DereferenceStreamDict(obj)
		if err != nil {
			return nil, fmt.Errorf("DSS dict entry Certs, array index %d: dereference stream dictionary: %w", i+1, err)
		}
		if sd == nil {
			return nil, fmt.Errorf("DSS dict entry Certs, array index %d: missing stream dictionary", i+1)
		}
		if err := sd.Decode(); err != nil {
			return nil, fmt.Errorf("DSS dict entry Certs, array index %d: decode stream: %w", i+1, err)
		}
		cert, err := parseCertificate(sd.Content)
		if err != nil {
			return nil, fmt.Errorf("DSS dict entry Certs, array index %d: parse certificate: %w", i+1, err)
		}
		certs = append(certs, cert)
	}

	return certs, nil
}

func mergeCerts(certLists ...[]*x509.Certificate) []*x509.Certificate {
	visited := map[string]bool{}
	var result []*x509.Certificate
	for _, list := range certLists {
		for _, cert := range list {
			if cert == nil {
				continue
			}
			fingerprint := string(cert.Raw)
			if !visited[fingerprint] {
				visited[fingerprint] = true
				result = append(result, cert)
			}
		}
	}
	return result
}

func extractCRLsFromDSS(ctx *model.Context) ([][]byte, error) {
	entry, found := ctx.DSS.Find("CRLs")
	if !found {
		return nil, nil
	}

	arr, err := ctx.DereferenceArray(entry)
	if err != nil {
		return nil, fmt.Errorf("DSS dict entry CRLs: dereference array: %w", err)
	}

	var crls [][]byte

	for i, obj := range arr {
		sd, _, err := ctx.DereferenceStreamDict(obj)
		if err != nil {
			return nil, fmt.Errorf("DSS dict entry CRLs, array index %d: dereference stream dictionary: %w", i+1, err)
		}
		if sd == nil {
			return nil, fmt.Errorf("DSS dict entry CRLs, array index %d: missing stream dictionary", i+1)
		}
		if err := sd.Decode(); err != nil {
			return nil, fmt.Errorf("DSS dict entry CRLs, array index %d: decode stream: %w", i+1, err)
		}
		crls = append(crls, sd.Content)
	}

	return crls, nil
}

func extractOCSPsFromDSS(ctx *model.Context) ([][]byte, error) {
	entry, found := ctx.DSS.Find("OCSPs")
	if !found {
		return nil, nil
	}

	arr, err := ctx.DereferenceArray(entry)
	if err != nil {
		return nil, fmt.Errorf("DSS dict entry OCSPs: dereference array: %w", err)
	}

	var ocsps [][]byte

	for i, obj := range arr {
		sd, _, err := ctx.DereferenceStreamDict(obj)
		if err != nil {
			return nil, fmt.Errorf("DSS dict entry OCSPs, array index %d: dereference stream dictionary: %w", i+1, err)
		}
		if sd == nil {
			return nil, fmt.Errorf("DSS dict entry OCSPs, array index %d: missing stream dictionary", i+1)
		}
		if err := sd.Decode(); err != nil {
			return nil, fmt.Errorf("DSS dict entry OCSPs, array index %d: decode stream: %w", i+1, err)
		}
		ocsps = append(ocsps, sd.Content)
	}

	return ocsps, nil
}

func validateP7(sigDict types.Dict, result *model.SignatureValidationResult) *pkcs7.PKCS7 {
	p7, err := p7(sigDict)
	if err != nil {
		if errors.Is(err, errCertificateParse) || errors.Is(err, pkcs7.ErrCertificateParse) {
			handleCertParseErr(err, result)
			return nil
		}
		switch {
		case errors.Is(err, pkcs7.ErrUnsupportedContentType),
			errors.Is(err, pkcs7.ErrUnsupportedAlgorithm),
			errors.Is(err, pkcs7.ErrAlgorithmMismatch):
			markUnsupportedEvidence(result)
		default:
			markMalformedEvidence(result)
		}
		result.AddProblem(fmt.Sprintf("pkcs7: %v", err))
		return nil
	}

	if len(p7.Signers) == 0 {
		markMalformedEvidence(result)
		result.AddProblem("pkcs7: message without signers")
		return nil
	}

	if len(p7.Signers) != 1 && result.Details.IsETSI_CAdES_detached() {
		markMalformedEvidence(result)
		result.AddProblem("pkcs7: \"ETSI.CAdES.detached\" requires a single signer")
		return nil
	}

	return p7
}
