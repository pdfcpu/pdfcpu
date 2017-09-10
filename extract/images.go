package extract

import (
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/hhrutter/pdfcpu/types"
)

// Stupid dump of image data to a file.
// Right now supported are:
// "DCTDecode" dumps to a jpg file.
// "JPXDecode" dumps to a jpx file.
func writeImage(fileName string, imageDict *types.PDFStreamDict, objNr int) (err error) {

	var filters string

	fpl := imageDict.FilterPipeline
	if fpl == nil {
		filters = "none"
	} else {
		var s []string
		for _, filter := range fpl {
			s = append(s, filter.Name)
		}
		filters = strings.Join(s, ",")
	}

	fileName = fileName + "_" + strconv.Itoa(objNr) + "_" + filters

	logDebugExtract.Printf("writeImage begin: %s objNR:%d\n", fileName, objNr)

	if fpl != nil {

		// Ignore filter chains with length > 1
		if len(fpl) > 1 {
			logInfoExtract.Printf("writeImage end: ignore %s, more than 1 filter.\n", fileName)
			return
		}

		// Ignore imageMasks
		if imageDict.BooleanEntry("ImageMask") {
			logInfoExtract.Printf("writeImage end: ignore %s, imageMask.\n", fileName)
			return
		}

		switch fpl[0].Name {

		case "DCTDecode":
			// Dump encoded chunk to file.
			logInfoExtract.Printf("writing %s\n", fileName+".jpg")
			err = ioutil.WriteFile(fileName+".jpg", imageDict.Raw, os.ModePerm)
			if err != nil {
				return
			}

		case "JPXDecode":
			// Dump encoded chunk to file.
			logInfoExtract.Printf("writing %s\n", fileName+".jpx")
			err = ioutil.WriteFile(fileName+".jpx", imageDict.Raw, os.ModePerm)
			if err != nil {
				return
			}

		default:
			logDebugExtract.Printf("writeImage end: ignore %s filter neither \"DCTDecode\" nor \"JPXDecode\"\n", fileName)
			return
		}
	}

	logDebugExtract.Printf("writeImage end")

	return
}

func sortIOKeys(m map[int]*types.ImageObject) (j []int) {
	for i := range m {
		j = append(j, i)
	}
	sort.Ints(j)
	return
}

func writeImageObject(ctx *types.PDFContext, objNumber int) (err error) {
	obj := ctx.Optimize.ImageObjects[objNumber]
	logDebugExtract.Printf("%s\n%s", obj.ResourceNamesString(), obj.ImageDict)
	fileName := ctx.Write.DirName + "/" + obj.ResourceNamesString()
	return writeImage(fileName, obj.ImageDict, objNumber)
}

func writeImages(ctx *types.PDFContext, selectedPages types.IntSet) (err error) {

	logDebugExtract.Println("writeImages begin")

	if selectedPages == nil || len(selectedPages) == 0 {

		logInfoExtract.Println("writeImages: pages == nil, extracting images for all pages")
		for _, i := range sortIOKeys(ctx.Optimize.ImageObjects) {
			writeImageObject(ctx, i)
		}

	} else {

		logErrorExtract.Println("writeImages: extracting images for selected images")

		for p, v := range selectedPages {

			if v {

				logInfoExtract.Printf("writeImages: writing images for page %d\n", p)

				objs := ctx.Optimize.PageImages[p-1]
				if len(objs) == 0 {
					// This page has no images.
					logInfoExtract.Printf("writeImages: Page %d does not have images to extract\n", p)
					continue
				}

				for i := range objs {
					writeImageObject(ctx, i)
				}

			}

		}

	}

	logDebugExtract.Println("writeImages end")

	return
}

// Images writes embedded image resources for selected pages to dirOut.
// Supported PDF filters: DCT, JPX
func Images(ctx *types.PDFContext, selectedPages types.IntSet) (err error) {

	logDebugExtract.Println("Images begin")

	if len(ctx.Optimize.ImageObjects) == 0 {
		logInfoExtract.Println("No image info available.")
		return
	}

	err = writeImages(ctx, selectedPages)
	if err != nil {
		return
	}

	logDebugExtract.Println("Images end")

	return
}
