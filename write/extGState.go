package write

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/hhrutter/pdflib/validate"
	"github.com/pkg/errors"
)

func writeSoftMaskTransferFunctionEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeSoftMaskTransferFunctionEntry begin: offset=%d ***\n", ctx.Write.Offset)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("writeSoftMaskTransferFunctionEntry: dict=%s required entry=%s missing.", dictName, entryName)
		}
		logInfoWriter.Printf("writeSoftMaskTransferFunctionEntry end: entry %s is nil\n", entryName)
		return
	}

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		return
	}

	if obj == nil {
		if required {
			return errors.Errorf("writeSoftMaskTransferFunctionEntry: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		// already written
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeSoftMaskTransferFunctionEntry: dict=%s entry=%s unsupported in version %s.\n", dictName, entryName, ctx.VersionString())
	}

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		if s != "Identity" {
			return errors.New("writeSoftMaskTransferFunctionEntry: corrupt name")
		}

	case types.PDFDict:
		err = processFunction(ctx, obj)
		if err != nil {
			return
		}

	case types.PDFStreamDict:
		err = processFunction(ctx, obj)
		if err != nil {
			return
		}

	default:
		return errors.Errorf("writeSoftMaskTransferFunctionEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	logInfoWriter.Printf("*** writeSoftMaskTransferFunctionEntry end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeSoftMaskDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// see 11.6.5.2

	logInfoWriter.Printf("*** writeSoftMaskDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "softMaskDict"

	// Type, name, optional
	_, _, err = writeNameEntry(ctx, dict, dictName, "Type", OPTIONAL, types.V10, func(s string) bool { return s == "Mask" })
	if err != nil {
		return err
	}

	// S, name, required
	_, _, err = writeNameEntry(ctx, dict, dictName, "S", REQUIRED, types.V10, func(s string) bool { return s == "Alpha" || s == "Luminosity" })
	if err != nil {
		return
	}

	// G, stream, required
	// A transparency group XObject (see “Transparency Group XObjects”)
	// to be used as the source of alpha or colour values for deriving the mask.
	streamDict, written, err := writeStreamDictEntry(ctx, dict, dictName, "G", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	if !written && streamDict != nil {
		err = writeXObjectStreamDict(ctx, *streamDict)
		if err != nil {
			return
		}
	}

	// TR (Optional) function or name
	// A function object (see “Functions”) specifying the transfer function
	// to be used in deriving the mask values.
	err = writeSoftMaskTransferFunctionEntry(ctx, dict, dictName, "TR", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// BC, number array, optional
	// Array of component values specifying the colour to be used
	// as the backdrop against which to composite the transparency group XObject G.
	_, _, err = writeNumberArrayEntry(ctx, dict, dictName, "BC", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeSoftMaskDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeLineDashPatternEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeLineDashPatternEntry begin: entry=%s offset=%d ***\n", entryName, ctx.Write.Offset)

	arr, written, err := writeArrayEntry(ctx, dict, dictName, entryName, required, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 2 })
	if err != nil {
		return
	}

	if written || arr == nil {
		// optional and nil or already written
		logInfoWriter.Printf("writeLineDashPatternEntry end: offset=%d\n", ctx.Write.Offset)
		return
	}

	a := *arr

	// dash array (user space units)
	array, ok := a[0].(types.PDFArray)
	if !ok {
		return errors.Errorf("writeLineDashPatternEntry: dict=%s entry \"%s\" corrupt dash array.", dictName, entryName)
	}

	_, _, err = writeIntegerArray(ctx, array)
	if err != nil {
		return
	}

	// dash phase (user space units)
	//i, ok := a[1].(PDFInteger)
	//if !ok {
	//	log.Fatalf("writeLineDashPatternEntry: dict=%s entry \"%s\" corrupt dash phase.", dictName, entryName)
	//}
	_, _, err = writeInteger(ctx, a[1], nil)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeLineDashPatternEntry end: entry=%s offset=%d ***\n", entryName, ctx.Write.Offset)

	return
}

func writeBlendModeEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeBlendModeEntry begin: offset=%d ***\n", ctx.Write.Offset)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("writeBlendModeEntry: dict=%s required entry \"%s\" missing.", entryName, dictName)
		}
		logInfoWriter.Printf("writeBlendModeEntry end: \"%s\" is nil, offset=%d\n", entryName, ctx.Write.Offset)
		return
	}

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return err
	}

	if written {
		return
	}

	if obj == nil {
		if required {
			return errors.Errorf("writeBlendModeEntry: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		// already written
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeBlendModeEntry: dict=%s entry=%s unsupported in version %s.\n", dictName, entryName, ctx.VersionString())
	}

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		if !validate.BlendMode(s) {
			return errors.Errorf("writeBlendModeEntry: invalid blend mode: %s\n", s)
		}

	case types.PDFArray:
		for _, obj := range obj {

			obj, written, err := writeObject(ctx, obj)
			if err != nil {
				return err
			}

			if written || obj == nil {
				continue
			}

			name, ok := obj.(types.PDFName)
			if !ok {
				return errors.Errorf("writeBlendModeEntry: corrupt blend mode array entry\n")
			}

			s := name.String()
			if !validate.BlendMode(s) {
				return errors.Errorf("writeBlendModeEntry: invalid blend mode array entry: %s\n", s)
			}
		}

	default:
		return errors.Errorf("writeBlendModeEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	logInfoWriter.Printf("*** writeBlendModeEntry end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeSoftMaskEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	// see 11.3.7.2 Source Shape and Opacity
	// see 11.6.4.3 Mask Shape and Opacity

	logInfoWriter.Printf("*** writeSoftMaskEntry begin: offset=%d ***\n", ctx.Write.Offset)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("writeSoftMaskEntry: dict=%s required entry \"%s\" missing.", entryName, dictName)
		}
		logInfoWriter.Printf("writeSoftMaskEntry end: \"%s\" is nil, offset=%d\n", entryName, ctx.Write.Offset)
		return
	}

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeSoftMaskEntry end: offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		if required {
			return errors.Errorf("writeSoftMaskEntry: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		logInfoWriter.Printf("writeSoftMaskEntry end: offset=%d\n", ctx.Write.Offset)
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeSoftMaskEntry: dict=%s entry=%s unsupported in version %s.\n", dictName, entryName, ctx.VersionString())
	}

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		if !validate.BlendMode(s) {
			err = errors.Errorf("writeSoftMaskEntry: invalid soft mask: %s\n", s)
		}

	case types.PDFDict:
		err = writeSoftMaskDict(ctx, obj)

	default:
		err = errors.Errorf("writeSoftMaskEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	logInfoWriter.Printf("*** writeSoftMaskEntry end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeBG2Entry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeBG2Entry begin: offset=%d ***\n", ctx.Write.Offset)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("writeBG2Entry: dict=%s required entry=%s missing.", dictName, entryName)
		}
		logInfoWriter.Printf("writeBG2Entry end: entry %s is nil\n", entryName)
		return
	}

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return err
	}

	if written {
		logInfoWriter.Printf("writeBG2Entry end: offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		if required {
			return errors.Errorf("writeBG2Entry: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		logInfoWriter.Printf("writeBG2Entry end: offset=%d\n", ctx.Write.Offset)
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeBG2Entry: dict=%s entry=%s unsupported in version %s.\n", dictName, entryName, ctx.VersionString())
	}

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		if s != "Default" {
			err = errors.New("writeBG2Entry: corrupt name")
		}

	case types.PDFDict:
		err = processFunction(ctx, obj)

	case types.PDFStreamDict:
		err = processFunction(ctx, obj)

	default:
		err = errors.Errorf("writeBG2Entry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	logInfoWriter.Printf("*** writeBG2Entry end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeUCR2Entry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeUCR2Entry begin: offset=%d ***\n", ctx.Write.Offset)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("writeUCR2Entry: dict=%s required entry=%s missing.", dictName, entryName)
		}
		logInfoWriter.Printf("writeUCR2Entry end: entry %s is nil\n", entryName)
		return
	}

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeUCR2Entry end: offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		if required {
			return errors.Errorf("writeUCR2Entry: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		logInfoWriter.Printf("writeUCR2Entry end: offset=%d\n", ctx.Write.Offset)
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeUCR2Entry: dict=%s entry=%s unsupported in version %s.\n", dictName, entryName, ctx.VersionString())
	}

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		if s != "Default" {
			err = errors.New("writeUCR2Entry: corrupt name")
		}

	case types.PDFDict:
		err = processFunction(ctx, obj)

	case types.PDFStreamDict:
		err = processFunction(ctx, obj)

	default:
		err = errors.Errorf("writeUCR2Entry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	logInfoWriter.Printf("*** writeUCR2Entry end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeTransferFunctionEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeTransferFunctionEntry begin: offset=%d ***\n", ctx.Write.Offset)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("writeTransferFunctionEntry: dict=%s required entry=%s missing.", dictName, entryName)
		}
		logInfoWriter.Printf("writeTransferFunctionEntry end: entry %s is nil\n", entryName)
		return
	}

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeTransferFunctionEntry end: offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		if required {
			return errors.Errorf("writeTransferFunctionEntry: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		logInfoWriter.Printf("writeTransferFunctionEntry end: offset=%d\n", ctx.Write.Offset)
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeTransferFunctionEntry: dict=%s entry=%s unsupported in version %s.\n", dictName, entryName, ctx.VersionString())
	}

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		if s != "Identity" {
			err = errors.New("writeTransferFunctionEntry: corrupt name")
		}

	case types.PDFArray:

		if len(obj) != 4 {
			return errors.New("writeTransferFunctionEntry: corrupt function array")
		}

		for _, obj := range obj {

			obj, written, err = writeObject(ctx, obj)
			if err != nil {
				return
			}

			if written || obj == nil {
				continue
			}

			err = processFunction(ctx, obj)
			if err != nil {
				return
			}

		}

	case types.PDFDict:
		err = processFunction(ctx, obj)

	case types.PDFStreamDict:
		err = processFunction(ctx, obj)

	default:
		err = errors.Errorf("writeTransferFunctionEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	logInfoWriter.Printf("*** writeTransferFunctionEntry end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeTR2Entry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeTR2Entry begin: offset=%d ***\n", ctx.Write.Offset)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("writeTR2Entry: dict=%s required entry=%s missing.", dictName, entryName)
		}
		logInfoWriter.Printf("writeTR2Entry end: entry %s is nil\n", entryName)
		return
	}

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeTR2Entry end: offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		if required {
			return errors.Errorf("writeTR2Entry: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		logInfoWriter.Printf("writeTR2Entry end: offset=%d\n", ctx.Write.Offset)
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeTR2Entry: dict=%s entry=%s unsupported in version %s.\n", dictName, entryName, ctx.VersionString())
	}

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		if s != "Identity" && s != "Default" {
			err = errors.Errorf("writeTR2Entry: corrupt name\n")
		}

	case types.PDFArray:

		if len(obj) != 4 {
			return errors.New("writeTR2Entry: corrupt function array")
		}

		for _, obj := range obj {

			obj, written, err = writeObject(ctx, obj)
			if err != nil {
				return
			}

			if written || obj == nil {
				continue
			}

			err = processFunction(ctx, obj)
			if err != nil {
				return
			}

		}

	case types.PDFDict:
		err = processFunction(ctx, obj)

	case types.PDFStreamDict:
		err = processFunction(ctx, obj)

	default:
		err = errors.Errorf("writeTR2Entry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	logInfoWriter.Printf("*** writeTR2Entry end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeSpotFunctionEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeSpotFunctionEntry begin: offset=%d ***\n", ctx.Write.Offset)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("writeSpotFunctionEntry: dict=%s required entry=%s missing.", dictName, entryName)
		}
		logInfoWriter.Printf("writeSpotFunctionEntry end: entry %s is nil\n", entryName)
		return
	}

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeSpotFunctionEntry end: offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		if required {
			return errors.Errorf("writeSpotFunctionEntry: dict=%s required entry \"%s\" missing.", dictName, entryName)
		}
		logInfoWriter.Printf("writeSpotFunctionEntry end: offset=%d\n", ctx.Write.Offset)
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeSpotFunctionEntry: dict=%s entry=%s unsupported in version %s.\n", dictName, entryName, ctx.VersionString())
	}

	switch obj := obj.(type) {

	case types.PDFName:
		s := obj.String()
		if !validate.SpotFunctionName(s) {
			err = errors.Errorf("writeSpotFunctionEntry: corrupt name\n")
		}

	case types.PDFDict:
		err = processFunction(ctx, obj)

	case types.PDFStreamDict:
		err = processFunction(ctx, obj)

	default:
		err = errors.Errorf("writeSpotFunctionEntry: dict=%s corrupt entry \"%s\"\n", dictName, entryName)

	}

	logInfoWriter.Printf("*** writeSpotFunctionEntry end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeType1HalftoneDict(ctx *types.PDFContext, dict types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeType1HalftoneDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "type1HalftoneDict"

	_, _, err = writeStringEntry(ctx, dict, dictName, "HalftoneName", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	_, _, err = writeNumberEntry(ctx, dict, dictName, "Frequency", REQUIRED, sinceVersion)
	if err != nil {
		return
	}

	_, _, err = writeNumberEntry(ctx, dict, dictName, "Angle", REQUIRED, sinceVersion)
	if err != nil {
		return
	}

	err = writeSpotFunctionEntry(ctx, dict, dictName, "Spotfunction", REQUIRED, sinceVersion)
	if err != nil {
		return
	}

	err = writeTransferFunctionEntry(ctx, dict, dictName, "TransferFunction", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	_, _, err = writeBooleanEntry(ctx, dict, dictName, "AccurateScreens", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeType1HalftoneDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeType5HalftoneDict(ctx *types.PDFContext, dict types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeType5HalftoneDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "type5HalftoneDict"

	_, _, err = writeStringEntry(ctx, dict, dictName, "HalftoneName", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	err = writeHalfToneEntry(ctx, dict, dictName, "Gray", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	err = writeHalfToneEntry(ctx, dict, dictName, "Red", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	err = writeHalfToneEntry(ctx, dict, dictName, "Green", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	err = writeHalfToneEntry(ctx, dict, dictName, "Blue", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	err = writeHalfToneEntry(ctx, dict, dictName, "Cyan", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	err = writeHalfToneEntry(ctx, dict, dictName, "Magenta", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	err = writeHalfToneEntry(ctx, dict, dictName, "Yellow", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	err = writeHalfToneEntry(ctx, dict, dictName, "Black", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	// TODO non standard color components missing.
	err = writeHalfToneEntry(ctx, dict, dictName, "Default", REQUIRED, sinceVersion)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeType5HalftoneDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeType6HalftoneStreamDict(ctx *types.PDFContext, dict types.PDFStreamDict, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeType6HalftoneStreamDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "type6HalftoneDict"

	_, _, err = writeStringEntry(ctx, dict.PDFDict, dictName, "HalftoneName", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, dict.PDFDict, dictName, "Width", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, dict.PDFDict, dictName, "Height", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	err = writeTransferFunctionEntry(ctx, dict.PDFDict, dictName, "TransferFunction", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeType6HalftoneStreamDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeType10HalftoneStreamDict(ctx *types.PDFContext, dict types.PDFStreamDict, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeType10HalftoneStreamDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "type10HalftoneDict"

	_, _, err = writeStringEntry(ctx, dict.PDFDict, dictName, "HalftoneName", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, dict.PDFDict, dictName, "Xsquare", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, dict.PDFDict, dictName, "Ysquare", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	err = writeTransferFunctionEntry(ctx, dict.PDFDict, dictName, "TransferFunction", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeType10HalftoneStreamDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeType16HalftoneStreamDict(ctx *types.PDFContext, dict types.PDFStreamDict, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeType16HalftoneStreamDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "type16HalftoneDict"

	_, _, err = writeStringEntry(ctx, dict.PDFDict, dictName, "HalftoneName", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, dict.PDFDict, dictName, "Width", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, dict.PDFDict, dictName, "Height", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, dict.PDFDict, dictName, "Width2", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	_, _, err = writeIntegerEntry(ctx, dict.PDFDict, dictName, "Height2", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	err = writeTransferFunctionEntry(ctx, dict.PDFDict, dictName, "TransferFunction", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeType16HalftoneStreamDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeHalfToneDict(ctx *types.PDFContext, dict types.PDFDict, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeHalfToneDict begin: offset=%d ***\n", ctx.Write.Offset)

	Type, _, err := writeNameEntry(ctx, dict, "halfToneDict", "Type", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	if Type != nil && *Type != "Halftone" {
		return errors.Errorf("writeHalfToneDict: unknown \"Type\": %s\n", *Type)
	}

	halftoneType, written, err := writeIntegerEntry(ctx, dict, "halfToneDict", "HalftoneType", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	if written {

	}

	switch *halftoneType {

	case 1:
		err = writeType1HalftoneDict(ctx, dict, sinceVersion)

	case 5:
		err = writeType5HalftoneDict(ctx, dict, sinceVersion)

	default:
		err = errors.Errorf("writeHalfToneDict: unknown halftoneTyp: %d\n", *halftoneType)

	}

	logInfoWriter.Printf("*** writeHalfToneDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeHalfToneStreamDict(ctx *types.PDFContext, dict types.PDFStreamDict, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeHalfToneStreamDict begin: offset=%d ***\n", ctx.Write.Offset)

	Type, _, err := writeNameEntry(ctx, dict.PDFDict, "writeHalfToneStreamDict", "Type", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	if Type != nil && *Type != "Halftone" {
		return errors.Errorf("writeHalfToneStreamDict: unknown \"Type\": %s\n", *Type)
	}

	halftoneType, written, err := writeIntegerEntry(ctx, dict.PDFDict, "writeHalfToneStreamDict", "HalftoneType", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	if written || halftoneType == nil {
		return
	}

	switch *halftoneType {

	case 6:
		err = writeType6HalftoneStreamDict(ctx, dict, sinceVersion)

	case 10:
		err = writeType10HalftoneStreamDict(ctx, dict, sinceVersion)

	case 16: // TODO since V1.3
		err = writeType16HalftoneStreamDict(ctx, dict, sinceVersion)

	default:
		err = errors.Errorf("writeHalfToneStreamDict: unknown halftoneTyp: %d\n", *halftoneType)

	}

	logInfoWriter.Printf("*** writeHalfToneStreamDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeHalfToneEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeHalfToneEntry begin: offset=%d ***\n", ctx.Write.Offset)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			return errors.Errorf("writeHalfToneEntry: dict=%s required entry=%s missing.", dictName, entryName)
		}
		logInfoWriter.Printf("writeHalfToneEntry end: entry %s is nil\n", entryName)
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeHalfToneEntry: dict=%s entry=%s unsupported in version %s.\n", dictName, entryName, ctx.VersionString())
	}

	switch obj := obj.(type) {

	case types.PDFName:
		if obj.String() != "Default" {
			err = errors.Errorf("writeHalfToneEntry: undefined name: %s\n", obj.String())
		}

	case types.PDFDict:
		err = writeHalfToneDict(ctx, obj, sinceVersion)

	case types.PDFStreamDict:
		err = writeHalfToneStreamDict(ctx, obj, sinceVersion)

	default:
		err = errors.New("writeHalfToneEntry: corrupt (stream)dict")
	}

	logInfoWriter.Printf("*** writeHalfToneEntry end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeExtGStateDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// 8.4.5 Graphics State Parameter Dictionaries

	logInfoWriter.Printf("*** writeExtGStateDict begin: offset=%d ***\n", ctx.Write.Offset)

	if dict.Type() != nil && *dict.Type() != "ExtGState" {
		return errors.New("writeExtGStateDict: corrupt dict type")
	}

	dictName := "extGStateDict"

	// LW, number, optional, since V1.3
	_, _, err = writeNumberEntry(ctx, dict, dictName, "LW", OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// LC, integer, optional, since V1.3
	_, _, err = writeIntegerEntry(ctx, dict, dictName, "LC", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// LJ, integer, optional, since V1.3
	_, _, err = writeIntegerEntry(ctx, dict, dictName, "LJ", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// ML, number, optional, since V1.3
	_, _, err = writeNumberEntry(ctx, dict, dictName, "ML", OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// D, array, optional, since V1.3, [dashArray dashPhase(integer)]
	err = writeLineDashPatternEntry(ctx, dict, dictName, "D", OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// RI, name, optional, since V1.3
	_, _, err = writeNameEntry(ctx, dict, dictName, "RI", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// OP, boolean, optional,
	_, _, err = writeBooleanEntry(ctx, dict, dictName, "OP", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// op, boolean, optional, since V1.3
	_, _, err = writeBooleanEntry(ctx, dict, dictName, "op", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// OPM, integer, optional, since V1.3
	_, _, err = writeIntegerEntry(ctx, dict, dictName, "OPM", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// Font, array, optional, since V1.3
	// TODO validate[font(indRef(fontDict)) size(number expressed in text space units)]
	_, _, err = writeArrayEntry(ctx, dict, dictName, "Font", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// BG, function, optional, black-generation function, see 10.3.4
	_, err = writeFunctionEntry(ctx, dict, dictName, "BG", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// BG2, function or name(/Default), optional, since V1.3
	err = writeBG2Entry(ctx, dict, dictName, "BG2", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// UCR, function, optional, undercolor-removal function, see 10.3.4
	_, err = writeFunctionEntry(ctx, dict, dictName, "UCR", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// UCR2, function or name(/Default), optional, since V1.3
	err = writeUCR2Entry(ctx, dict, dictName, "UCR2", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// TR, function, array of 4 functions or name(/Identity), optional, see 10.4 transfer functions
	err = writeTransferFunctionEntry(ctx, dict, dictName, "TR", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// TR2, function, array of 4 functions or name(/Identity,/Default), optional, since V1.3
	err = writeTR2Entry(ctx, dict, dictName, "TR2", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// HT, dict, stream or name, optional
	// half tone dictionary or stream or /Default, see 10.5
	err = writeHalfToneEntry(ctx, dict, dictName, "HT", OPTIONAL, types.V12)
	if err != nil {
		return
	}

	// FL, number, optional, since V1.3, flatness tolerance, see 10.6.2
	_, _, err = writeNumberEntry(ctx, dict, dictName, "FL", OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// SM, number, optional, since V1.3, smoothness tolerance
	_, _, err = writeNumberEntry(ctx, dict, dictName, "SM", OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// SA, boolean, optional, see 10.6.5 Automatic Stroke Adjustment
	_, _, err = writeBooleanEntry(ctx, dict, dictName, "SA", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// BM, name or array, optional, since V1.4
	// TODO Relaxed since V1.3
	err = writeBlendModeEntry(ctx, dict, dictName, "BM", OPTIONAL, types.V13)
	if err != nil {
		return err
	}

	// SMask, dict or name, optional, since V1.4
	// TODO relaxed, since V1.3
	err = writeSoftMaskEntry(ctx, dict, dictName, "SMask", OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// CA, number, optional, since V1.4, current stroking alpha constant, see 11.3.7.2 and 11.6.4.4
	// TODO relaxed, since V1.3
	_, _, err = writeNumberEntry(ctx, dict, dictName, "CA", OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// ca, number, optional, since V1.4, same as CA but for nonstroking operations.
	// TODO relaxed, since V1.3
	_, _, err = writeNumberEntry(ctx, dict, dictName, "ca", OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// AIS, boolean, optional, since V1.4
	// TODO relaxed, since V1.3, alpha source flag "alpha is shape"
	_, _, err = writeBooleanEntry(ctx, dict, dictName, "AIS", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// TK, boolean, optional, since V1.4, text knockout flag.
	_, _, err = writeBooleanEntry(ctx, dict, dictName, "TK", OPTIONAL, types.V14, nil)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeExtGStateDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeExtGStateResourceDict(ctx *types.PDFContext, obj interface{}) (err error) {

	logInfoWriter.Printf("*** writeExtGStateResourceDict begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("*** writeExtGStateResourceDict end: object already written. offset=%d ***\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("*** writeExtGStateResourceDict end: object is nil. offset=%d ***\n", ctx.Write.Offset)
		return
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		return errors.New("writeExtGStateResourceDict: corrupt dict")
	}

	// Iterate over extGState resource dictionary
	for _, obj := range dict.Dict {

		obj, written, err = writeObject(ctx, obj)
		if err != nil {
			return
		}

		if written {
			logInfoWriter.Printf("*** writeExtGStateResourceDict end: resource object already written. offset=%d ***\n", ctx.Write.Offset)
			continue
		}

		if obj == nil {
			logInfoWriter.Printf("*** writeExtGStateResourceDict end: resource object is nil. offset=%d ***\n", ctx.Write.Offset)
			continue
		}

		dict, ok := obj.(types.PDFDict)
		if !ok {
			return errors.New("writeExtGStateResourceDict end: corrupt extGState dict")
		}

		// Process extGStateDict
		err = writeExtGStateDict(ctx, dict)
		if err != nil {
			return
		}

	}

	logInfoWriter.Printf("*** writeExtGStateResourceDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}
