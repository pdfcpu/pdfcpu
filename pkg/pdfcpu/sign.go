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
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/sign"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type signatureValidationHandler func(
	io.ReaderAt,
	types.Dict,
	bool,
	bool,
	bool,
	int,
	*x509.CertPool,
	*model.SignatureValidationResult,
	*model.Context,
) error

// ValidateSignatures reports observed signature, certificate, timestamp and
// revocation evidence together with a local assessment.
func ValidateSignatures(ra io.ReaderAt, ctx *model.Context, all bool) ([]*model.SignatureValidationResult, error) {
	var results []*model.SignatureValidationResult

	if ctx.URSignature != nil {
		svr, err := validateURSignature(ctx.URSignature, ctx.URSignatureIncrement, ctx, ra)
		if err != nil {
			return nil, fmt.Errorf("usage rights signature: %w", err)
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
		for _, sig := range orderedSignatures(ctx.Signatures[inc]) {

			if i > 0 && sig.Type == model.SigTypeDTS {
				continue
			}

			svr, err := validateSignature(sig, ctx, ra, first, all, inc)
			if err != nil {
				return nil, fmt.Errorf("signature obj#%d: %w", sig.ObjNr, err)
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

func orderedSignatures(signatures map[int]model.Signature) []model.Signature {
	objNrs := make([]int, 0, len(signatures))
	for objNr := range signatures {
		objNrs = append(objNrs, objNr)
	}
	sort.Ints(objNrs)

	ordered := make([]model.Signature, 0, len(objNrs))
	for _, objNr := range objNrs {
		ordered = append(ordered, signatures[objNr])
	}
	return ordered
}

func checkForAbortAfterFirst(first bool, svr *model.SignatureValidationResult, ctx *model.Context) bool {
	if first {
		if ctx.CertifiedSigObjNr == 0 || (svr.Certified() && svr.Permissions() != model.CertifiedSigPermNoChangesAllowed) {
			return true
		}
	}
	return svr.Certified()
}

func validateURSignature(sigDict types.Dict, increment int, ctx *model.Context, ra io.ReaderAt) (*model.SignatureValidationResult, error) {
	sig := model.Signature{Type: model.SigTypeUR, Visible: false, Signed: true}
	result := model.SignatureValidationResult{Signature: sig}

	result.Status = model.SignatureStatusUnknown
	result.Reason = model.SignatureReasonUnknown
	result.DocModified = model.Unknown

	result.Details = model.SignatureDetails{}
	result.Details.SignerIdentity = "Unknown"

	signatureDetails(sigDict, ctx, &result)

	subFilter, f, ok := signatureSubFilter(sigDict, true, &result)
	if !ok {
		return &result, nil
	}

	if !recordSignedRevisionBoundaryEvidence(sigDict, ctx, increment, false, &result) {
		return &result, nil
	}

	if err := f(
		ra,
		sigDict,
		false,
		false,
		true,
		0,
		userCertificatePool(),
		&result,
		ctx,
	); err != nil {
		return &result, fmt.Errorf("signature dict entry SubFilter %s: validate: %w", subFilter, err)
	}
	applyHistoricalRevisionReporting(increment, sig.Type, &result)
	return &result, nil
}

func validateSignature(sig model.Signature, ctx *model.Context, ra io.ReaderAt, first, all bool, increment int) (*model.SignatureValidationResult, error) {
	sigField, err := ctx.DereferenceDict(*types.NewIndirectRef(sig.ObjNr, 0))
	if err != nil {
		return nil, fmt.Errorf("signature field dict: dereference: %w", err)
	}

	result := model.SignatureValidationResult{Signature: sig}

	result.Status = model.SignatureStatusUnknown
	result.Reason = model.SignatureReasonUnknown
	result.DocModified = model.Unknown

	result.Details = model.SignatureDetails{}
	result.Details.SignerIdentity = "Unknown"

	if sigField == nil {
		return nil, errors.New("signature field dict: missing")
	}

	fieldDetails(sigField, &result)

	indRef := sigField.IndirectRefEntry("V")
	if indRef == nil {
		return nil, errors.New("signature field dict entry V: missing signature dict reference")
	}

	sigDict, err := ctx.DereferenceDict(*indRef)
	if err != nil {
		return nil, fmt.Errorf("signature dict obj#%d: dereference: %w", indRef.ObjectNumber.Value(), err)
	}
	if sigDict == nil {
		return nil, fmt.Errorf("signature dict obj#%d: missing dictionary", indRef.ObjectNumber.Value())
	}

	subFilter, f, ok := signatureSubFilter(sigDict, false, &result)

	result.Signature.Certified = indRef.ObjectNumber.Value() == ctx.CertifiedSigObjNr
	if first && ctx.CertifiedSigObjNr == 0 {
		result.Signature.Authoritative = true
	}

	signatureDetails(sigDict, ctx, &result)

	if !ok {
		if _, err := detectPermissions(sigDict, ctx, &result); err != nil {
			return nil, fmt.Errorf("detect permissions: %w", err)
		}
		return &result, nil
	}

	if !recordSignedRevisionBoundaryEvidence(sigDict, ctx, increment, sig.Type == model.SigTypeDTS, &result) {
		return &result, nil
	}

	perms, err := detectPermissions(sigDict, ctx, &result)
	if err != nil {
		return nil, fmt.Errorf("detect permissions: %w", err)
	}

	if err := f(
		ra,
		sigDict,
		result.Signature.Certified,
		result.Signature.Authoritative,
		all,
		perms,
		userCertificatePool(),
		&result,
		ctx,
	); err != nil {
		return &result, fmt.Errorf("signature dict entry SubFilter %s: validate: %w", subFilter, err)
	}
	applyHistoricalRevisionReporting(increment, sig.Type, &result)
	return &result, nil
}

func signatureSubFilter(sigDict types.Dict, usageRights bool, result *model.SignatureValidationResult) (string, signatureValidationHandler, bool) {
	obj, found := sigDict.Find("SubFilter")
	if !found {
		recordSubFilterProblem(
			result,
			model.SignatureReasonMalformed,
			"signature dict entry SubFilter: malformed: missing",
		)
		return "", nil, false
	}
	name, ok := obj.(types.Name)
	if !ok {
		recordSubFilterProblem(
			result,
			model.SignatureReasonMalformed,
			fmt.Sprintf("signature dict entry SubFilter: malformed: expected name, got %T", obj),
		)
		return "", nil, false
	}

	subFilter := name.Value()
	result.Details.SubFilter = subFilter
	f := sigHandler(subFilter)
	if f == nil || usageRights && subFilter == "ETSI.RFC3161" {
		recordSubFilterProblem(
			result,
			model.SignatureReasonUnsupported,
			fmt.Sprintf("signature dict entry SubFilter: unsupported: value %s", subFilter),
		)
		return "", nil, false
	}
	return subFilter, f, true
}

func recordSubFilterProblem(result *model.SignatureValidationResult, reason model.SignatureReason, problem string) {
	result.Status = model.SignatureStatusUnknown
	result.Reason = reason
	result.DocModified = model.Unknown
	result.AddProblem(problem)
}

func applyHistoricalRevisionReporting(increment int, signatureType int, result *model.SignatureValidationResult) {
	if increment <= 0 || signatureType == model.SigTypeDTS || result == nil {
		return
	}
	if result.DocModified == model.False {
		result.DocModified = model.Unknown
	}
	if result.Reason == model.SignatureReasonDocNotModified {
		result.Reason = model.SignatureReasonUnknown
	}
}

type signedRevisionBoundaryEvidence struct {
	currentFileSize   int64
	signedRevisionEnd int64
	increment         int
	currentRevision   bool
	documentTimestamp bool
}

func recordSignedRevisionBoundaryEvidence(sigDict types.Dict, ctx *model.Context, increment int, documentTimestamp bool, result *model.SignatureValidationResult) bool {
	evidence, ok := collectSignedRevisionBoundaryEvidence(sigDict, ctx, increment, documentTimestamp)
	if !ok || !evidence.currentRevision || evidence.signedRevisionEnd == evidence.currentFileSize {
		return true
	}

	prefix := "read signed data"
	if evidence.documentTimestamp {
		prefix = "SubFilter ETSI.RFC3161: read signed data"
	}
	result.Reason = model.SignatureReasonMalformed
	result.AddProblem(fmt.Sprintf(
		"%s: malformed signature ByteRange: signed revision boundary mismatch: range ends at %d, current file size is %d, increment %d",
		prefix,
		evidence.signedRevisionEnd,
		evidence.currentFileSize,
		evidence.increment,
	))
	return false
}

func collectSignedRevisionBoundaryEvidence(sigDict types.Dict, ctx *model.Context, increment int, documentTimestamp bool) (signedRevisionBoundaryEvidence, bool) {
	if ctx == nil || ctx.Read == nil || ctx.Read.FileSize < 0 {
		return signedRevisionBoundaryEvidence{}, false
	}
	arr := sigDict.ArrayEntry("ByteRange")
	if len(arr) != 4 {
		return signedRevisionBoundaryEvidence{}, false
	}
	off, okOff := arr[2].(types.Integer)
	size, okSize := arr[3].(types.Integer)
	if !okOff || !okSize {
		return signedRevisionBoundaryEvidence{}, false
	}
	offValue := int64(off.Value())
	sizeValue := int64(size.Value())
	if offValue < 0 || sizeValue < 0 || offValue > math.MaxInt64-sizeValue {
		return signedRevisionBoundaryEvidence{}, false
	}
	return signedRevisionBoundaryEvidence{
		currentFileSize:   ctx.Read.FileSize,
		signedRevisionEnd: offValue + sizeValue,
		increment:         increment,
		currentRevision:   increment == 0 || documentTimestamp,
		documentTimestamp: documentTimestamp,
	}, true
}

func sigHandler(subFilter string) signatureValidationHandler {
	switch subFilter {
	case "adbe.x509.rsa_sha1": // deprecated as of PDF 2.0
		return validateX509RSASHA1Signature
	case "adbe.pkcs7.sha1": // deprecated as of PDF 2.0
		return validatePKCS7Signatures
	case "adbe.pkcs7.detached":
		return validatePKCS7Signatures
	case "ETSI.CAdES.detached":
		return validatePKCS7Signatures
	case "ETSI.RFC3161":
		return validateDTS
	}

	return nil
}

func validateX509RSASHA1Signature(
	ra io.ReaderAt,
	sigDict types.Dict,
	certified bool,
	authoritative bool,
	validateAll bool,
	perms int,
	rootCerts *x509.CertPool,
	result *model.SignatureValidationResult,
	ctx *model.Context,
) error {
	return sign.ValidateX509RSASHA1Signature(
		ra,
		sigDict,
		certified,
		authoritative,
		validateAll,
		perms,
		rootCerts,
		result,
		ctx,
	)
}

func validatePKCS7Signatures(
	ra io.ReaderAt,
	sigDict types.Dict,
	certified bool,
	authoritative bool,
	validateAll bool,
	perms int,
	rootCerts *x509.CertPool,
	result *model.SignatureValidationResult,
	ctx *model.Context,
) error {
	return sign.ValidatePKCS7Signatures(
		ra,
		sigDict,
		certified,
		authoritative,
		validateAll,
		perms,
		rootCerts,
		result,
		ctx,
	)
}

func validateDTS(
	ra io.ReaderAt,
	sigDict types.Dict,
	certified bool,
	authoritative bool,
	validateAll bool,
	perms int,
	rootCerts *x509.CertPool,
	result *model.SignatureValidationResult,
	ctx *model.Context,
) error {
	return sign.ValidateDTS(
		ra,
		sigDict,
		certified,
		authoritative,
		validateAll,
		perms,
		rootCerts,
		result,
		ctx,
	)
}

func fieldDetails(sigField types.Dict, result *model.SignatureValidationResult) {
	sl := sigField.StringLiteralEntry("T")
	if sl == nil {
		return
	}
	s, err := types.StringLiteralToString(*sl)
	if err != nil {
		result.AddProblem(fmt.Sprintf("signature field dict entry T: %v", err))
		return
	}
	result.Details.FieldName = strings.TrimSpace(s)
}

func signatureDetails(sigDict types.Dict, ctx *model.Context, result *model.SignatureValidationResult) {
	if sl := sigDict.StringLiteralEntry("Name"); sl != nil {
		s, err := types.StringLiteralToString(*sl)
		if err != nil {
			result.AddProblem(fmt.Sprintf("signature dict entry Name: %v", err))
		} else {
			result.Details.SignerName = strings.TrimSpace(s)
		}
	}

	if sl := sigDict.StringLiteralEntry("ContactInfo"); sl != nil {
		s, err := types.StringLiteralToString(*sl)
		if err != nil {
			result.AddProblem(fmt.Sprintf("signature dict entry ContactInfo: %v", err))
		} else {
			result.Details.ContactInfo = strings.TrimSpace(s)
		}
	}

	if sl := sigDict.StringLiteralEntry("Location"); sl != nil {
		s, err := types.StringLiteralToString(*sl)
		if err != nil {
			result.AddProblem(fmt.Sprintf("signature dict entry Location: %v", err))
		} else {
			result.Details.Location = strings.TrimSpace(s)
		}
	}

	if sl := sigDict.StringLiteralEntry("Reason"); sl != nil {
		s, err := types.StringLiteralToString(*sl)
		if err != nil {
			result.AddProblem(fmt.Sprintf("signature dict entry Reason: %v", err))
		} else {
			result.Details.Reason = strings.TrimSpace(s)
		}
	}

	if o, ok := sigDict.Find("M"); ok {
		// informational (cannot be relied upon for long term validation)
		s, err := ctx.DereferenceStringOrHexLiteral(o, model.V10, nil)
		if err != nil {
			result.AddProblem(fmt.Sprintf("signature dict entry M: %v", err))
		} else if t, ok := types.DateTime(s, ctx.XRefTable.ValidationMode == model.ValidationRelaxed); s != "" && ok {
			result.Details.SigningTime = t
		} else if s != "" {
			result.AddProblem("signature dict entry M: invalid signing time")
		}
	}
}

func detectPermissions(sigDict types.Dict, ctx *model.Context, result *model.SignatureValidationResult) (int, error) {
	arr := signatureReferences(sigDict, ctx, result)
	if len(arr) == 0 {
		return 0, nil
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

	for refIndex, obj := range arr {
		d, ok := signatureReferenceDict(obj, refIndex, ctx, result)
		if !ok {
			continue
		}
		if tm := d.NameEntry("TransformMethod"); tm == nil || *tm != "DocMDP" {
			continue
		}
		if perm, ok := docMDPPermission(d, refIndex, ctx, result); ok {
			return perm, nil
		}
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

func signatureReferences(sigDict types.Dict, ctx *model.Context, result *model.SignatureValidationResult) types.Array {
	o, found := sigDict.Find("Reference")
	if !found {
		return nil
	}

	arr, err := ctx.DereferenceArray(o)
	if err != nil {
		result.AddProblem(fmt.Sprintf("signature dict entry Reference: dereference array: %v", err))
		return nil
	}
	if len(arr) == 0 {
		result.AddProblem("signature dict entry Reference: empty array")
		return nil
	}
	return arr
}

func signatureReferenceDict(obj types.Object, refIndex int, ctx *model.Context, result *model.SignatureValidationResult) (types.Dict, bool) {
	d, err := ctx.DereferenceDict(obj)
	if err != nil {
		result.AddProblem(fmt.Sprintf(
			"signature dict entry Reference, reference index %d: dereference dictionary: %v",
			refIndex+1,
			err,
		))
		return nil, false
	}
	if d == nil {
		result.AddProblem(fmt.Sprintf(
			"signature dict entry Reference, reference index %d: missing dictionary",
			refIndex+1,
		))
		return nil, false
	}
	return d, true
}

func docMDPPermission(refDict types.Dict, refIndex int, ctx *model.Context, result *model.SignatureValidationResult) (int, bool) {
	o, found := refDict.Find("TransformParams")
	if !found {
		result.AddProblem(fmt.Sprintf(
			"signature dict entry Reference, reference index %d, signature reference dict entry TransformParams: missing",
			refIndex+1,
		))
		return 0, false
	}

	params, err := ctx.DereferenceDict(o)
	if err != nil {
		result.AddProblem(fmt.Sprintf(
			"signature dict entry Reference, reference index %d, signature reference dict entry TransformParams: dereference dictionary: %v",
			refIndex+1,
			err,
		))
		return 0, false
	}
	if params == nil {
		result.AddProblem(fmt.Sprintf(
			"signature dict entry Reference, reference index %d, signature reference dict entry TransformParams: missing dictionary",
			refIndex+1,
		))
		return 0, false
	}
	if typ := params.Type(); typ == nil || *typ != "TransformParams" {
		result.AddProblem(fmt.Sprintf(
			"signature dict entry Reference, reference index %d, TransformParams dict entry Type: expected TransformParams",
			refIndex+1,
		))
		return 0, false
	}

	return permissionEntry(params, refIndex, ctx, result)
}

func permissionEntry(params types.Dict, refIndex int, ctx *model.Context, result *model.SignatureValidationResult) (int, bool) {
	o, found := params.Find("P")
	if !found {
		return 2, true
	}

	p, err := ctx.DereferenceInteger(o)
	if err != nil {
		result.AddProblem(fmt.Sprintf(
			"signature dict entry Reference, reference index %d, TransformParams dict entry P: dereference integer: %v",
			refIndex+1,
			err,
		))
		return 0, false
	}
	if p == nil {
		result.AddProblem(fmt.Sprintf(
			"signature dict entry Reference, reference index %d, TransformParams dict entry P: missing integer",
			refIndex+1,
		))
		return 0, false
	}

	perm := p.Value()
	if perm < 1 || perm > 3 {
		result.AddProblem(fmt.Sprintf(
			"signature dict entry Reference, reference index %d, TransformParams dict entry P: invalid DocMDP permission: %d",
			refIndex+1,
			perm,
		))
		return 0, false
	}
	return perm, true
}
