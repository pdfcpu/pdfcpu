// Package write renders a PDF cross reference table to a PDF file.
package write

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/hhrutter/pdfcpu/crypto"
	"github.com/hhrutter/pdfcpu/filter"
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

var (
	logDebugWriter *log.Logger
	logInfoWriter  *log.Logger
	logErrorWriter *log.Logger
	logPages       *log.Logger
	logXRef        *log.Logger
)

func init() {

	logDebugWriter = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	//logDebugWriter = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	logInfoWriter = log.New(ioutil.Discard, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	logErrorWriter = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	logPages = log.New(ioutil.Discard, "PAGES: ", log.Ldate|log.Ltime|log.Lshortfile)
	logXRef = log.New(ioutil.Discard, "XREF: ", log.Ldate|log.Ltime|log.Lshortfile)
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

// Write root entry to disk.
func writeRootEntry(ctx *types.PDFContext, dict *types.PDFDict, dictName, entryName string, statsAttr int) (err error) {

	var obj interface{}

	obj, err = writeEntry(ctx, dict, dictName, entryName)
	if err != nil {
		return
	}

	if obj != nil {
		ctx.Stats.AddRootAttr(statsAttr)
	}

	return
}

// Write root entry to object stream.
func writeRootEntryToObjStream(ctx *types.PDFContext, dict *types.PDFDict, dictName, entryName string, statsAttr int) (err error) {

	ctx.Write.WriteToObjectStream = true

	err = writeRootEntry(ctx, dict, dictName, entryName, statsAttr)
	if err != nil {
		return
	}

	err = stopObjectStream(ctx)
	if err != nil {
		return
	}

	return
}

// Write page tree.
func writePages(ctx *types.PDFContext, rootDict *types.PDFDict) (err error) {

	// Page tree root (the top "Pages" dict) must be indirect reference.
	indRef := rootDict.IndirectRefEntry("Pages")
	if indRef == nil {
		err = errors.New("writePages: missing indirect obj for pages dict")
		return
	}

	// Manipulate page tree as needed for splitting, trimming or page extraction.
	if ctx.Write.ExtractPages != nil && len(ctx.Write.ExtractPages) > 0 {
		p := 0
		_, err = trimPagesDict(ctx, *indRef, &p)
		if err != nil {
			return
		}
	}

	// Embed all page tree objects into objects stream.
	ctx.Write.WriteToObjectStream = true

	// Write page tree.
	err = writePagesDict(ctx, *indRef, 0)
	if err != nil {
		return
	}

	err = stopObjectStream(ctx)
	if err != nil {
		return
	}

	return
}

func writeRootObject(ctx *types.PDFContext) (err error) {

	// => 7.7.2 Document Catalog

	// Entry	   	       opt	since		type			info
	//------------------------------------------------------------------------------------
	// Type			        n				string			"Catalog"
	// Version		        y	1.4			name			overrules header version if later
	// Extensions	        y	ISO 32000	dict			=> 7.12 Extensions Dictionary
	// Pages		        n	-			(dict)			=> 7.7.3 Page Tree
	// PageLabels	        y	1.3			number tree		=> 7.9.7 Number Trees, 12.4.2 Page Labels
	// Names		        y	1.2			dict			=> 7.7.4 Name Dictionary
	// Dests	    	    y	only 1.1	(dict)			=> 12.3.2.3 Named Destinations
	// ViewerPreferences    y	1.2			dict			=> 12.2 Viewer Preferences
	// PageLayout	        y	-			name			/SinglePage, /OneColumn etc.
	// PageMode		        y	-			name			/UseNone, /FullScreen etc.
	// Outlines		        y	-			(dict)			=> 12.3.3 Document Outline
	// Threads		        y	1.1			(array)			=> 12.4.3 Articles
	// OpenAction	        y	1.1			array or dict	=> 12.3.2 Destinations, 12.6 Actions
	// AA			        y	1.4			dict			=> 12.6.3 Trigger Events
	// URI			        y	1.1			dict			=> 12.6.4.7 URI Actions
	// AcroForm		        y	1.2			dict			=> 12.7.2 Interactive Form Dictionary
	// Metadata		        y	1.4			(stream)		=> 14.3.2 Metadata Streams
	// StructTreeRoot 	    y 	1.3			dict			=> 14.7.2 Structure Hierarchy
	// Markinfo		        y	1.4			dict			=> 14.7 Logical Structure
	// Lang			        y	1.4			string
	// SpiderInfo	        y	1.3			dict			=> 14.10.2 Web Capture Information Dictionary
	// OutputIntents 	    y	1.4			array			=> 14.11.5 Output Intents
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

	var dict *types.PDFDict

	dict, err = xRefTable.DereferenceDict(catalog)
	if err != nil {
		return
	}

	if dict == nil {
		err = errors.Errorf("writeRootObject: unable to dereference root dict")
		return
	}

	dictName := "rootDict"

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

	err = writePDFDictObject(ctx, objNumber, genNumber, *dict)
	if err != nil {
		return
	}

	logPages.Printf("writeRootObject: %s\n", dict)

	logDebugWriter.Printf("writeRootObject: new offset after rootDict = %d\n", ctx.Write.Offset)

	err = writeRootEntry(ctx, dict, dictName, "Version", types.RootVersion)
	if err != nil {
		return
	}

	err = writePages(ctx, dict)
	if err != nil {
		return
	}

	err = writeRootEntry(ctx, dict, dictName, "Extensions", types.RootExtensions)
	if err != nil {
		return
	}

	err = writeRootEntry(ctx, dict, dictName, "PageLabels", types.RootPageLabels)
	if err != nil {
		return
	}

	err = writeRootEntry(ctx, dict, dictName, "Names", types.RootNames)
	if err != nil {
		return
	}

	err = writeRootEntry(ctx, dict, dictName, "Dests", types.RootDests)
	if err != nil {
		return
	}

	err = writeRootEntry(ctx, dict, dictName, "ViewerPreferences", types.RootViewerPrefs)
	if err != nil {
		return
	}

	err = writeRootEntry(ctx, dict, dictName, "PageLayout", types.RootPageLayout)
	if err != nil {
		return
	}

	err = writeRootEntry(ctx, dict, dictName, "PageMode", types.RootPageMode)
	if err != nil {
		return
	}

	err = writeRootEntry(ctx, dict, dictName, "Outlines", types.RootOutlines)
	if err != nil {
		return
	}

	err = writeRootEntry(ctx, dict, dictName, "Threads", types.RootThreads)
	if err != nil {
		return
	}

	err = writeRootEntry(ctx, dict, dictName, "OpenAction", types.RootOpenAction)
	if err != nil {
		return
	}

	err = writeRootEntry(ctx, dict, dictName, "AA", types.RootAA)
	if err != nil {
		return
	}

	err = writeRootEntry(ctx, dict, dictName, "URI", types.RootURI)
	if err != nil {
		return err
	}

	err = writeRootEntry(ctx, dict, dictName, "AcroForm", types.RootAcroForm)
	if err != nil {
		return err
	}

	err = writeRootEntry(ctx, dict, dictName, "Metadata", types.RootMetadata)
	if err != nil {
		return err
	}

	err = writeRootEntryToObjStream(ctx, dict, dictName, "StructTreeRoot", types.RootStructTreeRoot)
	if err != nil {
		return
	}

	err = writeRootEntry(ctx, dict, dictName, "MarkInfo", types.RootMarkInfo)
	if err != nil {
		return
	}

	err = writeRootEntry(ctx, dict, dictName, "Lang", types.RootLang)
	if err != nil {
		return
	}

	err = writeRootEntry(ctx, dict, dictName, "SpiderInfo", types.RootSpiderInfo)
	if err != nil {
		return err
	}

	err = writeRootEntry(ctx, dict, dictName, "OutputIntents", types.RootOutputIntents)
	if err != nil {
		return err
	}

	err = writeRootEntry(ctx, dict, dictName, "PieceInfo", types.RootPieceInfo)
	if err != nil {
		return err
	}

	err = writeRootEntry(ctx, dict, dictName, "OCProperties", types.RootOCProperties)
	if err != nil {
		return
	}

	err = writeRootEntry(ctx, dict, dictName, "Perms", types.RootPerms)
	if err != nil {
		return
	}

	err = writeRootEntry(ctx, dict, dictName, "Legal", types.RootLegal)
	if err != nil {
		return
	}

	err = writeRootEntry(ctx, dict, dictName, "Requirements", types.RootRequirements)
	if err != nil {
		return
	}

	err = writeRootEntry(ctx, dict, dictName, "Collection", types.RootCollection)
	if err != nil {
		return
	}

	err = writeRootEntry(ctx, dict, dictName, "NeedsRendering", types.RootNeedsRendering)
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeRootObject: end offset=%d ***\n", ctx.Write.Offset)

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

	err = w.WriteEol()
	if err != nil {
		return
	}

	dict := types.NewPDFDict()
	dict.Insert("Size", types.PDFInteger(*xRefTable.Size))
	dict.Insert("Root", *xRefTable.Root)

	if xRefTable.Info != nil {
		dict.Insert("Info", *xRefTable.Info)
	}

	if ctx.Encrypt != nil && ctx.EncKey != nil {
		dict.Insert("Encrypt", *ctx.Encrypt)
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

	_, err = w.WriteString(fmt.Sprintf("%d %d%s", start, size, w.Eol))
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
			s = fmt.Sprintf("%010d %05d f%2s", *entry.Offset, *entry.Generation, w.Eol)
		} else {
			var off int64
			writeOffset, found := ctx.Write.Table[i]
			if found {
				off = writeOffset
			}
			s = fmt.Sprintf("%010d %05d n%2s", off, *entry.Generation, w.Eol)
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

		// Missing object remains missing.
		entry, found := xRefTable.Find(i)
		if !found {
			continue
		}

		// Free object
		if entry.Free {
			continue
		}

		// Object written
		if ctx.Write.HasWriteOffset(i) {
			// Resources may be cross referenced from different objects
			// eg. font descriptors may be shared by different font dicts.
			// Try to remove this object from the list of the potential duplicate objects.
			logDebugWriter.Printf("deleteRedundantObjects: remove duplicate obj #%d\n", i)
			delete(ctx.Optimize.DuplicateFontObjs, i)
			delete(ctx.Optimize.DuplicateImageObjs, i)
			delete(ctx.Optimize.DuplicateInfoObjects, i)
			continue
		}

		// Object not written

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

// After inserting the last object write the cross reference table to disk.
func writeXRefTable(ctx *types.PDFContext) (err error) {

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

	err = ctx.Write.WriteEol()
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

	err = ctx.Write.WriteEol()
	if err != nil {
		return
	}

	_, err = ctx.Write.WriteString("startxref")
	if err != nil {
		return
	}

	err = ctx.Write.WriteEol()
	if err != nil {
		return
	}

	_, err = ctx.Write.WriteString(fmt.Sprintf("%d", ctx.Write.Offset))
	if err != nil {
		return
	}

	err = ctx.Write.WriteEol()
	if err != nil {
		return
	}

	return
}

// int64ToBuf returns a byte slice with length byteCount representing integer i.
func int64ToBuf(i int64, byteCount int) (buf []byte) {

	j := 0
	var b []byte

	for k := i; k > 0; {
		b = append(b, byte(k&0xff))
		k >>= 8
		j++
	}

	// Swap byte order
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
	xRefStreamDict := types.NewPDFXRefStreamDict(ctx)
	xRefTableEntry := types.NewXRefTableEntryGen0(*xRefStreamDict)

	// Reuse free objects (including recycled objects from this run).
	var objNumber int
	objNumber, err = xRefTable.InsertAndUseRecycled(*xRefTableEntry)
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

	i2 := func(i int64) (byteCount int) {
		for i > 0 {
			i >>= 8
			byteCount++
		}
		return
	}(i2Base)

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

	err = w.WriteEol()
	if err != nil {
		return
	}

	_, err = w.WriteString("startxref")
	if err != nil {
		return
	}

	err = w.WriteEol()
	if err != nil {
		return
	}

	_, err = w.WriteString(fmt.Sprintf("%d", offset))
	if err != nil {
		return
	}

	err = w.WriteEol()
	if err != nil {
		return
	}

	logInfoWriter.Println("writeXRefStream end")

	return
}

func writeEncryptDict(ctx *types.PDFContext) (err error) {

	// Bail out unless we really have to write encrypted.
	if ctx.Encrypt == nil || ctx.EncKey == nil {
		return
	}

	indRef := *ctx.Encrypt
	objNumber := int(indRef.ObjectNumber)
	genNumber := int(indRef.GenerationNumber)

	var dict *types.PDFDict

	dict, err = ctx.DereferenceDict(indRef)
	if err != nil {
		return
	}

	return writePDFObject(ctx, objNumber, genNumber, dict.PDFString())
}

func handleEncryption(ctx *types.PDFContext) (err error) {

	var d *types.PDFDict

	if ctx.Decrypt != nil {

		// Flag is set for encryption or decryption.

		if *ctx.Decrypt {

			// Remove encryption.
			ctx.EncKey = nil

		} else {

			// Encrypt this document.

			dict := types.NewEncryptDict()

			ctx.E, err = crypto.SupportedEncryption(ctx, dict)
			if err != nil {
				return
			}

			if ctx.ID == nil {
				return errors.New("encrypt: missing ID")
			}

			var id []byte

			hl, ok := ((*ctx.ID)[0]).(types.PDFHexLiteral)
			if ok {
				id, err = hl.Bytes()
				if err != nil {
					return err
				}
			} else {
				sl, ok := ((*ctx.ID)[0]).(types.PDFStringLiteral)
				if !ok {
					return errors.New("encrypt: ID must contain PDFHexLiterals or PDFStringLiterals")
				}
				id, err = types.Unescape(sl.Value())
				if err != nil {
					return err
				}
			}

			ctx.E.ID = id

			//fmt.Printf("opw before: length:%d <%s>\n", len(ctx.E.O), ctx.E.O)
			ctx.E.O, err = crypto.O(ctx)
			if err != nil {
				return
			}
			//fmt.Printf("opw after: length:%d <%s> %0X\n", len(ctx.E.O), ctx.E.O, ctx.E.O)

			//fmt.Printf("upw before: length:%d <%s>\n", len(ctx.E.U), ctx.E.U)
			ctx.E.U, ctx.EncKey, err = crypto.U(ctx)
			if err != nil {
				return
			}
			if len(ctx.E.U) < 32 {
				pad := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
					0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
				//pad := "12345678901234567890123456789012"
				ctx.E.U = append(ctx.E.U, pad[:32-len(ctx.E.U)]...)
			}
			//fmt.Printf("upw after: length:%d <%s> %0X\n", len(ctx.E.U), ctx.E.U, ctx.E.U)
			//fmt.Printf("encKey = %0X\n", ctx.EncKey)

			dict.Update("U", types.PDFHexLiteral(hex.EncodeToString(ctx.E.U)))
			dict.Update("O", types.PDFHexLiteral(hex.EncodeToString(ctx.E.O)))

			xRefTableEntry := types.NewXRefTableEntryGen0(*dict)

			// Reuse free objects (including recycled objects from this run).
			var objNumber int
			objNumber, err = ctx.InsertAndUseRecycled(*xRefTableEntry)
			if err != nil {
				return
			}

			indRef := types.NewPDFIndirectRef(objNumber, 0)
			ctx.Encrypt = &indRef

		}
	} else if ctx.UserPWNew != nil || ctx.OwnerPWNew != nil {

		// Change user or owner password.

		if ctx.UserPWNew != nil {
			//fmt.Printf("change upw from <%s> to <%s>\n", ctx.UserPW, *ctx.UserPWNew)
			ctx.UserPW = *ctx.UserPWNew
		}

		if ctx.OwnerPWNew != nil {
			//fmt.Printf("change opw from <%s> to <%s>\n", ctx.OwnerPW, *ctx.OwnerPWNew)
			ctx.OwnerPW = *ctx.OwnerPWNew
		}

		//fmt.Printf("opw before: length:%d <%s>\n", len(ctx.E.O), ctx.E.O)
		ctx.E.O, err = crypto.O(ctx)
		if err != nil {
			return
		}
		//fmt.Printf("opw after: length:%d <%s> %0X\n", len(ctx.E.O), ctx.E.O, ctx.E.O)

		//fmt.Printf("upw before: length:%d <%s>\n", len(ctx.E.U), ctx.E.U)
		ctx.E.U, ctx.EncKey, err = crypto.U(ctx)
		if err != nil {
			return
		}
		if len(ctx.E.U) < 32 {
			pad := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
			//pad := "12345678901234567890123456789012"
			ctx.E.U = append(ctx.E.U, pad[:32-len(ctx.E.U)]...)
		}
		//fmt.Printf("upw after: length:%d <%s> %0X\n", len(ctx.E.U), ctx.E.U, ctx.E.U)
		//fmt.Printf("encKey = %0X\n", ctx.EncKey)

		// update EncryptDict.U+O
		d, err = ctx.EncryptDict()
		if err != nil {
			return
		}

		d.Update("U", types.PDFHexLiteral(hex.EncodeToString(ctx.E.U)))
		d.Update("O", types.PDFHexLiteral(hex.EncodeToString(ctx.E.O)))

	}

	// write xrefstream only if aleady using xrefstream.
	if ctx.Encrypt != nil && ctx.EncKey != nil && !ctx.Read.UsingXRefStreams {
		ctx.WriteObjectStream = false
		ctx.WriteXRefStream = false
	}

	return
}

func writeXRef(ctx *types.PDFContext) (err error) {

	if ctx.WriteXRefStream {
		// Write cross reference stream and generate objectstreams.
		return writeXRefStream(ctx)
	}

	// Write cross reference table section.
	return writeXRefTable(ctx)
}

// PDFFile generates a PDF file for the cross reference table contained in PDFContext.
func PDFFile(ctx *types.PDFContext) (err error) {

	fileName := ctx.Write.DirName + ctx.Write.FileName

	logInfoWriter.Printf("writing to %s...\n", fileName)

	file, err := os.Create(fileName)
	if err != nil {
		return errors.Wrapf(err, "can't create %s\n%s", fileName, err)
	}

	ctx.Write.Writer = bufio.NewWriter(file)

	defer func() {

		// The underlying bufio.Writer has already been flushed.

		// Processing error takes precedence.
		if err != nil {
			file.Close()
			return
		}

		// Do not miss out on closing errors.
		err = file.Close()

	}()

	err = handleEncryption(ctx)
	if err != nil {
		return
	}

	// Write a PDF file header stating the version of the used conforming writer.
	// This has to be the source version or any version higher.
	// For using objectstreams and xrefstreams we need at least PDF V1.5.
	v := ctx.Version()
	if ctx.Version() < types.V15 {
		v = types.V15
		logInfoWriter.Println("Ensure V1.5 for writing object & xref streams")
	}

	err = writeHeader(ctx.Write, v)
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
	err = writeDocumentInfoDict(ctx)
	if err != nil {
		return
	}

	logInfoWriter.Printf("offset after writeInfoObject: %d\n", ctx.Write.Offset)

	// Write offspec additional streams as declared in pdf trailer.
	if ctx.AdditionalStreams != nil {
		_, _, err = writeDeepObject(ctx, ctx.AdditionalStreams)
		if err != nil {
			return
		}
	}

	err = writeEncryptDict(ctx)
	if err != nil {
		return
	}

	// Mark redundant objects as free.
	// eg. duplicate resources, compressed objects, linearization dicts..
	err = deleteRedundantObjects(ctx)
	if err != nil {
		return
	}

	err = writeXRef(ctx)
	if err != nil {
		return
	}

	// Write pdf trailer.
	_, err = writeTrailer(ctx.Write)
	if err != nil {
		return
	}

	// Get file info for file just written,
	// but flush first to get correct file size.
	err = ctx.Write.Flush()
	if err != nil {
		return
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return
	}
	ctx.Write.FileSize = fileInfo.Size()

	ctx.Write.BinaryImageSize = ctx.Read.BinaryImageSize
	ctx.Write.BinaryFontSize = ctx.Read.BinaryFontSize

	logWriteStats(ctx)

	return
}
