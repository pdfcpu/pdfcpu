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
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pkg/errors"
)

// Write generates a PDF file for the cross reference table contained in Context.
func Write(ctx *Context) (err error) {
	// Create a writer for dirname and filename if not already supplied.
	if ctx.Write.Writer == nil {

		fileName := filepath.Join(ctx.Write.DirName, ctx.Write.FileName)
		log.CLI.Printf("writing to %s\n", fileName)

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

	}

	if err = prepareContextForWriting(ctx); err != nil {
		return err
	}

	// Since we support PDF Collections (since V1.7) for file attachments
	// we need to generate V1.7 PDF files.
	if err = writeHeader(ctx.Write, V17); err != nil {
		return err
	}

	// Ensure there is no root version.
	if ctx.RootVersion != nil {
		ctx.RootDict.Delete("Version")
	}

	log.Write.Printf("offset after writeHeader: %d\n", ctx.Write.Offset)

	// Write root object(aka the document catalog) and page tree.
	if err = writeRootObject(ctx); err != nil {
		return err
	}

	log.Write.Printf("offset after writeRootObject: %d\n", ctx.Write.Offset)

	// Write document information dictionary.
	if err = ctx.writeDocumentInfoDict(); err != nil {
		return err
	}

	log.Write.Printf("offset after writeInfoObject: %d\n", ctx.Write.Offset)

	// Write offspec additional streams as declared in pdf trailer.
	if err = writeAdditionalStreams(ctx); err != nil {
		return err
	}

	if err = writeEncryptDict(ctx); err != nil {
		return err
	}

	// Mark redundant objects as free.
	// eg. duplicate resources, compressed objects, linearization dicts..
	deleteRedundantObjects(ctx)

	if err = writeXRef(ctx); err != nil {
		return err
	}

	// Write pdf trailer.
	if err = writeTrailer(ctx.Write); err != nil {
		return err
	}

	if err = setFileSizeOfWrittenFile(ctx.Write); err != nil {
		return err
	}

	if ctx.Read != nil {
		ctx.Write.BinaryImageSize = ctx.Read.BinaryImageSize
		ctx.Write.BinaryFontSize = ctx.Read.BinaryFontSize
		logWriteStats(ctx)
	}

	return nil
}

// WriteIncrement writes a PDF increment..
func WriteIncrement(ctx *Context) error {

	// Write all modified objects that are part of this increment.
	for _, i := range ctx.Write.ObjNrs {
		if err := writeFlatObject(ctx, i); err != nil {
			return err
		}
	}

	if err := writeXRef(ctx); err != nil {
		return err
	}

	return writeTrailer(ctx.Write)
}

func prepareContextForWriting(ctx *Context) error {

	if err := ensureInfoDictAndFileID(ctx); err != nil {
		return err
	}

	return handleEncryption(ctx)
}

func writeAdditionalStreams(ctx *Context) error {

	if ctx.AdditionalStreams == nil {
		return nil
	}

	if _, _, err := writeDeepObject(ctx, ctx.AdditionalStreams); err != nil {
		return err
	}

	return nil
}

func ensureFileID(ctx *Context) error {

	fid, err := fileID(ctx)
	if err != nil {
		return err
	}

	if ctx.ID == nil {
		// Ensure ctx.ID
		ctx.ID = Array{fid, fid}
		return nil
	}

	// Update ctx.ID
	a := ctx.ID
	if len(a) != 2 {
		return errors.New("pdfcpu: ID must be an array with 2 elements")
	}

	a[1] = fid

	return nil
}

func ensureInfoDictAndFileID(ctx *Context) error {

	if err := ctx.ensureInfoDict(); err != nil {
		return err
	}

	return ensureFileID(ctx)
}

// Write root entry to disk.
func writeRootEntry(ctx *Context, d Dict, dictName, entryName string, statsAttr int) error {

	o, err := writeEntry(ctx, d, dictName, entryName)
	if err != nil {
		return err
	}

	if o != nil {
		ctx.Stats.AddRootAttr(statsAttr)
	}

	return nil
}

// Write root entry to object stream.
func writeRootEntryToObjStream(ctx *Context, d Dict, dictName, entryName string, statsAttr int) error {

	ctx.Write.WriteToObjectStream = true

	if err := writeRootEntry(ctx, d, dictName, entryName, statsAttr); err != nil {
		return err
	}

	return stopObjectStream(ctx)
}

// Write page tree.
func writePages(ctx *Context, rootDict Dict) error {

	// Page tree root (the top "Pages" dict) must be indirect reference.
	ir := rootDict.IndirectRefEntry("Pages")
	if ir == nil {
		return errors.New("pdfcpu: writePages: missing indirect obj for pages dict")
	}

	// Embed all page tree objects into objects stream.
	ctx.Write.WriteToObjectStream = true

	// Write page tree.
	p := 0
	if _, _, err := writePagesDict(ctx, ir, &p); err != nil {
		return err
	}

	return stopObjectStream(ctx)
}

func writeRootObject(ctx *Context) error {

	// => 7.7.2 Document Catalog

	xRefTable := ctx.XRefTable
	catalog := *xRefTable.Root
	objNumber := int(catalog.ObjectNumber)
	genNumber := int(catalog.GenerationNumber)

	log.Write.Printf("*** writeRootObject: begin offset=%d *** %s\n", ctx.Write.Offset, catalog)

	// Ensure corresponding and accurate name tree object graphs.
	// if !ctx.ApplyReducedFeatureSet() {
	if err := ctx.BindNameTrees(); err != nil {
		return err
	}
	// }

	d, err := xRefTable.DereferenceDict(catalog)
	if err != nil {
		return err
	}

	if d == nil {
		return errors.Errorf("pdfcpu: writeRootObject: unable to dereference root dict")
	}

	dictName := "rootDict"

	// if ctx.ApplyReducedFeatureSet() {
	// 	log.Write.Println("writeRootObject - reducedFeatureSet:exclude complex entries.")
	// 	d.Delete("Names")
	// 	d.Delete("Dests")
	// 	d.Delete("Outlines")
	// 	d.Delete("OpenAction")
	// 	d.Delete("AcroForm")
	// 	d.Delete("StructTreeRoot")
	// 	d.Delete("OCProperties")
	// }

	if err = writeDictObject(ctx, objNumber, genNumber, d); err != nil {
		return err
	}

	log.Write.Printf("writeRootObject: %s\n", d)

	log.Write.Printf("writeRootObject: new offset after rootDict = %d\n", ctx.Write.Offset)

	if err = writeRootEntry(ctx, d, dictName, "Version", RootVersion); err != nil {
		return err
	}

	if err = writePages(ctx, d); err != nil {
		return err
	}

	for _, e := range []struct {
		entryName string
		statsAttr int
	}{
		{"Extensions", RootExtensions},
		{"PageLabels", RootPageLabels},
		{"Names", RootNames},
		{"Dests", RootDests},
		{"ViewerPreferences", RootViewerPrefs},
		{"PageLayout", RootPageLayout},
		{"PageMode", RootPageMode},
		{"Outlines", RootOutlines},
		{"Threads", RootThreads},
		{"OpenAction", RootOpenAction},
		{"AA", RootAA},
		{"URI", RootURI},
		{"AcroForm", RootAcroForm},
		{"Metadata", RootMetadata},
	} {
		if err = writeRootEntry(ctx, d, dictName, e.entryName, e.statsAttr); err != nil {
			return err
		}
	}

	if err = writeRootEntryToObjStream(ctx, d, dictName, "StructTreeRoot", RootStructTreeRoot); err != nil {
		return err
	}

	for _, e := range []struct {
		entryName string
		statsAttr int
	}{
		{"MarkInfo", RootMarkInfo},
		{"Lang", RootLang},
		{"SpiderInfo", RootSpiderInfo},
		{"OutputIntents", RootOutputIntents},
		{"PieceInfo", RootPieceInfo},
		{"OCProperties", RootOCProperties},
		{"Perms", RootPerms},
		{"Legal", RootLegal},
		{"Requirements", RootRequirements},
		{"Collection", RootCollection},
		{"NeedsRendering", RootNeedsRendering},
	} {
		if err = writeRootEntry(ctx, d, dictName, e.entryName, e.statsAttr); err != nil {
			return err
		}
	}

	log.Write.Printf("*** writeRootObject: end offset=%d ***\n", ctx.Write.Offset)

	return nil
}

func writeTrailerDict(ctx *Context) error {

	log.Write.Printf("writeTrailerDict begin\n")

	w := ctx.Write
	xRefTable := ctx.XRefTable

	if _, err := w.WriteString("trailer"); err != nil {
		return err
	}

	if err := w.WriteEol(); err != nil {
		return err
	}

	d := NewDict()
	d.Insert("Size", Integer(*xRefTable.Size))
	d.Insert("Root", *xRefTable.Root)

	if xRefTable.Info != nil {
		d.Insert("Info", *xRefTable.Info)
	}

	if ctx.Encrypt != nil && ctx.EncKey != nil {
		d.Insert("Encrypt", *ctx.Encrypt)
	}

	if xRefTable.ID != nil {
		d.Insert("ID", xRefTable.ID)
	}

	if ctx.Write.Increment {
		d.Insert("Prev", Integer(*ctx.Write.OffsetPrevXRef))
	}

	if _, err := w.WriteString(d.PDFString()); err != nil {
		return err
	}

	log.Write.Printf("writeTrailerDict end\n")

	return nil
}

func writeXRefSubsection(ctx *Context, start int, size int) error {

	log.Write.Printf("writeXRefSubsection: start=%d size=%d\n", start, size)

	w := ctx.Write

	if _, err := w.WriteString(fmt.Sprintf("%d %d%s", start, size, w.Eol)); err != nil {
		return err
	}

	var lines []string

	for i := start; i < start+size; i++ {

		entry := ctx.XRefTable.Table[i]

		if entry.Compressed {
			return errors.New("pdfcpu: writeXRefSubsection: compressed entries present")
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

		if _, err := w.WriteString(s); err != nil {
			return err
		}
	}

	log.Write.Printf("\n%s\n", strings.Join(lines, ""))
	log.Write.Printf("writeXRefSubsection: end\n")

	return nil
}

func deleteRedundantObject(ctx *Context, objNr int) {

	if len(ctx.Write.SelectedPages) == 0 &&
		(ctx.Optimize.IsDuplicateFontObject(objNr) || ctx.Optimize.IsDuplicateImageObject(objNr)) {
		ctx.turnEntryToFree(objNr)
	}

	if ctx.IsLinearizationObject(objNr) || ctx.Optimize.IsDuplicateInfoObject(objNr) ||
		ctx.Read.IsObjectStreamObject(objNr) || ctx.Read.IsXRefStreamObject(objNr) {
		ctx.turnEntryToFree(objNr)
	}

}
func deleteRedundantObjects(ctx *Context) {

	if ctx.Optimize == nil {
		return
	}

	xRefTable := ctx.XRefTable

	log.Write.Printf("deleteRedundantObjects begin: Size=%d\n", *xRefTable.Size)

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
			log.Write.Printf("deleteRedundantObjects: remove duplicate obj #%d\n", i)
			delete(ctx.Optimize.DuplicateFontObjs, i)
			delete(ctx.Optimize.DuplicateImageObjs, i)
			delete(ctx.Optimize.DuplicateInfoObjects, i)
			continue
		}

		// Object not written

		if ctx.Read.Linearized && entry.Offset != nil {
			// This block applies to pre existing objects only.
			// Since there is no type entry for stream dicts associated with linearization dicts
			// we have to check every StreamDict that has not been written.
			if _, ok := entry.Object.(StreamDict); ok {

				if *entry.Offset == *xRefTable.OffsetPrimaryHintTable {
					xRefTable.LinearizationObjs[i] = true
					log.Write.Printf("deleteRedundantObjects: primaryHintTable at obj #%d\n", i)
				}

				if xRefTable.OffsetOverflowHintTable != nil &&
					*entry.Offset == *xRefTable.OffsetOverflowHintTable {
					xRefTable.LinearizationObjs[i] = true
					log.Write.Printf("deleteRedundantObjects: overflowHintTable at obj #%d\n", i)
				}

			}

		}

		deleteRedundantObject(ctx, i)

	}

	log.Write.Println("deleteRedundantObjects end")
}

func sortedWritableKeys(ctx *Context) []int {

	var keys []int

	for i, e := range ctx.Table {
		if !ctx.Write.Increment && e.Free || ctx.Write.HasWriteOffset(i) {
			keys = append(keys, i)
		}
	}

	sort.Ints(keys)

	return keys
}

// After inserting the last object write the cross reference table to disk.
func writeXRefTable(ctx *Context) error {

	keys := sortedWritableKeys(ctx)

	objCount := len(keys)
	log.Write.Printf("xref has %d entries\n", objCount)

	if _, err := ctx.Write.WriteString("xref"); err != nil {
		return err
	}

	if err := ctx.Write.WriteEol(); err != nil {
		return err
	}

	start := keys[0]
	size := 1

	for i := 1; i < len(keys); i++ {

		if keys[i]-keys[i-1] > 1 {

			if err := writeXRefSubsection(ctx, start, size); err != nil {
				return err
			}

			start = keys[i]
			size = 1
			continue
		}

		size++
	}

	if err := writeXRefSubsection(ctx, start, size); err != nil {
		return err
	}

	if err := writeTrailerDict(ctx); err != nil {
		return err
	}

	if err := ctx.Write.WriteEol(); err != nil {
		return err
	}

	if _, err := ctx.Write.WriteString("startxref"); err != nil {
		return err
	}

	if err := ctx.Write.WriteEol(); err != nil {
		return err
	}

	if _, err := ctx.Write.WriteString(fmt.Sprintf("%d", ctx.Write.Offset)); err != nil {
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

func createXRefStream(ctx *Context, i1, i2, i3 int, objNrs []int) ([]byte, *Array, error) {

	log.Write.Println("createXRefStream begin")

	xRefTable := ctx.XRefTable

	var (
		buf []byte
		a   Array
	)

	objCount := len(objNrs)
	log.Write.Printf("createXRefStream: xref has %d entries\n", objCount)

	start := objNrs[0]
	size := 0

	for i := 0; i < len(objNrs); i++ {

		j := objNrs[i]
		entry := xRefTable.Table[j]
		var s1, s2, s3 []byte

		if entry.Free {

			// unused
			log.Write.Printf("createXRefStream: unused i=%d nextFreeAt:%d gen:%d\n", j, int(*entry.Offset), int(*entry.Generation))

			s1 = int64ToBuf(0, i1)
			s2 = int64ToBuf(*entry.Offset, i2)
			s3 = int64ToBuf(int64(*entry.Generation), i3)

		} else if entry.Compressed {

			// in use, compressed into object stream
			log.Write.Printf("createXRefStream: compressed i=%d at objstr %d[%d]\n", j, int(*entry.ObjectStream), int(*entry.ObjectStreamInd))

			s1 = int64ToBuf(2, i1)
			s2 = int64ToBuf(int64(*entry.ObjectStream), i2)
			s3 = int64ToBuf(int64(*entry.ObjectStreamInd), i3)

		} else {

			off, found := ctx.Write.Table[j]
			if !found {
				return nil, nil, errors.Errorf("pdfcpu: createXRefStream: missing write offset for obj #%d\n", i)
			}

			// in use, uncompressed
			log.Write.Printf("createXRefStream: used i=%d offset:%d gen:%d\n", j, int(off), int(*entry.Generation))

			s1 = int64ToBuf(1, i1)
			s2 = int64ToBuf(off, i2)
			s3 = int64ToBuf(int64(*entry.Generation), i3)

		}

		log.Write.Printf("createXRefStream: written: %x %x %x \n", s1, s2, s3)

		buf = append(buf, s1...)
		buf = append(buf, s2...)
		buf = append(buf, s3...)

		if i > 0 && (objNrs[i]-objNrs[i-1] > 1) {

			a = append(a, Integer(start))
			a = append(a, Integer(size))

			start = objNrs[i]
			size = 1
			continue
		}

		size++
	}

	a = append(a, Integer(start))
	a = append(a, Integer(size))

	log.Write.Println("createXRefStream end")

	return buf, &a, nil
}

func writeXRefStream(ctx *Context) error {

	log.Write.Println("writeXRefStream begin")

	xRefTable := ctx.XRefTable
	xRefStreamDict := NewXRefStreamDict(ctx)
	xRefTableEntry := NewXRefTableEntryGen0(*xRefStreamDict)

	// Reuse free objects (including recycled objects from this run).
	objNumber, err := xRefTable.InsertAndUseRecycled(*xRefTableEntry)
	if err != nil {
		return err
	}

	xRefStreamDict.Insert("Size", Integer(*xRefTable.Size))

	// Include xref stream dict obj within xref stream dict.
	offset := ctx.Write.Offset
	ctx.Write.SetWriteOffset(objNumber)

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

	wArr := Array{Integer(i1), Integer(i2), Integer(i3)}
	xRefStreamDict.Insert("W", wArr)

	// Generate xRefStreamDict data = xref entries -> xRefStreamDict.Content
	objNrs := sortedWritableKeys(ctx)
	content, indArr, err := createXRefStream(ctx, i1, i2, i3, objNrs)
	if err != nil {
		return err
	}

	xRefStreamDict.Content = content
	xRefStreamDict.Insert("Index", *indArr)

	// Encode xRefStreamDict.Content -> xRefStreamDict.Raw
	if err = xRefStreamDict.StreamDict.Encode(); err != nil {
		return err
	}

	log.Write.Printf("writeXRefStream: xRefStreamDict: %s\n", xRefStreamDict)

	if err = writeStreamDictObject(ctx, objNumber, 0, xRefStreamDict.StreamDict); err != nil {
		return err
	}

	w := ctx.Write

	if _, err = w.WriteString("startxref"); err != nil {
		return err
	}

	if err = w.WriteEol(); err != nil {
		return err
	}

	if _, err = w.WriteString(fmt.Sprintf("%d", offset)); err != nil {
		return err
	}

	if err = w.WriteEol(); err != nil {
		return err
	}

	log.Write.Println("writeXRefStream end")

	return nil
}

func writeEncryptDict(ctx *Context) error {

	// Bail out unless we really have to write encrypted.
	if ctx.Encrypt == nil || ctx.EncKey == nil {
		return nil
	}

	ir := *ctx.Encrypt
	objNumber := int(ir.ObjectNumber)
	genNumber := int(ir.GenerationNumber)

	d, err := ctx.DereferenceDict(ir)
	if err != nil {
		return err
	}

	return writeObject(ctx, objNumber, genNumber, d.PDFString())
}

func setupEncryption(ctx *Context) error {

	var err error

	if ok := validateAlgorithm(ctx); !ok {
		return errors.New("pdfcpu: unsupported encryption algorithm")
	}

	d := newEncryptDict(
		ctx.EncryptUsingAES,
		ctx.EncryptKeyLength,
		ctx.Permissions,
	)

	if ctx.E, err = supportedEncryption(ctx, d); err != nil {
		return err
	}

	if ctx.ID == nil {
		return errors.New("pdfcpu: encrypt: missing ID")
	}

	if ctx.E.ID, err = ctx.IDFirstElement(); err != nil {
		return err
	}

	if err = calcOAndU(ctx, d); err != nil {
		return err
	}

	if err = writePermissions(ctx, d); err != nil {
		return err
	}

	xRefTableEntry := NewXRefTableEntryGen0(d)

	// Reuse free objects (including recycled objects from this run).
	objNumber, err := ctx.InsertAndUseRecycled(*xRefTableEntry)
	if err != nil {
		return err
	}

	ctx.Encrypt = NewIndirectRef(objNumber, 0)

	return nil
}

func updateEncryption(ctx *Context) error {

	d, err := ctx.EncryptDict()
	if err != nil {
		return err
	}

	if ctx.Cmd == SETPERMISSIONS {
		//fmt.Printf("updating permissions to: %v\n", ctx.UserAccessPermissions)
		ctx.E.P = int(ctx.Permissions)
		d.Update("P", Integer(ctx.E.P))
		// and moving on, U is dependent on P
	}

	// ctx.Cmd == CHANGEUPW or CHANGE OPW

	if ctx.UserPWNew != nil {
		//fmt.Printf("change upw from <%s> to <%s>\n", ctx.UserPW, *ctx.UserPWNew)
		ctx.UserPW = *ctx.UserPWNew
	}

	if ctx.OwnerPWNew != nil {
		//fmt.Printf("change opw from <%s> to <%s>\n", ctx.OwnerPW, *ctx.OwnerPWNew)
		ctx.OwnerPW = *ctx.OwnerPWNew
	}

	if ctx.E.R == 5 {

		if err = calcOAndU(ctx, d); err != nil {
			return err
		}

		return writePermissions(ctx, d)
	}

	//fmt.Printf("opw before: length:%d <%s>\n", len(ctx.E.O), ctx.E.O)
	if ctx.E.O, err = o(ctx); err != nil {
		return err
	}
	//fmt.Printf("opw after: length:%d <%s> %0X\n", len(ctx.E.O), ctx.E.O, ctx.E.O)
	d.Update("O", HexLiteral(hex.EncodeToString(ctx.E.O)))

	//fmt.Printf("upw before: length:%d <%s>\n", len(ctx.E.U), ctx.E.U)
	if ctx.E.U, ctx.EncKey, err = u(ctx); err != nil {
		return err
	}
	//fmt.Printf("upw after: length:%d <%s> %0X\n", len(ctx.E.U), ctx.E.U, ctx.E.U)
	//fmt.Printf("encKey = %0X\n", ctx.EncKey)
	d.Update("U", HexLiteral(hex.EncodeToString(ctx.E.U)))

	return nil
}

func handleEncryption(ctx *Context) error {

	if ctx.Cmd == ENCRYPT || ctx.Cmd == DECRYPT {

		if ctx.Cmd == DECRYPT {

			// Remove encryption.
			ctx.EncKey = nil

		} else {

			if err := setupEncryption(ctx); err != nil {
				return err
			}

			alg := "RC4"
			if ctx.EncryptUsingAES {
				alg = "AES"
			}
			log.CLI.Printf("using %s-%d\n", alg, ctx.EncryptKeyLength)
		}

	} else if ctx.UserPWNew != nil || ctx.OwnerPWNew != nil || ctx.Cmd == SETPERMISSIONS {

		if err := updateEncryption(ctx); err != nil {
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

func writeXRef(ctx *Context) error {

	if ctx.WriteXRefStream {
		// Write cross reference stream and generate objectstreams.
		return writeXRefStream(ctx)
	}

	// Write cross reference table section.
	return writeXRefTable(ctx)
}

func setFileSizeOfWrittenFile(w *WriteContext) error {
	if err := w.Flush(); err != nil {
		return err
	}

	// If writing is Writer based then f is nil.
	if w.Fp == nil {
		return nil
	}

	fileInfo, err := w.Fp.Stat()
	if err != nil {
		return err
	}

	w.FileSize = fileInfo.Size()

	return nil
}
