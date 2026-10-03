/*
Copyright 2018 The pdfcpu Authors.

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

var errMissingNameTreeKidsOrNames = errors.New("missing Kids or Names")

func validateDestsNameTreeValue(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {
	// Version check
	err := xRefTable.ValidateVersion("DestsNameTreeValue", sinceVersion)
	if err != nil {
		return err
	}

	_, err = validateDestination(xRefTable, o, 0, false)
	return err
}

func validateAPNameTreeValue(c context.Context, xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {
	// Version check
	err := xRefTable.ValidateVersion("APNameTreeValue", sinceVersion)
	if err != nil {
		return err
	}

	return validateXObjectStreamDict(c, xRefTable, o)
}

func validateJavaScriptNameTreeValueContext(c context.Context, xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {
	// Version check
	err := xRefTable.ValidateVersion("JavaScriptNameTreeValue", sinceVersion)
	if err != nil {
		return err
	}

	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return fmt.Errorf("JavaScript name tree value: dereference dict: %w", err)
	}

	// S, required, name
	_, err = validateNameEntry(xRefTable, d, 0, "JavaScript", "S", REQUIRED, model.V10, func(s string) bool {
		return s == "JavaScript"
	})
	if err != nil {
		return err
	}

	origin := activeContentOrigin{owner: activeContentOwnerNameTree, source: activeContentSourceJavaScriptNameTree}
	return validateActionDictObjectWithOrigin(c, xRefTable, d, o, "JavaScript name tree value", origin)
}

func validateJavaScriptNameTreeValue(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {
	return validateJavaScriptNameTreeValueContext(context.Background(), xRefTable, o, sinceVersion)
}

func validate3DResourcesNameTreeValue(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {
	if err := xRefTable.ValidateVersion("3DResourcesNameTreeValue", sinceVersion); err != nil {
		return err
	}
	_, err := xRefTable.Dereference(o)
	return err
}

func validatePagesNameTreeValue(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {
	// see 12.7.6

	// Version check
	err := xRefTable.ValidateVersion("PagesNameTreeValue", sinceVersion)
	if err != nil {
		return err
	}

	// Value is a page dict.

	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return fmt.Errorf("Pages name tree value: dereference page dict: %w", err)
	}

	if d == nil {
		return errors.New("Pages name tree value: missing page dict")
	}

	_, err = validateNameEntry(xRefTable, d, 0, "pageDict", "Type", REQUIRED, model.V10, func(s string) bool { return s == "Page" })

	return err
}

func validateTemplatesNameTreeValue(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {
	// see 12.7.6

	// Version check
	err := xRefTable.ValidateVersion("TemplatesNameTreeValue", sinceVersion)
	if err != nil {
		return err
	}

	// Value is a template dict.

	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return fmt.Errorf("Templates name tree value: dereference template dict: %w", err)
	}
	if d == nil {
		return errors.New("Templates name tree value: missing template dict")
	}

	_, err = validateNameEntry(xRefTable, d, 0, "templateDict", "Type", REQUIRED, model.V10, func(s string) bool { return s == "Template" })

	return err
}

func validateURLAliasDict(xRefTable *model.XRefTable, d types.Dict) error {
	dictName := "urlAliasDict"

	// U, required, ASCII string
	u, err := validateStringEntry(xRefTable, d, 0, dictName, "U", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// C, optional, array of strings
	_, err = validateStringArrayEntry(xRefTable, d, 0, dictName, "C", OPTIONAL, model.V10, nil)
	if err == nil && u != nil {
		collectDocumentLinkTarget(xRefTable, *u, linkSourceWebCaptureSource)
	}

	return err
}

func validateCommandSettingsDict(xRefTable *model.XRefTable, d types.Dict) error {
	// see 14.10.5.4

	dictName := "cmdSettingsDict"

	// G, optional, dict
	_, err := validateDictEntry(xRefTable, d, 0, dictName, "G", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// C, optional, dict
	_, err = validateDictEntry(xRefTable, d, 0, dictName, "C", OPTIONAL, model.V10, nil)

	return err
}

func validateCaptureCommandDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) error {
	dictName := "captureCommandDict"

	// URL, required, string
	commandURL, err := validateStringEntry(xRefTable, d, 0, dictName, "URL", REQUIRED, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s.URL: %w", dictName, err)
	}

	// L, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, 0, dictName, "L", OPTIONAL, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s.L: %w", dictName, err)
	}

	// F, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, 0, dictName, "F", OPTIONAL, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s.F: %w", dictName, err)
	}

	// P, optional, string or stream
	err = validateStringOrStreamEntry(xRefTable, d, ownerObjNr, dictName, "P", OPTIONAL, model.V10)
	if err != nil {
		return fmt.Errorf("%s.P: %w", dictName, err)
	}

	// CT, optional, ASCII string
	_, err = validateStringEntry(xRefTable, d, 0, dictName, "CT", OPTIONAL, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s.CT: %w", dictName, err)
	}

	// H, optional, string
	_, err = validateStringEntry(xRefTable, d, 0, dictName, "H", OPTIONAL, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s.H: %w", dictName, err)
	}

	// S, optional, command settings dict
	d1, err := validateDictEntry(xRefTable, d, 0, dictName, "S", OPTIONAL, model.V10, nil)
	if err != nil {
		return fmt.Errorf("%s.S: %w", dictName, err)
	}
	if d1 != nil {
		err = validateCommandSettingsDict(xRefTable, d1)
		if err != nil {
			return fmt.Errorf("%s.S: %w", dictName, err)
		}
	}
	if commandURL != nil {
		collectDocumentLinkTarget(xRefTable, *commandURL, linkSourceWebCaptureCommand)
	}

	return nil
}

func validateSourceInfoDictEntryAU(xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {
	o, err := validateEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case types.StringLiteral, types.HexLiteral:
		s, err := model.Text(o)
		if err != nil {
			return fmt.Errorf("dict=%s entry=%s: %w", dictName, entryName, err)
		}
		collectDocumentLinkTarget(xRefTable, s, linkSourceWebCaptureSource)

	case types.Dict:
		err = validateURLAliasDict(xRefTable, o)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("dict=%s entry=%s expected string or dict, got %T", dictName, entryName, o)

	}

	return nil
}

func validateSourceInfoDict(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int) (err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()

	dictName := "sourceInfoDict"

	// AU, required, ASCII string or dict
	err = validateSourceInfoDictEntryAU(xRefTable, d, dictName, "AU", REQUIRED, model.V10)
	if err != nil {
		return err
	}

	// E, optional, date
	_, err = validateDateEntry(xRefTable, d, 0, dictName, "E", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	// S, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, 0, dictName, "S", OPTIONAL, model.V10, func(i int) bool { return 0 <= i && i <= 2 })
	if err != nil {
		return err
	}

	// C, optional, indRef of command dict
	ir, err := validateIndRefEntry(xRefTable, d, ownerObjNr, dictName, "C", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	if ir != nil {
		commandObjNr := ir.ObjectNumber.Value()

		d1, err := xRefTable.DereferenceDict(*ir)
		if err != nil {
			return model.WithValidationErrorObject(err, commandObjNr)
		}

		err = validateCaptureCommandDict(xRefTable, d1, commandObjNr)
		return model.WithValidationErrorObject(err, commandObjNr)

	}

	return nil
}

func validateEntrySI(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string, required bool, sinceVersion model.Version) error {
	// see 14.10.5, table 355, source information dictionary

	siObjNr := validationEntryObjectNumber(ownerObjNr, d, entryName)
	o, err := validateEntry(xRefTable, d, ownerObjNr, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	switch o := o.(type) {

	case types.Dict:
		err = validateSourceInfoDict(xRefTable, o, siObjNr)
		if err != nil {
			return err
		}

	case types.Array:

		for i, v := range o {

			if v == nil {
				continue
			}

			objNr := validationObjectNumber(siObjNr, v)
			d1, err := xRefTable.DereferenceDict(v)
			if err != nil {
				err = fmt.Errorf("dict=%s entry=%s[%d]: dereference dict: %w", dictName, entryName, i, err)
				return model.WithValidationErrorObject(err, objNr)
			}

			err = validateSourceInfoDict(xRefTable, d1, objNr)
			if err != nil {
				return fmt.Errorf("dict=%s entry=%s[%d]: %w", dictName, entryName, i, err)
			}

		}

	}

	return nil
}

func validateWebCaptureContentSetDict(XRefTable *model.XRefTable, d types.Dict, ownerObjNr int) (err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()

	// see 14.10.4

	dictName := "webCaptureContentSetDict"

	// Type, optional, name
	_, err = validateNameEntry(XRefTable, d, 0, dictName, "Type", OPTIONAL, model.V10, func(s string) bool { return s == "SpiderContentSet" })
	if err != nil {
		return err
	}

	// S, required, name
	s, err := validateNameEntry(XRefTable, d, 0, dictName, "S", REQUIRED, model.V10, func(s string) bool { return s == "SPS" || s == "SIS" })
	if err != nil {
		return err
	}

	// ID, required, byte string
	_, err = validateStringEntry(XRefTable, d, 0, dictName, "ID", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// O, required, array of indirect references.
	_, err = validateIndRefArrayEntry(XRefTable, d, ownerObjNr, dictName, "O", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}

	// SI, required, source info dict or array of source info dicts
	err = validateEntrySI(XRefTable, d, ownerObjNr, dictName, "SI", REQUIRED, model.V10)
	if err != nil {
		return err
	}

	// CT, optional, string
	_, err = validateStringEntry(XRefTable, d, 0, dictName, "CT", OPTIONAL, model.V10, nil)
	if err != nil {
		return err
	}

	// TS, optional, date
	_, err = validateDateEntry(XRefTable, d, 0, dictName, "TS", OPTIONAL, model.V10)
	if err != nil {
		return err
	}

	// spider page set
	if *s == "SPS" {

		// T, optional, string
		_, err = validateStringEntry(XRefTable, d, 0, dictName, "T", OPTIONAL, model.V10, nil)
		if err != nil {
			return err
		}

		// TID, optional, byte string
		_, err = validateStringEntry(XRefTable, d, 0, dictName, "TID", OPTIONAL, model.V10, nil)
		if err != nil {
			return err
		}
	}

	// spider image set
	if *s == "SIS" {

		// R, required, integer or array of integers
		err = validateIntegerOrArrayOfIntegerEntry(XRefTable, d, 0, dictName, "R", REQUIRED, model.V10)

	}

	return err
}

func validateWebCaptureContentSet(xRefTable *model.XRefTable, o types.Object, ownerObjNr int) error {
	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return fmt.Errorf("dereference content set dict: %w", err)
	}
	if d == nil {
		return errors.New("missing content set dict")
	}
	return validateWebCaptureContentSetDict(xRefTable, d, validationObjectNumber(ownerObjNr, o))
}

func validateWebCaptureNameTreeValue(c context.Context, xRefTable *model.XRefTable, o types.Object, name string) error {
	value, err := xRefTable.Dereference(o)
	if err != nil {
		return fmt.Errorf("%s name tree value: dereference: %w", name, err)
	}
	ownerObjNr := validationObjectNumber(0, o)
	switch value := value.(type) {
	case types.Dict:
		return validateWebCaptureContentSet(xRefTable, value, ownerObjNr)
	case types.Array:
		for i, contentSet := range value {
			if err := contextutil.Check(c); err != nil {
				return err
			}
			if err := validateWebCaptureContentSet(xRefTable, contentSet, ownerObjNr); err != nil {
				return fmt.Errorf("%s name tree value[%d]: %w", name, i, err)
			}
		}
		return nil
	}
	return fmt.Errorf("%s name tree value: expected content set dict or array, got %T", name, value)
}

func validateIDSNameTreeValue(c context.Context, xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {
	// see 14.10.4

	if err := xRefTable.ValidateVersion("IDSNameTreeValue", sinceVersion); err != nil {
		return err
	}
	return validateWebCaptureNameTreeValue(c, xRefTable, o, "IDS")
}

func validateURLSNameTreeValue(c context.Context, xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {
	// see 14.10.4

	if err := xRefTable.ValidateVersion("URLSNameTreeValue", sinceVersion); err != nil {
		return err
	}
	return validateWebCaptureNameTreeValue(c, xRefTable, o, "URLS")
}

func validateEmbeddedFilesNameTreeValue(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {
	// see 7.11.4

	// Value is a file specification for an embedded file stream.

	// Version check
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V11
	}
	err := xRefTable.ValidateVersion("EmbeddedFilesNameTreeValue", sinceVersion)
	if err != nil {
		return err
	}

	objNr := validationObjectNumber(0, o)
	f, err := validateFileSpecificationWithoutLinkCollection(xRefTable, o)
	if err != nil {
		return fmt.Errorf("EmbeddedFiles name tree value: %w", err)
	}
	if !isEmbeddedFileSpecification(xRefTable, f) {
		if xRefTable.ValidationMode == model.ValidationRelaxed && isEmptyEmbeddedFileSpecification(xRefTable, f) {
			target, found := fileSpecificationTarget(xRefTable, f)
			if !found {
				target = "unnamed"
			}
			model.ShowSkipped(fmt.Sprintf(
				`EmbeddedFiles name tree value %q (obj#:%d): empty EF dictionary`, target, objNr,
			))
			return nil
		}
		return errors.New("EmbeddedFiles name tree value: expected matching embedded file stream")
	}
	return nil
}

func isEmptyEmbeddedFileSpecification(xRefTable *model.XRefTable, o types.Object) bool {
	if isURLFileSpecification(xRefTable, o) {
		return false
	}
	o, err := xRefTable.Dereference(o)
	if err != nil {
		return false
	}
	d, ok := o.(types.Dict)
	if !ok {
		return false
	}
	rawEF, found := d.Find("EF")
	if !found {
		return false
	}
	ef, err := xRefTable.DereferenceDict(rawEF)
	return err == nil && ef != nil && len(ef) == 0
}

func validateRichMediaAssetNameTreeValue(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {
	if err := xRefTable.ValidateVersion("RichMediaAssetNameTreeValue", sinceVersion); err != nil {
		return err
	}

	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return fmt.Errorf("RichMedia Assets name tree value: dereference file specification: %w", err)
	}
	if d == nil {
		return errors.New("RichMedia Assets name tree value: missing file specification")
	}
	if isURLFileSpecification(xRefTable, d) {
		return errors.New("RichMedia Assets name tree value: expected embedded file specification")
	}
	if _, err = validateFileSpecificationWithoutLinkCollection(xRefTable, o); err != nil {
		return fmt.Errorf("RichMedia Assets name tree value: %w", err)
	}

	rawEF, found := d.Find("EF")
	if !found {
		return errors.New("RichMedia Assets name tree value: missing EF dictionary")
	}
	ef, err := xRefTable.DereferenceDict(rawEF)
	if err != nil {
		return fmt.Errorf("RichMedia Assets name tree value EF: %w", err)
	}
	if ef == nil {
		return errors.New("RichMedia Assets name tree value: missing EF dictionary")
	}
	for _, key := range []string{"F", "UF"} {
		raw, found := ef.Find(key)
		if !found {
			continue
		}
		embeddedFile, err := xRefTable.Dereference(raw)
		if err != nil {
			return fmt.Errorf("RichMedia Assets name tree value EF.%s: %w", key, err)
		}
		if _, ok := embeddedFile.(types.StreamDict); ok {
			return nil
		}
	}

	return errors.New("RichMedia Assets name tree value: missing embedded file stream reference")
}

func validateSlideShowResource(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {
	if err := xRefTable.ValidateVersion("SlideShowResource", sinceVersion); err != nil {
		return err
	}

	resource, err := xRefTable.Dereference(o)
	if err != nil {
		return fmt.Errorf("slide show resource: dereference: %w", err)
	}
	if resource == nil {
		return errors.New("slide show resource: missing object")
	}

	var d types.Dict
	switch resource := resource.(type) {
	case types.Dict:
		d = resource
	case types.StreamDict:
		d = resource.Dict
	default:
		return fmt.Errorf("slide show resource: expected dictionary or stream, got %T", resource)
	}

	typeName, err := validateNameEntry(xRefTable, d, 0, "slideShowResource", "Type", REQUIRED, model.V10, nil)
	if err != nil {
		return err
	}
	isFileSpec := typeName.Value() == "Filespec" || typeName.Value() == "FileSpec"
	if xRefTable.ValidationMode == model.ValidationRelaxed && typeName.Value() == "F" {
		isFileSpec = true
	}
	if !isFileSpec {
		return nil
	}

	f, err := validateFileSpecificationWithoutLinkCollection(xRefTable, o)
	if err != nil {
		return fmt.Errorf("slide show resource file specification: %w", err)
	}
	if !isEmbeddedFileSpecification(xRefTable, f) {
		collectDocumentFileSpecificationTarget(
			xRefTable, f, linkTargetFile, linkSourceAlternatePresentation,
		)
	}
	return nil
}

func validateLegacySlideShowResource(xRefTable *model.XRefTable, o types.Object) error {
	resource, err := xRefTable.Dereference(o)
	if err != nil {
		return fmt.Errorf("slide show resource: dereference: %w", err)
	}
	if resource == nil {
		return errors.New("slide show resource: missing object")
	}

	var d types.Dict
	switch resource := resource.(type) {
	case types.Dict:
		d = resource
	case types.StreamDict:
		d = resource.Dict
	default:
		return nil
	}
	if _, found := d.Find("Type"); !found {
		return nil
	}
	return validateSlideShowResource(xRefTable, o, model.V14)
}

func validateLegacySlideShowResources(c context.Context, xRefTable *model.XRefTable, a types.Array, objNr int, startResource string) error {
	const (
		dictName  = "slideShowDict"
		entryName = "Resources"
	)
	if err := validateArrayPairs(a, objNr, dictName, entryName, 1); err != nil {
		return err
	}
	message := "slideShowDict.Resources: accepted legacy flat resource array"
	cause := model.WithValidationErrorObject(errors.New(message), objNr)
	xRefTable.AddValidationNotice(model.NewValidationNotice(
		model.NoticePhaseValidate, model.NoticeSkipped, message, cause,
	))
	found := false
	for i := 0; i < len(a); i += 2 {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		keyObjNr := validationObjectNumber(objNr, a[i])
		o, err := xRefTable.Dereference(a[i])
		if err != nil {
			return model.WithValidationErrorObject(fmt.Errorf("%s.%s[%d]: dereference key: %w", dictName, entryName, i, err), keyObjNr)
		}
		key, err := types.StringOrHexLiteral(o)
		if err != nil {
			return model.WithValidationErrorObject(fmt.Errorf("%s.%s[%d]: expected string: %w", dictName, entryName, i, err), keyObjNr)
		}
		found = found || *key == startResource
		resource := a[i+1]
		resourceObjNr := validationObjectNumber(objNr, resource)
		if _, ok := resource.(types.IndirectRef); !ok {
			return model.WithValidationErrorObject(
				fmt.Errorf("%s.%s[%d]: expected indirect reference", dictName, entryName, i+1), resourceObjNr,
			)
		}
		if err = validateLegacySlideShowResource(xRefTable, resource); err != nil {
			return model.WithValidationErrorObject(
				fmt.Errorf("%s.%s[%d]: %w", dictName, entryName, i+1, err), resourceObjNr,
			)
		}
	}
	if !found {
		return fmt.Errorf("%s.StartResource: resource %q not found", dictName, startResource)
	}
	return nil
}

func validateSlideShowResources(c context.Context, xRefTable *model.XRefTable, d types.Dict, startResource string) error {
	const (
		dictName  = "slideShowDict"
		entryName = "Resources"
	)
	rawResources := d[entryName]
	resourcesObjNr := validationObjectNumber(0, rawResources)
	resources, err := validateEntry(xRefTable, d, 0, dictName, entryName, REQUIRED, model.V14)
	if err != nil {
		return err
	}
	if a, ok := resources.(types.Array); ok && xRefTable.ValidationMode == model.ValidationRelaxed {
		return validateLegacySlideShowResources(c, xRefTable, a, resourcesObjNr, startResource)
	}
	resourcesDict, ok := resources.(types.Dict)
	if !ok {
		return model.WithValidationErrorObject(
			fmt.Errorf("dict=%s entry=%s invalid type %T", dictName, entryName, resources), resourcesObjNr,
		)
	}
	_, _, tree, err := validateNameTree(
		c, xRefTable, "SlideShowResources", resourcesDict, resourcesObjNr, true, rawResources,
	)
	if err != nil {
		return err
	}
	if _, found, err := tree.Value(c, startResource); err != nil {
		return err
	} else if !found {
		return fmt.Errorf("%s.StartResource: resource %q not found", dictName, startResource)
	}
	return nil
}

func validateSlideShowDict(c context.Context, xRefTable *model.XRefTable, d types.Dict) error {
	// see 13.5, table 297

	dictName := "slideShowDict"

	// Type, required, name, since V1.4
	_, err := validateNameEntry(xRefTable, d, 0, dictName, "Type", REQUIRED, model.V14, func(s string) bool { return s == "SlideShow" })
	if err != nil {
		return err
	}

	// Subtype, required, name, since V1.4
	_, err = validateNameEntry(xRefTable, d, 0, dictName, "Subtype", REQUIRED, model.V14, func(s string) bool { return s == "Embedded" })
	if err != nil {
		return err
	}

	// StartResource, required, byte string, since V1.4
	startResource, err := validateStringEntry(xRefTable, d, 0, dictName, "StartResource", REQUIRED, model.V14, nil)
	if err != nil {
		return err
	}

	// Resources, required name tree, since V1.4
	return validateSlideShowResources(c, xRefTable, d, *startResource)
}

func validateAlternatePresentationsNameTreeValue(c context.Context, xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {
	// see 13.5

	// Value is a slide show dict.

	// Version check
	err := xRefTable.ValidateVersion("AlternatePresentationsNameTreeValue", sinceVersion)
	if err != nil {
		return err
	}

	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return fmt.Errorf("AlternatePresentations name tree value: dereference slide show dict: %w", err)
	}

	if d != nil {
		err = validateSlideShowDict(c, xRefTable, d)
	}

	return err
}

func validateRenditionsNameTreeValue(c context.Context, xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error {
	// see 13.2.3

	// Value is a rendition object.

	// Version check
	err := xRefTable.ValidateVersion("RenditionsNameTreeValue", sinceVersion)
	if err != nil {
		return err
	}

	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return fmt.Errorf("Renditions name tree value: dereference rendition dict: %w", err)
	}

	if d != nil {
		err = validateRenditionDict(c, xRefTable, d, validationObjectNumber(0, o), sinceVersion)
	}

	return err
}

func validateIDTreeValueContext(c context.Context, xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) (err error) {
	objNr := validationObjectNumber(0, o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	// Version check
	err = xRefTable.ValidateVersion("IDTreeValue", sinceVersion)
	if err != nil {
		return err
	}

	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return fmt.Errorf("IDTree value: dereference structure element dict: %w", err)
	}
	if d == nil {
		return errors.New("IDTree value: missing structure element dict")
	}

	dictType, _, err := xRefTable.DereferenceNameEntry(d, "Type")
	if err != nil {
		return fmt.Errorf("IDTree value Type: %w", err)
	}
	if dictType == nil || dictType.Value() == "StructElem" {
		err = validateStructElementDictContext(c, xRefTable, d, true)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("IDTree value: unexpected dict Type %s, expected StructElem", dictType.Value())
	}

	return nil
}

func validateNameTreeValue(c context.Context, name string, xRefTable *model.XRefTable, o types.Object, ownerObjNr int) (err error) {
	objNr := validationObjectNumber(ownerObjNr, o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	// The values associated with the keys may be objects of any type.
	// Stream objects shall be specified by indirect object references.
	// Dictionary, array, and string objects should be specified by indirect object references.
	// Other PDF objects (nulls, numbers, booleans, and names) should be specified as direct objects.

	for k, v := range map[string]struct {
		validate            func(xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) error
		sinceVersion        model.Version
		sinceVersionRelaxed model.Version
	}{
		"Dests": {validateDestsNameTreeValue, model.V12, model.V12},
		"AP": {func(x *model.XRefTable, o types.Object, version model.Version) error {
			return validateAPNameTreeValue(c, x, o, version)
		}, model.V13, model.V13},
		"JavaScript": {func(x *model.XRefTable, o types.Object, version model.Version) error {
			return validateJavaScriptNameTreeValueContext(c, x, o, version)
		}, model.V13, model.V13},
		"3DResources": {validate3DResourcesNameTreeValue, model.V16, model.V16},
		"Pages":       {validatePagesNameTreeValue, model.V13, model.V13},
		"Templates":   {validateTemplatesNameTreeValue, model.V13, model.V13},
		"IDS": {func(x *model.XRefTable, o types.Object, version model.Version) error {
			return validateIDSNameTreeValue(c, x, o, version)
		}, model.V13, model.V13},
		"URLS": {func(x *model.XRefTable, o types.Object, version model.Version) error {
			return validateURLSNameTreeValue(c, x, o, version)
		}, model.V13, model.V13},
		"EmbeddedFiles":   {validateEmbeddedFilesNameTreeValue, model.V14, model.V11},
		"RichMediaAssets": {validateRichMediaAssetNameTreeValue, model.V10, model.V10},
		"SlideShowResources": {func(x *model.XRefTable, o types.Object, version model.Version) error {
			return validateSlideShowResource(x, o, version)
		}, model.V14, model.V14},
		"AlternatePresentations": {func(x *model.XRefTable, o types.Object, version model.Version) error {
			return validateAlternatePresentationsNameTreeValue(c, x, o, version)
		}, model.V14, model.V14},
		"Renditions": {func(x *model.XRefTable, o types.Object, version model.Version) error {
			return validateRenditionsNameTreeValue(c, x, o, version)
		}, model.V15, model.V15},
	} {
		if name == k {
			sinceVersion := v.sinceVersion
			if xRefTable.ValidationMode == model.ValidationRelaxed {
				sinceVersion = v.sinceVersionRelaxed
			}
			return v.validate(xRefTable, o, sinceVersion)
		}
	}

	return fmt.Errorf("name tree %s: unknown tree name", name)
}

func validateNameTreeValueContext(c context.Context, name string, x *model.XRefTable, o types.Object, owner int) error {
	if name == "IDTree" {
		return validateIDTreeValueContext(c, x, o, model.V13)
	}
	return validateNameTreeValue(c, name, x, o, owner)
}

func validateNameTreeDictNamesEntry(c context.Context, xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, name string, node *model.Node) (string, string, error) {
	//fmt.Printf("validateNameTreeDictNamesEntry begin %s\n", d)

	// Names: array of the form [key1 value1 key2 value2 ... key n value n]
	o, found := d.Find("Names")
	if !found {
		err := fmt.Errorf("name tree %s: %w", name, errMissingNameTreeKidsOrNames)
		return "", "", model.WithValidationErrorObject(err, ownerObjNr)
	}
	namesObjNr := validationObjectNumber(ownerObjNr, o)

	a, err := xRefTable.DereferenceArray(o)
	if err != nil {
		err = fmt.Errorf("name tree %s Names: dereference array: %w", name, err)
		return "", "", model.WithValidationErrorObject(err, namesObjNr)
	}
	if a == nil {
		err = fmt.Errorf("name tree %s: missing Names array", name)
		return "", "", model.WithValidationErrorObject(err, namesObjNr)
	}

	// arr length needs to be even because of contained key value pairs.
	entries := len(a)
	if entries%2 == 1 {
		if xRefTable.ValidationMode != model.ValidationRelaxed || name != "JavaScript" {
			err = fmt.Errorf("name tree %s Names: odd entry count %d", name, len(a))
			return "", "", model.WithValidationErrorObject(err, namesObjNr)
		}
		entries--
	}

	var key, firstKey, lastKey string

	for i := 0; i < entries; i++ {
		if err := contextutil.Check(c); err != nil {
			return "", "", err
		}
		o := a[i]

		if i%2 == 0 {
			keyObjNr := validationObjectNumber(namesObjNr, o)

			// TODO Do we really need to process indRefs here?
			o, err = xRefTable.Dereference(o)
			if err != nil {
				err = fmt.Errorf("name tree %s Names[%d]: dereference key: %w", name, i, err)
				return "", "", model.WithValidationErrorObject(err, keyObjNr)
			}

			k, err := types.StringOrHexLiteral(o)
			if err != nil {
				err = fmt.Errorf("name tree %s Names[%d]: expected string key: %w", name, i, err)
				return "", "", model.WithValidationErrorObject(err, keyObjNr)
			}

			key = *k

			if firstKey == "" {
				firstKey = key
			}

			lastKey = key

			continue
		}

		err = validateNameTreeValueContext(c, name, xRefTable, o, namesObjNr)
		if err != nil {
			return "", "", fmt.Errorf("name tree %s key %q: %w", name, key, err)
		}
		if name == "URLS" {
			collectDocumentLinkTarget(xRefTable, key, linkSourceWebCaptureURL)
		}

		node.AppendToNames(key, o)

	}

	return firstKey, lastKey, nil
}

func validateNameTreeDictLimitsEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, firstKey, lastKey string) error {
	a, err := validateStringArrayEntry(
		xRefTable, d, ownerObjNr, "nameTreeDict", "Limits", REQUIRED, model.V10,
		func(a types.Array) bool { return len(a) == 2 },
	)
	if err != nil {
		return err
	}

	o, err := xRefTable.Dereference(a[0])
	if err != nil {
		return fmt.Errorf("name tree Limits[0]: dereference: %w", err)
	}
	s, err := types.StringOrHexLiteral(o)
	if err != nil {
		return fmt.Errorf("name tree Limits[0]: expected string: %w", err)
	}
	fkv := *s

	o, err = xRefTable.Dereference(a[1])
	if err != nil {
		return fmt.Errorf("name tree Limits[1]: dereference: %w", err)
	}
	s, err = types.StringOrHexLiteral(o)
	if err != nil {
		return fmt.Errorf("name tree Limits[1]: expected string: %w", err)
	}
	lkv := *s

	if xRefTable.ValidationMode == model.ValidationRelaxed {
		if fkv != firstKey {
			fkv = firstKey
		}

		if lkv != lastKey {
			lkv = lastKey
		}
	}

	if firstKey != fkv || lastKey != lkv {
		return fmt.Errorf("name tree leaf limits: first key %s, expected %s; last key %s, expected %s", fkv, firstKey, lkv, lastKey)
	}

	return nil
}

func validateNameTree(c context.Context, xRefTable *model.XRefTable, name string, d types.Dict, ownerObjNr int, root bool, rootObject ...types.Object) (string, string, *model.Node, error) {
	return validateNameTreeDepth(c, xRefTable, name, d, ownerObjNr, root, 0, rootObject...)
}

func nameTreeKidContext(name string, o types.Object, i int) string {
	if ir, ok := o.(types.IndirectRef); ok {
		return fmt.Sprintf("name tree %s Kids[%d] obj#%d", name, i, ir.ObjectNumber.Value())
	}
	return fmt.Sprintf("name tree %s Kids[%d]", name, i)
}

func validateNameTreeKids(c context.Context, xRefTable *model.XRefTable, name string, a types.Array, ownerObjNr int, node *model.Node, depth int, specViolations *[]error, visits ...*treeVisit) (string, string, error) {
	visit := treeTraversal(visits, true)
	var kmin, kmax string

	for i, o := range a {
		if err := contextutil.Check(c); err != nil {
			return "", "", err
		}
		kidObjNr := validationObjectNumber(ownerObjNr, o)

		d, err := xRefTable.DereferenceDict(o)
		if err != nil {
			err = fmt.Errorf("%s: dereference dict: %w", nameTreeKidContext(name, o, i), err)
			return "", "", model.WithValidationErrorObject(err, kidObjNr)
		}
		if d == nil {
			err = fmt.Errorf("%s: missing dict", nameTreeKidContext(name, o, i))
			return "", "", model.WithValidationErrorObject(err, kidObjNr)
		}

		kminKid, kmaxKid, kidNode, err := validateNameTreeChild(c,
			xRefTable,
			name,
			d,
			kidObjNr,
			o,
			depth+1,
			specViolations,
			visit,
		)
		if err != nil {
			err = model.WrapRecursionError(nameTreeKidContext(name, o, i), err)
			if xRefTable.ValidationMode == model.ValidationStrict || contextutil.Check(c) != nil || fatalTreeError(err) {
				return "", "", err
			}
			if errors.Is(err, errMissingNameTreeKidsOrNames) {
				*specViolations = append(*specViolations, err)
			}
			continue
		}
		kmax = kmaxKid
		if kmin == "" {
			kmin = kminKid
		}

		node.Kids = append(node.Kids, kidNode)
	}

	return kmin, kmax, nil
}

func validateNameTreeDepth(c context.Context, xRefTable *model.XRefTable, name string, d types.Dict, ownerObjNr int, root bool, depth int, rootObject ...types.Object) (string, string, *model.Node, error) {
	var specViolations []error
	kmin, kmax, node, err := validateNameTreeDepthWithViolations(c,
		xRefTable,
		name,
		d,
		ownerObjNr,
		root,
		depth,
		&specViolations,
		newTreeVisit(true, rootObject...),
	)
	if err == nil {
		showDigestedSpecViolations(specViolations)
	}
	return kmin, kmax, node, err
}

func validateNameTreeDepthWithViolations(c context.Context, xRefTable *model.XRefTable, name string, d types.Dict, ownerObjNr int, root bool, depth int, specViolations *[]error, visits ...*treeVisit) (kmin, kmax string, node *model.Node, err error) {
	visit := treeTraversal(visits, true)
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()

	if err := checkValidationTree(c, xRefTable, fmt.Sprintf("name tree %s", name), depth); err != nil {
		return "", "", nil, err
	}

	//fmt.Printf("validateNameTree begin %s\n", d)

	// see 7.7.4

	// A node has "Kids" or "Names" entry.

	//fmt.Printf("validateNameTree %s\n", name)

	node = &model.Node{D: d}

	// Kids: array of indirect references to the immediate children of this node.
	// if Kids present then recurse
	if o, found := d.Find("Kids"); found {
		kidsObjNr := validationObjectNumber(ownerObjNr, o)

		// Intermediate node

		a, err := xRefTable.DereferenceArray(o)
		if err != nil {
			return "", "", nil, fmt.Errorf("name tree %s Kids: dereference array: %w", name, err)
		}
		if a == nil {
			return "", "", nil, fmt.Errorf("name tree %s: missing Kids array", name)
		}

		if len(a) == 0 {
			if xRefTable.ValidationMode == model.ValidationStrict {
				return "", "", nil, fmt.Errorf("name tree %s: empty Kids array", name)
			}
			return "", "", nil, nil
		}

		kmin, kmax, err = validateNameTreeKids(c, xRefTable, name, a, kidsObjNr, node, depth, specViolations, visit)
		if err != nil {
			return "", "", nil, err
		}
	} else {

		// Leaf node
		kmin, kmax, err = validateNameTreeDictNamesEntry(c, xRefTable, d, ownerObjNr, name, node)
		if err != nil {
			if root &&
				xRefTable.ValidationMode == model.ValidationRelaxed &&
				errors.Is(err, errMissingNameTreeKidsOrNames) {
				*specViolations = append(*specViolations, err)
				return "", "", node, nil
			}
			return "", "", nil, err
		}
	}

	if !root {

		// Verify calculated key range.
		err = validateNameTreeDictLimitsEntry(xRefTable, d, ownerObjNr, kmin, kmax)
		if err != nil {
			return "", "", nil, fmt.Errorf("name tree %s Limits: %w", name, err)
		}
	}

	// We track limits for all nodes internally.
	node.Kmin = kmin
	node.Kmax = kmax

	//fmt.Println("validateNameTree end")

	return kmin, kmax, node, nil
}

func validateNameTreeChild(c context.Context, xRefTable *model.XRefTable, name string, d types.Dict, ownerObjNr int, o types.Object, depth int, specViolations *[]error, visit *treeVisit) (string, string, *model.Node, error) {
	n, err := visit.enter(o)
	if err != nil {
		return "", "", nil, err
	}
	defer visit.leave(n)
	return validateNameTreeDepthWithViolations(c, xRefTable, name, d, ownerObjNr, false, depth, specViolations, visit)
}
