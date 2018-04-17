// Package write renders a PDF cross reference table to a PDF file.
package write

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/hhrutter/pdfcpu/crypto"
	"github.com/hhrutter/pdfcpu/filter"
	"github.com/hhrutter/pdfcpu/log"
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

// PDFFile generates a PDF file for the cross reference table contained in PDFContext.
func PDFFile(ctx *types.PDFContext) error {

	fileName := ctx.Write.DirName + ctx.Write.FileName

	log.Info.Printf("writing to %s\n", fileName)

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
		return err
	}

	// Since we support PDF Collections (since V1.7) for file attachments
	// we need to always generate V1.7 PDF filess.
	err = writeHeader(ctx.Write, types.V17)
	if err != nil {
		return err
	}

	log.Debug.Printf("offset after writeHeader: %d\n", ctx.Write.Offset)

	// Write root object(aka the document catalog) and page tree.
	err = writeRootObject(ctx)
	if err != nil {
		return err
	}

	log.Debug.Printf("offset after writeRootObject: %d\n", ctx.Write.Offset)

	// Write document information dictionary.
	err = writeDocumentInfoDict(ctx)
	if err != nil {
		return err
	}

	log.Debug.Printf("offset after writeInfoObject: %d\n", ctx.Write.Offset)

	// Write offspec additional streams as declared in pdf trailer.
	if ctx.AdditionalStreams != nil {
		_, _, err = writeDeepObject(ctx, ctx.AdditionalStreams)
		if err != nil {
			return err
		}
	}

	err = writeEncryptDict(ctx)
	if err != nil {
		return err
	}

	// Mark redundant objects as free.
	// eg. duplicate resources, compressed objects, linearization dicts..
	deleteRedundantObjects(ctx)

	err = writeXRef(ctx)
	if err != nil {
		return err
	}

	// Write pdf trailer.
	_, err = writeTrailer(ctx.Write)
	if err != nil {
		return err
	}

	err = setFileSizeOfWrittenFile(ctx.Write, file)
	if err != nil {
		return err
	}

	if ctx.Read != nil {
		ctx.Write.BinaryImageSize = ctx.Read.BinaryImageSize
		ctx.Write.BinaryFontSize = ctx.Read.BinaryFontSize
		logWriteStats(ctx)
	}

	return nil
}

// Write root entry to disk.
func writeRootEntry(ctx *types.PDFContext, dict *types.PDFDict, dictName, entryName string, statsAttr int) error {

	obj, err := writeEntry(ctx, dict, dictName, entryName)
	if err != nil {
		return err
	}

	if obj != nil {
		ctx.Stats.AddRootAttr(statsAttr)
	}

	return nil
}

// Write root entry to object stream.
func writeRootEntryToObjStream(ctx *types.PDFContext, dict *types.PDFDict, dictName, entryName string, statsAttr int) error {

	ctx.Write.WriteToObjectStream = true

	err := writeRootEntry(ctx, dict, dictName, entryName, statsAttr)
	if err != nil {
		return err
	}

	return stopObjectStream(ctx)
}

// Write page tree.
func writePages(ctx *types.PDFContext, rootDict *types.PDFDict) error {

	// Page tree root (the top "Pages" dict) must be indirect reference.
	indRef := rootDict.IndirectRefEntry("Pages")
	if indRef == nil {
		return errors.New("writePages: missing indirect obj for pages dict")
	}

	// Manipulate page tree as needed for splitting, trimming or page extraction.
	if ctx.Write.ExtractPages != nil && len(ctx.Write.ExtractPages) > 0 {
		p := 0
		_, err := trimPagesDict(ctx, indRef, &p)
		if err != nil {
			return err
		}
	}

	// Embed all page tree objects into objects stream.
	ctx.Write.WriteToObjectStream = true

	// Write page tree.
	err := writePagesDict(ctx, indRef, 0)
	if err != nil {
		return err
	}

	return stopObjectStream(ctx)
}

func writeRootObject(ctx *types.PDFContext) error {

	// => 7.7.2 Document Catalog

	xRefTable := ctx.XRefTable

	catalog := *xRefTable.Root
	objNumber := int(catalog.ObjectNumber)
	genNumber := int(catalog.GenerationNumber)

	log.Debug.Printf("*** writeRootObject: begin offset=%d *** %s\n", ctx.Write.Offset, catalog)

	// Ensure corresponding and accurate name tree object graphs.
	if !ctx.Write.ReducedFeatureSet() {
		err := ctx.XRefTable.BindNameTrees()
		if err != nil {
			return err
		}
	}

	var dict *types.PDFDict

	dict, err := xRefTable.DereferenceDict(catalog)
	if err != nil {
		return err
	}

	if dict == nil {
		return errors.Errorf("writeRootObject: unable to dereference root dict")
	}

	dictName := "rootDict"

	if ctx.Write.ReducedFeatureSet() {
		log.Debug.Println("writeRootObject: exclude complex entries on split,trim and page extraction.")
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
		return err
	}

	log.Debug.Printf("writeRootObject: %s\n", dict)

	log.Debug.Printf("writeRootObject: new offset after rootDict = %d\n", ctx.Write.Offset)

	err = writeRootEntry(ctx, dict, dictName, "Version", types.RootVersion)
	if err != nil {
		return err
	}

	err = writePages(ctx, dict)
	if err != nil {
		return err
	}

	for _, e := range []struct {
		entryName string
		statsAttr int
	}{
		{"Extensions", types.RootExtensions},
		{"PageLabels", types.RootPageLabels},
		{"Names", types.RootNames},
		{"Dests", types.RootDests},
		{"ViewerPreferences", types.RootViewerPrefs},
		{"PageLayout", types.RootPageLayout},
		{"PageMode", types.RootPageMode},
		{"Outlines", types.RootOutlines},
		{"Threads", types.RootThreads},
		{"OpenAction", types.RootOpenAction},
		{"AA", types.RootAA},
		{"URI", types.RootURI},
		{"AcroForm", types.RootAcroForm},
		{"Metadata", types.RootMetadata},
	} {
		err = writeRootEntry(ctx, dict, dictName, e.entryName, e.statsAttr)
		if err != nil {
			return err
		}
	}

	err = writeRootEntryToObjStream(ctx, dict, dictName, "StructTreeRoot", types.RootStructTreeRoot)
	if err != nil {
		return err
	}

	for _, e := range []struct {
		entryName string
		statsAttr int
	}{
		{"MarkInfo", types.RootMarkInfo},
		{"Lang", types.RootLang},
		{"SpiderInfo", types.RootSpiderInfo},
		{"OutputIntents", types.RootOutputIntents},
		{"PieceInfo", types.RootPieceInfo},
		{"OCProperties", types.RootOCProperties},
		{"Perms", types.RootPerms},
		{"Legal", types.RootLegal},
		{"Requirements", types.RootRequirements},
		{"Collection", types.RootCollection},
		{"NeedsRendering", types.RootNeedsRendering},
	} {
		err = writeRootEntry(ctx, dict, dictName, e.entryName, e.statsAttr)
		if err != nil {
			return err
		}
	}

	log.Debug.Printf("*** writeRootObject: end offset=%d ***\n", ctx.Write.Offset)

	return nil
}

func writeTrailerDict(ctx *types.PDFContext) error {

	log.Debug.Printf("writeTrailerDict begin\n")

	w := ctx.Write
	xRefTable := ctx.XRefTable

	_, err := w.WriteString("trailer")
	if err != nil {
		return err
	}

	err = w.WriteEol()
	if err != nil {
		return err
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
		return err
	}

	log.Debug.Printf("writeTrailerDict end\n")

	return nil
}

func writeXRefSubsection(ctx *types.PDFContext, start int, size int) error {

	log.Debug.Printf("writeXRefSubsection: start=%d size=%d\n", start, size)

	w := ctx.Write

	_, err := w.WriteString(fmt.Sprintf("%d %d%s", start, size, w.Eol))
	if err != nil {
		return err
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
			return err
		}
	}

	log.Debug.Printf("\n%s\n", strings.Join(lines, ""))
	log.Debug.Printf("writeXRefSubsection: end\n")

	return nil
}

func deleteRedundantObject(ctx *types.PDFContext, objNr int) {

	if ctx.Write.ExtractPageNr == 0 &&
		(ctx.Optimize.IsDuplicateFontObject(objNr) || ctx.Optimize.IsDuplicateImageObject(objNr)) {
		ctx.DeleteObject(objNr)
	}

	if ctx.IsLinearizationObject(objNr) || ctx.Optimize.IsDuplicateInfoObject(objNr) ||
		ctx.Read.IsObjectStreamObject(objNr) || ctx.Read.IsXRefStreamObject(objNr) {
		ctx.DeleteObject(objNr)
	}

}
func deleteRedundantObjects(ctx *types.PDFContext) {

	if ctx.Optimize == nil {
		return
	}

	xRefTable := ctx.XRefTable

	log.Debug.Printf("deleteRedundantObjects begin: Size=%d\n", *xRefTable.Size)

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
			log.Debug.Printf("deleteRedundantObjects: remove duplicate obj #%d\n", i)
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
					log.Debug.Printf("deleteRedundantObjects: primaryHintTable at obj #%d\n", i)
				}

				if xRefTable.OffsetOverflowHintTable != nil &&
					*entry.Offset == *xRefTable.OffsetOverflowHintTable {
					xRefTable.LinearizationObjs[i] = true
					log.Debug.Printf("deleteRedundantObjects: overflowHintTable at obj #%d\n", i)
				}

			}

		}

		deleteRedundantObject(ctx, i)

	}

	log.Debug.Println("deleteRedundantObjects end")
}

func sortedWritableKeys(ctx *types.PDFContext) []int {

	var keys []int

	for i, e := range ctx.Table {
		if e.Free || ctx.Write.HasWriteOffset(i) {
			keys = append(keys, i)
		}
	}

	sort.Ints(keys)

	return keys
}

// After inserting the last object write the cross reference table to disk.
func writeXRefTable(ctx *types.PDFContext) error {

	err := ctx.EnsureValidFreeList()
	if err != nil {
		return err
	}

	keys := sortedWritableKeys(ctx)

	objCount := len(keys)
	log.Debug.Printf("xref has %d entries\n", objCount)

	_, err = ctx.Write.WriteString("xref")
	if err != nil {
		return err
	}

	err = ctx.Write.WriteEol()
	if err != nil {
		return err
	}

	start := keys[0]
	size := 1

	for i := 1; i < len(keys); i++ {

		if keys[i]-keys[i-1] > 1 {

			err = writeXRefSubsection(ctx, start, size)
			if err != nil {
				return err
			}

			start = keys[i]
			size = 1
			continue
		}

		size++
	}

	err = writeXRefSubsection(ctx, start, size)
	if err != nil {
		return err
	}

	err = writeTrailerDict(ctx)
	if err != nil {
		return err
	}

	err = ctx.Write.WriteEol()
	if err != nil {
		return err
	}

	_, err = ctx.Write.WriteString("startxref")
	if err != nil {
		return err
	}

	err = ctx.Write.WriteEol()
	if err != nil {
		return err
	}

	_, err = ctx.Write.WriteString(fmt.Sprintf("%d", ctx.Write.Offset))
	if err != nil {
		return err
	}

	return ctx.Write.WriteEol()
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

func createXRefStream(ctx *types.PDFContext, i1, i2, i3 int) ([]byte, *types.PDFArray, error) {

	log.Debug.Println("createXRefStream begin")

	xRefTable := ctx.XRefTable

	var (
		buf []byte
		arr types.PDFArray
	)

	var keys []int
	for i, e := range xRefTable.Table {
		if e.Free || ctx.Write.HasWriteOffset(i) {
			keys = append(keys, i)
		}
	}
	sort.Ints(keys)

	objCount := len(keys)
	log.Debug.Printf("createXRefStream: xref has %d entries\n", objCount)

	start := keys[0]
	size := 0

	for i := 0; i < len(keys); i++ {

		j := keys[i]
		entry := xRefTable.Table[j]
		var s1, s2, s3 []byte

		if entry.Free {

			// unused
			log.Debug.Printf("createXRefStream: unused i=%d nextFreeAt:%d gen:%d\n", j, int(*entry.Offset), int(*entry.Generation))

			s1 = int64ToBuf(0, i1)
			s2 = int64ToBuf(*entry.Offset, i2)
			s3 = int64ToBuf(int64(*entry.Generation), i3)

		} else if entry.Compressed {

			// in use, compressed into object stream
			log.Debug.Printf("createXRefStream: compressed i=%d at objstr %d[%d]\n", j, int(*entry.ObjectStream), int(*entry.ObjectStreamInd))

			s1 = int64ToBuf(2, i1)
			s2 = int64ToBuf(int64(*entry.ObjectStream), i2)
			s3 = int64ToBuf(int64(*entry.ObjectStreamInd), i3)

		} else {

			off, found := ctx.Write.Table[j]
			if !found {
				return nil, nil, errors.Errorf("createXRefStream: missing write offset for obj #%d\n", i)
			}

			// in use, uncompressed
			log.Debug.Printf("createXRefStream: used i=%d offset:%d gen:%d\n", j, int(off), int(*entry.Generation))

			s1 = int64ToBuf(1, i1)
			s2 = int64ToBuf(off, i2)
			s3 = int64ToBuf(int64(*entry.Generation), i3)

		}

		log.Debug.Printf("createXRefStream: written: %x %x %x \n", s1, s2, s3)

		buf = append(buf, s1...)
		buf = append(buf, s2...)
		buf = append(buf, s3...)

		if i > 0 && (keys[i]-keys[i-1] > 1) {

			arr = append(arr, types.PDFInteger(start))
			arr = append(arr, types.PDFInteger(size))

			start = keys[i]
			size = 1
			continue
		}

		size++
	}

	arr = append(arr, types.PDFInteger(start))
	arr = append(arr, types.PDFInteger(size))

	log.Debug.Println("createXRefStream end")

	return buf, &arr, nil
}

func writeXRefStream(ctx *types.PDFContext) error {

	log.Debug.Println("writeXRefStream begin")

	xRefTable := ctx.XRefTable
	xRefStreamDict := types.NewPDFXRefStreamDict(ctx)
	xRefTableEntry := types.NewXRefTableEntryGen0(*xRefStreamDict)

	// Reuse free objects (including recycled objects from this run).
	var objNumber int
	objNumber, err := xRefTable.InsertAndUseRecycled(*xRefTableEntry)
	if err != nil {
		return err
	}

	// After the last insert of an object.
	err = xRefTable.EnsureValidFreeList()
	if err != nil {
		return err
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
		return byteCount
	}(i2Base)

	i3 := 2 // scale for max objectstream index <= 0x ff ff

	wArr := types.PDFArray{types.PDFInteger(i1), types.PDFInteger(i2), types.PDFInteger(i3)}
	xRefStreamDict.Insert("W", wArr)

	// Generate xRefStreamDict data = xref entries -> xRefStreamDict.Content
	content, indArr, err := createXRefStream(ctx, i1, i2, i3)
	if err != nil {
		return err
	}

	xRefStreamDict.Content = content
	xRefStreamDict.Insert("Index", *indArr)

	// Encode xRefStreamDict.Content -> xRefStreamDict.Raw
	err = filter.EncodeStream(&xRefStreamDict.PDFStreamDict)
	if err != nil {
		return err
	}

	log.Debug.Printf("writeXRefStream: xRefStreamDict: %s\n", xRefStreamDict)

	err = writePDFStreamDictObject(ctx, objNumber, 0, xRefStreamDict.PDFStreamDict)
	if err != nil {
		return err
	}

	w := ctx.Write

	err = w.WriteEol()
	if err != nil {
		return err
	}

	_, err = w.WriteString("startxref")
	if err != nil {
		return err
	}

	err = w.WriteEol()
	if err != nil {
		return err
	}

	_, err = w.WriteString(fmt.Sprintf("%d", offset))
	if err != nil {
		return err
	}

	err = w.WriteEol()
	if err != nil {
		return err
	}

	log.Debug.Println("writeXRefStream end")

	return nil
}

func writeEncryptDict(ctx *types.PDFContext) error {

	// Bail out unless we really have to write encrypted.
	if ctx.Encrypt == nil || ctx.EncKey == nil {
		return nil
	}

	indRef := *ctx.Encrypt
	objNumber := int(indRef.ObjectNumber)
	genNumber := int(indRef.GenerationNumber)

	var dict *types.PDFDict

	dict, err := ctx.DereferenceDict(indRef)
	if err != nil {
		return err
	}

	return writePDFObject(ctx, objNumber, genNumber, dict.PDFString())
}

func setupEncryption(ctx *types.PDFContext) error {

	var err error

	dict := crypto.NewEncryptDict(ctx.EncryptUsingAES, ctx.EncryptUsing128BitKey, ctx.UserAccessPermissions)

	ctx.E, err = crypto.SupportedEncryption(ctx, dict)
	if err != nil {
		return err
	}

	if ctx.ID == nil {
		return errors.New("encrypt: missing ID")
	}

	var id []byte
	id, err = ctx.IDFirstElement()
	if err != nil {
		return err
	}

	ctx.E.ID = id

	//fmt.Printf("opw before: length:%d <%s>\n", len(ctx.E.O), ctx.E.O)
	ctx.E.O, err = crypto.O(ctx)
	if err != nil {
		return err
	}
	//fmt.Printf("opw after: length:%d <%s> %0X\n", len(ctx.E.O), ctx.E.O, ctx.E.O)

	//fmt.Printf("upw before: length:%d <%s>\n", len(ctx.E.U), ctx.E.U)
	ctx.E.U, ctx.EncKey, err = crypto.U(ctx)
	if err != nil {
		return err
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
		return err
	}

	ctx.Encrypt = types.NewPDFIndirectRef(objNumber, 0)

	return nil
}

func updateEncryption(ctx *types.PDFContext) error {

	d, err := ctx.EncryptDict()
	if err != nil {
		return err
	}

	if ctx.Mode == types.ADDPERMISSIONS {
		//fmt.Printf("updating permissions to: %v\n", ctx.UserAccessPermissions)
		ctx.E.P = int(ctx.UserAccessPermissions)
		d.Update("P", types.PDFInteger(ctx.E.P))
		// and moving on, U is dependent on P
	}

	// Change user or owner password.
	//fmt.Println("change upw or opw")

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
		return err
	}
	//fmt.Printf("opw after: length:%d <%s> %0X\n", len(ctx.E.O), ctx.E.O, ctx.E.O)
	d.Update("O", types.PDFHexLiteral(hex.EncodeToString(ctx.E.O)))

	//fmt.Printf("upw before: length:%d <%s>\n", len(ctx.E.U), ctx.E.U)
	ctx.E.U, ctx.EncKey, err = crypto.U(ctx)
	if err != nil {
		return err
	}
	//fmt.Printf("upw after: length:%d <%s> %0X\n", len(ctx.E.U), ctx.E.U, ctx.E.U)
	//fmt.Printf("encKey = %0X\n", ctx.EncKey)
	d.Update("U", types.PDFHexLiteral(hex.EncodeToString(ctx.E.U)))

	return nil
}

func handleEncryption(ctx *types.PDFContext) error {

	if ctx.Mode == types.ENCRYPT || ctx.Mode == types.DECRYPT {

		if ctx.Mode == types.DECRYPT {

			// Remove encryption.
			ctx.EncKey = nil

		} else {

			err := setupEncryption(ctx)
			if err != nil {
				return err
			}

		}

	} else if ctx.UserPWNew != nil || ctx.OwnerPWNew != nil || ctx.Mode == types.ADDPERMISSIONS {

		err := updateEncryption(ctx)
		if err != nil {
			return err
		}

	}

	// write xrefstream if using xrefstream only.
	if ctx.Encrypt != nil && ctx.EncKey != nil && !ctx.Read.UsingXRefStreams {
		ctx.WriteObjectStream = false
		ctx.WriteXRefStream = false
	}

	return nil
}

func writeXRef(ctx *types.PDFContext) error {

	if ctx.WriteXRefStream {
		// Write cross reference stream and generate objectstreams.
		return writeXRefStream(ctx)
	}

	// Write cross reference table section.
	return writeXRefTable(ctx)
}

func setFileSizeOfWrittenFile(w *types.WriteContext, f *os.File) error {

	// Get file info for file just written but flush first to get correct file size.

	err := w.Flush()
	if err != nil {
		return err
	}

	fileInfo, err := f.Stat()
	if err != nil {
		return err
	}

	w.FileSize = fileInfo.Size()

	return nil
}
