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
	"fmt"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func mediaEntryError(d types.Dict, entryName string, err error) error {
	return model.WithValidationErrorObject(err, validationEntryObjectNumber(0, d, entryName))
}

func validateMinimumBitDepthDict(xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version) error {
	// see table 269

	dictName := "minBitDepthDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, 0, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MinBitDepth" })
	if err != nil {
		return err
	}

	// V, required, integer
	_, err = validateIntegerEntry(xRefTable, d, 0, dictName, "V", REQUIRED, sinceVersion, func(i int) bool { return i >= 0 })
	if err != nil {
		return err
	}

	// M, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, 0, dictName, "M", OPTIONAL, sinceVersion, nil)

	return err
}

func validateMinimumScreenSizeDict(xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version) error {
	// see table 269

	dictName := "minBitDepthDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, 0, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MinScreenSize" })
	if err != nil {
		return err
	}

	// V, required, integer array, length 2
	_, err = validateIntegerArrayEntry(xRefTable, d, 0, dictName, "V", REQUIRED, sinceVersion, func(a types.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// M, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, 0, dictName, "M", OPTIONAL, sinceVersion, nil)

	return err
}

func validateSoftwareIdentifierDict(xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version) error {
	// see table 292

	dictName := "swIdDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, 0, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "SoftwareIdentifier" })
	if err != nil {
		return err
	}

	// U, required, ASCII string
	_, err = validateStringEntry(xRefTable, d, 0, dictName, "U", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	// L, optional, array
	_, err = validateArrayEntry(xRefTable, d, 0, dictName, "L", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// LI, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "LI", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// H, optional, array
	_, err = validateArrayEntry(xRefTable, d, 0, dictName, "H", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// HI, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "HI", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// OS, optional, array
	_, err = validateStringArrayEntry(xRefTable, d, 0, dictName, "OS", OPTIONAL, sinceVersion, nil)

	return err
}

func validateMediaCriteriaDictEntryD(xRefTable *model.XRefTable, d types.Dict, dictName string, required bool, sinceVersion model.Version) (err error) {
	defer func() {
		err = mediaEntryError(d, "D", err)
	}()

	rawEntry := d["D"]
	d1, err := validateDictEntry(xRefTable, d, 0, dictName, "D", required, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", dictEntryContext(dictName, "D", rawEntry), err)
	}

	if d1 != nil {
		err = validateMinimumBitDepthDict(xRefTable, d1, sinceVersion)
		if err != nil {
			return fmt.Errorf("%s: %w", dictEntryContext(dictName, "D", rawEntry), err)
		}
	}

	return nil
}

func validateMediaCriteriaDictEntryZ(xRefTable *model.XRefTable, d types.Dict, dictName string, required bool, sinceVersion model.Version) (err error) {
	defer func() {
		err = mediaEntryError(d, "Z", err)
	}()

	rawEntry := d["Z"]
	d1, err := validateDictEntry(xRefTable, d, 0, dictName, "Z", required, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", dictEntryContext(dictName, "Z", rawEntry), err)
	}

	if d1 != nil {
		err = validateMinimumScreenSizeDict(xRefTable, d1, sinceVersion)
		if err != nil {
			return fmt.Errorf("%s: %w", dictEntryContext(dictName, "Z", rawEntry), err)
		}
	}

	return nil
}

func validateMediaCriteriaDictEntryV(xRefTable *model.XRefTable, d types.Dict, dictName string, required bool, sinceVersion model.Version) (err error) {
	arrayObjNr := validationEntryObjectNumber(0, d, "V")
	defer func() {
		err = model.WithValidationErrorObject(err, arrayObjNr)
	}()

	a, err := validateArrayEntry(xRefTable, d, 0, dictName, "V", required, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.V: %w", dictName, err)
	}

	for i, v := range a {
		objNr := validationObjectNumber(arrayObjNr, v)

		if v == nil {
			continue
		}

		d, err := xRefTable.DereferenceDict(v)
		if err != nil {
			err = fmt.Errorf("%s.V[%d]: dereference software identifier dict: %w", dictName, i, err)
			return model.WithValidationErrorObject(err, objNr)
		}

		if d != nil {
			err = validateSoftwareIdentifierDict(xRefTable, d, sinceVersion)
			if err != nil {
				err = fmt.Errorf("%s.V[%d]: %w", dictName, i, err)
				return model.WithValidationErrorObject(err, objNr)
			}
		}

	}

	return nil
}

func validateMediaCriteriaDict(xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version) error {
	// see table 268

	dictName := "mediaCritDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, 0, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaCriteria" })
	if err != nil {
		return err
	}

	// A, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "A", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// C, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "C", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// O, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "O", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// S, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "S", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// R, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, 0, dictName, "R", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// D, optional, dict
	err = validateMediaCriteriaDictEntryD(xRefTable, d, dictName, OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// Z, optional, dict
	err = validateMediaCriteriaDictEntryZ(xRefTable, d, dictName, OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// V, optional, array
	err = validateMediaCriteriaDictEntryV(xRefTable, d, dictName, OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// P, optional, array
	_, err = validateNameArrayEntry(xRefTable, d, 0, dictName, "P", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a) == 1 || len(a) == 2 })
	if err != nil {
		return err
	}

	// L, optional, array
	_, err = validateStringArrayEntry(xRefTable, d, 0, dictName, "L", OPTIONAL, sinceVersion, nil)

	return err
}

func validateMediaPermissionsDict(xRefTable *model.XRefTable, d types.Dict, dictName string, sinceVersion model.Version) (err error) {
	defer func() {
		err = mediaEntryError(d, "P", err)
	}()

	// see table 275
	d1, err := validateDictEntry(xRefTable, d, 0, dictName, "P", OPTIONAL, sinceVersion, nil)
	if err != nil || d1 == nil {
		return err
	}

	dictName = "mediaPermissionDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, d1, 0, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaPermissions" })
	if err != nil {
		return err
	}

	// TF, optional, ASCII string
	validateTempFilePolicy := func(s string) bool {
		return types.MemberOf(s, []string{"TEMPNEVER", "TEMPEXTRACT", "TEMPACCESS", "TEMPALWAYS"})
	}
	_, err = validateStringEntry(xRefTable, d1, 0, dictName, "TF", OPTIONAL, sinceVersion, validateTempFilePolicy)

	return err
}

func validateMediaPlayerInfoDict(xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version) error {
	// see table 291

	dictName := "mediaPlayerInfoDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, 0, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaPlayerInfo" })
	if err != nil {
		return err
	}

	// PID, required, software identifier dict
	d1, err := validateDictEntry(xRefTable, d, 0, dictName, "PID", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}
	err = validateSoftwareIdentifierDict(xRefTable, d1, sinceVersion)
	if err != nil {
		return mediaEntryError(d, "PID", err)
	}

	// MH, optional, dict
	_, err = validateDictEntry(xRefTable, d, 0, dictName, "MH", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// BE, optional, dict
	_, err = validateDictEntry(xRefTable, d, 0, dictName, "BE", OPTIONAL, sinceVersion, nil)

	return err
}

func validateMediaPlayersDict(xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version) (err error) {
	// see 13.2.7.2

	dictName := "mediaPlayersDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, d, 0, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaPlayers" })
	if err != nil {
		return err
	}

	// MU, optional, array of media player info dicts
	arrayObjNr := validationEntryObjectNumber(0, d, "MU")
	defer func() {
		err = model.WithValidationErrorObject(err, arrayObjNr)
	}()

	a, err := validateArrayEntry(xRefTable, d, 0, dictName, "MU", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.MU: %w", dictName, err)
	}

	for i, v := range a {
		objNr := validationObjectNumber(arrayObjNr, v)

		if v == nil {
			continue
		}

		d, err := xRefTable.DereferenceDict(v)
		if err != nil {
			err = fmt.Errorf("%s.MU[%d]: dereference media player info dict: %w", dictName, i, err)
			return model.WithValidationErrorObject(err, objNr)
		}

		if d == nil {
			continue
		}

		err = validateMediaPlayerInfoDict(xRefTable, d, sinceVersion)
		if err != nil {
			err = fmt.Errorf("%s.MU[%d]: %w", dictName, i, err)
			return model.WithValidationErrorObject(err, objNr)
		}

	}

	return nil

}

func validateFileSpecOrFormXObjectEntry(c context.Context, xRefTable *model.XRefTable, d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version) error {
	rawEntry := d[entryName]
	o, err := validateEntry(xRefTable, d, 0, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		if err != nil {
			return fmt.Errorf("%s: %w", dictEntryContext(dictName, entryName, rawEntry), err)
		}
		return nil
	}

	if err := validateFileSpecificationOrFormObject(c, xRefTable, rawEntry); err != nil {
		return fmt.Errorf("%s: %w", dictEntryContext(dictName, entryName, rawEntry), err)
	}
	return nil
}

func validateMediaClipDataDict(c context.Context, xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version) error {
	// see 13.2.4.2

	dictName := "mediaClipDataDict"

	// D, required, file specification or stream
	err := validateFileSpecOrFormXObjectEntry(c, xRefTable, d, dictName, "D", REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	// CT, optional, ASCII string
	_, err = validateStringEntry(xRefTable, d, 0, dictName, "CT", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// P, optional, media permissions dict
	err = validateMediaPermissionsDict(xRefTable, d, dictName, sinceVersion)
	if err != nil {
		return err
	}

	// Alt, optional, string array
	err = validateMultiLanguageTextEntry(xRefTable, d, 0, dictName, "Alt", sinceVersion)
	if err != nil {
		return err
	}

	// PL, optional, media players dict
	d1, err := validateDictEntry(xRefTable, d, 0, dictName, "PL", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.PL: %w", dictName, err)
	}
	if d1 != nil {
		err = validateMediaPlayersDict(xRefTable, d1, sinceVersion)
		if err != nil {
			err = fmt.Errorf("%s.PL: %w", dictName, err)
			return mediaEntryError(d, "PL", err)
		}
	}

	// MH, optional, dict
	d1, err = validateDictEntry(xRefTable, d, 0, dictName, "MH", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		// BU, optional, ASCII string
		_, err = validateStringEntry(xRefTable, d1, 0, "", "BU", OPTIONAL, sinceVersion, nil)
		if err != nil {
			return mediaEntryError(d, "MH", err)
		}
	}

	// BE. optional, dict
	d1, err = validateDictEntry(xRefTable, d, 0, dictName, "BE", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		// BU, optional, ASCII string
		_, err = validateStringEntry(xRefTable, d1, 0, "", "BU", OPTIONAL, sinceVersion, nil)
		return mediaEntryError(d, "BE", err)
	}

	return err
}

func validateTimespanDict(xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version) error {
	dictName := "timespanDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, 0, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Timespan" })
	if err != nil {
		return err
	}

	// S, required, name
	_, err = validateNameEntry(xRefTable, d, 0, dictName, "S", REQUIRED, sinceVersion, func(s string) bool { return s == "S" })
	if err != nil {
		return err
	}

	// V, required, number
	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "V", REQUIRED, sinceVersion, nil)

	return err
}

func validateMediaOffsetDict(xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version) error {
	// see 13.2.6.2

	dictName := "mediaOffsetDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, 0, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaOffset" })
	if err != nil {
		return err
	}

	// S, required, name
	subType, err := validateNameEntry(xRefTable, d, 0, dictName, "S", REQUIRED, sinceVersion, func(s string) bool { return types.MemberOf(s, []string{"T", "F", "M"}) })
	if err != nil {
		return err
	}

	switch *subType {

	case "T":
		d1, err := validateDictEntry(xRefTable, d, 0, dictName, "T", REQUIRED, sinceVersion, nil)
		if err != nil {
			return err
		}
		err = validateTimespanDict(xRefTable, d1, sinceVersion)
		if err != nil {
			return mediaEntryError(d, "T", err)
		}

	case "F":
		_, err = validateIntegerEntry(xRefTable, d, 0, dictName, "F", REQUIRED, sinceVersion, func(i int) bool { return i >= 0 })
		if err != nil {
			return err
		}

	case "M":
		_, err = validateStringEntry(xRefTable, d, 0, dictName, "M", REQUIRED, sinceVersion, nil)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateMediaClipSectionDictMHBE(xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version) error {
	dictName := "mediaClipSectionMHBE"

	d1, err := validateDictEntry(xRefTable, d, 0, dictName, "B", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateMediaOffsetDict(xRefTable, d1, sinceVersion)
		if err != nil {
			return mediaEntryError(d, "B", err)
		}
	}

	d1, err = validateDictEntry(xRefTable, d, 0, dictName, "E", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateMediaOffsetDict(xRefTable, d1, sinceVersion)
		return mediaEntryError(d, "E", err)
	}

	return err
}

type mediaClipTraversal struct {
	c         context.Context
	xRefTable *model.XRefTable
	ancestors map[int]bool
}

func newMediaClipTraversal(c context.Context, xRefTable *model.XRefTable) *mediaClipTraversal {
	return &mediaClipTraversal{c: c, xRefTable: xRefTable, ancestors: map[int]bool{}}
}

func mediaClipObjectIdentity(o types.Object) int {
	ir, ok := o.(types.IndirectRef)
	if !ok {
		return 0
	}
	return ir.ObjectNumber.Value()
}

func (t *mediaClipTraversal) enter(objNr int) error {
	if objNr <= 0 {
		return nil
	}
	if t.ancestors[objNr] {
		return model.ErrMediaClipCycle
	}
	t.ancestors[objNr] = true
	return nil
}

func (t *mediaClipTraversal) leave(objNr int) {
	if objNr > 0 {
		delete(t.ancestors, objNr)
	}
}

func mediaClipEntryError(d types.Dict, dictName, entryName string, rawEntry types.Object, err error) error {
	err = fmt.Errorf("%s: %w", dictEntryContext(dictName, entryName, rawEntry), err)
	return mediaEntryError(d, entryName, err)
}

func (t *mediaClipTraversal) validateEntry(d types.Dict, dictName, entryName string, required bool, sinceVersion model.Version, depth int) error {
	rawEntry, found := d.Find(entryName)
	if err := contextutil.Check(t.c); err != nil {
		return mediaClipEntryError(d, dictName, entryName, rawEntry, err)
	}

	mediaClipObjNr := 0
	if found && rawEntry != nil {
		if err := t.xRefTable.CheckRecursionDepth("MediaClip chain", depth); err != nil {
			return mediaClipEntryError(d, dictName, entryName, rawEntry, err)
		}
		mediaClipObjNr = mediaClipObjectIdentity(rawEntry)
		if err := t.enter(mediaClipObjNr); err != nil {
			return mediaClipEntryError(d, dictName, entryName, rawEntry, err)
		}
		defer t.leave(mediaClipObjNr)
	}

	d1, err := validateDictEntry(t.xRefTable, d, 0, dictName, entryName, required, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", dictEntryContext(dictName, entryName, rawEntry), err)
	}
	if d1 == nil {
		return nil
	}
	if err = t.validateDict(d1, sinceVersion, depth); err != nil {
		return mediaClipEntryError(d, dictName, entryName, rawEntry, err)
	}
	return nil
}

func (t *mediaClipTraversal) validateSectionDict(d types.Dict, sinceVersion model.Version, depth int) error {
	// see 13.2.4.3

	dictName := "mediaClipSectionDict"
	xRefTable := t.xRefTable

	// D, required, media clip dict
	if err := t.validateEntry(d, dictName, "D", REQUIRED, sinceVersion, depth+1); err != nil {
		return err
	}

	// Alt, optional, string array
	err := validateMultiLanguageTextEntry(xRefTable, d, 0, dictName, "Alt", sinceVersion)
	if err != nil {
		return err
	}

	// MH, optional, dict
	d1, err := validateDictEntry(xRefTable, d, 0, dictName, "MH", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateMediaClipSectionDictMHBE(xRefTable, d1, sinceVersion)
		if err != nil {
			return mediaEntryError(d, "MH", err)
		}
	}

	// BE, optional, dict
	d1, err = validateDictEntry(xRefTable, d, 0, dictName, "BE", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateMediaClipSectionDictMHBE(xRefTable, d1, sinceVersion)
		return mediaEntryError(d, "BE", err)
	}

	return err
}

func (t *mediaClipTraversal) validateDict(d types.Dict, sinceVersion model.Version, depth int) error {
	// see 13.2.4

	dictName := "mediaClipDict"
	xRefTable := t.xRefTable

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, 0, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaClip" })
	if err != nil {
		return err
	}

	// S, required, name
	subType, err := validateNameEntry(xRefTable, d, 0, dictName, "S", REQUIRED, sinceVersion, func(s string) bool { return s == "MCD" || s == "MCS" })
	if err != nil {
		return err
	}

	// N, optional, text string
	_, err = validateStringEntry(xRefTable, d, 0, dictName, "N", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	if *subType == "MCD" {
		err = validateMediaClipDataDict(t.c, xRefTable, d, sinceVersion)
		if err != nil {
			return fmt.Errorf("%s data: %w", dictName, err)
		}
	}

	if *subType == "MCS" {
		err = t.validateSectionDict(d, sinceVersion, depth)
		if err != nil {
			return fmt.Errorf("%s section: %w", dictName, err)
		}
	}

	return nil
}

func validateMediaDurationDict(xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version) error {
	dictName := "mediaDurationDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, 0, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaDuration" })
	if err != nil {
		return err
	}

	// S, required, name
	validate := func(s string) bool { return types.MemberOf(s, []string{"I", "F", "T"}) }
	s, err := validateNameEntry(xRefTable, d, 0, dictName, "S", REQUIRED, sinceVersion, validate)
	if err != nil {
		return err
	}

	// T, required if S == "T", timespann dict
	d1, err := validateDictEntry(xRefTable, d, 0, dictName, "T", *s == "T", sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateTimespanDict(xRefTable, d1, sinceVersion)
		return mediaEntryError(d, "T", err)
	}

	return err
}

func validateMediaPlayParamsMHBEDict(xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version) error {
	dictName := "mediaPlayParamsMHBEDict"

	// V, optional, integer
	_, err := validateIntegerEntry(xRefTable, d, 0, dictName, "V", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// C, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "C", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// F, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, 0, dictName, "RT", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// D, optional, media duration dict
	d1, err := validateDictEntry(xRefTable, d, 0, dictName, "D", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateMediaDurationDict(xRefTable, d1, sinceVersion)
		if err != nil {
			return mediaEntryError(d, "D", err)
		}
	}

	// A, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "A", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// RC, optional, number
	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "RC", OPTIONAL, sinceVersion, nil)

	return err
}

func validateMediaPlayParamsDict(xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version) error {
	// see 13.2.5

	dictName := "mediaPlayParamsDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, 0, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaPlayParams" })
	if err != nil {
		return err
	}

	// PL, optional, media players dict
	d1, err := validateDictEntry(xRefTable, d, 0, dictName, "PL", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateMediaPlayersDict(xRefTable, d1, sinceVersion)
		if err != nil {
			return mediaEntryError(d, "PL", err)
		}
	}

	// MH, optional, dict
	d1, err = validateDictEntry(xRefTable, d, 0, dictName, "MH", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateMediaPlayParamsMHBEDict(xRefTable, d1, sinceVersion)
		if err != nil {
			return mediaEntryError(d, "MH", err)
		}
	}

	// BE, optional, dict
	d1, err = validateDictEntry(xRefTable, d, 0, dictName, "BE", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateMediaPlayParamsMHBEDict(xRefTable, d1, sinceVersion)
		return mediaEntryError(d, "BE", err)
	}

	return err
}

func validateFloatingWindowsParameterDict(xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version) error {
	// see table 284

	dictName := "floatWinParamsDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, 0, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "FWParams" })
	if err != nil {
		return err
	}

	// D, required, array of integers
	_, err = validateIntegerArrayEntry(xRefTable, d, 0, dictName, "D", REQUIRED, sinceVersion, func(a types.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// RT, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, 0, dictName, "RT", OPTIONAL, sinceVersion, func(i int) bool { return types.IntMemberOf(i, []int{0, 1, 2, 3}) })
	if err != nil {
		return err
	}

	// P, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, 0, dictName, "P", OPTIONAL, sinceVersion, func(i int) bool { return types.IntMemberOf(i, []int{0, 1, 2, 3, 4, 5, 6, 7, 8}) })
	if err != nil {
		return err
	}

	// O, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, 0, dictName, "O", OPTIONAL, sinceVersion, func(i int) bool { return types.IntMemberOf(i, []int{0, 1, 2}) })
	if err != nil {
		return err
	}

	// T, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "T", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// UC, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, 0, dictName, "UC", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// R, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, 0, dictName, "R", OPTIONAL, sinceVersion, func(i int) bool { return types.IntMemberOf(i, []int{0, 1, 2}) })
	if err != nil {
		return err
	}

	// TT, optional, string array
	err = validateMultiLanguageTextEntry(xRefTable, d, 0, dictName, "TT", sinceVersion)

	return err
}

func validateScreenParametersMHBEDict(xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version) error {
	dictName := "screenParmsMHBEDict"

	w := 3

	// W, optional, integer
	i, err := validateIntegerEntry(xRefTable, d, 0, dictName, "W", OPTIONAL, sinceVersion, func(i int) bool { return types.IntMemberOf(i, []int{0, 1, 2, 3}) })
	if err != nil {
		return err
	}
	if i != nil {
		w = (*i).Value()
	}

	// B, optional, array of 3 numbers
	_, err = validateNumberArrayEntry(xRefTable, d, 0, dictName, "B", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	// O, optional, number
	_, err = validateNumberEntry(xRefTable, d, 0, dictName, "O", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// M, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, 0, dictName, "M", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// F, required if W == 0, floating windows parameter dict
	d1, err := validateDictEntry(xRefTable, d, 0, dictName, "F", w == 0, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateFloatingWindowsParameterDict(xRefTable, d1, sinceVersion)
		return mediaEntryError(d, "F", err)
	}

	return err
}

func validateScreenParametersDict(xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version) error {
	// see 13.2.

	dictName := "screenParmsDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, 0, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaScreenParams" })
	if err != nil {
		return err
	}

	// MH, optional, dict
	d1, err := validateDictEntry(xRefTable, d, 0, dictName, "MH", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateScreenParametersMHBEDict(xRefTable, d1, sinceVersion)
		if err != nil {
			return mediaEntryError(d, "MH", err)
		}
	}

	// BE. optional. dict
	d1, err = validateDictEntry(xRefTable, d, 0, dictName, "BE", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateScreenParametersMHBEDict(xRefTable, d1, sinceVersion)
		return mediaEntryError(d, "BE", err)
	}

	return err
}

func validateMediaRenditionDict(c context.Context, xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version) error {
	// table 271

	dictName := "mediaRendDict"

	// C, optional, dict
	if err := newMediaClipTraversal(c, xRefTable).validateEntry(
		d, dictName, "C", OPTIONAL, sinceVersion, 0,
	); err != nil {
		return err
	}

	// P, required if C not present, dict
	rawEntry := d["P"]
	d1, err := validateDictEntry(xRefTable, d, 0, dictName, "P", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", dictEntryContext(dictName, "P", rawEntry), err)
	}
	if d1 != nil {
		err = validateMediaPlayParamsDict(xRefTable, d1, sinceVersion)
		if err != nil {
			err = fmt.Errorf("%s: %w", dictEntryContext(dictName, "P", rawEntry), err)
			return mediaEntryError(d, "P", err)
		}
	}

	// SP, optional, dict
	rawEntry = d["SP"]
	d1, err = validateDictEntry(xRefTable, d, 0, dictName, "SP", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", dictEntryContext(dictName, "SP", rawEntry), err)
	}
	if d1 != nil {
		err = validateScreenParametersDict(xRefTable, d1, sinceVersion)
		if err != nil {
			err = fmt.Errorf("%s: %w", dictEntryContext(dictName, "SP", rawEntry), err)
			return mediaEntryError(d, "SP", err)
		}
	}

	return nil
}

func validateSelectorRenditionDict(c context.Context, xRefTable *model.XRefTable, d types.Dict, sinceVersion model.Version, depth int, visit *model.RenditionVisit) (err error) {
	// table 272

	dictName := "selectorRendDict"

	arrayObjNr := validationEntryObjectNumber(0, d, "R")
	defer func() {
		err = model.WithValidationErrorObject(err, arrayObjNr)
	}()

	a, err := validateArrayEntry(xRefTable, d, 0, dictName, "R", REQUIRED, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.R: %w", dictName, err)
	}

	for i, v := range a {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		objNr := validationObjectNumber(arrayObjNr, v)

		if v == nil {
			continue
		}

		d, err := xRefTable.DereferenceDict(v)
		if err != nil {
			err = fmt.Errorf("%s: dereference rendition dict: %w", objectContext(fmt.Sprintf("%s.R[%d]", dictName, i), v), err)
			return model.WithValidationErrorObject(err, objNr)
		}

		if d == nil {
			continue
		}

		err = validateRenditionDictDepth(c, xRefTable, d, validationObjectNumber(0, v), sinceVersion, depth+1, visit)
		if err != nil {
			err = model.WrapRecursionError(objectContext(fmt.Sprintf("%s.R[%d]", dictName, i), v), err)
			return model.WithValidationErrorObject(err, objNr)
		}

	}

	return nil
}

func validateRenditionDictEntryMH(xRefTable *model.XRefTable, d types.Dict, dictName string, sinceVersion model.Version) (err error) {
	defer func() {
		err = mediaEntryError(d, "MH", err)
	}()

	d1, err := validateDictEntry(xRefTable, d, 0, dictName, "MH", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.MH: %w", dictName, err)
	}

	if d1 != nil {

		d2, err := validateDictEntry(xRefTable, d1, 0, "MHDict", "C", OPTIONAL, sinceVersion, nil)
		if err != nil {
			return fmt.Errorf("%s.MH.C: %w", dictName, err)
		}

		if d2 != nil {
			if err := validateMediaCriteriaDict(xRefTable, d2, sinceVersion); err != nil {
				err = fmt.Errorf("%s.MH.C: %w", dictName, err)
				return mediaEntryError(d1, "C", err)
			}
		}

	}

	return nil
}

func validateRenditionDictEntryBE(xRefTable *model.XRefTable, d types.Dict, dictName string, sinceVersion model.Version) (err error) {
	defer func() {
		err = mediaEntryError(d, "BE", err)
	}()

	d1, err := validateDictEntry(xRefTable, d, 0, dictName, "BE", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.BE: %w", dictName, err)
	}

	if d1 != nil {

		d2, err := validateDictEntry(xRefTable, d1, 0, "BEDict", "C", OPTIONAL, sinceVersion, nil)
		if err != nil {
			return fmt.Errorf("%s.BE.C: %w", dictName, err)
		}

		if d2 != nil {
			if err := validateMediaCriteriaDict(xRefTable, d2, sinceVersion); err != nil {
				err = fmt.Errorf("%s.BE.C: %w", dictName, err)
				return mediaEntryError(d1, "C", err)
			}
		}

	}

	return nil
}

func validateRenditionDict(c context.Context, xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, sinceVersion model.Version) (err error) {
	return validateRenditionDictDepth(c, xRefTable, d, ownerObjNr, sinceVersion, 0, model.NewRenditionVisit())
}

func validateRenditionDictDepth(c context.Context, xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, sinceVersion model.Version, depth int, visit *model.RenditionVisit) (err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()

	if err := contextutil.Check(c); err != nil {
		return err
	}
	if err := xRefTable.CheckRecursionDepth("rendition graph", depth); err != nil {
		return err
	}
	if err := visit.Enter(ownerObjNr); err != nil {
		return err
	}
	defer visit.Leave(ownerObjNr)
	if visit.AlreadyValidated(ownerObjNr, depth) {
		return nil
	}

	dictName := "renditionDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, d, 0, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Rendition" })
	if err != nil {
		return fmt.Errorf("%s.Type: %w", dictName, err)
	}

	// S, required, name
	renditionType, err := validateNameEntry(xRefTable, d, 0, dictName, "S", REQUIRED, sinceVersion, func(s string) bool { return s == "MR" || s == "SR" })
	if err != nil {
		return fmt.Errorf("%s.S: %w", dictName, err)
	}

	// N, optional, text string
	_, err = validateStringEntry(xRefTable, d, 0, dictName, "N", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.N: %w", dictName, err)
	}

	// MH, optional, dict
	err = validateRenditionDictEntryMH(xRefTable, d, dictName, sinceVersion)
	if err != nil {
		return err
	}

	// BE, optional, dict
	err = validateRenditionDictEntryBE(xRefTable, d, dictName, sinceVersion)
	if err != nil {
		return err
	}

	if *renditionType == "MR" {
		err = validateMediaRenditionDict(c, xRefTable, d, sinceVersion)
		if err != nil {
			return fmt.Errorf("%s media rendition: %w", dictName, err)
		}
	}

	if *renditionType == "SR" {
		err = validateSelectorRenditionDict(c, xRefTable, d, sinceVersion, depth, visit)
		if err != nil {
			return fmt.Errorf("%s selector rendition: %w", dictName, err)
		}
	}

	visit.MarkValidated(ownerObjNr, depth)
	return nil
}
