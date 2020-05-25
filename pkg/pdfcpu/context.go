/*
Copyright 2018 The pdfcpu Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pdfcpu

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/log"
)

// Context represents an environment for processing PDF files.
type Context struct {
	*Configuration
	*XRefTable
	Read         *ReadContext
	Optimize     *OptimizationContext
	Write        *WriteContext
	writingPages bool // true, when writing page dicts.
	dest         bool // true when writing a destination within a page.
}

// NewContext initializes a new Context.
func NewContext(rs io.ReadSeeker, conf *Configuration) (*Context, error) {

	if conf == nil {
		conf = NewDefaultConfiguration()
	}

	ctx := &Context{
		conf,
		newXRefTable(conf.ValidationMode),
		newReadContext(rs),
		newOptimizationContext(),
		NewWriteContext(conf.Eol),
		false,
		false,
	}

	return ctx, nil
}

// ResetWriteContext prepares an existing WriteContext for a new file to be written.
func (ctx *Context) ResetWriteContext() {
	ctx.Write = NewWriteContext(ctx.Write.Eol)
}

func (rc *ReadContext) logReadContext(logStr *[]string) {
	if rc.UsingObjectStreams {
		*logStr = append(*logStr, "using object streams\n")
	}
	if rc.UsingXRefStreams {
		*logStr = append(*logStr, "using xref streams\n")
	}
	if rc.Linearized {
		*logStr = append(*logStr, "is linearized file\n")
	}
	if rc.Hybrid {
		*logStr = append(*logStr, "is hybrid reference file\n")
	}
}

func (ctx *Context) String() string {

	var logStr []string

	logStr = append(logStr, "*************************************************************************************************\n")
	logStr = append(logStr, fmt.Sprintf("HeaderVersion: %s\n", ctx.HeaderVersion))

	if ctx.RootVersion != nil {
		logStr = append(logStr, fmt.Sprintf("RootVersion: %s\n", ctx.RootVersion))
	}

	logStr = append(logStr, fmt.Sprintf("has %d pages\n", ctx.PageCount))

	ctx.Read.logReadContext(&logStr)

	if ctx.Tagged {
		logStr = append(logStr, "is tagged file\n")
	}

	logStr = append(logStr, "XRefTable:\n")
	logStr = append(logStr, fmt.Sprintf("                     Size: %d\n", *ctx.XRefTable.Size))
	logStr = append(logStr, fmt.Sprintf("              Root object: %s\n", *ctx.Root))

	if ctx.Info != nil {
		logStr = append(logStr, fmt.Sprintf("              Info object: %s\n", *ctx.Info))
	}

	if ctx.ID != nil {
		logStr = append(logStr, fmt.Sprintf("                ID object: %s\n", ctx.ID))
	}

	if ctx.Encrypt != nil {
		logStr = append(logStr, fmt.Sprintf("           Encrypt object: %s\n", *ctx.Encrypt))
	}

	if ctx.AdditionalStreams != nil && len(*ctx.AdditionalStreams) > 0 {

		var objectNumbers []string
		for _, k := range *ctx.AdditionalStreams {
			indRef, _ := k.(IndirectRef)
			objectNumbers = append(objectNumbers, fmt.Sprintf("%d", int(indRef.ObjectNumber)))
		}
		sort.Strings(objectNumbers)

		logStr = append(logStr, fmt.Sprintf("        AdditionalStreams: %s\n\n", strings.Join(objectNumbers, ",")))
	}

	logStr = append(logStr, fmt.Sprintf("XRefTable with %d entres:\n", len(ctx.Table)))

	// Print sorted object list.
	logStr = ctx.list(logStr)

	// Print free list.
	logStr, err := ctx.freeList(logStr)
	if err != nil {
		log.Info.Fatalln(err)
	}

	// Print list of any missing objects.
	if len(ctx.XRefTable.Table) < *ctx.XRefTable.Size {
		if count, mstr := ctx.MissingObjects(); count > 0 {
			logStr = append(logStr, fmt.Sprintf("%d missing objects: %s\n", count, *mstr))
		}
	}

	logStr = append(logStr, fmt.Sprintf("\nTotal pages: %d\n", ctx.PageCount))
	logStr = ctx.Optimize.collectFontInfo(logStr)
	logStr = ctx.Optimize.collectImageInfo(logStr)
	logStr = append(logStr, "\n")

	return strings.Join(logStr, "")
}

func (ctx *Context) units() string {
	u := "points"
	switch ctx.Units {
	case INCHES:
		u = "inches"
	case CENTIMETRES:
		u = "cm"
	case MILLIMETRES:
		u = "mm"
	}
	return u
}

func (ctx *Context) convertToUnits(d Dim) Dim {
	switch ctx.Units {
	case INCHES:
		return d.ToInches()
	case CENTIMETRES:
		return d.ToCentimetres()
	case MILLIMETRES:
		return d.ToMillimetres()
	}
	return d
}

func (ctx *Context) addKeywordsToInfoDigest(ss *[]string) error {
	if len(ctx.Keywords) == 0 {
		return nil
	}
	kwl, err := KeywordsList(ctx.XRefTable)
	if err != nil {
		return err
	}
	for i, l := range kwl {
		if i == 0 {
			*ss = append(*ss, fmt.Sprintf("%20s: %s", "Keywords", l))
			continue
		}
		*ss = append(*ss, fmt.Sprintf("%20s  %s", "", l))
	}
	return nil
}

func (ctx *Context) addPropertiesToInfoDigest(ss *[]string) error {
	if len(ctx.Properties) == 0 {
		return nil
	}
	first := true
	for k, v := range ctx.Properties {
		if first {
			*ss = append(*ss, fmt.Sprintf("%20s: %s = %s", "Properties", k, v))
			first = false
			continue
		}
		*ss = append(*ss, fmt.Sprintf("%20s  %s = %s", "", k, v))
	}
	return nil
}

func (ctx *Context) addPermissionsToInfoDigest(ss *[]string) {
	l := Permissions(ctx)
	if len(l) == 1 {
		*ss = append(*ss, fmt.Sprintf("%20s: %s", "Permissions", l[0]))
	} else {
		*ss = append(*ss, fmt.Sprintf("%20s:", "Permissions"))
		for _, s := range l {
			*ss = append(*ss, s)
		}
	}
}

// InfoDigest returns info about ctx.
func (ctx *Context) InfoDigest() ([]string, error) {
	var separator = "............................................"
	var ss []string
	v := ctx.HeaderVersion
	if ctx.RootVersion != nil {
		v = ctx.RootVersion
	}
	ss = append(ss, fmt.Sprintf("%20s: %s", "PDF version", v))
	ss = append(ss, fmt.Sprintf("%20s: %d", "Page count", ctx.PageCount))

	pd, err := ctx.PageDims()
	if err != nil {
		return nil, err
	}

	m := map[Dim]bool{}
	for _, d := range pd {
		m[d] = true
	}

	units := ctx.units()
	s := "Page size:"
	for d := range m {
		dc := ctx.convertToUnits(d)
		ss = append(ss, fmt.Sprintf("%21s %.2f x %.2f %s", s, dc.Width, dc.Height, units))
		s = ""
	}

	ss = append(ss, fmt.Sprintf(separator))
	ss = append(ss, fmt.Sprintf("%20s: %s", "Title", ctx.Title))
	ss = append(ss, fmt.Sprintf("%20s: %s", "Author", ctx.Author))
	ss = append(ss, fmt.Sprintf("%20s: %s", "Subject", ctx.Subject))
	ss = append(ss, fmt.Sprintf("%20s: %s", "PDF Producer", ctx.Producer))
	ss = append(ss, fmt.Sprintf("%20s: %s", "Content creator", ctx.Creator))
	ss = append(ss, fmt.Sprintf("%20s: %s", "Creation date", ctx.CreationDate))
	ss = append(ss, fmt.Sprintf("%20s: %s", "Modification date", ctx.ModDate))

	if err := ctx.addKeywordsToInfoDigest(&ss); err != nil {
		return nil, err
	}

	if err := ctx.addPropertiesToInfoDigest(&ss); err != nil {
		return nil, err
	}

	ss = append(ss, fmt.Sprintf(separator))

	s = "No"
	if ctx.Tagged {
		s = "Yes"
	}
	ss = append(ss, fmt.Sprintf("              Tagged: %s", s))

	s = "No"
	if ctx.Read.Hybrid {
		s = "Yes"
	}
	ss = append(ss, fmt.Sprintf("              Hybrid: %s", s))

	s = "No"
	if ctx.Read.Linearized {
		s = "Yes"
	}
	ss = append(ss, fmt.Sprintf("          Linearized: %s", s))

	s = "No"
	if ctx.Read.UsingXRefStreams {
		s = "Yes"
	}
	ss = append(ss, fmt.Sprintf("  Using XRef streams: %s", s))

	s = "No"
	if ctx.Read.UsingObjectStreams {
		s = "Yes"
	}
	ss = append(ss, fmt.Sprintf("Using object streams: %s", s))

	s = "No"
	if ctx.Watermarked {
		s = "Yes"
	}
	ss = append(ss, fmt.Sprintf("         Watermarked: %s", s))

	ss = append(ss, fmt.Sprintf(separator))

	s = "No"
	if ctx.Encrypt != nil {
		s = "Yes"
	}
	ss = append(ss, fmt.Sprintf("%20s: %s", "Encrypted", s))

	ctx.addPermissionsToInfoDigest(&ss)

	//if ctx.ID != nil {
	//	ss = append(ss, fmt.Sprintf("Id: %s", ctx.ID))
	//}

	return ss, nil
}

// ReadContext represents the context for reading a PDF file.
type ReadContext struct {
	FileName            string // The input PDF-File.
	FileSize            int64
	rs                  io.ReadSeeker
	EolCount            int    // 1 or 2 characters used for eol.
	BinaryTotalSize     int64  // total stream data
	BinaryImageSize     int64  // total image stream data
	BinaryFontSize      int64  // total font stream data (fontfiles)
	BinaryImageDuplSize int64  // total obsolet image stream data after optimization
	BinaryFontDuplSize  int64  // total obsolet font stream data after optimization
	Linearized          bool   // File is linearized.
	Hybrid              bool   // File is a hybrid PDF file.
	UsingObjectStreams  bool   // File is using object streams.
	ObjectStreams       IntSet // All object numbers of any object streams found which need to be decoded.
	UsingXRefStreams    bool   // File is using xref streams.
	XRefStreams         IntSet // All object numbers of any xref streams found.
}

func newReadContext(rs io.ReadSeeker) *ReadContext {
	return &ReadContext{
		rs:            rs,
		ObjectStreams: IntSet{},
		XRefStreams:   IntSet{},
	}
}

// IsObjectStreamObject returns true if object i is a an object stream.
// All compressed objects are object streams.
func (rc *ReadContext) IsObjectStreamObject(i int) bool {
	return rc.ObjectStreams[i]
}

// ObjectStreamsString returns a formatted string and the number of object stream objects.
func (rc *ReadContext) ObjectStreamsString() (int, string) {

	var objs []int
	for k := range rc.ObjectStreams {
		if rc.ObjectStreams[k] {
			objs = append(objs, k)
		}
	}
	sort.Ints(objs)

	var objStreams []string
	for _, i := range objs {
		objStreams = append(objStreams, fmt.Sprintf("%d", i))
	}

	return len(objStreams), strings.Join(objStreams, ",")
}

// IsXRefStreamObject returns true if object #i is a an xref stream.
func (rc *ReadContext) IsXRefStreamObject(i int) bool {
	return rc.XRefStreams[i]
}

// XRefStreamsString returns a formatted string and the number of xref stream objects.
func (rc *ReadContext) XRefStreamsString() (int, string) {

	var objs []int
	for k := range rc.XRefStreams {
		if rc.XRefStreams[k] {
			objs = append(objs, k)
		}
	}
	sort.Ints(objs)

	var xrefStreams []string
	for _, i := range objs {
		xrefStreams = append(xrefStreams, fmt.Sprintf("%d", i))
	}

	return len(xrefStreams), strings.Join(xrefStreams, ",")
}

// LogStats logs stats for read file.
func (rc *ReadContext) LogStats(optimized bool) {

	log := log.Stats

	textSize := rc.FileSize - rc.BinaryTotalSize // = non binary content = non stream data

	log.Println("Original:")
	log.Printf("File Size            : %s (%d bytes)\n", ByteSize(rc.FileSize), rc.FileSize)
	log.Printf("Total Binary Data    : %s (%d bytes) %4.1f%%\n", ByteSize(rc.BinaryTotalSize), rc.BinaryTotalSize, float32(rc.BinaryTotalSize)/float32(rc.FileSize)*100)
	log.Printf("Total Text   Data    : %s (%d bytes) %4.1f%%\n\n", ByteSize(textSize), textSize, float32(textSize)/float32(rc.FileSize)*100)

	// Only when optimizing we get details about resource data usage.
	if optimized {

		// Image stream data of original file.
		binaryImageSize := rc.BinaryImageSize + rc.BinaryImageDuplSize

		// Font stream data of original file. (just font files)
		binaryFontSize := rc.BinaryFontSize + rc.BinaryFontDuplSize

		// Content stream data, other font related stream data.
		binaryOtherSize := rc.BinaryTotalSize - binaryImageSize - binaryFontSize

		log.Println("Breakup of binary data:")
		log.Printf("images               : %s (%d bytes) %4.1f%%\n", ByteSize(binaryImageSize), binaryImageSize, float32(binaryImageSize)/float32(rc.BinaryTotalSize)*100)
		log.Printf("fonts                : %s (%d bytes) %4.1f%%\n", ByteSize(binaryFontSize), binaryFontSize, float32(binaryFontSize)/float32(rc.BinaryTotalSize)*100)
		log.Printf("other                : %s (%d bytes) %4.1f%%\n\n", ByteSize(binaryOtherSize), binaryOtherSize, float32(binaryOtherSize)/float32(rc.BinaryTotalSize)*100)
	}
}

// ReadFileSize returns the size of the input file, if there is one.
func (rc *ReadContext) ReadFileSize() int {
	if rc == nil {
		return 0
	}
	return int(rc.FileSize)
}

// OptimizationContext represents the context for the optimiziation of a PDF file.
type OptimizationContext struct {

	// Font section
	PageFonts         []IntSet            // For each page a registry of font object numbers.
	FontObjects       map[int]*FontObject // FontObject lookup table by font object number.
	Fonts             map[string][]int    // All font object numbers registered for a font name.
	DuplicateFonts    map[int]Dict        // Registry of duplicate font dicts.
	DuplicateFontObjs IntSet              // The set of objects that represents the union of the object graphs of all duplicate font dicts.

	// Image section
	PageImages         []IntSet             // For each page a registry of image object numbers.
	ImageObjects       map[int]*ImageObject // ImageObject lookup table by image object number.
	DuplicateImages    map[int]*StreamDict  // Registry of duplicate image dicts.
	DuplicateImageObjs IntSet               // The set of objects that represents the union of the object graphs of all duplicate image dicts.

	DuplicateInfoObjects IntSet // Possible result of manual info dict modification.
	NonReferencedObjs    []int  // Objects that are not referenced.

	FixTable  map[int]bool // map for visited objects during xreftable traversal for fixing references to free objects.
	NullObjNr *int         // objNr of a regular null object, to be used for fixing references to free objects.
}

func newOptimizationContext() *OptimizationContext {
	return &OptimizationContext{
		FontObjects:          map[int]*FontObject{},
		Fonts:                map[string][]int{},
		DuplicateFonts:       map[int]Dict{},
		DuplicateFontObjs:    IntSet{},
		ImageObjects:         map[int]*ImageObject{},
		DuplicateImages:      map[int]*StreamDict{},
		DuplicateImageObjs:   IntSet{},
		DuplicateInfoObjects: IntSet{},
		FixTable:             map[int]bool{},
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

// WriteContext represents the context for writing a PDF file.
type WriteContext struct {

	// The PDF-File which gets generated.
	*bufio.Writer
	DirName             string
	FileName            string
	FileSize            int64
	SelectedPages       IntSet        // For split, trim and extract.
	BinaryTotalSize     int64         // total stream data, counts 100% all stream data written.
	BinaryImageSize     int64         // total image stream data written = Read.BinaryImageSize.
	BinaryFontSize      int64         // total font stream data (fontfiles) = copy of Read.BinaryFontSize.
	Table               map[int]int64 // object write offsets
	Offset              int64         // current write offset
	WriteToObjectStream bool          // if true start to embed objects into object streams and obey ObjectStreamMaxObjects.
	CurrentObjStream    *int          // if not nil, any new non-stream-object gets added to the object stream with this object number.
	Eol                 string        // end of line char sequence
}

// NewWriteContext returns a new WriteContext.
func NewWriteContext(eol string) *WriteContext {
	return &WriteContext{SelectedPages: IntSet{}, Table: map[int]int64{}, Eol: eol}
}

// SetWriteOffset saves the current write offset to the PDFDestination.
func (wc *WriteContext) SetWriteOffset(objNumber int) {
	wc.Table[objNumber] = wc.Offset
}

// HasWriteOffset returns true if an object has already been written to PDFDestination.
func (wc *WriteContext) HasWriteOffset(objNumber int) bool {
	_, found := wc.Table[objNumber]
	return found
}

// LogStats logs stats for written file.
func (wc *WriteContext) LogStats() {

	fileSize := wc.FileSize
	binaryTotalSize := wc.BinaryTotalSize  // stream data
	textSize := fileSize - binaryTotalSize // non stream data

	binaryImageSize := wc.BinaryImageSize
	binaryFontSize := wc.BinaryFontSize
	binaryOtherSize := binaryTotalSize - binaryImageSize - binaryFontSize // content streams

	log.Stats.Println("Optimized:")
	log.Stats.Printf("File Size            : %s (%d bytes)\n", ByteSize(fileSize), fileSize)
	log.Stats.Printf("Total Binary Data    : %s (%d bytes) %4.1f%%\n", ByteSize(binaryTotalSize), binaryTotalSize, float32(binaryTotalSize)/float32(fileSize)*100)
	log.Stats.Printf("Total Text   Data    : %s (%d bytes) %4.1f%%\n\n", ByteSize(textSize), textSize, float32(textSize)/float32(fileSize)*100)

	log.Stats.Println("Breakup of binary data:")
	log.Stats.Printf("images               : %s (%d bytes) %4.1f%%\n", ByteSize(binaryImageSize), binaryImageSize, float32(binaryImageSize)/float32(binaryTotalSize)*100)
	log.Stats.Printf("fonts                : %s (%d bytes) %4.1f%%\n", ByteSize(binaryFontSize), binaryFontSize, float32(binaryFontSize)/float32(binaryTotalSize)*100)
	log.Stats.Printf("other                : %s (%d bytes) %4.1f%%\n\n", ByteSize(binaryOtherSize), binaryOtherSize, float32(binaryOtherSize)/float32(binaryTotalSize)*100)
}

// WriteEol writes an end of line sequence.
func (wc *WriteContext) WriteEol() error {

	_, err := wc.WriteString(wc.Eol)

	return err
}
