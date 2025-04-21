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
	"crypto/dsa"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/hhrutter/pkcs7"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

const CertifiedSigPermsNotSupported = "Certified signature detected. Permission validation not supported."

func validateCertChains(
	chains [][]*x509.Certificate, // All chain paths for cert leading to a root CA.
	rootCerts *x509.CertPool,
	signer *model.Signer,
	signingTime *time.Time,
	crls [][]byte,
	ocsps [][]byte,
	result *model.SignatureValidationResult,
	conf *model.Configuration) {

	var cd *model.CertificateDetails

	// TODO Process all chains.
	chain := chains[0]

	for i, cert := range chain {

		certDetails := model.CertificateDetails{}

		if signer.Certificate == nil {
			signer.Certificate = &certDetails
		} else {
			cd.IssuerCertificate = &certDetails
		}
		cd = &certDetails

		if ok := setupCertDetails(cert, &certDetails, signer, signingTime, result, i); !ok {
			continue
		}

		selfSigned, err := isSelfSigned(cert)
		if selfSigned {
			certDetails.SelfSigned = true
		}
		if err != nil {
			signer.AddProblem(fmt.Sprintf("selfSigned cert verification for against public key failed: %s: %v\n", certInfo(cert), err))
			if result.Reason == model.SignatureReasonUnknown {
				result.Reason = model.SignatureReasonSelfSignedCertErr
			}
			certDetails.Trust.Status = model.False
			certDetails.Trust.Reason = "certificate not trusted"
			continue
		}

		if selfSigned || certDetails.CA {
			certDetails.Trust.Status = model.True
			certDetails.Trust.Reason = "CA"
			if selfSigned {
				certDetails.Trust.Reason = "self signed"
			}
			continue
		}

		if certDetails.Expired && signingTime == nil && len(crls) == 0 && len(ocsps) == 0 {
			certDetails.Trust.Status = model.False
			certDetails.Trust.Reason = "certificate expired"
			continue
		}

		setTrustStatus(&certDetails, result)

		var issuer *x509.Certificate
		if len(chain) > 1 {
			issuer = chain[1]
		}
		checkRevocation(cert, issuer, rootCerts, signer, &certDetails, signingTime, crls, ocsps, result, conf)
	}
}

func setupCertDetails(
	cert *x509.Certificate,
	certDetails *model.CertificateDetails,
	signer *model.Signer,
	signingTime *time.Time,
	result *model.SignatureValidationResult,
	i int) bool {

	certDetails.Leaf = i == 0
	certDetails.Subject = cert.Subject.CommonName
	certDetails.Issuer = cert.Issuer.CommonName
	certDetails.SerialNumber = cert.SerialNumber.Text(16)
	certDetails.Version = cert.Version
	certDetails.ValidFrom = cert.NotBefore
	certDetails.ValidThru = cert.NotAfter

	ts := time.Now()
	if signingTime != nil {
		ts = *signingTime
	}
	certDetails.Expired = ts.Before(cert.NotBefore) || ts.After(cert.NotAfter)

	certDetails.Usage = certUsage(cert)
	certDetails.Qualified = qualifiedCertificate(cert)
	certDetails.CA = cert.IsCA

	certDetails.SignAlg = cert.PublicKeyAlgorithm.String()

	keySize, ok := getKeySize(cert, signer, certDetails, result)
	if !ok {
		return false
	}
	certDetails.KeySize = keySize

	return true
}

func getKeySize(cert *x509.Certificate, signer *model.Signer, certDetails *model.CertificateDetails, result *model.SignatureValidationResult) (int, bool) {
	keySize, err := publicKeySize(cert)
	if err == nil {
		return keySize, true
	}
	signer.AddProblem(fmt.Sprintf("%v", err))
	if result.Reason == model.SignatureReasonUnknown {
		result.Reason = model.SignatureReasonCertNotTrusted
	}
	certDetails.Trust.Status = model.False
	certDetails.Trust.Reason = "certificate not trusted"
	return 0, false
}

func setTrustStatus(certDetails *model.CertificateDetails, result *model.SignatureValidationResult) {
	if result.Reason == model.SignatureReasonCertNotTrusted {
		certDetails.Trust.Status = model.False
		certDetails.Trust.Reason = "certificate not trusted"
	} else {
		certDetails.Trust.Status = model.True
		certDetails.Trust.Reason = "cert chain up to root CA is trusted"
	}
}

func signedData(ra io.ReaderAt, sigDict types.Dict) ([]byte, error) {
	arr := sigDict.ArrayEntry("ByteRange")
	if len(arr) != 4 {
		return nil, errors.New("pdfcpu: invalid signature dict - missing \"ByteRange\"")
	}
	return bytesForByteRange(ra, arr)
}

func bytesForByteRange(ra io.ReaderAt, arr types.Array) ([]byte, error) {
	off1 := int64((arr[0].(types.Integer)).Value())
	size1 := int64((arr[1].(types.Integer)).Value())
	off2 := int64((arr[2].(types.Integer)).Value())
	size2 := int64((arr[3].(types.Integer)).Value())

	buf1 := make([]byte, size1)
	_, err := ra.ReadAt(buf1, off1)
	if err != nil {
		return nil, err
	}

	buf2 := make([]byte, size2)
	_, err = ra.ReadAt(buf2, off2)
	if err != nil {
		return nil, err
	}

	return append(buf1, buf2...), nil
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

func qualifiedCertificate(cert *x509.Certificate) bool {
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
	certMap := make(map[string]*x509.Certificate)
	for _, cert := range certs {
		certMap[string(cert.RawSubject)] = cert
	}

	current := cert

	var sorted []*x509.Certificate

	for current != nil && len(sorted) < len(certs) {
		sorted = append(sorted, current)
		current = certMap[string(current.RawIssuer)]
	}

	return sorted
}

func publicKeySize(cert *x509.Certificate) (int, error) {
	switch pubKey := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		return pubKey.Size() * 8, nil
	case *ecdsa.PublicKey:
		return pubKey.Curve.Params().BitSize, nil
	case ed25519.PublicKey:
		return 256, nil
	case *dsa.PublicKey:
		return pubKey.Y.BitLen(), nil
	default:
		return 0, errors.Errorf("unknown public key type %T", pubKey)
	}
}

func handleCertVerifyErr(err error, cert *x509.Certificate, signer *model.Signer, result *model.SignatureValidationResult) {
	switch certErr := err.(type) {
	case x509.UnknownAuthorityError:
		if result.Reason == model.SignatureReasonUnknown {
			result.Reason = model.SignatureReasonCertNotTrusted
		}
	case x509.CertificateInvalidError:
		if certErr.Reason == x509.Expired {
			if result.Reason == model.SignatureReasonUnknown {
				result.Reason = model.SignatureReasonCertExpired
			}
		} else {
			if result.Reason == model.SignatureReasonUnknown {
				result.Reason = model.SignatureReasonCertInvalid
			}
		}
	default:
		if result.Reason == model.SignatureReasonUnknown {
			result.Reason = model.SignatureReasonCertInvalid
		}
	}
	signer.AddProblem(fmt.Sprintf("certificate verification failed for %s: %v", certInfo(cert), err))
}

func certInfo(cert *x509.Certificate) string {
	return fmt.Sprintf("serial=%q", cert.SerialNumber.Text(16))
}

func processDSS(ctx *model.Context, signer *model.Signer) ([]*x509.Certificate, [][]byte, [][]byte, bool) {
	ok := true
	dssCerts, err := extractCertsFromDSS(ctx)
	if err != nil {
		signer.AddProblem(fmt.Sprintf("DSS: extract certs: %v", err))
		ok = false
	}

	dssCRLs, err := extractCRLsFromDSS(ctx)
	if err != nil {
		signer.AddProblem(fmt.Sprintf("DSS: extract crls %v", err))
		ok = false
	}

	dssOCSPs, err := extractOCSPsFromDSS(ctx)
	if err != nil {
		signer.AddProblem(fmt.Sprintf("DSS: extract ocsps %v", err))
		ok = false
	}

	if _, ok := ctx.DSS.Find("VRI"); ok {
		signer.AddProblem("DSS: VRI currently unsupported")
		ok = false
	}

	return dssCerts, dssCRLs, dssOCSPs, ok
}

func extractCertsFromDSS(ctx *model.Context) ([]*x509.Certificate, error) {
	entry, found := ctx.DSS.Find("Certs")
	if !found {
		return nil, nil
	}

	arr, err := ctx.DereferenceArray(entry)
	if err != nil {
		return nil, err
	}

	var certs []*x509.Certificate

	for _, obj := range arr {
		sd, _, err := ctx.DereferenceStreamDict(obj)
		if err != nil {
			return nil, err
		}
		if sd == nil {
			return nil, errors.New("invalid DSS cert streamdict")
		}
		if err := sd.Decode(); err != nil {
			return nil, err
		}
		cert, err := x509.ParseCertificate(sd.Content)
		if err != nil {
			return nil, err
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
	entry, found := ctx.DSS.Find("CLRs")
	if !found {
		return nil, nil
	}

	arr, err := ctx.DereferenceArray(entry)
	if err != nil {
		return nil, err
	}

	var crls [][]byte

	for _, obj := range arr {
		sd, _, err := ctx.DereferenceStreamDict(obj)
		if err != nil {
			return nil, err
		}
		if sd == nil {
			return nil, errors.New("invalid DSS CRL streamdict")
		}
		if err := sd.Decode(); err != nil {
			return nil, err
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
		return nil, err
	}

	var ocsps [][]byte

	for _, obj := range arr {
		sd, _, err := ctx.DereferenceStreamDict(obj)
		if err != nil {
			return nil, err
		}
		if sd == nil {
			return nil, errors.New("invalid DSS OCSP streamdict")
		}
		if err := sd.Decode(); err != nil {
			return nil, err
		}
		ocsps = append(ocsps, sd.Content)
	}

	return ocsps, nil
}

func validateP7(sigDict types.Dict, result *model.SignatureValidationResult) *pkcs7.PKCS7 {
	p7, err := p7(sigDict)
	if err != nil {
		result.Reason = model.SignatureReasonInternal
		result.AddProblem(fmt.Sprintf("pkcs5: %v", err))
		return nil
	}

	if len(p7.Signers) == 0 {
		result.Reason = model.SignatureReasonInternal
		result.AddProblem("pkcs7: message without signers")
		return nil
	}

	if len(p7.Signers) != 1 && result.Details.IsETSI_CAdES_detached() {
		result.Reason = model.SignatureReasonInternal
		result.AddProblem("pkcs7: \"ETSI.CAdES.detached\" requires a single signer")
		return nil
	}

	return p7
}
