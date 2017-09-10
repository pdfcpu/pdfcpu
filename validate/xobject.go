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

func validateReferenceDictPageEntry(xRefTable *types.XRefTable, obj interface{}) (err error) {

	logInfoValidate.Printf("*** validateReferenceDictPageEntry begin ***")

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		logInfoValidate.Println("validateReferenceDictPageEntry end: is nil.")
		return
	}

	switch obj.(type) {

	case types.PDFInteger:
		// no further processing

	case types.PDFStringLiteral:
		// no further processing

	case types.PDFHexLiteral:
		// no further processing

	default:
		err = errors.New("validateReferenceDictPageEntry: corrupt type")

	}

	logInfoValidate.Println("*** validateReferenceDictPageEntry end ***")

	return
}

func validateReferenceDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// see 8.10.4 Reference XObjects

	logInfoValidate.Println("*** validateReferenceDict begin ***")

	// F, file spec, required
	_, err = validateFileSpecEntry(xRefTable, dict, "refDict", "F", REQUIRED, types.V10)
	if err != nil {
		return
	}

	// Page, integer or text string, required
	obj, ok := dict.Find("Page")
	if !ok {
		return errors.New("validateReferenceDict: missing required entry \"Page\"")
	}

	err = validateReferenceDictPageEntry(xRefTable, obj)
	if err != nil {
		return
	}

	// ID, string array, optional
	_, err = validateStringArrayEntry(xRefTable, dict, "refDict", "ID", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateReferenceDict end ***")

	return
}

func validateOPIDictV13(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// 14.11.7 Open Prepresse interface (OPI)

	logInfoValidate.Println("*** validateOPIDictV13 begin ***")

	dictName := "opiDictV13"

	// Type, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "OPI" })
	if err != nil {
		return
	}

	// Version, required, number
	_, err = validateNumberEntry(xRefTable, dict, dictName, "Version", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// F, required, file specification
	_, err = validateFileSpecEntry(xRefTable, dict, dictName, "F", REQUIRED, types.V10)
	if err != nil {
		return
	}

	// ID, optional, byte string
	_, err = validateStringEntry(xRefTable, dict, dictName, "ID", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Comments, optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "Comments", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Size, required, array of integers, len 2
	_, err = validateIntegerArrayEntry(xRefTable, dict, dictName, "Size", REQUIRED, types.V10, func(a types.PDFArray) bool { return len(a) == 2 })
	if err != nil {
		return
	}

	// CropRect, required, array of integers, len 4
	_, err = validateIntegerArrayEntry(xRefTable, dict, dictName, "CropRect", REQUIRED, types.V10, func(a types.PDFArray) bool { return len(a) == 4 })
	if err != nil {
		return
	}

	// CropFixed, optional, array of numbers, len 4
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "CropFixed", OPTIONAL, types.V10, func(a types.PDFArray) bool { return len(a) == 4 })
	if err != nil {
		return
	}

	// Position, required, array of numbers, len 8
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Position", REQUIRED, types.V10, func(a types.PDFArray) bool { return len(a) == 8 })
	if err != nil {
		return
	}

	// Resolution, optional, array of numbers, len 2
	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Resolution", OPTIONAL, types.V10, func(a types.PDFArray) bool { return len(a) == 2 })
	if err != nil {
		return
	}

	// ColorType, optional, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "ColorType", OPTIONAL, types.V10, func(s string) bool { return s == "Process" || s == "Spot" || s == "Separation" })
	if err != nil {
		return
	}

	// Color, optional, array, len 5
	_, err = validateArrayEntry(xRefTable, dict, dictName, "Color", OPTIONAL, types.V10, func(a types.PDFArray) bool { return len(a) == 5 })
	if err != nil {
		return
	}

	// Tint, optional, number
	_, err = validateNumberEntry(xRefTable, dict, dictName, "Tint", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Overprint, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "Overprint", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// ImageType, optional, array of integers, len 2
	_, err = validateIntegerArrayEntry(xRefTable, dict, dictName, "ImageType", OPTIONAL, types.V10, func(a types.PDFArray) bool { return len(a) == 2 })
	if err != nil {
		return
	}

	// GrayMap, optional, array of integers
	_, err = validateIntegerArrayEntry(xRefTable, dict, dictName, "GrayMap", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Transparency, optional, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "Transparency", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Tags, optional, array
	_, err = validateArrayEntry(xRefTable, dict, dictName, "Tags", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateOPIDictV13 end ***")

	return
}

func validateOPIDictInks(xRefTable *types.XRefTable, obj interface{}) (err error) {

	logInfoValidate.Printf("*** validateOPIDictInks begin ***")

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		logInfoValidate.Println("validateOPIDictInks end: is nil")
		return
	}

	switch obj := obj.(type) {

	case types.PDFName:
		if colorant := obj.String(); colorant != "full_color" && colorant != "registration" {
			err = errors.New("validateOPIDictInks: corrupt colorant name")
		}

	case types.PDFArray:
		// no further processing

	default:
		err = errors.New("validateOPIDictInks: corrupt type")

	}

	logInfoValidate.Printf("*** validateOPIDictInks end ***")

	return
}

func validateOPIDictV20(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// 14.11.7 Open Prepresse interface (OPI)

	logInfoValidate.Println("*** validateOPIDictV20 begin ***")

	dictName := "opiDictV20"

	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "OPI" })
	if err != nil {
		return
	}

	_, err = validateFloatEntry(xRefTable, dict, dictName, "Version", REQUIRED, types.V10, func(f float64) bool { return f == 2.0 })
	if err != nil {
		return
	}

	_, err = validateFileSpecEntry(xRefTable, dict, dictName, "F", REQUIRED, types.V10)
	if err != nil {
		return
	}

	_, err = validateStringEntry(xRefTable, dict, dictName, "MainImage", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	_, err = validateArrayEntry(xRefTable, dict, dictName, "Tags", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	_, err = validateNumberArrayEntry(xRefTable, dict, dictName, "Size", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil {
		return
	}

	_, err = validateRectangleEntry(xRefTable, dict, dictName, "CropRect", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	_, err = validateBooleanEntry(xRefTable, dict, dictName, "Overprint", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	if obj, found := dict.Find("Inks"); found {
		err = validateOPIDictInks(xRefTable, obj)
		if err != nil {
			return
		}
	}

	_, err = validateIntegerArrayEntry(xRefTable, dict, dictName, "IncludedImageDimensions", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil {
		return
	}

	_, err = validateIntegerEntry(xRefTable, dict, dictName, "IncludedImageQuality", OPTIONAL, types.V10, func(i int) bool { return i >= 1 && i <= 3 })
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateOPIDictV20 end ***")

	return
}

func validateOPIVersionDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	// 14.11.7 Open Prepresse interface (OPI)

	logInfoValidate.Println("*** validateOPIVersionDict begin ***")

	if dict.Len() != 1 {
		return errors.New("validateOPIVersionDict: must have exactly one entry keyed 1.3 or 2.0")
	}

	for opiVersion, obj := range dict.Dict {

		if !validateOPIVersion(opiVersion) {
			return errors.New("validateOPIVersionDict: invalid OPI version")
		}

		dict, err := validateDict(xRefTable, obj)
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

	return
}

func validateMaskStreamDict(xRefTable *types.XRefTable, streamDict *types.PDFStreamDict) (err error) {

	logInfoValidate.Println("*** validateMaskStreamDict begin ***")

	if streamDict.Type() != nil && *streamDict.Type() != "XObject" {
		return errors.New("validateMaskStreamDict: corrupt imageStreamDict type")
	}

	if streamDict.Subtype() == nil || *streamDict.Subtype() != "Image" {
		return errors.New("validateMaskStreamDict: corrupt imageStreamDict subtype")
	}

	err = validateImageStreamDict(xRefTable, streamDict, isNoAlternateImageStreamDict)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateMaskStreamDict end: ***")

	return
}

func validateMaskEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, required bool, sinceVersion types.PDFVersion) (err error) {

	// stream ("explicit masking", another Image XObject) or array of colors ("color key masking")

	logInfoValidate.Println("*** validateMaskEntry begin ***")

	entryName := "Mask"

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("validateMaskEntry: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		logInfoValidate.Printf("validateMaskEntry end: \"%s\" is nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			return errors.Errorf("validateMaskEntry: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		logInfoValidate.Println("validateMaskEntry end")
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateMaskEntry: unsupported in version %s.\n", xRefTable.VersionString())
	}

	switch obj := obj.(type) {

	case types.PDFStreamDict:
		err = validateMaskStreamDict(xRefTable, &obj)

	case types.PDFArray:
		// no further processing

	default:

		err = errors.Errorf("validateMaskEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	logInfoValidate.Println("*** validateMaskEntry end ***")

	return
}

func validateAlternateImageStreamDicts(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateAlternateImageStreamDicts begin ***")

	arr, err := validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil {
		return
	}

	if arr == nil {
		if required {
			return errors.Errorf("validateAlternateImageStreamDicts: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		logInfoValidate.Printf("validateAlternateImageStreamDicts end: dictName=%s", dictName)
		return
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

	return
}

func validateImageStreamDict(xRefTable *types.XRefTable, streamDict *types.PDFStreamDict, isAlternate bool) (err error) {

	logInfoValidate.Println("*** validateImageStreamDict begin ***")

	dict := streamDict.PDFDict

	dictName := "imageStreamDict"

	// Width, integer, required
	_, err = validateIntegerEntry(xRefTable, &dict, dictName, "Width", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// Height, integer, required
	_, err = validateIntegerEntry(xRefTable, &dict, dictName, "Height", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// ImageMask, boolean, optional
	imageMask, err := validateBooleanEntry(xRefTable, &streamDict.PDFDict, "imageStreamDict", "ImageMask", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	isImageMask := imageMask != nil && *imageMask == true

	// ColorSpace, name or array, required unless used filter is JPXDecode; not allowed for imagemasks.
	if !isImageMask {

		required := REQUIRED

		if streamDict.HasSoleFilterNamed("JPXDecode") {
			required = OPTIONAL
		}

		if streamDict.HasSoleFilterNamed("CCITTFaxDecode") && xRefTable.ValidationMode == types.ValidationRelaxed {
			required = OPTIONAL
		}

		err = validateColorSpaceEntry(xRefTable, &streamDict.PDFDict, "imageStreamDict", "ColorSpace", required, ExcludePatternCS)
		if err != nil {
			return
		}

	}

	// BitsPerComponent, integer
	required := REQUIRED
	if streamDict.HasSoleFilterNamed("JPXDecode") {
		required = OPTIONAL
	}
	_, err = validateIntegerEntry(xRefTable, &dict, dictName, "BitsPerComponent", required, types.V10, nil)
	if err != nil {
		return
	}

	// Intent, name, optional, since V1.0
	_, err = validateNameEntry(xRefTable, &dict, dictName, "Intent", OPTIONAL, types.V11, validateRenderingIntent)
	if err != nil {
		return
	}

	// Mask, stream or array, optional since V1.3; not allowed for image masks.
	if !isImageMask {
		err = validateMaskEntry(xRefTable, &dict, dictName, OPTIONAL, types.V13)
		if err != nil {
			return
		}
	}

	// Decode, array, optional
	_, err = validateNumberArrayEntry(xRefTable, &dict, dictName, "Decode", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Interpolate, boolean, optional
	_, err = validateBooleanEntry(xRefTable, &dict, dictName, "Interpolate", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Alternates, array, optional, since V1.3
	if !isAlternate {
		err = validateAlternateImageStreamDicts(xRefTable, &dict, dictName, "Alternates", OPTIONAL, types.V13)
		if err != nil {
			return
		}
	}

	// SMask, stream, optional, since V1.4
	sd, err := validateStreamDictEntry(xRefTable, &dict, dictName, "SMask", OPTIONAL, types.V14, nil)
	if err != nil {
		return
	}

	if sd != nil {
		err = validateImageStreamDict(xRefTable, sd, isNoAlternateImageStreamDict)
		if err != nil {
			return
		}
	}

	// SMaskInData, integer, optional
	_, err = validateIntegerEntry(xRefTable, &dict, dictName, "SMaskInData", OPTIONAL, types.V10, func(i int) bool { return i >= 0 && i <= 2 })
	if err != nil {
		return
	}

	// Name, name, required for V10
	// Shall no longer be used.
	_, err = validateNameEntry(xRefTable, &dict, dictName, "Name", xRefTable.Version() == types.V10, types.V10, nil)
	if err != nil {
		return
	}

	// StructParent, integer, optional
	_, err = validateIntegerEntry(xRefTable, &dict, dictName, "StructParent", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// ID, byte string, optional, since V1.3
	_, err = validateStringEntry(xRefTable, &dict, dictName, "ID", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// OPI, dict, optional since V1.2
	d, err := validateDictEntry(xRefTable, &dict, dictName, "OPI", OPTIONAL, types.V12, nil)
	if err != nil {
		return
	}

	if d != nil {
		err = validateOPIVersionDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	// Metadata, stream, optional since V1.4
	sinceVersion := types.V14
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}
	err = validateMetadata(xRefTable, &streamDict.PDFDict, OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	// OC, dict, optional since V1.5
	d, err = validateDictEntry(xRefTable, &dict, dictName, "OC", OPTIONAL, types.V15, nil)
	if err != nil {
		return
	}

	if d != nil {
		err = validateOptionalContentGroupDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	logInfoValidate.Println("*** validateImageStreamDict end ***")

	return
}

func validateFormStreamDict(xRefTable *types.XRefTable, streamDict *types.PDFStreamDict) (err error) {

	// 8.10 Form XObjects

	logInfoValidate.Println("*** validateFormStreamDict begin ***")

	dict := streamDict.PDFDict

	dictName := "formStreamDict"

	_, err = validateIntegerEntry(xRefTable, &dict, dictName, "FormType", OPTIONAL, types.V10, func(i int) bool { return i == 1 })
	if err != nil {
		return
	}

	_, err = validateRectangleEntry(xRefTable, &dict, dictName, "BBox", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	_, err = validateNumberArrayEntry(xRefTable, &dict, dictName, "Matrix", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 6 })
	if err != nil {
		return
	}

	// Resources, dict, optional, since V1.2
	if obj, ok := streamDict.Find("Resources"); ok {
		_, err = validateResourceDict(xRefTable, obj)
	}

	// Group, dict, optional, since V1.4
	err = validatePageEntryGroup(xRefTable, &dict, OPTIONAL, types.V14)
	if err != nil {
		return
	}

	// Ref, dict, optional, since V1.4
	d, err := validateDictEntry(xRefTable, &dict, dictName, "Ref", OPTIONAL, types.V14, nil)
	if err != nil {
		return
	}

	if d != nil {
		err = validateReferenceDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	// Metadata, stream, optional, since V1.4
	err = validateMetadata(xRefTable, &streamDict.PDFDict, OPTIONAL, types.V14)
	if err != nil {
		return
	}

	// PieceInfo, dict, optional, since V1.3
	hasPieceInfo, err := validatePieceInfo(xRefTable, &streamDict.PDFDict, OPTIONAL, types.V13)
	if err != nil {
		return err
	}

	// LastModified, date, required if PieceInfo present, since V1.3
	lm, err := validateDateEntry(xRefTable, &dict, dictName, "LastModified", OPTIONAL, types.V13)
	if err != nil {
		return
	}

	if hasPieceInfo && lm == nil {
		err = errors.New("validateFormStreamDict: missing \"LastModified\" (required by \"PieceInfo\")")
		return
	}

	// StructParent, integer
	sp, err := validateIntegerEntry(xRefTable, &dict, dictName, "StructParent", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// StructParents, integer
	sps, err := validateIntegerEntry(xRefTable, &dict, dictName, "StructParents", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}
	if sp != nil && sps != nil {
		return errors.New("validateFormStreamDict: only \"StructParent\" or \"StructParents\" allowed")
	}

	// OPI, dict, optional, since V1.2
	d, err = validateDictEntry(xRefTable, &dict, dictName, "OPI", OPTIONAL, types.V12, nil)
	if err != nil {
		return
	}

	if d != nil {
		err = validateOPIVersionDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	// OC, dict, optional, since V1.5
	d, err = validateDictEntry(xRefTable, &dict, dictName, "OC", OPTIONAL, types.V15, nil)
	if err != nil {
		return
	}

	if d != nil {
		err = validateOptionalContentGroupDict(xRefTable, d)
		if err != nil {
			return
		}
	}

	// Name, name, optional (required in 1.0)
	required := xRefTable.Version() == types.V10
	_, err = validateNameEntry(xRefTable, &dict, dictName, "Name", required, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateFormStreamDict end ***")

	return
}

func validateXObjectStreamDict(xRefTable *types.XRefTable, streamDict *types.PDFStreamDict) (err error) {

	// see 8.8 External Objects

	logInfoValidate.Println("*** validateXObjectStreamDict begin ***")

	_, err = validateNameEntry(xRefTable, &streamDict.PDFDict, "xObjectStreamDict", "Type", OPTIONAL, types.V10, func(s string) bool { return s == "XObject" })
	if err != nil {
		return
	}

	required := REQUIRED
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		required = OPTIONAL
	}
	subtype, err := validateNameEntry(xRefTable, &streamDict.PDFDict, "xObjectStreamDict", "Subtype", required, types.V10, nil)
	if err != nil {
		return
	}

	if subtype == nil {

		// relaxed
		_, found := streamDict.Find("BBox")
		if found {

			err = validateFormStreamDict(xRefTable, streamDict)
			if err != nil {
				return
			}

			logInfoValidate.Println("validateXObjectStreamDict end")
			return
		}

		// Relaxed for page Thumb
		err = validateImageStreamDict(xRefTable, streamDict, isNoAlternateImageStreamDict)
		if err != nil {
			return
		}

		logInfoValidate.Println("validateXObjectStreamDict end")
		return
	}

	switch *subtype {

	case "Form":
		err = validateFormStreamDict(xRefTable, streamDict)

	case "Image":
		err = validateImageStreamDict(xRefTable, streamDict, isNoAlternateImageStreamDict)

	case "PS":
		err = errors.Errorf("validateXObjectStreamDict: PostScript XObjects should not be used")

	default:
		err = errors.Errorf("validateXObjectStreamDict: unknown Subtype: %s\n", *subtype)

	}

	logInfoValidate.Println("*** validateXObjectStreamDict end  ***")

	return
}

func validateGroupAttributesDict(xRefTable *types.XRefTable, obj interface{}) (err error) {

	// see 11.6.6 Transparency Group XObjects

	logInfoValidate.Println("*** validateGroupAttributesDict begin ***")

	dict, err := validateDict(xRefTable, obj)
	if err != nil {
		return
	}

	dictName := "groupAttributesDict"

	// Type, name, optional
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "Group" })
	if err != nil {
		return
	}

	// S, name, required
	_, err = validateNameEntry(xRefTable, dict, dictName, "S", REQUIRED, types.V10, func(s string) bool { return s == "Transparency" })
	if err != nil {
		return
	}

	// CS, colorSpace, optional
	err = validateColorSpaceEntry(xRefTable, dict, dictName, "CS", OPTIONAL, ExcludePatternCS)
	if err != nil {
		return
	}

	// I, boolean, optional
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "I", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateGroupAttributesDict end ***")

	return
}

func validateXObjectResourceDict(xRefTable *types.XRefTable, obj interface{}) (err error) {

	logInfoValidate.Println("*** validateXObjectResourceDict begin ***")

	dict, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Println("validateXObjectResourceDict end: object is nil.")
		return
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

	return
}
