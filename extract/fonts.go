package extract

import (
	"io/ioutil"
	"os"
	"sort"
	"strconv"

	"github.com/hhrutter/pdflib/filter"
	"github.com/hhrutter/pdflib/optimize"
	"github.com/hhrutter/pdflib/types"
)

func writeFont(ctx *types.PDFContext, fileName, extension string, fontFileIndRef types.PDFIndirectRef, objNr int) (err error) {

	fileName = fileName + "_" + strconv.Itoa(objNr) + "." + extension

	logDebugExtract.Printf("writeFont begin: writing to %s\n", fileName)

	streamDict, err := ctx.DereferenceStreamDict(fontFileIndRef)
	if err != nil {
		return
	}

	// Decode streamDict if used filter supported only.
	err = filter.DecodeStream(streamDict)
	if err != nil {
		return
	}

	// Dump decoded chunk to file.
	err = ioutil.WriteFile(fileName, streamDict.Content, os.ModePerm)
	if err != nil {
		return
	}

	logDebugExtract.Printf("writeFont end")

	return
}

func sortFOKeys(m map[int]*types.FontObject) (j []int) {
	for i := range m {
		j = append(j, i)
	}
	sort.Ints(j)
	return
}

func writeFontObject(ctx *types.PDFContext, objNumber int, fontFileIndRefs *map[types.PDFIndirectRef]bool) (err error) {

	fontObject := ctx.Optimize.FontObjects[objNumber]

	// Only embedded fonts have binary data.
	if !fontObject.Embedded() {
		logInfoExtract.Printf("writeFonts: ignoring - non embedded font\n")
		return
	}

	dict, err := optimize.FontDescriptor(ctx.XRefTable, fontObject.FontDict, objNumber)
	if err != nil || dict == nil {
		logInfoExtract.Printf("writeFonts: ignoring - no fontDescriptor available\n")
		return
	}

	indRef := optimize.FontDescriptorFontFileIndirectObjectRef(*dict)
	if indRef == nil {
		logInfoExtract.Printf("writeFonts: ignoring - no font file available\n")
		return
	}

	if (*fontFileIndRefs)[*indRef] {
		logInfoExtract.Printf("writeFonts: ignoring - already written\n")
		return
	}

	fontType := fontObject.SubType()

	fileName := ctx.Write.DirName + "/" + fontObject.FontName + "_" + fontType

	switch fontType {

	case "TrueType":
		// ttf ... true type file
		// ttc ... true type collection
		// This is just me guessing..
		err = writeFont(ctx, fileName, "ttf", *indRef, objNumber)
		if err != nil {
			return err
		}

	default:
		logInfoExtract.Printf("writeFonts: ignoring - unsupported fonttype: %s\n", fontType)
		return
	}

	(*fontFileIndRefs)[*indRef] = true

	return
}

func writeFonts(ctx *types.PDFContext, selectedPages types.IntSet) (err error) {

	logDebugExtract.Println("writeFonts begin")

	oc := ctx.Optimize

	fontFileIndRefs := map[types.PDFIndirectRef]bool{}

	if selectedPages == nil || len(selectedPages) == 0 {

		for _, i := range sortFOKeys(oc.FontObjects) {
			logInfoExtract.Printf("writeFonts: processing fontobject %d\n", i)
			writeFontObject(ctx, i, &fontFileIndRefs)
		}

	} else {

		for p, v := range selectedPages {
			if v {
				logErrorExtract.Printf("writeFonts: writing fonts for page %d\n", p)
				objs := oc.PageFonts[p-1]
				if len(objs) == 0 {
					// This page has no fonts.
					logErrorExtract.Printf("writeFonts: Page %d does not have fonts to extract\n", p)
					continue
				}
				for i := range objs {
					writeFontObject(ctx, i, &fontFileIndRefs)
				}
			}
		}

	}

	logDebugExtract.Println("writeFonts end")

	return
}

// Fonts writes embedded font files for selected pages to dirOut.
// Supported font types: TrueType.
func Fonts(ctx *types.PDFContext, selectedPages types.IntSet) (err error) {

	logDebugExtract.Println("Fonts begin")

	if len(ctx.Optimize.FontObjects) == 0 {
		logInfoExtract.Println("No font info available.")
		return
	}

	err = writeFonts(ctx, selectedPages)
	if err != nil {
		return
	}

	logDebugExtract.Println("Fonts end")

	return
}
