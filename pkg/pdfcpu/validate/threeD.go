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

package validate

import (
	"context"
	"errors"
	"fmt"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type threeDViewSelectorKind int

const (
	threeDStreamViewSelector threeDViewSelectorKind = iota
	threeDAnnotationViewSelector
	threeDActionViewSelector
)

func threeDV17SinceVersion(xRefTable *model.XRefTable) model.Version {
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		return model.V16
	}
	return model.V17
}

func showRelaxed3DVersionViolation(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string) {
	o, found := d.Find(entryName)
	if !found || o == nil || xRefTable.ValidationMode != model.ValidationRelaxed || xRefTable.Version() >= model.V17 {
		return
	}
	showDigestedVersionViolation(xRefTable, fmt.Sprintf("dict=%s entry=%s", dictName, entryName))
}

func validate3DAnnotationAppearance(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName string) error {
	appearance, err := validateDictEntry(xRefTable, d, ownerObjNr, dictName, "AP", OPTIONAL, model.V16, nil)
	if err != nil {
		return err
	}
	entryName := "AP"
	if appearance != nil {
		ownerObjNr = validationEntryObjectNumber(ownerObjNr, d, "AP")
		normal, err := validateEntry(xRefTable, appearance, ownerObjNr, "appearanceDict", "N", OPTIONAL, model.V16)
		if err != nil || normal != nil {
			return err
		}
		entryName = "AP.N"
	}
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		model.ShowSkipped(fmt.Sprintf("3D annotation: missing %q", entryName))
		return nil
	}
	err = missingRequiredEntryError(dictName, entryName, "add a normal appearance or validate in relaxed mode")
	return model.WithValidationErrorObject(err, ownerObjNr)
}

func validate3DActivationDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) error {
	// see 13.6.2 and table 299
	dictName := "3DActivationDict"
	entries := []struct {
		name   string
		values map[string]bool
	}{
		{name: "A", values: map[string]bool{"PO": true, "PV": true, "XA": true}},
		{name: "AIS", values: map[string]bool{"I": true, "L": true}},
		{name: "D", values: map[string]bool{"PC": true, "PI": true, "XD": true}},
		{name: "DIS", values: map[string]bool{"U": true, "I": true, "L": true}},
	}
	for _, entry := range entries {
		if _, err := validateNameEntry(
			xRefTable, d, ownerObjNr, dictName, entry.name, OPTIONAL, model.V16,
			func(s string) bool { return entry.values[s] },
		); err != nil {
			return err
		}
	}
	for _, entryName := range []string{"TB", "NP"} {
		if _, err := validateBooleanEntry(
			xRefTable, d, ownerObjNr, dictName, entryName, OPTIONAL, threeDV17SinceVersion(xRefTable), nil,
		); err != nil {
			return err
		}
		showRelaxed3DVersionViolation(xRefTable, d, dictName, entryName)
	}
	return nil
}

func validate3DPathMember(c context.Context, xRefTable *model.XRefTable, raw types.Object, objNr int) error {
	if err := contextutil.Check(c); err != nil {
		return model.WithValidationErrorObject(err, objNr)
	}
	o, err := xRefTable.Dereference(raw)
	if err != nil {
		return model.WithValidationErrorObject(err, objNr)
	}
	if o == nil {
		return model.WithValidationErrorObject(errors.New("expected text string"), objNr)
	}
	_, err = xRefTable.DereferenceStringOrHexLiteral(o, model.V16, nil)
	return model.WithValidationErrorObject(err, objNr)
}

func validate3DViewPath(c context.Context, xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) error {
	const dictName = "3DViewDict"
	raw, found := d.Find("U3DPath")
	if !found {
		return missingRequiredEntryError(dictName, "U3DPath", "required when MS is U3D")
	}
	o, err := validateEntry(xRefTable, d, ownerObjNr, dictName, "U3DPath", REQUIRED, model.V16)
	if err != nil {
		return err
	}
	switch o := o.(type) {
	case types.StringLiteral, types.HexLiteral:
		_, err = xRefTable.DereferenceStringOrHexLiteral(raw, model.V16, nil)
		return err
	case types.Array:
		if len(o) == 0 {
			return fmt.Errorf("dict=%s entry=U3DPath: expected at least one string", dictName)
		}
		arrayObjNr := validationEntryObjectNumber(ownerObjNr, d, "U3DPath")
		for i, value := range o {
			if err = validate3DPathMember(c, xRefTable, value, validationObjectNumber(arrayObjNr, value)); err != nil {
				return fmt.Errorf("dict=%s entry=U3DPath[%d]: %w", dictName, i, err)
			}
		}
		return nil
	default:
		return fmt.Errorf("dict=%s entry=U3DPath: invalid type %T", dictName, o)
	}
}

func validate3DDictArrayEntry(c context.Context, xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, sinceVersion model.Version, validate func(context.Context, *model.XRefTable, types.Dict, int) error) error {
	a, err := validateArrayEntry(xRefTable, d, ownerObjNr, dictName, entryName, OPTIONAL, sinceVersion, nil)
	if err != nil || a == nil {
		return err
	}
	entryObjNr := validationEntryObjectNumber(ownerObjNr, d, entryName)
	for i, raw := range a {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		objNr := validationObjectNumber(entryObjNr, raw)
		entry, err := xRefTable.DereferenceDict(raw)
		if err != nil {
			return model.WithValidationErrorObject(err, objNr)
		}
		if entry == nil {
			return model.WithValidationErrorObject(fmt.Errorf("dict=%s entry=%s[%d]: expected dictionary", dictName, entryName, i), objNr)
		}
		if validate != nil {
			if err = validate(c, xRefTable, entry, objNr); err != nil {
				return err
			}
		}
	}
	return nil
}

func validate3DViewNestedEntries(c context.Context, xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) error {
	const dictName = "3DViewDict"
	for _, entryName := range []string{"P", "BG"} {
		if _, err := validateDictEntry(xRefTable, d, ownerObjNr, dictName, entryName, OPTIONAL, model.V16, nil); err != nil {
			return err
		}
	}
	for _, entryName := range []string{"RM", "LS"} {
		if _, err := validateDictEntry(
			xRefTable, d, ownerObjNr, dictName, entryName, OPTIONAL, threeDV17SinceVersion(xRefTable), nil,
		); err != nil {
			return err
		}
		showRelaxed3DVersionViolation(xRefTable, d, dictName, entryName)
	}
	if _, err := validateStreamDictEntry(xRefTable, d, ownerObjNr, dictName, "O", OPTIONAL, model.V16, nil); err != nil {
		return err
	}
	for _, entryName := range []string{"SA", "NA"} {
		if err := validate3DDictArrayEntry(
			c, xRefTable, d, ownerObjNr, dictName, entryName, threeDV17SinceVersion(xRefTable), nil,
		); err != nil {
			return err
		}
		showRelaxed3DVersionViolation(xRefTable, d, dictName, entryName)
	}
	if _, err := validateBooleanEntry(
		xRefTable, d, ownerObjNr, dictName, "NR", OPTIONAL, threeDV17SinceVersion(xRefTable), nil,
	); err != nil {
		return err
	}
	showRelaxed3DVersionViolation(xRefTable, d, dictName, "NR")
	return nil
}

func validate3DViewDict(c context.Context, xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) error {
	// see 13.6.4 and table 304
	if err := contextutil.Check(c); err != nil {
		return err
	}
	const dictName = "3DViewDict"
	if _, err := validateNameEntry(xRefTable, d, ownerObjNr, dictName, "Type", OPTIONAL, model.V16, func(s string) bool {
		return s == "3DView"
	}); err != nil {
		return err
	}
	if _, err := validateStringEntry(xRefTable, d, ownerObjNr, dictName, "XN", REQUIRED, model.V16, nil); err != nil {
		return err
	}
	if _, err := validateStringEntry(xRefTable, d, ownerObjNr, dictName, "IN", OPTIONAL, model.V16, nil); err != nil {
		return err
	}
	ms, err := validateNameEntry(xRefTable, d, ownerObjNr, dictName, "MS", OPTIONAL, model.V16, func(s string) bool {
		return s == "M" || s == "U3D"
	})
	if err != nil {
		return err
	}
	if ms != nil && ms.Value() == "M" {
		if _, err = validateNumberArrayEntry(xRefTable, d, ownerObjNr, dictName, "C2W", REQUIRED, model.V16, func(a types.Array) bool {
			return len(a) == 12
		}); err != nil {
			return err
		}
	}
	if ms != nil && ms.Value() == "U3D" {
		if err = validate3DViewPath(c, xRefTable, d, ownerObjNr); err != nil {
			return err
		}
	}
	if ms != nil {
		if _, err = validateNumberEntry(xRefTable, d, ownerObjNr, dictName, "CO", OPTIONAL, model.V16, func(f float64) bool {
			return f >= 0
		}); err != nil {
			return err
		}
	}
	return validate3DViewNestedEntries(c, xRefTable, d, ownerObjNr)
}

func valid3DViewSelectorName(name types.Name, kind threeDViewSelectorKind) bool {
	switch kind {
	case threeDStreamViewSelector:
		return name == "F" || name == "L"
	case threeDAnnotationViewSelector:
		return name == "F" || name == "L" || name == "D"
	case threeDActionViewSelector:
		return name == "F" || name == "L" || name == "D" || name == "N" || name == "P"
	}
	return false
}

func validate3DViewSelector(c context.Context, xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, kind threeDViewSelectorKind) error {
	raw, _ := d.Find(entryName)
	o, err := validateEntry(xRefTable, d, ownerObjNr, dictName, entryName, required, model.V16)
	if err != nil || o == nil {
		return err
	}
	objNr := validationObjectNumber(ownerObjNr, raw)
	switch o := o.(type) {
	case types.Dict:
		return validate3DViewDict(c, xRefTable, o, objNr)
	case types.Integer:
		if o >= 0 {
			return nil
		}
		err = fmt.Errorf("dict=%s entry=%s: negative 3D view index %d", dictName, entryName, o)
		return model.WithValidationErrorObject(err, objNr)
	case types.StringLiteral, types.HexLiteral:
		_, err = xRefTable.DereferenceStringOrHexLiteral(raw, model.V16, nil)
		return model.WithValidationErrorObject(err, objNr)
	case types.Name:
		if kind == threeDStreamViewSelector && o == "DEFAULT" && xRefTable.ValidationMode == model.ValidationRelaxed {
			model.ShowSkipped(fmt.Sprintf(
				`dict=%s entry=%s: legacy 3D view selector name %q treated as absent`, dictName, entryName, o,
			))
			return nil
		}
		if valid3DViewSelectorName(o, kind) {
			return nil
		}
		err = fmt.Errorf("dict=%s entry=%s: invalid 3D view selector name %q", dictName, entryName, o)
		return model.WithValidationErrorObject(err, objNr)
	}
	err = fmt.Errorf("dict=%s entry=%s: invalid 3D view selector type %T", dictName, entryName, o)
	return model.WithValidationErrorObject(err, objNr)
}

func validate3DAnimationStyleDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) error {
	const dictName = "3DAnimationStyleDict"
	sinceVersion := threeDV17SinceVersion(xRefTable)
	if _, err := validateNameEntry(xRefTable, d, ownerObjNr, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool {
		return s == "3DAnimationStyle"
	}); err != nil {
		return err
	}
	subtype, err := validateNameEntry(xRefTable, d, ownerObjNr, dictName, "Subtype", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	// See table 301: play count and time multiplier are ignored for None.
	if subtype == nil || subtype.Value() == "None" {
		return nil
	}
	if subtype.Value() != "Linear" && subtype.Value() != "Oscillating" {
		model.ShowSkipped(fmt.Sprintf("3D animation style subtype %q treated as None", subtype.Value()))
		return nil
	}
	if _, err = validateIntegerEntry(xRefTable, d, ownerObjNr, dictName, "PC", OPTIONAL, sinceVersion, nil); err != nil {
		return err
	}
	_, err = validateNumberEntry(xRefTable, d, ownerObjNr, dictName, "TM", OPTIONAL, sinceVersion, func(f float64) bool {
		return f > 0
	})
	return err
}

func validate3DResources(c context.Context, xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) error {
	raw, found := d.Find("Resources")
	if !found {
		return nil
	}
	objNr := validationObjectNumber(ownerObjNr, raw)
	resources, err := validateDictEntry(xRefTable, d, ownerObjNr, "3DStreamDict", "Resources", OPTIONAL, model.V16, nil)
	if err != nil || resources == nil {
		return err
	}
	_, _, _, err = validateNameTree(c, xRefTable, "3DResources", resources, objNr, true, raw)
	return err
}

func validate3DStreamSubtype(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) error {
	subtype, err := validateNameEntry(xRefTable, d, ownerObjNr, "3DStreamDict", "Subtype", REQUIRED, model.V16, nil)
	if err != nil {
		return err
	}
	valid := subtype.Value() == "U3D" || subtype.Value() == "PRC" && xRefTable.Version() >= model.V20
	if valid {
		return nil
	}
	err = fmt.Errorf("dict=3DStreamDict entry=Subtype: unsupported 3D format %q", subtype.Value())
	if xRefTable.ValidationMode == model.ValidationStrict {
		return model.WithValidationErrorObject(err, validationEntryObjectNumber(ownerObjNr, d, "Subtype"))
	}
	model.ShowSkipped(err.Error())
	return nil
}

func collect3DOnInstantiate(xRefTable *model.XRefTable, d types.Dict, streamObjNr, ownerObjNr int) {
	addActiveContentNotice(xRefTable, activeContentFinding{
		kind:       activeContentJavaScript,
		owner:      activeContentOwnerAnnotation,
		source:     activeContentSource3DOnInstantiate,
		pageNr:     xRefTable.CurPage,
		objNr:      validationEntryObjectNumber(streamObjNr, d, "OnInstantiate"),
		ownerObjNr: ownerObjNr,
	})
}

func validate3DStreamDict(c context.Context, xRefTable *model.XRefTable, sd types.StreamDict, streamObjNr, ownerObjNr int) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	d := sd.Dict
	if _, err := validateNameEntry(xRefTable, d, streamObjNr, "3DStreamDict", "Type", OPTIONAL, model.V16, func(s string) bool {
		return s == "3D"
	}); err != nil {
		return err
	}
	if err := validate3DStreamSubtype(xRefTable, d, streamObjNr); err != nil {
		return err
	}
	if err := validate3DDictArrayEntry(c, xRefTable, d, streamObjNr, "3DStreamDict", "VA", model.V16, validate3DViewDict); err != nil {
		return err
	}
	if err := validate3DViewSelector(
		c, xRefTable, d, streamObjNr, "3DStreamDict", "DV", OPTIONAL, threeDStreamViewSelector,
	); err != nil {
		return err
	}
	if err := validate3DResources(c, xRefTable, d, streamObjNr); err != nil {
		return err
	}
	js, err := validateStreamDictEntry(xRefTable, d, streamObjNr, "3DStreamDict", "OnInstantiate", OPTIONAL, model.V16, nil)
	if err != nil {
		return err
	}
	if js != nil {
		collect3DOnInstantiate(xRefTable, d, streamObjNr, ownerObjNr)
	}
	sinceVersion := threeDV17SinceVersion(xRefTable)
	animation, err := validateDictEntry(xRefTable, d, streamObjNr, "3DStreamDict", "AN", OPTIONAL, sinceVersion, nil)
	if err != nil || animation == nil {
		return err
	}
	showRelaxed3DVersionViolation(xRefTable, d, "3DStreamDict", "AN")
	return validate3DAnimationStyleDict(xRefTable, animation, validationEntryObjectNumber(streamObjNr, d, "AN"))
}

func validate3DReferenceDict(c context.Context, xRefTable *model.XRefTable, d types.Dict, ownerObjNr, annotObjNr int) error {
	const dictName = "3DReferenceDict"
	if _, err := validateNameEntry(xRefTable, d, ownerObjNr, dictName, "Type", OPTIONAL, model.V16, func(s string) bool {
		return s == "3DRef"
	}); err != nil {
		return err
	}
	streamObjNr := validationEntryObjectNumber(ownerObjNr, d, "3DD")
	required := xRefTable.ValidationMode == model.ValidationStrict
	o, err := validateEntry(xRefTable, d, ownerObjNr, dictName, "3DD", required, model.V16)
	if err != nil {
		return err
	}
	if o == nil {
		model.ShowSkipped(`3D reference dictionary: missing "3DD"`)
		return nil
	}
	sd, ok := o.(types.StreamDict)
	if !ok {
		err := fmt.Errorf("dict=%s entry=3DD: expected 3D stream, got %T", dictName, o)
		return model.WithValidationErrorObject(err, streamObjNr)
	}
	return validate3DStreamDict(c, xRefTable, sd, streamObjNr, annotObjNr)
}

func validate3DDataEntry(c context.Context, xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) error {
	raw, _ := d.Find("3DD")
	o, err := validateEntry(xRefTable, d, ownerObjNr, "3D", "3DD", REQUIRED, model.V16)
	if err != nil {
		return err
	}
	objNr := validationObjectNumber(ownerObjNr, raw)
	switch o := o.(type) {
	case types.StreamDict:
		return validate3DStreamDict(c, xRefTable, o, objNr, ownerObjNr)
	case types.Dict:
		return validate3DReferenceDict(c, xRefTable, o, objNr, ownerObjNr)
	default:
		err = fmt.Errorf("dict=3D entry=3DD: expected 3D stream or reference dictionary, got %T", o)
		return model.WithValidationErrorObject(err, objNr)
	}
}

func validateAnnotationDict3D(c context.Context, xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName string) error {
	// see 13.6.2
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if err := validate3DAnnotationAppearance(xRefTable, d, ownerObjNr, dictName); err != nil {
		return err
	}
	if err := validate3DDataEntry(c, xRefTable, d, ownerObjNr); err != nil {
		return err
	}
	if err := validate3DViewSelector(
		c, xRefTable, d, ownerObjNr, dictName, "3DV", OPTIONAL, threeDAnnotationViewSelector,
	); err != nil {
		return err
	}
	activation, err := validateDictEntry(xRefTable, d, ownerObjNr, dictName, "3DA", OPTIONAL, model.V16, nil)
	if err != nil {
		return err
	}
	if activation != nil {
		if err = validate3DActivationDict(xRefTable, activation, validationEntryObjectNumber(ownerObjNr, d, "3DA")); err != nil {
			return err
		}
	}
	if _, err = validateBooleanEntry(xRefTable, d, ownerObjNr, dictName, "3DI", OPTIONAL, model.V16, nil); err != nil {
		return err
	}
	_, err = validateRectangleEntry(xRefTable, d, ownerObjNr, dictName, "3DB", OPTIONAL, model.V16, nil)
	return err
}
