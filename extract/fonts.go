package extract

import (
	"io/ioutil"
	"os"
	"sort"
	"strconv"

	"github.com/hhrutter/pdfcpu/filter"
	"github.com/hhrutter/pdfcpu/log"
	"github.com/hhrutter/pdfcpu/optimize"
	"github.com/hhrutter/pdfcpu/types"
)

func writeFont(ctx *types.PDFContext, fileName, extension string, fontFileIndRef *types.PDFIndirectRef, objNr int) error {

	fileName = fileName + "_" + strconv.Itoa(objNr) + "." + extension

	log.Info.Printf("writing %s\n", fileName)
	log.Debug.Printf("writeFont begin: writing to %s\n", fileName)

	streamDict, err := ctx.DereferenceStreamDict(*fontFileIndRef)
	if err != nil {
		return err
	}

	// Decode streamDict if used filter is supported only.
	err = filter.DecodeStream(streamDict)
	if err == filter.ErrUnsupportedFilter {
		return nil
	}
	if err != nil {
		return err
	}

	// Dump decoded chunk to file.
	err = ioutil.WriteFile(fileName, streamDict.Content, os.ModePerm)
	if err != nil {
		return err
	}

	log.Debug.Printf("writeFont end")

	return nil
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
func writeFontObject(ctx *types.PDFContext, objNumber int, fontFileIndRefs map[types.PDFIndirectRef]bool) error {

	fontObject := ctx.Optimize.FontObjects[objNumber]

	// Only embedded fonts have binary data.
	if !fontObject.Embedded() {
		log.Debug.Printf("writeFonts: ignoring - non embedded font\n")
		return nil
	}

	dict, err := optimize.FontDescriptor(ctx.XRefTable, fontObject.FontDict, objNumber)
	if err != nil || dict == nil {
		log.Debug.Printf("writeFonts: ignoring - no fontDescriptor available\n")
		return nil
	}

	indRef := optimize.FontDescriptorFontFileIndirectObjectRef(dict)
	if indRef == nil {
		log.Debug.Printf("writeFonts: ignoring - no font file available\n")
		return nil
	}

	if fontFileIndRefs[*indRef] {
		log.Debug.Printf("writeFonts: ignoring - already written\n")
		return nil
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
		log.Info.Printf("writeFonts: ignoring - unsupported fonttype: %s\n", fontType)
		return nil
	}

	fontFileIndRefs[*indRef] = true

	return nil
}

func writeFonts(ctx *types.PDFContext, selectedPages types.IntSet) error {

	log.Debug.Println("writeFonts begin")

	fontFileIndRefs := map[types.PDFIndirectRef]bool{}

	if selectedPages == nil || len(selectedPages) == 0 {

		for _, i := range sortFOKeys(ctx.Optimize.FontObjects) {
			log.Debug.Printf("writeFonts: processing fontobject %d\n", i)
			writeFontObject(ctx, i, fontFileIndRefs)
		}

	} else {

		for p, v := range selectedPages {

			if v {
				log.Info.Printf("writeFonts: writing fonts for page %d\n", p)
				for i := range ctx.Optimize.PageFonts[p-1] {
					writeFontObject(ctx, i, fontFileIndRefs)
				}
			}

		}

	}

	log.Debug.Println("writeFonts end")

	return nil
}

// Fonts writes embedded font files for selected pages to dirOut.
// Supported font types: TrueType.
func Fonts(ctx *types.PDFContext, selectedPages types.IntSet) error {

	log.Info.Println("extracting fonts")
	log.Debug.Println("Fonts begin")

	if len(ctx.Optimize.FontObjects) == 0 {
		log.Debug.Println("No font info available.")
		return nil
	}

	err := writeFonts(ctx, selectedPages)
	if err != nil {
		return err
	}

	log.Debug.Println("Fonts end")

	return err
}
