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
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
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

// RFC 6960 requires ThisUpdate to be sufficiently recent but leaves the exact window to local policy.
const defaultOCSPResponseMaxAge = 24 * time.Hour

const maxArchivedOCSPFailureCauses = 16

const maxRevocationResponseBytes int64 = 64 << 20

var errRevocationResponseTooLarge = errors.New("revocation response exceeds size limit")

// OtherRevInfo represents an additional revocation-information value in a
// RevocationInfoArchival attribute.
type OtherRevInfo struct {
	Type  asn1.ObjectIdentifier
	Value []byte
}

// RevocationInfoArchival represents embedded CRL, OCSP and other revocation
// information carried by an Adobe revocationInfoArchival attribute.
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
	crls [][]byte,
	ocsps [][]byte,
	result *model.SignatureValidationResult,
	conf *model.Configuration) {
	revocationDetails, err := checkCertificateRevocation(cert, issuer, rootCerts, signer, crls, ocsps, conf)
	if revocationDetails != nil {
		certDetails.Revocation = *revocationDetails
	}
	if err != nil {
		signer.AddProblem(fmt.Sprintf("certificate revocation check for %s: %v", certInfo(cert), err))
		certDetails.Revocation.Reason = fmt.Sprintf("%v", err)
		if result.Reason == model.SignatureReasonUnknown {
			result.Reason = model.SignatureReasonCertRevocationUnknown
		}
		return
	}

	// The assessed signing certificate is revoked and considered invalid.
	if certDetails.Revocation.Status == model.False {
		if result.Reason == model.SignatureReasonUnknown {
			result.Reason = model.SignatureReasonCertRevoked
		}
		return
	}

	// The certificate revocation status is unknown.
	if certDetails.Revocation.Status == model.Unknown {
		if result.Reason == model.SignatureReasonUnknown {
			result.Reason = model.SignatureReasonCertRevocationUnknown
		}
	}
}

func checkCertificateRevocation(
	cert, issuer *x509.Certificate,
	rootCerts *x509.CertPool,
	signer *model.Signer,
	crls [][]byte,
	ocsps [][]byte,
	conf *model.Configuration) (*model.RevocationDetails, error) {
	// Hybrid Approach - configure your preferredCertRevocationChecker in config.yml

	var f1, f2 func(
		cert, issuer *x509.Certificate,
		rootCerts *x509.CertPool,
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

	firstDetails, firstErr := f1(cert, issuer, rootCerts, f1bbb, conf)
	if firstErr == nil && revocationConcluded(firstDetails) {
		return firstDetails, nil
	}

	s := "CRL"
	if pcrc == model.OCSP {
		s = "OCSP"
	}
	if firstErr != nil {
		signer.AddProblem(fmt.Sprintf(
			"certificate revocation check for %s using %s: %v",
			certInfo(cert),
			s,
			firstErr,
		))
	}

	// Fall back revocation checker.
	secondDetails, secondErr := f2(cert, issuer, rootCerts, f2bbb, conf)
	details := mergeRevocationDetails(secondDetails, firstDetails)
	if secondErr != nil {
		return details, errors.Join(secondErr, firstErr)
	}
	return details, nil
}

func checkCertAgainstCRL(
	cert, issuer *x509.Certificate,
	rootCerts *x509.CertPool,
	crls [][]byte,
	conf *model.Configuration) (*model.RevocationDetails, error) {
	var archivedDetails *model.RevocationDetails
	var archivedErr error
	if len(crls) > 0 {
		archivedDetails, archivedErr = processArchivedCRLs(cert, issuer, crls)
	}

	if conf.Offline {
		return archivedDetails, errors.Join(errors.New("CRL: offline"), archivedErr)
	}

	if len(cert.CRLDistributionPoints) == 0 {
		return archivedDetails, errors.Join(errors.New("CRL: certificate has no distribution point"), archivedErr)
	}

	client := revocationHTTPClient(time.Duration(conf.TimeoutCRL)*time.Second, conf.AllowedRevocationHosts)
	currentDetails, err := processCurrentCRLs(cert, issuer, client)
	details := mergeRevocationDetails(currentDetails, archivedDetails)
	if revocationConcluded(currentDetails) {
		return details, nil
	}
	return details, errors.Join(err, archivedErr)
}

func processArchivedCRLs(
	_ *x509.Certificate,
	issuer *x509.Certificate,
	crls [][]byte,
) (*model.RevocationDetails, error) {
	var (
		observations []*model.CRLEvidence
		failures     []error
	)
	for i, bb := range crls {
		crl, err := x509.ParseRevocationList(bb)
		if err != nil {
			location := fmt.Sprintf("archived CRL %d", i+1)
			failures = appendIndexedFailure(failures, location, fmt.Errorf("parse list: %w", err))
			observations = append(observations, failedCRLEvidence(
				model.RevocationEvidenceSourceArchived,
				i+1,
				location,
				err,
			))
			continue
		}

		evidence := assessCRL(crl, issuer, model.RevocationEvidenceSourceArchived, false)
		evidence.Applicable = model.Unknown
		evidence.Index = i + 1
		evidence.Location = fmt.Sprintf("archived CRL %d", i+1)
		observations = append(observations, evidence)
	}
	details := crlRevocationDetails(
		model.Unknown,
		"CRL: archived evidence applicability unavailable",
		lastCRLEvidence(observations),
		observations,
	)
	return details, joinFailures(failures)
}

func processCurrentCRLs(
	cert, issuer *x509.Certificate,
	client *http.Client,
) (*model.RevocationDetails, error) {
	now := time.Now()
	var (
		observations          []*model.CRLEvidence
		authenticatedEvidence *model.CRLEvidence
		failures              []error
	)

	for i, url := range cert.CRLDistributionPoints {
		if err := validateRevocationURLString(url); err != nil {
			failures = append(failures, fmt.Errorf("CRL: fetch %s: %w", url, err))
			observations = append(observations, failedCRLEvidence(
				model.RevocationEvidenceSourceOnline,
				i+1,
				url,
				err,
			))
			continue
		}
		resp, err := client.Get(url)
		if err != nil {
			failures = append(failures, fmt.Errorf("CRL: fetch %s: %w", url, err))
			observations = append(observations, failedCRLEvidence(
				model.RevocationEvidenceSourceOnline,
				i+1,
				url,
				err,
			))
			continue
		}

		crlData, err := readAndCloseResponse(resp)
		if err != nil {
			failures = append(failures, fmt.Errorf("CRL: responder %s: %w", url, err))
			observations = append(observations, failedCRLEvidence(
				model.RevocationEvidenceSourceOnline,
				i+1,
				url,
				err,
			))
			continue
		}

		crl, err := x509.ParseRevocationList(crlData)
		if err != nil {
			failures = append(failures, fmt.Errorf("CRL: parse response from %s: %w", url, err))
			observations = append(observations, failedCRLEvidence(
				model.RevocationEvidenceSourceOnline,
				i+1,
				url,
				err,
			))
			continue
		}

		applicable := !now.Before(crl.ThisUpdate) && !now.After(crl.NextUpdate)
		evidence := assessCRL(crl, issuer, model.RevocationEvidenceSourceOnline, applicable)
		evidence.Index = i + 1
		evidence.Location = url
		observations = append(observations, evidence)
		if !authenticatedApplicableCRL(evidence) {
			continue
		}
		authenticatedEvidence = evidence

		for _, revoked := range crl.RevokedCertificateEntries {
			if revoked.SerialNumber.Cmp(cert.SerialNumber) == 0 {
				return crlRevocationDetails(
					model.False,
					"CRL: certificate status revoked",
					evidence,
					observations,
				), nil
			}
		}
	}

	if authenticatedEvidence != nil {
		return crlRevocationDetails(
			model.True,
			"CRL: certificate status good",
			authenticatedEvidence,
			observations,
		), nil
	}
	details := crlRevocationDetails(
		model.Unknown,
		"CRL: no applicable authenticated revocation list",
		lastCRLEvidence(observations),
		observations,
	)
	return details, errors.Join(failures...)
}

func assessCRL(
	crl *x509.RevocationList,
	issuer *x509.Certificate,
	source model.RevocationEvidenceSource,
	applicable bool,
) *model.CRLEvidence {
	evidence := &model.CRLEvidence{
		AssessmentScope: model.AssessmentScopeLocal,
		Source:          source,
		IssuerMatched:   model.Unknown,
		SignatureValid:  model.Unknown,
		Applicable:      boolStatus(applicable),
	}
	if crl == nil {
		evidence.Applicable = model.Unknown
		return evidence
	}
	evidence.Entries = crlRevocationEntries(crl)
	if unsupported := unsupportedCRLExtensions(crl); len(unsupported) > 0 {
		evidence.Applicable = model.Unknown
		evidence.Error = fmt.Sprintf(
			"unsupported CRL extensions: %v",
			unsupported,
		)
	}
	if issuer == nil {
		return evidence
	}
	evidence.IssuerMatched = boolStatus(bytes.Equal(crl.RawIssuer, issuer.RawSubject))
	evidence.SignatureValid = boolStatus(crl.CheckSignatureFrom(issuer) == nil)
	return evidence
}

func unsupportedCRLExtensions(crl *x509.RevocationList) []asn1.ObjectIdentifier {
	var unsupported []asn1.ObjectIdentifier
	for _, ext := range crl.Extensions {
		if !ext.Critical &&
			!ext.Id.Equal(oidDeltaCRLIndicator) &&
			!ext.Id.Equal(oidIssuingDistributionPoint) &&
			!ext.Id.Equal(oidFreshestCRL) {
			continue
		}
		unsupported = append(unsupported, ext.Id)
	}
	return unsupported
}

func authenticatedApplicableCRL(evidence *model.CRLEvidence) bool {
	return evidence != nil &&
		evidence.IssuerMatched == model.True &&
		evidence.SignatureValid == model.True &&
		evidence.Applicable == model.True
}

func crlRevocationEntries(crl *x509.RevocationList) []model.CRLRevocationEntry {
	entries := make([]model.CRLRevocationEntry, 0, len(crl.RevokedCertificateEntries))
	for _, entry := range crl.RevokedCertificateEntries {
		entries = append(entries, model.CRLRevocationEntry{
			SerialNumber:   entry.SerialNumber.Text(16),
			RevocationTime: entry.RevocationTime,
			ReasonCode:     entry.ReasonCode,
		})
	}
	return entries
}

func failedCRLEvidence(
	source model.RevocationEvidenceSource,
	index int,
	location string,
	err error,
) *model.CRLEvidence {
	return &model.CRLEvidence{
		AssessmentScope: model.AssessmentScopeLocal,
		Source:          source,
		Index:           index,
		Location:        location,
		Error:           err.Error(),
		IssuerMatched:   model.Unknown,
		SignatureValid:  model.Unknown,
		Applicable:      model.Unknown,
	}
}

func crlRevocationDetails(
	status int,
	reason string,
	evidence *model.CRLEvidence,
	observations []*model.CRLEvidence,
) *model.RevocationDetails {
	return &model.RevocationDetails{
		Status: status,
		Reason: reason,
		CRL:    evidence,
		CRLs:   observations,
	}
}

func lastCRLEvidence(observations []*model.CRLEvidence) *model.CRLEvidence {
	if len(observations) == 0 {
		return nil
	}
	return observations[len(observations)-1]
}

func revocationConcluded(details *model.RevocationDetails) bool {
	return details != nil && (details.Status == model.True || details.Status == model.False)
}

func mergeRevocationDetails(
	primary, observations *model.RevocationDetails,
) *model.RevocationDetails {
	if primary == nil {
		return observations
	}
	if observations == nil {
		return primary
	}
	primary.CRLs = append(
		append([]*model.CRLEvidence(nil), observations.CRLs...),
		primary.CRLs...,
	)
	if primary.CRL == nil {
		primary.CRL = observations.CRL
	}
	if primary.OCSP == nil {
		primary.OCSP = observations.OCSP
	}
	primary.OCSPs = append(
		append([]*model.OCSPEvidence(nil), observations.OCSPs...),
		primary.OCSPs...,
	)
	return primary
}

func boolStatus(ok bool) int {
	if ok {
		return model.True
	}
	return model.False
}

func readAndCloseResponseWithLimit(resp *http.Response, maxBytes int64) (bb []byte, err error) {
	if resp == nil {
		return nil, errors.New("missing response")
	}
	if resp.Body == nil {
		return nil, errors.New("missing response body")
	}
	if maxBytes <= 0 {
		return nil, fmt.Errorf("invalid response size limit: %d", maxBytes)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close response body: %w", closeErr))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}

	if resp.ContentLength > maxBytes {
		return nil, fmt.Errorf("%w: maximum %d bytes", errRevocationResponseTooLarge, maxBytes)
	}

	bb, err = io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	if int64(len(bb)) > maxBytes {
		return nil, fmt.Errorf("%w: maximum %d bytes", errRevocationResponseTooLarge, maxBytes)
	}
	return bb, nil
}

func readAndCloseResponse(resp *http.Response) ([]byte, error) {
	return readAndCloseResponseWithLimit(resp, maxRevocationResponseBytes)
}

func checkCertViaOCSP(
	cert, issuer *x509.Certificate,
	_ *x509.CertPool,
	ocsps [][]byte,
	conf *model.Configuration) (*model.RevocationDetails, error) {
	if issuer == nil {
		return nil, errors.New("OCSP: certificate issuer unavailable")
	}

	var archivedDetails *model.RevocationDetails
	var archivedErr error
	if len(ocsps) > 0 {
		archivedDetails, archivedErr = processArchivedOCSPResponses(cert, issuer, ocsps)
	}

	if conf.Offline {
		return archivedDetails, errors.Join(errors.New("OCSP: offline"), archivedErr)
	}

	if len(cert.OCSPServer) == 0 {
		return archivedDetails, errors.Join(errors.New("OCSP: certificate has no responder URL"), archivedErr)
	}

	client := revocationHTTPClient(time.Duration(conf.TimeoutOCSP)*time.Second, conf.AllowedRevocationHosts)
	rd, err := processCurrentOCSPResponses(cert, issuer, client)
	details := mergeRevocationDetails(rd, archivedDetails)
	if revocationConcluded(rd) {
		return details, nil
	}
	return details, errors.Join(err, archivedErr)
}

func processArchivedOCSPResponses(
	cert, issuer *x509.Certificate,
	ocsps [][]byte) (*model.RevocationDetails, error) {
	var (
		failures     []error
		observations []*model.OCSPEvidence
	)
	for i, bb := range ocsps {
		candidate, err := processArchivedOCSPResponse(cert, issuer, bb)
		location := fmt.Sprintf("archived response %d", i+1)
		if err != nil {
			failures = appendIndexedFailure(failures, location, err)
			observations = append(observations, failedOCSPCandidateEvidence(
				candidate,
				model.RevocationEvidenceSourceArchived,
				i+1,
				location,
				err,
			))
			continue
		}
		candidate.evidence.Index = i + 1
		candidate.evidence.Location = location
		observations = append(observations, candidate.evidence)
	}

	details := inconclusiveOCSPDetails(
		"OCSP: archived response applicability unavailable",
		observations,
	)
	if len(failures) == 0 {
		return details, nil
	}
	return details, fmt.Errorf("OCSP: no valid archived response: %w", joinFailures(failures))
}

type ocspCandidate struct {
	details  *model.RevocationDetails
	evidence *model.OCSPEvidence
}

func processArchivedOCSPResponse(
	cert, issuer *x509.Certificate,
	bb []byte,
) (*ocspCandidate, error) {
	resp, err := ocsp.ParseResponseForCert(bb, cert, issuer)
	if err != nil {
		return nil, fmt.Errorf("parse response for certificate: %w", err)
	}
	candidate := observedOCSPCandidate(model.RevocationEvidenceSourceArchived, resp)
	if err := observeArchivedOCSPResponder(resp, issuer, candidate.evidence); err != nil {
		return candidate, err
	}
	if candidate.evidence.Responder == model.OCSPResponderDelegated &&
		resp.Certificate != nil &&
		hasNoCheckExtension(resp.Certificate) {
		candidate.evidence.ResponderRevocation = model.True
	}
	candidate.details = &model.RevocationDetails{
		Status: model.Unknown,
		Reason: "OCSP: archived response applicability unavailable",
		OCSP:   candidate.evidence,
	}
	return candidate, nil
}

func observeArchivedOCSPResponder(
	resp *ocsp.Response,
	issuer *x509.Certificate,
	evidence *model.OCSPEvidence,
) error {
	cert, delegated, err := resolveOCSPResponderCertificate(resp, issuer)
	if err != nil {
		return err
	}
	evidence.Responder = model.OCSPResponderIssuer
	if delegated {
		evidence.Responder = model.OCSPResponderDelegated
	}
	if err := resp.CheckSignatureFrom(cert); err != nil {
		evidence.ResponseSignatureValid = model.False
		evidence.Authenticated = model.False
		return fmt.Errorf("OCSP: verify response signature: %w", err)
	}
	evidence.ResponseSignatureValid = model.True
	if !delegated {
		evidence.Authenticated = model.True
		return nil
	}
	if !slices.Contains(cert.ExtKeyUsage, x509.ExtKeyUsageOCSPSigning) {
		evidence.ResponderCertificateOCSPSigningValid = model.False
		evidence.Authenticated = model.False
		return errors.New("OCSP: responder certificate missing OCSP signing EKU")
	}
	evidence.ResponderCertificateOCSPSigningValid = model.True
	if issuer == nil {
		return errors.New("OCSP: authorize delegated responder: certificate issuer unavailable")
	}
	if err := cert.CheckSignatureFrom(issuer); err != nil {
		evidence.ResponderCertificateIssuedByIssuer = model.False
		evidence.Authenticated = model.False
		return fmt.Errorf("OCSP: verify delegated responder issuer: %w", err)
	}
	evidence.ResponderCertificateIssuedByIssuer = model.True
	return nil
}

func appendIndexedFailure(failures []error, location string, err error) []error {
	if len(failures) < maxArchivedOCSPFailureCauses {
		return append(failures, fmt.Errorf("%s: %w", location, err))
	}
	if len(failures) == maxArchivedOCSPFailureCauses {
		return append(failures, errors.New("additional revocation failures omitted"))
	}
	return failures
}

func joinFailures(failures []error) error {
	if len(failures) == 1 {
		return failures[0]
	}
	return errors.Join(failures...)
}

func ocspRevocationDetails(
	status int,
	evidence *model.OCSPEvidence,
) (*model.RevocationDetails, error) {
	evidence.CertificateStatus = ocspCertificateStatus(status)
	if evidence.Responder == model.OCSPResponderDelegated &&
		evidence.ResponderRevocation != model.True {
		return &model.RevocationDetails{
			Status: model.Unknown,
			Reason: "OCSP: delegated responder revocation not assessed",
			OCSP:   evidence,
		}, nil
	}
	switch status {
	case ocsp.Good:
		return &model.RevocationDetails{
			Status: model.True,
			Reason: "OCSP: certificate status good",
			OCSP:   evidence,
		}, nil
	case ocsp.Revoked:
		return &model.RevocationDetails{
			Status: model.False,
			Reason: "OCSP: certificate status revoked",
			OCSP:   evidence,
		}, nil
	case ocsp.Unknown:
		return &model.RevocationDetails{
			Status: model.Unknown,
			Reason: "OCSP: certificate status unknown",
			OCSP:   evidence,
		}, nil
	}
	return nil, errors.New("OCSP: unexpected response status")
}

func ocspCertificateStatus(status int) int {
	switch status {
	case ocsp.Good:
		return model.True
	case ocsp.Revoked:
		return model.False
	}
	return model.Unknown
}

func authenticatedOCSPEvidence(
	source model.RevocationEvidenceSource,
	responder model.OCSPResponder,
	resp *ocsp.Response,
) *model.OCSPEvidence {
	evidence := &model.OCSPEvidence{
		AssessmentScope:        model.AssessmentScopeLocal,
		Source:                 source,
		ProducedAt:             resp.ProducedAt,
		ThisUpdate:             resp.ThisUpdate,
		NextUpdate:             resp.NextUpdate,
		RevokedAt:              resp.RevokedAt,
		Applicable:             model.True,
		Responder:              responder,
		Authenticated:          model.True,
		ResponseSignatureValid: model.True,
		CertificateStatus:      ocspCertificateStatus(resp.Status),
	}
	if responder == model.OCSPResponderDelegated {
		evidence.ResponderCertificateIssuedByIssuer = model.True
		evidence.ResponderCertificateOCSPSigningValid = model.True
	}
	if responder == model.OCSPResponderDelegated &&
		resp != nil &&
		resp.Certificate != nil &&
		hasNoCheckExtension(resp.Certificate) {
		evidence.ResponderRevocation = model.True
	}
	return evidence
}

func failedOCSPEvidence(
	source model.RevocationEvidenceSource,
	index int,
	location string,
	err error,
) *model.OCSPEvidence {
	return &model.OCSPEvidence{
		AssessmentScope:                      model.AssessmentScopeLocal,
		Source:                               source,
		Index:                                index,
		Location:                             location,
		Error:                                err.Error(),
		Responder:                            model.OCSPResponderUnspecified,
		Applicable:                           model.Unknown,
		Authenticated:                        model.Unknown,
		ResponseSignatureValid:               model.Unknown,
		ResponderCertificateIssuedByIssuer:   model.Unknown,
		ResponderCertificateOCSPSigningValid: model.Unknown,
		CertificateStatus:                    model.Unknown,
		ResponderRevocation:                  model.Unknown,
	}
}

func failedOCSPCandidateEvidence(
	candidate *ocspCandidate,
	source model.RevocationEvidenceSource,
	index int,
	location string,
	err error,
) *model.OCSPEvidence {
	if candidate == nil || candidate.evidence == nil {
		return failedOCSPEvidence(source, index, location, err)
	}
	evidence := candidate.evidence
	evidence.Index = index
	evidence.Location = location
	evidence.Error = err.Error()
	return evidence
}

func observedOCSPCandidate(
	source model.RevocationEvidenceSource,
	resp *ocsp.Response,
) *ocspCandidate {
	return &ocspCandidate{
		evidence: &model.OCSPEvidence{
			AssessmentScope:                      model.AssessmentScopeLocal,
			Source:                               source,
			ProducedAt:                           resp.ProducedAt,
			ThisUpdate:                           resp.ThisUpdate,
			NextUpdate:                           resp.NextUpdate,
			RevokedAt:                            resp.RevokedAt,
			Applicable:                           model.Unknown,
			Responder:                            model.OCSPResponderUnspecified,
			Authenticated:                        model.Unknown,
			ResponseSignatureValid:               model.Unknown,
			ResponderCertificateIssuedByIssuer:   model.Unknown,
			ResponderCertificateOCSPSigningValid: model.Unknown,
			CertificateStatus:                    ocspCertificateStatus(resp.Status),
			ResponderRevocation:                  model.Unknown,
		},
	}
}

func inconclusiveOCSPDetails(
	reason string,
	observations []*model.OCSPEvidence,
) *model.RevocationDetails {
	var evidence *model.OCSPEvidence
	if len(observations) > 0 {
		evidence = observations[len(observations)-1]
	}
	return &model.RevocationDetails{
		Status: model.Unknown,
		Reason: reason,
		OCSP:   evidence,
		OCSPs:  observations,
	}
}

func concludeOCSPCandidates(
	candidates []*ocspCandidate,
	observations []*model.OCSPEvidence,
	defaultReason string,
) *model.RevocationDetails {
	return inconclusiveOCSPDetails(inconclusiveOCSPReason(candidates, defaultReason), observations)
}

func inconclusiveOCSPReason(candidates []*ocspCandidate, defaultReason string) string {
	for _, candidate := range candidates {
		if candidate != nil &&
			candidate.details != nil &&
			candidate.details.Reason == "OCSP: delegated responder revocation not assessed" {
			return candidate.details.Reason
		}
	}
	return defaultReason
}

func processCurrentOCSPResponses(
	cert, issuer *x509.Certificate,
	client *http.Client) (*model.RevocationDetails, error) {
	ocspRequest, err := ocsp.CreateRequest(cert, issuer, nil)
	if err != nil {
		return nil, fmt.Errorf("OCSP: create request: %w", err)
	}

	now := time.Now()
	var (
		failures     []error
		observations []*model.OCSPEvidence
		candidates   []*ocspCandidate
	)
	for i, ocspURL := range cert.OCSPServer {
		candidate, err := processCurrentOCSPResponse(cert, issuer, client, ocspRequest, ocspURL, now)
		if err != nil {
			location := fmt.Sprintf("responder %d (%s)", i+1, ocspURL)
			failures = appendIndexedFailure(failures, location, err)
			observations = append(observations, failedOCSPCandidateEvidence(
				candidate,
				model.RevocationEvidenceSourceOnline,
				i+1,
				ocspURL,
				err,
			))
			continue
		}
		candidate.evidence.Index = i + 1
		candidate.evidence.Location = ocspURL
		observations = append(observations, candidate.evidence)
		candidates = append(candidates, candidate)
		if revocationConcluded(candidate.details) {
			candidate.details.OCSPs = observations
			return candidate.details, nil
		}
	}

	details := concludeOCSPCandidates(
		candidates,
		observations,
		"OCSP: no conclusive current response",
	)
	if revocationConcluded(details) || len(failures) == 0 {
		return details, nil
	}
	return details, fmt.Errorf("OCSP: no conclusive current response: %w", joinFailures(failures))
}

func processCurrentOCSPResponse(
	cert, issuer *x509.Certificate,
	client *http.Client,
	request []byte,
	ocspURL string,
	now time.Time,
) (*ocspCandidate, error) {
	if err := validateRevocationURLString(ocspURL); err != nil {
		return nil, err
	}
	resp, err := client.Post(ocspURL, "application/ocsp-request", io.NopCloser(bytes.NewReader(request)))
	if err != nil {
		return nil, fmt.Errorf("OCSP: send request to %s: %w", ocspURL, err)
	}

	ocspResponseData, err := readAndCloseResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("OCSP: responder %s: %w", ocspURL, err)
	}

	ocspResponse, err := ocsp.ParseResponseForCert(
		ocspResponseData,
		cert,
		issuer,
	)
	if err != nil {
		return nil, fmt.Errorf("OCSP: parse response for certificate: %w", err)
	}
	candidate := observedOCSPCandidate(model.RevocationEvidenceSourceOnline, ocspResponse)

	if err := checkCurrentOCSPResponse(ocspResponse, now); err != nil {
		candidate.evidence.Applicable = model.False
		return candidate, err
	}
	candidate.evidence.Applicable = model.True

	responder, err := authenticateOCSPResponder(ocspResponse, issuer, now)
	if err != nil {
		return candidate, err
	}

	candidate.evidence = authenticatedOCSPEvidence(
		model.RevocationEvidenceSourceOnline,
		responder,
		ocspResponse,
	)
	candidate.details, err = ocspRevocationDetails(ocspResponse.Status, candidate.evidence)
	return candidate, err
}

func checkCurrentOCSPResponse(resp *ocsp.Response, now time.Time) error {
	const skew = 5 * time.Minute

	// ProducedAt should not be before this update.
	if !resp.ProducedAt.IsZero() && resp.ProducedAt.Before(resp.ThisUpdate) {
		// TODO Warning instead of error
		return errors.New("OCSP: ProducedAt precedes ThisUpdate")
	}

	// ProducedAt should not be in the future (with tolerance).
	if !resp.ProducedAt.IsZero() && resp.ProducedAt.After(now.Add(skew)) {
		return errors.New("OCSP: ProducedAt is in the future")
	}

	// ThisUpdate should not be in the future (with tolerance).
	if resp.ThisUpdate.After(now.Add(skew)) {
		return errors.New("OCSP: ThisUpdate is in the future")
	}
	if resp.ThisUpdate.IsZero() {
		return errors.New("OCSP: ThisUpdate is missing")
	}
	if resp.NextUpdate.IsZero() {
		if resp.ThisUpdate.Before(now.Add(-defaultOCSPResponseMaxAge)) {
			return errors.New("OCSP: ThisUpdate exceeds maximum response age")
		}
		return nil
	}

	// NextUpdate should not be in the past (expired).
	if resp.NextUpdate.Before(now) {
		return errors.New("OCSP: NextUpdate precedes current time")
	}

	return nil
}

func checkResponderCert(
	resp *ocsp.Response,
	issuer *x509.Certificate,
	validationTime time.Time,
) error {
	_, err := authenticateOCSPResponder(resp, issuer, validationTime)
	return err
}

func authenticateOCSPResponder(
	resp *ocsp.Response,
	issuer *x509.Certificate,
	validationTime time.Time,
) (model.OCSPResponder, error) {
	cert, delegated, err := resolveOCSPResponderCertificate(resp, issuer)
	if err != nil {
		return model.OCSPResponderUnspecified, err
	}

	if err := resp.CheckSignatureFrom(cert); err != nil {
		return model.OCSPResponderUnspecified, fmt.Errorf("OCSP: verify response signature: %w", err)
	}
	if delegated {
		// Must have OCSP Signing EKU
		if found := slices.Contains(cert.ExtKeyUsage, x509.ExtKeyUsageOCSPSigning); !found {
			return model.OCSPResponderDelegated, errors.New("OCSP: responder certificate missing OCSP signing EKU")
		}

		if err := verifyResponderCertificate(cert, issuer, validationTime); err != nil {
			return model.OCSPResponderDelegated, err
		}

		// The OCSP No Check extension permits skipping responder-certificate revocation checks.
		if hasNoCheckExtension(cert) {
			return model.OCSPResponderDelegated, nil
		}
	}

	// TODO check delegated responder-certificate revocation.

	if delegated {
		return model.OCSPResponderDelegated, nil
	}
	return model.OCSPResponderIssuer, nil
}

func resolveOCSPResponderCertificate(
	resp *ocsp.Response,
	issuer *x509.Certificate,
) (*x509.Certificate, bool, error) {
	if resp.Certificate == nil {
		if issuer == nil {
			return nil, false, errors.New("OCSP: responder certificate unavailable")
		}
		return issuer, false, nil
	}
	if issuer != nil && resp.Certificate.Equal(issuer) {
		return issuer, false, nil
	}
	return resp.Certificate, true, nil
}

func verifyResponderCertificate(
	cert, issuer *x509.Certificate,
	validationTime time.Time,
) error {
	if issuer == nil {
		return errors.New("OCSP: authorize delegated responder: certificate issuer unavailable")
	}

	issuerRoots := x509.NewCertPool()
	issuerRoots.AddCert(issuer)
	if _, err := cert.Verify(x509.VerifyOptions{
		Roots:       issuerRoots,
		CurrentTime: validationTime,
		KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageOCSPSigning},
	}); err != nil {
		return fmt.Errorf("OCSP: authorize delegated responder: %w", err)
	}
	// RFC 6960 permits independent responders only through explicit responder-to-CA configuration.
	return nil
}

func hasNoCheckExtension(cert *x509.Certificate) bool {
	for _, ext := range cert.Extensions {
		if ext.Id.Equal(oidOCSPNoCheck) {
			return true
		}
	}
	return false
}
