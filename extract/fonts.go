package extract

import (
	"io/ioutil"
	"os"
	"sort"
	"strconv"

	"github.com/hhrutter/pdfcpu/filter"
	"github.com/hhrutter/pdfcpu/optimize"
	"github.com/hhrutter/pdfcpu/types"
)

func writeFont(ctx *types.PDFContext, fileName, extension string, fontFileIndRef *types.PDFIndirectRef, objNr int) (err error) {

	fileName = fileName + "_" + strconv.Itoa(objNr) + "." + extension

	logInfoExtract.Printf("writing %s\n", fileName)
	logDebugExtract.Printf("writeFont begin: writing to %s\n", fileName)

	streamDict, err := ctx.DereferenceStreamDict(*fontFileIndRef)
	if err != nil {
		return
	}

	// Decode streamDict if used filter is supported only.
	err = filter.DecodeStream(streamDict)
	if err == filter.ErrUnsupportedFilter {
		err = nil
		return
	}
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

// Stupid dump of the font file for this font object.
// Right now only True Type fonts are supported.
func writeFontObject(ctx *types.PDFContext, objNumber int, fontFileIndRefs map[types.PDFIndirectRef]bool) (err error) {

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

	indRef := optimize.FontDescriptorFontFileIndirectObjectRef(dict)
	if indRef == nil {
		logInfoExtract.Printf("writeFonts: ignoring - no font file available\n")
		return
	}

	if fontFileIndRefs[*indRef] {
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
		err = writeFont(ctx, fileName, "ttf", indRef, objNumber)
		if err != nil {
			return err
		}

	default:
		logInfoExtract.Printf("writeFonts: ignoring - unsupported fonttype: %s\n", fontType)
		return
	}

	fontFileIndRefs[*indRef] = true

	return
}

func writeFonts(ctx *types.PDFContext, selectedPages types.IntSet) (err error) {

	logDebugExtract.Println("writeFonts begin")

	fontFileIndRefs := map[types.PDFIndirectRef]bool{}

	if selectedPages == nil || len(selectedPages) == 0 {

		for _, i := range sortFOKeys(ctx.Optimize.FontObjects) {
			logDebugExtract.Printf("writeFonts: processing fontobject %d\n", i)
			writeFontObject(ctx, i, fontFileIndRefs)
		}

	} else {

		for p, v := range selectedPages {

			if v {
				logInfoExtract.Printf("writeFonts: writing fonts for page %d\n", p)
				for i := range ctx.Optimize.PageFonts[p-1] {
					writeFontObject(ctx, i, fontFileIndRefs)
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
