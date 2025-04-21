/*
Copyright 2025 The pdf Authors.

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

package pdfcpu

import (
	"crypto/x509"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/sign"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// ValidateSignatures validates all digital signatures of ctx.
func ValidateSignatures(ra io.ReaderAt, ctx *model.Context, all bool) ([]*model.SignatureValidationResult, error) {
	var results []*model.SignatureValidationResult

	if ctx.URSignature != nil {
		svr, err := validateURSignature(ctx.URSignature, ctx, ra)
		if err != nil {
			return nil, err
		}
		results = append(results, svr)
	}

	incrs := make([]int, 0, len(ctx.Signatures))
	for k := range ctx.Signatures {
		incrs = append(incrs, k)
	}
	sort.Ints(incrs)

	first, ok := true, false

	// NOTE: Long term validation is restricted to processing the latest doc timestamp (contained in the last increment).

	// Process all increments chronologically in reverse order.
	for i, inc := range incrs {
		for _, sig := range ctx.Signatures[inc] {

			if i > 0 && sig.Type == model.SigTypeDTS {
				continue
			}

			svr, err := validateSignature(sig, ctx, ra, first, all)
			if err != nil {
				return nil, err
			}
			results = append(results, svr)

			if sig.Type == model.SigTypeDTS {
				continue
			}

			if all {
				first = false
				continue
			}

			if checkForAbortAfterFirst(first, svr, ctx) {
				ok = true
				break
			}

			first = false
		}
		if ok {
			break
		}
	}

	return results, nil
}

func checkForAbortAfterFirst(first bool, svr *model.SignatureValidationResult, ctx *model.Context) bool {
	if first {
		if ctx.CertifiedSigObjNr == 0 || (svr.Certified() && svr.Permissions() != model.CertifiedSigPermNoChangesAllowed) {
			return true
		}
	}
	return svr.Certified()
}

func validateURSignature(sigDict types.Dict, ctx *model.Context, ra io.ReaderAt) (*model.SignatureValidationResult, error) {
	sig := model.Signature{Type: model.SigTypeUR, Visible: false, Signed: true}
	result := model.SignatureValidationResult{Signature: sig}

	result.Status = model.SignatureStatusUnknown
	result.Reason = model.SignatureReasonUnknown
	result.DocModified = model.Unknown

	result.Details = model.SignatureDetails{}
	result.Details.SignerIdentity = "Unknown"

	if err := resultDetails(sigDict, ctx, &result.Details); err != nil {
		return nil, err
	}

	subFilter := sigDict.NameEntry("SubFilter")
	if subFilter == nil {
		result.AddProblem("missing sigDict \"SubFilter\"")
		result.Reason = model.SignatureReasonInternal
		return &result, nil
	}
	result.Details.SubFilter = *subFilter

	var f func(
		ra io.ReaderAt,
		sigDict types.Dict,
		certified bool,
		authoriative bool,
		validateAll bool,
		perms int,
		rootCerts *x509.CertPool,
		result *model.SignatureValidationResult,
		ctx *model.Context) error

	switch *subFilter {
	case "adbe.x509.rsa_sha1": // deprecated as of PDF 2.0
		f = sign.ValidateX509RSASHA1Signature
	case "adbe.pkcs7.sha1": // deprecated as of PDF 2.0
		f = sign.ValidatePKCS7Signatures
	case "adbe.pkcs7.detached":
		f = sign.ValidatePKCS7Signatures
	case "ETSI.CAdES.detached":
		f = sign.ValidatePKCS7Signatures
	//case "ETSI.RFC3161":
	// TODO: Contents shall be the TimeStampToken as specified in Internet RFC 3161 as updated by Internet RFC 5816.
	default:
		result.AddProblem(fmt.Sprintf("unsupported subFilter: %s", *subFilter))
		return &result, nil
	}

	return &result, f(ra, sigDict, false, false, true, 0, model.UserCertPool, &result, ctx)
}

func validateSignature(sig model.Signature, ctx *model.Context, ra io.ReaderAt, first, all bool) (*model.SignatureValidationResult, error) {
	sigField, err := ctx.DereferenceDict(*types.NewIndirectRef(sig.ObjNr, 0))
	if err != nil {
		return nil, err
	}

	result := model.SignatureValidationResult{Signature: sig}

	result.Status = model.SignatureStatusUnknown
	result.Reason = model.SignatureReasonUnknown
	result.DocModified = model.Unknown

	result.Details = model.SignatureDetails{}
	result.Details.SignerIdentity = "Unknown"

	if sigField == nil {
		result.AddProblem("missing signature field")
		result.Reason = model.SignatureReasonInternal
		return &result, nil
	}

	if sl := sigField.StringLiteralEntry("T"); sl != nil {
		s, err := types.StringLiteralToString(*sl)
		if err != nil {
			return nil, err
		}
		result.Details.FieldName = strings.TrimSpace(s)
	}

	indRef := sigField.IndirectRefEntry("V")
	if indRef == nil {
		result.AddProblem("missing signature dict")
		result.Reason = model.SignatureReasonInternal
		return &result, nil
	}

	sigDict, err := ctx.DereferenceDict(*indRef)
	if err != nil {
		result.AddProblem(fmt.Sprintf("%v", err))
		result.Reason = model.SignatureReasonInternal
		return &result, nil
	}

	subFilter := sigDict.NameEntry("SubFilter")
	if subFilter == nil {
		result.AddProblem("missing sigDict \"SubFilter\"")
		result.Reason = model.SignatureReasonInternal
		return &result, nil
	}
	result.Details.SubFilter = *subFilter

	result.Signature.Certified = indRef.ObjectNumber.Value() == ctx.CertifiedSigObjNr
	if first && ctx.CertifiedSigObjNr == 0 {
		result.Signature.Authoritative = true
	}

	if err := resultDetails(sigDict, ctx, &result.Details); err != nil {
		return nil, err
	}

	perms, err := detectPermissions(sigDict, ctx)
	if err != nil {
		return nil, err
	}

	f := sigHandler(*subFilter)

	if f == nil {
		result.AddProblem(fmt.Sprintf("unsupported subFilter: %s", *subFilter))
		return &result, nil
	}

	return &result, f(ra, sigDict, result.Signature.Certified, result.Signature.Authoritative, all, perms, model.UserCertPool, &result, ctx)
}

func sigHandler(subFilter string) func(
	ra io.ReaderAt,
	sigDict types.Dict,
	certified bool,
	authoriative bool,
	validateAll bool,
	perms int,
	rootCerts *x509.CertPool,
	result *model.SignatureValidationResult,
	ctx *model.Context) error {

	switch subFilter {
	case "adbe.x509.rsa_sha1": // deprecated as of PDF 2.0
		return sign.ValidateX509RSASHA1Signature
	case "adbe.pkcs7.sha1": // deprecated as of PDF 2.0
		return sign.ValidatePKCS7Signatures
	case "adbe.pkcs7.detached":
		return sign.ValidatePKCS7Signatures
	case "ETSI.CAdES.detached":
		return sign.ValidatePKCS7Signatures
	case "ETSI.RFC3161":
		return sign.ValidateDTS
	}

	return nil
}

func resultDetails(sigDict types.Dict, ctx *model.Context, resultDetails *model.SignatureDetails) error {
	if sl := sigDict.StringLiteralEntry("Name"); sl != nil {
		s, err := types.StringLiteralToString(*sl)
		if err != nil {
			return err
		}
		resultDetails.SignerName = strings.TrimSpace(s)
	}

	if sl := sigDict.StringLiteralEntry("ContactInfo"); sl != nil {
		s, err := types.StringLiteralToString(*sl)
		if err != nil {
			return err
		}
		resultDetails.ContactInfo = strings.TrimSpace(s)
	}

	if sl := sigDict.StringLiteralEntry("Location"); sl != nil {
		s, err := types.StringLiteralToString(*sl)
		if err != nil {
			return err
		}
		resultDetails.Location = strings.TrimSpace(s)
	}

	if sl := sigDict.StringLiteralEntry("Reason"); sl != nil {
		s, err := types.StringLiteralToString(*sl)
		if err != nil {
			return err
		}
		resultDetails.Reason = strings.TrimSpace(s)
	}

	if o, ok := sigDict.Find("M"); ok {
		// informational (cannot be relied upon for long term validation)
		s, err := ctx.DereferenceStringOrHexLiteral(o, model.V10, nil)
		if err != nil {
			return err
		}
		if s != "" {
			if t, ok := types.DateTime(s, ctx.XRefTable.ValidationMode == model.ValidationRelaxed); ok {
				resultDetails.SigningTime = t
			}
		}
	}

	return nil
}

func detectPermissions(sigDict types.Dict, ctx *model.Context) (int, error) {
	o, found := sigDict.Find("Reference")
	if !found {
		return 0, nil
	}

	arr, err := ctx.DereferenceArray(o)
	if err != nil || len(arr) == 0 {
		return 0, err
	}

	// Process signature reference dicts.

	// TODO Process UR3 Params
	// <Reference, [
	// 			<<
	// 				<Data, (8530 0 R)>
	// 				<TransformMethod, UR3>
	// 				<TransformParams, <<
	// 					<Annots, [Create Delete Modify Copy Import Export]>
	// 					<Document, [FullSave]>
	// 					<Form, [Add FillIn Delete SubmitStandalone]>
	// 					<Signature, [Modify]>
	// 					<Type, TransformParams>
	// 					<V, 2.2>
	// 				>>>
	// 				<Type, SigRef>
	// 			>>
	// 			]>

	for _, obj := range arr {
		d, err := ctx.DereferenceDict(obj)
		if err != nil {
			return 0, err
		}
		if tm := d.NameEntry("TransformMethod"); tm == nil || *tm != "DocMDP" {
			continue
		}
		d1 := d.DictEntry("TransformParams")
		if len(d1) == 0 {
			continue
		}
		typ := d1.Type()
		if typ == nil || *typ != "TransformParams" {
			continue
		}
		i := d1.IntEntry("P")
		if i != nil {
			if *i < 1 || *i > 3 {
				return 0, errors.Errorf("invalid DocMDP permissions detected: %d ", *i)
			}
			return *i, nil
		}
		return 2, nil // default
	}

	/*
		array of signature reference dictionaries:

				<Reference, [ sig ref dict
					<<
						<Data, (5 0 R)>
						<DigestLocation, [0 0]>
						<DigestMethod, MD5>
						<DigestValue, (aa)>
						<TransformMethod, DocMDP>  Modification Detection and Prevention
						<TransformParams, <<
							<P, 1>
							<Type, TransformParams>
							<V, 1.2> constant
						>>>
						<Type, SigRef>
					>>
					]>

				parse the xref tables across all incremental updates.
				Detect and classify new or modified objects added after the signed byte range.
	*/

	return 0, nil
}
