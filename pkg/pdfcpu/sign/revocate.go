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
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ocsp"
)

const (
	crlReasonUnspecified = iota
	crlReasonKeyCompromise
	crlReasonCACompromise
	crlReasonAffiliationChanged
	crlReasonSuperseded
	crlReasonCessationOfOperation
	crlReasonCertificateHold
	_ // unused
	crlReasonRemoveFromCRL
	crlReasonPrivilegeWithdrawn
	crlReasonAACompromise
)

type OtherRevInfo struct {
	Type  asn1.ObjectIdentifier
	Value []byte
}

type RevocationInfoArchival struct {
	CRLs         []asn1.RawValue `asn1:"optional,explicit,tag:0"` // [0] EXPLICIT SEQUENCE of CRLs, OPTIONAL          RFC 5280
	OCSPs        []asn1.RawValue `asn1:"optional,explicit,tag:1"` // [1] EXPLICIT SEQUENCE of OCSPResponse, OPTIONAL  RFC 6960
	OtherRevInfo []OtherRevInfo  `asn1:"optional,explicit,tag:2"` // [2] EXPLICIT SEQUENCE of OtherRevInfo, OPTIONAL
}

func checkRevocation(
	cert, issuer *x509.Certificate,
	rootCerts *x509.CertPool,
	signer *model.Signer,
	certDetails *model.CertificateDetails,
	signingTime *time.Time,
	crls [][]byte,
	ocsps [][]byte,
	result *model.SignatureValidationResult,
	conf *model.Configuration) {

	revocationDetails, err := checkCertificateRevocation(cert, issuer, rootCerts, signer, signingTime, crls, ocsps, conf)
	if err != nil {
		signer.AddProblem(fmt.Sprintf("certificate revocation check failed: %v", err))
		certDetails.Revocation.Reason = fmt.Sprintf("%v", err)
		if result.Reason == model.SignatureReasonUnknown {
			result.Reason = model.SignatureReasonCertNotTrusted
		}
		return
	}

	certDetails.Revocation = *revocationDetails

	// The certificate is revoked and considered invalid.
	if certDetails.Revocation.Status == model.False {
		if result.Reason == model.SignatureReasonUnknown {
			result.Reason = model.SignatureReasonCertRevoked
		}
		return
	}

	// The certificate revocation status is unknown.
	if certDetails.Revocation.Status == model.Unknown {
		if result.Reason == model.SignatureReasonUnknown {
			result.Reason = model.SignatureReasonCertNotTrusted
		}
	}
}

func checkCertificateRevocation(
	cert, issuer *x509.Certificate,
	rootCerts *x509.CertPool,
	signer *model.Signer,
	signingTime *time.Time,
	crls [][]byte,
	ocsps [][]byte,
	conf *model.Configuration) (*model.RevocationDetails, error) {

	// Hybrid Approach - configure your preferredCertRevocationChecker in config.yml

	var f1, f2 func(
		cert, issuer *x509.Certificate,
		rootCerts *x509.CertPool,
		signingTime *time.Time,
		bbb [][]byte, // crls or ocsps
		conf *model.Configuration) (*model.RevocationDetails, error)

	pcrc := conf.PreferredCertRevocationChecker

	if len(crls) > 0 && len(ocsps) == 0 {
		pcrc = model.CRL
	}
	if len(crls) == 0 && len(ocsps) > 0 {
		pcrc = model.OCSP
	}

	f1, f2 = checkCertAgainstCRL, checkCertViaOCSP
	f1bbb, f2bbb := crls, ocsps
	if pcrc == model.OCSP {
		f1, f2 = f2, f1
		f1bbb, f2bbb = f2bbb, f1bbb
	}

	revocationDetails, err := f1(cert, issuer, rootCerts, signingTime, f1bbb, conf)
	if err == nil {
		return revocationDetails, nil
	}

	s := "CRL"
	if pcrc == model.OCSP {
		s = "OCSP"
	}
	signer.AddProblem(fmt.Sprintf("%s certificate revocation check failed: %v", s, err))

	// Fall back revocation checker.
	return f2(cert, issuer, rootCerts, signingTime, f2bbb, conf)
}

func checkCertAgainstCRL(
	cert, issuer *x509.Certificate,
	rootCerts *x509.CertPool,
	signingTime *time.Time,
	crls [][]byte,
	conf *model.Configuration) (*model.RevocationDetails, error) {

	if signingTime != nil && len(crls) > 0 {
		// Assumption: signingTime in the past
		rd, err := processArchivedCRLs(cert, *signingTime, crls)
		if err != nil || rd != nil {
			return rd, err
		}
	}

	if conf.Offline {
		return nil, errors.New("offline: unable to check CRLs")
	}

	if len(cert.CRLDistributionPoints) == 0 {
		return nil, errors.New("no CRL distribution points found")
	}

	return processCurrentCRLs(cert, conf)
}

func processArchivedCRLs(cert *x509.Certificate, signingTime time.Time, crls [][]byte) (*model.RevocationDetails, error) {

	const (
		reasonUnspecified   = 0
		reasonKeyCompromise = 1
		reasonCACompromise  = 2
	)

	ok := false
	for _, bb := range crls {
		crl, err := x509.ParseRevocationList(bb)
		if err != nil {
			return nil, errors.Errorf("failed to process archived CRL: %v", err)
		}

		if crl.NextUpdate.IsZero() || crl.ThisUpdate.After(signingTime) || crl.NextUpdate.Before(signingTime) {
			continue
		}

		ok = true

		for _, revoked := range crl.RevokedCertificateEntries {
			if revoked.SerialNumber.Cmp(cert.SerialNumber) != 0 {
				continue
			}

			switch revoked.ReasonCode {
			case reasonUnspecified, reasonKeyCompromise, reasonCACompromise:
				if !revoked.RevocationTime.After(signingTime) {
					return &model.RevocationDetails{
						Status: model.False,
						Reason: fmt.Sprintf("CRL: revoked due to %v at %v (before or at signing time)", revoked.ReasonCode, revoked.RevocationTime),
					}, nil
				}

			default:
				if revoked.RevocationTime.Before(signingTime) {
					return &model.RevocationDetails{
						Status: model.False,
						Reason: fmt.Sprintf("CRL: revoked due to %v at %v (before signing time)", revoked.ReasonCode, revoked.RevocationTime),
					}, nil
				}
				return &model.RevocationDetails{Status: model.True, Reason: "revoked after signing time, not relevant for timestamp"}, nil
			}
		}
	}
	if ok {
		return &model.RevocationDetails{Status: model.True, Reason: "not revoked (CRL check ok)"}, nil
	}
	return nil, nil
}

func processCurrentCRLs(cert *x509.Certificate, conf *model.Configuration) (*model.RevocationDetails, error) {
	client := &http.Client{
		Timeout: time.Duration(conf.TimeoutCRL) * time.Second,
	}

	now := time.Now()

	for _, url := range cert.CRLDistributionPoints {

		resp, err := client.Get(url)
		if err != nil {
			return nil, errors.Errorf("failed to fetch CRL from %s: %v", url, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, errors.Errorf("CRL responder at: %s returned http status: %d", url, resp.StatusCode)
		}

		crlData, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Errorf("CRL: read error: %v", err)
		}

		crl, err := x509.ParseRevocationList(crlData)
		if err != nil {
			return nil, errors.Errorf("CRL: parse error: %v", err)
		}

		if now.Before(crl.ThisUpdate) || now.After(crl.NextUpdate) {
			continue
		}

		for _, revoked := range crl.RevokedCertificateEntries {
			if revoked.SerialNumber.Cmp(cert.SerialNumber) == 0 {
				return &model.RevocationDetails{Status: model.False, Reason: "revoked (CRL check not ok)"}, nil
			}
		}
	}

	return &model.RevocationDetails{Status: model.True, Reason: "not revoked (CRL check ok)"}, nil
}

func checkCertViaOCSP(
	cert, issuer *x509.Certificate,
	rootCerts *x509.CertPool,
	signingTime *time.Time,
	ocsps [][]byte,
	conf *model.Configuration) (*model.RevocationDetails, error) {

	if conf.Offline {
		return nil, errors.New("offline: unable to contact OSCP responder") // / unable to verify OSCP certificate")
	}

	client := &http.Client{
		Timeout: time.Duration(conf.TimeoutOCSP) * time.Second,
	}

	if issuer == nil {
		c, err := getIssuerCertificate(cert, rootCerts, client)
		if err != nil {
			return nil, errors.Errorf("OCSP: failed to load certificate issuer: %v", err)
		}
		issuer = c
	}

	if signingTime != nil && len(ocsps) > 0 {
		// Assumption: signingTime in the past
		rd, err := processArchivedOCSPResponses(cert, issuer, rootCerts, *signingTime, ocsps, client)
		if err != nil || rd != nil {
			return rd, err
		}
	}

	if len(cert.OCSPServer) == 0 {
		return nil, errors.New("no OCSP responder found in certificate")
	}

	return processCurrentOCSPResponses(cert, issuer, rootCerts, client)
}

func processArchivedOCSPResponses(
	cert, issuer *x509.Certificate,
	rootCerts *x509.CertPool,
	signingTime time.Time,
	ocsps [][]byte,
	client *http.Client) (*model.RevocationDetails, error) {

	var lastErr error

	for _, bb := range ocsps {
		resp, err := ocsp.ParseResponseForCert(bb, cert, issuer)
		if err != nil {
			lastErr = err
			continue
		}

		if err := checkArchivedOCSPResponse(resp, signingTime); err != nil {
			return nil, err
		}

		if err := checkResponderCert(resp, rootCerts); err != nil {
			return nil, err
		}

		switch resp.Status {
		case ocsp.Good:
			return &model.RevocationDetails{Status: model.True, Reason: "not revoked (OCSP responder says \"Good\")"}, nil
		case ocsp.Revoked:
			return &model.RevocationDetails{Status: model.False, Reason: "revoked (OCSP responder says \"Revoked\")"}, nil
		case ocsp.Unknown:
			return &model.RevocationDetails{Status: model.Unknown, Reason: "OCSP responder returned \"Unknown\""}, nil
		}
	}

	if lastErr != nil {
		return nil, errors.Errorf("no valid OCSP response found, last parse error: %v", lastErr)
	}
	return nil, errors.New("no valid OCSP response found")
}

func checkArchivedOCSPResponse(resp *ocsp.Response, signingTime time.Time) error {
	const skew = 5 * time.Minute

	// ProducedAt should not be before this update.
	if !resp.ProducedAt.IsZero() && resp.ProducedAt.Before(resp.ThisUpdate) {
		// TODO Warning instead of error
		return errors.New("OCSP: response ProducedAt is before ThisUpdate")
	}

	// ProducedAt should not be after signing time (with tolerance).
	if !resp.ProducedAt.IsZero() && resp.ProducedAt.After(signingTime.Add(skew)) {
		return errors.New("OCSP: response is suspicious")
	}

	// NextUpdate should not be before signing time (expired).
	if !resp.NextUpdate.IsZero() && resp.NextUpdate.Before(signingTime) {
		return errors.New("OCSP: response is expired")
	}

	// ThisUpdate should not be after signing time (with tolerance).
	if resp.ThisUpdate.After(signingTime.Add(skew)) {
		return errors.New("OCSP: ThisUpdate is after signing time")
	}

	return nil
}

func processCurrentOCSPResponses(
	cert, issuer *x509.Certificate,
	rootCerts *x509.CertPool,
	client *http.Client) (*model.RevocationDetails, error) {

	ocspRequest, err := ocsp.CreateRequest(cert, issuer, nil)
	if err != nil {
		return nil, errors.Errorf("OCSP: failed to create request: %v", err)
	}

	ocspURL := cert.OCSPServer[0]

	resp, err := client.Post(ocspURL, "application/ocsp-request", io.NopCloser(bytes.NewReader(ocspRequest)))
	if err != nil {
		return nil, errors.Errorf("OCSP: failed to send request to %s: %v", ocspURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("OCSP responder at: %s returned http status: %d", ocspURL, resp.StatusCode)
	}

	ocspResponseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Errorf("OCSP: failed to read response: %v", err)
	}

	ocspResponse, err := ocsp.ParseResponse(ocspResponseData, nil)
	if err != nil {
		return nil, errors.Errorf("OCSP: failed to parse response: %v", err)
	}

	if err := checkCurrentOCSPResponse(ocspResponse); err != nil {
		return nil, err
	}

	if err := checkResponderCert(ocspResponse, rootCerts); err != nil {
		return nil, err
	}

	switch ocspResponse.Status {
	case ocsp.Good:
		return &model.RevocationDetails{Status: model.True, Reason: "not revoked (OCSP responder says \"Good\")"}, nil
	case ocsp.Revoked:
		return &model.RevocationDetails{Status: model.False, Reason: "revoked (OCSP responder says \"Revoked\")"}, nil
	case ocsp.Unknown:
		return &model.RevocationDetails{Status: model.Unknown, Reason: "OCSP responder says \"Unknown\""}, nil
	}

	return nil, errors.New("unexpected OCSP response")
}

func checkCurrentOCSPResponse(resp *ocsp.Response) error {
	const skew = 5 * time.Minute
	now := time.Now()

	// ProducedAt should not be before this update.
	if !resp.ProducedAt.IsZero() && resp.ProducedAt.Before(resp.ThisUpdate) {
		// TODO Warning instead of error
		return errors.New("OCSP: response ProducedAt is before ThisUpdate")
	}

	// ProducedAt should not be in the future (with tolerance).
	if !resp.ProducedAt.IsZero() && resp.ProducedAt.After(now.Add(skew)) {
		return errors.Errorf("OCSP: response ProducedAt (%v) is in the future", resp.ProducedAt)
	}

	// ThisUpdate should not be in the future (with tolerance).
	if resp.ThisUpdate.After(now.Add(skew)) {
		return errors.Errorf("OCSP: ThisUpdate (%v) is in the future", resp.ThisUpdate)
	}

	// NextUpdate should not be in the past (expired).
	if !resp.NextUpdate.IsZero() && resp.NextUpdate.Before(now) {
		return errors.Errorf("OCSP: response is expired (NextUpdate: %v < now: %v)", resp.NextUpdate, now)
	}

	return nil
}

func checkResponderCert(resp *ocsp.Response, rootCerts *x509.CertPool) error {
	cert, err := findOCSPResponderCert(resp, rootCerts)
	if err != nil {
		return errors.Errorf("OCSP: failed to find responder certificate: %v", err)
	}

	// Validate OCSP response signature using responder's certificate
	if err := resp.CheckSignatureFrom(cert); err != nil {
		return errors.Errorf("OCSP: invalid response signature: %v", err)
	}

	// Check if the OCSP responder has the No Check extension
	if hasNoCheckExtension(cert) {
		return errors.New("OCSP: disabled for cert by responder")
	}

	// Must have OCSP Signing EKU
	if found := slices.Contains(resp.Certificate.ExtKeyUsage, x509.ExtKeyUsageOCSPSigning); !found {
		return errors.New("OCSP signer cert missing OCSP Signing EKU")
	}

	// TODO check if resp.Certificate chains up to issuer

	return nil
}

func findOCSPResponderCert(resp *ocsp.Response, rootCerts *x509.CertPool) (*x509.Certificate, error) {
	if resp.Certificate != nil {
		return resp.Certificate, nil
	}
	for _, rawCert := range rootCerts.Subjects() {
		cert, err := x509.ParseCertificate(rawCert)
		if err == nil && bytes.Equal(cert.SubjectKeyId, resp.ResponderKeyHash) {
			return cert, nil
		}
	}
	return nil, errors.New("OCSP: responder certificate unavailable")
}

func hasNoCheckExtension(cert *x509.Certificate) bool {
	for _, ext := range cert.Extensions {
		if ext.Id.Equal(oidOCSPNoCheck) {
			return true
		}
	}
	return false
}

func getIssuerCertificate(cert *x509.Certificate, pool *x509.CertPool, client *http.Client) (*x509.Certificate, error) {
	// Try to find the issuer in the provided CertPool
	for _, potentialIssuer := range pool.Subjects() {
		candidate, err := x509.ParseCertificate(potentialIssuer)
		if err == nil && cert.CheckSignatureFrom(candidate) == nil {
			return candidate, nil // Found the issuer
		}
	}
	return nil, errors.Errorf("issuer certificate not found")
}
