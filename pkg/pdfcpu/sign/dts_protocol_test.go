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

package sign

import (
	"crypto"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/pkcs7"
)

// TestTimestampingEKUProfiles verifies RFC 3161 accepts only a critical,
// exclusive timestamping EKU and reports unsupported profiles as evidence.
func TestTimestampingEKUProfiles(t *testing.T) {
	tests := []struct {
		name string
		cert *x509.Certificate
		want string
		ok   bool
	}{
		{
			name: "Missing",
			cert: &x509.Certificate{},
			want: "timestamping EKU missing",
		},
		{
			name: "Mixed",
			cert: timestampingProfileCertificate(
				true,
				x509.ExtKeyUsageTimeStamping,
				x509.ExtKeyUsageCodeSigning,
			),
			want: "timestamping EKU must be exclusive",
		},
		{
			name: "Noncritical",
			cert: timestampingProfileCertificate(false, x509.ExtKeyUsageTimeStamping),
			want: "timestamping EKU extension must be critical",
		},
		{
			name: "Valid",
			cert: timestampingProfileCertificate(true, x509.ExtKeyUsageTimeStamping),
			ok:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer := &model.Signer{}
			result := unknownSignatureResult()

			if got := checkTimestampingEKU(tt.cert, signer, result); got != tt.ok {
				t.Fatalf("got accepted=%t, want %t", got, tt.ok)
			}
			if tt.ok {
				if len(signer.Problems) != 0 || result.Reason != model.SignatureReasonUnknown {
					t.Fatalf("valid timestamping profile produced evidence: signer=%+v result=%+v", signer, result)
				}
				return
			}
			if result.Status != model.SignatureStatusUnknown ||
				result.Reason != model.SignatureReasonUnsupported {
				t.Fatalf("unsupported profile was not Unknown evidence: %+v", result)
			}
			if len(signer.Problems) != 1 || !strings.Contains(signer.Problems[0], tt.want) {
				t.Fatalf("got Problems %v, want %q", signer.Problems, tt.want)
			}
			if signer.Certificate != nil {
				t.Fatalf("unsupported profile was assessed as trusted: %+v", signer.Certificate)
			}
		})
	}
}

// TestTSTInfoProfileRequirements verifies version and genTime are validated
// before document timestamp evidence can be authenticated.
func TestTSTInfoProfileRequirements(t *testing.T) {
	tests := []struct {
		name   string
		info   *TSTInfo
		reason model.SignatureReason
		want   string
		ok     bool
	}{
		{
			name:   "Missing",
			reason: model.SignatureReasonTimestampTokenInvalid,
			want:   "timestamp info missing",
		},
		{
			name:   "UnsupportedVersion",
			info:   &TSTInfo{Version: 2, GenTime: time.Now()},
			reason: model.SignatureReasonUnsupported,
			want:   "unsupported timestamp info version 2",
		},
		{
			name:   "MissingGenTime",
			info:   &TSTInfo{Version: 1},
			reason: model.SignatureReasonTimestampTokenInvalid,
			want:   "timestamp info genTime missing",
		},
		{
			name: "Valid",
			info: &TSTInfo{Version: 1, GenTime: time.Now()},
			ok:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer := &model.Signer{}
			result := unknownSignatureResult()

			if got := checkTSTInfoProfile(tt.info, signer, result); got != tt.ok {
				t.Fatalf("got accepted=%t, want %t", got, tt.ok)
			}
			if tt.ok {
				return
			}
			if result.Status != model.SignatureStatusUnknown || result.Reason != tt.reason {
				t.Fatalf("unexpected profile result: %+v", result)
			}
			if len(signer.Problems) != 1 || !strings.Contains(signer.Problems[0], tt.want) {
				t.Fatalf("got Problems %v, want %q", signer.Problems, tt.want)
			}
		})
	}
}

// TestDTSSignedAttributesRequired verifies DTS signature verification cannot
// fall back to a CMS profile without signed attributes.
func TestDTSSignedAttributesRequired(t *testing.T) {
	signer := &model.Signer{}
	result := unknownSignatureResult()

	if checkDTSSignedAttributes(pkcs7.SignerInfo{}, signer, result) {
		t.Fatal("CMS signer without signed attributes was accepted")
	}
	if result.Status != model.SignatureStatusUnknown ||
		result.Reason != model.SignatureReasonUnsupported ||
		len(signer.Problems) != 1 ||
		!strings.Contains(signer.Problems[0], "signed attributes missing") {
		t.Fatalf("missing signed-attribute evidence: signer=%+v result=%+v", signer, result)
	}
}

// TestESSCertificateBinding verifies the ESS attribute binds exactly one
// supported attribute to the actual signer certificate.
func TestESSCertificateBinding(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)

	tests := []struct {
		name string
		attr pkcs7.Attribute
	}{
		{
			name: "SigningCertificate",
			attr: timestampSigningCertificateAttributeV1(t, fixture.cert, true),
		},
		{
			name: "SigningCertificateV2DefaultSHA256",
			attr: timestampSigningCertificateAttributeV2(
				t,
				fixture.cert,
				0,
				nil,
				true,
			),
		},
		{
			name: "SigningCertificateV2ExplicitSHA256",
			attr: timestampSigningCertificateAttributeV2(
				t,
				fixture.cert,
				crypto.SHA256,
				pkcs7.OIDDigestAlgorithmSHA256,
				false,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p7Signer := appendTimestampAttribute(t, fixture.signer, tt.attr)
			if err := validateESSCertificateBinding(p7Signer, fixture.cert); err != nil {
				t.Fatalf("valid ESS certificate binding rejected: %v", err)
			}
		})
	}
}

// TestESSCertificateBindingCardinality verifies a signer has exactly one
// supported ESS signing-certificate attribute.
func TestESSCertificateBindingCardinality(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)
	v1 := timestampSigningCertificateAttributeV1(t, fixture.cert, false)
	v2 := timestampSigningCertificateAttributeV2(
		t,
		fixture.cert,
		0,
		nil,
		false,
	)

	tests := []struct {
		name   string
		signer pkcs7.SignerInfo
	}{
		{name: "Missing", signer: fixture.signer},
		{
			name: "Duplicate",
			signer: appendTimestampAttribute(
				t,
				appendTimestampAttribute(t, fixture.signer, v1),
				v1,
			),
		},
		{
			name: "BothVersions",
			signer: appendTimestampAttribute(
				t,
				appendTimestampAttribute(t, fixture.signer, v1),
				v2,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateESSCertificateBinding(tt.signer, fixture.cert)
			if !errors.Is(err, pkcs7.ErrMalformedAttribute) {
				t.Fatalf("got %v, want ErrMalformedAttribute", err)
			}
		})
	}
}

// TestESSCertificateBindingFailureClassification verifies malformed and
// unsupported ESS data remain reportable, while a certificate mismatch makes
// the timestamp evidence invalid.
func TestESSCertificateBindingFailureClassification(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)
	malformed := pkcs7.Attribute{
		Type:  oidSigningCertificateV2,
		Value: asn1.RawValue{FullBytes: []byte{0x30}},
	}
	unsupported := timestampSigningCertificateAttributeV2(
		t,
		fixture.cert,
		0,
		asn1.ObjectIdentifier{1, 2, 3, 4},
		false,
	)
	mismatch := timestampSigningCertificateAttributeV2(
		t,
		fixture.cert,
		crypto.SHA256,
		pkcs7.OIDDigestAlgorithmSHA256,
		false,
	)
	mismatch.Value = signingCertificateV2{
		Certs: []essCertIDv2{{
			HashAlgorithm: algorithmIdentifierRaw(t, pkcs7.OIDDigestAlgorithmSHA256),
			CertHash:      make([]byte, sha256.Size),
		}},
	}

	tests := []struct {
		name       string
		attr       pkcs7.Attribute
		class      error
		wantStatus model.SignatureStatus
		wantReason model.SignatureReason
	}{
		{
			name:       "Malformed",
			attr:       malformed,
			class:      pkcs7.ErrMalformedAttribute,
			wantStatus: model.SignatureStatusUnknown,
			wantReason: model.SignatureReasonMalformed,
		},
		{
			name:       "UnsupportedHash",
			attr:       unsupported,
			class:      pkcs7.ErrUnsupportedAlgorithm,
			wantStatus: model.SignatureStatusUnknown,
			wantReason: model.SignatureReasonUnsupported,
		},
		{
			name:       "CertificateHashMismatch",
			attr:       mismatch,
			class:      errESSCertificateMismatch,
			wantStatus: model.SignatureStatusInvalid,
			wantReason: model.SignatureReasonTimestampTokenInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p7Signer := appendTimestampAttribute(t, fixture.signer, tt.attr)
			err := validateESSCertificateBinding(p7Signer, fixture.cert)
			if !errors.Is(err, tt.class) {
				t.Fatalf("got %v, want classification %v", err, tt.class)
			}

			signer := &model.Signer{}
			result := unknownSignatureResult()
			if checkTimestampSigningCertificate(p7Signer, fixture.cert, signer, result) {
				t.Fatal("invalid ESS certificate binding was accepted")
			}
			if result.Status != tt.wantStatus || result.Reason != tt.wantReason {
				t.Fatalf("unexpected evidence classification: %+v", result)
			}
			if len(signer.Problems) != 1 {
				t.Fatalf("got Problems %v, want one reportable problem", signer.Problems)
			}
		})
	}
}

// TestESSCertificateBindingIssuerSerial verifies an optional ESS
// issuer-and-serial identifier must identify the actual TSA certificate.
func TestESSCertificateBindingIssuerSerial(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)
	valid := timestampSigningCertificateAttributeV1(t, fixture.cert, true)

	wrongSerialCert := *fixture.cert
	wrongSerialCert.SerialNumber = new(big.Int).Add(fixture.cert.SerialNumber, big.NewInt(1))
	wrongSerial := timestampSigningCertificateAttributeV1(t, &wrongSerialCert, true)
	certDigest := sha1.Sum(fixture.cert.Raw)
	wrongSerial.Value = replaceTimestampCertificateHash(
		t,
		wrongSerial.Value,
		certDigest[:],
	)

	wrongIssuerCert := *fixture.cert
	wrongIssuerCert.RawIssuer = []byte{0x30, 0x00}
	wrongIssuer := timestampSigningCertificateAttributeV1(t, &wrongIssuerCert, true)
	wrongIssuer.Value = replaceTimestampCertificateHash(
		t,
		wrongIssuer.Value,
		certDigest[:],
	)

	tests := []struct {
		name string
		attr pkcs7.Attribute
		ok   bool
	}{
		{name: "Match", attr: valid, ok: true},
		{name: "SerialMismatch", attr: wrongSerial},
		{name: "IssuerMismatch", attr: wrongIssuer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateESSCertificateBinding(
				appendTimestampAttribute(t, fixture.signer, tt.attr),
				fixture.cert,
			)
			if tt.ok && err != nil {
				t.Fatalf("matching issuer and serial rejected: %v", err)
			}
			if !tt.ok && !errors.Is(err, errESSCertificateMismatch) {
				t.Fatalf("got %v, want certificate mismatch", err)
			}
		})
	}
}

// TestESSCertificateKeyUsage verifies an existing KeyUsage extension
// must permit signature creation.
func TestESSCertificateKeyUsage(t *testing.T) {
	fixture := newDetachedP7SignerFixture(t, nil, nil)
	attr := timestampSigningCertificateAttributeV2(
		t,
		fixture.cert,
		0,
		nil,
		false,
	)

	absent := *fixture.cert
	absent.Extensions = removeCertificateExtension(absent.Extensions, oidExtensionKeyUsage)
	absent.KeyUsage = 0

	contentCommitment := *fixture.cert
	contentCommitment.KeyUsage = x509.KeyUsageContentCommitment

	certSignOnly := *fixture.cert
	certSignOnly.KeyUsage = x509.KeyUsageCertSign

	tests := []struct {
		name string
		cert *x509.Certificate
		ok   bool
	}{
		{name: "ExtensionAbsent", cert: &absent, ok: true},
		{name: "DigitalSignature", cert: fixture.cert, ok: true},
		{name: "ContentCommitment", cert: &contentCommitment, ok: true},
		{name: "CertSignOnly", cert: &certSignOnly},
	}

	p7Signer := appendTimestampAttribute(t, fixture.signer, attr)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateESSCertificateBinding(p7Signer, tt.cert)
			if tt.ok && err != nil {
				t.Fatalf("permitted KeyUsage rejected: %v", err)
			}
			if !tt.ok && !errors.Is(err, errUnsupportedESSCertificateProfile) {
				t.Fatalf("got %v, want unsupported ESS certificate profile", err)
			}
		})
	}
}

// TestDTSCertificatePathDoesNotUseClaimedGenTime verifies a TSA certificate
// expired now is not made locally valid by the TSTInfo genTime claim.
func TestDTSCertificatePathDoesNotUseClaimedGenTime(t *testing.T) {
	genTime := time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second)
	key := testRSAKey(t)
	template := testCertTemplate("DTS TSA", true)
	template.NotBefore = genTime.Add(-time.Hour)
	template.NotAfter = genTime.Add(time.Hour)
	template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageTimeStamping}
	cert := testCertificate(t, template, template, &key.PublicKey, key)
	for i := range cert.Extensions {
		if cert.Extensions[i].Id.Equal(oidExtensionExtendedKeyUsage) {
			cert.Extensions[i].Critical = true
		}
	}
	if err := validateTimestampingEKU(cert); err != nil {
		t.Fatalf("invalid test TSA profile: %v", err)
	}

	roots := x509.NewCertPool()
	roots.AddCert(cert)
	signer := &model.Signer{}
	result := unknownSignatureResult()
	ctx := &model.Context{
		Configuration: model.NewDefaultConfiguration(),
		XRefTable:     &model.XRefTable{},
	}
	pathValidated, err := validateDTSCert(
		cert,
		[]*x509.Certificate{cert},
		roots,
		nil,
		nil,
		signer,
		result,
		ctx,
	)
	if err != nil {
		t.Fatal(err)
	}
	if pathValidated {
		t.Fatalf("claimed genTime validated an expired TSA path: signer=%+v result=%+v", signer, result)
	}
	if signer.Certificate == nil || !signer.Certificate.ValidationTime.Time.IsZero() {
		t.Fatalf("genTime selected certificate assessment time: %+v", signer.Certificate)
	}
	evidence := timestampEvidence{
		Kind:                  timestampKindDocument,
		SigningTime:           genTime,
		Present:               true,
		DigestVerified:        true,
		SignatureVerified:     true,
		CorrectProfile:        true,
		LocalTSAPathValidated: pathValidated,
	}
	finalizeDTSValidationResult(result, ctx, evidence, completedLocalSignatureAssessment())
	if result.Status != model.SignatureStatusUnknown ||
		result.Reason != model.SignatureReasonCertExpired ||
		signer.Certificate == nil ||
		!signer.Certificate.Expired ||
		!signer.Certificate.ValidationTime.Time.IsZero() ||
		!ctx.DTS.IsZero() {
		t.Fatalf("claimed genTime escaped evidence boundary: signer=%+v result=%+v", signer, result)
	}
}

func timestampingProfileCertificate(
	critical bool,
	usages ...x509.ExtKeyUsage,
) *x509.Certificate {
	return &x509.Certificate{
		ExtKeyUsage: usages,
		Extensions: []pkix.Extension{{
			Id:       oidExtensionExtendedKeyUsage,
			Critical: critical,
		}},
	}
}

func appendTimestampAttribute(
	t *testing.T,
	signer pkcs7.SignerInfo,
	attribute pkcs7.Attribute,
) pkcs7.SignerInfo {
	t.Helper()
	attr := signer.AuthenticatedAttributes[0]
	attr.Type = attribute.Type
	value, err := asn1.Marshal(attribute.Value)
	if err != nil {
		t.Fatal(err)
	}
	attr.Value = asn1.RawValue{
		Class:      asn1.ClassUniversal,
		Tag:        asn1.TagSet,
		IsCompound: true,
		Bytes:      value,
	}
	signer.AuthenticatedAttributes = append(signer.AuthenticatedAttributes, attr)
	return signer
}

func timestampSigningCertificateAttributeV1(
	t *testing.T,
	cert *x509.Certificate,
	withIssuerSerial bool,
) pkcs7.Attribute {
	t.Helper()
	digest := sha1.Sum(cert.Raw)
	certID := essCertID{CertHash: digest[:]}
	if withIssuerSerial {
		certID.IssuerSerial = timestampIssuerSerialRaw(t, cert)
	}
	return pkcs7.Attribute{
		Type: oidSigningCertificate,
		Value: signingCertificate{
			Certs: []essCertID{certID},
		},
	}
}

func timestampSigningCertificateAttributeV2(
	t *testing.T,
	cert *x509.Certificate,
	hash crypto.Hash,
	hashOID asn1.ObjectIdentifier,
	withIssuerSerial bool,
) pkcs7.Attribute {
	t.Helper()
	if hash == 0 {
		hash = crypto.SHA256
	}
	digest := hash.New()
	if _, err := digest.Write(cert.Raw); err != nil {
		t.Fatal(err)
	}
	certID := essCertIDv2{CertHash: digest.Sum(nil)}
	if hashOID != nil {
		certID.HashAlgorithm = algorithmIdentifierRaw(t, hashOID)
	}
	if withIssuerSerial {
		certID.IssuerSerial = timestampIssuerSerialRaw(t, cert)
	}
	return pkcs7.Attribute{
		Type: oidSigningCertificateV2,
		Value: signingCertificateV2{
			Certs: []essCertIDv2{certID},
		},
	}
}

func timestampIssuerSerialRaw(
	t *testing.T,
	cert *x509.Certificate,
) asn1.RawValue {
	t.Helper()
	return asn1ValueRaw(t, essIssuerSerial{
		Issuer: []asn1.RawValue{{
			Class:      asn1.ClassContextSpecific,
			Tag:        4,
			IsCompound: true,
			Bytes:      cert.RawIssuer,
		}},
		SerialNumber: new(big.Int).Set(cert.SerialNumber),
	})
}

func algorithmIdentifierRaw(
	t *testing.T,
	oid asn1.ObjectIdentifier,
) asn1.RawValue {
	t.Helper()
	return asn1ValueRaw(t, pkix.AlgorithmIdentifier{Algorithm: oid})
}

func asn1ValueRaw(t *testing.T, value any) asn1.RawValue {
	t.Helper()
	der, err := asn1.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var raw asn1.RawValue
	rest, err := asn1.Unmarshal(der, &raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(rest) > 0 {
		t.Fatalf("test ASN.1 value has trailing data: %x", rest)
	}
	return raw
}

func replaceTimestampCertificateHash(
	t *testing.T,
	value any,
	hash []byte,
) any {
	t.Helper()
	attribute, ok := value.(signingCertificate)
	if !ok {
		t.Fatalf("unexpected signing-certificate value %T", value)
	}
	attribute.Certs[0].CertHash = append([]byte(nil), hash...)
	return attribute
}

func removeCertificateExtension(
	extensions []pkix.Extension,
	oid asn1.ObjectIdentifier,
) []pkix.Extension {
	filtered := make([]pkix.Extension, 0, len(extensions))
	for _, extension := range extensions {
		if !extension.Id.Equal(oid) {
			filtered = append(filtered, extension)
		}
	}
	return filtered
}
