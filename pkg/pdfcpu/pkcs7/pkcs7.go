/*
Copyright (c) 2015 Andrew Smith
Copyright 2026 The pdfcpu Authors.

Licensed under the MIT License. See LICENSE in this directory.
*/

// Package pkcs7 implements parsing and generation of some PKCS#7 structures.
package pkcs7

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"fmt"
	"sort"

	_ "crypto/sha1"   // #nosec G505 -- required to validate existing legacy PKCS#7 signatures.
	_ "crypto/sha256" // register crypto.SHA224 and crypto.SHA256
	_ "crypto/sha512" // register crypto.SHA384 and crypto.SHA512
)

// PKCS7 represents a parsed PKCS#7 structure.
type PKCS7 struct {
	Content      []byte
	ContentType  asn1.ObjectIdentifier
	Certificates []*x509.Certificate
	CRLs         []*x509.RevocationList
	Signers      []SignerInfo
}

type contentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue `asn1:"explicit,optional,tag:0"`
}

// ErrUnsupportedContentType is returned when a PKCS7 content is not supported.
// Currently only Signed Data (1.2.840.113549.1.7.2) is supported.
var ErrUnsupportedContentType = errors.New("pkcs7: cannot parse data: unimplemented content type")

// ErrEmptyInput identifies missing PKCS#7 input data.
var ErrEmptyInput = errors.New("pkcs7: input data is empty")

// ErrUnsupportedAlgorithm identifies a cryptographic algorithm unsupported by this package.
var ErrUnsupportedAlgorithm = errors.New("pkcs7: unsupported algorithm")

// ErrAlgorithmMismatch identifies inconsistent CMS digest and signature algorithms.
var ErrAlgorithmMismatch = errors.New("pkcs7: algorithm mismatch")

// ErrInvalidPSSParameters identifies malformed or unsupported RSASSA-PSS parameters.
var ErrInvalidPSSParameters = errors.New("pkcs7: invalid RSASSA-PSS parameters")

// ErrSignatureMismatch identifies an actual cryptographic signature mismatch.
var ErrSignatureMismatch = errors.New("pkcs7: cryptographic signature mismatch")

// ErrCertificateParse identifies malformed certificate data embedded in a PKCS#7 message.
var ErrCertificateParse = errors.New("pkcs7: certificate parse error")

// ErrMissingSignerIdentifier identifies an incomplete signer issuer-and-serial identifier.
var ErrMissingSignerIdentifier = errors.New("pkcs7: missing signer identifier")

// ErrMalformedAttribute identifies a missing or malformed signed attribute.
var ErrMalformedAttribute = errors.New("pkcs7: malformed attribute")

// CertificateParseError reports malformed certificate data embedded in a
// PKCS#7 message while preserving the underlying ASN.1 or X.509 parse error.
type CertificateParseError struct {
	cause error
}

// Error implements error.
func (e *CertificateParseError) Error() string {
	return fmt.Sprintf("%s: %v", ErrCertificateParse, e.cause)
}

// Is classifies this error as ErrCertificateParse.
func (e *CertificateParseError) Is(target error) bool {
	return target == ErrCertificateParse
}

// Unwrap returns the underlying ASN.1 or X.509 parse error.
func (e *CertificateParseError) Unwrap() error {
	return e.cause
}

func certificateParseError(err error) error {
	return &CertificateParseError{cause: err}
}

// PKCS#7 content, attribute, digest, signature and curve object identifiers.
var (
	// Signed Data OIDs
	OIDData                   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}
	OIDSignedData             = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}
	oidAttributeContentType   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 3}
	oidAttributeMessageDigest = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 4}

	// Digest Algorithms
	OIDDigestAlgorithmSHA1   = asn1.ObjectIdentifier{1, 3, 14, 3, 2, 26}
	OIDDigestAlgorithmSHA256 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}
	OIDDigestAlgorithmSHA384 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 2}
	OIDDigestAlgorithmSHA512 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 3}

	OIDDigestAlgorithmDSA     = asn1.ObjectIdentifier{1, 2, 840, 10040, 4, 1}
	OIDDigestAlgorithmDSASHA1 = asn1.ObjectIdentifier{1, 2, 840, 10040, 4, 3}

	OIDDigestAlgorithmECDSASHA1   = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 1}
	OIDDigestAlgorithmECDSASHA256 = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 2}
	OIDDigestAlgorithmECDSASHA384 = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 3}
	OIDDigestAlgorithmECDSASHA512 = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 4}

	// Signature Algorithms
	OIDEncryptionAlgorithmRSA       = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 1}
	OIDEncryptionAlgorithmRSASHA1   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 5}
	OIDEncryptionAlgorithmRSAPSS    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 10}
	OIDEncryptionAlgorithmRSASHA256 = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 11}
	OIDEncryptionAlgorithmRSASHA384 = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 12}
	OIDEncryptionAlgorithmRSASHA512 = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 13}

	OIDEncryptionAlgorithmECDSAP256   = asn1.ObjectIdentifier{1, 2, 840, 10045, 3, 1, 7} // 256 bit elliptic curve
	OIDEncryptionAlgorithmECDSAP384   = asn1.ObjectIdentifier{1, 3, 132, 0, 34}          // 384-bit elliptic curve
	OIDEncryptionAlgorithmECDSAP521   = asn1.ObjectIdentifier{1, 3, 132, 0, 35}          // 512-bit elliptic curve!
	OIDEncryptionAlgorithmECPUBLICKEY = asn1.ObjectIdentifier{1, 2, 840, 10045, 2, 1}    // ecPublicKey

	OIDEncryptionAlgorithmEd25519 = asn1.ObjectIdentifier{1, 3, 101, 112}
)

// HashForOID returns the Go hash corresponding to oid.
func HashForOID(oid asn1.ObjectIdentifier) (crypto.Hash, error) {
	switch {
	case oid.Equal(OIDDigestAlgorithmSHA1), oid.Equal(OIDDigestAlgorithmECDSASHA1),
		oid.Equal(OIDDigestAlgorithmDSA), oid.Equal(OIDDigestAlgorithmDSASHA1),
		oid.Equal(OIDEncryptionAlgorithmRSA):
		return crypto.SHA1, nil
	case oid.Equal(OIDDigestAlgorithmSHA256), oid.Equal(OIDDigestAlgorithmECDSASHA256):
		return crypto.SHA256, nil
	case oid.Equal(OIDDigestAlgorithmSHA384), oid.Equal(OIDDigestAlgorithmECDSASHA384):
		return crypto.SHA384, nil
	case oid.Equal(OIDDigestAlgorithmSHA512), oid.Equal(OIDDigestAlgorithmECDSASHA512):
		return crypto.SHA512, nil
	}
	return crypto.Hash(0), fmt.Errorf("%w: digest OID %s", ErrUnsupportedAlgorithm, oid)
}

// OIDForEncryptionAlgorithm takes the private key type of the signer and
// the OID of a digest algorithm to return the appropriate signerInfo.DigestEncryptionAlgorithm
func OIDForEncryptionAlgorithm(pkey crypto.PrivateKey, OIDDigestAlg asn1.ObjectIdentifier) (asn1.ObjectIdentifier, error) {
	switch pkey.(type) {
	case *rsa.PrivateKey:
		switch {
		case OIDDigestAlg.Equal(OIDDigestAlgorithmSHA1):
			return OIDEncryptionAlgorithmRSASHA1, nil
		case OIDDigestAlg.Equal(OIDDigestAlgorithmSHA256):
			return OIDEncryptionAlgorithmRSASHA256, nil
		case OIDDigestAlg.Equal(OIDDigestAlgorithmSHA384):
			return OIDEncryptionAlgorithmRSASHA384, nil
		case OIDDigestAlg.Equal(OIDDigestAlgorithmSHA512):
			return OIDEncryptionAlgorithmRSASHA512, nil
		}
		return nil, fmt.Errorf(
			"%w: digest OID %s for RSA private key",
			ErrUnsupportedAlgorithm,
			OIDDigestAlg,
		)
	case *ecdsa.PrivateKey:
		switch {
		case OIDDigestAlg.Equal(OIDDigestAlgorithmSHA1):
			return OIDDigestAlgorithmECDSASHA1, nil
		case OIDDigestAlg.Equal(OIDDigestAlgorithmSHA256):
			return OIDDigestAlgorithmECDSASHA256, nil
		case OIDDigestAlg.Equal(OIDDigestAlgorithmSHA384):
			return OIDDigestAlgorithmECDSASHA384, nil
		case OIDDigestAlg.Equal(OIDDigestAlgorithmSHA512):
			return OIDDigestAlgorithmECDSASHA512, nil
		}
		return nil, fmt.Errorf(
			"%w: digest OID %s for ECDSA private key",
			ErrUnsupportedAlgorithm,
			OIDDigestAlg,
		)
	}
	return nil, fmt.Errorf("%w: private key type %T", ErrUnsupportedAlgorithm, pkey)
}

// Parse decodes a BER- or DER-encoded PKCS#7 package.
func Parse(data []byte) (*PKCS7, error) {
	if len(data) == 0 {
		return nil, ErrEmptyInput
	}
	var info contentInfo
	der, err := ber2der(data)
	if err != nil {
		return nil, fmt.Errorf("pkcs7: parse BER: %w", err)
	}
	rest, err := asn1.Unmarshal(der, &info)
	if err != nil {
		return nil, fmt.Errorf("pkcs7: parse content info: %w", err)
	}
	if len(rest) > 0 {
		err := asn1.SyntaxError{Msg: "trailing data"}
		return nil, fmt.Errorf("pkcs7: parse content info: %w", err)
	}

	switch {
	case info.ContentType.Equal(OIDSignedData):
		p7, err := parseSignedData(info.Content.Bytes)
		if err != nil {
			return nil, fmt.Errorf("pkcs7: parse signed data: %w", err)
		}
		return p7, nil
	}
	return nil, ErrUnsupportedContentType
}

func isCertMatchForIssuerAndSerial(cert *x509.Certificate, ias issuerAndSerial) bool {
	if cert == nil || cert.SerialNumber == nil {
		return false
	}
	return cert.SerialNumber.Cmp(ias.SerialNumber) == 0 && bytes.Equal(cert.RawIssuer, ias.IssuerName.FullBytes)
}

func validateSignerIdentifier(ias issuerAndSerial) error {
	if ias.SerialNumber == nil || len(ias.IssuerName.FullBytes) == 0 {
		return ErrMissingSignerIdentifier
	}
	return nil
}

// Attribute represents a key-value attribute whose value must be marshalable by
// encoding/asn1.
type Attribute struct {
	Type  asn1.ObjectIdentifier
	Value interface{}
}

type attributes struct {
	types  []asn1.ObjectIdentifier
	values []interface{}
}

// Add adds the attribute, maintaining insertion order
func (attrs *attributes) Add(attrType asn1.ObjectIdentifier, value interface{}) {
	attrs.types = append(attrs.types, attrType)
	attrs.values = append(attrs.values, value)
}

type sortableAttribute struct {
	SortKey   []byte
	Attribute attribute
}

type attributeSet []sortableAttribute

func (sa attributeSet) Len() int {
	return len(sa)
}

func (sa attributeSet) Less(i, j int) bool {
	return bytes.Compare(sa[i].SortKey, sa[j].SortKey) < 0
}

func (sa attributeSet) Swap(i, j int) {
	sa[i], sa[j] = sa[j], sa[i]
}

func (sa attributeSet) Attributes() []attribute {
	attrs := make([]attribute, len(sa))
	for i, attr := range sa {
		attrs[i] = attr.Attribute
	}
	return attrs
}

func (attrs *attributes) ForMarshalling() ([]attribute, error) {
	if attrs == nil {
		return nil, nil
	}
	if len(attrs.types) != len(attrs.values) {
		return nil, fmt.Errorf(
			"pkcs7: marshal attributes: mismatched types and values: %d types, %d values",
			len(attrs.types),
			len(attrs.values),
		)
	}
	sortables := make(attributeSet, len(attrs.types))
	for i := range sortables {
		attrType := attrs.types[i]
		attrValue := attrs.values[i]
		asn1Value, err := asn1.Marshal(attrValue)
		if err != nil {
			return nil, fmt.Errorf(
				"pkcs7: marshal attributes: attribute index %d: encode value: %w",
				i+1,
				err,
			)
		}
		attr := attribute{
			Type:  attrType,
			Value: asn1.RawValue{Tag: asn1.TagSet, IsCompound: true, Bytes: asn1Value},
		}
		encoded, err := asn1.Marshal(attr)
		if err != nil {
			return nil, fmt.Errorf(
				"pkcs7: marshal attributes: attribute index %d: encode attribute: %w",
				i+1,
				err,
			)
		}
		sortables[i] = sortableAttribute{
			SortKey:   encoded,
			Attribute: attr,
		}
	}
	sort.Sort(sortables)
	return sortables.Attributes(), nil
}
