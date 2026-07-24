/*
Copyright 2026 The pdfcpu Authors.

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

package pkcs7

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"math/big"
	"slices"
	"strings"
	"testing"
	"time"
)

var _ func(*x509.Certificate, SignerInfo, []byte) error = CheckSignature

var _ func(*x509.Certificate, SignerInfo, []byte, asn1.ObjectIdentifier) error = CheckSignatureWithContentType

func TestCheckSignatureRejectsMissingCertificateWithoutPanic(t *testing.T) {
	requireErrorWithoutPanic(t, "signer certificate is missing", func() error {
		return CheckSignature(nil, SignerInfo{}, nil)
	})
}

func TestCheckSignaturePreservesUnsupportedAlgorithm(t *testing.T) {
	tests := []struct {
		name       string
		encryption asn1.ObjectIdentifier
		digest     asn1.ObjectIdentifier
	}{
		{"unknown signature", asn1.ObjectIdentifier{1, 2, 3}, OIDDigestAlgorithmSHA256},
		{"RSA digest", OIDEncryptionAlgorithmRSA, asn1.ObjectIdentifier{1, 2, 3}},
		{"RSA-PSS digest", OIDEncryptionAlgorithmRSAPSS, OIDDigestAlgorithmSHA1},
		{"DSA digest", OIDDigestAlgorithmDSA, OIDDigestAlgorithmSHA512},
		{"ECDSA digest", OIDEncryptionAlgorithmECPUBLICKEY, asn1.ObjectIdentifier{1, 2, 3}},
	}
	cert := &x509.Certificate{}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			signer := signerWithAlgorithms(tt.encryption, tt.digest)
			err := CheckSignature(cert, signer, []byte("content"))
			if !errors.Is(err, ErrUnsupportedAlgorithm) {
				t.Fatalf("got %v, want ErrUnsupportedAlgorithm", err)
			}
			if !strings.Contains(err.Error(), "verify signature: select algorithm") {
				t.Fatalf("got %v, want algorithm-selection phase", err)
			}
		})
	}
}

func TestCheckSignaturePreservesCryptographicMismatch(t *testing.T) {
	cert := &x509.Certificate{
		PublicKey: &rsa.PublicKey{
			N: rsaTestModulus(),
			E: 65537,
		},
	}
	signer := signerWithAlgorithms(OIDEncryptionAlgorithmRSA, OIDDigestAlgorithmSHA256)
	signer.EncryptedDigest = []byte{0x01}

	err := CheckSignature(cert, signer, []byte("content"))
	if !errors.Is(err, ErrSignatureMismatch) {
		t.Fatalf("got %v, want ErrSignatureMismatch", err)
	}
	if !errors.Is(err, rsa.ErrVerification) {
		t.Fatalf("got %v, want rsa.ErrVerification", err)
	}
	if !strings.Contains(err.Error(), "verify cryptographic signature") {
		t.Fatalf("got %v, want cryptographic-verification phase", err)
	}
}

func TestCheckSignatureRejectsMutatedEncapsulatedContent(t *testing.T) {
	content := []byte("signed content")
	cert, p7 := signedCMSFixture(t, OIDData, content)
	p7.Content = []byte("mutated content")

	err := CheckSignature(cert, p7.Signers[0], p7.Content)
	var digestMismatchErr *MessageDigestMismatchError
	if !errors.As(err, &digestMismatchErr) {
		t.Fatalf("got %v, want MessageDigestMismatchError", err)
	}
	if !strings.Contains(err.Error(), "verify signed content digest") {
		t.Fatalf("got %v, want content-digest phase", err)
	}
}

func TestCheckSignatureValidatesSignedContentType(t *testing.T) {
	cert, p7 := signedCMSFixture(t, OIDData, []byte("signed content"))
	signer := p7.Signers[0]
	tests := []struct {
		name   string
		mutate func(*SignerInfo)
	}{
		{
			name: "Missing",
			mutate: func(signer *SignerInfo) {
				signer.AuthenticatedAttributes = slices.DeleteFunc(
					signer.AuthenticatedAttributes,
					func(attr attribute) bool {
						return attr.Type.Equal(oidAttributeContentType)
					},
				)
			},
		},
		{
			name: "Duplicate",
			mutate: func(signer *SignerInfo) {
				for _, attr := range signer.AuthenticatedAttributes {
					if attr.Type.Equal(oidAttributeContentType) {
						signer.AuthenticatedAttributes = append(signer.AuthenticatedAttributes, attr)
						return
					}
				}
			},
		},
		{
			name: "Mismatch",
			mutate: func(signer *SignerInfo) {
				for i := range signer.AuthenticatedAttributes {
					if signer.AuthenticatedAttributes[i].Type.Equal(oidAttributeContentType) {
						signer.AuthenticatedAttributes[i].Value = attributeValue(t, OIDSignedData)
						return
					}
				}
			},
		},
		{
			name: "MultipleValues",
			mutate: func(signer *SignerInfo) {
				for i := range signer.AuthenticatedAttributes {
					if signer.AuthenticatedAttributes[i].Type.Equal(oidAttributeContentType) {
						first := attributeValue(t, OIDData)
						second := attributeValue(t, OIDData)
						signer.AuthenticatedAttributes[i].Value.Bytes = append(first.Bytes, second.Bytes...)
						return
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutated := signer
			mutated.AuthenticatedAttributes = append([]attribute(nil), signer.AuthenticatedAttributes...)
			tt.mutate(&mutated)

			err := CheckSignature(cert, mutated, p7.Content)
			if !errors.Is(err, ErrMalformedAttribute) {
				t.Fatalf("got %v, want ErrMalformedAttribute", err)
			}
			if !strings.Contains(err.Error(), "validate signed content type") {
				t.Fatalf("got %v, want signed-content-type phase", err)
			}
		})
	}
}

func TestCheckSignatureValidatesSignedMessageDigestCardinality(t *testing.T) {
	cert, p7 := signedCMSFixture(t, OIDData, []byte("signed content"))
	signer := p7.Signers[0]
	tests := []struct {
		name   string
		mutate func(*SignerInfo)
	}{
		{
			name: "Missing",
			mutate: func(signer *SignerInfo) {
				signer.AuthenticatedAttributes = slices.DeleteFunc(
					signer.AuthenticatedAttributes,
					func(attr attribute) bool {
						return attr.Type.Equal(oidAttributeMessageDigest)
					},
				)
			},
		},
		{
			name: "Duplicate",
			mutate: func(signer *SignerInfo) {
				for _, attr := range signer.AuthenticatedAttributes {
					if attr.Type.Equal(oidAttributeMessageDigest) {
						signer.AuthenticatedAttributes = append(signer.AuthenticatedAttributes, attr)
						return
					}
				}
			},
		},
		{
			name: "MultipleValues",
			mutate: func(signer *SignerInfo) {
				for i := range signer.AuthenticatedAttributes {
					if signer.AuthenticatedAttributes[i].Type.Equal(oidAttributeMessageDigest) {
						first := attributeValue(t, []byte{1})
						second := attributeValue(t, []byte{2})
						signer.AuthenticatedAttributes[i].Value.Bytes = append(first.Bytes, second.Bytes...)
						return
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutated := signer
			mutated.AuthenticatedAttributes = append([]attribute(nil), signer.AuthenticatedAttributes...)
			tt.mutate(&mutated)

			err := CheckSignature(cert, mutated, p7.Content)
			if !errors.Is(err, ErrMalformedAttribute) {
				t.Fatalf("got %v, want ErrMalformedAttribute", err)
			}
			if !strings.Contains(err.Error(), "verify signed content digest") {
				t.Fatalf("got %v, want content-digest phase", err)
			}
		})
	}
}

func TestCheckSignatureRejectsMutatedDTSContent(t *testing.T) {
	contentType := asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 1, 4}
	original := testTSTInfo{
		Version:      1,
		Policy:       asn1.ObjectIdentifier{1, 2, 3, 4},
		SerialNumber: big.NewInt(1),
		GenTime:      time.Date(2026, time.July, 26, 12, 0, 0, 0, time.UTC),
	}
	original.MessageImprint.HashAlgorithm = pkix.AlgorithmIdentifier{Algorithm: OIDDigestAlgorithmSHA256}
	original.MessageImprint.HashedMessage = []byte{1, 2, 3, 4}
	content := marshalTestTSTInfo(t, original)
	cert, p7 := signedCMSFixture(t, contentType, content)

	tests := []struct {
		name   string
		mutate func(*testTSTInfo)
	}{
		{
			name: "GenTime",
			mutate: func(info *testTSTInfo) {
				info.GenTime = info.GenTime.Add(time.Minute)
			},
		},
		{
			name: "MessageImprint",
			mutate: func(info *testTSTInfo) {
				info.MessageImprint.HashedMessage[0] ^= 0xff
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutated := original
			mutated.MessageImprint.HashedMessage = append(
				[]byte(nil),
				original.MessageImprint.HashedMessage...,
			)
			tt.mutate(&mutated)
			mutatedP7 := *p7
			mutatedP7.Content = marshalTestTSTInfo(t, mutated)

			err := CheckSignatureWithContentType(
				cert,
				mutatedP7.Signers[0],
				mutatedP7.Content,
				mutatedP7.ContentType,
			)
			var digestMismatchErr *MessageDigestMismatchError
			if !errors.As(err, &digestMismatchErr) {
				t.Fatalf("got %v, want MessageDigestMismatchError", err)
			}
		})
	}
}

type testTSTInfo struct {
	Version        int
	Policy         asn1.ObjectIdentifier
	MessageImprint struct {
		HashAlgorithm pkix.AlgorithmIdentifier
		HashedMessage []byte
	}
	SerialNumber *big.Int
	GenTime      time.Time
}

func marshalTestTSTInfo(t *testing.T, info testTSTInfo) []byte {
	t.Helper()
	content, err := asn1.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func signedCMSFixture(
	t *testing.T,
	contentType asn1.ObjectIdentifier,
	content []byte,
) (*x509.Certificate, *PKCS7) {
	t.Helper()
	cert, key := signerCertificate(t, "CMS Signer")
	digest := sha256.Sum256(content)
	signedData, err := NewSignedData()
	if err != nil {
		t.Fatal(err)
	}
	encodedContent, err := asn1.Marshal(content)
	if err != nil {
		t.Fatal(err)
	}
	signedData.sd.ContentInfo = contentInfo{
		ContentType: contentType,
		Content: asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			IsCompound: true,
			Bytes:      encodedContent,
		},
	}
	if err := signedData.AddSigner(
		cert,
		key,
		digest[:],
		OIDDigestAlgorithmSHA256,
		SignerInfoConfig{},
	); err != nil {
		t.Fatal(err)
	}
	raw, err := signedData.Finish()
	if err != nil {
		t.Fatal(err)
	}
	p7, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckSignatureWithContentType(cert, p7.Signers[0], p7.Content, p7.ContentType); err != nil {
		t.Fatalf("valid CMS fixture: %v", err)
	}
	return cert, p7
}

func TestCheckSignaturePreservesUnsupportedPublicKey(t *testing.T) {
	cert := &x509.Certificate{}
	signer := signerWithAlgorithms(OIDEncryptionAlgorithmRSA, OIDDigestAlgorithmSHA256)
	var got error
	requireErrorWithoutPanic(t, "verify cryptographic signature", func() error {
		got = CheckSignature(cert, signer, []byte("content"))
		return got
	})
	if !errors.Is(got, x509.ErrUnsupportedAlgorithm) {
		t.Fatalf("got %v, want x509.ErrUnsupportedAlgorithm", got)
	}
}

func TestCheckSignatureRejectsTypedNilRSAPublicKeyWithoutPanic(t *testing.T) {
	cert := &x509.Certificate{PublicKey: (*rsa.PublicKey)(nil)}
	signer := signerWithAlgorithms(OIDEncryptionAlgorithmRSA, OIDDigestAlgorithmSHA256)
	requireErrorWithoutPanic(t, "RSA public key is invalid", func() error {
		return CheckSignature(cert, signer, []byte("content"))
	})
}

func TestCheckSignatureRejectsInvalidRSAPublicKeyWithoutPanic(t *testing.T) {
	cert := &x509.Certificate{PublicKey: &rsa.PublicKey{E: 65537}}
	signer := signerWithAlgorithms(OIDEncryptionAlgorithmRSA, OIDDigestAlgorithmSHA256)
	requireErrorWithoutPanic(t, "RSA public key is invalid", func() error {
		return CheckSignature(cert, signer, []byte("content"))
	})
}

func TestCheckSignaturePreservesAttributeMarshalCause(t *testing.T) {
	cert := &x509.Certificate{}
	signer := signerWithAlgorithms(OIDEncryptionAlgorithmRSA, OIDDigestAlgorithmSHA256)
	signer.AuthenticatedAttributes = []attribute{{Type: asn1.ObjectIdentifier{3}}}

	err := CheckSignature(cert, signer, nil)
	if err == nil {
		t.Fatal("expected attribute marshal error")
	}
	if !strings.Contains(err.Error(), "marshal authenticated attributes") {
		t.Fatalf("got %v, want attribute-marshalling phase", err)
	}
	var structuralErr asn1.StructuralError
	if !errors.As(err, &structuralErr) {
		t.Fatalf("got %v, want preserved ASN.1 structural cause", err)
	}
}

func TestGetSignatureAlgorithmRejectsDigestMismatch(t *testing.T) {
	tests := []struct {
		name      string
		signature asn1.ObjectIdentifier
		digest    asn1.ObjectIdentifier
	}{
		{"RSA", OIDEncryptionAlgorithmRSASHA256, OIDDigestAlgorithmSHA384},
		{"ECDSA", OIDDigestAlgorithmECDSASHA384, OIDDigestAlgorithmSHA256},
		{"DSA", OIDDigestAlgorithmDSASHA1, OIDDigestAlgorithmSHA256},
		{"Ed25519", OIDEncryptionAlgorithmEd25519, OIDDigestAlgorithmSHA256},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getSignatureAlgorithm(
				pkix.AlgorithmIdentifier{Algorithm: tt.signature},
				pkix.AlgorithmIdentifier{Algorithm: tt.digest},
			)
			if !errors.Is(err, ErrAlgorithmMismatch) {
				t.Fatalf("got %v, want ErrAlgorithmMismatch", err)
			}
		})
	}
}

func TestCheckSignaturePreservesAlgorithmProfileSentinels(t *testing.T) {
	tests := []struct {
		name   string
		signer SignerInfo
		want   error
	}{
		{
			name: "AlgorithmMismatch",
			signer: signerWithAlgorithms(
				OIDEncryptionAlgorithmRSASHA256,
				OIDDigestAlgorithmSHA384,
			),
			want: ErrAlgorithmMismatch,
		},
		{
			name: "InvalidPSSParameters",
			signer: SignerInfo{
				DigestAlgorithm: pkix.AlgorithmIdentifier{
					Algorithm: OIDDigestAlgorithmSHA256,
				},
				DigestEncryptionAlgorithm: pkix.AlgorithmIdentifier{
					Algorithm: OIDEncryptionAlgorithmRSAPSS,
					Parameters: testRSAPSSParameters(
						t,
						OIDDigestAlgorithmSHA256,
						OIDDigestAlgorithmSHA256,
						32,
						2,
					),
				},
			},
			want: ErrInvalidPSSParameters,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckSignature(&x509.Certificate{}, tt.signer, []byte("content"))
			if !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
			if !strings.Contains(err.Error(), "verify signature: select algorithm") {
				t.Fatalf("got %v, want algorithm-selection phase", err)
			}
		})
	}
}

func TestParseRSAPSSParameters(t *testing.T) {
	raw := testRSAPSSParameters(
		t,
		OIDDigestAlgorithmSHA256,
		OIDDigestAlgorithmSHA384,
		31,
		2,
	)
	params, err := parseRSAPSSParameters(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !params.HashAlgorithm.Equal(OIDDigestAlgorithmSHA256) ||
		!params.MGFHashAlgorithm.Equal(OIDDigestAlgorithmSHA384) ||
		params.SaltLength != 31 ||
		params.TrailerField != 2 {
		t.Fatalf("unexpected RSASSA-PSS parameters: %+v", params)
	}

	defaults, err := parseRSAPSSParameters(asn1.RawValue{})
	if err != nil {
		t.Fatal(err)
	}
	if !defaults.HashAlgorithm.Equal(OIDDigestAlgorithmSHA1) ||
		!defaults.MGFHashAlgorithm.Equal(OIDDigestAlgorithmSHA1) ||
		defaults.SaltLength != 20 ||
		defaults.TrailerField != 1 {
		t.Fatalf("unexpected RSASSA-PSS defaults: %+v", defaults)
	}
}

func TestParseRSAPSSParametersPreservesASN1Cause(t *testing.T) {
	_, err := parseRSAPSSParameters(asn1.RawValue{
		FullBytes: []byte{0x30, 0x03, 0x02, 0x01},
	})
	if !errors.Is(err, ErrInvalidPSSParameters) {
		t.Fatalf("got %v, want ErrInvalidPSSParameters", err)
	}
	var syntaxErr asn1.SyntaxError
	var structuralErr asn1.StructuralError
	if !errors.As(err, &syntaxErr) && !errors.As(err, &structuralErr) {
		t.Fatalf("got %v, want preserved ASN.1 cause", err)
	}
}

func TestRSAPSSAlgorithmConsistency(t *testing.T) {
	tests := []struct {
		name       string
		parameters asn1.RawValue
		digest     asn1.ObjectIdentifier
		want       error
	}{
		{
			"HashMismatch",
			testRSAPSSParameters(t, OIDDigestAlgorithmSHA256, OIDDigestAlgorithmSHA256, 32, 1),
			OIDDigestAlgorithmSHA384,
			ErrAlgorithmMismatch,
		},
		{
			"MGFHashMismatch",
			testRSAPSSParameters(t, OIDDigestAlgorithmSHA256, OIDDigestAlgorithmSHA384, 32, 1),
			OIDDigestAlgorithmSHA256,
			ErrAlgorithmMismatch,
		},
		{
			"NegativeSaltLength",
			testRSAPSSParameters(t, OIDDigestAlgorithmSHA256, OIDDigestAlgorithmSHA256, -1, 1),
			OIDDigestAlgorithmSHA256,
			ErrInvalidPSSParameters,
		},
		{
			"TrailerField",
			testRSAPSSParameters(t, OIDDigestAlgorithmSHA256, OIDDigestAlgorithmSHA256, 32, 2),
			OIDDigestAlgorithmSHA256,
			ErrInvalidPSSParameters,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getSignatureAlgorithm(
				pkix.AlgorithmIdentifier{
					Algorithm:  OIDEncryptionAlgorithmRSAPSS,
					Parameters: tt.parameters,
				},
				pkix.AlgorithmIdentifier{Algorithm: tt.digest},
			)
			if !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCheckSignatureUsesRSAPSSSaltLength(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	content := []byte("CMS content")
	digest := sha256.Sum256(content)
	signature, err := rsa.SignPSS(
		rand.Reader,
		key,
		crypto.SHA256,
		digest[:],
		&rsa.PSSOptions{SaltLength: 31, Hash: crypto.SHA256},
	)
	if err != nil {
		t.Fatal(err)
	}
	signer := signerWithAlgorithms(OIDEncryptionAlgorithmRSAPSS, OIDDigestAlgorithmSHA256)
	signer.DigestEncryptionAlgorithm.Parameters = testRSAPSSParameters(
		t,
		OIDDigestAlgorithmSHA256,
		OIDDigestAlgorithmSHA256,
		31,
		1,
	)
	signer.EncryptedDigest = signature

	if err := CheckSignature(&x509.Certificate{PublicKey: &key.PublicKey}, signer, content); err != nil {
		t.Fatalf("verify RSASSA-PSS with encoded salt length: %v", err)
	}
}

func TestRSAPSSAlgorithmConsistencyAcceptsSupportedParameters(t *testing.T) {
	algorithm, err := getSignatureAlgorithm(
		pkix.AlgorithmIdentifier{
			Algorithm: OIDEncryptionAlgorithmRSAPSS,
			Parameters: testRSAPSSParameters(
				t,
				OIDDigestAlgorithmSHA256,
				OIDDigestAlgorithmSHA256,
				32,
				1,
			),
		},
		pkix.AlgorithmIdentifier{Algorithm: OIDDigestAlgorithmSHA256},
	)
	if err != nil {
		t.Fatal(err)
	}
	if algorithm != x509.SHA256WithRSAPSS {
		t.Fatalf("got %s, want SHA256WithRSAPSS", algorithm)
	}
}

func testRSAPSSParameters(
	t *testing.T,
	hash, mgfHash asn1.ObjectIdentifier,
	saltLength, trailerField int,
) asn1.RawValue {
	t.Helper()
	mgfHashDER, err := asn1.Marshal(pkix.AlgorithmIdentifier{
		Algorithm:  mgfHash,
		Parameters: asn1.RawValue{FullBytes: asn1.NullBytes},
	})
	if err != nil {
		t.Fatal(err)
	}
	der, err := asn1.Marshal(rsaPSSParametersASN1{
		HashAlgorithm: pkix.AlgorithmIdentifier{
			Algorithm:  hash,
			Parameters: asn1.RawValue{FullBytes: asn1.NullBytes},
		},
		MaskGenAlgorithm: pkix.AlgorithmIdentifier{
			Algorithm:  oidMaskGenAlgorithmMGF1,
			Parameters: asn1.RawValue{FullBytes: mgfHashDER},
		},
		SaltLength:   saltLength,
		TrailerField: trailerField,
	})
	if err != nil {
		t.Fatal(err)
	}
	return asn1.RawValue{FullBytes: der}
}

func signerWithAlgorithms(encryption, digest asn1.ObjectIdentifier) SignerInfo {
	return SignerInfo{
		DigestEncryptionAlgorithm: pkix.AlgorithmIdentifier{Algorithm: encryption},
		DigestAlgorithm:           pkix.AlgorithmIdentifier{Algorithm: digest},
	}
}

func rsaTestModulus() *big.Int {
	n := new(big.Int).Lsh(big.NewInt(1), 2047)
	return n.Add(n, big.NewInt(1))
}
