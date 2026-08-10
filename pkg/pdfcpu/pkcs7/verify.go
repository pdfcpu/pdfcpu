/*
Copyright (c) 2015 Andrew Smith
Copyright 2026 The pdfcpu Authors.

Licensed under the MIT License. See LICENSE in this directory.
*/

package pkcs7

import (
	"bytes"
	"crypto"
	"crypto/dsa"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/subtle"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"fmt"
	"time"
)

// VerifyMessageDigestDetached verifies signer's authenticated message digest
// against signedData.
func VerifyMessageDigestDetached(signer SignerInfo, signedData []byte) error {
	hash, err := HashForOID(signer.DigestAlgorithm.Algorithm)
	if err != nil {
		return fmt.Errorf("pkcs7: verify detached message digest: select algorithm: %w", err)
	}

	var digest []byte
	if err := unmarshalAttribute(signer.AuthenticatedAttributes, oidAttributeMessageDigest, &digest); err != nil {
		return fmt.Errorf("pkcs7: verify detached message digest: read message-digest attribute: %w", err)
	}

	if err := compareMessageDigest(hash, digest, signedData); err != nil {
		return fmt.Errorf("pkcs7: verify detached message digest: %w", err)
	}
	return nil
}

// VerifyMessageDigestEmbedded verifies the SHA-1 document digest mandated by the legacy adbe.pkcs7.sha1 profile.
// It supports validation of existing PDFs only.
func VerifyMessageDigestEmbedded(digest, signedData []byte) error {
	// #nosec G401 -- SHA-1 is mandated by the legacy profile and is used for verification only.
	if err := compareMessageDigest(crypto.SHA1, digest, signedData); err != nil {
		return fmt.Errorf("pkcs7: verify embedded message digest: %w", err)
	}
	return nil
}

// VerifyMessageDigestTSToken verifies a timestamp-token digest against
// signedData using oidHashAlg.
func VerifyMessageDigestTSToken(oidHashAlg asn1.ObjectIdentifier, digest, signedData []byte) error {
	hash, err := HashForOID(oidHashAlg)
	if err != nil {
		return fmt.Errorf("pkcs7: verify timestamp-token message digest: select algorithm: %w", err)
	}

	if err := compareMessageDigest(hash, digest, signedData); err != nil {
		return fmt.Errorf("pkcs7: verify timestamp-token message digest: %w", err)
	}
	return nil
}

func compareMessageDigest(hash crypto.Hash, expected, data []byte) error {
	if !hash.Available() {
		return fmt.Errorf("%w: hash %d is unavailable", ErrUnsupportedAlgorithm, hash)
	}
	h := hash.New()
	_, _ = h.Write(data)
	actual := h.Sum(nil)
	if subtle.ConstantTimeCompare(expected, actual) != 1 {
		return &MessageDigestMismatchError{
			ExpectedDigest: expected,
			ActualDigest:   actual,
		}
	}
	return nil
}

// CheckSignature verifies signer's content binding and encrypted digest using
// cert. Signed attributes, when present, are checked against id-data content.
func CheckSignature(
	cert *x509.Certificate,
	signer SignerInfo,
	content []byte,
) error {
	return checkSignature(cert, signer, content, OIDData)
}

// CheckSignatureWithContentType verifies signer's content binding and
// encrypted digest using cert and the encapsulated content type.
func CheckSignatureWithContentType(
	cert *x509.Certificate,
	signer SignerInfo,
	content []byte,
	contentType asn1.ObjectIdentifier,
) error {
	return checkSignature(cert, signer, content, contentType)
}

func checkSignature(
	cert *x509.Certificate,
	signer SignerInfo,
	content []byte,
	contentType asn1.ObjectIdentifier,
) error {
	if cert == nil {
		return errors.New("pkcs7: verify signature: signer certificate is missing")
	}
	sigalg, err := getSignatureAlgorithm(signer.DigestEncryptionAlgorithm, signer.DigestAlgorithm)
	if err != nil {
		return fmt.Errorf("pkcs7: verify signature: select algorithm: %w", err)
	}

	signedData := content
	var contentBindingErr error
	if len(signer.AuthenticatedAttributes) > 0 {
		signedData, err = marshalAttributes(signer.AuthenticatedAttributes)
		if err != nil {
			return fmt.Errorf("pkcs7: verify signature: marshal authenticated attributes: %w", err)
		}
		if err := validateSignedContentType(signer, contentType); err != nil {
			return fmt.Errorf("pkcs7: verify signature: validate signed content type: %w", err)
		}
		if err := VerifyMessageDigestDetached(signer, content); err != nil {
			wrappedErr := fmt.Errorf("pkcs7: verify signature: verify signed content digest: %w", err)
			var digestMismatchErr *MessageDigestMismatchError
			if !errors.As(err, &digestMismatchErr) {
				return wrappedErr
			}
			contentBindingErr = wrappedErr
		}
	}

	if err := validateCertificatePublicKey(cert.PublicKey); err != nil {
		return fmt.Errorf("pkcs7: verify signature: verify cryptographic signature: %w", err)
	}
	if err := verifyCryptographicSignature(cert, signer, sigalg, signedData); err != nil {
		return fmt.Errorf("pkcs7: verify signature: verify cryptographic signature: %w", err)
	}
	return contentBindingErr
}

func validateSignedContentType(
	signer SignerInfo,
	contentType asn1.ObjectIdentifier,
) error {
	var signedContentType asn1.ObjectIdentifier
	if err := unmarshalAttribute(
		signer.AuthenticatedAttributes,
		oidAttributeContentType,
		&signedContentType,
	); err != nil {
		return err
	}
	if len(contentType) == 0 {
		return fmt.Errorf("%w: encapsulated content type is missing", ErrMalformedAttribute)
	}
	if !signedContentType.Equal(contentType) {
		return fmt.Errorf(
			"%w: signed content type OID %s does not match encapsulated content type OID %s",
			ErrMalformedAttribute,
			signedContentType,
			contentType,
		)
	}
	return nil
}

func verifyCryptographicSignature(
	cert *x509.Certificate,
	signer SignerInfo,
	sigalg x509.SignatureAlgorithm,
	signedData []byte,
) error {
	if !signer.DigestEncryptionAlgorithm.Algorithm.Equal(OIDEncryptionAlgorithmRSAPSS) {
		err := cert.CheckSignature(sigalg, signedData, signer.EncryptedDigest)
		return classifyCryptographicVerificationError(err)
	}
	params, err := parseRSAPSSParameters(signer.DigestEncryptionAlgorithm.Parameters)
	if err != nil {
		return err
	}
	hash, err := HashForOID(params.HashAlgorithm)
	if err != nil {
		return err
	}
	publicKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("RSASSA-PSS requires an RSA public key: %w", x509.ErrUnsupportedAlgorithm)
	}
	digest := hash.New()
	_, _ = digest.Write(signedData)
	err = rsa.VerifyPSS(
		publicKey,
		hash,
		digest.Sum(nil),
		signer.EncryptedDigest,
		&rsa.PSSOptions{SaltLength: params.SaltLength, Hash: hash},
	)
	return classifyCryptographicVerificationError(err)
}

func classifyCryptographicVerificationError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, x509.ErrUnsupportedAlgorithm) {
		return err
	}
	var insecureAlgorithmErr x509.InsecureAlgorithmError
	if errors.As(err, &insecureAlgorithmErr) {
		return err
	}
	return fmt.Errorf("%w: %w", ErrSignatureMismatch, err)
}

func validateCertificatePublicKey(publicKey any) error {
	switch key := publicKey.(type) {
	case nil:
		return fmt.Errorf("signer certificate public key is missing: %w", x509.ErrUnsupportedAlgorithm)
	case *rsa.PublicKey:
		return validateRSAPublicKey(key)
	case *ecdsa.PublicKey:
		return validateECDSAPublicKey(key)
	case *dsa.PublicKey:
		return validateDSAPublicKey(key)
	}
	return nil
}

func validateRSAPublicKey(key *rsa.PublicKey) error {
	if key == nil || key.N == nil || key.N.Sign() <= 0 || key.E < 2 {
		return fmt.Errorf("signer certificate RSA public key is invalid: %w", x509.ErrUnsupportedAlgorithm)
	}
	return nil
}

func validateECDSAPublicKey(key *ecdsa.PublicKey) error {
	if key == nil || key.Curve == nil || key.X == nil || key.Y == nil {
		return fmt.Errorf("signer certificate ECDSA public key is invalid: %w", x509.ErrUnsupportedAlgorithm)
	}
	return nil
}

func validateDSAPublicKey(key *dsa.PublicKey) error {
	if key == nil || key.P == nil || key.Q == nil || key.G == nil || key.Y == nil {
		return fmt.Errorf("signer certificate DSA public key is invalid: %w", x509.ErrUnsupportedAlgorithm)
	}
	return nil
}

func parseRawCertificateSet(raw asn1.RawContent) (certs []*x509.Certificate, crls []*x509.RevocationList, err error) {
	if len(raw) == 0 {
		return nil, nil, nil
	}

	var wrapper asn1.RawValue
	rest, err := asn1.Unmarshal(raw, &wrapper)
	if err != nil {
		return nil, nil, certificateParseError(fmt.Errorf("decode certificate-set wrapper: %w", err))
	}
	if len(rest) > 0 {
		err := asn1.SyntaxError{Msg: "trailing data after certificate-set wrapper"}
		return nil, nil, certificateParseError(fmt.Errorf("decode certificate-set wrapper: %w", err))
	}

	switch {
	// Context-specific wrapper [0] IMPLICIT
	case wrapper.Class == asn1.ClassContextSpecific && wrapper.Tag == 0:
		if !wrapper.IsCompound {
			err := asn1.StructuralError{Msg: "certificate-set wrapper is not constructed"}
			return nil, nil, certificateParseError(err)
		}
		return parseCertificateSetEntries(wrapper.Bytes)

	// Universal SET
	case wrapper.Class == asn1.ClassUniversal && wrapper.Tag == asn1.TagSet:
		if !wrapper.IsCompound {
			err := asn1.StructuralError{Msg: "certificate-set wrapper is not constructed"}
			return nil, nil, certificateParseError(err)
		}
		return parseCertificateSetEntries(wrapper.Bytes)

	// Not a SET, try single certificate
	default:
		return parseSingleCertificateSetEntry(raw)
	}
}

func parseCertificateSetEntries(rest []byte) (certs []*x509.Certificate, crls []*x509.RevocationList, err error) {
	for index := 1; len(rest) > 0; index++ {
		var entry asn1.RawValue
		next, err := asn1.Unmarshal(rest, &entry)
		if err != nil {
			cause := fmt.Errorf("certificate-set entry %d: decode ASN.1: %w", index, err)
			return nil, nil, certificateParseError(cause)
		}

		cert, certErr := x509.ParseCertificate(entry.FullBytes)
		if certErr == nil {
			certs = append(certs, cert)
			rest = next
			continue
		}

		if crl, err := x509.ParseRevocationList(entry.FullBytes); err == nil {
			crls = append(crls, crl)
			rest = next
			continue
		}

		if entry.Class == asn1.ClassUniversal && entry.Tag == asn1.TagSequence {
			cause := fmt.Errorf("certificate-set entry %d: parse certificate: %w", index, certErr)
			return nil, nil, certificateParseError(cause)
		}

		// CertificateChoices permits non-certificate context-specific entries.
		if entry.Class != asn1.ClassContextSpecific {
			msg := fmt.Sprintf("certificate-set entry %d: unsupported certificate choice", index)
			return nil, nil, certificateParseError(asn1.StructuralError{Msg: msg})
		}
		rest = next
	}

	return certs, crls, nil
}

func parseSingleCertificateSetEntry(raw []byte) ([]*x509.Certificate, []*x509.RevocationList, error) {
	cert, certErr := x509.ParseCertificate(raw)
	if certErr == nil {
		return []*x509.Certificate{cert}, nil, nil
	}
	if crl, err := x509.ParseRevocationList(raw); err == nil {
		return nil, []*x509.RevocationList{crl}, nil
	}
	cause := fmt.Errorf("certificate-set entry 1: parse certificate: %w", certErr)
	return nil, nil, certificateParseError(cause)
}

// TODO relaxed flag
func parseSignedData(data []byte) (*PKCS7, error) {
	var sd signedData
	rest, err := asn1.Unmarshal(data, &sd)
	if err != nil {
		return nil, fmt.Errorf("pkcs7: decode signed data: %w", err)
	}
	if len(rest) > 0 {
		err := asn1.SyntaxError{Msg: "trailing data after signed data"}
		return nil, fmt.Errorf("pkcs7: decode signed data: %w", err)
	}
	if err := validateSignerDigestAlgorithms(sd.DigestAlgorithmIdentifiers, sd.SignerInfos); err != nil {
		return nil, fmt.Errorf("pkcs7: validate digest algorithms: %w", err)
	}

	// Locate misplaced CRLs in SignedData.certificates.
	// CRLs may be illegally embedded inside the certificates SET.
	certs, embeddedCRLs, err := parseRawCertificateSet(sd.Certificates.Raw)
	if err != nil {
		return nil, fmt.Errorf("pkcs7: parse certificate set: %w", err)
	}

	content, err := parseEncapsulatedContent(sd.ContentInfo.Content.Bytes)
	if err != nil {
		return nil, fmt.Errorf("pkcs7: parse encapsulated content: %w", err)
	}

	crls := append([]*x509.RevocationList(nil), embeddedCRLs...)
	for i, rv := range sd.CRLs {
		rl, err := x509.ParseRevocationList(rv.FullBytes)
		if err != nil {
			return nil, fmt.Errorf("pkcs7: CRL index %d: parse revocation list: %w", i+1, err)
		}
		crls = append(crls, rl)
	}
	return &PKCS7{
		Content:      content,
		ContentType:  sd.ContentInfo.ContentType,
		Certificates: certs,
		CRLs:         crls,
		Signers:      sd.SignerInfos}, nil
}

func validateSignerDigestAlgorithms(
	declared []pkix.AlgorithmIdentifier,
	signers []SignerInfo,
) error {
	for i, signer := range signers {
		found := false
		for _, algorithm := range declared {
			if algorithm.Algorithm.Equal(signer.DigestAlgorithm.Algorithm) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf(
				"%w: signer %d digest OID %s is absent from SignedData digestAlgorithms",
				ErrAlgorithmMismatch,
				i+1,
				signer.DigestAlgorithm.Algorithm,
			)
		}
	}
	return nil
}

func parseEncapsulatedContent(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var value asn1.RawValue
	rest, err := asn1.Unmarshal(data, &value)
	if err != nil {
		return nil, fmt.Errorf("decode ASN.1: %w", err)
	}
	if len(rest) > 0 {
		err := asn1.SyntaxError{Msg: "trailing data after encapsulated content"}
		return nil, fmt.Errorf("decode ASN.1: %w", err)
	}
	return parseOctetString(value)
}

func parseOctetString(value asn1.RawValue) ([]byte, error) {
	if value.Class != asn1.ClassUniversal || value.Tag != asn1.TagOctetString {
		return nil, asn1.StructuralError{Msg: "encapsulated content is not an OCTET STRING"}
	}
	if !value.IsCompound {
		return value.Bytes, nil
	}

	var content []byte
	for index, rest := 1, value.Bytes; len(rest) > 0; index++ {
		var segment asn1.RawValue
		next, err := asn1.Unmarshal(rest, &segment)
		if err != nil {
			return nil, fmt.Errorf("OCTET STRING segment %d: decode ASN.1: %w", index, err)
		}
		bb, err := parseOctetString(segment)
		if err != nil {
			return nil, fmt.Errorf("OCTET STRING segment %d: %w", index, err)
		}
		content = append(content, bb...)
		rest = next
	}
	return content, nil
}

// VerifyCertChain builds all chains from ee through certs to a root in
// truststore.
//
// When verifying chains that may have expired, currentTime can be set to a past date
// to allow the verification to pass. If unset, currentTime is set to the current UTC time.
func VerifyCertChain(
	ee *x509.Certificate,
	certs []*x509.Certificate,
	truststore *x509.CertPool,
	currentTime time.Time,
) ([][]*x509.Certificate, error) {
	if ee == nil {
		return nil, errors.New("pkcs7: verify certificate chain: end-entity certificate is missing")
	}
	intermediates := x509.NewCertPool()
	for i, intermediate := range certs {
		if intermediate == nil {
			return nil, fmt.Errorf(
				"pkcs7: verify certificate chain: intermediate certificate index %d is missing",
				i+1,
			)
		}
		intermediates.AddCert(intermediate)
	}
	verifyOptions := x509.VerifyOptions{
		Roots:         truststore,
		Intermediates: intermediates,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		CurrentTime:   currentTime,
	}
	chains, err := ee.Verify(verifyOptions)
	if err != nil {
		return nil, fmt.Errorf("pkcs7: verify certificate chain: %w", err)
	}
	return chains, nil
}

// MessageDigestMismatchError is returned when the signer data digest does not
// match the computed digest for the contained content.
type MessageDigestMismatchError struct {
	ExpectedDigest []byte
	ActualDigest   []byte
}

func (err *MessageDigestMismatchError) Error() string {
	return fmt.Sprintf("pkcs7: message digest mismatch: expected %X, actual %X", err.ExpectedDigest, err.ActualDigest)
}

func getSignatureAlgorithm(digestEncryption, digest pkix.AlgorithmIdentifier) (x509.SignatureAlgorithm, error) {
	if algorithm, expectedDigest, ok := directSignatureAlgorithm(digestEncryption.Algorithm); ok {
		if !digest.Algorithm.Equal(expectedDigest) {
			return x509.UnknownSignatureAlgorithm, fmt.Errorf(
				"%w: signature algorithm OID %s requires digest OID %s, got %s",
				ErrAlgorithmMismatch,
				digestEncryption.Algorithm,
				expectedDigest,
				digest.Algorithm,
			)
		}
		return algorithm, nil
	}
	if digestEncryption.Algorithm.Equal(OIDEncryptionAlgorithmRSA) {
		return rsaSignatureAlgorithm(digest.Algorithm)
	}
	if digestEncryption.Algorithm.Equal(OIDEncryptionAlgorithmRSAPSS) {
		return rsaPSSSignatureAlgorithm(digestEncryption.Parameters, digest.Algorithm)
	}
	if digestEncryption.Algorithm.Equal(OIDDigestAlgorithmDSA) ||
		digestEncryption.Algorithm.Equal(OIDDigestAlgorithmDSASHA1) {
		return dsaSignatureAlgorithm(digest.Algorithm)
	}
	if isEllipticCurveAlgorithm(digestEncryption.Algorithm) {
		return ecdsaSignatureAlgorithm(digest.Algorithm)
	}
	return x509.UnknownSignatureAlgorithm, fmt.Errorf(
		"%w: signature algorithm OID %s",
		ErrUnsupportedAlgorithm,
		digestEncryption.Algorithm,
	)
}

func directSignatureAlgorithm(
	oid asn1.ObjectIdentifier,
) (x509.SignatureAlgorithm, asn1.ObjectIdentifier, bool) {
	switch {
	case oid.Equal(OIDDigestAlgorithmECDSASHA1):
		return x509.ECDSAWithSHA1, OIDDigestAlgorithmSHA1, true
	case oid.Equal(OIDDigestAlgorithmECDSASHA256):
		return x509.ECDSAWithSHA256, OIDDigestAlgorithmSHA256, true
	case oid.Equal(OIDDigestAlgorithmECDSASHA384):
		return x509.ECDSAWithSHA384, OIDDigestAlgorithmSHA384, true
	case oid.Equal(OIDDigestAlgorithmECDSASHA512):
		return x509.ECDSAWithSHA512, OIDDigestAlgorithmSHA512, true
	case oid.Equal(OIDEncryptionAlgorithmRSASHA1):
		return x509.SHA1WithRSA, OIDDigestAlgorithmSHA1, true
	case oid.Equal(OIDEncryptionAlgorithmRSASHA256):
		return x509.SHA256WithRSA, OIDDigestAlgorithmSHA256, true
	case oid.Equal(OIDEncryptionAlgorithmRSASHA384):
		return x509.SHA384WithRSA, OIDDigestAlgorithmSHA384, true
	case oid.Equal(OIDEncryptionAlgorithmRSASHA512):
		return x509.SHA512WithRSA, OIDDigestAlgorithmSHA512, true
	case oid.Equal(OIDDigestAlgorithmDSASHA1):
		return x509.DSAWithSHA1, OIDDigestAlgorithmSHA1, true
	case oid.Equal(OIDEncryptionAlgorithmEd25519):
		return x509.PureEd25519, OIDDigestAlgorithmSHA512, true
	}
	return x509.UnknownSignatureAlgorithm, nil, false
}

func rsaSignatureAlgorithm(digest asn1.ObjectIdentifier) (x509.SignatureAlgorithm, error) {
	switch {
	case digest.Equal(OIDDigestAlgorithmSHA1):
		return x509.SHA1WithRSA, nil
	case digest.Equal(OIDDigestAlgorithmSHA256):
		return x509.SHA256WithRSA, nil
	case digest.Equal(OIDDigestAlgorithmSHA384):
		return x509.SHA384WithRSA, nil
	case digest.Equal(OIDDigestAlgorithmSHA512):
		return x509.SHA512WithRSA, nil
	}
	return x509.UnknownSignatureAlgorithm, fmt.Errorf(
		"%w: digest OID %s for rsaEncryption",
		ErrUnsupportedAlgorithm,
		digest,
	)
}

var oidMaskGenAlgorithmMGF1 = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 8}

type rsaPSSParametersASN1 struct {
	HashAlgorithm    pkix.AlgorithmIdentifier `asn1:"optional,explicit,tag:0"`
	MaskGenAlgorithm pkix.AlgorithmIdentifier `asn1:"optional,explicit,tag:1"`
	SaltLength       int                      `asn1:"optional,explicit,tag:2,default:20"`
	TrailerField     int                      `asn1:"optional,explicit,tag:3,default:1"`
}

type parsedRSAPSSParameters struct {
	HashAlgorithm    asn1.ObjectIdentifier
	MGFHashAlgorithm asn1.ObjectIdentifier
	SaltLength       int
	TrailerField     int
}

func parseRSAPSSParameters(raw asn1.RawValue) (parsedRSAPSSParameters, error) {
	params := parsedRSAPSSParameters{
		HashAlgorithm:    OIDDigestAlgorithmSHA1,
		MGFHashAlgorithm: OIDDigestAlgorithmSHA1,
		SaltLength:       20,
		TrailerField:     1,
	}
	if len(raw.FullBytes) == 0 {
		return params, nil
	}

	wire := rsaPSSParametersASN1{SaltLength: 20, TrailerField: 1}
	rest, err := asn1.Unmarshal(raw.FullBytes, &wire)
	if err != nil {
		return params, fmt.Errorf("%w: decode parameters: %w", ErrInvalidPSSParameters, err)
	}
	if len(rest) > 0 {
		err := asn1.SyntaxError{Msg: "trailing data"}
		return params, fmt.Errorf("%w: decode parameters: %w", ErrInvalidPSSParameters, err)
	}
	if len(wire.HashAlgorithm.Algorithm) > 0 {
		if !nullOrAbsentParameters(wire.HashAlgorithm.Parameters) {
			return params, fmt.Errorf("%w: hash algorithm parameters must be NULL or absent", ErrInvalidPSSParameters)
		}
		params.HashAlgorithm = wire.HashAlgorithm.Algorithm
	}
	if len(wire.MaskGenAlgorithm.Algorithm) > 0 {
		if !wire.MaskGenAlgorithm.Algorithm.Equal(oidMaskGenAlgorithmMGF1) {
			return params, fmt.Errorf(
				"%w: mask generation algorithm OID %s is not MGF1",
				ErrInvalidPSSParameters,
				wire.MaskGenAlgorithm.Algorithm,
			)
		}
		var mgfHash pkix.AlgorithmIdentifier
		rest, err := asn1.Unmarshal(wire.MaskGenAlgorithm.Parameters.FullBytes, &mgfHash)
		if err != nil {
			return params, fmt.Errorf("%w: decode MGF1 hash algorithm: %w", ErrInvalidPSSParameters, err)
		}
		if len(rest) > 0 {
			err := asn1.SyntaxError{Msg: "trailing data"}
			return params, fmt.Errorf("%w: decode MGF1 hash algorithm: %w", ErrInvalidPSSParameters, err)
		}
		if !nullOrAbsentParameters(mgfHash.Parameters) {
			return params, fmt.Errorf("%w: MGF1 hash parameters must be NULL or absent", ErrInvalidPSSParameters)
		}
		params.MGFHashAlgorithm = mgfHash.Algorithm
	}
	if wire.SaltLength < 0 {
		return params, fmt.Errorf("%w: negative salt length %d", ErrInvalidPSSParameters, wire.SaltLength)
	}
	params.SaltLength = wire.SaltLength
	params.TrailerField = wire.TrailerField
	return params, nil
}

func nullOrAbsentParameters(raw asn1.RawValue) bool {
	return len(raw.FullBytes) == 0 || bytes.Equal(raw.FullBytes, asn1.NullBytes)
}

func rsaPSSSignatureAlgorithm(
	raw asn1.RawValue,
	digest asn1.ObjectIdentifier,
) (x509.SignatureAlgorithm, error) {
	params, err := parseRSAPSSParameters(raw)
	if err != nil {
		return x509.UnknownSignatureAlgorithm, err
	}
	if !params.HashAlgorithm.Equal(digest) {
		return x509.UnknownSignatureAlgorithm, fmt.Errorf(
			"%w: signer digest OID %s does not match RSASSA-PSS hash OID %s",
			ErrAlgorithmMismatch,
			digest,
			params.HashAlgorithm,
		)
	}
	if !params.MGFHashAlgorithm.Equal(params.HashAlgorithm) {
		return x509.UnknownSignatureAlgorithm, fmt.Errorf(
			"%w: RSASSA-PSS MGF1 hash OID %s does not match hash OID %s",
			ErrAlgorithmMismatch,
			params.MGFHashAlgorithm,
			params.HashAlgorithm,
		)
	}
	if params.TrailerField != 1 {
		return x509.UnknownSignatureAlgorithm, fmt.Errorf(
			"%w: unsupported trailer field %d",
			ErrInvalidPSSParameters,
			params.TrailerField,
		)
	}
	switch {
	case params.HashAlgorithm.Equal(OIDDigestAlgorithmSHA256):
		return x509.SHA256WithRSAPSS, nil
	case params.HashAlgorithm.Equal(OIDDigestAlgorithmSHA384):
		return x509.SHA384WithRSAPSS, nil
	case params.HashAlgorithm.Equal(OIDDigestAlgorithmSHA512):
		return x509.SHA512WithRSAPSS, nil
	}
	return x509.UnknownSignatureAlgorithm, fmt.Errorf(
		"%w: digest OID %s for RSASSA-PSS",
		ErrUnsupportedAlgorithm,
		params.HashAlgorithm,
	)
}

func dsaSignatureAlgorithm(digest asn1.ObjectIdentifier) (x509.SignatureAlgorithm, error) {
	switch {
	case digest.Equal(OIDDigestAlgorithmSHA1):
		return x509.DSAWithSHA1, nil
	case digest.Equal(OIDDigestAlgorithmSHA256):
		return x509.DSAWithSHA256, nil
	}
	return x509.UnknownSignatureAlgorithm, fmt.Errorf(
		"%w: digest OID %s for DSA",
		ErrUnsupportedAlgorithm,
		digest,
	)
}

func isEllipticCurveAlgorithm(oid asn1.ObjectIdentifier) bool {
	return oid.Equal(OIDEncryptionAlgorithmECPUBLICKEY) ||
		oid.Equal(OIDEncryptionAlgorithmECDSAP256) ||
		oid.Equal(OIDEncryptionAlgorithmECDSAP384) ||
		oid.Equal(OIDEncryptionAlgorithmECDSAP521)
}

func ecdsaSignatureAlgorithm(digest asn1.ObjectIdentifier) (x509.SignatureAlgorithm, error) {
	switch {
	case digest.Equal(OIDDigestAlgorithmSHA1):
		return x509.ECDSAWithSHA1, nil
	case digest.Equal(OIDDigestAlgorithmSHA256):
		return x509.ECDSAWithSHA256, nil
	case digest.Equal(OIDDigestAlgorithmSHA384):
		return x509.ECDSAWithSHA384, nil
	case digest.Equal(OIDDigestAlgorithmSHA512):
		return x509.ECDSAWithSHA512, nil
	}
	return x509.UnknownSignatureAlgorithm, fmt.Errorf(
		"%w: digest OID %s for ECDSA",
		ErrUnsupportedAlgorithm,
		digest,
	)
}

// GetCertFromCertsByIssuerAndSerial returns the certificate matching ias.
// It returns ErrMissingSignerIdentifier when ias is incomplete.
func GetCertFromCertsByIssuerAndSerial(
	certs []*x509.Certificate,
	ias issuerAndSerial,
) (*x509.Certificate, error) {
	if err := validateSignerIdentifier(ias); err != nil {
		return nil, fmt.Errorf("pkcs7: find signer certificate: %w", err)
	}
	for _, cert := range certs {
		if isCertMatchForIssuerAndSerial(cert, ias) {
			return cert, nil
		}
	}
	return nil, nil
}

func unmarshalAttribute(attrs []attribute, attributeType asn1.ObjectIdentifier, out interface{}) error {
	match := -1
	for i, attr := range attrs {
		if attr.Type.Equal(attributeType) {
			if match >= 0 {
				return fmt.Errorf(
					"%w, OID %s: duplicate attribute indexes %d and %d",
					ErrMalformedAttribute,
					attributeType,
					match+1,
					i+1,
				)
			}
			match = i
		}
	}
	if match < 0 {
		return fmt.Errorf("%w, OID %s: missing", ErrMalformedAttribute, attributeType)
	}

	rest, err := asn1.Unmarshal(attrs[match].Value.Bytes, out)
	if err != nil {
		return fmt.Errorf(
			"%w, attribute index %d, OID %s: decode ASN.1: %w",
			ErrMalformedAttribute,
			match+1,
			attributeType,
			err,
		)
	}
	if len(rest) > 0 {
		err := asn1.SyntaxError{Msg: "trailing data after attribute value"}
		return fmt.Errorf(
			"%w, attribute index %d, OID %s: decode ASN.1: %w",
			ErrMalformedAttribute,
			match+1,
			attributeType,
			err,
		)
	}
	return nil
}
