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
// Right now supported:
// "DCTDecode" dumps to a jpg file.
// "JPXDecode" dumps to a jpx file.
func writeImage(fileName string, imageDict *types.PDFStreamDict, objNr int) error {

	fpl := imageDict.FilterPipeline
	if fpl == nil {
		return nil
	}

	var s []string
	for _, filter := range fpl {
		s = append(s, filter.Name)
	}
	filters := strings.Join(s, ",")

	fileName = fileName + "_" + strconv.Itoa(objNr) + "_" + filters

	logDebugExtract.Printf("writeImage begin: %s objNR:%d\n", fileName, objNr)

	// Ignore filter chains with length > 1
	if len(fpl) > 1 {
		logInfoExtract.Printf("writeImage end: ignore %s, more than 1 filter.\n", fileName)
		return nil
	}

	// Ignore imageMasks
	if im := imageDict.BooleanEntry("ImageMask"); im != nil && *im {
		logInfoExtract.Printf("writeImage end: ignore %s, imageMask.\n", fileName)
		return nil
	}

	switch fpl[0].Name {

	case "DCTDecode":
		// Dump encoded chunk to file.
		logInfoExtract.Printf("writing %s\n", fileName+".jpg")
		err := ioutil.WriteFile(fileName+".jpg", imageDict.Raw, os.ModePerm)
		if err != nil {
			return err
		}

	case "JPXDecode":
		// Dump encoded chunk to file.
		logInfoExtract.Printf("writing %s\n", fileName+".jpx")
		err := ioutil.WriteFile(fileName+".jpx", imageDict.Raw, os.ModePerm)
		if err != nil {
			return err
		}

	default:
		logDebugExtract.Printf("writeImage end: ignore %s filter neither \"DCTDecode\" nor \"JPXDecode\"\n", fileName)
		return nil
	}

	logDebugExtract.Printf("writeImage end")

	return nil
}

func sortIOKeys(m map[int]*types.ImageObject) (j []int) {
	for i := range m {
		j = append(j, i)
	}
	sort.Ints(j)
	return
}

func writeImageObject(ctx *types.PDFContext, objNumber int) error {
	obj := ctx.Optimize.ImageObjects[objNumber]
	logDebugExtract.Printf("%s\n%s", obj.ResourceNamesString(), obj.ImageDict)
	fileName := ctx.Write.DirName + "/" + obj.ResourceNamesString()
	return writeImage(fileName, obj.ImageDict, objNumber)
}

func writeImages(ctx *types.PDFContext, selectedPages types.IntSet) error {

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
				for i := range ctx.Optimize.PageImages[p-1] {
					writeImageObject(ctx, i)
				}
			}

		}

	}

	logDebugExtract.Println("writeImages end")

	return nil
}

// Images writes embedded image resources for selected pages to dirOut.
// Supported PDF filters: DCT, JPX
func Images(ctx *types.PDFContext, selectedPages types.IntSet) error {

	logDebugExtract.Println("Images begin")

	if len(ctx.Optimize.ImageObjects) == 0 {
		logInfoExtract.Println("No image info available.")
		return nil
	}

	err := writeImages(ctx, selectedPages)
	if err != nil {
		return err
	}

	logDebugExtract.Println("Images end")

	return nil
}
