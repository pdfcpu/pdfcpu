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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

func validatePageBoundaries(xRefTable *model.XRefTable, d types.Dict, dictName string, vp *model.ViewerPreferences) error {
	validate := func(s string) bool {
		return types.MemberOf(s, []string{"MediaBox", "CropBox", "BleedBox", "TrimBox", "ArtBox"})
	}

	n, err := validateNameEntry(xRefTable, d, dictName, "ViewArea", OPTIONAL, model.V14, validate)
	if err != nil {
		return err
	}
	if n != nil {
		vp.ViewArea = model.PageBoundaryFor(n.String())
	}

	n, err = validateNameEntry(xRefTable, d, dictName, "PrintArea", OPTIONAL, model.V14, validate)
	if err != nil {
		return err
	}
	if n != nil {
		vp.PrintArea = model.PageBoundaryFor(n.String())
	}

	n, err = validateNameEntry(xRefTable, d, dictName, "ViewClip", OPTIONAL, model.V14, validate)
	if err != nil {
		return err
	}
	if n != nil {
		vp.ViewClip = model.PageBoundaryFor(n.String())
	}

	n, err = validateNameEntry(xRefTable, d, dictName, "PrintClip", OPTIONAL, model.V14, validate)
	if err != nil {
		return err
	}
	if n != nil {
		vp.PrintClip = model.PageBoundaryFor(n.String())
	}

	return nil
}

func validatePrintPageRange(xRefTable *model.XRefTable, d types.Dict, dictName string, vp *model.ViewerPreferences) error {
	validate := func(arr types.Array) bool {
		if len(arr) > 0 && len(arr)%2 > 0 {
			return false
		}
		for i := 0; i < len(arr); i += 2 {
			if arr[i].(types.Integer) >= arr[i+1].(types.Integer) {
				return false
			}
		}
		return true
	}

	arr, err := validateIntegerArrayEntry(xRefTable, d, dictName, "PrintPageRange", OPTIONAL, model.V17, validate)
	if err != nil {
		return err
	}

	if len(arr) > 0 {
		vp.PrintPageRange = arr
	}

	return nil
}

func validateEnforcePrintScaling(xRefTable *model.XRefTable, d types.Dict, dictName string, vp *model.ViewerPreferences) error {
	validate := func(arr types.Array) bool {
		if len(arr) != 1 {
			return false
		}
		return arr[0].String() == "PrintScaling"
	}

	arr, err := validateNameArrayEntry(xRefTable, d, dictName, "Enforce", OPTIONAL, model.V20, validate)
	if err != nil {
		return err
	}

	if len(arr) > 0 {
		if vp.PrintScaling != nil && *vp.PrintScaling == model.PrintScalingAppDefault {
			return errors.New("pdfcpu: viewpreference \"Enforce[\"PrintScaling\"]\" needs \"PrintScaling\" <> \"AppDefault\"")
		}
		vp.Enforce = types.NewNameArray("PrintScaling")
	}

	return nil
}

func validatePrinterPreferences(xRefTable *model.XRefTable, d types.Dict, dictName string, vp *model.ViewerPreferences) error {
	sinceVersion := model.V16
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}
	validate := func(s string) bool {
		return types.MemberOf(s, []string{"None", "AppDefault"})
	}
	n, err := validateNameEntry(xRefTable, d, dictName, "PrintScaling", OPTIONAL, sinceVersion, validate)
	if err != nil {
		return err
	}
	if n != nil {
		vp.PrintScaling = model.PrintScalingFor(n.String())
	}

	validate = func(s string) bool {
		return types.MemberOf(s, []string{"Simplex", "DuplexFlipShortEdge", "DuplexFlipLongEdge"})
	}
	n, err = validateNameEntry(xRefTable, d, dictName, "Duplex", OPTIONAL, model.V17, validate)
	if err != nil {
		return err
	}
	if n != nil {
		vp.Duplex = model.PaperHandlingFor(n.String())
	}

	vp.PickTrayByPDFSize, err = validateFlexBooleanEntry(xRefTable, d, dictName, "PickTrayByPDFSize", OPTIONAL, model.V17)
	if err != nil {
		return err
	}

	vp.NumCopies, err = validateIntegerEntry(xRefTable, d, dictName, "NumCopies", OPTIONAL, model.V17, func(i int) bool { return i >= 1 })
	if err != nil {
		return err
	}

	if err := validatePrintPageRange(xRefTable, d, dictName, vp); err != nil {
		return err
	}

	return validateEnforcePrintScaling(xRefTable, d, dictName, vp)
}

func validateViewerPreferencesFlags(xRefTable *model.XRefTable, d types.Dict, dictName string, vp *model.ViewerPreferences) error {
	var err error
	vp.HideToolbar, err = validateFlexBooleanEntry(xRefTable, d, dictName, "HideToolbar", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	vp.HideMenubar, err = validateFlexBooleanEntry(xRefTable, d, dictName, "HideMenubar", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	vp.HideWindowUI, err = validateFlexBooleanEntry(xRefTable, d, dictName, "HideWindowUI", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	vp.FitWindow, err = validateFlexBooleanEntry(xRefTable, d, dictName, "FitWindow", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	vp.CenterWindow, err = validateFlexBooleanEntry(xRefTable, d, dictName, "CenterWindow", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	sinceVersion := model.V14
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V10
	}
	vp.DisplayDocTitle, err = validateFlexBooleanEntry(xRefTable, d, dictName, "DisplayDocTitle", OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	return nil
}

func validateViewerPreferences(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	// => 12.2 Viewer Preferences

	dictName := "rootDict"

	d, err := validateDictEntry(xRefTable, rootDict, dictName, "ViewerPreferences", required, sinceVersion, nil)
	if err != nil || d == nil {
		return err
	}

	vp := model.ViewerPreferences{}
	xRefTable.ViewerPref = &vp

	dictName = "ViewerPreferences"

	if err := validateViewerPreferencesFlags(xRefTable, d, dictName, &vp); err != nil {
		return err
	}

	validate := func(s string) bool {
		return types.MemberOf(s, []string{"UseNone", "UseOutlines", "UseThumbs", "UseOC"})
	}
	n, err := validateNameEntry(xRefTable, d, dictName, "NonFullScreenPageMode", OPTIONAL, model.V10, validate)
	if err != nil {
		return err
	}
	if n != nil {
		vp.NonFullScreenPageMode = (*model.NonFullScreenPageMode)(model.PageModeFor(n.String()))
	}

	validate = func(s string) bool { return types.MemberOf(s, []string{"L2R", "R2L"}) }
	n, err = validateNameEntry(xRefTable, d, dictName, "Direction", OPTIONAL, model.V13, validate)
	if err != nil {
		return err
	}
	if n != nil {
		vp.Direction = model.DirectionFor(n.String())
	}

	if err := validatePageBoundaries(xRefTable, d, dictName, &vp); err != nil {
		return err
	}

	return validatePrinterPreferences(xRefTable, d, dictName, &vp)
}
