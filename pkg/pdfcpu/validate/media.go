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

import pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"

func validateMinimumBitDepthDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	// see table 269

	dictName := "minBitDepthDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MinBitDepth" })
	if err != nil {
		return err
	}

	// V, required, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "V", REQUIRED, sinceVersion, func(i int) bool { return i >= 0 })
	if err != nil {
		return err
	}

	// M, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "M", OPTIONAL, sinceVersion, nil)

	return err
}

func validateMinimumScreenSizeDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	// see table 269

	dictName := "minBitDepthDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MinScreenSize" })
	if err != nil {
		return err
	}

	// V, required, integer array, length 2
	_, err = validateIntegerArrayEntry(xRefTable, d, dictName, "V", REQUIRED, sinceVersion, func(a pdf.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// M, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "M", OPTIONAL, sinceVersion, nil)

	return err
}

func validateSoftwareIdentifierDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	// see table 292

	dictName := "swIdDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "SoftwareIdentifier" })
	if err != nil {
		return err
	}

	// U, required, ASCII string
	_, err = validateStringEntry(xRefTable, d, dictName, "U", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	// L, optional, array
	_, err = validateArrayEntry(xRefTable, d, dictName, "L", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// LI, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "LI", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// H, optional, array
	_, err = validateArrayEntry(xRefTable, d, dictName, "H", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// HI, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "HI", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// OS, optional, array
	_, err = validateStringArrayEntry(xRefTable, d, dictName, "OS", OPTIONAL, sinceVersion, nil)

	return err
}

func validateMediaCriteriaDictEntryD(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, required bool, sinceVersion pdf.Version) error {

	d1, err := validateDictEntry(xRefTable, d, dictName, "D", required, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = validateMinimumBitDepthDict(xRefTable, d1, sinceVersion)
	}

	return err
}

func validateMediaCriteriaDictEntryZ(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, required bool, sinceVersion pdf.Version) error {

	d1, err := validateDictEntry(xRefTable, d, dictName, "Z", required, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d1 != nil {
		err = validateMinimumScreenSizeDict(xRefTable, d1, sinceVersion)
	}

	return err
}

func validateMediaCriteriaDictEntryV(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, required bool, sinceVersion pdf.Version) error {

	a, err := validateArrayEntry(xRefTable, d, dictName, "V", required, sinceVersion, nil)
	if err != nil {
		return err
	}

	if a != nil {

		for _, v := range a {

			if v == nil {
				continue
			}

			d, err := xRefTable.DereferenceDict(v)
			if err != nil {
				return err
			}

			if d != nil {
				err = validateSoftwareIdentifierDict(xRefTable, d, sinceVersion)
				if err != nil {
					return err
				}
			}

		}

	}

	return nil
}

func validateMediaCriteriaDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	// see table 268

	dictName := "mediaCritDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaCriteria" })
	if err != nil {
		return err
	}

	// A, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "A", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// C, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "C", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// O, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "O", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// S, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "S", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// R, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "R", OPTIONAL, sinceVersion, nil)
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
	_, err = validateNameArrayEntry(xRefTable, d, dictName, "P", OPTIONAL, sinceVersion, func(a pdf.Array) bool { return len(a) == 1 || len(a) == 2 })
	if err != nil {
		return err
	}

	// L, optional, array
	_, err = validateStringArrayEntry(xRefTable, d, dictName, "L", OPTIONAL, sinceVersion, nil)

	return err
}

func validateMediaPermissionsDict(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, sinceVersion pdf.Version) error {

	// see table 275
	d1, err := validateDictEntry(xRefTable, d, dictName, "P", OPTIONAL, sinceVersion, nil)
	if err != nil || d1 == nil {
		return err
	}

	dictName = "mediaPermissionDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, d1, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaPermissions" })
	if err != nil {
		return err
	}

	// TF, optional, ASCII string
	validateTempFilePolicy := func(s string) bool {
		return pdf.MemberOf(s, []string{"TEMPNEVER", "TEMPEXTRACT", "TEMPACCESS", "TEMPALWAYS"})
	}
	_, err = validateStringEntry(xRefTable, d1, dictName, "TF", OPTIONAL, sinceVersion, validateTempFilePolicy)

	return err
}

func validateMediaPlayerInfoDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	// see table 291

	dictName := "mediaPlayerInfoDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaPlayerInfo" })
	if err != nil {
		return err
	}

	// PID, required, software identifier dict
	d1, err := validateDictEntry(xRefTable, d, dictName, "PID", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}
	err = validateSoftwareIdentifierDict(xRefTable, d1, sinceVersion)
	if err != nil {
		return err
	}

	// MH, optional, dict
	_, err = validateDictEntry(xRefTable, d, dictName, "MH", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// BE, optional, dict
	_, err = validateDictEntry(xRefTable, d, dictName, "BE", OPTIONAL, sinceVersion, nil)

	return err
}

func validateMediaPlayersDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	// see 13.2.7.2

	dictName := "mediaPlayersDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaPlayers" })
	if err != nil {
		return err
	}

	// MU, optional, array of media player info dicts
	a, err := validateArrayEntry(xRefTable, d, dictName, "MU", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	if a != nil {

		for _, v := range a {

			if v == nil {
				continue
			}

			d, err := xRefTable.DereferenceDict(v)
			if err != nil {
				return err
			}

			if d == nil {
				continue
			}

			err = validateMediaPlayerInfoDict(xRefTable, d, sinceVersion)
			if err != nil {
				return err
			}

		}

	}

	return nil

}

func validateFileSpecOrFormXObjectEntry(xRefTable *pdf.XRefTable, d pdf.Dict, dictName, entryName string, required bool, sinceVersion pdf.Version) error {

	o, err := validateEntry(xRefTable, d, dictName, entryName, required, sinceVersion)
	if err != nil || o == nil {
		return err
	}

	return validateFileSpecificationOrFormObject(xRefTable, o)
}

func validateMediaClipDataDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	// see 13.2.4.2

	dictName := "mediaClipDataDict"

	// D, required, file specification or stream
	err := validateFileSpecOrFormXObjectEntry(xRefTable, d, dictName, "D", REQUIRED, sinceVersion)
	if err != nil {
		return err
	}

	// CT, optional, ASCII string
	_, err = validateStringEntry(xRefTable, d, dictName, "CT", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// P, optional, media permissions dict
	err = validateMediaPermissionsDict(xRefTable, d, dictName, sinceVersion)
	if err != nil {
		return err
	}

	// Alt, optional, string array
	_, err = validateStringArrayEntry(xRefTable, d, dictName, "Alt", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// PL, optional, media players dict
	d1, err := validateDictEntry(xRefTable, d, dictName, "PL", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateMediaPlayersDict(xRefTable, d1, sinceVersion)
		if err != nil {
			return err
		}
	}

	// MH, optional, dict
	d1, err = validateDictEntry(xRefTable, d, dictName, "MH", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		// BU, optional, ASCII string
		_, err = validateStringEntry(xRefTable, d1, "", "BU", OPTIONAL, sinceVersion, nil)
		if err != nil {
			return err
		}
	}

	// BE. optional, dict
	d1, err = validateDictEntry(xRefTable, d, dictName, "BE", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		// BU, optional, ASCII string
		_, err = validateStringEntry(xRefTable, d1, "", "BU", OPTIONAL, sinceVersion, nil)
	}

	return err
}

func validateTimespanDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	dictName := "timespanDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Timespan" })
	if err != nil {
		return err
	}

	// S, required, name
	_, err = validateNameEntry(xRefTable, d, dictName, "S", REQUIRED, sinceVersion, func(s string) bool { return s == "S" })
	if err != nil {
		return err
	}

	// V, required, number
	_, err = validateNumberEntry(xRefTable, d, dictName, "V", REQUIRED, sinceVersion, nil)

	return err
}

func validateMediaOffsetDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	// see 13.2.6.2

	dictName := "mediaOffsetDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaOffset" })
	if err != nil {
		return err
	}

	// S, required, name
	subType, err := validateNameEntry(xRefTable, d, dictName, "S", REQUIRED, sinceVersion, func(s string) bool { return pdf.MemberOf(s, []string{"T", "F", "M"}) })
	if err != nil {
		return err
	}

	switch *subType {

	case "T":
		d1, err := validateDictEntry(xRefTable, d, dictName, "T", REQUIRED, sinceVersion, nil)
		if err != nil {
			return err
		}
		err = validateTimespanDict(xRefTable, d1, sinceVersion)
		if err != nil {
			return err
		}

	case "F":
		_, err = validateIntegerEntry(xRefTable, d, dictName, "F", REQUIRED, sinceVersion, func(i int) bool { return i >= 0 })
		if err != nil {
			return err
		}

	case "M":
		_, err = validateStringEntry(xRefTable, d, dictName, "M", REQUIRED, sinceVersion, nil)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateMediaClipSectionDictMHBE(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	dictName := "mediaClipSectionMHBE"

	d1, err := validateDictEntry(xRefTable, d, dictName, "B", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateMediaOffsetDict(xRefTable, d1, sinceVersion)
		if err != nil {
			return err
		}
	}

	d1, err = validateDictEntry(xRefTable, d, dictName, "E", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateMediaOffsetDict(xRefTable, d1, sinceVersion)
	}

	return err
}

func validateMediaClipSectionDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	// see 13.2.4.3

	dictName := "mediaClipSectionDict"

	// D, required, media clip dict
	d1, err := validateDictEntry(xRefTable, d, dictName, "D", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}
	err = validateMediaClipDict(xRefTable, d1, sinceVersion)
	if err != nil {
		return err
	}

	// Alt, optional, string array
	_, err = validateStringArrayEntry(xRefTable, d, dictName, "Alt", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// MH, optional, dict
	d1, err = validateDictEntry(xRefTable, d, dictName, "MH", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateMediaClipSectionDictMHBE(xRefTable, d1, sinceVersion)
		if err != nil {
			return err
		}
	}

	// BE, optional, dict
	d1, err = validateDictEntry(xRefTable, d, dictName, "BE", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateMediaClipSectionDictMHBE(xRefTable, d1, sinceVersion)
	}

	return err
}

func validateMediaClipDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	// see 13.2.4

	dictName := "mediaClipDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaClip" })
	if err != nil {
		return err
	}

	// S, required, name
	subType, err := validateNameEntry(xRefTable, d, dictName, "S", REQUIRED, sinceVersion, func(s string) bool { return s == "MCD" || s == "MCS" })
	if err != nil {
		return err
	}

	// N, optional, text string
	_, err = validateStringEntry(xRefTable, d, dictName, "N", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	if *subType == "MCD" {
		err = validateMediaClipDataDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}
	}

	if *subType == "MCS" {
		err = validateMediaClipSectionDict(xRefTable, d, sinceVersion)
	}

	return err
}

func validateMediaDurationDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	dictName := "mediaDurationDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaDuration" })
	if err != nil {
		return err
	}

	// S, required, name
	validate := func(s string) bool { return pdf.MemberOf(s, []string{"I", "F", "T"}) }
	s, err := validateNameEntry(xRefTable, d, dictName, "S", REQUIRED, sinceVersion, validate)
	if err != nil {
		return err
	}

	// T, required if S == "T", timespann dict
	d1, err := validateDictEntry(xRefTable, d, dictName, "T", *s == "T", sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateTimespanDict(xRefTable, d1, sinceVersion)
	}

	return err
}

func validateMediaPlayParamsMHBEDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	dictName := "mediaPlayParamsMHBEDict"

	// V, optional, integer
	_, err := validateIntegerEntry(xRefTable, d, dictName, "V", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// C, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "C", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// F, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "RT", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// D, optional, media duration dict
	d1, err := validateDictEntry(xRefTable, d, dictName, "D", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateMediaDurationDict(xRefTable, d1, sinceVersion)
		if err != nil {
			return err
		}
	}

	// A, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "A", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// RC, optional, number
	_, err = validateNumberEntry(xRefTable, d, dictName, "RC", OPTIONAL, sinceVersion, nil)

	return err
}

func validateMediaPlayParamsDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	// see 13.2.5

	dictName := "mediaPlayParamsDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaPlayParams" })
	if err != nil {
		return err
	}

	// PL, optional, media players dict
	d1, err := validateDictEntry(xRefTable, d, dictName, "PL", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateMediaPlayersDict(xRefTable, d1, sinceVersion)
		if err != nil {
			return err
		}
	}

	// MH, optional, dict
	d1, err = validateDictEntry(xRefTable, d, dictName, "MH", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateMediaPlayParamsMHBEDict(xRefTable, d1, sinceVersion)
		if err != nil {
			return err
		}
	}

	// BE, optional, dict
	d1, err = validateDictEntry(xRefTable, d, dictName, "BE", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateMediaPlayParamsMHBEDict(xRefTable, d1, sinceVersion)
	}

	return err
}

func validateFloatingWindowsParameterDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	// see table 284

	dictName := "floatWinParamsDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "FWParams" })
	if err != nil {
		return err
	}

	// D, required, array of integers
	_, err = validateIntegerArrayEntry(xRefTable, d, dictName, "D", REQUIRED, sinceVersion, func(a pdf.Array) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// RT, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "RT", OPTIONAL, sinceVersion, func(i int) bool { return pdf.IntMemberOf(i, []int{0, 1, 2, 3}) })
	if err != nil {
		return err
	}

	// P, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "P", OPTIONAL, sinceVersion, func(i int) bool { return pdf.IntMemberOf(i, []int{0, 1, 2, 3, 4, 5, 6, 7, 8}) })
	if err != nil {
		return err
	}

	// O, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "O", OPTIONAL, sinceVersion, func(i int) bool { return pdf.IntMemberOf(i, []int{0, 1, 2}) })
	if err != nil {
		return err
	}

	// T, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "T", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// UC, optional, boolean
	_, err = validateBooleanEntry(xRefTable, d, dictName, "UC", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// R, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "R", OPTIONAL, sinceVersion, func(i int) bool { return pdf.IntMemberOf(i, []int{0, 1, 2}) })
	if err != nil {
		return err
	}

	// TT, optional, string array
	_, err = validateStringArrayEntry(xRefTable, d, dictName, "TT", OPTIONAL, sinceVersion, nil)

	return err
}

func validateScreenParametersMHBEDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	dictName := "screenParmsMHBEDict"

	w := 3

	// W, optional, integer
	i, err := validateIntegerEntry(xRefTable, d, dictName, "W", OPTIONAL, sinceVersion, func(i int) bool { return pdf.IntMemberOf(i, []int{0, 1, 2, 3}) })
	if err != nil {
		return err
	}
	if i != nil {
		w = (*i).Value()
	}

	// B, optional, array of 3 numbers
	_, err = validateNumberArrayEntry(xRefTable, d, dictName, "B", OPTIONAL, sinceVersion, func(a pdf.Array) bool { return len(a) == 3 })
	if err != nil {
		return err
	}

	// O, optional, number
	_, err = validateNumberEntry(xRefTable, d, dictName, "O", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// M, optional, integer
	_, err = validateIntegerEntry(xRefTable, d, dictName, "M", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	// F, required if W == 0, floating windows parameter dict
	d1, err := validateDictEntry(xRefTable, d, dictName, "F", w == 0, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateFloatingWindowsParameterDict(xRefTable, d1, sinceVersion)
	}

	return err
}

func validateScreenParametersDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	// see 13.2.

	dictName := "screenParmsDict"

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "MediaScreenParams" })
	if err != nil {
		return err
	}

	// MH, optional, dict
	d1, err := validateDictEntry(xRefTable, d, dictName, "MH", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateScreenParametersMHBEDict(xRefTable, d1, sinceVersion)
		if err != nil {
			return err
		}
	}

	// BE. optional. dict
	d1, err = validateDictEntry(xRefTable, d, dictName, "BE", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateScreenParametersMHBEDict(xRefTable, d1, sinceVersion)
	}

	return err
}

func validateMediaRenditionDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	// table 271

	dictName := "mediaRendDict"

	// C, optional, dict
	d1, err := validateDictEntry(xRefTable, d, dictName, "C", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateMediaClipDict(xRefTable, d1, sinceVersion)
		if err != nil {
			return err
		}
	}

	// P, required if C not present, dict
	d1, err = validateDictEntry(xRefTable, d, dictName, "P", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateMediaPlayParamsDict(xRefTable, d1, sinceVersion)
		if err != nil {
			return err
		}
	}

	// SP, optional, dict
	d1, err = validateDictEntry(xRefTable, d, dictName, "SP", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}
	if d1 != nil {
		err = validateScreenParametersDict(xRefTable, d1, sinceVersion)
	}

	return err
}

func validateSelectorRenditionDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) error {

	// table 272

	dictName := "selectorRendDict"

	a, err := validateArrayEntry(xRefTable, d, dictName, "R", REQUIRED, sinceVersion, nil)
	if err != nil {
		return err
	}

	for _, v := range a {

		if v == nil {
			continue
		}

		d, err := xRefTable.DereferenceDict(v)
		if err != nil {
			return err
		}

		if d == nil {
			continue
		}

		err = validateRenditionDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}

	}

	return nil
}

func validateRenditionDictEntryMH(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, sinceVersion pdf.Version) error {

	d1, err := validateDictEntry(xRefTable, d, dictName, "MH", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d1 != nil {

		d2, err := validateDictEntry(xRefTable, d1, "MHDict", "C", OPTIONAL, sinceVersion, nil)
		if err != nil {
			return err
		}

		if d2 != nil {
			return validateMediaCriteriaDict(xRefTable, d2, sinceVersion)
		}

	}

	return nil
}

func validateRenditionDictEntryBE(xRefTable *pdf.XRefTable, d pdf.Dict, dictName string, sinceVersion pdf.Version) (err error) {

	d1, err := validateDictEntry(xRefTable, d, dictName, "BE", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d1 != nil {

		d2, err := validateDictEntry(xRefTable, d1, "BEDict", "C", OPTIONAL, sinceVersion, nil)
		if err != nil {
			return err
		}

		return validateMediaCriteriaDict(xRefTable, d2, sinceVersion)

	}

	return nil
}

func validateRenditionDict(xRefTable *pdf.XRefTable, d pdf.Dict, sinceVersion pdf.Version) (err error) {

	dictName := "renditionDict"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, d, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Rendition" })
	if err != nil {
		return err
	}

	// S, required, name
	renditionType, err := validateNameEntry(xRefTable, d, dictName, "S", REQUIRED, sinceVersion, func(s string) bool { return s == "MR" || s == "SR" })
	if err != nil {
		return
	}

	// N, optional, text string
	_, err = validateStringEntry(xRefTable, d, dictName, "N", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
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
		err = validateMediaRenditionDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}
	}

	if *renditionType == "SR" {
		err = validateSelectorRenditionDict(xRefTable, d, sinceVersion)
	}

	return err
}
