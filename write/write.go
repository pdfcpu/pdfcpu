// Package write contains code that writes PDF data from memory to a file.
package write

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/hhrutter/pdflib/filter"
	"github.com/hhrutter/pdflib/types"
	"github.com/hhrutter/pdflib/validate"
	"github.com/pkg/errors"
)

const (

	// REQUIRED is used for required dict entries.
	REQUIRED = true

	// OPTIONAL is used for optional dict entries.
	OPTIONAL = false

	// ObjectStreamMaxObjects limits the number of objects within an object stream written.
	ObjectStreamMaxObjects = 100
)

var (
	logDebugWriter *log.Logger
	logInfoWriter  *log.Logger
	logErrorWriter *log.Logger
	logPages       *log.Logger
	logXRef        *log.Logger

	eol string
)

func init() {

	logDebugWriter = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	logInfoWriter = log.New(ioutil.Discard, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	logErrorWriter = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	logPages = log.New(ioutil.Discard, "PAGES: ", log.Ldate|log.Ltime|log.Lshortfile)
	logXRef = log.New(ioutil.Discard, "XREF: ", log.Ldate|log.Ltime|log.Lshortfile)

	eol = types.EolLF
}

// Verbose controls logging output.
func Verbose(verbose bool) {
	out := ioutil.Discard
	if verbose {
		out = os.Stdout
	}
	logInfoWriter = log.New(out, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	logPages = log.New(out, "PAGES: ", log.Ldate|log.Ltime|log.Lshortfile)
	logXRef = log.New(out, "XREF: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func writeVersion(ctx *types.PDFContext, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeVersion begin: offset=%d ***\n", ctx.Write.Offset)

	name, written, err := writeNameEntry(ctx, *rootDict, "rootDict", "Version", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if name != nil && !written {
		ctx.Stats.AddRootAttr(types.RootVersion)
	}

	logInfoWriter.Printf("*** writeVersion end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePages(ctx *types.PDFContext, rootDict *types.PDFDict) (pagesIndRef *types.PDFIndirectRef, err error) {

	logInfoWriter.Printf("*** writePages begin: offset=%d ***\n", ctx.Write.Offset)

	pagesIndRef = rootDict.IndirectRefEntry("Pages")
	if pagesIndRef == nil {
		err = errors.New("writePages: missing indirect obj for pages dict")
		return
	}

	if ctx.Write.ExtractPages != nil && len(ctx.Write.ExtractPages) > 0 {
		p := 0
		_, err = trimPagesDict(ctx, *pagesIndRef, &p)
		if err != nil {
			return
		}
	}

	err = writePagesDict(ctx, *pagesIndRef, 0, false, false)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writePages end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeExtensions(ctx *types.PDFContext, rootDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 7.12 Extensions Dictionary

	logInfoWriter.Printf("*** writeExtensions begin: offset=%d ***\n", ctx.Write.Offset)

	dict, written, err := writeDictEntry(ctx, rootDict, "rootDict", "Extensions", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("*** writeExtensions end: dict already written. offset=%d ***\n", ctx.Write.Offset)
		return
	}

	if dict == nil {
		logInfoWriter.Printf("writeExtensions end: dict is nil.\n")
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeExtensions: unsupported in version %s.\n", ctx.VersionString())
	}

	err = errors.New("*** writeExtensions: not supported ***")

	//dest.XRefTable.Stats.AddRootAttr(types.RootExtensions)

	logInfoWriter.Printf("*** writeExtensions end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePageLabels(ctx *types.PDFContext, rootDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// optional since PDF 1.3
	// => 7.9.7 Number Trees, 12.4.2 Page Labels

	// PDFDict or indirect ref to PDFDict
	// <Nums, [0 (170 0 R)]> or indirect ref

	logInfoWriter.Printf("*** writePageLabels begin: offset=%d ***\n", ctx.Write.Offset)

	indRef := rootDict.IndirectRefEntry("PageLabels")
	if indRef == nil {
		if required {
			err = errors.Errorf("writePageLabels: required entry \"PageLabels\" missing")
			return
		}
		logInfoWriter.Printf("writePageLabels end: indRef is nil.\n")
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writePageLabels: unsupported in version %s.\n", ctx.VersionString())
	}

	err = writeNumberTree(ctx, "PageLabel", *indRef, true)
	if err != nil {
		return
	}

	ctx.Stats.AddRootAttr(types.RootPageLabels)

	logInfoWriter.Printf("*** writePageLabels end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeNames(ctx *types.PDFContext, rootDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 7.7.4 Name Dictionary

	// all values are name trees or indirect refs.

	/*
		<Kids, [(86 0 R)]>

		86:
		<Limits, [(F1) (P.9)]>
		<Names, [(F1) (87 0 R) (F2) ...

		87: named destination dict
		<D, [(158 0 R) XYZ]>
	*/

	logInfoWriter.Printf("*** writeNames begin: offset=%d ***\n", ctx.Write.Offset)

	dict, written, err := writeDictEntry(ctx, rootDict, "rootDict", "Names", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeNames end: dict already written. offset=%d\n", ctx.Write.Offset)
		return
	}

	if dict == nil {
		logInfoWriter.Printf("writeNames end: dict is nil.\n")
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeNames: unsupported in version %s.\n", ctx.VersionString())
	}

	for treeName, value := range dict.Dict {

		if ok := validate.NameTreeName(treeName); !ok {
			return errors.Errorf("writeNames: unknown name tree name: %s\n", treeName)
		}

		indRef, ok := value.(types.PDFIndirectRef)
		if !ok {
			// TODO: All values are name trees or indirect refs.
			return errors.New("writeNames: name tree must be indirect ref")
		}

		logInfoWriter.Printf("writing Nametree: %s\n", treeName)
		err = writeNameTree(ctx, treeName, indRef, true)
		if err != nil {
			return
		}

	}

	ctx.Stats.AddRootAttr(types.RootNames)

	logInfoWriter.Printf("*** writeNames end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeNamedDestinations(ctx *types.PDFContext, rootDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 12.3.2.3 Named Destinations

	// indRef or dict with destination array values.

	logInfoWriter.Printf("*** writeNamedDestinations begin: offset=%d ***\n", ctx.Write.Offset)

	dict, written, err := writeDictEntry(ctx, rootDict, "rootDict", "Dests", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("*** writeNamedDestinations end: dict already written. offset=%d ***\n", ctx.Write.Offset)
		return
	}

	if dict == nil {
		logInfoWriter.Printf("writeNamedDestinations end: dict is nil.\n")
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeNamedDestinations: unsupported in version %s.\n", ctx.VersionString())
	}

	for _, value := range dict.Dict {
		err = writeDestination(ctx, value)
		if err != nil {
			return
		}
	}

	logInfoWriter.Printf("*** writeNamedDestinations end: offset=%d ***\n", ctx.Write.Offset)

	ctx.Stats.AddRootAttr(types.RootDests)

	return
}

func writeViewerPreferences(ctx *types.PDFContext, rootDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 12.2 Viewer Preferences

	logInfoWriter.Printf("*** writeViewerPreferences begin: offset=%d ***\n", ctx.Write.Offset)

	dict, written, err := writeDictEntry(ctx, rootDict, "rootDict", "ViewerPreferences", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeViewerPreferences end: object already written. offset=%d\n", ctx.Write.Offset)
		return
	}

	if dict == nil {
		logInfoWriter.Printf("writeViewerPreferences end: dict is nil.\n")
		return
	}

	_, _, err = writeBooleanEntry(ctx, *dict, "ViewerPreferences", "HideToolbar", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	_, _, err = writeBooleanEntry(ctx, *dict, "ViewerPreferences", "HideMenubar", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	_, _, err = writeBooleanEntry(ctx, *dict, "ViewerPreferences", "HideWindowUI", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	_, _, err = writeBooleanEntry(ctx, *dict, "ViewerPreferences", "FitWindow", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	_, _, err = writeBooleanEntry(ctx, *dict, "ViewerPreferences", "CenterWindow", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// TODO relaxed, no version check
	//writeBooleanEntry(dest, dest, dict, "ViewerPreferences", "DisplayDocTitle", OPTIONAL, V14, nil)
	_, _, err = writeBooleanEntry(ctx, *dict, "ViewerPreferences", "DisplayDocTitle", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	_, _, err = writeNameEntry(ctx, *dict, "ViewerPreferences", "NonFullScreenPageMode", OPTIONAL, types.V10, validate.ViewerPreferencesNonFullScreenPageMode)
	if err != nil {
		return
	}

	_, _, err = writeNameEntry(ctx, *dict, "ViewerPreferences", "Direction", OPTIONAL, types.V13, validate.ViewerPreferencesDirection)
	if err != nil {
		return
	}

	_, _, err = writeNameEntry(ctx, *dict, "ViewerPreferences", "ViewArea", OPTIONAL, types.V14, nil)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeViewerPreferences end: offset=%d ***\n", ctx.Write.Offset)

	ctx.Stats.AddRootAttr(types.RootViewerPrefs)

	return
}

func writePageLayout(ctx *types.PDFContext, rootDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writePageLayout begin: offset=%d ***\n", ctx.Write.Offset)

	pageLayout, written, err := writeNameEntry(ctx, rootDict, "rootDict", "PageLayout", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if !written && pageLayout != nil {
		ctx.Stats.AddRootAttr(types.RootPageLayout)
	}

	logInfoWriter.Printf("*** writePageLayout end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePageMode(ctx *types.PDFContext, rootDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writePageMode begin: offset=%d ***\n", ctx.Write.Offset)

	pageMode, written, err := writeNameEntry(ctx, rootDict, "rootDict", "PageMode", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if !written && pageMode != nil {
		ctx.Stats.AddRootAttr(types.RootPageMode)
	}

	logInfoWriter.Printf("*** writePageMode end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeBeadDict(ctx *types.PDFContext, indRefBeadDict, indRefThreadDict, indRefPreviousBead, indRefLastBead types.PDFIndirectRef) (err error) {

	objNumber := indRefBeadDict.ObjectNumber.Value()

	logInfoWriter.Printf("*** writeBeadDict begin: objectNumber=%d ***", objNumber)

	dictName := "beadDict"
	sinceVersion := types.V10

	dict, err := ctx.DereferenceDict(indRefBeadDict)
	if err != nil {
		return
	}

	if dict == nil {
		err = errors.Errorf("writeBeadDict: obj#%d missing dict", objNumber)
		return
	}

	// Write optional entry Type, must be "Bead".
	_, _, err = writeNameEntry(ctx, *dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Bead" })
	if err != nil {
		return
	}

	// Write entry T, must refer to threadDict.
	indRefT, _, err := writeIndRefEntry(ctx, *dict, dictName, "T", REQUIRED, sinceVersion)
	if err != nil {
		return
	}

	if !indRefT.Equals(indRefThreadDict) {
		err = errors.Errorf("writeBeadDict: obj#%d invalid entry T (backpointer to ThreadDict)", objNumber)
		return
	}

	// Write required entry R, must be rectangle.
	_, _, err = writeRectangleEntry(ctx, *dict, dictName, "R", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	// Write required entry P, must be indRef to pageDict.
	pageDict, _, err := writeDictEntry(ctx, *dict, dictName, "P", REQUIRED, sinceVersion, nil)
	if err != nil || pageDict == nil || pageDict.Type() == nil || *pageDict.Type() == "Page" {
		return
	}

	// Write required entry V, must refer to previous bead.
	previousBeadIndRef, _, err := writeIndRefEntry(ctx, *dict, dictName, "V", REQUIRED, sinceVersion)
	if err != nil {
		return
	}

	if !previousBeadIndRef.Equals(indRefPreviousBead) {
		err = errors.Errorf("writeBeadDict: obj#%d invalid entry V, corrupt previous Bead indirect reference", objNumber)
		return
	}

	// Write required entry N, must refer to last bead.
	nextBeadIndRef, _, err := writeIndRefEntry(ctx, *dict, dictName, "N", REQUIRED, sinceVersion)
	if err != nil {
		return
	}

	// Recurse until next bead equals last bead.
	if !nextBeadIndRef.Equals(indRefLastBead) {
		err = writeBeadDict(ctx, indRefBeadDict, indRefThreadDict, indRefBeadDict, indRefLastBead)
		if err != nil {
			return err
		}
	}

	logInfoWriter.Printf("*** validateBeadDict end: objectNumber=%d ***", objNumber)

	return
}

func writeFirstBeadDict(ctx *types.PDFContext, indRefBeadDict, indRefThreadDict types.PDFIndirectRef) (err error) {

	logInfoWriter.Printf("*** writeFirstBeadDict begin beadDictObj#%d threadDictObj#%d ***",
		indRefBeadDict.ObjectNumber.Value(), indRefThreadDict.ObjectNumber.Value())

	dictName := "firstBeadDict"
	sinceVersion := types.V10

	dict, err := ctx.DereferenceDict(indRefBeadDict)
	if err != nil {
		return
	}

	if dict == nil {
		err = errors.New("writeFirstBeadDict: missing dict")
		return
	}

	_, _, err = writeNameEntry(ctx, *dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Bead" })
	if err != nil {
		return
	}

	indRefT, _, err := writeIndRefEntry(ctx, *dict, dictName, "T", REQUIRED, sinceVersion)
	if err != nil {
		return
	}

	if !indRefT.Equals(indRefThreadDict) {
		err = errors.New("writeFirstBeadDict: invalid entry T (backpointer to ThreadDict)")
		return
	}

	_, _, err = writeRectangleEntry(ctx, *dict, dictName, "R", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	pageDict, written, err := writeDictEntry(ctx, *dict, dictName, "P", REQUIRED, sinceVersion, nil)
	if err != nil || !written && (pageDict == nil || pageDict.Type() == nil || *pageDict.Type() != "Page") {
		return errors.New("validateFirstBeadDict: invalid page dict")
	}

	previousBeadIndRef, _, err := writeIndRefEntry(ctx, *dict, dictName, "V", REQUIRED, sinceVersion)
	if err != nil {
		return
	}

	nextBeadIndRef, _, err := writeIndRefEntry(ctx, *dict, dictName, "N", REQUIRED, sinceVersion)
	if err != nil {
		return
	}

	// if N and V reference same bead dict, must be the first and only one.
	if previousBeadIndRef.Equals(*nextBeadIndRef) {
		if !indRefBeadDict.Equals(*previousBeadIndRef) {
			err = errors.New("writeFirstBeadDict: corrupt chain of beads")
			return
		}
		logInfoWriter.Println("*** writeFirstBeadDict end single bead ***")
		return
	}

	err = writeBeadDict(ctx, *nextBeadIndRef, indRefThreadDict, indRefBeadDict, *previousBeadIndRef)
	if err != nil {
		return
	}

	logInfoWriter.Println("*** writeFirstBeadDict end ***")

	return
}

func writeThreadDict(ctx *types.PDFContext, obj interface{}, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeThreadDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "threadDict"

	indRefThreadDict, ok := obj.(types.PDFIndirectRef)
	if !ok {
		err = errors.New("writeThreadDict: not an indirect ref")
		return
	}

	objNumber := indRefThreadDict.ObjectNumber.Value()

	dict, written, err := writeDict(ctx, obj)
	if err != nil || written || dict == nil {
		return
	}

	_, _, err = writeNameEntry(ctx, *dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Thread" })
	if err != nil {
		return
	}

	// Write optional thread information dict entry.
	obj, found := dict.Find("I")
	if found && obj != nil {
		_, err = writeDocumentInfoDict(ctx, obj, false)
		if err != nil {
			return
		}
	}

	firstBeadDict := dict.IndirectRefEntry("F")
	if firstBeadDict == nil {
		err = errors.Errorf("writeThreadDict: obj#%d required indirect entry \"F\" missing", objNumber)
		return
	}

	// Write the list of beads starting with the first bead dict.
	err = writeFirstBeadDict(ctx, *firstBeadDict, indRefThreadDict)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeThreadDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeThreads(ctx *types.PDFContext, rootDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 12.4.3 Articles

	logInfoWriter.Printf("*** writeThreads begin: offset=%d ***\n", ctx.Write.Offset)

	indRef := rootDict.IndirectRefEntry("Threads")
	if indRef == nil {
		if required {
			err = errors.Errorf("writeThreads: required entry \"Threads\" missing")
			return
		}
		logInfoWriter.Printf("writeThreads end: object is nil.\n")
		return
	}

	obj, written, err := writeIndRef(ctx, *indRef)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeThreads end: object already written. offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeThreads end: object is nil.\n")
		return
	}

	arr, ok := obj.(types.PDFArray)
	if !ok {
		return errors.New("writeThreads: corrupt threads dict")
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeThreads: unsupported in version %s.\n", ctx.VersionString())
	}

	for _, obj := range arr {

		if obj == nil {
			continue
		}

		err = writeThreadDict(ctx, obj, sinceVersion)
		if err != nil {
			return
		}

	}

	logInfoWriter.Printf("*** writeThreads end: offset=%d ***\n", ctx.Write.Offset)

	ctx.Stats.AddRootAttr(types.RootThreads)

	return
}

// TODO implement
func writeAA(ctx *types.PDFContext, obj interface{}) (written bool, err error) {

	// => 12.6.3 Trigger Events

	logInfoWriter.Printf("*** writeAA begin: offset=%d ***\n", ctx.Write.Offset)

	err = errors.New("*** writeAA: not supported ***")

	// dest.XRefTable.Stats.AddRootAttr(types.RootAA)

	logInfoWriter.Printf("*** writeAA end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeURI(ctx *types.PDFContext, rootDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 12.6.4.7 URI Actions

	//	<URI
	//		<Base, ()>
	//	>

	// Must be dict with one entry: Base.file

	logInfoWriter.Printf("*** writeURI begin: offset=%d ***\n", ctx.Write.Offset)

	dict, written, err := writeDictEntry(ctx, rootDict, "rootDict", "URI", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeURI end: dict already written. offset=%d\n", ctx.Write.Offset)
		return
	}

	if dict == nil {
		logInfoWriter.Printf("writeURI end: dict is nil.\n")
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeURI: unsupported in version %s.\n", ctx.VersionString())
	}

	// TODO optional entry?
	if dict.PDFStringLiteralEntry("Base") == nil {
		return errors.New("writeURI: corrupt URI dict: missing entry \"Base\"")
	}

	logInfoWriter.Printf("*** writeURI end: offset=%d ***\n", ctx.Write.Offset)

	ctx.Stats.AddRootAttr(types.RootURI)

	return
}

func writeMetadata(ctx *types.PDFContext, dict types.PDFDict, required bool, sinceVersion types.PDFVersion) (written bool, err error) {

	logInfoWriter.Printf("*** writeMetadata begin: offset=%d ***\n", ctx.Write.Offset)

	// => 14.3 Metadata
	// In general, any PDF stream or dictionary may have metadata attached to it
	// as long as the stream or dictionary represents an actual information resource,
	// as opposed to serving as an implementation artifact.
	// Some PDF constructs are considered implementational, and hence may not have associated metadata.

	streamDict, w, err := writeStreamDictEntry(ctx, dict, "dict", "Metadata", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if w {
		logInfoWriter.Printf("writeMetaData end: streamDict already written. offset=%d\n", ctx.Write.Offset)
		return
	}

	if streamDict == nil {
		logInfoWriter.Printf("writeMetaData end: streamDict is nil\n")
		return
	}

	// Version check
	// TODO Relaxed no version check
	//if dest.XRefTable.Version() < sinceVersion {
	//	log.Fatalf("writeMetadata: unsupported in version %s.\n", dest.XRefTable.VersionString())
	//}

	if t := streamDict.Type(); t != nil && *t != "Metadata" {
		err = errors.New("writeMetadata: corrupt metadata stream dict")
	}

	if subt := streamDict.Subtype(); subt != nil && *subt != "XML" {
		err = errors.New("writeMetadata: corrupt metadata stream dict")
	}

	logInfoWriter.Printf("*** writeMetadata end: offset=%d ***\n", ctx.Write.Offset)

	written = true

	return
}

func writeMarkInfo(ctx *types.PDFContext, rootDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 14.7 Logical Structure

	logInfoWriter.Printf("*** writeMarkInfo begin: offset=%d ***\n", ctx.Write.Offset)

	dict, written, err := writeDictEntry(ctx, rootDict, "rootDict", "MarkInfo", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeMarkInfo end: dict already written. offset=%d\n", ctx.Write.Offset)
		return
	}

	if dict == nil {
		logInfoWriter.Printf("writeMarkInfo end: dict is nil.\n")
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeMarkInfo: unsupported in version %s.\n", ctx.VersionString())
	}

	var isTaggedPDF bool

	// Marked, optional, boolean
	marked, _, err := writeBooleanEntry(ctx, *dict, "markInfoDict", "Marked", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if marked != nil {
		isTaggedPDF = (*marked).Value()
	}

	// Suspects: optional, since V1.6, boolean
	suspects, _, err := writeBooleanEntry(ctx, *dict, "markInfoDict", "Suspects", OPTIONAL, types.V16, nil)
	if err != nil {
		return
	}

	if suspects != nil && (*suspects).Value() {
		isTaggedPDF = false
	}

	ctx.Tagged = isTaggedPDF

	// UserProperties: optional, since V1.6, boolean
	_, _, err = writeBooleanEntry(ctx, *dict, "markInfoDict", "UserProperties", OPTIONAL, types.V16, nil)
	if err != nil {
		return
	}

	ctx.Stats.AddRootAttr(types.RootMarkInfo)

	logInfoWriter.Printf("*** writeMarkInfo end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeLang(ctx *types.PDFContext, rootDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeLang begin: offset=%d ***\n", ctx.Write.Offset)

	lang, written, err := writeStringEntry(ctx, rootDict, "rootDict", "Lang", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if !written && lang != nil {
		ctx.Stats.AddRootAttr(types.RootLang)
	}

	logInfoWriter.Printf("*** writeLang end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeSpiderInfo(ctx *types.PDFContext, rootDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// 14.10.2 Web Capture Information Dictionary

	logInfoWriter.Printf("*** writeSpiderInfo begin: offset=%d ***\n", ctx.Write.Offset)

	dict, written, err := writeDictEntry(ctx, rootDict, "rootDict", "SpiderInfo", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeSpiderInfo end: dict already written. offset=%declared\n", ctx.Write.Offset)
		return
	}

	if dict == nil {
		logInfoWriter.Printf("writeSpiderInfo end: dict is nil.\n")
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeSpiderInfo: unsupported in version %s.\n", ctx.VersionString())
	}

	err = errors.New("*** writeSpiderInfo: not supported ***")

	// dest.XRefTable.Stats.AddRootAttr(types.RootSpiderInfo)

	logInfoWriter.Printf("*** writeSpiderInfo end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeOutputIntentDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeOutputIntentDict begin: offset=%d ***\n", ctx.Write.Offset)

	if t := dict.Type(); t != nil && *t != "OutputIntent" {
		return errors.New("writeOutputIntentDict: outputIntents corrupted Type")
	}

	// S: required, name
	writeNameEntry(ctx, dict, "outputIntentDict", "S", REQUIRED, types.V10, nil)

	// OutputCondition, optional, text string
	_, _, err = writeStringEntry(ctx, dict, "outputIntentDict", "OutputCondition", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// OutputConditionIdentifier, required, text string
	_, _, err = writeStringEntry(ctx, dict, "outputIntentDict", "OutputConditionIdentifier", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// RegistryName, optional, text string
	_, _, err = writeStringEntry(ctx, dict, "outputIntentDict", "RegistryName", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Info, text string
	// TODO Required if OutputConditionIdentifier does not specify a standard production condition; optional otherwise
	_, _, err = writeStringEntry(ctx, dict, "outputIntentDict", "Info", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// DestOutputProfile, streamDict
	// TODO Required if OutputConditionIdentifier does not specify a standard production condition; optional otherwise
	_, _, err = writeStreamDictEntry(ctx, dict, "outputIntentDict", "DestOutputProfile", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeOutputIntentDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeOutputIntents(ctx *types.PDFContext, rootDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 14.11.5 Output Intents

	logInfoWriter.Printf("*** writeOutputIntents begin: offset=%d ***\n", ctx.Write.Offset)

	arr, written, err := writeArrayEntry(ctx, rootDict, "rootDict", "OutputIntents", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeOutputIntents end: already written.\n")
		return
	}

	if arr == nil {
		logInfoWriter.Printf("writeOutputIntents end: array is nil.\n")
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeOutputIntents: unsupported in version %s.\n", ctx.VersionString())
	}

	for _, value := range *arr {

		o, written, err := writeObject(ctx, value)
		if err != nil {
			return err
		}

		if written || o == nil {
			continue
		}

		dict, ok := o.(types.PDFDict)
		if !ok {
			return errors.New("writeOutputIntents: corrupt outputIntentDict")
		}

		err = writeOutputIntentDict(ctx, dict)
		if err != nil {
			return err
		}
	}

	ctx.Stats.AddRootAttr(types.RootOutputIntents)

	logInfoWriter.Printf("*** writeOutputIntents end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePieceDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writePieceDict begin: offset=%d ***\n", ctx.Write.Offset)

	var written bool

	for _, obj := range dict.Dict {

		obj, written, err = writeObject(ctx, obj)
		if err != nil {
			return
		}

		if written {
			logInfoWriter.Printf("writePieceDict: object already written.\n")
			continue
		}

		if obj == nil {
			logInfoWriter.Printf("writePieceDict: object is nil.\n")
			continue
		}

		dict, ok := obj.(types.PDFDict)
		if !ok {
			err = errors.New("writePieceDict: corrupt dict")
			return
		}

		_, _, err = writeDateEntry(ctx, dict, "PagePieceDataDict", "LastModified", REQUIRED, types.V10)
		if err != nil {
			return err
		}

		_, err = writeAnyEntry(ctx, dict, "Private", OPTIONAL)
		if err != nil {
			return err
		}

	}

	logInfoWriter.Printf("*** writePieceDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePieceInfo(ctx *types.PDFContext, dict types.PDFDict, required bool, sinceVersion types.PDFVersion) (hasPieceInfo bool, err error) {

	// 14.5 Page-Piece Dictionaries

	logInfoWriter.Printf("*** writePieceInfo begin: offset=%d ***\n", ctx.Write.Offset)

	pieceDict, written, err := writeDictEntry(ctx, dict, "dict", "PieceInfo", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writePieceInfo end: pieceDict already written.\n")
		return
	}

	if pieceDict == nil {
		logInfoWriter.Printf("writePieceInfo end: pieceDict is nil.\n")
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		err = errors.Errorf("writePieceInfo: unsupported in version %s.\n", ctx.VersionString())
		return
	}

	err = writePieceDict(ctx, *pieceDict)
	if err != nil {
		return
	}

	hasPieceInfo = true

	logInfoWriter.Printf("*** writePieceInfo end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeOptionalContentGroupArray(ctx *types.PDFContext, dict types.PDFDict, dictName string, dictEntry string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeOptionalContentGroupArray begin: offset=%d ***\n", ctx.Write.Offset)

	logInfoWriter.Printf("*** writeOptionalContentGroupArray end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeOptContentConfigDictIntentEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, dictEntry string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeOptContentConfigDictIntentEntry begin: offset=%d ***\n", ctx.Write.Offset)

	obj, found := dict.Find(dictEntry)
	if !found || obj == nil {
		return
	}

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written || obj == nil {
		return
	}

	switch obj := obj.(type) {

	case types.PDFName:
		if !validate.OptContentConfigDictIntent(obj.String()) {
			err = errors.Errorf("writeOptContentConfigDictIntentEntry: invalid entry")
		}

	case types.PDFArray:
		for _, obj := range obj {
			_, _, err := writeName(ctx, obj, validate.OptContentConfigDictIntent)
			if err != nil {
				return err
			}
		}

	default:
		err = errors.Errorf("writeOptContentConfigDictIntentEntry: must be stream dict or array")
		return
	}

	logInfoWriter.Printf("*** writeOptContentConfigDictIntentEntry end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeUsageApplicationDictArray(ctx *types.PDFContext, dict types.PDFDict, dictName string, dictEntry string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeUsageApplicationDictArray begin: offset=%d ***\n", ctx.Write.Offset)

	logInfoWriter.Printf("*** writeUsageApplicationDictArray end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeOptContentConfigDictOrderEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, dictEntry string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeOptContentConfigDictOrderEntry begin: offset=%d ***\n", ctx.Write.Offset)

	logInfoWriter.Printf("*** writeOptContentConfigDictOrderEntry end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeRBGroupsEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, dictEntry string, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeRBGroupsEntry begin: offset=%d ***\n", ctx.Write.Offset)

	logInfoWriter.Printf("*** writeRBGroupsEntry end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeOptionalContentConfigurationDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeOptionalContentConfigurationDict begin: offset=%d ***\n", ctx.Write.Offset)

	dictName := "optContentConfigDict"

	_, _, err = writeStringEntry(ctx, dict, dictName, "Name", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	_, _, err = writeStringEntry(ctx, dict, dictName, "Creator", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	baseState, _, err := writeNameEntry(ctx, dict, dictName, "BaseState", OPTIONAL, types.V10, validate.BaseState)
	if err != nil {
		return
	}

	if baseState != nil {

		if baseState.String() != "ON" {
			err = writeOptionalContentGroupArray(ctx, dict, dictName, "ON", OPTIONAL, types.V10)
			if err != nil {
				return err
			}
		}

		if baseState.String() != "OFF" {
			err = writeOptionalContentGroupArray(ctx, dict, dictName, "OFF", OPTIONAL, types.V10)
			if err != nil {
				return err
			}
		}

	}

	err = writeOptContentConfigDictIntentEntry(ctx, dict, dictName, "Intent", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	err = writeUsageApplicationDictArray(ctx, dict, dictName, "AS", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	err = writeOptContentConfigDictOrderEntry(ctx, dict, dictName, "Order", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	_, _, err = writeNameEntry(ctx, dict, dictName, "ListMode", OPTIONAL, types.V10, validate.ListMode)
	if err != nil {
		return
	}

	err = writeRBGroupsEntry(ctx, dict, dictName, "RBGroups", OPTIONAL, types.V10)
	if err != nil {
		return err
	}

	err = writeOptionalContentGroupArray(ctx, dict, dictName, "Locked", OPTIONAL, types.V16)
	if err != nil {
		return err
	}

	logInfoWriter.Printf("*** writeOptionalContentConfigurationDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeOCProperties(ctx *types.PDFContext, rootDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// aka optional content properties dict.

	// => 8.11.4 Configuring Optional Content

	logInfoWriter.Printf("*** writeOCProperties begin: offset=%d ***\n", ctx.Write.Offset)

	dict, written, err := writeDictEntry(ctx, rootDict, "rootDict", "OCProperties", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeOCProperties end: object already written. offset=%d\n", ctx.Write.Offset)
		return
	}

	if dict == nil {
		logInfoWriter.Printf("writeOCProperties end: object is nil.\n")
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeOCProperties: unsupported in version %s.\n", ctx.VersionString())
	}

	// "OCGs" required array of already written indRefs
	_, _, err = writeIndRefArrayEntry(ctx, *dict, "optContPropsDict", "OCGs", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// "D" required dict, default viewing optional content configuration dict.
	d, written, err := writeDictEntry(ctx, *dict, "optContPropsDict", "D", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}
	if !written {
		err = writeOptionalContentConfigurationDict(ctx, *d)
		if err != nil {
			return
		}
	}

	// "Configs" optional array of alternate optional content configuration dicts.
	arr, written, err := writeArrayEntry(ctx, *dict, "optContPropsDict", "Configs", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if arr != nil && !written {
		for _, value := range *arr {
			d, _, err = writeDict(ctx, value)
			if err != nil {
				return
			}
			err = writeOptionalContentConfigurationDict(ctx, *d)
			if err != nil {
				return
			}
		}
	}

	ctx.Stats.AddRootAttr(types.RootOCProperties)

	logInfoWriter.Printf("*** writeOCProperties end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writePermissions(ctx *types.PDFContext, rootDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 12.8.4 Permissions

	logInfoWriter.Printf("*** writePermissions begin: offset=%d ***\n", ctx.Write.Offset)

	dict, written, err := writeDictEntry(ctx, rootDict, "rootDict", "Permissions", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writePermissions end: dict already written. offset=%d ***\n", ctx.Write.Offset)
		return
	}

	if dict == nil {
		logInfoWriter.Printf("writePermissions end: dict is nil.\n")
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writePermissions: unsupported in version %s.\n", ctx.VersionString())
	}

	err = errors.New("*** writePermissions: not supported ***")

	// dest.XRefTable.Stats.AddRootAttr(types.RootPerms)

	logInfoWriter.Printf("*** writePermissions end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeLegal(ctx *types.PDFContext, rootDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 12.8.5 Legal Content Attestations

	logInfoWriter.Printf("*** writeLegal begin: offset=%d ***\n", ctx.Write.Offset)

	dict, written, err := writeDictEntry(ctx, rootDict, "rootDict", "Legal", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeLegal end: dict already written. offset=%d\n", ctx.Write.Offset)
		return
	}

	if dict == nil {
		logInfoWriter.Printf("writeLegal end: dict is nil.\n")
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeLegal: unsupported in version %s.\n", ctx.VersionString())
	}

	err = errors.New("***writeLegal: not supported ***")

	// ctx.Stats.AddRootAttr(types.RootLegal)

	logInfoWriter.Printf("*** writeLegal end: offset=%d ***\n", ctx.Write.Offset)
	return
}

// TODO implement
func writeRequirements(ctx *types.PDFContext, rootDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 12.10 Document Requirements

	logInfoWriter.Printf("*** writeRequirements begin: offset=%d ***\n", ctx.Write.Offset)

	arr, written, err := writeArrayEntry(ctx, rootDict, "rootDict", "Requirements", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeRequirements end: array already written. offset=%d\n", ctx.Write.Offset)
		return
	}

	if arr == nil {
		logInfoWriter.Printf("writeRequirements end: array is nil.\n")
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeRequirements: unsupported in version %s.\n", ctx.VersionString())
	}

	err = errors.New("*** writeRequirements: not supported ***")

	// dest.XRefTable.Stats.AddRootAttr(types.RootRequirements)

	logInfoWriter.Printf("*** writeRequirements end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeCollection(ctx *types.PDFContext, rootDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 12.3.5 Collections

	logInfoWriter.Printf("*** writeCollection begin: offset=%d ***\n", ctx.Write.Offset)

	dict, written, err := writeDictEntry(ctx, rootDict, "rootDict", "Collection", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeCollection end: dict already written. offset=%d\n", ctx.Write.Offset)
		return
	}

	if dict == nil {
		logInfoWriter.Printf("writeCollection end: dict is nil.\n")
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeCollection: unsupported in version %s.\n", ctx.VersionString())
	}

	err = errors.New("*** writeCollection: not supported ***")

	// dest.XRefTable.Stats.AddRootAttr(types.RootCollection)

	logInfoWriter.Printf("*** writeCollection end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeNeedsRendering(ctx *types.PDFContext, rootDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeNeedsRendering begin: offset=%d ***\n", ctx.Write.Offset)

	needsRendering, written, err := writeBooleanEntry(ctx, rootDict, "rootDict", "NeedsRendering", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if !written && needsRendering != nil {
		ctx.Stats.AddRootAttr(types.RootNeedsRendering)
	}

	logInfoWriter.Printf("*** writeNeedsRendering end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeRootObject(ctx *types.PDFContext) (err error) {

	// => 7.7.2 Document Catalog

	// Entry	   	       opt	since		type			info
	//------------------------------------------------------------------------------------
	//*Type			        n				string			"Catalog"
	//*Version		        y	1.4			name			overrules header version if later
	// Extensions	        y	ISO 32000	dict			=> 7.12 Extensions Dictionary
	//*Pages		        n	-			(dict)			=> 7.7.3 Page Tree
	//*PageLabels	        y	1.3			number tree		=> 7.9.7 Number Trees, 12.4.2 Page Labels
	//*Names		        y	1.2			dict			=> 7.7.4 Name Dictionary
	// Dests	    	    y	only 1.1	(dict)			=> 12.3.2.3 Named Destinations
	//*ViewerPreferences    y	1.2			dict			=> 12.2 Viewer Preferences
	// PageLayout	        y	-			name			/SinglePage, /OneColumn etc.
	// PageMode		        y	-			name			/UseNone, /FullScreen etc.
	//*Outlines		        y	-			(dict)			=> 12.3.3 Document Outline
	// Threads		        y	1.1			(array)			=> 12.4.3 Articles
	//*OpenAction	        y	1.1			array or dict	=> 12.3.2 Destinations, 12.6 Actions
	// AA			        y	1.4			dict			=> 12.6.3 Trigger Events
	//*URI			        y	1.1			dict			=> 12.6.4.7 URI Actions
	//*AcroForm		        y	1.2			dict			=> 12.7.2 Interactive Form Dictionary
	//*Metadata		        y	1.4			(stream)		=> 14.3.2 Metadata Streams
	//*StructTreeRoot 	    y 	1.3			dict			=> 14.7.2 Structure Hierarchy
	//*Markinfo		        y	1.4			dict			=> 14.7 Logical Structure
	// Lang			        y	1.4			string
	// SpiderInfo	        y	1.3			dict			=> 14.10.2 Web Capture Information Dictionary
	//*OutputIntents 	    y	1.4			array			=> 14.11.5 Output Intents
	// PieceInfo	        y	1.4			dict			=> 14.5 Page-Piece Dictionaries
	// OCProperties	        y	1.5			dict			=> 8.11.4 Configuring Optional Content
	// Perms		        y	1.5			dict			=> 12.8.4 Permissions
	// Legal		        y	1.5			dict			=> 12.8.5 Legal Content Attestations
	// Requirements	        y	1.7			array			=> 12.10 Document Requirements
	// Collection	        y	1.7			dict			=> 12.3.5 Collections
	// NeedsRendering 	    y	1.7			boolean			=> XML Forms Architecture (XFA) Spec.

	xRefTable := ctx.XRefTable

	catalog := *xRefTable.Root
	objNumber := int(catalog.ObjectNumber)
	genNumber := int(catalog.GenerationNumber)

	logPages.Printf("*** writeRootObject: begin offset=%d *** %s\n", ctx.Write.Offset, catalog)

	obj, err := xRefTable.Dereference(catalog)
	if err != nil || obj == nil {
		err = errors.Errorf("writeRootObject: unable to dereference root dict")
		return
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		return errors.Errorf("writeRootObject: corrupt root dict")
	}

	if ctx.Write.ReducedFeatureSet() {
		logDebugWriter.Println("writeRootObject: exclude complex entries on split,trim and page extraction.")
		dict.Delete("Names")
		dict.Delete("Dests")
		dict.Delete("Outlines")
		dict.Delete("OpenAction")
		dict.Delete("AcroForm")
		dict.Delete("StructTreeRoot")
		dict.Delete("OCProperties")
	}

	err = writePDFDictObject(ctx, objNumber, genNumber, dict)
	if err != nil {
		return
	}

	logPages.Printf("writeRootObject: %s\n", dict)

	logDebugWriter.Printf("writeRootObject: new offset after rootDict = %d\n", ctx.Write.Offset)

	err = writeVersion(ctx, &dict, OPTIONAL, types.V14)
	if err != nil {
		return
	}

	// Embedd all page tree objects into objects stream.
	ctx.Write.WriteToObjectStream = true

	pagesIndRef, err := writePages(ctx, &dict)
	if err != nil {
		return
	}

	err = stopObjectStream(ctx)
	if err != nil {
		return
	}

	// TODO implement, since ISO 32000
	err = writeExtensions(ctx, dict, OPTIONAL, types.V10)
	if err != nil {
		return
	}

	err = writePageLabels(ctx, dict, OPTIONAL, types.V13)
	if err != nil {
		return
	}

	err = writeNames(ctx, dict, OPTIONAL, types.V12)
	if err != nil {
		return
	}

	err = writeNamedDestinations(ctx, dict, OPTIONAL, types.V11)
	if err != nil {
		return
	}

	err = writeViewerPreferences(ctx, dict, OPTIONAL, types.V12)
	if err != nil {
		return
	}

	err = writePageLayout(ctx, dict, OPTIONAL, types.V10)
	if err != nil {
		return
	}

	err = writePageMode(ctx, dict, OPTIONAL, types.V10)
	if err != nil {
		return
	}

	err = writeOutlines(ctx, dict, OPTIONAL, types.V10)
	if err != nil {
		return
	}

	err = writeThreads(ctx, dict, OPTIONAL, types.V11)
	if err != nil {
		return
	}

	err = writeOpenAction(ctx, dict, OPTIONAL, types.V11)
	if err != nil {
		return
	}

	if obj, found := dict.Find("AA"); found {
		// TODO implement
		written, err := writeAA(ctx, obj)
		if err != nil {
			return err
		}
		if written {
			ctx.Stats.AddRootAttr(types.RootAA)
		}
	}

	err = writeURI(ctx, dict, OPTIONAL, types.V11)
	if err != nil {
		return err
	}

	// Write widget annotations as part of given form field hierarchies.
	err = writeAcroForm(ctx, dict, OPTIONAL, types.V12)
	if err != nil {
		return err
	}

	if !ctx.Write.ReducedFeatureSet() {

		// Write remainder of annotations after AcroForm processing only.
		written, err := writePagesAnnotations(ctx, *pagesIndRef)
		if err != nil {
			return err
		}

		if written {
			ctx.Stats.AddPageAttr(types.PageAnnots)
		}

	} else {
		logDebugWriter.Printf("writeRootObject: exclude PageAnnotations: len=%d extractPage=%d\n", len(ctx.Write.ExtractPages), ctx.Write.ExtractPageNr)
	}

	// Relaxed to V1.3
	written, err := writeMetadata(ctx, dict, OPTIONAL, types.V13)
	if err != nil {
		return err
	}
	if written {
		ctx.Stats.AddRootAttr(types.RootMetadata)
	}

	// Embedd all struct tree objects into objects stream.
	ctx.Write.WriteToObjectStream = true

	err = writeStructTree(ctx, dict, OPTIONAL, types.V13)
	if err != nil {
		return
	}

	err = stopObjectStream(ctx)
	if err != nil {
		return
	}

	err = writeMarkInfo(ctx, dict, OPTIONAL, types.V14)
	if err != nil {
		return
	}

	err = writeLang(ctx, dict, OPTIONAL, types.V10)
	if err != nil {
		return
	}

	err = writeSpiderInfo(ctx, dict, OPTIONAL, types.V13)
	if err != nil {
		return err
	}

	err = writeOutputIntents(ctx, dict, OPTIONAL, types.V14)
	if err != nil {
		return err
	}

	hasPieceInfo, err := writePieceInfo(ctx, dict, OPTIONAL, types.V14)
	if err != nil {
		return err
	}
	if hasPieceInfo {
		ctx.Stats.AddRootAttr(types.RootPieceInfo)
		// requires InfoDict.LastModified
	}

	// TODO Required if document contains optional content.
	// Relaxed from 1.5 to V1.4
	err = writeOCProperties(ctx, dict, OPTIONAL, types.V14)
	if err != nil {
		return
	}

	err = writePermissions(ctx, dict, OPTIONAL, types.V15)
	if err != nil {
		return
	}

	err = writeLegal(ctx, dict, OPTIONAL, types.V17)
	if err != nil {
		return
	}

	err = writeRequirements(ctx, dict, OPTIONAL, types.V17)
	if err != nil {
		return
	}

	err = writeCollection(ctx, dict, OPTIONAL, types.V17)
	if err != nil {
		return
	}

	err = writeNeedsRendering(ctx, dict, OPTIONAL, types.V17)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeRootObject: end offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeAdditionalStreams(ctx *types.PDFContext) (err error) {

	logInfoWriter.Printf("writeAdditionalStreams begin: offset=%d\n", ctx.Write.Offset)

	if len(ctx.AdditionalStreams) == 0 {
		logInfoWriter.Printf("writeAdditionalStreams end: no additional streams\n")
		return nil
	}

	for _, indRef := range ctx.AdditionalStreams {

		obj, written, err := writeIndRef(ctx, indRef)
		if err != nil {
			return err
		}

		if written || obj == nil {
			continue
		}

	}

	logInfoWriter.Printf("writeAdditionalStreams end: offset=%d\n", ctx.Write.Offset)

	return
}

func writeTrailerDict(ctx *types.PDFContext) (err error) {

	logInfoWriter.Printf("writeTrailerDict begin\n")

	w := ctx.Write
	xRefTable := ctx.XRefTable

	_, err = w.WriteString("trailer")
	if err != nil {
		return
	}

	_, err = w.WriteString(eol)
	if err != nil {
		return
	}

	dict := types.NewPDFDict()
	dict.Insert("Size", types.PDFInteger(*xRefTable.Size))
	dict.Insert("Root", *xRefTable.Root)

	if xRefTable.Info != nil {
		dict.Insert("Info", *xRefTable.Info)
	}

	if xRefTable.ID != nil {
		dict.Insert("ID", *xRefTable.ID)
	}

	_, err = w.WriteString(dict.PDFString())
	if err != nil {
		return
	}

	logInfoWriter.Printf("writeTrailerDict end\n")

	return
}

func writeXRefSubsection(ctx *types.PDFContext, start int, size int) (err error) {

	logXRef.Printf("writeXRefSubsection: start=%d size=%d\n", start, size)

	w := ctx.Write

	_, err = w.WriteString(fmt.Sprintf("%d %d%s", start, size, eol))
	if err != nil {
		return
	}

	var lines []string

	for i := start; i < start+size; i++ {

		entry := ctx.XRefTable.Table[i]

		if entry.Compressed {
			return errors.New("writeXRefSubsection: compressed entries present")
		}

		var s string

		if entry.Free {
			s = fmt.Sprintf("%010d %05d f%2s", *entry.Offset, *entry.Generation, eol)
		} else {
			var off int64
			writeOffset, found := ctx.Write.Table[i]
			if found {
				off = writeOffset
			}
			s = fmt.Sprintf("%010d %05d n%2s", off, *entry.Generation, eol)
		}

		lines = append(lines, fmt.Sprintf("%d: %s", i, s))

		_, err = w.WriteString(s)
		if err != nil {
			return
		}
	}

	logXRef.Printf("\n%s\n", strings.Join(lines, ""))

	logXRef.Printf("writeXRefSubsection: end\n")

	return
}

func deleteRedundantObjects(ctx *types.PDFContext) (err error) {

	xRefTable := ctx.XRefTable
	logInfoWriter.Printf("deleteRedundantObjects begin: Size=%d\n", *xRefTable.Size)

	for i := 0; i < *xRefTable.Size; i++ {

		entry, found := xRefTable.Find(i)
		if !found {
			// missing object remains missing.
			continue
		}

		if entry.Free {
			continue
		}

		// object written to dest

		if ctx.Write.HasWriteOffset(i) {
			// Resources may be cross referenced from different directions.
			// eg. font descriptors may be shared by different font dicts.
			// Try to remove this object from the list of the potential duplicate objects.
			logDebugWriter.Printf("deleteRedundantObjects: remove duplicate obj #%d\n", i)
			delete(ctx.Optimize.DuplicateFontObjs, i)
			delete(ctx.Optimize.DuplicateImageObjs, i)
			delete(ctx.Optimize.DuplicateInfoObjects, i)
			continue
		}

		// object not written to dest

		if ctx.Read.Linearized {

			// Since there is no type entry for stream dicts associated with linearization dicts
			// we have to check every PDFStreamDict that has not been written.
			if _, ok := entry.Object.(types.PDFStreamDict); ok {

				if *entry.Offset == *xRefTable.OffsetPrimaryHintTable {
					xRefTable.LinearizationObjs[i] = true
					logDebugWriter.Printf("deleteRedundantObjects: primaryHintTable at obj #%d\n", i)
				}

				if xRefTable.OffsetOverflowHintTable != nil &&
					*entry.Offset == *xRefTable.OffsetOverflowHintTable {
					xRefTable.LinearizationObjs[i] = true
					logDebugWriter.Printf("deleteRedundantObjects: overflowHintTable at obj #%d\n", i)
				}

			}
		}

		if ctx.Write.ExtractPageNr == 0 &&
			(ctx.Optimize.IsDuplicateFontObject(i) || ctx.Optimize.IsDuplicateImageObject(i)) {
			xRefTable.DeleteObject(i)
		}

		if xRefTable.IsLinearizationObject(i) || ctx.Optimize.IsDuplicateInfoObject(i) ||
			ctx.Read.IsObjectStreamObject(i) || ctx.Read.IsXRefStreamObject(i) {
			xRefTable.DeleteObject(i)
		}

	}

	logInfoWriter.Println("deleteRedundantObjects end")

	return
}

func writeXRefTable(ctx *types.PDFContext) (err error) {

	// After the last insert of an object.
	err = ctx.EnsureValidFreeList()
	if err != nil {
		return
	}

	xRefTable := ctx.XRefTable

	var keys []int
	for i, e := range xRefTable.Table {
		if e.Free || ctx.Write.HasWriteOffset(i) {
			keys = append(keys, i)
		}
	}
	sort.Ints(keys)

	objCount := len(keys)
	logXRef.Printf("xref has %d entries\n", objCount)

	_, err = ctx.Write.WriteString("xref")
	if err != nil {
		return
	}

	_, err = ctx.Write.WriteString(eol)
	if err != nil {
		return
	}

	start := keys[0]
	size := 1

	for i := 1; i < len(keys); i++ {

		if keys[i]-keys[i-1] > 1 {

			err = writeXRefSubsection(ctx, start, size)
			if err != nil {
				return
			}

			start = keys[i]
			size = 1
			continue
		}

		size++
	}

	err = writeXRefSubsection(ctx, start, size)
	if err != nil {
		return
	}

	err = writeTrailerDict(ctx)
	if err != nil {
		return
	}

	_, err = ctx.Write.WriteString(eol)
	if err != nil {
		return
	}

	_, err = ctx.Write.WriteString("startxref")
	if err != nil {
		return
	}

	_, err = ctx.Write.WriteString(eol)
	if err != nil {
		return
	}

	_, err = ctx.Write.WriteString(fmt.Sprintf("%d", ctx.Write.Offset))
	if err != nil {
		return
	}

	_, err = ctx.Write.WriteString(eol)
	if err != nil {
		return
	}

	return
}

func startObjectStream(ctx *types.PDFContext) (err error) {

	// See 7.5.7 Object streams
	// When new object streams and compressed objects are created, they shall always be assigned new object numbers,
	// not old ones taken from the free list.

	logInfoWriter.Println("startObjectStream begin")

	xRefTable := ctx.XRefTable
	objStreamDict := types.NewPDFObjectStreamDict()
	xRefTableEntry := types.NewXRefTableEntryGen0()
	xRefTableEntry.Object = *objStreamDict

	objNumber, ok := xRefTable.InsertNew(*xRefTableEntry)
	if !ok {
		return errors.Errorf("startObjectStream: Problem inserting entry for %d", objNumber)
	}

	ctx.Write.CurrentObjStream = &objNumber

	logInfoWriter.Println("startObjectStream end")

	return
}

func stopObjectStream(ctx *types.PDFContext) (err error) {

	logInfoWriter.Println("stopObjectStream begin")

	xRefTable := ctx.XRefTable

	if !ctx.Write.WriteToObjectStream {
		err = errors.Errorf("stopObjectStream: Not writing to object stream.")
		return
	}

	if ctx.Write.CurrentObjStream == nil {
		ctx.Write.WriteToObjectStream = false
		logInfoWriter.Println("stopObjectStream end (no content)")
		return
	}

	entry, _ := xRefTable.FindTableEntry(*ctx.Write.CurrentObjStream, 0)
	objStreamDict, _ := (entry.Object).(types.PDFObjectStreamDict)
	//logDebugWriter.Printf("stopObjectStream objStreamDict:\n%v", objStreamDict)

	// When we are ready to write: append prolog and content
	objStreamDict.Finalize()
	//logDebugWriter.Printf("stopObjectStream Content:\n%s", hex.Dump(objStreamDict.Content))

	// Encode objStreamDict.Content -> objStreamDict.Raw
	// and wipe (decoded) content to free up memory.
	err = filter.EncodeStream(&objStreamDict.PDFStreamDict)
	if err != nil {
		return
	}

	// Release memory.
	objStreamDict.Content = nil

	objStreamDict.PDFStreamDict.Insert("First", types.PDFInteger(objStreamDict.FirstObjOffset))
	objStreamDict.PDFStreamDict.Insert("N", types.PDFInteger(objStreamDict.ObjCount))

	// for each objStream execute at the end right before xRefStreamDict gets written.
	logInfoWriter.Printf("stopObjectStream: objStreamDict: %s\n", objStreamDict)

	err = writePDFStreamDictObject(ctx, *ctx.Write.CurrentObjStream, 0, objStreamDict.PDFStreamDict)
	if err != nil {
		return
	}

	// Release memory.
	objStreamDict.Raw = nil

	ctx.Write.CurrentObjStream = nil
	ctx.Write.WriteToObjectStream = false

	logInfoWriter.Println("stopObjectStream end")

	return
}

func int64ByteCount(i int64) (byteCount int) {

	for i > 0 {
		i >>= 8
		byteCount++
	}

	return
}

func int64ToBuf(i int64, byteCount int) (buf []byte) {

	j := 0
	var b []byte

	for k := i; k > 0; {
		b = append(b, byte(k&0xff))
		k >>= 8
		j++

	}

	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}

	if j < byteCount {
		buf = append(bytes.Repeat([]byte{0}, byteCount-j), b...)
	} else {
		buf = b
	}

	return
}

func createXRefStream(ctx *types.PDFContext, i1, i2, i3 int) (buf []byte, indArr types.PDFArray, err error) {

	logDebugWriter.Println("createXRefStream begin")

	xRefTable := ctx.XRefTable

	var keys []int
	for i, e := range xRefTable.Table {
		if e.Free || ctx.Write.HasWriteOffset(i) {
			keys = append(keys, i)
		}
	}
	sort.Ints(keys)

	objCount := len(keys)
	logDebugWriter.Printf("createXRefStream: xref has %d entries\n", objCount)

	start := keys[0]
	size := 0

	for i := 0; i < len(keys); i++ {

		j := keys[i]
		entry := xRefTable.Table[j]
		var s1, s2, s3 []byte

		if entry.Free {

			// unused
			logDebugWriter.Printf("createXRefStream: unused i=%d nextFreeAt:%d gen:%d\n", j, int(*entry.Offset), int(*entry.Generation))

			s1 = int64ToBuf(0, i1)
			s2 = int64ToBuf(*entry.Offset, i2)
			s3 = int64ToBuf(int64(*entry.Generation), i3)

		} else if entry.Compressed {

			// in use, compressed into object stream
			logDebugWriter.Printf("createXRefStream: compressed i=%d at objstr %d[%d]\n", j, int(*entry.ObjectStream), int(*entry.ObjectStreamInd))

			s1 = int64ToBuf(2, i1)
			s2 = int64ToBuf(int64(*entry.ObjectStream), i2)
			s3 = int64ToBuf(int64(*entry.ObjectStreamInd), i3)

		} else {

			off, found := ctx.Write.Table[j]
			if !found {
				err = errors.Errorf("createXRefStream: missing write offset for obj #%d\n", i)
				return
			}

			// in use, uncompressed
			logDebugWriter.Printf("createXRefStream: used i=%d offset:%d gen:%d\n", j, int(off), int(*entry.Generation))

			s1 = int64ToBuf(1, i1)
			s2 = int64ToBuf(off, i2)
			s3 = int64ToBuf(int64(*entry.Generation), i3)

		}

		logDebugWriter.Printf("createXRefStream: written: %x %x %x \n", s1, s2, s3)

		buf = append(buf, s1...)
		buf = append(buf, s2...)
		buf = append(buf, s3...)

		if i > 0 && (keys[i]-keys[i-1] > 1) {

			indArr = append(indArr, types.PDFInteger(start))
			indArr = append(indArr, types.PDFInteger(size))

			start = keys[i]
			size = 1
			continue
		}

		size++
	}

	indArr = append(indArr, types.PDFInteger(start))
	indArr = append(indArr, types.PDFInteger(size))

	logDebugWriter.Println("createXRefStream end")

	return
}

func writeXRefStream(ctx *types.PDFContext) (err error) {

	logInfoWriter.Println("writeXRefStream begin")

	xRefTable := ctx.XRefTable
	xRefStreamDict := types.NewPDFXRefStreamDict(xRefTable)
	xRefTableEntry := types.NewXRefTableEntryGen0()
	xRefTableEntry.Object = *xRefStreamDict

	// Reuse free objects (including recycled objects from this run).
	objNumber, err := xRefTable.InsertAndUseRecycled(*xRefTableEntry)
	if err != nil {
		return
	}

	// After the last insert of an object.
	err = xRefTable.EnsureValidFreeList()
	if err != nil {
		return
	}

	xRefStreamDict.Insert("Size", types.PDFInteger(*xRefTable.Size))

	offset := ctx.Write.Offset

	i2Base := int64(*ctx.Size)
	if offset > i2Base {
		i2Base = offset
	}

	i1 := 1 // 0, 1 or 2 always fit into 1 byte.
	i2 := int64ByteCount(i2Base)
	i3 := 2 // scale for max objectstream index <= 0x ff ff
	wArr := types.PDFArray{types.PDFInteger(i1), types.PDFInteger(i2), types.PDFInteger(i3)}
	xRefStreamDict.Insert("W", wArr)

	// Generate xRefStreamDict data = xref entries -> xRefStreamDict.Content
	content, indArr, err := createXRefStream(ctx, i1, i2, i3)
	if err != nil {
		return
	}

	xRefStreamDict.Content = content
	xRefStreamDict.Insert("Index", indArr)

	// Encode xRefStreamDict.Content -> xRefStreamDict.Raw
	err = filter.EncodeStream(&xRefStreamDict.PDFStreamDict)
	if err != nil {
		return
	}

	logInfoWriter.Printf("writeXRefStream: xRefStreamDict: %s\n", xRefStreamDict)

	err = writePDFStreamDictObject(ctx, objNumber, 0, xRefStreamDict.PDFStreamDict)
	if err != nil {
		return
	}

	w := ctx.Write

	_, err = w.WriteString(eol)
	if err != nil {
		return
	}

	_, err = w.WriteString("startxref")
	if err != nil {
		return
	}

	_, err = w.WriteString(eol)
	if err != nil {
		return
	}

	_, err = w.WriteString(fmt.Sprintf("%d", offset))
	if err != nil {
		return
	}

	_, err = w.WriteString(eol)
	if err != nil {
		return
	}

	logInfoWriter.Println("writeXRefStream end")

	return
}

// PDFFile generates a PDF file for the cross reference table contained in PDFContext.
func PDFFile(ctx *types.PDFContext) (err error) {

	fileName := ctx.Write.DirName + ctx.Write.FileName

	logInfoWriter.Printf("writing to %s...\n", fileName)

	file, err := os.Create(fileName)
	if err != nil {
		return errors.Wrapf(err, "can't create %s\n%s", fileName, err)
	}

	defer func() {
		ctx.Write.Flush()
		file.Close()
	}()

	//ctx.Write.File = file

	w := bufio.NewWriter(file)
	ctx.Write.Writer = w

	// Write a PDF file header stating the version of the used conforming writer.
	// This has to be the source version or any version higher.
	// For using objectstreams and xrefstreams we need at least PDF V1.5.
	if ctx.XRefTable.Version() < types.V15 {
		v, _ := types.Version("1.5")
		ctx.XRefTable.RootVersion = &v
		logInfoWriter.Println("Ensure V1.5 for writing object & xref streams")
	}

	err = writeHeader(ctx)
	if err != nil {
		return
	}

	logInfoWriter.Printf("offset after writeHeader: %d\n", ctx.Write.Offset)

	// Write root object(aka the document catalog) and page tree.
	err = writeRootObject(ctx)
	if err != nil {
		return
	}

	logInfoWriter.Printf("offset after writeRootObject: %d\n", ctx.Write.Offset)

	// Write document information dictionary.
	err = writeDocumentInfoObject(ctx)
	if err != nil {
		return
	}

	logInfoWriter.Printf("offset after writeInfoObject: %d\n", ctx.Write.Offset)

	// Write offspec additional streams as declared in pdf trailer.
	err = writeAdditionalStreams(ctx)
	if err != nil {
		return
	}

	// Mark redundant objects as free.
	// eg. duplicate resources, compressed objects, linearization dicts..
	err = deleteRedundantObjects(ctx)
	if err != nil {
		return
	}

	if ctx.WriteXRefStream {
		// Write cross reference stream and generate objectstreams.
		err = writeXRefStream(ctx)
		if err != nil {
			return
		}
	} else {
		// Write cross reference table section.
		err = writeXRefTable(ctx)
		if err != nil {
			return
		}
	}

	// Write pdf trailer.
	_, err = writeTrailer(ctx)
	if err != nil {
		return err
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}
	ctx.Write.FileSize = fileInfo.Size()

	// Refactor, calculate
	ctx.Write.BinaryImageSize = ctx.Read.BinaryImageSize
	ctx.Write.BinaryFontSize = ctx.Read.BinaryFontSize

	logWriteStats(ctx)

	return nil
}
