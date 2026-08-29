/*
Copyright 2023 The pdfcpu Authors.

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

package validate

import (
	"errors"
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func validateDirection(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName string, vp *model.ViewerPreferences) error {
	validate := func(s string) bool {
		return types.MemberOf(s, []string{"L2R", "R2L"})
	}

	n, err := validateNameEntry(xRefTable, d, ownerObjNr, dictName, "Direction", OPTIONAL, model.V13, validate)
	if err != nil {
		if xRefTable.ValidationMode == model.ValidationStrict {
			return fmt.Errorf("%s.Direction: %w", dictName, err)
		}
		s, err := validateStringEntry(xRefTable, d, ownerObjNr, dictName, "Direction", OPTIONAL, model.V13, validate)
		if err != nil {
			return fmt.Errorf("%s.Direction: %w", dictName, err)
		}
		if s != nil {
			vp.Direction = model.DirectionFor(*s)
		}
		return nil
	}

	if n != nil {
		vp.Direction = model.DirectionFor(n.String())
	}

	return nil
}

func validatePageBoundaries(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName string, vp *model.ViewerPreferences) error {
	validate := func(s string) bool {
		return types.MemberOf(s, []string{"MediaBox", "CropBox", "BleedBox", "TrimBox", "ArtBox"})
	}

	n, err := validateNameEntry(xRefTable, d, ownerObjNr, dictName, "ViewArea", OPTIONAL, model.V14, validate)
	if err != nil {
		return fmt.Errorf("%s.ViewArea: %w", dictName, err)
	}
	if n != nil {
		vp.ViewArea = model.PageBoundaryFor(n.String())
	}

	n, err = validateNameEntry(xRefTable, d, ownerObjNr, dictName, "PrintArea", OPTIONAL, model.V14, validate)
	if err != nil {
		return fmt.Errorf("%s.PrintArea: %w", dictName, err)
	}
	if n != nil {
		vp.PrintArea = model.PageBoundaryFor(n.String())
	}

	n, err = validateNameEntry(xRefTable, d, ownerObjNr, dictName, "ViewClip", OPTIONAL, model.V14, validate)
	if err != nil {
		return fmt.Errorf("%s.ViewClip: %w", dictName, err)
	}
	if n != nil {
		vp.ViewClip = model.PageBoundaryFor(n.String())
	}

	n, err = validateNameEntry(xRefTable, d, ownerObjNr, dictName, "PrintClip", OPTIONAL, model.V14, validate)
	if err != nil {
		return fmt.Errorf("%s.PrintClip: %w", dictName, err)
	}
	if n != nil {
		vp.PrintClip = model.PageBoundaryFor(n.String())
	}

	return nil
}

func validatePrintPageRange(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName string, vp *model.ViewerPreferences) error {
	arr, err := validateIntegerArrayEntry(
		xRefTable, d, ownerObjNr, dictName, "PrintPageRange", OPTIONAL, model.V17,
		func(a types.Array) bool { return len(a)%2 == 0 },
	)
	if err != nil {
		return fmt.Errorf("%s.PrintPageRange: %w", dictName, err)
	}
	if len(arr) == 0 {
		return nil
	}

	pageRange := make([]int, len(arr))
	for j, o := range arr {
		objNr := validationObjectNumber(ownerObjNr, o)
		i, err := xRefTable.DereferenceInteger(o)
		if err != nil || i == nil {
			if err == nil {
				err = fmt.Errorf("missing integer at index %d", j)
			}
			err = fmt.Errorf("%s.PrintPageRange[%d]: %w", dictName, j, err)
			return model.WithValidationErrorObject(err, objNr)
		}
		pageRange[j] = i.Value()
	}

	for j := 0; j < len(pageRange); j += 2 {
		if pageRange[j] >= pageRange[j+1] {
			err := fmt.Errorf("dict=%s entry=PrintPageRange invalid dict entry", dictName)
			err = model.WithValidationErrorObject(err, validationEntryObjectNumber(ownerObjNr, d, "PrintPageRange"))
			return fmt.Errorf("%s.PrintPageRange: %w", dictName, err)
		}
	}

	vp.PrintPageRange = pageRange

	return nil
}

func validateEnforcePrintScaling(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName string, vp *model.ViewerPreferences) error {
	validate := func(arr types.Array) bool {
		return len(arr) == 1
	}

	arr, err := validateNameArrayEntry(xRefTable, d, ownerObjNr, dictName, "Enforce", OPTIONAL, model.V20, validate)
	if err != nil {
		return fmt.Errorf("%s.Enforce: %w", dictName, err)
	}

	if len(arr) > 0 {
		objNr := validationObjectNumber(validationEntryObjectNumber(ownerObjNr, d, "Enforce"), arr[0])
		n, err := xRefTable.DereferenceName(arr[0], model.V20, func(s string) bool { return s == "PrintScaling" })
		if err != nil {
			err = fmt.Errorf("%s.Enforce[0]: %w", dictName, err)
			return model.WithValidationErrorObject(err, objNr)
		}
		if vp.PrintScaling != nil && *vp.PrintScaling == model.PrintScalingAppDefault {
			err := errors.New(`ViewerPreferences.Enforce: PrintScaling requires PrintScaling != "AppDefault"`)
			return model.WithValidationErrorObject(err, ownerObjNr)
		}
		vp.Enforce = []string{n.Value()}
	}

	return nil
}

func validatePrinterPreferences(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName string, vp *model.ViewerPreferences) error {
	sinceVersion := model.V16
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	validate := func(s string) bool {
		return types.MemberOf(s, []string{"None", "AppDefault"})
	}
	n, err := validateNameEntry(xRefTable, d, ownerObjNr, dictName, "PrintScaling", OPTIONAL, sinceVersion, validate)
	if err != nil {
		if xRefTable.ValidationMode == model.ValidationStrict {
			return fmt.Errorf("%s.PrintScaling: %w", dictName, err)
		}
		// Ignore in relaxed mode.
	}
	if n != nil {
		vp.PrintScaling = model.PrintScalingFor(n.String())
	}

	validate = func(s string) bool {
		return types.MemberOf(s, []string{"Simplex", "DuplexFlipShortEdge", "DuplexFlipLongEdge"})
	}
	sinceVersion = model.V17
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V14
	}
	n, err = validateNameEntry(xRefTable, d, ownerObjNr, dictName, "Duplex", OPTIONAL, sinceVersion, validate)
	if err != nil {
		return fmt.Errorf("%s.Duplex: %w", dictName, err)
	}
	if n != nil {
		vp.Duplex = model.PaperHandlingFor(n.String())
	}

	sinceVersion = model.V17
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V15
	}
	vp.PickTrayByPDFSize, err = validateFlexBooleanEntry(
		xRefTable, d, ownerObjNr, dictName, "PickTrayByPDFSize", OPTIONAL, sinceVersion,
	)
	if err != nil {
		return fmt.Errorf("%s.PickTrayByPDFSize: %w", dictName, err)
	}

	sinceVersion = model.V17
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V15
	}
	vp.NumCopies, err = validateIntegerEntry(
		xRefTable, d, ownerObjNr, dictName, "NumCopies", OPTIONAL, sinceVersion, func(i int) bool { return i >= 1 },
	)
	if err != nil {
		return fmt.Errorf("%s.NumCopies: %w", dictName, err)
	}

	if err := validatePrintPageRange(xRefTable, d, ownerObjNr, dictName, vp); err != nil {
		return err
	}

	return validateEnforcePrintScaling(xRefTable, d, ownerObjNr, dictName, vp)
}

func validateViewerPreferencesFlags(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName string, vp *model.ViewerPreferences) error {
	var err error
	vp.HideToolbar, err = validateFlexBooleanEntry(xRefTable, d, ownerObjNr, dictName, "HideToolbar", OPTIONAL, model.V10)
	if err != nil {
		return fmt.Errorf("%s.HideToolbar: %w", dictName, err)
	}

	vp.HideMenubar, err = validateFlexBooleanEntry(xRefTable, d, ownerObjNr, dictName, "HideMenubar", OPTIONAL, model.V10)
	if err != nil {
		return fmt.Errorf("%s.HideMenubar: %w", dictName, err)
	}

	vp.HideWindowUI, err = validateFlexBooleanEntry(xRefTable, d, ownerObjNr, dictName, "HideWindowUI", OPTIONAL, model.V10)
	if err != nil {
		return fmt.Errorf("%s.HideWindowUI: %w", dictName, err)
	}

	vp.FitWindow, err = validateFlexBooleanEntry(xRefTable, d, ownerObjNr, dictName, "FitWindow", OPTIONAL, model.V10)
	if err != nil {
		return fmt.Errorf("%s.FitWindow: %w", dictName, err)
	}

	vp.CenterWindow, err = validateFlexBooleanEntry(xRefTable, d, ownerObjNr, dictName, "CenterWindow", OPTIONAL, model.V10)
	if err != nil {
		return fmt.Errorf("%s.CenterWindow: %w", dictName, err)
	}

	sinceVersion := model.V14
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V10
	}
	vp.DisplayDocTitle, err = validateFlexBooleanEntry(
		xRefTable, d, ownerObjNr, dictName, "DisplayDocTitle", OPTIONAL, sinceVersion,
	)
	if err != nil {
		return fmt.Errorf("%s.DisplayDocTitle: %w", dictName, err)
	}

	return nil
}

func digestViewerPreferencesArray(xRefTable *model.XRefTable, arr types.Array, ownerObjNr int) error {
	model.ShowDigestedSpecViolation("viewer preferences array instead of dict")
	for i, v := range arr {
		objNr := validationObjectNumber(ownerObjNr, v)
		o, err := xRefTable.Dereference(v)
		if err != nil {
			err = fmt.Errorf("rootDict.ViewerPreferences[%d]: %w", i, err)
			return model.WithValidationErrorObject(err, objNr)
		}
		if _, ok := o.(types.Name); !ok {
			err := fmt.Errorf("rootDict.ViewerPreferences[%d]: expected name, got %T", i, o)
			return model.WithValidationErrorObject(err, objNr)
		}
	}
	return nil
}

func validateViewerPreferences(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 12.2 Viewer Preferences

	dictName := "rootDict"
	rawPreferences, _ := rootDict.Find("ViewerPreferences")
	preferencesObjNr := validationObjectNumber(validationRootObjectNumber(xRefTable), rawPreferences)
	d, err := validateDictEntry(
		xRefTable, rootDict, validationRootObjectNumber(xRefTable), dictName, "ViewerPreferences", required, sinceVersion, nil,
	)
	if err != nil {
		if xRefTable.ValidationMode == model.ValidationStrict {
			return fmt.Errorf("rootDict.ViewerPreferences: %w", err)
		}
		arr, err := validateArrayEntry(
			xRefTable, rootDict, validationRootObjectNumber(xRefTable), dictName, "ViewerPreferences", required, sinceVersion, nil,
		)
		if err != nil || len(arr) == 0 {
			if err != nil {
				return fmt.Errorf("rootDict.ViewerPreferences: %w", err)
			}
			return nil
		}
		// For an out-of-spec viewer preferences array, we assume it only contains boolean flags set to true.
		return digestViewerPreferencesArray(xRefTable, arr, preferencesObjNr)
	}

	if d == nil {
		return nil
	}

	vp := model.ViewerPreferences{}
	xRefTable.ViewerPref = &vp

	dictName = "ViewerPreferences"

	if err := validateViewerPreferencesFlags(xRefTable, d, preferencesObjNr, dictName, &vp); err != nil {
		return err
	}

	vv := []string{"UseNone", "UseOutlines", "UseThumbs", "UseOC"}
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		vv = append(vv, "PageOnly")
	}
	validate := func(s string) bool {
		return types.MemberOf(s, vv)
	}
	n, err := validateNameEntry(
		xRefTable, d, preferencesObjNr, dictName, "NonFullScreenPageMode", OPTIONAL, model.V10, validate,
	)
	if err != nil {
		return fmt.Errorf("%s.NonFullScreenPageMode: %w", dictName, err)
	}
	if n != nil {
		vp.NonFullScreenPageMode = (*model.NonFullScreenPageMode)(model.PageModeFor(n.String()))
	}

	if err := validateDirection(xRefTable, d, preferencesObjNr, dictName, &vp); err != nil {
		return err
	}

	if err := validatePageBoundaries(xRefTable, d, preferencesObjNr, dictName, &vp); err != nil {
		return err
	}

	return validatePrinterPreferences(xRefTable, d, preferencesObjNr, dictName, &vp)
}
