package write

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/hhrutter/pdflib/validate"
	"github.com/pkg/errors"
)

func writePageContents(ctx *types.PDFContext, dict types.PDFDict) (hasContents bool, err error) {

	logInfoWriter.Printf("*** writePageContents begin: offset=%d ***\n", ctx.Write.Offset)

	obj, found := dict.Find("Contents")
	if !found {
		return
	}

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		// contents are optional
		return
	}

	if written {
		err = errors.Errorf("writePageContents end: already written, offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writePageContents end: is nil, offset=%d\n", ctx.Write.Offset)
		return
	}

	switch obj := obj.(type) {

	case types.PDFStreamDict:
		// no further processing, page content already written.
		hasContents = true

	case types.PDFArray:
		// process array of content stream dicts.

		logInfoWriter.Printf("writePageContents: writing content array\n")

		for _, obj := range obj {

			obj, written, err = writeStreamDict(ctx, obj)
			if err != nil {
				return
			}

			if written {
				logInfoWriter.Printf("writePageContents end: already written, offset=%d\n", ctx.Write.Offset)
				continue
			}

			if obj == nil {
				logInfoWriter.Printf("writePageContents end: is nil, offset=%d\n", ctx.Write.Offset)
				continue
			}

			hasContents = true

		}

	default:
		err = errors.Errorf("writePageContents: page content must be stream dict or array")
		return
	}

	if hasContents {
		ctx.Stats.AddPageAttr(types.PageContents)
	}

	logInfoWriter.Printf("*** writePageContents end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePageResources(ctx *types.PDFContext, dict types.PDFDict, hasResources, hasContents bool) (err error) {

	logInfoWriter.Printf("*** writePageResources begin: offset=%d ***\n", ctx.Write.Offset)

	if obj, found := dict.Find("Resources"); found {

		written, err := writeResourceDict(ctx, obj)
		if err != nil {
			return err
		}

		// Leave here where called from writePageDict.
		if written {
			ctx.Stats.AddPageAttr(types.PageResources)
		}

		return nil
	}

	if !hasResources && hasContents {
		err = errors.New("writePageResources: missing required entry \"Resources\" - should be inheritated")
	}

	logInfoWriter.Printf("*** writePageResources end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePageEntryMediaBox(ctx *types.PDFContext, dict types.PDFDict, required bool, sinceVersion types.PDFVersion) (hasMediaBox bool, err error) {

	logInfoWriter.Printf("*** writeMediaBox begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeRectangleEntry(ctx, dict, "pagesDict", "MediaBox", required, sinceVersion, nil)
	if err != nil {
		return
	}
	if !written && obj != nil {
		ctx.Stats.AddPageAttr(types.PageMediaBox)
		hasMediaBox = true
	}

	logInfoWriter.Printf("*** writeMediaBox end: hasMediaBox=%v offset=%d ***\n", hasMediaBox, ctx.Write.Offset)

	return
}

func writePageEntryCropBox(ctx *types.PDFContext, dict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writePageEntryCropBox begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeRectangleEntry(ctx, dict, "pagesDict", "CropBox", required, sinceVersion, nil)
	if err != nil {
		return
	}
	if !written && obj != nil {
		ctx.Stats.AddPageAttr(types.PageCropBox)
	}

	logInfoWriter.Printf("*** writePageEntryCropBox end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePageEntryBleedBox(ctx *types.PDFContext, dict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writePageEntryBleedBox begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeRectangleEntry(ctx, dict, "pagesDict", "BleedBox", required, sinceVersion, nil)
	if err != nil {
		return
	}
	if !written && obj != nil {
		ctx.Stats.AddPageAttr(types.PageBleedBox)
	}

	logInfoWriter.Printf("*** writePageEntryBleedBox end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePageEntryTrimBox(ctx *types.PDFContext, dict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writePageEntryTrimBox begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeRectangleEntry(ctx, dict, "pagesDict", "TrimBox", required, sinceVersion, nil)
	if err != nil {
		return
	}
	if !written && obj != nil {
		ctx.Stats.AddPageAttr(types.PageTrimBox)
	}

	logInfoWriter.Printf("*** writePageEntryTrimBox end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePageEntryArtBox(ctx *types.PDFContext, dict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writePageEntryArtBox begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeRectangleEntry(ctx, dict, "pagesDict", "ArtBox", required, sinceVersion, nil)
	if err != nil {
		return
	}
	if !written && obj != nil {
		ctx.Stats.AddPageAttr(types.PageArtBox)
	}

	logInfoWriter.Printf("*** writePageEntryArtBox end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeBoxStyleDictEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (d *types.PDFDict, written bool, err error) {

	logInfoWriter.Printf("*** writeBoxStyleDictEntry begin: offset=%d ***\n", ctx.Write.Offset)

	d, written, err = writeDictEntry(ctx, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeBoxStyleDictEntry end: already written. offset=%d \n", ctx.Write.Offset)
		return
	}

	if d == nil {
		logInfoWriter.Printf("writeBoxStyleDictEntry end: is nil.\n")
		return
	}

	dictName = "boxStyleDict"

	// C, number array with 3 elements, optional
	// TODO entries between 0.0 and 1.0
	_, _, err = writeNumberArrayEntry(ctx, *d, dictName, "C", OPTIONAL, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 3 })
	if err != nil {
		return
	}

	// W, number, optional
	_, _, err = writeNumberEntry(ctx, *d, dictName, "W", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	// S, name, optional
	_, _, err = writeNameEntry(ctx, *d, dictName, "S", OPTIONAL, sinceVersion, validate.GuideLineStyle)
	if err != nil {
		return
	}

	// D, array, optional, since V1.3, [dashArray dashPhase(integer)]
	err = writeLineDashPatternEntry(ctx, *d, dictName, "D", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeBoxStyleDictEntry end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePageBoxColorInfo(ctx *types.PDFContext, pagesDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// box color information dict
	// see 14.11.2.2

	logInfoWriter.Printf("*** writeBoxColorInfo begin: offset=%d ***\n", ctx.Write.Offset)

	dict, written, err := writeDictEntry(ctx, pagesDict, "", "BoxColorInfo", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written || dict == nil {
		return
	}

	var hasBoxColorInfo bool

	d, written, err := writeBoxStyleDictEntry(ctx, *dict, "boxColorInfoDict", "CropBox", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}
	if !written && d != nil {
		hasBoxColorInfo = true
	}

	d, written, err = writeBoxStyleDictEntry(ctx, *dict, "boxColorInfoDict", "BleedBox", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}
	if !written && d != nil {
		hasBoxColorInfo = true
	}

	d, written, err = writeBoxStyleDictEntry(ctx, *dict, "boxColorInfoDict", "TrimBox", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}
	if !written && d != nil {
		hasBoxColorInfo = true
	}

	d, written, err = writeBoxStyleDictEntry(ctx, *dict, "boxColorInfoDict", "ArtBox", OPTIONAL, sinceVersion)
	if err != nil {
		return
	}
	if !written && d != nil {
		hasBoxColorInfo = true
	}

	if hasBoxColorInfo {
		ctx.Stats.AddPageAttr(types.PageBoxColorInfo)
	}

	logInfoWriter.Printf("*** writeBoxColorInfo end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePageEntryRotate(ctx *types.PDFContext, dict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writePageEntryRotate begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeIntegerEntry(ctx, dict, "pagesDict", "Rotate", required, sinceVersion, validate.Rotate)
	if err != nil {
		return
	}

	if !written && obj != nil {
		ctx.Stats.AddPageAttr(types.PageRotate)
	}

	logInfoWriter.Printf("*** writePageEntryRotate end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePageEntryGroup(ctx *types.PDFContext, dict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// TODO validate Version

	logInfoWriter.Printf("*** writePageEntryGroup begin: offset=%d ***\n", ctx.Write.Offset)

	if obj, ok := dict.Find("Group"); ok {
		written, err := writeGroupAttributesDict(ctx, obj)
		if err != nil {
			return err
		}
		if written {
			ctx.Stats.AddPageAttr(types.PageGroup)
		}
	}

	logInfoWriter.Printf("*** writePageEntryGroup end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePageEntryThumb(ctx *types.PDFContext, dict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writePageEntryThumb begin: offset=%d ***\n", ctx.Write.Offset)

	streamDict, written, err := writeStreamDictEntry(ctx, dict, "pagesDict", "Thumb", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writePageEntryThumb end: already written. offset=%d \n", ctx.Write.Offset)
		return
	}

	if streamDict == nil {
		logInfoWriter.Printf("writePageEntryThumb end: is nil.\n")
		return
	}

	err = writeXObjectStreamDict(ctx, *streamDict)
	if err != nil {
		return
	}

	ctx.Stats.AddPageAttr(types.PageThumb)

	logInfoWriter.Printf("*** writePageEntryThumb end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePageEntryB(ctx *types.PDFContext, dict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// only makes sense if Threads entry in document root exists.

	logInfoWriter.Printf("*** writePageEntryB begin: offset=%d ***\n", ctx.Write.Offset)

	arr, written, err := writeIndRefArrayEntry(ctx, dict, "pagesDict", "B", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writePageEntryB end: already written. offset=%d \n", ctx.Write.Offset)
		return
	}

	if arr == nil {
		logInfoWriter.Printf("writePageEntryB end: is nil.\n")
		return
	}

	// TODO verify indRefs against article beads.

	ctx.Stats.AddPageAttr(types.PageB)

	logInfoWriter.Printf("*** writePageEntryB end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePageEntryDur(ctx *types.PDFContext, dict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writePageEntryDur begin: offset=%d ***\n", ctx.Write.Offset)

	dur, written, err := writeNumberEntry(ctx, dict, "pagesDict", "Dur", required, sinceVersion)
	if err != nil {
		return
	}

	if !written && dur != nil {
		ctx.Stats.AddPageAttr(types.PageDur)
	}

	logInfoWriter.Printf("*** writePageEntryDur end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePageEntryTrans(ctx *types.PDFContext, pagesDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writePageEntryTrans begin: offset=%d ***\n", ctx.Write.Offset)

	dict, written, err := writeDictEntry(ctx, pagesDict, "pagesDict", "Trans", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writePageEntryTrans end: already written. offset=%d\n", ctx.Write.Offset)
		return
	}

	if dict == nil {
		logInfoWriter.Printf("writePageEntryTrans end: is nil.\n")
		return
	}

	dictName := "transitionDict"

	transStyle, _, err := writeNameEntry(ctx, *dict, dictName, "S", OPTIONAL, types.V10, validate.TransitionStyle)
	if err != nil {
		return
	}

	// TODO: validate >= 0
	_, _, err = writeNumberEntry(ctx, *dict, dictName, "D", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	_, written, err = writeNameEntry(ctx, *dict, dictName, "Dm", OPTIONAL, types.V10, validate.TransitionDimension)
	if err != nil {
		return
	}
	if written && (transStyle == nil || !(*transStyle == "Split" || *transStyle == "Blinds")) {
		return errors.New("writePageEntryTrans: entry \"Dm\" requires entry \"S\" of value \"Split\" or \"Blind\"")
	}

	_, written, err = writeNameEntry(ctx, *dict, dictName, "M", OPTIONAL, types.V10, validate.TransitionDirectionOfMotion)
	if err != nil {
		return
	}
	if written && (transStyle == nil || !(*transStyle == "Split" || *transStyle == "Box" || *transStyle == "Fly")) {
		return errors.New("writePageEntryTrans: entry \"M\" requires entry \"S\" of value \"Split\", \"Box\" or \"Fly\"")
	}

	// TODO: "Di" number or name

	// TODO: validate >= 0
	_, written, err = writeNumberEntry(ctx, *dict, dictName, "SS", OPTIONAL, types.V15)
	if err != nil {
		return
	}
	if written && (transStyle == nil || *transStyle != "Fly") {
		return errors.New("writePageEntryTrans: entry \"SS\" requires entry \"S\" of value \"Fly\"")
	}

	_, written, err = writeBooleanEntry(ctx, *dict, dictName, "B", OPTIONAL, types.V15, nil)
	if err != nil {
		return
	}
	if written && (transStyle == nil || *transStyle != "Fly") {
		return errors.New("writePageEntryTrans: entry \"B\" requires entry \"S\" of value \"Fly\"")
	}

	ctx.Stats.AddPageAttr(types.PageTrans)

	logInfoWriter.Printf("*** writePageEntryTrans begin: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePageEntryStructParents(ctx *types.PDFContext, dict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writePageEntryStructParents begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeIntegerEntry(ctx, dict, "pagesDict", "StructParents", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if !written && obj != nil {
		ctx.Stats.AddPageAttr(types.PageStructParents)
	}

	logInfoWriter.Printf("*** writePageEntryStructParents end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePageEntryID(ctx *types.PDFContext, dict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writePageEntryID begin: offset=%d ***\n", ctx.Write.Offset)

	id, written, err := writeStringEntry(ctx, dict, "pagesDict", "ID", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if !written && id != nil {
		ctx.Stats.AddPageAttr(types.PageID)
	}

	logInfoWriter.Printf("*** writePageEntryID end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePageEntryPZ(ctx *types.PDFContext, dict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// Preferred zoom factor, number

	logInfoWriter.Printf("*** writePageEntryPZ begin: offset=%d ***\n", ctx.Write.Offset)

	pz, written, err := writeNumberEntry(ctx, dict, "pagesDict", "PZ", required, sinceVersion)
	if err != nil {
		return
	}

	if !written && pz != nil {
		ctx.Stats.AddPageAttr(types.PagePZ)
	}

	logInfoWriter.Printf("*** writePageEntryPZ end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeDeviceColorantEntry(ctx *types.PDFContext, dict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeDeviceColorantEntry begin: offset=%d ***\n", ctx.Write.Offset)

	obj, found := dict.Find("DeviceColorant")
	if !found || obj == nil {
		return
	}

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeDeviceColorantEntry end: already written. offset=%d \n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeDeviceColorantEntry end: is nil.\n")
		return
	}

	// TODO validate colorant

	switch obj.(type) {

	case types.PDFName:
		// no further processing

	case types.PDFStringLiteral:
		// no further processing

	default:
		err = errors.Errorf("writeDeviceColorantEntry: must be name or string.")
		return
	}

	logInfoWriter.Printf("*** writeDeviceColorantEntry end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePageEntrySeparationInfo(ctx *types.PDFContext, pagesDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writePageEntrySeparationInfo begin: offset=%d ***\n", ctx.Write.Offset)

	dict, written, err := writeDictEntry(ctx, pagesDict, "pagesDict", "SeparationInfo", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writePageEntrySeparationInfo end: already written. offset=%d \n", ctx.Write.Offset)
		return
	}

	if dict == nil {
		logInfoWriter.Printf("writePageEntrySeparationInfo end: is nil.\n")
		return
	}

	_, _, err = writeIndRefArrayEntry(ctx, *dict, "separationDict", "Pages", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	err = writeDeviceColorantEntry(ctx, *dict, OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	arr, written, err := writeArrayEntry(ctx, *dict, "separationDict", "ColorSpace", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}
	if !written && arr != nil {
		// TODO Separation or DeviceN color space only
		err = writeColorSpaceArray(ctx, *arr, ExcludePatternCS)
		if err != nil {
			return
		}
	}

	ctx.Stats.AddPageAttr(types.PageSeparationInfo)

	logInfoWriter.Printf("*** writePageEntrySeparationInfo end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePageEntryTabs(ctx *types.PDFContext, dict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writePageEntryTabs begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeNameEntry(ctx, dict, "pagesDict", "Tabs", required, sinceVersion, validate.Tabs)
	if err != nil {
		return
	}

	if !written && obj != nil {
		ctx.Stats.AddPageAttr(types.PageTabs)
	}

	logInfoWriter.Printf("*** writePageEntryTabs end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePageEntryTemplateInstantiated(ctx *types.PDFContext, dict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writePageEntryTemplateInstantiated begin: offset=%d ***\n", ctx.Write.Offset)

	name, written, err := writeNameEntry(ctx, dict, "pagesDict", "TemplateInstantiated", required, sinceVersion, nil)
	if err != nil {
		return
	}

	// TODO Validate against root name dictionary

	if !written && name != nil {
		ctx.Stats.AddPageAttr(types.PageTemplateInstantiated)
	}

	logInfoWriter.Printf("*** writePageEntryTemplateInstantiated end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writePageEntryPresSteps(ctx *types.PDFContext, pagesDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writePageEntryPresSteps begin: offset=%d ***\n", ctx.Write.Offset)

	dict, written, err := writeDictEntry(ctx, pagesDict, "pagesDict", "PresSteps", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writePageEntryPresSteps end: already written. offset=%d \n", ctx.Write.Offset)
		return
	}

	if dict == nil {
		logInfoWriter.Printf("writePageEntryPresSteps end: is nil.\n")
		return
	}

	err = errors.New("*** writePageEntryPresSteps: not supported ***")

	// source.XRefTable.Stats.AddPageAttr(types.PagePresSteps)

	logInfoWriter.Printf("*** writePageEntryPresSteps end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePageEntryUserUnit(ctx *types.PDFContext, dict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writePageEntryUserUnit begin: offset=%d ***\n", ctx.Write.Offset)

	// TODO validate positive number
	obj, written, err := writeNumberEntry(ctx, dict, "pagesDict", "UserUnit", required, sinceVersion)
	if err != nil {
		return
	}

	if !written && obj != nil {
		ctx.Stats.AddPageAttr(types.PageUserUnit)
	}

	logInfoWriter.Printf("*** writePageEntryUserUnit end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writePageEntryVP(ctx *types.PDFContext, pagesDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writePageEntryVP begin: offset=%d ***\n", ctx.Write.Offset)

	dict, written, err := writeDictEntry(ctx, pagesDict, "pagesDict", "VP", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writePageEntryVP end: already written. offset=%d \n", ctx.Write.Offset)
		return
	}

	if dict == nil {
		logInfoWriter.Printf("writePageEntryVP end: is nil.\n")
		return
	}

	err = errors.New("*** writePageEntryVP: not supported ***")

	// source.XRefTable.Stats.AddPageAttr(types.PageVP)

	logInfoWriter.Printf("*** writePageEntryVP end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePageDict(ctx *types.PDFContext, objNumber, genNumber int, pagesDict types.PDFDict, hasResources, hasMediaBox bool) (err error) {

	logPages.Printf("*** writePageDict begin: hasMediaBox=%v obj#%d offset=%d ***\n", hasMediaBox, objNumber, ctx.Write.Offset)

	logInfoWriter.Printf("writePageDict: object #%d gets writeoffset: %d\n", objNumber, ctx.Write.Offset)

	// For extracted pages we do not generate Annotations.
	if ctx.Write.ReducedFeatureSet() {
		pagesDict.Delete("Annots")
	}

	err = writePDFDictObject(ctx, objNumber, genNumber, pagesDict)
	if err != nil {
		return
	}

	logDebugWriter.Printf("writePageDict: new offset = %d\n", ctx.Write.Offset)

	if indref := pagesDict.IndirectRefEntry("Parent"); indref == nil {
		return errors.New("writePageDict: missing parent")
	}

	// Contents
	hasContents, err := writePageContents(ctx, pagesDict)
	if err != nil {
		return err
	}

	// Resources
	err = writePageResources(ctx, pagesDict, hasResources, hasContents)
	if err != nil {
		return err
	}

	// MediaBox
	_, err = writePageEntryMediaBox(ctx, pagesDict, !hasMediaBox, types.V10)
	if err != nil {
		return
	}

	// CropBox
	err = writePageEntryCropBox(ctx, pagesDict, OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// BleedBox
	err = writePageEntryBleedBox(ctx, pagesDict, OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// TrimBox
	err = writePageEntryTrimBox(ctx, pagesDict, OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// ArtBox
	err = writePageEntryArtBox(ctx, pagesDict, OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// BoxColorInfo
	err = writePageBoxColorInfo(ctx, pagesDict, OPTIONAL, types.V14)
	if err != nil {
		return
	}

	// PieceInfo
	// Relaxed: sinceVersion = V10 must since V13
	hasPieceInfo, err := writePieceInfo(ctx, pagesDict, OPTIONAL, types.V10)
	if err != nil {
		return err
	}
	if hasPieceInfo {
		ctx.Stats.AddPageAttr(types.PagePieceInfo)
	}

	// LastModified
	// lm, ...
	_, _, err = writeDateEntry(ctx, pagesDict, "pageDict", "LastModified", OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// Relaxed: Disable
	//if hasPieceInfo && lm == nil {
	//	return errors.New("writePageDict: missing \"LastModified\" (required by \"PieceInfo\")")
	//}

	// Rotate
	err = writePageEntryRotate(ctx, pagesDict, OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// Group
	err = writePageEntryGroup(ctx, pagesDict, OPTIONAL, types.V14)
	if err != nil {
		return
	}

	// Annotations
	// delay until processing of AcroForms.
	// see ...

	// Thumb
	err = writePageEntryThumb(ctx, pagesDict, OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// B
	err = writePageEntryB(ctx, pagesDict, OPTIONAL, types.V11)
	if err != nil {
		return
	}

	// Dur
	err = writePageEntryDur(ctx, pagesDict, OPTIONAL, types.V11)
	if err != nil {
		return
	}

	// Trans
	err = writePageEntryTrans(ctx, pagesDict, OPTIONAL, types.V11)
	if err != nil {
		return
	}

	// AA
	written, err := writeAdditionalActions(ctx, &pagesDict, "pageDict", "AA", OPTIONAL, types.V14, "page")
	if err != nil {
		return
	}
	if written {
		ctx.Stats.AddPageAttr(types.PageAA)
	}

	// Metadata
	written, err = writeMetadata(ctx, pagesDict, OPTIONAL, types.V14)
	if err != nil {
		return
	}
	if written {
		ctx.Stats.AddPageAttr(types.PageMetadata)
	}

	// StructParents
	err = writePageEntryStructParents(ctx, pagesDict, OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// ID
	err = writePageEntryID(ctx, pagesDict, OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// PZ
	err = writePageEntryPZ(ctx, pagesDict, OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// SeparationInfo
	err = writePageEntrySeparationInfo(ctx, pagesDict, OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// Tabs
	err = writePageEntryTabs(ctx, pagesDict, OPTIONAL, types.V15)
	if err != nil {
		return
	}

	// TemplateInstantiateds
	err = writePageEntryTemplateInstantiated(ctx, pagesDict, OPTIONAL, types.V15)
	if err != nil {
		return
	}

	// PresSteps
	err = writePageEntryPresSteps(ctx, pagesDict, OPTIONAL, types.V15)
	if err != nil {
		return
	}

	// UserUnit
	err = writePageEntryUserUnit(ctx, pagesDict, OPTIONAL, types.V16)
	if err != nil {
		return
	}

	// VP
	err = writePageEntryVP(ctx, pagesDict, OPTIONAL, types.V16)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writePageDict end: obj#%d offset=%d ***\n", objNumber, ctx.Write.Offset)

	return
}

// func writePagesDictOrig(ctx *types.PDFContext, indRef types.PDFIndirectRef, hasResources, hasMediaBox bool) (err error) {

// 	logPages.Printf("*** writePagesDict begin: hasMediaBox=%v obj#%d offset=%d ***\n", hasMediaBox, indRef.ObjectNumber, ctx.Write.Offset)

// 	obj, written, err := writeIndRef(ctx, indRef)
// 	if err != nil {
// 		return
// 	}

// 	if written || obj == nil {
// 		return errors.Errorf("writePagesDict end: obj#%d offset=%d *** nil or already written", indRef.ObjectNumber, ctx.Write.Offset)
// 	}

// 	dict, ok := obj.(types.PDFDict)
// 	if !ok {
// 		return errors.New("writePagesDict: corrupt pages dict")
// 	}

// 	// Get number of pages of this PDF file.
// 	count, ok := dict.Find("Count")
// 	if !ok {
// 		return errors.New("writePagesDict: missing \"Count\"")
// 	}

// 	pageCount := int(count.(types.PDFInteger))
// 	logPages.Printf("writePagesDict: This page node has %d pages\n", pageCount)

// 	// Resources: optional, dict
// 	if obj, ok := dict.Find("Resources"); ok {
// 		written, err = writeResourceDict(ctx, obj)
// 		if err != nil {
// 			return
// 		}
// 		if written {
// 			ctx.Stats.AddPageAttr(types.PageResources)
// 			hasResources = true
// 		}
// 	}

// 	// MediaBox: optional, rectangle
// 	hasPageNodeMediaBox, err := writePageEntryMediaBox(ctx, dict, OPTIONAL, types.V10)
// 	if err != nil {
// 		return
// 	}

// 	if hasPageNodeMediaBox {
// 		hasMediaBox = true
// 	}

// 	// CropBox: optional, rectangle
// 	err = writePageEntryCropBox(ctx, dict, OPTIONAL, types.V10)
// 	if err != nil {
// 		return
// 	}

// 	// Rotate:  optional, integer
// 	err = writePageEntryRotate(ctx, dict, OPTIONAL, types.V10)
// 	if err != nil {
// 		return
// 	}

// 	// Iterate over page tree.
// 	kidsArray := dict.PDFArrayEntry("Kids")
// 	if kidsArray == nil {
// 		return errors.New("writePagesDict: corrupt \"Kids\" entry")
// 	}

// 	for _, obj := range *kidsArray {

// 		if obj == nil {
// 			logDebugWriter.Println("writePagesDict: kid is nil")
// 			continue
// 		}

// 		// Dereference next page node dict.
// 		indRef, ok := obj.(types.PDFIndirectRef)
// 		if !ok {
// 			return errors.New("writePagesDict: missing indirect reference for kid")
// 		}

// 		logInfoWriter.Printf("writePagesDict: PageNode: %s\n", indRef)

// 		objNumber := int(indRef.ObjectNumber)
// 		genNumber := int(indRef.GenerationNumber)

// 		if ctx.Write.HasWriteOffset(objNumber) {
// 			logInfoWriter.Printf("writePagesDict: object #%d already written.\n", objNumber)
// 			continue
// 		}

// 		pageNodeDict, err := ctx.DereferenceDict(indRef)
// 		if err != nil {
// 			return errors.New("writePagesDict: cannot dereference pageNodeDict")
// 		}

// 		if pageNodeDict == nil {
// 			return errors.New("validatePagesDict: pageNodeDict is null")
// 		}

// 		dictType := pageNodeDict.Type()
// 		if dictType == nil {
// 			return errors.New("writePagesDict: missing pageNodeDict type")
// 		}

// 		switch *dictType {

// 		case "Pages":
// 			// Recurse over pagetree
// 			err = writePagesDictOrig(ctx, indRef, hasResources, hasMediaBox)
// 			if err != nil {
// 				return err
// 			}

// 		case "Page":
// 			err = writePageDict(ctx, objNumber, genNumber, *pageNodeDict, hasResources, hasMediaBox)
// 			if err != nil {
// 				return err
// 			}

// 		default:
// 			return errors.Errorf("writePagesDict: Unexpected dict type: %s", *dictType)

// 		}

// 	}

// 	logPages.Printf("*** writePagesDict end: obj#%d offset=%d ***\n", indRef.ObjectNumber, ctx.Write.Offset)

// 	return
// }

func locateKidForPageNumber(ctx *types.PDFContext, kidsArray *types.PDFArray, pageCount *int, pageNumber int) (kid interface{}, err error) {

	for _, obj := range *kidsArray {

		if obj == nil {
			logDebugWriter.Println("locateKidForPageNumber: kid is nil")
			continue
		}

		// Dereference next page node dict.
		indRef, ok := obj.(types.PDFIndirectRef)
		if !ok {
			return nil, errors.New("locateKidForPageNumber: missing indirect reference for kid")
		}

		logInfoWriter.Printf("locateKidForPageNumber: PageNode: %s pageCount:%d extractPageNr:%d\n", indRef, *pageCount, pageNumber)

		dict, err := ctx.DereferenceDict(indRef)
		if err != nil {
			return nil, errors.New("locateKidForPageNumber: cannot dereference pageNodeDict")
		}

		if dict == nil {
			return nil, errors.New("locateKidForPageNumber: pageNodeDict is null")
		}

		dictType := dict.Type()
		if dictType == nil {
			return nil, errors.New("locateKidForPageNumber: missing pageNodeDict type")
		}

		switch *dictType {

		case "Pages":
			// Get number of pages of this PDF file.
			count, ok := dict.Find("Count")
			if !ok {
				return nil, errors.New("locateKidForPageNumber: missing \"Count\"")
			}
			pCount := int(count.(types.PDFInteger))

			if *pageCount+pCount < ctx.Write.ExtractPageNr {
				*pageCount += pCount
				logInfoWriter.Printf("locateKidForPageNumber: pageTree is no match")
			} else {
				logInfoWriter.Printf("locateKidForPageNumber: pageTree is a match")
				return obj, nil
			}

		case "Page":
			*pageCount++
			if *pageCount == ctx.Write.ExtractPageNr {
				logInfoWriter.Printf("locateKidForPageNumber: page is a match")
				return obj, nil
			}

			logInfoWriter.Printf("locateKidForPageNumber: page is no match")

		default:
			return nil, errors.Errorf("locateKidForPageNumber: Unexpected dict type: %s", *dictType)
		}

	}

	return nil, errors.Errorf("locateKidForPageNumber: Unable to locate kid: pageCount:%d extractPageNr:%d\n", *pageCount, pageNumber)
}

func writePagesDict(ctx *types.PDFContext, indRef types.PDFIndirectRef, pageCount int, hasResources, hasMediaBox bool) (err error) {

	logPages.Printf("*** writePagesDict begin: hasMediaBox=%v obj#%d offset=%d ***\n", hasMediaBox, indRef.ObjectNumber, ctx.Write.Offset)

	xRefTable := ctx.XRefTable
	objNumber := int(indRef.ObjectNumber)
	genNumber := int(indRef.GenerationNumber)

	if ctx.Write.HasWriteOffset(objNumber) {
		return errors.Errorf("writePagesDict end: obj#%d offset=%d *** nil or already written", indRef.ObjectNumber, ctx.Write.Offset)
	}

	obj, err := xRefTable.Dereference(indRef)
	if err != nil {
		err = errors.Wrapf(err, "writePagesDict: unable to dereference indirect object #%d", objNumber)
		return
	}

	if obj == nil {
		return errors.Errorf("writePagesDict end: obj#%d offset=%d *** nil or already written", indRef.ObjectNumber, ctx.Write.Offset)
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		return errors.Errorf("writePagesDict: corrupt pages dict, obj#%d", objNumber)
	}

	// Get number of pages of this PDF file.
	count, ok := dict.Find("Count")
	if !ok {
		return errors.New("writePagesDict: missing \"Count\"")
	}

	c := int(count.(types.PDFInteger))
	logPages.Printf("writePagesDict: This page node has %d pages\n", c)

	if c == 0 {
		logPages.Printf("writePagesDict: Ignore empty pages dict.\n")
		return
	}

	kidsArrayOrig := dict.PDFArrayEntry("Kids")
	if kidsArrayOrig == nil {
		return errors.New("writePagesDict: corrupt \"Kids\" entry")
	}

	// This is for Split and Extract when all we generate is a single page.
	if ctx.Write.ExtractPageNr > 0 {

		// Identify the kid containing the leaf for the page we are looking for aka the ExtractPageNr.
		// pageCount is either already the number of the page we are looking for and we have identified the kid for its page dict
		// or the number of pages before processing the next page tree containing the page we are looking for.
		// We need to write all original pagetree nodes leading to a specific leave in order not to miss any inheritated resources.
		kid, err := locateKidForPageNumber(ctx, kidsArrayOrig, &pageCount, ctx.Write.ExtractPageNr)
		if err != nil {
			return err
		}

		if pageCount == ctx.Write.ExtractPageNr {
			// The identified kid is the page node for the page we are looking for.
			logInfoWriter.Printf("writePagesDict: found page to be extracted, pageCount=%d, extractPageNr=%d\n", pageCount, ctx.Write.ExtractPageNr)
		} else {
			// The identified kid is the page tree containing the page we are looking for.
			logInfoWriter.Printf("writePagesDict: pageCount=%d, extractPageNr=%d\n", pageCount, ctx.Write.ExtractPageNr)
		}

		// Modify KidsArray to hold a single entry for this kid
		dict.Update("Kids", types.PDFArray{kid})

		// Set Count =1
		dict.Update("Count", types.PDFInteger(1))

	}

	err = writePDFDictObject(ctx, objNumber, genNumber, dict)
	if err != nil {
		return
	}

	logInfoWriter.Printf("writePagesDict: %s\n", dict)

	// Resources: optional, dict
	if obj, ok := dict.Find("Resources"); ok {
		var written bool
		written, err = writeResourceDict(ctx, obj)
		if err != nil {
			return
		}
		if written {
			ctx.Stats.AddPageAttr(types.PageResources)
			hasResources = true
		}
	}

	// MediaBox: optional, rectangle
	hasPageNodeMediaBox, err := writePageEntryMediaBox(ctx, dict, OPTIONAL, types.V10)
	if err != nil {
		return
	}

	if hasPageNodeMediaBox {
		hasMediaBox = true
	}

	// CropBox: optional, rectangle
	err = writePageEntryCropBox(ctx, dict, OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// Rotate:  optional, integer
	err = writePageEntryRotate(ctx, dict, OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// Iterate over page tree.
	kidsArray := dict.PDFArrayEntry("Kids")
	if kidsArray == nil {
		return errors.New("writePagesDict: corrupt \"Kids\" entry")
	}

	for _, obj := range *kidsArray {

		if obj == nil {
			logDebugWriter.Println("writePagesDict: kid is nil")
			continue
		}

		// Dereference next page node dict.
		indRef, ok := obj.(types.PDFIndirectRef)
		if !ok {
			return errors.New("writePagesDict: missing indirect reference for kid")
		}

		logInfoWriter.Printf("writePagesDict: PageNode: %s\n", indRef)

		objNumber := int(indRef.ObjectNumber)
		genNumber := int(indRef.GenerationNumber)

		if ctx.Write.HasWriteOffset(objNumber) {
			logInfoWriter.Printf("writePagesDict: object #%d already written.\n", objNumber)
			continue
		}

		pageNodeDict, err := ctx.DereferenceDict(indRef)
		if err != nil {
			return errors.New("writePagesDict: cannot dereference pageNodeDict")
		}

		if pageNodeDict == nil {
			return errors.New("validatePagesDict: pageNodeDict is null")
		}

		dictType := pageNodeDict.Type()
		if dictType == nil {
			return errors.New("writePagesDict: missing pageNodeDict type")
		}

		switch *dictType {

		case "Pages":
			// Recurse over pagetree
			err = writePagesDict(ctx, indRef, pageCount, hasResources, hasMediaBox)
			if err != nil {
				return err
			}

		case "Page":
			err = writePageDict(ctx, objNumber, genNumber, *pageNodeDict, hasResources, hasMediaBox)
			if err != nil {
				return err
			}

		default:
			return errors.Errorf("writePagesDict: Unexpected dict type: %s", *dictType)

		}

	}

	dict.Update("Kids", *kidsArrayOrig)
	dict.Update("Count", count)

	logPages.Printf("*** writePagesDict end: obj#%d offset=%d ***\n", indRef.ObjectNumber, ctx.Write.Offset)

	return
}

func trimPagesDict(ctx *types.PDFContext, indRef types.PDFIndirectRef, pageCount *int) (count int, err error) {

	xRefTable := ctx.XRefTable
	objNumber := int(indRef.ObjectNumber)

	obj, err := xRefTable.Dereference(indRef)
	if err != nil {
		err = errors.Wrapf(err, "trimPagesDict: unable to dereference indirect object #%d", objNumber)
		return
	}

	if obj == nil {
		return 0, errors.Errorf("trimPagesDict end: obj#%d offset=%d *** nil or already written", indRef.ObjectNumber, ctx.Write.Offset)
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		return 0, errors.Errorf("trimPagesDict: corrupt pages dict, obj#%d", objNumber)
	}

	// Get number of pages of this PDF file.
	c, ok := dict.Find("Count")
	if !ok {
		return 0, errors.New("trimPagesDict: missing \"Count\"")
	}

	logPages.Printf("trimPagesDict: This page node has %d pages\n", int(c.(types.PDFInteger)))

	// Iterate over page tree.
	kidsArray := dict.PDFArrayEntry("Kids")
	if kidsArray == nil {
		return 0, errors.New("trimPagesDict: corrupt \"Kids\" entry")
	}

	arr := types.PDFArray{}

	for _, obj := range *kidsArray {

		if obj == nil {
			logDebugWriter.Println("trimPagesDict: kid is nil")
			continue
		}

		// Dereference next page node dict.
		indRef, ok := obj.(types.PDFIndirectRef)
		if !ok {
			return 0, errors.New("trimPagesDict: missing indirect reference for kid")
		}

		logInfoWriter.Printf("trimPagesDict: PageNode: %s\n", indRef)

		objNumber := int(indRef.ObjectNumber)

		if ctx.Write.HasWriteOffset(objNumber) {
			logInfoWriter.Printf("trimPagesDict: object #%d already written.\n", objNumber)
			continue
		}

		pageNodeDict, err := ctx.DereferenceDict(indRef)
		if err != nil {
			return 0, errors.New("trimPagesDict: cannot dereference pageNodeDict")
		}

		if pageNodeDict == nil {
			return 0, errors.New("trimPagesDict: pageNodeDict is null")
		}

		dictType := pageNodeDict.Type()
		if dictType == nil {
			return 0, errors.New("writePagesDict: missing pageNodeDict type")
		}

		switch *dictType {

		case "Pages":
			// Recurse over pagetree
			trimmedCount, err := trimPagesDict(ctx, indRef, pageCount)
			if err != nil {
				return 0, err
			}

			if trimmedCount > 0 {
				count += trimmedCount
				arr = append(arr, obj)
			}

		case "Page":
			*pageCount++
			if ctx.Write.ExtractPage(*pageCount) {
				count++
				arr = append(arr, obj)
			}

		default:
			return 0, errors.Errorf("trimPagesDict: Unexpected dict type: %s", *dictType)

		}

	}

	logPages.Printf("trimPagesDict end: This page node is trimmed to %d pages\n", count)
	dict.Update("Count", types.PDFInteger(count))

	logPages.Printf("trimPagesDict end: updated kids: %s\n", arr)
	dict.Update("Kids", arr)

	return
}
