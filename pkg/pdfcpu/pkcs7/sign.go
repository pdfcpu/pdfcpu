/*
Copyright (c) 2015 Andrew Smith
Copyright 2026 The pdfcpu Authors.

Licensed under the MIT License. See LICENSE in this directory.
*/

package pkcs7

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sort"
)

// SignedData is an opaque data structure for creating signed data payloads
type SignedData struct {
	sd    signedData
	certs []*x509.Certificate
}

// NewSignedData initializes a PKCS7 SignedData struct that is ready to be signed via AddSigner.
func NewSignedData() (*SignedData, error) {
	sd := signedData{
		ContentInfo: contentInfo{ContentType: OIDData},
		Version:     1,
	}
	return &SignedData{sd: sd}, nil
}

// SignerInfoConfig are optional values to include when adding a signer
type SignerInfoConfig struct {
	ExtraSignedAttributes   []Attribute
	ExtraUnsignedAttributes []Attribute
}

type signedData struct {
	Version                    int                        `asn1:"default:1"`
	DigestAlgorithmIdentifiers []pkix.AlgorithmIdentifier `asn1:"set"`
	ContentInfo                contentInfo
	Certificates               rawCertificates `asn1:"optional,tag:0"`
	CRLs                       []asn1.RawValue `asn1:"optional,tag:1"`
	SignerInfos                []SignerInfo    `asn1:"set"`
}

// SignerInfo contains the algorithms, attributes, identity and encrypted
// digest for one PKCS#7 signer.
type SignerInfo struct {
	Version                   int `asn1:"default:1"`
	IssuerAndSerialNumber     issuerAndSerial
	DigestAlgorithm           pkix.AlgorithmIdentifier
	AuthenticatedAttributes   []attribute `asn1:"optional,omitempty,tag:0"` // RFC5652: signedAttrs [0] IMPLICIT SignedAttributes OPTIONAL
	DigestEncryptionAlgorithm pkix.AlgorithmIdentifier
	EncryptedDigest           []byte      `asn1:"octet"`
	UnauthenticatedAttributes []attribute `asn1:"optional,omitempty,tag:1"` // RFC5652: unsignedAttrs [1] IMPLICIT UnsignedAttributes OPTIONAL
}

type attribute struct {
	Type  asn1.ObjectIdentifier
	Value asn1.RawValue `asn1:"set"`
}

func marshalAttributes(attrs []attribute) ([]byte, error) {
	encodedAttributes, err := asn1.Marshal(struct {
		A []attribute `asn1:"set"`
	}{A: attrs})
	if err != nil {
		return nil, fmt.Errorf("pkcs7: marshal attributes: encode set: %w", err)
	}

	// Remove the leading sequence octets
	var raw asn1.RawValue
	rest, err := asn1.Unmarshal(encodedAttributes, &raw)
	if err != nil {
		return nil, fmt.Errorf("pkcs7: marshal attributes: read encoded set: %w", err)
	}
	if len(rest) > 0 {
		err := asn1.SyntaxError{Msg: "trailing data after encoded attribute set"}
		return nil, fmt.Errorf("pkcs7: marshal attributes: read encoded set: %w", err)
	}
	if raw.Class != asn1.ClassUniversal || raw.Tag != asn1.TagSequence || !raw.IsCompound {
		err := asn1.StructuralError{Msg: "encoded attribute set has invalid wrapper"}
		return nil, fmt.Errorf("pkcs7: marshal attributes: read encoded set: %w", err)
	}
	return raw.Bytes, nil
}

type rawCertificates struct {
	Raw asn1.RawContent
}

type issuerAndSerial struct {
	IssuerName   asn1.RawValue
	SerialNumber *big.Int
}

func addDigestAlgorithmUnique(list []pkix.AlgorithmIdentifier, oid asn1.ObjectIdentifier) []pkix.AlgorithmIdentifier {
	for _, alg := range list {
		if alg.Algorithm.Equal(oid) {
			return list
		}
	}
	return append(list, pkix.AlgorithmIdentifier{Algorithm: oid})
}

// AddSigner is a wrapper around AddSignerChain that adds a signer without a parent.
func (sd *SignedData) AddSigner(cert *x509.Certificate, pkey crypto.PrivateKey, messageDigest []byte, digestOid asn1.ObjectIdentifier, config SignerInfoConfig) error {
	var parents []*x509.Certificate
	return sd.AddSignerChain(cert, pkey, messageDigest, digestOid, parents, config)
}

// AddSignerChain signs attributes about the content and adds certificates
// and signers infos to the Signed Data. The certificate and private key
// of the end-entity signer are used to issue the signature, and any
// parent of that end-entity that need to be added to the list of
// certifications can be specified in the parents slice.
//
// The signature algorithm used to hash the data is the one of the end-entity certificate aka the cert.
func (sd *SignedData) AddSignerChain(cert *x509.Certificate, pkey crypto.PrivateKey, messageDigest []byte, digestOid asn1.ObjectIdentifier, parents []*x509.Certificate, config SignerInfoConfig) error {
	if err := validateSignerCertificates(sd, cert, parents); err != nil {
		return fmt.Errorf("pkcs7: add signer: %w", err)
	}
	encryptionOid, err := OIDForEncryptionAlgorithm(pkey, digestOid)
	if err != nil {
		return fmt.Errorf("pkcs7: add signer: select signature algorithm: %w", err)
	}
	hash, err := HashForOID(digestOid)
	if err != nil {
		return fmt.Errorf("pkcs7: add signer: select digest algorithm: %w", err)
	}
	if len(messageDigest) != hash.Size() {
		return fmt.Errorf(
			"pkcs7: add signer: message digest length %d, want %d",
			len(messageDigest),
			hash.Size(),
		)
	}
	if err := validateSigningKey(cert, pkey); err != nil {
		return fmt.Errorf("pkcs7: add signer: validate signing key: %w", err)
	}
	if err := validateExtraSignedAttributes(config.ExtraSignedAttributes); err != nil {
		return fmt.Errorf("pkcs7: add signer: %w", err)
	}

	attrs := &attributes{}
	attrs.Add(oidAttributeContentType, sd.sd.ContentInfo.ContentType)
	attrs.Add(oidAttributeMessageDigest, messageDigest)
	for _, attr := range config.ExtraSignedAttributes {
		attrs.Add(attr.Type, attr.Value)
	}
	authAttrs, err := attrs.ForMarshalling()
	if err != nil {
		return fmt.Errorf("pkcs7: add signer: marshal authenticated attributes: %w", err)
	}

	attrs = &attributes{}
	for _, attr := range config.ExtraUnsignedAttributes {
		attrs.Add(attr.Type, attr.Value)
	}
	unauthAttrs, err := attrs.ForMarshalling()
	if err != nil {
		return fmt.Errorf("pkcs7: add signer: marshal unauthenticated attributes: %w", err)
	}

	signature, err := signAttributes(authAttrs, pkey, hash)
	if err != nil {
		return fmt.Errorf("pkcs7: add signer: sign authenticated attributes: %w", err)
	}

	signerInfo := SignerInfo{
		Version:                   1, // RFC5652: If the SignerIdentifier is the CHOICE issuerAndSerialNumber, then the version MUST be 1
		IssuerAndSerialNumber:     signerIdentifier(cert),
		DigestAlgorithm:           pkix.AlgorithmIdentifier{Algorithm: digestOid},
		DigestEncryptionAlgorithm: pkix.AlgorithmIdentifier{Algorithm: encryptionOid},
		EncryptedDigest:           signature,
		AuthenticatedAttributes:   authAttrs,
		UnauthenticatedAttributes: unauthAttrs,
	}

	sd.sd.DigestAlgorithmIdentifiers = addDigestAlgorithmUnique(sd.sd.DigestAlgorithmIdentifiers, digestOid)
	sd.certs = append(sd.certs, cert)
	sd.certs = append(sd.certs, parents...)
	sd.sd.SignerInfos = append(sd.sd.SignerInfos, signerInfo)
	return nil
}

func validateSignerCertificates(sd *SignedData, cert *x509.Certificate, parents []*x509.Certificate) error {
	if sd == nil {
		return errors.New("signed data is missing")
	}
	if cert == nil {
		return errors.New("signer certificate is missing")
	}
	if len(cert.Raw) == 0 {
		return errors.New("signer certificate DER is missing")
	}
	if cert.SerialNumber == nil || cert.SerialNumber.Sign() <= 0 {
		return errors.New("signer certificate serial number is missing or invalid")
	}
	if len(cert.RawIssuer) == 0 {
		return errors.New("signer certificate issuer is missing")
	}
	if len(sd.sd.ContentInfo.ContentType) == 0 {
		return errors.New("signed data content type is missing")
	}
	for i, parent := range parents {
		if parent == nil {
			return fmt.Errorf("parent certificate index %d is missing", i+1)
		}
		if len(parent.Raw) == 0 {
			return fmt.Errorf("parent certificate index %d DER is missing", i+1)
		}
	}
	return nil
}

func validateExtraSignedAttributes(attrs []Attribute) error {
	seen := map[string]int{}
	for i, attr := range attrs {
		if attr.Type.Equal(oidAttributeContentType) || attr.Type.Equal(oidAttributeMessageDigest) {
			return fmt.Errorf("signed attribute index %d duplicates a mandatory attribute", i+1)
		}
		key := attr.Type.String()
		if previous, ok := seen[key]; ok {
			return fmt.Errorf(
				"signed attribute indexes %d and %d have duplicate OID %s",
				previous,
				i+1,
				attr.Type,
			)
		}
		seen[key] = i + 1
	}
	return nil
}

func validateSigningKey(cert *x509.Certificate, pkey crypto.PrivateKey) error {
	if err := validatePrivateSigningKey(pkey); err != nil {
		return fmt.Errorf("private key: %w", err)
	}
	if err := validateCertificatePublicKey(cert.PublicKey); err != nil {
		return fmt.Errorf("certificate public key: %w", err)
	}
	certKey, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return fmt.Errorf("marshal certificate public key: %w", err)
	}
	signer := pkey.(crypto.Signer)
	privateKey, err := x509.MarshalPKIXPublicKey(signer.Public())
	if err != nil {
		return fmt.Errorf("marshal private-key public key: %w", err)
	}
	if !bytes.Equal(certKey, privateKey) {
		return errors.New("private key does not match signer certificate")
	}
	return nil
}

func validatePrivateSigningKey(pkey crypto.PrivateKey) error {
	switch key := pkey.(type) {
	case *rsa.PrivateKey:
		if key == nil {
			return errors.New("RSA private key is missing")
		}
		if err := key.Validate(); err != nil {
			return fmt.Errorf("RSA private key is invalid: %w", err)
		}
	case *ecdsa.PrivateKey:
		if key == nil || key.Curve == nil || key.X == nil || key.Y == nil || key.D == nil {
			return errors.New("ECDSA private key is missing or invalid")
		}
		if !key.Curve.IsOnCurve(key.X, key.Y) {
			return errors.New("ECDSA private key public point is invalid")
		}
	default:
		return fmt.Errorf("%w: private key type %T", ErrUnsupportedAlgorithm, pkey)
	}
	return nil
}

func signerIdentifier(cert *x509.Certificate) issuerAndSerial {
	return issuerAndSerial{
		IssuerName:   asn1.RawValue{FullBytes: cert.RawIssuer},
		SerialNumber: new(big.Int).Set(cert.SerialNumber),
	}
}

// AddCertificate adds cert to the payload.
func (sd *SignedData) AddCertificate(cert *x509.Certificate) error {
	if sd == nil {
		return errors.New("pkcs7: add certificate: signed data is missing")
	}
	if cert == nil {
		return errors.New("pkcs7: add certificate: certificate is missing")
	}
	if len(cert.Raw) == 0 {
		return errors.New("pkcs7: add certificate: certificate DER is missing")
	}
	if _, err := x509.ParseCertificate(cert.Raw); err != nil {
		return fmt.Errorf("pkcs7: add certificate: parse DER: %w", certificateParseError(err))
	}
	sd.certs = append(sd.certs, cert)
	return nil
}

// Even though, the tag & length are stripped out during marshalling the
// RawContent, we have to encode it into the RawContent. If its missing,
// then `asn1.Marshal()` will strip out the certificate wrapper instead.
func marshalCertificateBytes(certs []byte) (rawCertificates, error) {
	val := asn1.RawValue{
		Bytes:      certs,
		Class:      asn1.ClassContextSpecific,
		Tag:        0,
		IsCompound: true,
	}
	b, err := asn1.Marshal(val)
	if err != nil {
		return rawCertificates{}, fmt.Errorf("encode ASN.1 wrapper: %w", err)
	}
	return rawCertificates{Raw: b}, nil
}

func marshalCertificates(certs []*x509.Certificate) (rawCertificates, error) {
	if len(certs) == 0 {
		return rawCertificates{}, nil
	}
	der := make([][]byte, 0, len(certs))
	for i, cert := range certs {
		if cert == nil {
			return rawCertificates{}, fmt.Errorf("certificate index %d is missing", i+1)
		}
		if len(cert.Raw) == 0 {
			return rawCertificates{}, fmt.Errorf("certificate index %d DER is missing", i+1)
		}
		if _, err := x509.ParseCertificate(cert.Raw); err != nil {
			return rawCertificates{}, fmt.Errorf(
				"certificate index %d: parse DER: %w",
				i+1,
				certificateParseError(err),
			)
		}
		der = append(der, bytes.Clone(cert.Raw))
	}
	sort.Slice(der, func(i, j int) bool {
		return bytes.Compare(der[i], der[j]) < 0
	})
	var certsBuf []byte
	for i, bb := range der {
		if i == 0 || !bytes.Equal(bb, der[i-1]) {
			certsBuf = append(certsBuf, bb...)
		}
	}
	rawCerts, err := marshalCertificateBytes(certsBuf)
	if err != nil {
		return rawCertificates{}, fmt.Errorf("marshal certificate set: %w", err)
	}
	return rawCerts, nil
}

// Finish encodes sd as PKCS#7 signed data.
func (sd *SignedData) Finish() ([]byte, error) {
	if sd == nil {
		return nil, errors.New("pkcs7: finish signed data: signed data is missing")
	}
	certsRaw, err := marshalCertificates(sd.certs)
	if err != nil {
		return nil, fmt.Errorf("pkcs7: finish signed data: certificates: %w", err)
	}
	signedData := sd.sd
	signedData.Certificates = certsRaw
	inner, err := asn1.Marshal(signedData)
	if err != nil {
		return nil, fmt.Errorf("pkcs7: finish signed data: encode signed data: %w", err)
	}
	// Wrap in outer ContentInfo [0] EXPLICIT
	outer := contentInfo{
		ContentType: OIDSignedData,
		Content: asn1.RawValue{
			Class:      2,
			Tag:        0,
			Bytes:      inner,
			IsCompound: true,
		},
	}
	data, err := asn1.Marshal(outer)
	if err != nil {
		return nil, fmt.Errorf("pkcs7: finish signed data: encode content info: %w", err)
	}
	return data, nil
}

// signs the DER encoded form of the attributes with the private key
func signAttributes(attrs []attribute, pkey crypto.PrivateKey, digestAlg crypto.Hash) ([]byte, error) {
	return signAttributesWithRandom(attrs, pkey, digestAlg, rand.Reader)
}

func signAttributesWithRandom(
	attrs []attribute,
	pkey crypto.PrivateKey,
	digestAlg crypto.Hash,
	random io.Reader,
) ([]byte, error) {
	attrBytes, err := marshalAttributes(attrs)
	if err != nil {
		return nil, fmt.Errorf("pkcs7: sign attributes: marshal attributes: %w", err)
	}
	if !digestAlg.Available() {
		return nil, fmt.Errorf("%w: signing hash %d is unavailable", ErrUnsupportedAlgorithm, digestAlg)
	}
	if err := validatePrivateSigningKey(pkey); err != nil {
		return nil, fmt.Errorf("pkcs7: sign attributes: validate private key: %w", err)
	}
	if random == nil {
		return nil, errors.New("pkcs7: sign attributes: random source is missing")
	}
	h := digestAlg.New()
	_, _ = h.Write(attrBytes)
	hash := h.Sum(nil)
	key := pkey.(crypto.Signer)
	signature, err := key.Sign(random, hash, digestAlg)
	if err != nil {
		return nil, fmt.Errorf("pkcs7: sign attributes: sign digest: %w", err)
	}
	if len(signature) == 0 {
		return nil, errors.New("pkcs7: sign attributes: signer returned an empty signature")
	}
	return signature, nil
}
