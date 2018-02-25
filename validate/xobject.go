package validate

import (
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

const (

	// ExcludePatternCS ...
	ExcludePatternCS = true

	// IncludePatternCS ...
	IncludePatternCS = false

	isAlternateImageStreamDict   = true
	isNoAlternateImageStreamDict = false
)

func validateReferenceDictPageEntry(xRefTable *types.XRefTable, obj interface{}) error {

	logInfoValidate.Printf("*** validateReferenceDictPageEntry begin ***")

	obj, err := xRefTable.Dereference(obj)
	if err != nil {
		return err
	}
	if obj == nil {
		logInfoValidate.Println("validateReferenceDictPageEntry end: is nil.")
		return nil
	}

	switch obj.(type) {

	case types.PDFInteger:
		// no further processing

	case types.PDFStringLiteral:
		// no further processing

	case types.PDFHexLiteral:
		// no further processing

	default:
		return errors.New("validateReferenceDictPageEntry: corrupt type")

	}

	logInfoValidate.Println("*** validateReferenceDictPageEntry end ***")

	return nil
}

func validateReferenceDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// see 8.10.4 Reference XObjects

	logInfoValidate.Println("*** validateReferenceDict begin ***")

	// F, file spec, required
	_, err := validateFileSpecEntry(xRefTable, dict, "refDict", "F", REQUIRED, types.V10)
	if err != nil {
		return err
	}

	// Page, integer or text string, required
	obj, ok := dict.Find("Page")
	if !ok {
		return errors.New("validateReferenceDict: missing required entry \"Page\"")
	}

	err = validateReferenceDictPageEntry(xRefTable, obj)
	if err != nil {
		return err
	}

	// ID, string array, optional
	_, err = validateStringArrayEntry(xRefTable, dict, "refDict", "ID", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateReferenceDict end ***")

	return nil
}

func validateOPIDictV13Part1(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string) error {

	// Type, optional, name
	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "OPI" })
	if err != nil {
		return err
	}

	// Version, required, number
	_, err = validateNumberEntry(xRefTable, dict, dictName, "Version", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	// F, required, file specification
	_, err = validateFileSpecEntry(xRefTable, dict, dictName, "F", REQUIRED, types.V10)
	if err != nil {
		return err
	}

	// ID, optional, byte string
	_, err = validateStringEntry(xRefTable, dict, dictName, "ID", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// Comments, optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "Comments", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// Size, required, array of integers, len 2
	_, err = validateIntegerArrayEntry(xRefTable, dict, dictName, "Size", REQUIRED, types.V10, func(a types.PDFArray) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// CropRect, required, array of integers, len 4
	_, err = validateIntegerArrayEntry(xRefTable, dict, dictName, "CropRect", REQUIRED, types.V10, func(a types.PDFArray) bool { return len(a) == 4 })
	if err != nil {
		return err
	}

	// CropFixed, optional, array of numbers, len 4
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "CropFixed", OPTIONAL, types.V10, func(a types.PDFArray) bool { return len(a) == 4 })
	if err != nil {
		return err
	}

	// Position, required, array of numbers, len 8
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Position", REQUIRED, types.V10, func(a types.PDFArray) bool { return len(a) == 8 })
	if err != nil {
		return err
	}

	return nil
}

func validateOPIDictV13Part2(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string) error {

	// Resolution, optional, array of numbers, len 2
	_, err := validateNumberArrayEntry(xRefTable, dict, dictName, "Resolution", OPTIONAL, types.V10, func(a types.PDFArray) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// ColorType, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "ColorType", OPTIONAL, types.V10, func(s string) bool { return s == "Process" || s == "Spot" || s == "Separation" })
	if err != nil {
		return err
	}

	// Color, optional, array, len 5
	_, err = validateArrayEntry(xRefTable, dict, dictName, "Color", OPTIONAL, types.V10, func(a types.PDFArray) bool { return len(a) == 5 })
	if err != nil {
		return err
	}

	// Tint, optional, number
	_, err = validateNumberEntry(xRefTable, dict, dictName, "Tint", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// Overprint, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "Overprint", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// ImageType, optional, array of integers, len 2
	_, err = validateIntegerArrayEntry(xRefTable, dict, dictName, "ImageType", OPTIONAL, types.V10, func(a types.PDFArray) bool { return len(a) == 2 })
	if err != nil {
		return err
	}

	// GrayMap, optional, array of integers
	_, err = validateIntegerArrayEntry(xRefTable, dict, dictName, "GrayMap", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// Transparency, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "Transparency", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// Tags, optional, array
	_, err = validateArrayEntry(xRefTable, dict, dictName, "Tags", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	return nil
}

func validateOPIDictV13(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// 14.11.7 Open Prepresse interface (OPI)

	logInfoValidate.Println("*** validateOPIDictV13 begin ***")

	dictName := "opiDictV13"

	err := validateOPIDictV13Part1(xRefTable, dict, dictName)
	if err != nil {
		return err
	}

	err = validateOPIDictV13Part2(xRefTable, dict, dictName)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateOPIDictV13 end ***")

	return nil
}

func validateOPIDictInks(xRefTable *types.XRefTable, obj interface{}) error {

	logInfoValidate.Printf("*** validateOPIDictInks begin ***")

	obj, err := xRefTable.Dereference(obj)
	if err != nil {
		return err
	}
	if obj == nil {
		logInfoValidate.Println("validateOPIDictInks end: is nil")
		return nil
	}

	switch obj := obj.(type) {

	case types.PDFName:
		if colorant := obj.String(); colorant != "full_color" && colorant != "registration" {
			return errors.New("validateOPIDictInks: corrupt colorant name")
		}

	case types.PDFArray:
		// no further processing

	default:
		return errors.New("validateOPIDictInks: corrupt type")

	}

	logInfoValidate.Printf("*** validateOPIDictInks end ***")

	return nil
}

func validateOPIDictV20(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// 14.11.7 Open Prepresse interface (OPI)

	logInfoValidate.Println("*** validateOPIDictV20 begin ***")

	dictName := "opiDictV20"

	_, err := validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "OPI" })
	if err != nil {
		return err
	}

	_, err = validateFloatEntry(xRefTable, dict, dictName, "Version", REQUIRED, types.V10, func(f float64) bool { return f == 2.0 })
	if err != nil {
		return err
	}

	_, err = validateFileSpecEntry(xRefTable, dict, dictName, "F", REQUIRED, types.V10)
	if err != nil {
		return err
	}

	_, err = validateStringEntry(xRefTable, dict, dictName, "MainImage", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateArrayEntry(xRefTable, dict, dictName, "Tags", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Size", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil {
		return err
	}

	_, err = validateRectangleEntry(xRefTable, dict, dictName, "CropRect", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateBooleanEntry(xRefTable, dict, dictName, "Overprint", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	if obj, found := dict.Find("Inks"); found {
		err = validateOPIDictInks(xRefTable, obj)
		if err != nil {
			return err
		}
	}

	_, err = validateIntegerArrayEntry(xRefTable, dict, dictName, "IncludedImageDimensions", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil {
		return err
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "IncludedImageQuality", OPTIONAL, types.V10, func(i int) bool { return i >= 1 && i <= 3 })
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateOPIDictV20 end ***")

	return nil
}

func validateOPIVersionDict(xRefTable *types.XRefTable, dict *types.PDFDict) error {

	// 14.11.7 Open Prepresse interface (OPI)

	logInfoValidate.Println("*** validateOPIVersionDict begin ***")

	if dict.Len() != 1 {
		return errors.New("validateOPIVersionDict: must have exactly one entry keyed 1.3 or 2.0")
	}

	for opiVersion, obj := range dict.Dict {

		if !validateOPIVersion(opiVersion) {
			return errors.New("validateOPIVersionDict: invalid OPI version")
		}

		dict, err := xRefTable.DereferenceDict(obj)
		if err != nil {
			return err
		}

		if dict == nil {
			continue
		}

		if opiVersion == "1.3" {
			err = validateOPIDictV13(xRefTable, dict)
			if err != nil {
				return err
			}
		} else {
			err = validateOPIDictV20(xRefTable, dict)
			if err != nil {
				return err
			}
		}

	}

	logInfoValidate.Println("*** validateOPIVersionDict end ***")

	return nil
}

func validateMaskStreamDict(xRefTable *types.XRefTable, streamDict *types.PDFStreamDict) error {

	logInfoValidate.Println("*** validateMaskStreamDict begin ***")

	if streamDict.Type() != nil && *streamDict.Type() != "XObject" {
		return errors.New("validateMaskStreamDict: corrupt imageStreamDict type")
	}

	if streamDict.Subtype() == nil || *streamDict.Subtype() != "Image" {
		return errors.New("validateMaskStreamDict: corrupt imageStreamDict subtype")
	}

	err := validateImageStreamDict(xRefTable, streamDict, isNoAlternateImageStreamDict)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateMaskStreamDict end: ***")

	return nil
}

func validateMaskEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, required bool, sinceVersion types.PDFVersion) error {

	// stream ("explicit masking", another Image XObject) or array of colors ("color key masking")

	logInfoValidate.Println("*** validateMaskEntry begin ***")

	entryName := "Mask"

	obj, err := dict.Entry(dictName, entryName, required)
	if err != nil || obj == nil {
		return err
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return err
	}
	if obj == nil {
		if required {
			return errors.Errorf("validateMaskEntry: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		logInfoValidate.Println("validateMaskEntry end")
		return nil
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateMaskEntry: unsupported in version %s.\n", xRefTable.VersionString())
	}

	switch obj := obj.(type) {

	case types.PDFStreamDict:
		err = validateMaskStreamDict(xRefTable, &obj)
		if err != nil {
			return err
		}

	case types.PDFArray:
		// no further processing

	default:

		return errors.Errorf("validateMaskEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	logInfoValidate.Println("*** validateMaskEntry end ***")

	return nil
}

func validateAlternateImageStreamDicts(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) error {

	logInfoValidate.Println("*** validateAlternateImageStreamDicts begin ***")

	arr, err := validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil {
		return err
	}
	if arr == nil {
		if required {
			return errors.Errorf("validateAlternateImageStreamDicts: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		logInfoValidate.Printf("validateAlternateImageStreamDicts end: dictName=%s", dictName)
		return nil
	}

	for _, obj := range *arr {

		streamDict, err := validateStreamDict(xRefTable, obj)
		if err != nil {
			return err
		}

		if streamDict == nil {
			continue
		}

		err = validateImageStreamDict(xRefTable, streamDict, isAlternateImageStreamDict)
		if err != nil {
			return err
		}
	}

	logInfoValidate.Println("*** validateAlternateImageStreamDicts begin ***")

	return nil
}

func validateImageStreamDictPart1(xRefTable *types.XRefTable, streamDict *types.PDFStreamDict, dictName string) (isImageMask bool, err error) {

	dict := streamDict.PDFDict

	// Width, integer, required
	_, err = validateIntegerEntry(xRefTable, &dict, dictName, "Width", REQUIRED, types.V10, nil)
	if err != nil {
		return false, err
	}

	// Height, integer, required
	_, err = validateIntegerEntry(xRefTable, &dict, dictName, "Height", REQUIRED, types.V10, nil)
	if err != nil {
		return false, err
	}

	// ImageMask, boolean, optional
	imageMask, err := validateBooleanEntry(xRefTable, &dict, "imageStreamDict", "ImageMask", OPTIONAL, types.V10, nil)
	if err != nil {
		return false, err
	}

	isImageMask = imageMask != nil && *imageMask == true

	// ColorSpace, name or array, required unless used filter is JPXDecode; not allowed for imagemasks.
	if !isImageMask {

		required := REQUIRED

		if streamDict.HasSoleFilterNamed("JPXDecode") {
			required = OPTIONAL
		}

		if streamDict.HasSoleFilterNamed("CCITTFaxDecode") && xRefTable.ValidationMode == types.ValidationRelaxed {
			required = OPTIONAL
		}

		err = validateColorSpaceEntry(xRefTable, &dict, "imageStreamDict", "ColorSpace", required, ExcludePatternCS)
		if err != nil {
			return false, err
		}

	}

	return isImageMask, nil
}

func validateImageStreamDictPart2(xRefTable *types.XRefTable, streamDict *types.PDFStreamDict, dictName string, isImageMask, isAlternate bool) error {

	dict := streamDict.PDFDict

	// BitsPerComponent, integer
	required := REQUIRED
	if streamDict.HasSoleFilterNamed("JPXDecode") {
		required = OPTIONAL
	}
	_, err := validateIntegerEntry(xRefTable, &dict, dictName, "BitsPerComponent", required, types.V10, nil)
	if err != nil {
		return err
	}

	// Intent, name, optional, since V1.0
	_, err = validateNameEntry(xRefTable, &dict, dictName, "Intent", OPTIONAL, types.V11, validateRenderingIntent)
	if err != nil {
		return err
	}

	// Mask, stream or array, optional since V1.3; not allowed for image masks.
	if !isImageMask {
		err = validateMaskEntry(xRefTable, &dict, dictName, OPTIONAL, types.V13)
		if err != nil {
			return err
		}
	}

	// Decode, array, optional
	_, err = validateNumberArrayEntry(xRefTable, &dict, dictName, "Decode", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// Interpolate, boolean, optional
	_, err = validateBooleanEntry(xRefTable, &dict, dictName, "Interpolate", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	// Alternates, array, optional, since V1.3
	if !isAlternate {
		err = validateAlternateImageStreamDicts(xRefTable, &dict, dictName, "Alternates", OPTIONAL, types.V13)
		if err != nil {
			return err
		}
	}

	// SMask, stream, optional, since V1.4
	sinceVersion := types.V14
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}
	sd, err := validateStreamDictEntry(xRefTable, &dict, dictName, "SMask", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return err
	}

	if sd != nil {
		err = validateImageStreamDict(xRefTable, sd, isNoAlternateImageStreamDict)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateImageStreamDict(xRefTable *types.XRefTable, streamDict *types.PDFStreamDict, isAlternate bool) error {

	logInfoValidate.Println("*** validateImageStreamDict begin ***")

	dict := streamDict.PDFDict

	dictName := "imageStreamDict"

	var isImageMask bool

	isImageMask, err := validateImageStreamDictPart1(xRefTable, streamDict, dictName)
	if err != nil {
		return err
	}

	err = validateImageStreamDictPart2(xRefTable, streamDict, dictName, isImageMask, isAlternate)
	if err != nil {
		return err
	}

	// SMaskInData, integer, optional
	_, err = validateIntegerEntry(xRefTable, &dict, dictName, "SMaskInData", OPTIONAL, types.V10, func(i int) bool { return i >= 0 && i <= 2 })
	if err != nil {
		return err
	}

	// Name, name, required for V10
	// Shall no longer be used.
	_, err = validateNameEntry(xRefTable, &dict, dictName, "Name", xRefTable.Version() == types.V10, types.V10, nil)
	if err != nil {
		return err
	}

	// StructParent, integer, optional
	_, err = validateIntegerEntry(xRefTable, &dict, dictName, "StructParent", OPTIONAL, types.V13, nil)
	if err != nil {
		return err
	}

	// ID, byte string, optional, since V1.3
	_, err = validateStringEntry(xRefTable, &dict, dictName, "ID", OPTIONAL, types.V13, nil)
	if err != nil {
		return err
	}

	// OPI, dict, optional since V1.2
	err = validateEntryOPI(xRefTable, &dict, dictName, OPTIONAL, types.V12)
	if err != nil {
		return err
	}

	// Metadata, stream, optional since V1.4
	err = validateMetadata(xRefTable, &streamDict.PDFDict, OPTIONAL, types.V14)
	if err != nil {
		return err
	}

	// OC, dict, optional since V1.5
	err = validateEntryOC(xRefTable, &dict, dictName, OPTIONAL, types.V15)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateImageStreamDict end ***")

	return nil
}

func validateFormStreamDictPart1(xRefTable *types.XRefTable, streamDict *types.PDFStreamDict, dictName string) error {

	dict := streamDict.PDFDict

	_, err := validateIntegerEntry(xRefTable, &dict, dictName, "FormType", OPTIONAL, types.V10, func(i int) bool { return i == 1 })
	if err != nil {
		return err
	}

	_, err = validateRectangleEntry(xRefTable, &dict, dictName, "BBox", REQUIRED, types.V10, nil)
	if err != nil {
		return err
	}

	_, err = validateNumberArrayEntry(xRefTable, &dict, dictName, "Matrix", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 6 })
	if err != nil {
		return err
	}

	// Resources, dict, optional, since V1.2
	if obj, ok := streamDict.Find("Resources"); ok {
		_, err = validateResourceDict(xRefTable, obj)
	}

	// Group, dict, optional, since V1.4
	err = validatePageEntryGroup(xRefTable, &dict, OPTIONAL, types.V14)
	if err != nil {
		return err
	}

	// Ref, dict, optional, since V1.4
	d, err := validateDictEntry(xRefTable, &dict, dictName, "Ref", OPTIONAL, types.V14, nil)
	if err != nil {
		return err
	}
	if d != nil {
		err = validateReferenceDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	// Metadata, stream, optional, since V1.4
	return validateMetadata(xRefTable, &dict, OPTIONAL, types.V14)
}

func validateEntryOC(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, required bool, sinceVersion types.PDFVersion) error {

	var d *types.PDFDict

	d, err := validateDictEntry(xRefTable, dict, dictName, "OC", required, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d != nil {
		err = validateOptionalContentGroupDict(xRefTable, d, sinceVersion)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateEntryOPI(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, required bool, sinceVersion types.PDFVersion) error {

	d, err := validateDictEntry(xRefTable, dict, dictName, "OPI", required, sinceVersion, nil)
	if err != nil {
		return err
	}

	if d != nil {
		err = validateOPIVersionDict(xRefTable, d)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateFormStreamDictPart2(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string) error {

	// PieceInfo, dict, optional, since V1.3
	hasPieceInfo, err := validatePieceInfo(xRefTable, dict, OPTIONAL, types.V13)
	if err != nil {
		return err
	}

	// LastModified, date, required if PieceInfo present, since V1.3
	lm, err := validateDateEntry(xRefTable, dict, dictName, "LastModified", OPTIONAL, types.V13)
	if err != nil {
		return err
	}

	if hasPieceInfo && lm == nil {
		err = errors.New("validateFormStreamDictPart2: missing \"LastModified\" (required by \"PieceInfo\")")
		return err
	}

	// StructParent, integer
	sp, err := validateIntegerEntry(xRefTable, dict, dictName, "StructParent", OPTIONAL, types.V13, nil)
	if err != nil {
		return err
	}

	// StructParents, integer
	sps, err := validateIntegerEntry(xRefTable, dict, dictName, "StructParents", OPTIONAL, types.V13, nil)
	if err != nil {
		return err
	}
	if sp != nil && sps != nil {
		return errors.New("validateFormStreamDictPart2: only \"StructParent\" or \"StructParents\" allowed")
	}

	// OPI, dict, optional, since V1.2
	err = validateEntryOPI(xRefTable, dict, dictName, OPTIONAL, types.V12)
	if err != nil {
		return err
	}

	// OC, dict, optional, since V1.5
	err = validateEntryOC(xRefTable, dict, dictName, OPTIONAL, types.V15)
	if err != nil {
		return err
	}

	// Name, name, optional (required in 1.0)
	required := xRefTable.Version() == types.V10
	_, err = validateNameEntry(xRefTable, dict, dictName, "Name", required, types.V10, nil)
	if err != nil {
		return err
	}

	return nil
}

func validateFormStreamDict(xRefTable *types.XRefTable, streamDict *types.PDFStreamDict) error {

	// 8.10 Form XObjects

	logInfoValidate.Println("*** validateFormStreamDict begin ***")

	dictName := "formStreamDict"

	err := validateFormStreamDictPart1(xRefTable, streamDict, dictName)
	if err != nil {
		return err
	}

	err = validateFormStreamDictPart2(xRefTable, &streamDict.PDFDict, dictName)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateFormStreamDict end ***")

	return nil
}

func validateXObjectStreamDict(xRefTable *types.XRefTable, streamDict *types.PDFStreamDict) error {

	// see 8.8 External Objects

	logInfoValidate.Println("*** validateXObjectStreamDict begin ***")

	_, err := validateNameEntry(xRefTable, &streamDict.PDFDict, "xObjectStreamDict", "Type", OPTIONAL, types.V10, func(s string) bool { return s == "XObject" })
	if err != nil {
		return err
	}

	required := REQUIRED
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		required = OPTIONAL
	}
	subtype, err := validateNameEntry(xRefTable, &streamDict.PDFDict, "xObjectStreamDict", "Subtype", required, types.V10, nil)
	if err != nil {
		return err
	}

	if subtype == nil {

		// relaxed
		_, found := streamDict.Find("BBox")
		if found {

			err = validateFormStreamDict(xRefTable, streamDict)
			if err != nil {
				return err
			}

			logInfoValidate.Println("validateXObjectStreamDict end")
			return nil
		}

		// Relaxed for page Thumb
		err = validateImageStreamDict(xRefTable, streamDict, isNoAlternateImageStreamDict)
		if err != nil {
			return err
		}

		logInfoValidate.Println("validateXObjectStreamDict end")
		return nil
	}

	switch *subtype {

	case "Form":
		err = validateFormStreamDict(xRefTable, streamDict)

	case "Image":
		err = validateImageStreamDict(xRefTable, streamDict, isNoAlternateImageStreamDict)

	case "PS":
		err = errors.Errorf("validateXObjectStreamDict: PostScript XObjects should not be used")

	default:
		return errors.Errorf("validateXObjectStreamDict: unknown Subtype: %s\n", *subtype)

	}

	logInfoValidate.Println("*** validateXObjectStreamDict end  ***")

	return err
}

func validateGroupAttributesDict(xRefTable *types.XRefTable, obj interface{}) error {

	// see 11.6.6 Transparency Group XObjects

	logInfoValidate.Println("*** validateGroupAttributesDict begin ***")

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil || dict == nil {
		return err
	}

	dictName := "groupAttributesDict"

	// Type, name, optional
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "Group" })
	if err != nil {
		return err
	}

	// S, name, required
	_, err = validateNameEntry(xRefTable, dict, dictName, "S", REQUIRED, types.V10, func(s string) bool { return s == "Transparency" })
	if err != nil {
		return err
	}

	// CS, colorSpace, optional
	err = validateColorSpaceEntry(xRefTable, dict, dictName, "CS", OPTIONAL, ExcludePatternCS)
	if err != nil {
		return err
	}

	// I, boolean, optional
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "I", OPTIONAL, types.V10, nil)
	if err != nil {
		return err
	}

	logInfoValidate.Println("*** validateGroupAttributesDict end ***")

	return nil
}

func validateXObjectResourceDict(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) error {

	logInfoValidate.Println("*** validateXObjectResourceDict begin ***")

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return err
	}
	if dict == nil {
		logInfoValidate.Println("validateXObjectResourceDict end: object is nil.")
		return nil
	}

	// Iterate over xObject resource dictionary
	for _, obj := range dict.Dict {

		sd, err := xRefTable.DereferenceStreamDict(obj)
		if err != nil {
			return err
		}

		if sd == nil {
			logInfoValidate.Printf("validateXObjectResourceDict end: font resource object is nil.")
			continue
		}

		// Process XObject dict
		err = validateXObjectStreamDict(xRefTable, sd)
		if err != nil {
			return err
		}
	}

	logInfoValidate.Println("*** validateXObjectResourceDict end ***")

	return nil
}
