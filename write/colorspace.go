package write

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/hhrutter/pdflib/validate"
	"github.com/pkg/errors"
)

func writeCalGrayColorSpace(ctx *types.PDFContext, arr types.PDFArray, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeCalGrayColorSpace begin: offset=%d ***\n", ctx.Write.Offset)

	if len(arr) != 2 {
		return errors.Errorf("writeCalGrayColorSpace: invalid array length %d (expected 2) \n.", len(arr))
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeCalGrayColorSpace: unsupported in version %s.\n", ctx.VersionString())
	}

	dict, written, err := writeDict(ctx, arr[1])
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeCalGrayColorSpace end: offset=%d\n", ctx.Write.Offset)
		return
	}

	if dict == nil {
		logInfoWriter.Printf("writeCalGrayColorSpace end: offset=%d\n", ctx.Write.Offset)
		return
	}

	dictName := "calGrayCSDict"

	_, _, err = writeNumberArrayEntry(ctx, *dict, dictName, "WhitePoint", REQUIRED, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 3 })
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, *dict, dictName, "BlackPoint", OPTIONAL, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 3 })
	if err != nil {
		return
	}

	_, _, err = writeNumberEntry(ctx, *dict, dictName, "Gamma", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeCalGrayColorSpace end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeCalRGBColorSpace(ctx *types.PDFContext, arr types.PDFArray, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeCalRGBColorSpace begin: offset=%d ***\n", ctx.Write.Offset)

	if len(arr) != 2 {
		return errors.Errorf("writeCalRGBColorSpace: invalid array length %d (expected 2) \n.", len(arr))
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeCalRGBColorSpace: unsupported in version %s.\n", ctx.VersionString())
	}

	dict, written, err := writeDict(ctx, arr[1])
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeCalRGBColorSpace end: offset=%d\n", ctx.Write.Offset)
		return
	}

	if dict == nil {
		logInfoWriter.Printf("writeCalRGBColorSpace end: offset=%d\n", ctx.Write.Offset)
		return
	}

	dictName := "calRGBCSDict"

	_, _, err = writeNumberArrayEntry(ctx, *dict, dictName, "WhitePoint", REQUIRED, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 3 })
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, *dict, dictName, "BlackPoint", OPTIONAL, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 3 })
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, *dict, dictName, "Gamma", OPTIONAL, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 3 })
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, *dict, dictName, "Matrix", OPTIONAL, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 9 })
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeCalRGBColorSpace end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeLabColorSpace(ctx *types.PDFContext, arr types.PDFArray, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeLabColorSpace begin: offset=%d ***\n", ctx.Write.Offset)

	if len(arr) != 2 {
		return errors.Errorf("writeLabColorSpace: invalid array length %d (expected 2) \n.", len(arr))
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeLabColorSpace: unsupported in version %s.\n", ctx.VersionString())
	}

	dict, written, err := writeDict(ctx, arr[1])
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeLabColorSpace end: offset=%d\n", ctx.Write.Offset)
		return
	}

	if dict == nil {
		logInfoWriter.Printf("writeLabColorSpace end: offset=%d\n", ctx.Write.Offset)
		return
	}

	dictName := "labCSDict"

	_, _, err = writeNumberArrayEntry(ctx, *dict, dictName, "WhitePoint", REQUIRED, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 3 })
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, *dict, dictName, "BlackPoint", OPTIONAL, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 3 })
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, *dict, dictName, "Range", OPTIONAL, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 4 })
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeLabColorSpace end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeICCBasedColorSpace(ctx *types.PDFContext, arr types.PDFArray, sinceVersion types.PDFVersion) (err error) {

	// see 8.6.5.5

	logInfoWriter.Printf("*** writeICCBasedColorSpace begin: offset=%d ***\n", ctx.Write.Offset)

	if len(arr) != 2 {
		return errors.Errorf("writeICCBasedColorSpace: invalid array length %d (expected 2) \n.", len(arr))
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeICCBasedColorSpace: unsupported in version %s.\n", ctx.VersionString())
	}

	streamDict, written, err := writeStreamDict(ctx, arr[1])
	if err != nil {
		return
	}

	if written || streamDict == nil {
		return
	}

	dict := streamDict.PDFDict
	dictName := "ICCBasedColorSpace"

	N, _, err := writeIntegerEntry(ctx, dict, dictName, "N", REQUIRED, sinceVersion, validate.ICCBasedColorSpaceEntryN)
	if err != nil {
		return
	}

	// TODO sinceVersion
	err = writeColorSpaceEntry(ctx, dict, dictName, "Alternate", OPTIONAL, ExcludePatternCS)
	if err != nil {
		return
	}

	_, _, err = writeNumberArrayEntry(ctx, dict, dictName, "Range", OPTIONAL, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 2*N.Value() })
	if err != nil {
		return
	}

	// Metadata, stream, optional since V1.4
	_, err = writeMetadata(ctx, dict, OPTIONAL, types.V14)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeICCBasedColorSpace end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeIndexedColorSpaceLookuptable(ctx *types.PDFContext, obj interface{}, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeIndexedColorSpaceLookupTable begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeIndexedColorSpaceLookuptable end: already written\n")
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeIndexedColorSpaceLookuptable end: is nil\n")
		return
	}

	switch obj.(type) {

	case types.PDFStringLiteral:
		if ctx.Version() < types.V12 {
			err = errors.Errorf("writeIndexedColorSpaceLookupTable: string literal unsupported in version %s.\n", ctx.VersionString())
		}

	case types.PDFHexLiteral:
		if ctx.Version() < types.V12 {
			err = errors.Errorf("writeIndexedColorSpaceLookupTable: hex literal unsupported in version %s.\n", ctx.VersionString())
		}

	case types.PDFStreamDict:
		// no further processing

	default:
		err = errors.Errorf("writeIndexedColorSpaceLookuptable: invalid type\n")

	}

	logInfoWriter.Printf("*** writeIndexedColorSpaceLookupTable end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeIndexedColorSpace(ctx *types.PDFContext, arr types.PDFArray, sinceVersion types.PDFVersion) (err error) {

	// see 8.6.6.3

	logInfoWriter.Printf("*** writeIndexedColorSpace begin: offset=%d ***\n", ctx.Write.Offset)

	if len(arr) != 4 {
		return errors.Errorf("writeIndexedColorSpace: invalid array length %d (expected 4) \n.", len(arr))
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeIndexedColorSpace: unsupported in version %s.\n", ctx.VersionString())
	}

	// arr[1] base: basecolorspace noPatternCS, TODO noIndexedCS
	err = writeColorSpace(ctx, arr[1], ExcludePatternCS)
	if err != nil {
		return
	}

	// arr[2] hival: 0 <= int <= 255
	_, _, err = writeInteger(ctx, arr[2], func(i int) bool { return i >= 0 && i <= 255 })
	if err != nil {
		return
	}

	// arr[3] lookup: stream or TODO byte string(since V1.2)
	err = writeIndexedColorSpaceLookuptable(ctx, arr[3], sinceVersion)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeIndexedColorSpace end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePatternColorSpace(ctx *types.PDFContext, arr types.PDFArray, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writePatternColorSpace begin: offset=%d ***\n", ctx.Write.Offset)

	if len(arr) < 1 || len(arr) > 2 {
		return errors.Errorf("writePatternColorSpace: invalid array length %d (expected 1 or 2) \n.", len(arr))
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writePatternColorSpace: unsupported in version %s.\n", ctx.VersionString())
	}

	// 8.7.3.3: arr[1]: name of underlying color space, any cs except PatternCS
	if len(arr) == 2 {
		err = writeColorSpace(ctx, arr[1], ExcludePatternCS)
	}

	logInfoWriter.Printf("*** writePatternColorSpace end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeSeparationColorSpace(ctx *types.PDFContext, arr types.PDFArray, sinceVersion types.PDFVersion) (err error) {

	// see 8.6.6.4

	logInfoWriter.Printf("*** writeSeparationColorSpace begin: offset=%d ***\n", ctx.Write.Offset)

	if len(arr) != 4 {
		return errors.Errorf("writeSeparationColorSpace: invalid array length %d (expected 4) \n.", len(arr))
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeSeparationColorSpace: unsupported in version %s.\n", ctx.VersionString())
	}

	// arr[1]: colorant name, arbitrary
	_, _, err = writeName(ctx, arr[1], nil)
	if err != nil {
		return
	}

	// arr[2]: alternate space: TODO noSpecialCS
	err = writeColorSpace(ctx, arr[2], ExcludePatternCS)
	if err != nil {
		return
	}

	// arr[3]: tintTransform, function
	_, err = writeFunction(ctx, arr[3])
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeSeparationColorSpace end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeDeviceNColorSpaceColorantsDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeDeviceNColorSpaceColorantsDict begin: offset=%d ***\n", ctx.Write.Offset)

	for _, obj := range dict.Dict {

		arr, written, err := writeArray(ctx, obj)
		if err != nil {
			return err
		}

		if !written && arr != nil {
			writeSeparationColorSpace(ctx, *arr, types.V12)
		}

	}

	logInfoWriter.Printf("*** writeDeviceNColorSpaceColorantsDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeDeviceNColorSpaceProcessDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// ColorSpace, required, colorSpace
	// Components, required, array
	// e.g.,
	//	<<
	//	  <ColorSpace, DeviceCMYK>
	//	  <Components, [Cyan Magenta Yellow Black]>
	//  >>

	logInfoWriter.Printf("*** writeDeviceNColorSpaceProcessDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "DeviceNCSProcessDict"

	// TODO only Device or CIE colorspace allowed
	err = writeColorSpaceEntry(ctx, dict, dictName, "ColorSpace", REQUIRED, true)
	if err != nil {
		return
	}

	_, _, err = writeNameArrayEntry(ctx, dict, dictName, "Components", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeDeviceNColorSpaceProcessDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeDeviceNColorSpaceSoliditiesDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeDeviceNColorSpaceSoliditiesDict begin: offset=%d ***\n", ctx.Write.Offset)

	for _, obj := range dict.Dict {
		_, _, err = writeFloat(ctx, obj, func(f float64) bool { return f >= 0.0 && f <= 1.0 })
		if err != nil {
			return
		}
	}

	logInfoWriter.Printf("*** writeDeviceNColorSpaceSoliditiesDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeDeviceNColorSpaceDotGainDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeDeviceNColorSpaceDotGainDict begin: offset=%d ***\n", ctx.Write.Offset)

	for _, obj := range dict.Dict {
		_, err = writeFunction(ctx, obj)
		if err != nil {
			return
		}
	}

	logInfoWriter.Printf("*** writeDeviceNColorSpaceDotGainDict end: offset=%d***\n", ctx.Write.Offset)

	return
}

func writeDeviceNColorSpaceMixingHintsDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeDeviceNColorSpaceMixingHintsDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "deviceNCSMixingHintsDict"

	d, written, err := writeDictEntry(ctx, dict, dictName, "Solidities", OPTIONAL, types.V11, nil)
	if err != nil {
		return
	}

	if !written && d != nil {
		err = writeDeviceNColorSpaceSoliditiesDict(ctx, *d)
		if err != nil {
			return
		}
	}

	_, _, err = writeNameArrayEntry(ctx, dict, dictName, "PrintingOrder", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	d, written, err = writeDictEntry(ctx, dict, dictName, "DotGain", OPTIONAL, types.V11, nil)
	if err != nil {
		return
	}

	if !written && d != nil {
		err = writeDeviceNColorSpaceDotGainDict(ctx, *d)
		if err != nil {
			return
		}
	}

	logInfoWriter.Printf("*** writeDeviceNColorSpaceMixingHintsDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeDeviceNColorSpaceAttributesDict(ctx *types.PDFContext, obj interface{}) (err error) {

	logInfoWriter.Printf("*** writeDeviceNColorSpaceAttributesDict begin: offset=%d ***\n", ctx.Write.Offset)

	dict, written, err := writeDict(ctx, obj)
	if err != nil {
		return
	}

	if written || dict == nil {
		return
	}

	dictName := "deviceNCSAttributesDict"

	_, _, err = writeNameEntry(ctx, *dict, dictName, "SubType", OPTIONAL, types.V16, func(s string) bool { return s == "DeviceN" || s == "NChannel" })
	if err != nil {
		return
	}

	d, written, err := writeDictEntry(ctx, *dict, dictName, "Colorants", OPTIONAL, types.V11, nil)
	if err != nil {
		return
	}

	if !written && d != nil {
		err = writeDeviceNColorSpaceColorantsDict(ctx, *d)
		if err != nil {
			return
		}
	}

	// TODO Relaxed from 1.6 to 1.4
	version := types.V16
	if ctx.XRefTable.ValidationMode == types.ValidationRelaxed {
		version = types.V14
	}
	d, written, err = writeDictEntry(ctx, *dict, dictName, "Process", OPTIONAL, version, nil)
	if err != nil {
		return
	}

	if !written && d != nil {
		err = writeDeviceNColorSpaceProcessDict(ctx, *d)
		if err != nil {
			return
		}
	}

	d, written, err = writeDictEntry(ctx, *dict, dictName, "MixingHints", OPTIONAL, types.V16, nil)
	if err != nil {
		return
	}

	if !written && d != nil {
		err = writeDeviceNColorSpaceMixingHintsDict(ctx, *d)
		if err != nil {
			return
		}
	}

	logInfoWriter.Printf("*** writeDeviceNAttributesDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeDeviceNColorSpace(ctx *types.PDFContext, arr types.PDFArray, sinceVersion types.PDFVersion) (err error) {

	// see 8.6.6.5

	logInfoWriter.Printf("*** writeDeviceNColorSpace begin: offset=%d ***\n", ctx.Write.Offset)

	if len(arr) < 4 || len(arr) > 5 {
		return errors.Errorf("writeDeviceNColorSpace: invalid array length %d (expected 4 or 5) \n.", len(arr))
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeDeviceNColorSpace: unsupported in version %s.\n", ctx.VersionString())
	}

	// arr[1]: array of names specifying the individual color components
	// length subject to implementation limit.
	// TODO validation??
	_, _, err = writeNameArray(ctx, arr[1])
	if err != nil {
		return
	}

	// arr[2]: alternate space: TODO noSpecialCS
	err = writeColorSpace(ctx, arr[2], ExcludePatternCS)
	if err != nil {
		return
	}

	// arr[3]: tintTransform, function
	_, err = writeFunction(ctx, arr[3])
	if err != nil {
		return
	}

	// arr[4]: color space attributes dict, optional
	if len(arr) == 5 {
		err = writeDeviceNColorSpaceAttributesDict(ctx, arr[4])
		if err != nil {
			return
		}
	}

	logInfoWriter.Printf("*** writeDeviceNColorSpace end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeColorSpaceArray(ctx *types.PDFContext, arr types.PDFArray, excludePatternCS bool) (err error) {

	// see 8.6 Color Spaces

	logInfoWriter.Printf("*** writeColorSpaceArray begin: offset=%d ***\n", ctx.Write.Offset)

	name, ok := arr[0].(types.PDFName)
	if !ok {
		return errors.New("writeColorSpaceArray: corrupt Colorspace dict")
	}

	switch name {

	// CIE-based
	case "CalGray":
		err = writeCalGrayColorSpace(ctx, arr, types.V11)

	case "CalRGB":
		err = writeCalRGBColorSpace(ctx, arr, types.V11)

	case "Lab":
		err = writeLabColorSpace(ctx, arr, types.V11)

	case "ICCBased":
		err = writeICCBasedColorSpace(ctx, arr, types.V13)

	// Special
	case "Indexed":
		err = writeIndexedColorSpace(ctx, arr, types.V11)

	case "Pattern":
		if excludePatternCS {
			return errors.New("writeColorSpaceArray: Pattern color space not allowed")
		}
		err = writePatternColorSpace(ctx, arr, types.V12)

	case "Separation":
		err = writeSeparationColorSpace(ctx, arr, types.V12)

	case "DeviceN":
		err = writeDeviceNColorSpace(ctx, arr, types.V13)

	default:
		err = errors.Errorf("writeColorSpaceArray: undefined color space: %s\n", name)
	}

	if err == nil {
		logInfoWriter.Printf("*** writeColorSpaceArray end: offset=%d ***\n", ctx.Write.Offset)
	}

	return
}

func writeColorSpaceEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string, required bool, excludePatternCS bool) (err error) {

	logInfoWriter.Printf("*** writeColorSpaceEntry begin: dictName=%s offset=%d ***\n", dictName, ctx.Write.Offset)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("writeColorSpaceEntry: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		logInfoWriter.Printf("writeColorSpaceEntry end: \"%s\" is nil, offset=%d\n", entryName, ctx.Write.Offset)
		return
	}

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeColorSpaceEntry end: dictName=%s offset=%d\n", dictName, ctx.Write.Offset)
		return
	}

	if obj == nil {
		if required {
			return errors.Errorf("writeColorSpaceEntry: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		logInfoWriter.Printf("writeColorSpaceEntry end: dictName=%s offset=%d\n", dictName, ctx.Write.Offset)
		return
	}

	switch obj := obj.(type) {

	case types.PDFName:
		if ok := validate.DeviceColorSpaceName(obj.String()); !ok {
			err = errors.Errorf("writeColorSpaceEntry: Name:%s\n", obj.String())
		}

	case types.PDFArray:
		err = writeColorSpaceArray(ctx, obj, excludePatternCS)

	default:
		err = errors.Errorf("writeColorSpaceEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	logInfoWriter.Printf("*** writeColorSpaceEntry end: dictName=%s offset=%d ***\n", dictName, ctx.Write.Offset)

	return
}

func writeColorSpace(ctx *types.PDFContext, obj interface{}, excludePatternCS bool) (err error) {

	logInfoWriter.Printf("*** writeColorSpace begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return err
	}

	if written {
		logInfoWriter.Printf("writeColorSpace end: resource object already written. offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeColorSpace end: resource object is nil. offset=%d\n", ctx.Write.Offset)
		return
	}

	switch obj := obj.(type) {

	case types.PDFName:
		if ok := validate.DeviceColorSpaceName(obj.String()) || validate.SpecialColorSpaceName(obj.String()); !ok {
			err = errors.Errorf("writeColorSpace: invalid device color space name: %v\n", obj)
		}

	case types.PDFArray:
		err = writeColorSpaceArray(ctx, obj, excludePatternCS)

	default:
		err = errors.New("writeColorSpace: corrupt obj typ, must be Name or Array")

	}

	logInfoWriter.Printf("*** writeColorSpace end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeColorSpaceResourceDict(ctx *types.PDFContext, obj interface{}) (err error) {

	// see 8.6 Color Spaces

	logInfoWriter.Printf("*** writeColorSpaceResourceDict begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return err
	}

	if written {
		logInfoWriter.Printf("writeColorSpaceResourceDict end: object already written. offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeColorSpaceResourceDict end: object is nil. offset=%d\n", ctx.Write.Offset)
		return
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		return errors.New("writeColorSpaceResourceDict: corrupt dict")
	}

	// Iterate over colorspace resource dictionary
	for key, obj := range dict.Dict {

		logInfoWriter.Printf("writeColorSpaceResourceDict: processing entry: %s\n", key)

		// Process colorspace
		err = writeColorSpace(ctx, obj, IncludePatternCS)
		if err != nil {
			return
		}
	}

	logInfoWriter.Printf("*** writeColorSpaceResourceDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}
