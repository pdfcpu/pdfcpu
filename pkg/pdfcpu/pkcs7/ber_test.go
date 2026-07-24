/*
Copyright (c) 2015 Andrew Smith
Copyright 2026 The pdfcpu Authors.

Licensed under the MIT License. See LICENSE in this directory.
*/

package pkcs7

import (
	"bytes"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestBer2Der(t *testing.T) {
	t.Parallel()
	// indefinite length fixture
	ber := []byte{0x30, 0x80, 0x02, 0x01, 0x01, 0x00, 0x00}
	expected := []byte{0x30, 0x03, 0x02, 0x01, 0x01}
	der, err := ber2der(ber)
	if err != nil {
		t.Fatalf("ber2der failed with error: %v", err)
	}
	if !bytes.Equal(der, expected) {
		t.Errorf("ber2der result did not match.\n\tExpected: % X\n\tActual: % X", expected, der)
	}

	if der2, err := ber2der(der); err != nil {
		t.Errorf("ber2der on DER bytes failed with error: %v", err)
	} else {
		if !bytes.Equal(der, der2) {
			t.Error("ber2der is not idempotent")
		}
	}
	var thing struct {
		Number int
	}
	rest, err := asn1.Unmarshal(der, &thing)
	if err != nil {
		t.Errorf("Cannot parse resulting DER because: %v", err)
	} else if len(rest) > 0 {
		t.Errorf("Resulting DER has trailing data: % X", rest)
	}
}

func TestBer2Der_Negatives(t *testing.T) {
	t.Parallel()
	fixtures := []struct {
		Input         []byte
		ErrorContains string
	}{
		{[]byte{0x30, 0x85}, "length uses more than four octets"},
		{[]byte{0x30, 0x84, 0x80, 0x0, 0x0, 0x0}, "length exceeds supported range"},
		{[]byte{0x30, 0x82, 0x0, 0x1}, "length has leading zero"},
		{[]byte{0x30, 0x80, 0x1, 0x2, 0x1, 0x2}, "missing indefinite-length terminator"},
		{[]byte{0x30, 0x80, 0x1, 0x2}, "content length exceeds available data"},
		{[]byte{0x30, 0x03, 0x01, 0x02}, "content length exceeds available data"},
		{[]byte{0x30}, "length offset outside BER data"},
	}

	for _, fixture := range fixtures {
		_, err := ber2der(fixture.Input)
		if err == nil {
			t.Errorf("No error thrown. Expected: %s", fixture.ErrorContains)
		}
		if !strings.Contains(err.Error(), fixture.ErrorContains) {
			t.Errorf("Unexpected error thrown.\n\tExpected: /%s/\n\tActual: %s", fixture.ErrorContains, err.Error())
		}
	}
}

type failingASN1Object struct {
	err error
}

func (o failingASN1Object) EncodeTo(*bytes.Buffer) error {
	return o.err
}

func TestASN1StructuredEncodePreservesChildCause(t *testing.T) {
	t.Parallel()
	cause := errors.New("encode failure")
	obj := asn1Structured{
		tagBytes: []byte{0x30},
		content:  []asn1Object{failingASN1Object{err: cause}},
	}

	err := obj.EncodeTo(new(bytes.Buffer))
	if !errors.Is(err, cause) {
		t.Fatalf("expected encoding cause, got: %v", err)
	}
	if !strings.Contains(err.Error(), "encode child 1") {
		t.Fatalf("expected child context, got: %v", err)
	}
}

func TestASN1StructuredEncodeRejectsNilChild(t *testing.T) {
	t.Parallel()
	obj := asn1Structured{
		tagBytes: []byte{0x30},
		content:  []asn1Object{nil},
	}

	requireErrorWithoutPanic(t, "missing ASN.1 object", func() error {
		return obj.EncodeTo(new(bytes.Buffer))
	})
}

func TestEncodeLengthRejectsNegativeLength(t *testing.T) {
	t.Parallel()
	err := encodeLength(new(bytes.Buffer), -1)
	if err == nil || !strings.Contains(err.Error(), "negative length") {
		t.Fatalf("expected negative-length error, got: %v", err)
	}
}

func TestBer2DerRejectsMalformedBoundariesWithoutPanic(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ber  []byte
		want string
	}{
		{"unterminated high tag", []byte{0x1f}, "unterminated high-tag-number"},
		{"high tag leading zero", []byte{0x1f, 0x80, 0x00}, "high-tag-number has leading zero"},
		{"truncated long length", []byte{0x30, 0x82, 0x01}, "length octets exceed available data"},
		{"missing terminator", []byte{0x30, 0x80, 0x02, 0x01, 0x01}, "missing indefinite-length terminator"},
		{"child crosses boundary", []byte{0x30, 0x02, 0x02, 0x01, 0x01}, "exceeds parent boundary"},
		{"unexpected terminator", []byte{0x30, 0x02, 0x00, 0x00}, "unexpected end-of-content marker"},
		{"trailing object", []byte{0x02, 0x01, 0x01, 0x05, 0x00}, "trailing data at offset 3"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			requireErrorWithoutPanic(t, tt.want, func() error {
				_, err := ber2der(tt.ber)
				return err
			})
		})
	}
}

func TestBer2DerAcceptsPDFSignatureZeroPadding(t *testing.T) {
	t.Parallel()
	der, err := ber2der([]byte{0x02, 0x01, 0x01, 0x00, 0x00})
	if err != nil {
		t.Fatalf("expected zero padding to be accepted: %v", err)
	}
	if want := []byte{0x02, 0x01, 0x01}; !bytes.Equal(der, want) {
		t.Fatalf("expected %x, got %x", want, der)
	}
}

func TestBERReadersRejectInvalidOffsetsWithoutPanic(t *testing.T) {
	t.Parallel()
	ber := []byte{0x02, 0x01, 0x01}
	tests := []struct {
		name string
		call func() error
		want string
	}{
		{
			"tag negative offset",
			func() error {
				_, _, _, _, err := readTag(ber, -1)
				return err
			},
			"offset outside BER data",
		},
		{
			"length negative offset",
			func() error {
				_, _, _, err := readLength(ber, -1)
				return err
			},
			"length offset outside BER data",
		},
		{
			"object past end",
			func() error {
				_, _, err := readObject(ber, len(ber))
				return err
			},
			"read tag at offset 3",
		},
		{
			"invalid structured boundary",
			func() error {
				_, _, err := readStructuredContent(ber, 2, 1, false)
				return err
			},
			"constructed-content boundary outside BER data",
		},
		{
			"termination past end",
			func() error {
				_, err := isIndefiniteTermination(ber, len(ber))
				return err
			},
			"missing indefinite-length terminator",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			requireErrorWithoutPanic(t, tt.want, tt.call)
		})
	}
}

func requireErrorWithoutPanic(t *testing.T, want string, call func() error) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	err := call()
	if err == nil {
		t.Fatalf("expected error containing %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got: %v", want, err)
	}
}

func TestBer2Der_NestedMultipleIndefinite(t *testing.T) {
	t.Parallel()
	// indefinite length fixture
	ber := []byte{0x30, 0x80, 0x30, 0x80, 0x02, 0x01, 0x01, 0x00, 0x00, 0x30, 0x80, 0x02, 0x01, 0x02, 0x00, 0x00, 0x00, 0x00}
	expected := []byte{0x30, 0x0A, 0x30, 0x03, 0x02, 0x01, 0x01, 0x30, 0x03, 0x02, 0x01, 0x02}

	der, err := ber2der(ber)
	if err != nil {
		t.Fatalf("ber2der failed with error: %v", err)
	}
	if bytes.Compare(der, expected) != 0 {
		t.Errorf("ber2der result did not match.\n\tExpected: % X\n\tActual: % X", expected, der)
	}

	if der2, err := ber2der(der); err != nil {
		t.Errorf("ber2der on DER bytes failed with error: %v", err)
	} else {
		if !bytes.Equal(der, der2) {
			t.Error("ber2der is not idempotent")
		}
	}
	var thing struct {
		Nest1 struct {
			Number int
		}
		Nest2 struct {
			Number int
		}
	}
	rest, err := asn1.Unmarshal(der, &thing)
	if err != nil {
		t.Errorf("Cannot parse resulting DER because: %v", err)
	} else if len(rest) > 0 {
		t.Errorf("Resulting DER has trailing data: % X", rest)
	}
}

func TestVerifyIndefiniteLengthBer(t *testing.T) {
	t.Parallel()
	decoded := mustDecodePEM([]byte(testPKCS7))

	_, err := ber2der(decoded)
	if err != nil {
		t.Errorf("cannot parse indefinite length ber: %v", err)
	}
}

func mustDecodePEM(data []byte) []byte {
	var block *pem.Block
	block, rest := pem.Decode(data)
	if len(rest) != 0 {
		panic(fmt.Errorf("unexpected remaining PEM block during decode"))
	}
	return block.Bytes
}

const testPKCS7 = `
-----BEGIN PKCS7-----
MIAGCSqGSIb3DQEHAqCAMIACAQExDzANBglghkgBZQMEAgEFADCABgkqhkiG9w0B
BwGggCSABIIDfXsiQWdlbnRBY3Rpb25PdmVycmlkZXMiOnsiQWdlbnRPdmVycmlk
ZXMiOnsiRmlsZUV4aXN0c0JlaGF2aW9yIjoiT1ZFUldSSVRFIn19LCJBcHBsaWNh
dGlvbklkIjoiZTA0NDIzZTQtN2E2Ny00ZjljLWIyOTEtOTllNjNjMWMyMTU4Iiwi
QXBwbGljYXRpb25OYW1lIjoibWthbmlhLXhyZF9zYW0uY2R3c19lY2hvc2VydmVy
IiwiRGVwbG95bWVudENyZWF0b3IiOiJ1c2VyIiwiRGVwbG95bWVudEdyb3VwSWQi
OiJmYWI5MjEwZi1mNmM3LTQyODUtYWEyZC03Mzc2MGQ4ODE3NmEiLCJEZXBsb3lt
ZW50R3JvdXBOYW1lIjoibWthbmlhLXhyZF9zYW0uY2R3c19lY2hvc2VydmVyX2Rn
IiwiRGVwbG95bWVudElkIjoiZC1UREUxVTNXREEiLCJEZXBsb3ltZW50VHlwZSI6
IklOX1BMQUNFIiwiR2l0SHViQWNjZXNzVG9rZW4iOm51bGwsIkluc3RhbmNlR3Jv
dXBJZCI6ImZhYjkyMTBmLWY2YzctNDI4NS1hYTJkLTczNzYwZDg4MTc2YSIsIlJl
dmlzaW9uIjp7IkFwcFNwZWNDb250ZW50IjpudWxsLCJDb2RlQ29tbWl0UmV2aXNp
b24iOm51bGwsIkdpdEh1YlJldmlzaW9uIjpudWxsLCJHaXRSZXZpc2lvbiI6bnVs
bCwiUmV2aXNpb25UeXBlIjoiUzMiLCJTM1JldmlzaW9uIjp7IkJ1Y2tldCI6Im1r
YW5pYS1jZHdzLWRlcGxveS1idWNrZXQiLCJCdW5kbGVUeXBlIjoiemlwIiwiRVRh
ZyI6bnVsbCwiS2V5IjoieHJkOjpzYW0uY2R3czo6ZWNob3NlcnZlcjo6MTo6Lnpp
cCIsIlZlcnNpb24iOm51bGx9fSwiUzNSZXZpc2lvbiI6eyJCdWNrZXQiOiJta2Fu
aWEtY2R3cy1kZXBsb3ktYnVja2V0IiwiQnVuZGxlVHlwZSI6InppcCIsIkVUYWci
Om51bGwsIktleSI6InhyZDo6c2FtLmNkd3M6OmVjaG9zZXJ2ZXI6OjE6Oi56aXAi
LCJWZXJzaW9uIjpudWxsfSwiVGFyZ2V0UmV2aXNpb24iOm51bGx9AAAAAAAAoIAw
ggWbMIIEg6ADAgECAhAGrjFMK45t2jcNHtjY1DjEMA0GCSqGSIb3DQEBCwUAMEYx
CzAJBgNVBAYTAlVTMQ8wDQYDVQQKEwZBbWF6b24xFTATBgNVBAsTDFNlcnZlciBD
QSAxQjEPMA0GA1UEAxMGQW1hem9uMB4XDTIwMTExMjAwMDAwMFoXDTIxMTAxNTIz
NTk1OVowNDEyMDAGA1UEAxMpY29kZWRlcGxveS1zaWduZXItdXMtZWFzdC0yLmFt
YXpvbmF3cy5jb20wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDit4f+
I4BSv4rBV/8bJ+f4KqBwTCt9iJeau/r9liQfMgj/C1M2E+aa++u8BtY/LQstB44v
v6KqcaiOyWpkD9OsUty9qb4eNTPF2Y4jpNsi/Hfw0phsd9gLun2foppILmL4lZIG
lBhTeEwv6qV4KbyXOG9abHOX32+jVFtM1rbzHNFvz90ysfZp16TBAi7IRKEZeXvd
MvlJJMAJtAoblxiDIS3A1csY1G4XHYET8xIoCop3mqEZEtAxUUP2epdXXdhD2U0G
7alSRS54o91QW1Dp3A13lu1A1nds9CkWlPkDTpKSUG/qN5y5+6dCCGaydgL5krMs
R79bCrR1sEKm5hi1AgMBAAGjggKVMIICkTAfBgNVHSMEGDAWgBRZpGYGUqB7lZI8
o5QHJ5Z0W/k90DAdBgNVHQ4EFgQUPF5qTbnTDYhmp7tGmmL/jTmLoHMwNAYDVR0R
BC0wK4IpY29kZWRlcGxveS1zaWduZXItdXMtZWFzdC0yLmFtYXpvbmF3cy5jb20w
DgYDVR0PAQH/BAQDAgWgMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjA7
BgNVHR8ENDAyMDCgLqAshipodHRwOi8vY3JsLnNjYTFiLmFtYXpvbnRydXN0LmNv
bS9zY2ExYi5jcmwwIAYDVR0gBBkwFzALBglghkgBhv1sAQIwCAYGZ4EMAQIBMHUG
CCsGAQUFBwEBBGkwZzAtBggrBgEFBQcwAYYhaHR0cDovL29jc3Auc2NhMWIuYW1h
em9udHJ1c3QuY29tMDYGCCsGAQUFBzAChipodHRwOi8vY3J0LnNjYTFiLmFtYXpv
bnRydXN0LmNvbS9zY2ExYi5jcnQwDAYDVR0TAQH/BAIwADCCAQQGCisGAQQB1nkC
BAIEgfUEgfIA8AB2APZclC/RdzAiFFQYCDCUVo7jTRMZM7/fDC8gC8xO8WTjAAAB
dboejIcAAAQDAEcwRQIgeqoKXbST17TCEzM1BMWx/jjyVQVBIN3LG17U4OaV364C
IQDPUSJZhJm7uqGea6+VwqeDe/vGuGSuJzkDwTIOeIXPaAB2AFzcQ5L+5qtFRLFe
mtRW5hA3+9X6R9yhc5SyXub2xw7KAAABdboejNQAAAQDAEcwRQIgEKIAwwhjUcq2
iwzBAagdy+fTiKnBY1Yjf6wOeRpwXfMCIQC8wM3nxiWrGgIpdzzgDvFhZZTV3N81
JWcYAu+srIVOhTANBgkqhkiG9w0BAQsFAAOCAQEAer9kml53XFy4ZSVzCbdsIFYP
Ohu7LDf5iffHBVZFnGOEVOmiPYYkNwi9R6EHIYaAs7G7GGLCp/6tdc+G4eF1j6wB
IkmXZcxMTxk/87R+S+36yDLg1GBZvqttLfexj0TRVAfVLJc7FjLXAW2+wi7YyNe8
X17lWBwHxa1r5KgweJshGzYVUsgMTSx0aJ+93ZnqplBp9x+9DSQNqqNlBgxFANxs
ux+dfpduyLd8VLqtlECGC07tYE4mBaAjMiNjCZRWMp8ya/Z6J/bJZ27IDGA4dXzm
l9NNnlbuUDAenAByUqE+0b78J6EmmdAVf+N8siriMg02FdP3lAXJLE8tDeZp8AAA
MYICIDCCAhwCAQEwWjBGMQswCQYDVQQGEwJVUzEPMA0GA1UEChMGQW1hem9uMRUw
EwYDVQQLEwxTZXJ2ZXIgQ0EgMUIxDzANBgNVBAMTBkFtYXpvbgIQBq4xTCuObdo3
DR7Y2NQ4xDANBglghkgBZQMEAgEFAKCBmDAYBgkqhkiG9w0BCQMxCwYJKoZIhvcN
AQcBMBwGCSqGSIb3DQEJBTEPFw0yMTA2MjQxOTU1MzFaMC0GCSqGSIb3DQEJNDEg
MB4wDQYJYIZIAWUDBAIBBQChDQYJKoZIhvcNAQELBQAwLwYJKoZIhvcNAQkEMSIE
IP7gMuT2H0/AhgPgj3Eo0NWCIdQOBjJO18coNKIaOnJYMA0GCSqGSIb3DQEBCwUA
BIIBAJX+e87q0YvRon9/ENTvE0FoYMzYblID2Reek6L217ZlZ6pUuRsc4ghhJ5Yh
WZeOCaLwi4mrnQ5/+DGKkJ4a/w5sqFTwtJIGIIAuDCn/uDm8kIDUVkbeznSOLoPA
67cxiqgIdqZ5pqUoid2YsDj20owrGDG4wUF6ZvhM9g/5va3CAhxqvTE2HwjhHTfz
Cgl8Nlvalz7YxXEf2clFEiEVa1fVaGMl9pCyedAmTfd6hoivcpAsopvXfVaaaR2y
iuZidpUfFhSk+Ls7TU/kB74ckfUGj5q/5HcKJgb/S+FYUV7eu0ewzTyW1uRl/d0U
Tb7e7EjgDGJsjOTMdTrMfv8ho8kAAAAAAAA=
-----END PKCS7-----
`
