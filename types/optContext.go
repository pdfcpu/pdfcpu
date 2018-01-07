package types

import (
	"fmt"
	"sort"
	"strings"
)

// OptimizationContext represents the context for the optimiziation of a PDF file.
type OptimizationContext struct {

	// Font section
	PageFonts         []IntSet
	FontObjects       map[int]*FontObject
	Fonts             map[string][]int
	DuplicateFontObjs IntSet
	DuplicateFonts    map[int]*PDFDict

	// Image section
	PageImages         []IntSet
	ImageObjects       map[int]*ImageObject
	DuplicateImageObjs IntSet
	DuplicateImages    map[int]*PDFStreamDict

	DuplicateInfoObjects IntSet // Possible result of manual info dict modification.

	NonReferencedObjs []int // Objects that are not referenced.
}

func newOptimizationContext() *OptimizationContext {
	return &OptimizationContext{
		FontObjects:          map[int]*FontObject{},
		Fonts:                map[string][]int{},
		DuplicateFontObjs:    IntSet{},
		DuplicateFonts:       map[int]*PDFDict{},
		ImageObjects:         map[int]*ImageObject{},
		DuplicateImageObjs:   IntSet{},
		DuplicateImages:      map[int]*PDFStreamDict{},
		DuplicateInfoObjects: IntSet{},
	}
}

// IsDuplicateFontObject returns true if object #i is a duplicate font object.
func (oc *OptimizationContext) IsDuplicateFontObject(i int) bool {
	return oc.DuplicateFontObjs[i]
}

// DuplicateFontObjectsString returns a formatted string and the number of objs.
func (oc *OptimizationContext) DuplicateFontObjectsString() (int, string) {

	var objs []int
	for k := range oc.DuplicateFontObjs {
		if oc.DuplicateFontObjs[k] {
			objs = append(objs, k)
		}
	}
	sort.Ints(objs)

	var dupFonts []string
	for _, i := range objs {
		dupFonts = append(dupFonts, fmt.Sprintf("%d", i))
	}

	return len(dupFonts), strings.Join(dupFonts, ",")
}

// IsDuplicateImageObject returns true if object #i is a duplicate image object.
func (oc *OptimizationContext) IsDuplicateImageObject(i int) bool {
	return oc.DuplicateImageObjs[i]
}

// DuplicateImageObjectsString returns a formatted string and the number of objs.
func (oc *OptimizationContext) DuplicateImageObjectsString() (int, string) {

	var objs []int
	for k := range oc.DuplicateImageObjs {
		if oc.DuplicateImageObjs[k] {
			objs = append(objs, k)
		}
	}
	sort.Ints(objs)

	var dupImages []string
	for _, i := range objs {
		dupImages = append(dupImages, fmt.Sprintf("%d", i))
	}

	return len(dupImages), strings.Join(dupImages, ",")
}

// IsDuplicateInfoObject returns true if object #i is a duplicate info object.
func (oc *OptimizationContext) IsDuplicateInfoObject(i int) bool {
	return oc.DuplicateInfoObjects[i]
}

// DuplicateInfoObjectsString returns a formatted string and the number of objs.
func (oc *OptimizationContext) DuplicateInfoObjectsString() (int, string) {

	var objs []int
	for k := range oc.DuplicateInfoObjects {
		if oc.DuplicateInfoObjects[k] {
			objs = append(objs, k)
		}
	}
	sort.Ints(objs)

	var dupInfos []string
	for _, i := range objs {
		dupInfos = append(dupInfos, fmt.Sprintf("%d", i))
	}

	return len(dupInfos), strings.Join(dupInfos, ",")
}

// NonReferencedObjsString returns a formatted string and the number of objs.
func (oc *OptimizationContext) NonReferencedObjsString() (int, string) {

	var s []string
	for _, o := range oc.NonReferencedObjs {
		s = append(s, fmt.Sprintf("%d", o))
	}

	return len(oc.NonReferencedObjs), strings.Join(s, ",")
}

// Prepare info gathered about font usage in form of a string array.
func (oc *OptimizationContext) collectFontInfo(logStr []string) []string {

	// Print available font info.
	if len(oc.Fonts) == 0 || len(oc.PageFonts) == 0 {
		return append(logStr, "No font info available.\n")
	}

	fontHeader := "obj     prefix     Fontname                       Subtype    Encoding             Embedded ResourceIds\n"

	// Log fonts usage per page.
	for i, fontObjectNumbers := range oc.PageFonts {

		if len(fontObjectNumbers) == 0 {
			continue
		}

		logStr = append(logStr, fmt.Sprintf("\nFonts for page %d:\n", i+1))
		logStr = append(logStr, fontHeader)

		var objectNumbers []int
		for k := range fontObjectNumbers {
			objectNumbers = append(objectNumbers, k)
		}
		sort.Ints(objectNumbers)

		for _, objectNumber := range objectNumbers {
			fontObject := oc.FontObjects[objectNumber]
			logStr = append(logStr, fmt.Sprintf("#%-6d %s", objectNumber, fontObject))
		}
	}

	// Log all fonts sorted by object number.
	logStr = append(logStr, fmt.Sprintf("\nFontobjects:\n"))
	logStr = append(logStr, fontHeader)

	var objectNumbers []int
	for k := range oc.FontObjects {
		objectNumbers = append(objectNumbers, k)
	}
	sort.Ints(objectNumbers)

	for _, objectNumber := range objectNumbers {
		fontObject := oc.FontObjects[objectNumber]
		logStr = append(logStr, fmt.Sprintf("#%-6d %s", objectNumber, fontObject))
	}

	// Log all fonts sorted by fontname.
	logStr = append(logStr, fmt.Sprintf("\nFonts:\n"))
	logStr = append(logStr, fontHeader)

	var fontNames []string
	for k := range oc.Fonts {
		fontNames = append(fontNames, k)
	}
	sort.Strings(fontNames)

	for _, fontName := range fontNames {
		for _, objectNumber := range oc.Fonts[fontName] {
			fontObject := oc.FontObjects[objectNumber]
			logStr = append(logStr, fmt.Sprintf("#%-6d %s", objectNumber, fontObject))
		}
	}

	logStr = append(logStr, fmt.Sprintf("\nDuplicate Fonts:\n"))

	// Log any duplicate fonts.
	if len(oc.DuplicateFonts) > 0 {

		var objectNumbers []int
		for k := range oc.DuplicateFonts {
			objectNumbers = append(objectNumbers, k)
		}
		sort.Ints(objectNumbers)

		var f []string

		for _, i := range objectNumbers {
			f = append(f, fmt.Sprintf("%d", i))
		}

		logStr = append(logStr, strings.Join(f, ","))
	}

	return append(logStr, "\n")
}

// Prepare info gathered about image usage in form of a string array.
func (oc *OptimizationContext) collectImageInfo(logStr []string) []string {

	// Print available image info.
	if len(oc.ImageObjects) == 0 {
		return append(logStr, "\nNo image info available.\n")
	}

	imageHeader := "obj     ResourceIds\n"

	// Log images per page.
	for i, imageObjectNumbers := range oc.PageImages {

		if len(imageObjectNumbers) == 0 {
			continue
		}

		logStr = append(logStr, fmt.Sprintf("\nImages for page %d:\n", i+1))
		logStr = append(logStr, imageHeader)

		var objectNumbers []int
		for k := range imageObjectNumbers {
			objectNumbers = append(objectNumbers, k)
		}
		sort.Ints(objectNumbers)

		for _, objectNumber := range objectNumbers {
			imageObject := oc.ImageObjects[objectNumber]
			logStr = append(logStr, fmt.Sprintf("#%-6d %s\n", objectNumber, imageObject.ResourceNamesString()))
		}
	}

	// Log all images sorted by object number.
	logStr = append(logStr, fmt.Sprintf("\nImageobjects:\n"))
	logStr = append(logStr, imageHeader)

	var objectNumbers []int
	for k := range oc.ImageObjects {
		objectNumbers = append(objectNumbers, k)
	}
	sort.Ints(objectNumbers)

	for _, objectNumber := range objectNumbers {
		imageObject := oc.ImageObjects[objectNumber]
		logStr = append(logStr, fmt.Sprintf("#%-6d %s\n", objectNumber, imageObject.ResourceNamesString()))
	}

	logStr = append(logStr, fmt.Sprintf("\nDuplicate Images:\n"))

	// Log any duplicate images.
	if len(oc.DuplicateImages) > 0 {

		var objectNumbers []int
		for k := range oc.DuplicateImages {
			objectNumbers = append(objectNumbers, k)
		}
		sort.Ints(objectNumbers)

		var f []string

		for _, i := range objectNumbers {
			f = append(f, fmt.Sprintf("%d", i))
		}

		logStr = append(logStr, strings.Join(f, ","))
	}

	return logStr
}
