package write

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/hhrutter/pdflib/validate"
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

func writeReferenceDictPageEntry(ctx *types.PDFContext, obj interface{}) (err error) {

	logInfoWriter.Printf("*** writeReferenceDictPageEntry begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeReferenceDictPageEntry end: already written, offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeReferenceDictPageEntry end: is nil, offset=%d\n", ctx.Write.Offset)
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
		err = errors.New("writeReferenceDictPageEntry: corrupt type")

	}

	logInfoWriter.Printf("*** writeReferenceDictPageEntry end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeReferenceDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// see 8.10.4 Reference XObjects

	logInfoWriter.Printf("*** writeReferenceDict begin: offset=%d ***\n", ctx.Write.Offset)

	// F, file spec, required
	_, err = writeFileSpecEntry(ctx, dict, "refDict", "F", REQUIRED, types.V10)
	if err != nil {
		return
	}

	// Page, integer or text string, required
	obj, ok := dict.Find("Page")
	if !ok {
		return errors.New("writeReferenceDict: missing required entry \"Page\"")
	}

	err = writeReferenceDictPageEntry(ctx, obj)
	if err != nil {
		return
	}

	// ID, string array, optional
	_, _, err = writeStringArrayEntry(ctx, dict, "refDict", "ID", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeReferenceDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeUsageDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// see 8.11.4.4

	logInfoWriter.Printf("*** writeUsage begin: offset=%d ***\n", ctx.Write.Offset)

	err = errors.New("*** writeUsage: unsupported ***")

	logInfoWriter.Printf("*** writeUsage end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeOptionalContentGroupIntent(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeOptionalContentGroupIntent begin: offset=%d ***\n", ctx.Write.Offset)

	err = errors.New("*** writeOptionalContentGroupintent: unsupported ***")

	logInfoWriter.Printf("*** writeOptionalContentGroupIntent end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeOptionalContentGroupDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// see 8.11 Optional Content

	logInfoWriter.Printf("*** writeOptionalContentGroupDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "optionalContentGroupDict"

	_, _, err = writeNameEntry(ctx, dict, dictName, "Type", REQUIRED, types.V10, func(s string) bool { return s == "OCG" })
	if err != nil {
		return
	}

	_, _, err = writeStringEntry(ctx, dict, dictName, "Name", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// TODO Ausbauen!
	err = writeOptionalContentGroupIntent(ctx, dict, dictName, "Intent", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// TODO Ausbauen!
	d, written, err := writeDictEntry(ctx, dict, dictName, "Usage", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	if !written && d != nil {
		err = writeUsageDict(ctx, *d)
		if err != nil {
			return
		}
	}

	logInfoWriter.Printf("*** writeOptionalContentGroupDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeOPIDictV13(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// 14.11.7 Open Prepresse interface (OPI)

	logInfoWriter.Printf("*** writeOPIDictV13 begin: offset=%d ***\n", ctx.Write.Offset)

	err = errors.New("*** writeOPIDictV13: unsupported ***")

	logInfoWriter.Printf("*** writeOPIDictV13 end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeOPIDictInkArray(ctx *types.PDFContext, arr types.PDFArray) (err error) {

	logInfoWriter.Printf("*** writeOPIDictInkArray begin: offset=%d ***\n", ctx.Write.Offset)

	// name, name1, real1, name2, real2 ...
	err = errors.New("*** writeOPIDictInkArray: unsupported ***")

	logInfoWriter.Printf("*** writeOPIDictInkArray end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeOPIDictInks(ctx *types.PDFContext, obj interface{}) (err error) {

	logInfoWriter.Printf("*** writeOPIDictInks begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeOPIDictInks end: already written, offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeOPIDictInks end: is nil, offset=%d\n", ctx.Write.Offset)
		return
	}

	switch obj := obj.(type) {

	case types.PDFName:
		if colorant := obj.String(); colorant != "full_color" && colorant != "registration" {
			err = errors.New("writeOPIDictInks: corrupt colorant name")
		}

	case types.PDFArray:
		err = writeOPIDictInkArray(ctx, obj)

	default:
		err = errors.New("writeOPIDictInks: corrupt type")

	}

	logInfoWriter.Printf("*** writeOPIDictInks end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeOPIDictV20(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// 14.11.7 Open Prepresse interface (OPI)

	logInfoWriter.Printf("*** writeOPIDictV20 begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "opiDictV20"

	_, _, err = writeNameEntry(ctx, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "OPI" })
	if err != nil {
		return
	}

	_, _, err = writeFloatEntry(ctx, dict, dictName, "Version", REQUIRED, types.V10, func(f float64) bool { return f == 2.0 })
	if err != nil {
		return
	}

	_, err = writeFileSpecEntry(ctx, dict, dictName, "F", REQUIRED, types.V10)
	if err != nil {
		return
	}

	_, _, err = writeStringEntry(ctx, dict, dictName, "MainImage", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	_, _, err = writeArrayEntry(ctx, dict, dictName, "Tags", OPTIONAL, types.V10, nil) // TODO validate
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, dict, dictName, "Size", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil {
		return
	}

	_, _, err = writeRectangleEntry(ctx, dict, dictName, "CropRect", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	_, _, err = writeBooleanEntry(ctx, dict, dictName, "Overprint", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	if obj, found := dict.Find("Inks"); found {
		err = writeOPIDictInks(ctx, obj)
		if err != nil {
			return
		}
	}

	_, _, err = writeIntegerArrayEntry(ctx, dict, dictName, "IncludedImageDimensions", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, dict, dictName, "IncludedImageQuality", OPTIONAL, types.V10, func(i int) bool { return i >= 1 && i <= 3 })
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeOPIDictV20 end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeOPIVersionDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// 14.11.7 Open Prepresse interface (OPI)

	logInfoWriter.Printf("*** writeOPIVersionDict begin: offset=%d ***\n", ctx.Write.Offset)

	if dict.Len() != 1 {
		return errors.New("writeOPIVersionDict: must have exactly one entry keyed 1.3 or 2.0")
	}

	for opiVersion, obj := range dict.Dict {

		if !validate.OPIVersion(opiVersion) {
			return errors.New("writeOPIVersionDict: invalid OPI version")
		}

		dict, written, err := writeDict(ctx, obj)
		if err != nil {
			return err
		}

		if written || dict == nil {
			continue
		}

		if opiVersion == "1.3" {
			err = writeOPIDictV13(ctx, *dict)
			if err != nil {
				return err
			}
		} else {
			err = writeOPIDictV20(ctx, *dict)
			if err != nil {
				return err
			}
		}

	}

	logInfoWriter.Printf("*** writeOPIVersionDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeMaskStreamDict(ctx *types.PDFContext, streamDict types.PDFStreamDict) (err error) {

	logInfoWriter.Printf("*** writeMaskStreamDict begin: offset=%d ***\n", ctx.Write.Offset)

	if streamDict.Type() != nil && *streamDict.Type() != "XObject" {
		return errors.New("writeMaskStreamDict: corrupt imageStreamDict type")
	}

	if streamDict.Subtype() == nil || *streamDict.Subtype() != "Image" {
		return errors.New("writeMaskStreamDict: corrupt imageStreamDict subtype")
	}

	err = writeImageStreamDict(ctx, streamDict, isNoAlternateImageStreamDict)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeMaskStreamDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeMaskEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, required bool, sinceVersion types.PDFVersion) (err error) {

	// stream ("explicit masking", another Image XObject) or array of colors ("color key masking")

	logInfoWriter.Printf("*** writeMaskEntry begin: offset=%d ***\n", ctx.Write.Offset)

	entryName := "Mask"

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("writeMaskEntry: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		logInfoWriter.Printf("writeMaskEntry end: \"%s\" is nil, offset=%d\n", entryName, ctx.Write.Offset)
		return
	}

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeMaskEntry end: offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		if required {
			return errors.Errorf("writeMaskEntry: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		logInfoWriter.Printf("writeMaskEntry end: offset=%d\n", ctx.Write.Offset)
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeMaskEntry: unsupported in version %s.\n", ctx.VersionString())
	}

	switch obj := obj.(type) {

	case types.PDFStreamDict:
		err = writeMaskStreamDict(ctx, obj)

	case types.PDFArray:
		if ok := validate.ColorKeyMaskArray(obj); !ok {
			err = errors.New("writeMaskEntry: invalid color key mask array")
		}

	default:

		err = errors.Errorf("writeMaskEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	logInfoWriter.Printf("*** writeMaskEntry end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeAlternateImageStreamDicts(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeAlternateImageStreamDicts begin: offset=%d ***\n", ctx.Write.Offset)

	arr, written, err := writeArrayEntry(ctx, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeAlternateImageStreamDicts end: dictName=%s offset=%d\n", dictName, ctx.Write.Offset)
		return
	}

	if arr == nil {
		if required {
			return errors.Errorf("writeAlternateImageStreamDicts: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		logInfoWriter.Printf("writeAlternateImageStreamDicts end: dictName=%s offset=%d\n", dictName, ctx.Write.Offset)
		return
	}

	for _, obj := range *arr {

		streamDict, written, err := writeStreamDict(ctx, obj)
		if err != nil {
			return err
		}

		if written || streamDict == nil {
			continue
		}

		err = writeImageStreamDict(ctx, *streamDict, isAlternateImageStreamDict)
		if err != nil {
			return err
		}
	}

	logInfoWriter.Printf("*** writeAlternateImageStreamDicts begin: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeImageStreamDict(ctx *types.PDFContext, streamDict types.PDFStreamDict, isAlternate bool) (err error) {

	logInfoWriter.Printf("*** writeImageStreamDict begin: offset=%d ***\n", ctx.Write.Offset)

	dict := streamDict.PDFDict

	dictName := "imageStreamDict"

	// Width, integer, required
	_, _, err = writeIntegerEntry(ctx, dict, dictName, "Width", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// Height, integer, required
	_, _, err = writeIntegerEntry(ctx, dict, dictName, "Height", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// ImageMask, boolean, optional
	imageMask, _, err := writeBooleanEntry(ctx, streamDict.PDFDict, "imageStreamDict", "ImageMask", OPTIONAL, types.V10, nil)
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

		// relaxed:
		if streamDict.HasSoleFilterNamed("CCITTFaxDecode") {
			required = OPTIONAL
		}

		err = writeColorSpaceEntry(ctx, streamDict.PDFDict, "imageStreamDict", "ColorSpace", required, ExcludePatternCS)
		if err != nil {
			return
		}

	}

	// BitsPerComponent, integer, TODO required unless used filter is JPXDecode.
	_, _, err = writeIntegerEntry(ctx, dict, dictName, "BitsPerComponent", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// Intent, name, optional, since V1.0
	_, _, err = writeNameEntry(ctx, dict, dictName, "Intent", OPTIONAL, types.V11, validate.RenderingIntent)
	if err != nil {
		return
	}

	// Mask, stream or array, optional since V1.3; not allowed for image masks.
	if !isImageMask {
		err = writeMaskEntry(ctx, dict, dictName, OPTIONAL, types.V13)
		if err != nil {
			return
		}
	}

	// Decode, array, optional
	_, _, err = writeNumberArrayEntry(ctx, dict, dictName, "Decode", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Interpolate, boolean, optional
	_, _, err = writeBooleanEntry(ctx, dict, dictName, "Interpolate", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Alternates, array, optional, since V1.3
	if !isAlternate {
		err = writeAlternateImageStreamDicts(ctx, dict, dictName, "Alternates", OPTIONAL, types.V13)
		if err != nil {
			return
		}
	}

	// SMask, stream, optional, since V1.4
	sd, written, err := writeStreamDictEntry(ctx, dict, dictName, "SMask", OPTIONAL, types.V14, nil)
	if err != nil {
		return
	}

	if !written && sd != nil {
		err = writeImageStreamDict(ctx, *sd, isNoAlternateImageStreamDict)
		if err != nil {
			return
		}
	}

	// SMaskInData, integer, optional TODO since V1.5 if used filter is JPXDecode
	_, _, err = writeIntegerEntry(ctx, dict, dictName, "SMaskInData", OPTIONAL, types.V15, func(i int) bool { return i >= 0 && i <= 2 })
	if err != nil {
		return
	}

	// Name, name, required TODO in V1.0 only.
	// Shall no longer be used.
	_, _, err = writeNameEntry(ctx, dict, dictName, "Name", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// StructParent, integer, TODO required since V1.3 if image is structural content item.
	_, _, err = writeIntegerEntry(ctx, dict, dictName, "StructParent", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// ID, byte string, optional, since V1.3
	_, _, err = writeStringEntry(ctx, dict, dictName, "ID", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// OPI, dict, optional since V1.2
	d, written, err := writeDictEntry(ctx, dict, dictName, "OPI", OPTIONAL, types.V12, nil)
	if err != nil {
		return
	}

	if !written && d != nil {
		err = writeOPIVersionDict(ctx, *d)
		if err != nil {
			return
		}
	}

	// Metadata, stream, optional since V1.4
	// Relaxed to V1.3
	_, err = writeMetadata(ctx, streamDict.PDFDict, OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// OC, dict, optional since V1.5
	d, written, err = writeDictEntry(ctx, dict, dictName, "OC", OPTIONAL, types.V15, nil)
	if err != nil {
		return
	}

	if !written && d != nil {
		err = writeOptionalContentGroupDict(ctx, *d)
		if err != nil {
			return
		}
	}

	logInfoWriter.Printf("*** writeImageStreamDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeGroupAttributesDict(ctx *types.PDFContext, obj interface{}) (written bool, err error) {

	// see 11.6.6 Transparency Group XObjects

	logInfoWriter.Printf("*** writeGroupAttributesDict begin: offset=%d ***\n", ctx.Write.Offset)

	dict, written, err := writeDict(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("*** writeGroupAttributesDict end: object already written, offset=%d ***\n", ctx.Write.Offset)
		return
	}

	dictName := "groupAttributesDict"

	// Type, name, optional
	_, _, err = writeNameEntry(ctx, *dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "Group" })
	if err != nil {
		return
	}

	// S, name, required
	_, _, err = writeNameEntry(ctx, *dict, dictName, "S", REQUIRED, types.V10, func(s string) bool { return s == "Transparency" })
	if err != nil {
		return
	}

	// CS, colorSpace, "sometimes required" TODO restrict allowed colorSpace type
	err = writeColorSpaceEntry(ctx, *dict, dictName, "CS", OPTIONAL, ExcludePatternCS)
	if err != nil {
		return
	}

	// I, boolean, optional
	_, _, err = writeBooleanEntry(ctx, *dict, dictName, "I", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeGroupAttributesDict end: offset=%d ***\n", ctx.Write.Offset)

	written = true

	return
}

func writeFormStreamDict(ctx *types.PDFContext, streamDict types.PDFStreamDict) (err error) {

	// 8.10 Form XObjects

	// pdfTex may produce custom entries:
	//
	//	TODO <PTEX.FileName, (./figures/fig209.pdf)>
	//	<PTEX.InfoDict, (1404 0 R)>
	//	<PTEX.PageNumber, 1>

	logInfoWriter.Printf("*** writeFormStreamDict begin: offset=%d ***\n", ctx.Write.Offset)

	dict := streamDict.PDFDict

	dictName := "formStreamDict"

	_, _, err = writeIntegerEntry(ctx, dict, dictName, "FormType", OPTIONAL, types.V10, func(i int) bool { return i == 1 })
	if err != nil {
		return
	}

	_, _, err = writeRectangleEntry(ctx, dict, dictName, "BBox", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, dict, dictName, "Matrix", OPTIONAL, types.V10, func(arr types.PDFArray) bool { return len(arr) == 6 })
	if err != nil {
		return
	}

	// Resources, dict, optional, since V1.2
	if obj, ok := streamDict.Find("Resources"); ok {
		_, err = writeResourceDict(ctx, obj)
	}

	// Group, dict, optional, since V1.4
	err = writePageEntryGroup(ctx, dict, OPTIONAL, types.V14)
	if err != nil {
		return
	}

	// Ref, dict, optional, since V1.4
	d, written, err := writeDictEntry(ctx, dict, dictName, "Ref", OPTIONAL, types.V14, nil)
	if err != nil {
		return
	}

	if !written && d != nil {
		err = writeReferenceDict(ctx, *d)
		if err != nil {
			return
		}
	}

	// Metadata, stream, optional, since V1.4
	_, err = writeMetadata(ctx, streamDict.PDFDict, OPTIONAL, types.V14)
	if err != nil {
		return
	}

	// PieceInfo, dict, optional, since V1.3
	hasPieceInfo, err := writePieceInfo(ctx, streamDict.PDFDict, OPTIONAL, types.V13)
	if err != nil {
		return err
	}

	// LastModified, date, required if PieceInfo present, since V1.3
	lm, _, err := writeDateEntry(ctx, dict, dictName, "LastModified", OPTIONAL, types.V13)
	if err != nil {
		return
	}

	if hasPieceInfo && lm == nil {
		err = errors.New("writeFormStreamDict: missing \"LastModified\" (required by \"PieceInfo\")")
		return
	}

	// StructParent, integer // TODO validate either StructParent or StructParents entry present,
	_, _, err = writeIntegerEntry(ctx, dict, dictName, "StructParent", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// StructParents, integer
	_, _, err = writeIntegerEntry(ctx, dict, dictName, "StructParents", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// OPI, dict, optional, since V1.2
	d, written, err = writeDictEntry(ctx, dict, dictName, "OPI", OPTIONAL, types.V12, nil)
	if err != nil {
		return
	}

	if !written && d != nil {
		err = writeOPIVersionDict(ctx, *d)
		if err != nil {
			return
		}
	}

	// OC, dict, optional, since V1.5
	d, written, err = writeDictEntry(ctx, dict, dictName, "OC", OPTIONAL, types.V15, nil)
	if err != nil {
		return
	}

	if !written && d != nil {
		err = writeOptionalContentGroupDict(ctx, *d)
		if err != nil {
			return
		}
	}

	// Name, name, optional (required in 1.0)
	required := ctx.Version() == types.V10
	_, _, err = writeNameEntry(ctx, dict, dictName, "Name", required, types.V10, nil)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeFormStreamDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeXObjectStreamDict(ctx *types.PDFContext, streamDict types.PDFStreamDict) (err error) {

	// see 8.8 External Objects

	logInfoWriter.Printf("*** writeXObjectStreamDict begin: offset=%d ***\n", ctx.Write.Offset)

	_, _, err = writeNameEntry(ctx, streamDict.PDFDict, "xObjectStreamDict", "Type", OPTIONAL, types.V10, func(s string) bool { return s == "XObject" })
	if err != nil {
		return
	}

	required := REQUIRED
	if ctx.XRefTable.ValidationMode == types.ValidationRelaxed {
		required = OPTIONAL
	}
	subtype, _, err := writeNameEntry(ctx, streamDict.PDFDict, "xObjectStreamDict", "Subtype", required, types.V10, nil)
	if err != nil {
		return
	}

	if subtype == nil {

		// relaxed
		_, found := streamDict.Find("BBox")
		if found {

			err = writeFormStreamDict(ctx, streamDict)
			if err != nil {
				return
			}

			logInfoWriter.Println("writeXObjectStreamDict end")
			return
		}

		// Relaxed for page Thumb
		err = writeImageStreamDict(ctx, streamDict, isNoAlternateImageStreamDict)
		if err != nil {
			return
		}

		logInfoWriter.Printf("writeXObjectStreamDict end: offset=%d\n", ctx.Write.Offset)
		return
	}

	switch *subtype {

	case "Form":
		err = writeFormStreamDict(ctx, streamDict)

	case "Image":
		err = writeImageStreamDict(ctx, streamDict, isNoAlternateImageStreamDict)

	case "PS":
		err = errors.Errorf("writeXObjectStreamDict: PostScript XObjects should not be used")

	default:
		err = errors.Errorf("writeXObjectStreamDict: unknown Subtype: %s\n", *subtype)

	}

	logInfoWriter.Printf("*** writeXObjectStreamDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeXObjectResourceDict(ctx *types.PDFContext, obj interface{}) (err error) {

	logInfoWriter.Printf("*** writeXObjectResourceDict begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return err
	}

	if written {
		logInfoWriter.Printf("writeXObjectResourceDict end: object already written. offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeXObjectResourceDict end: object is nil. offset=%d\n", ctx.Write.Offset)
		return
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		return errors.New("writeXObjectResourceDict: corrupt font dict")
	}

	// Iterate over xObject resource dictionary
	for _, obj := range dict.Dict {

		obj, written, err = writeObject(ctx, obj)
		if err != nil {
			return
		}

		if written {
			logInfoWriter.Printf("writeXObjectResourceDict end: font resource object already written. offset=%d\n", ctx.Write.Offset)
			continue
		}

		if obj == nil {
			logInfoWriter.Printf("writeXObjectResourceDict end: font resource object is nil. offset=%d\n", ctx.Write.Offset)
			continue
		}

		streamDict, ok := obj.(types.PDFStreamDict)
		if !ok {
			return errors.New("writeXObjectResourceDict: corrupt font dict")
		}

		// Process XObject dict
		err = writeXObjectStreamDict(ctx, streamDict)
		if err != nil {
			return
		}
	}

	logInfoWriter.Printf("*** writeXObjectResourceDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}
