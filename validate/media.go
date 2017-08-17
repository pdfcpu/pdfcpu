package validate

import (
	"github.com/EndFirstCorp/pdflib/types"
	"github.com/pkg/errors"
)

func validateMinimumBitDepthDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see table 269

	logInfoValidate.Println("*** validateMinimumBitDepthDict: begin ***")

	dictName := "minBitDepthDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "MinBitDepth" })
	if err != nil {
		return
	}

	// V, required, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "V", REQUIRED, types.V10, func(i int) bool { return i >= 0 })
	if err != nil {
		return
	}

	// M, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "M", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateMinimumBitDepthDict: end ***")

	return
}

func validateMinimumScreenSizeDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see table 269

	logInfoValidate.Println("*** validateMinimumScreenSizeDict: begin ***")

	dictName := "minBitDepthDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "MinScreenSize" })
	if err != nil {
		return
	}

	// V, required, integer array, length 2
	_, err = validateIntegerArrayEntry(xRefTable, dict, dictName, "V", REQUIRED, types.V10, func(a types.PDFArray) bool { return len(a) == 2 })
	if err != nil {
		return
	}

	// M, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "M", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateMinimumScreenSizeDict: end ***")

	return
}

func validateSoftwareIdentifierDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see table 292

	logInfoValidate.Println("*** validateSoftwareIdentifierDict: begin ***")

	dictName := "swIdDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "SoftwareIdentifier" })
	if err != nil {
		return
	}

	// U, required, ASCII string
	_, err = validateStringEntry(xRefTable, dict, dictName, "U", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// L, optional, array
	_, err = validateArrayEntry(xRefTable, dict, dictName, "L", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// LI, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "LI", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// H, optional, array
	_, err = validateArrayEntry(xRefTable, dict, dictName, "H", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// HI, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "HI", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// OS, optional, array
	_, err = validateStringArrayEntry(xRefTable, dict, dictName, "OS", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateSoftwareIdentifierDict: end ***")

	return
}

func validateMediaCriteriaDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see table 268

	logInfoValidate.Println("*** validateMediaCriteriaDict: begin ***")

	dictName := "mediaCritDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "MediaCriteria" })
	if err != nil {
		return
	}

	// A, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "A", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// C, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "C", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// O, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "O", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// S, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "S", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// R, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "R", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	var d *types.PDFDict

	// D, optional, dict

	d, err = validateDictEntry(xRefTable, dict, dictName, "D", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if d != nil {
		err = validateMinimumBitDepthDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	// Z, optional, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "Z", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if d != nil {
		err = validateMinimumScreenSizeDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	// V, optional, array
	var a *types.PDFArray
	a, err = validateArrayEntry(xRefTable, dict, dictName, "V", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if a != nil {
		for _, v := range *a {
			if v == nil {
				continue
			}
			d, err = xRefTable.DereferenceDict(v)
			if err != nil {
				return
			}
			if d != nil {
				err = validateSoftwareIdentifierDict(xRefTable, d)
				if err != nil {
					return
				}
			}
		}
	}

	// P, optional, array
	_, err = validateNameArrayEntry(xRefTable, dict, dictName, "P", OPTIONAL, types.V10, func(a types.PDFArray) bool { return len(a) == 2 })
	if err != nil {
		return
	}

	// L, optional, array
	_, err = validateStringArrayEntry(xRefTable, dict, dictName, "L", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateMediaCriteriaDict: end ***")

	return
}

func validateMediaPermissionsDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see table 275

	logInfoValidate.Println("*** validateMediaPermissionsDict: begin ***")

	dictName := "mediaPermissionDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "MediaPermissions" })
	if err != nil {
		return
	}

	// TF, optional, ASCII string
	validateTempFilePolicy := func(s string) bool {
		return memberOf(s, []string{"TEMPNEVER", "TEMPEXTRACT", "TEMPACCESS", "TEMPALWAYS"})
	}
	_, err = validateStringEntry(xRefTable, dict, dictName, "TF", OPTIONAL, types.V10, validateTempFilePolicy)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateMediaPermissionsDict: end ***")

	return
}

func validateMediaPlayerInfoDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see table 291

	logInfoValidate.Println("*** validateMediaPlayerInfoDict: begin ***")

	dictName := "mediaPlayerInfoDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "MediaPlayerInfo" })
	if err != nil {
		return
	}

	var d *types.PDFDict

	// PID, required, software identifier dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "PID", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}
	err = validateSoftwareIdentifierDict(xRefTable, d)
	if err != nil {
		return
	}

	// MH, optional, dict
	_, err = validateDictEntry(xRefTable, dict, dictName, "MH", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// BE, optional, dict
	_, err = validateDictEntry(xRefTable, dict, dictName, "BE", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateMediaPlayerInfoDict: end ***")

	return
}

func validateMediaPlayersDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 13.2.7.2

	logInfoValidate.Println("*** validateMediaPlayersDict: begin ***")

	dictName := "mediaPlayersDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "MediaPlayers" })
	if err != nil {
		return
	}

	var (
		a *types.PDFArray
		d *types.PDFDict
	)

	// MU, optional, array of media player info dicts
	a, err = validateArrayEntry(xRefTable, dict, dictName, "MU", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if a != nil {
		for _, v := range *a {
			if v == nil {
				continue
			}
			d, err = xRefTable.DereferenceDict(v)
			if err != nil {
				return
			}
			if d == nil {
				continue
			}
			err = validateMediaPlayerInfoDict(xRefTable, d)
			if err != nil {
				return
			}
		}
	}

	logInfoValidate.Println("*** validateMediaPlayersDict: end ***")

	return

}

func validateFileSpecOrFormXObjectEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateFileSpecOrFormXObjectEntry: begin ***")

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("validateFileSpecOrFormXObjectEntry: missing entry \"%s\"", entryName)
		}
		return
	}

	_, err = validateFileSpecificationOrFormObject(xRefTable, obj)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateFileSpecOrFormXObjectEntry: begin ***")

	return
}

func validateMediaClipDataDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 13.2.4.2

	logInfoValidate.Println("*** validateMediaClipDataDict: begin ***")

	dictName := "mediaClipDataDict"

	// D, required, file specification or stream
	err = validateFileSpecOrFormXObjectEntry(xRefTable, dict, dictName, "D", REQUIRED, types.V10)
	if err != nil {
		return
	}

	// CT, optional, ASCII string
	_, err = validateNameEntry(xRefTable, dict, dictName, "CT", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	var d *types.PDFDict

	// P, optional, media permissions dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "P", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if d != nil {
		err = validateMediaPermissionsDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	// Alt, optional, string array
	_, err = validateStringArrayEntry(xRefTable, dict, dictName, "Alt", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// PL, optional, media players dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "PL", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if d != nil {
		err = validateMediaPlayersDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	// MH, optional, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "MH", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if d != nil {
		// BU, optional, ASCII string
		_, err = validateStringEntry(xRefTable, d, "", "BU", OPTIONAL, types.V10, nil)
	}

	// BE. optional, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "BE", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if d != nil {
		// BU, optional, ASCII string
		_, err = validateStringEntry(xRefTable, d, "", "BU", OPTIONAL, types.V10, nil)
	}

	logInfoValidate.Println("*** validateMediaClipDataDict: end ***")

	return
}

func validateTimespanDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validateTimespanDict: begin ***")

	dictName := "timespanDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "Timespan" })
	if err != nil {
		return
	}

	// S, required, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "S", REQUIRED, types.V10, func(s string) bool { return s == "S" })
	if err != nil {
		return
	}

	// V, required, number
	_, err = validateNumberEntry(xRefTable, dict, dictName, "V", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateTimespanDict: end ***")

	return
}

func validateMediaOffsetDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 13.2.6.2

	logInfoValidate.Println("*** validateMediaOffsetDict: begin ***")

	dictName := "mediaOffsetDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "MediaOffset" })
	if err != nil {
		return
	}

	// S, required, name
	subType, err := validateNameEntry(xRefTable, dict, dictName, "S", REQUIRED, types.V10, func(s string) bool { return memberOf(s, []string{"T", "F", "M"}) })
	if err != nil {
		return
	}

	var d *types.PDFDict

	switch *subType {

	case "T":
		d, err = validateDictEntry(xRefTable, dict, dictName, "T", REQUIRED, types.V10, nil)
		if err != nil {
			return
		}
		err = validateTimespanDict(xRefTable, d)
		if err != nil {
			return
		}

	case "F":
		_, err = validateIntegerEntry(xRefTable, dict, dictName, "F", REQUIRED, types.V10, func(i int) bool { return i >= 0 })
		if err != nil {
			return
		}

	case "M":
		_, err = validateStringEntry(xRefTable, dict, dictName, "M", REQUIRED, types.V10, nil)
		if err != nil {
			return
		}

	}

	logInfoValidate.Println("*** validateMediaOffsetDict: end ***")

	return
}

func validateMediaClipSectionDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 13.2.4.3

	logInfoValidate.Println("*** validateMediaClipSectionDict: begin ***")

	dictName := "mediaClipSectionDict"

	var d *types.PDFDict

	// D, required, media clip dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "D", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}
	err = validateMediaClipDict(xRefTable, d)
	if err != nil {
		return
	}

	// Alt, optional, string array
	_, err = validateStringArrayEntry(xRefTable, dict, dictName, "Alt", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// MH, optional, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "MH", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if d != nil {
		err = validateMediaOffsetDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	// BE, optional, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "BE", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if d != nil {
		err = validateMediaOffsetDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	logInfoValidate.Println("*** validateMediaClipSectionDict: end ***")

	return
}

func validateMediaClipDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 13.2.4

	logInfoValidate.Println("*** validateMediaClipDict: begin ***")

	dictName := "mediaClipDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "MediaClip" })
	if err != nil {
		return
	}

	// S, required, name
	subType, err := validateNameEntry(xRefTable, dict, dictName, "S", REQUIRED, types.V10, func(s string) bool { return s == "MCD" || s == "MCS" })
	if err != nil {
		return
	}

	// N, optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "N", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	if *subType == "MCD" {
		err = validateMediaClipDataDict(xRefTable, dict)
		if err != nil {
			return
		}
	}

	if *subType == "MCS" {
		err = validateMediaClipSectionDict(xRefTable, dict)
		if err != nil {
			return
		}
	}

	logInfoValidate.Println("*** validateMediaClipDict: end ***")

	return
}

func validateMediaPlayParametersDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 13.2.5

	logInfoValidate.Println("*** validateMediaPlayParametersDict: begin ***")

	dictName := "mediaPlayParmsDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "MediaPlayParams" })
	if err != nil {
		return
	}

	var d *types.PDFDict

	// PL, optional, media players dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "PL", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if d != nil {
		err = validateMediaPlayersDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	// MH, optional, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "MH", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if d != nil {
		err = validateMediaOffsetDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	// BE, optional, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "BE", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if d != nil {
		err = validateMediaOffsetDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	logInfoValidate.Println("*** validateMediaPlayParametersDict: end ***")

	return
}

func validateFloatingWindowsParameterDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see table 284

	logInfoValidate.Println("*** validateFloatingWindowsParameterDict: begin ***")

	dictName := "floatWinParamsDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "FWParams" })
	if err != nil {
		return
	}

	// D, required, array of integers
	_, err = validateIntegerArrayEntry(xRefTable, dict, dictName, "D", REQUIRED, types.V10, func(a types.PDFArray) bool { return len(a) == 2 })
	if err != nil {
		return
	}

	// RT, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "RT", OPTIONAL, types.V10, func(i int) bool { return intMemberOf(i, []int{0, 1, 2, 3}) })
	if err != nil {
		return
	}

	// P, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "P", OPTIONAL, types.V10, func(i int) bool { return intMemberOf(i, []int{0, 1, 2, 3, 4, 5, 6, 7, 8}) })
	if err != nil {
		return
	}

	// O, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "O", OPTIONAL, types.V10, func(i int) bool { return intMemberOf(i, []int{0, 1, 2}) })
	if err != nil {
		return
	}

	// T, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "T", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// UC, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "UC", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// R, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "R", OPTIONAL, types.V10, func(i int) bool { return intMemberOf(i, []int{0, 1, 2}) })
	if err != nil {
		return
	}

	// TT, optional, string array
	_, err = validateStringArrayEntry(xRefTable, dict, dictName, "TT", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateFloatingWindowsParameterDict: end ***")

	return
}

func validateScreenParametersMHBEDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validateScreenParametersMHBEDict: begin ***")

	dictName := "screenParmsMHBEDict"

	w := 3

	// W, optional, integer
	i, err := validateIntegerEntry(xRefTable, dict, dictName, "W", OPTIONAL, types.V10, func(i int) bool { return intMemberOf(i, []int{0, 1, 2, 3}) })
	if err != nil {
		return
	}
	if i != nil {
		w = (*i).Value()
	}

	// B, optional, array of 3 numbers
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "B", OPTIONAL, types.V10, func(a types.PDFArray) bool { return len(a) == 3 })
	if err != nil {
		return
	}

	// O, optional, number
	_, err = validateNumberEntry(xRefTable, dict, dictName, "O", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// M, optional, integer
	_, err = validateIntegerEntry(xRefTable, dict, dictName, "M", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	var d *types.PDFDict

	// F, required if W == 0, floating windows parameter dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "F", w == 0, types.V10, nil)
	if err != nil {
		return
	}
	if d != nil {
		err = validateFloatingWindowsParameterDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	logInfoValidate.Println("*** validateScreenParametersMHBEDict: end ***")

	return
}

func validateScreenParametersDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 13.2.

	logInfoValidate.Println("*** validateScreenParametersDict: begin ***")

	dictName := "screenParmsDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "MediaScreenParams" })
	if err != nil {
		return
	}

	var d *types.PDFDict

	// MH, optional, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "MH", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if d != nil {
		err = validateScreenParametersMHBEDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	// BE. optional. dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "BE", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if d != nil {
		err = validateScreenParametersMHBEDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	logInfoValidate.Println("*** validateScreenParametersDict: end ***")

	return
}

func validateMediaRenditionDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// table 271

	logInfoValidate.Println("*** validateMediaRenditionDict: begin ***")

	dictName := "mediaRendDict"

	var d *types.PDFDict

	// C, optional, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "C", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if d != nil {
		err = validateMediaClipDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	// P, required if C not present, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "P", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if d != nil {
		err = validateMediaPlayParametersDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	// SP, optional, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "SP", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if d != nil {
		err = validateScreenParametersDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	logInfoValidate.Println("*** validateMediaRenditionDict: end ***")

	return
}

func validateSelectorRenditionDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// table 272

	logInfoValidate.Println("*** validateSelectorRenditionDict: begin ***")

	dictName := "selectorRendDict"

	var a *types.PDFArray

	a, err = validateArrayEntry(xRefTable, dict, dictName, "R", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	var d *types.PDFDict

	for _, v := range *a {

		if v == nil {
			continue
		}

		d, err = xRefTable.DereferenceDict(v)
		if err != nil {
			return
		}

		if d == nil {
			continue
		}

		err = validateRenditionDict(xRefTable, d)
		if err != nil {
			return
		}

	}

	logInfoValidate.Println("*** validateSelectorRenditionDict: end ***")

	return
}

func validateRenditionDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Printf("*** validateRenditionDict: begin ***")

	dictName := "renditionDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "Rendition" })
	if err != nil {
		return
	}

	// S, required, name
	renditionType, err := validateNameEntry(xRefTable, dict, dictName, "S", REQUIRED, types.V10, func(s string) bool { return s == "MR" || s == "SR" })
	if err != nil {
		return
	}

	// N, optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "N", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// MH, optional, dict
	var d *types.PDFDict
	d, err = validateDictEntry(xRefTable, dict, dictName, "MH", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if d != nil {
		d, err = validateDictEntry(xRefTable, d, "MHDict", "C", OPTIONAL, types.V10, nil)
		if err != nil {
			return
		}
		if d != nil {
			err = validateMediaCriteriaDict(xRefTable, d)
			if err != nil {
				return
			}
		}
	}

	// BE, optional, dict
	d, err = validateDictEntry(xRefTable, dict, dictName, "BE", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if d != nil {
		d, err = validateDictEntry(xRefTable, d, "BEDict", "C", OPTIONAL, types.V10, nil)
		if err != nil {
			return
		}
		err = validateMediaCriteriaDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	if *renditionType == "MR" {
		err = validateMediaRenditionDict(xRefTable, dict)
		if err != nil {
			return
		}
	}

	if *renditionType == "SR" {
		err = validateSelectorRenditionDict(xRefTable, dict)
		if err != nil {
			return
		}
	}

	logInfoValidate.Printf("*** validateRenditionDict: end ***")

	return
}
