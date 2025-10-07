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

	"github.com/angel-one/pdfcpu/pkg/filter"
	"github.com/angel-one/pdfcpu/pkg/log"
	"github.com/angel-one/pdfcpu/pkg/pdfcpu/model"
	"github.com/angel-one/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

func writeObjects(ctx *model.Context) error {
	// Write root object(aka the document catalog) and page tree.
	if err := writeRootObject(ctx); err != nil {
		return err
	}

	if log.WriteEnabled() {
		log.Write.Printf("offset after writeRootObject: %d\n", ctx.Write.Offset)
	}

	// Write document information dictionary.
	if err := writeDocumentInfoDict(ctx); err != nil {
		return err
	}

	if log.WriteEnabled() {
		log.Write.Printf("offset after writeInfoObject: %d\n", ctx.Write.Offset)
	}

	// Write offspec additional streams as declared in pdf trailer.
	if err := writeAdditionalStreams(ctx); err != nil {
		return err
	}

	return writeEncryptDict(ctx)
}

// WriteContext generates a PDF file for the cross reference table contained in Context.
func WriteContext(ctx *model.Context) (err error) {
	// Create a writer for dirname and filename if not already supplied.
	if ctx.Write.Writer == nil {

		fileName := filepath.Join(ctx.Write.DirName, ctx.Write.FileName)
		if log.CLIEnabled() {
			log.CLI.Printf("writing to %s\n", fileName)
		}

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

	// if exists metadata, update from info dict
	// else if v2 create from scratch
	// else nothing just write info dict

	// We support PDF Collections (since V1.7) for file attachments
	v := model.V17

	if ctx.XRefTable.Version() == model.V20 {
		v = model.V20
	}

	if err = writeHeader(ctx.Write, v); err != nil {
		return err
	}

	// Ensure there is no root version.
	if ctx.RootVersion != nil {
		ctx.RootDict.Delete("Version")
	}

	if log.WriteEnabled() {
		log.Write.Printf("offset after writeHeader: %d\n", ctx.Write.Offset)
	}

	if err := writeObjects(ctx); err != nil {
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
func WriteIncrement(ctx *model.Context) error {
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

func prepareContextForWriting(ctx *model.Context) error {
	if err := ensureInfoDictAndFileID(ctx); err != nil {
		return err
	}

	return handleEncryption(ctx)
}

func writeAdditionalStreams(ctx *model.Context) error {
	if ctx.AdditionalStreams == nil {
		return nil
	}

	if _, _, err := writeDeepObject(ctx, ctx.AdditionalStreams); err != nil {
		return err
	}

	return nil
}

func ensureFileID(ctx *model.Context) error {
	fid, err := fileID(ctx)
	if err != nil {
		return err
	}

	if ctx.ID == nil {
		// Ensure ctx.ID
		ctx.ID = types.Array{fid, fid}
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

func ensureInfoDictAndFileID(ctx *model.Context) error {
	if ctx.XRefTable.Version() < model.V20 {
		if err := ensureInfoDict(ctx); err != nil {
			return err
		}
	}

	return ensureFileID(ctx)
}

// Write root entry to disk.
func writeRootEntry(ctx *model.Context, d types.Dict, dictName, entryName string, statsAttr int) error {
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
func writeRootEntryToObjStream(ctx *model.Context, d types.Dict, dictName, entryName string, statsAttr int) error {
	ctx.Write.WriteToObjectStream = true

	if err := writeRootEntry(ctx, d, dictName, entryName, statsAttr); err != nil {
		return err
	}

	return stopObjectStream(ctx)
}

// Write page tree.
func writePages(ctx *model.Context, rootDict types.Dict) error {
	// Page tree root (the top "Pages" dict) must be indirect reference.
	indRef := rootDict.IndirectRefEntry("Pages")
	if indRef == nil {
		return errors.New("pdfcpu: writePages: missing indirect obj for pages dict")
	}

	// Embed all page tree objects into objects stream.
	ctx.Write.WriteToObjectStream = true

	// Write page tree.
	p := 0
	if _, _, err := writePagesDict(ctx, indRef, &p); err != nil {
		return err
	}

	return stopObjectStream(ctx)
}

func writeRootAttrsBatch1(ctx *model.Context, d types.Dict, dictName string) error {

	if err := writeAcroFormRootEntry(ctx, d, dictName); err != nil {
		return err
	}

	for _, e := range []struct {
		entryName string
		statsAttr int
	}{
		{"Extensions", model.RootExtensions},
		{"PageLabels", model.RootPageLabels},
		{"Names", model.RootNames},
		{"Dests", model.RootDests},
		{"ViewerPreferences", model.RootViewerPrefs},
		{"PageLayout", model.RootPageLayout},
		{"PageMode", model.RootPageMode},
		{"Outlines", model.RootOutlines},
		{"Threads", model.RootThreads},
		{"OpenAction", model.RootOpenAction},
		{"AA", model.RootAA},
		{"URI", model.RootURI},
		//{"AcroForm", model.RootAcroForm},
		{"Metadata", model.RootMetadata},
	} {
		if err := writeRootEntry(ctx, d, dictName, e.entryName, e.statsAttr); err != nil {
			return err
		}
	}

	return nil
}

func writeRootAttrsBatch2(ctx *model.Context, d types.Dict, dictName string) error {
	for _, e := range []struct {
		entryName string
		statsAttr int
	}{
		{"MarkInfo", model.RootMarkInfo},
		{"Lang", model.RootLang},
		{"SpiderInfo", model.RootSpiderInfo},
		{"OutputIntents", model.RootOutputIntents},
		{"PieceInfo", model.RootPieceInfo},
		{"OCProperties", model.RootOCProperties},
		{"Perms", model.RootPerms},
		{"Legal", model.RootLegal},
		{"Requirements", model.RootRequirements},
		{"Collection", model.RootCollection},
		{"NeedsRendering", model.RootNeedsRendering},
	} {
		if err := writeRootEntry(ctx, d, dictName, e.entryName, e.statsAttr); err != nil {
			return err
		}
	}

	return nil
}

func writeRootObject(ctx *model.Context) error {
	// => 7.7.2 Document Catalog

	xRefTable := ctx.XRefTable
	catalog := *xRefTable.Root
	objNumber := int(catalog.ObjectNumber)
	genNumber := int(catalog.GenerationNumber)

	if log.WriteEnabled() {
		log.Write.Printf("*** writeRootObject: begin offset=%d *** %s\n", ctx.Write.Offset, catalog)
	}

	// Ensure corresponding and accurate name tree object graphs.
	if !ctx.ApplyReducedFeatureSet() {
		if err := ctx.BindNameTrees(); err != nil {
			return err
		}
	}

	d, err := xRefTable.DereferenceDict(catalog)
	if err != nil {
		return err
	}

	if d == nil {
		return errors.Errorf("pdfcpu: writeRootObject: unable to dereference root dict")
	}

	dictName := "rootDict"

	if ctx.ApplyReducedFeatureSet() {
		log.Write.Println("writeRootObject - reducedFeatureSet:exclude complex entries.")
		d.Delete("Names")
		d.Delete("Dests")
		d.Delete("Outlines")
		d.Delete("OpenAction")
		d.Delete("StructTreeRoot")
		d.Delete("OCProperties")
	}

	if err = writeDictObject(ctx, objNumber, genNumber, d); err != nil {
		return err
	}

	if log.WriteEnabled() {
		log.Write.Printf("writeRootObject: %s\n", d)
		log.Write.Printf("writeRootObject: new offset after rootDict = %d\n", ctx.Write.Offset)
	}

	if err = writeRootEntry(ctx, d, dictName, "Version", model.RootVersion); err != nil {
		return err
	}

	if err = writePages(ctx, d); err != nil {
		return err
	}

	if err := writeRootAttrsBatch1(ctx, d, dictName); err != nil {
		return err
	}

	if err = writeRootEntryToObjStream(ctx, d, dictName, "StructTreeRoot", model.RootStructTreeRoot); err != nil {
		return err
	}

	if err := writeRootAttrsBatch2(ctx, d, dictName); err != nil {
		return err
	}

	if log.WriteEnabled() {
		log.Write.Printf("*** writeRootObject: end offset=%d ***\n", ctx.Write.Offset)
	}

	return nil
}

func writeTrailerDict(ctx *model.Context) error {
	if log.WriteEnabled() {
		log.Write.Printf("writeTrailerDict begin\n")
	}

	w := ctx.Write
	xRefTable := ctx.XRefTable

	if _, err := w.WriteString("trailer"); err != nil {
		return err
	}

	if err := w.WriteEol(); err != nil {
		return err
	}

	d := types.NewDict()
	d.Insert("Size", types.Integer(*xRefTable.Size))
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
		d.Insert("Prev", types.Integer(*ctx.Write.OffsetPrevXRef))
	}

	if _, err := w.WriteString(d.PDFString()); err != nil {
		return err
	}

	if log.WriteEnabled() {
		log.Write.Printf("writeTrailerDict end\n")
	}

	return nil
}

func writeXRefSubsection(ctx *model.Context, start int, size int) error {
	if log.WriteEnabled() {
		log.Write.Printf("writeXRefSubsection: start=%d size=%d\n", start, size)
	}

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

	if log.WriteEnabled() {
		log.Write.Printf("\n%s\n", strings.Join(lines, ""))
		log.Write.Printf("writeXRefSubsection: end\n")
	}

	return nil
}

func deleteRedundantObject(ctx *model.Context, objNr int) {
	if len(ctx.Write.SelectedPages) == 0 &&
		(ctx.Optimize.IsDuplicateFontObject(objNr) || ctx.Optimize.IsDuplicateImageObject(objNr)) {
		ctx.FreeObject(objNr)
	}

	if ctx.IsLinearizationObject(objNr) || ctx.Optimize.IsDuplicateInfoObject(objNr) ||
		ctx.Read.IsObjectStreamObject(objNr) {
		ctx.FreeObject(objNr)
	}

}

func detectLinearizationObjs(xRefTable *model.XRefTable, entry *model.XRefTableEntry, i int) {
	if _, ok := entry.Object.(types.StreamDict); ok {

		if *entry.Offset == *xRefTable.OffsetPrimaryHintTable {
			xRefTable.LinearizationObjs[i] = true
			if log.WriteEnabled() {
				log.Write.Printf("detectLinearizationObjs: primaryHintTable at obj #%d\n", i)
			}
		}

		if xRefTable.OffsetOverflowHintTable != nil &&
			*entry.Offset == *xRefTable.OffsetOverflowHintTable {
			xRefTable.LinearizationObjs[i] = true
			if log.WriteEnabled() {
				log.Write.Printf("detectLinearizationObjs: overflowHintTable at obj #%d\n", i)
			}
		}

	}
}

func deleteRedundantObjects(ctx *model.Context) {
	if ctx.Optimize == nil {
		return
	}

	xRefTable := ctx.XRefTable

	if log.WriteEnabled() {
		log.Write.Printf("deleteRedundantObjects begin: Size=%d\n", *xRefTable.Size)
	}

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
			if log.WriteEnabled() {
				log.Write.Printf("deleteRedundantObjects: remove duplicate obj #%d\n", i)
			}
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
			detectLinearizationObjs(xRefTable, entry, i)
		}

		deleteRedundantObject(ctx, i)
	}

	if log.WriteEnabled() {
		log.Write.Println("deleteRedundantObjects end")
	}
}

func sortedWritableKeys(ctx *model.Context) []int {
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
func writeXRefTable(ctx *model.Context) error {
	keys := sortedWritableKeys(ctx)

	objCount := len(keys)
	if log.WriteEnabled() {
		log.Write.Printf("xref has %d entries\n", objCount)
	}

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

func createXRefStream(ctx *model.Context, i1, i2, i3 int, objNrs []int) ([]byte, *types.Array, error) {
	if log.WriteEnabled() {
		log.Write.Println("createXRefStream begin")
	}

	xRefTable := ctx.XRefTable

	var (
		buf []byte
		a   types.Array
	)

	objCount := len(objNrs)
	if log.WriteEnabled() {
		log.Write.Printf("createXRefStream: xref has %d entries\n", objCount)
	}

	start := objNrs[0]
	size := 0

	for i := 0; i < len(objNrs); i++ {

		j := objNrs[i]
		entry := xRefTable.Table[j]
		var s1, s2, s3 []byte

		if entry.Free {

			// unused
			if log.WriteEnabled() {
				log.Write.Printf("createXRefStream: unused i=%d nextFreeAt:%d gen:%d\n", j, int(*entry.Offset), int(*entry.Generation))
			}

			s1 = int64ToBuf(0, i1)
			s2 = int64ToBuf(*entry.Offset, i2)
			s3 = int64ToBuf(int64(*entry.Generation), i3)

		} else if entry.Compressed {

			// in use, compressed into object stream
			if log.WriteEnabled() {
				log.Write.Printf("createXRefStream: compressed i=%d at objstr %d[%d]\n", j, int(*entry.ObjectStream), int(*entry.ObjectStreamInd))
			}

			s1 = int64ToBuf(2, i1)
			s2 = int64ToBuf(int64(*entry.ObjectStream), i2)
			s3 = int64ToBuf(int64(*entry.ObjectStreamInd), i3)

		} else {

			off, found := ctx.Write.Table[j]
			if !found {
				return nil, nil, errors.Errorf("pdfcpu: createXRefStream: missing write offset for obj #%d\n", i)
			}

			// in use, uncompressed
			if log.WriteEnabled() {
				log.Write.Printf("createXRefStream: used i=%d offset:%d gen:%d\n", j, int(off), int(*entry.Generation))
			}

			s1 = int64ToBuf(1, i1)
			s2 = int64ToBuf(off, i2)
			s3 = int64ToBuf(int64(*entry.Generation), i3)

		}

		if log.WriteEnabled() {
			log.Write.Printf("createXRefStream: written: %x %x %x \n", s1, s2, s3)
		}

		buf = append(buf, s1...)
		buf = append(buf, s2...)
		buf = append(buf, s3...)

		if i > 0 && (objNrs[i]-objNrs[i-1] > 1) {

			a = append(a, types.Integer(start))
			a = append(a, types.Integer(size))

			start = objNrs[i]
			size = 1
			continue
		}

		size++
	}

	a = append(a, types.Integer(start))
	a = append(a, types.Integer(size))

	if log.WriteEnabled() {
		log.Write.Println("createXRefStream end")
	}

	return buf, &a, nil
}

// NewXRefStreamDict creates a new PDFXRefStreamDict object.
func newXRefStreamDict(ctx *model.Context) *types.XRefStreamDict {
	sd := types.StreamDict{Dict: types.NewDict()}
	sd.Insert("Type", types.Name("XRef"))
	sd.Insert("Filter", types.Name(filter.Flate))
	sd.FilterPipeline = []types.PDFFilter{{Name: filter.Flate, DecodeParms: nil}}
	sd.Insert("Root", *ctx.Root)
	if ctx.Info != nil {
		sd.Insert("Info", *ctx.Info)
	}
	if ctx.ID != nil {
		sd.Insert("ID", ctx.ID)
	}
	if ctx.Encrypt != nil && ctx.EncKey != nil {
		sd.Insert("Encrypt", *ctx.Encrypt)
	}
	if ctx.Write.Increment {
		sd.Insert("Prev", types.Integer(*ctx.Write.OffsetPrevXRef))
	}
	return &types.XRefStreamDict{StreamDict: sd}
}

func writeXRefStream(ctx *model.Context) error {
	if log.WriteEnabled() {
		log.Write.Println("writeXRefStream begin")
	}

	xRefTable := ctx.XRefTable
	xRefStreamDict := newXRefStreamDict(ctx)
	xRefTableEntry := model.NewXRefTableEntryGen0(*xRefStreamDict)

	// Reuse free objects (including recycled objects from this run).
	objNumber, err := xRefTable.InsertAndUseRecycled(*xRefTableEntry)
	if err != nil {
		return err
	}

	xRefStreamDict.Insert("Size", types.Integer(*xRefTable.Size))

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

	wArr := types.Array{types.Integer(i1), types.Integer(i2), types.Integer(i3)}
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

	if log.WriteEnabled() {
		log.Write.Printf("writeXRefStream: xRefStreamDict: %s\n", xRefStreamDict)
	}

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

	if log.WriteEnabled() {
		log.Write.Println("writeXRefStream end")
	}

	return nil
}

func writeEncryptDict(ctx *model.Context) error {
	// Bail out unless we really have to write encrypted.
	if ctx.Encrypt == nil || ctx.EncKey == nil {
		return nil
	}

	indRef := *ctx.Encrypt
	objNumber := int(indRef.ObjectNumber)
	genNumber := int(indRef.GenerationNumber)

	d, err := ctx.DereferenceDict(indRef)
	if err != nil {
		return err
	}

	return writeObject(ctx, objNumber, genNumber, d.PDFString())
}

func setupEncryption(ctx *model.Context) error {
	var err error

	if ok := validateAlgorithm(ctx); !ok {
		return errors.New("pdfcpu: unsupported encryption algorithm (PDF 2.0 assumes AES/256)")
	}

	d := newEncryptDict(
		ctx.XRefTable.Version(),
		ctx.EncryptUsingAES,
		ctx.EncryptKeyLength,
		int16(ctx.Permissions),
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

	xRefTableEntry := model.NewXRefTableEntryGen0(d)

	// Reuse free objects (including recycled objects from this run).
	objNumber, err := ctx.InsertAndUseRecycled(*xRefTableEntry)
	if err != nil {
		return err
	}

	ctx.Encrypt = types.NewIndirectRef(objNumber, 0)

	return nil
}

func updateEncryption(ctx *model.Context) error {
	if ctx.Encrypt == nil {
		return errors.New("pdfcpu: This file is not encrypted - nothing written.")
	}

	d, err := ctx.EncryptDict()
	if err != nil {
		return err
	}

	if ctx.Cmd == model.SETPERMISSIONS {
		//fmt.Printf("updating permissions to: %v\n", ctx.UserAccessPermissions)
		ctx.E.P = int(ctx.Permissions)
		d.Update("P", types.Integer(ctx.E.P))
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

	if ctx.E.R == 5 || ctx.E.R == 6 {

		if err = calcOAndU(ctx, d); err != nil {
			return err
		}

		// Calc Perms for rev 5, 6.
		return writePermissions(ctx, d)
	}

	//fmt.Printf("opw before: length:%d <%s>\n", len(ctx.E.O), ctx.E.O)
	if ctx.E.O, err = o(ctx); err != nil {
		return err
	}
	//fmt.Printf("opw after: length:%d <%s> %0X\n", len(ctx.E.O), ctx.E.O, ctx.E.O)
	d.Update("O", types.HexLiteral(hex.EncodeToString(ctx.E.O)))

	//fmt.Printf("upw before: length:%d <%s>\n", len(ctx.E.U), ctx.E.U)
	if ctx.E.U, ctx.EncKey, err = u(ctx); err != nil {
		return err
	}
	//fmt.Printf("upw after: length:%d <%s> %0X\n", len(ctx.E.U), ctx.E.U, ctx.E.U)
	//fmt.Printf("encKey = %0X\n", ctx.EncKey)
	d.Update("U", types.HexLiteral(hex.EncodeToString(ctx.E.U)))

	return nil
}

func handleEncryption(ctx *model.Context) error {

	if ctx.Cmd == model.ENCRYPT || ctx.Cmd == model.DECRYPT {

		if ctx.Cmd == model.DECRYPT {

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
			if log.CLIEnabled() {
				log.CLI.Printf("using %s-%d\n", alg, ctx.EncryptKeyLength)
			}
		}

	} else if ctx.UserPWNew != nil || ctx.OwnerPWNew != nil || ctx.Cmd == model.SETPERMISSIONS {

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

func writeXRef(ctx *model.Context) error {
	if ctx.WriteXRefStream {
		// Write cross reference stream and generate objectstreams.
		return writeXRefStream(ctx)
	}

	// Write cross reference table section.
	return writeXRefTable(ctx)
}

func setFileSizeOfWrittenFile(w *model.WriteContext) error {
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
