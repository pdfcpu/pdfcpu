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
	"context"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/scan"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

const (
	defaultBufSize = 1024
	maximumBufSize = 1024 * 1024
)

var (
	ErrWrongPassword         = errors.New("pdfcpu: please provide the correct password")
	ErrCorruptHeader         = errors.New("pdfcpu: no header version available")
	ErrReferenceDoesNotExist = errors.New("pdfcpu: referenced object does not exist")

	zero int64 = 0
)

// ReadFile reads in a PDF file and builds an internal structure holding its cross reference table aka the PDF model context.
func ReadFile(inFile string, conf *model.Configuration) (*model.Context, error) {
	return ReadFileWithContext(context.Background(), inFile, conf)
}

// ReadFileContext reads in a PDF file and builds an internal structure holding its cross reference table aka the PDF model context.
// If the passed Go context is cancelled, reading will be interrupted.
func ReadFileWithContext(c context.Context, inFile string, conf *model.Configuration) (*model.Context, error) {
	if log.InfoEnabled() {
		log.Info.Printf("reading %s..\n", inFile)
	}

	f, err := os.Open(inFile)
	if err != nil {
		return nil, errors.Wrapf(err, "can't open %q", inFile)
	}

	defer func() {
		f.Close()
	}()

	return ReadWithContext(c, f, conf)
}

// Read takes a readSeeker and generates a PDF model context,
// an in-memory representation containing a cross reference table.
func Read(rs io.ReadSeeker, conf *model.Configuration) (*model.Context, error) {
	return ReadWithContext(context.Background(), rs, conf)
}

// Read takes a readSeeker and generates a PDF model context,
// an in-memory representation containing a cross reference table.
// If the passed Go context is cancelled, reading will be interrupted.
func ReadWithContext(c context.Context, rs io.ReadSeeker, conf *model.Configuration) (*model.Context, error) {
	if log.ReadEnabled() {
		log.Read.Println("Read: begin")
	}

	ctx, err := model.NewContext(rs, conf)
	if err != nil {
		return nil, err
	}

	if ctx.Read.FileSize == 0 {
		return nil, errors.New("The file could not be opened because it is empty.")
	}

	if log.InfoEnabled() {
		if ctx.Reader15 {
			log.Info.Println("PDF Version 1.5 conforming reader")
		} else {
			log.Info.Println("PDF Version 1.4 conforming reader - no object streams or xrefstreams allowed")
		}
	}

	// Populate xRefTable.
	if err = readXRefTable(c, ctx); err != nil {
		return nil, errors.Wrap(err, "Read: xRefTable failed")
	}

	// Make all objects explicitly available (load into memory) in corresponding xRefTable entries.
	// Also decode any involved object streams.
	if err = dereferenceXRefTable(c, ctx, conf); err != nil {
		return nil, err
	}

	// Some PDFWriters write an incorrect Size into trailer.
	if *ctx.XRefTable.Size != ctx.MaxObjNr+1 {
		*ctx.XRefTable.Size = ctx.MaxObjNr + 1
		model.ShowRepaired("trailer size")
	}

	if log.ReadEnabled() {
		log.Read.Println("Read: end")
	}

	return ctx, nil
}

// fillBuffer reads from r until buf is full or read returns an error.
// Unlike io.ReadAtLeast fillBuffer does not return ErrUnexpectedEOF
// if an EOF happens after reading some but not all the bytes.
// Special thanks go to Rene Kaufmann.
func fillBuffer(r io.Reader, buf []byte) (int, error) {
	var n int
	var err error

	for n < len(buf) && err == nil {
		var nn int
		nn, err = r.Read(buf[n:])
		n += nn
	}

	if n > 0 && err == io.EOF {
		return n, nil
	}

	return n, err
}

func newPositionedReader(rs io.ReadSeeker, offset *int64) (*bufio.Reader, error) {
	if _, err := rs.Seek(*offset, io.SeekStart); err != nil {
		return nil, err
	}

	if log.ReadEnabled() {
		log.Read.Printf("newPositionedReader: positioned to offset: %d\n", *offset)
	}

	return bufio.NewReader(rs), nil
}

// Get the file offset of the last XRefSection.
// Go to end of file and search backwards for the first occurrence of startxref {offset} %%EOF
func offsetLastXRefSection(ctx *model.Context, skip int64) (*int64, error) {
	rs := ctx.Read.RS

	var (
		prevBuf, workBuf []byte
		bufSize          int64 = 512
		offset           int64
	)

	if ctx.Read.FileSize < bufSize {
		bufSize = ctx.Read.FileSize
	}

	for i := 1; offset == 0; i++ {

		off, err := rs.Seek(-int64(i)*bufSize-skip, io.SeekEnd)
		if err != nil {
			return nil, errors.New("the file may be damaged.")
		}

		if log.ReadEnabled() {
			log.Read.Printf("scanning for offsetLastXRefSection starting at %d\n", off)
		}

		curBuf := make([]byte, bufSize)

		if _, err = fillBuffer(rs, curBuf); err != nil {
			return nil, err
		}

		workBuf = curBuf
		if prevBuf != nil {
			workBuf = append(curBuf, prevBuf...)
		}

		j := strings.LastIndex(string(workBuf), "startxref")
		if j == -1 {
			prevBuf = curBuf
			continue
		}

		p := workBuf[j+len("startxref")+1:]
		posEOF := strings.Index(string(p), "%%EOF")
		if posEOF == -1 {
			return nil, errors.New("pdfcpu: no matching %%EOF for startxref")
		}

		p = p[:posEOF]
		offset, err = strconv.ParseInt(strings.TrimSpace(string(p)), 10, 64)
		if err != nil {
			return nil, errors.New("pdfcpu: corrupted last xref section")
		}
		if offset >= ctx.Read.FileSize {
			offset = 0
		}
	}

	if log.ReadEnabled() {
		log.Read.Printf("Offset last xrefsection: %d\n", offset)
	}

	return &offset, nil
}

func createXRefTableEntry(entryType string, objNr int, offset, offExtra int64, generation int) (model.XRefTableEntry, bool) {
	entry := model.XRefTableEntry{Offset: &offset, Generation: &generation}

	if entryType == "n" {

		// in use object

		if log.ReadEnabled() {
			log.Read.Printf("createXRefTableEntry: Object #%d is in use at offset=%d, generation=%d\n", objNr, offset, generation)
		}

		if offset == 0 {
			if objNr == 0 {
				entry.Free = true
				model.ShowRepaired("obj#0")
				return entry, true
			}
			if log.InfoEnabled() {
				log.Info.Printf("createXRefTableEntry: Skip entry for in use object #%d with offset 0\n", objNr)
			}
			return entry, false
		}

		*entry.Offset += offExtra

		return entry, true
	}

	// free object

	if log.ReadEnabled() {
		log.Read.Printf("createXRefTableEntry: Object #%d is unused, next free is object#%d, generation=%d\n", objNr, offset, generation)
	}

	entry.Free = true

	return entry, true
}

func decodeSubsection(fields []string, repairOff int) (int64, int, string, error) {
	offset, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return 0, 0, "", err
	}
	offset += int64(repairOff)

	generation, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, 0, "", err
	}

	entryType := fields[2]
	if entryType != "f" && entryType != "n" {
		return 0, 0, "", errors.New("pdfcpu: decodeSubsection: corrupt xref subsection entryType")
	}

	return offset, generation, entryType, nil
}

// Read next subsection entry and generate corresponding xref table entry.
func parseXRefTableEntry(xRefTable *model.XRefTable, s *bufio.Scanner, objNr int, offExtra int64, repairOff int) error {
	if log.ReadEnabled() {
		log.Read.Println("parseXRefTableEntry: begin")
	}

	line, err := scanLine(s)
	if err != nil {
		return err
	}

	if xRefTable.Exists(objNr) {
		if log.ReadEnabled() {
			log.Read.Printf("parseXRefTableEntry: end - Skip entry %d - already assigned\n", objNr)
		}
		return nil
	}

	fields := strings.Fields(line)
	if len(fields) != 3 ||
		len(fields[0]) != 10 || len(fields[1]) != 5 || len(fields[2]) != 1 {
		return errors.New("pdfcpu: parseXRefTableEntry: corrupt xref subsection header")
	}

	offset, generation, entryType, err := decodeSubsection(fields, repairOff)
	if err != nil {
		return err
	}

	entry, ok := createXRefTableEntry(entryType, objNr, offset, offExtra, generation)
	if !ok {
		return nil
	}

	if log.ReadEnabled() {
		log.Read.Printf("parseXRefTableEntry: Insert new xreftable entry for Object %d\n", objNr)
	}

	xRefTable.Table[objNr] = &entry

	if log.ReadEnabled() {
		log.Read.Println("parseXRefTableEntry: end")
	}

	return nil
}

// Process xRef table subsection and create corresponding xRef table entries.
func parseXRefTableSubSection(xRefTable *model.XRefTable, s *bufio.Scanner, fields []string, offExtra int64, repairOff int) error {
	if log.ReadEnabled() {
		log.Read.Println("parseXRefTableSubSection: begin")
	}

	startObjNumber, err := strconv.Atoi(fields[0])
	if err != nil {
		return err
	}

	objCount, err := strconv.Atoi(fields[1])
	if err != nil {
		return err
	}

	if log.ReadEnabled() {
		log.Read.Printf("detected xref subsection, startObj=%d length=%d\n", startObjNumber, objCount)
	}

	// Process all entries of this subsection into xRefTable entries.
	for i := 0; i < objCount; i++ {
		if err = parseXRefTableEntry(xRefTable, s, startObjNumber+i, offExtra, repairOff); err != nil {
			return err
		}
	}

	if log.ReadEnabled() {
		log.Read.Println("parseXRefTableSubSection: end")
	}

	return nil
}

// Parse compressed object.
func compressedObject(c context.Context, s string) (types.Object, error) {
	if log.ReadEnabled() {
		log.Read.Println("compressedObject: begin")
	}

	o, err := model.ParseObjectContext(c, &s)
	if err != nil {
		return nil, err
	}

	d, ok := o.(types.Dict)
	if !ok {
		// return trivial Object: Integer, Array, etc.
		if log.ReadEnabled() {
			log.Read.Println("compressedObject: end, any other than dict")
		}
		return o, nil
	}

	streamLength, streamLengthRef := d.Length()
	if streamLength == nil && streamLengthRef == nil {
		// return Dict
		if log.ReadEnabled() {
			log.Read.Println("compressedObject: end, dict")
		}
		return d, nil
	}

	return nil, errors.New("pdfcpu: compressedObject: stream objects are not to be stored in an object stream")
}

// Parse all objects of an object stream and save them into objectStreamDict.ObjArray.
func parseObjectStream(c context.Context, osd *types.ObjectStreamDict) error {
	if log.ReadEnabled() {
		log.Read.Printf("parseObjectStream begin: decoding %d objects.\n", osd.ObjCount)
	}

	decodedContent := osd.Content
	if decodedContent == nil {
		// The actual content will be decoded lazily, only decode the prolog here.
		var err error
		decodedContent, err = osd.DecodeLength(int64(osd.FirstObjOffset))
		if err != nil {
			return err
		}
	}
	prolog := decodedContent[:osd.FirstObjOffset]

	// The separator used in the prolog shall be white space
	// but some PDF writers use 0x00.
	prolog = bytes.ReplaceAll(prolog, []byte{0x00}, []byte{0x20})

	objs := strings.Fields(string(prolog))
	if len(objs)%2 > 0 {
		return errors.New("pdfcpu: parseObjectStream: corrupt object stream dict")
	}

	// e.g., 10 0 11 25 = 2 Objects: #10 @ offset 0, #11 @ offset 25

	var objArray types.Array

	var offsetOld int

	for i := 0; i < len(objs); i += 2 {

		if err := c.Err(); err != nil {
			return err
		}

		offset, err := strconv.Atoi(objs[i+1])
		if err != nil {
			return err
		}

		offset += osd.FirstObjOffset

		if i > 0 {
			o := types.NewLazyObjectStreamObject(osd, offsetOld, offset, compressedObject)
			objArray = append(objArray, o)
		}

		if i == len(objs)-2 {
			o := types.NewLazyObjectStreamObject(osd, offset, -1, compressedObject)
			objArray = append(objArray, o)
		}

		offsetOld = offset
	}

	osd.ObjArray = objArray

	if log.ReadEnabled() {
		log.Read.Println("parseObjectStream end")
	}

	return nil
}

func createXRefTableEntryFromXRefStream(entryType int64, objNr int, c2, c3, offExtra int64, objStreams types.IntSet) model.XRefTableEntry {
	var xRefTableEntry model.XRefTableEntry

	switch entryType {

	case 0x00:
		// free object
		if log.ReadEnabled() {
			log.Read.Printf("createXRefTableEntryFromXRefStream: Object #%d is unused, next free is object#%d, generation=%d\n", objNr, c2, c3)
		}
		g := int(c3)

		xRefTableEntry =
			model.XRefTableEntry{
				Free:       true,
				Compressed: false,
				Offset:     &c2,
				Generation: &g}

	case 0x01:
		// in use object
		if log.ReadEnabled() {
			log.Read.Printf("createXRefTableEntryFromXRefStream: Object #%d is in use at offset=%d, generation=%d\n", objNr, c2, c3)
		}
		g := int(c3)

		c2 += offExtra

		xRefTableEntry =
			model.XRefTableEntry{
				Free:       false,
				Compressed: false,
				Offset:     &c2,
				Generation: &g}

	case 0x02:
		// compressed object
		// generation always 0.
		if log.ReadEnabled() {
			log.Read.Printf("createXRefTableEntryFromXRefStream: Object #%d is compressed at obj %5d[%d]\n", objNr, c2, c3)
		}
		objNumberRef := int(c2)
		objIndex := int(c3)

		xRefTableEntry =
			model.XRefTableEntry{
				Free:            false,
				Compressed:      true,
				ObjectStream:    &objNumberRef,
				ObjectStreamInd: &objIndex}

		objStreams[objNumberRef] = true
	}

	return xRefTableEntry
}

// For each object embedded in this xRefStream create the corresponding xRef table entry.
func extractXRefTableEntriesFromXRefStream(buf []byte, offExtra int64, xsd *types.XRefStreamDict, ctx *model.Context) error {
	if log.ReadEnabled() {
		log.Read.Printf("extractXRefTableEntriesFromXRefStream begin")
	}

	// Note:
	// A value of zero for an element in the W array indicates that the corresponding field shall not be present in the stream,
	// and the default value shall be used, if there is one.
	// If the first element is zero, the type field shall not be present, and shall default to type 1.

	i1 := xsd.W[0]
	i2 := xsd.W[1]
	i3 := xsd.W[2]

	xrefEntryLen := i1 + i2 + i3
	if log.ReadEnabled() {
		log.Read.Printf("extractXRefTableEntriesFromXRefStream: begin xrefEntryLen = %d\n", xrefEntryLen)
	}

	if xrefEntryLen != 0 && len(buf)%xrefEntryLen > 0 {
		return errors.New("pdfcpu: extractXRefTableEntriesFromXRefStream: corrupt xrefstream")
	}

	objCount := len(xsd.Objects)
	if log.ReadEnabled() {
		log.Read.Printf("extractXRefTableEntriesFromXRefStream: objCount:%d %v\n", objCount, xsd.Objects)
		log.Read.Printf("extractXRefTableEntriesFromXRefStream: len(buf):%d objCount*xrefEntryLen:%d\n", len(buf), objCount*xrefEntryLen)
	}
	if len(buf) < objCount*xrefEntryLen {
		// Sometimes there is an additional xref entry not accounted for by "Index".
		// We ignore such entries and do not treat this as an error.
		return errors.New("pdfcpu: extractXRefTableEntriesFromXRefStream: corrupt xrefstream")
	}

	j := 0

	// bufToInt64 interprets the content of buf as an int64.
	bufToInt64 := func(buf []byte) (i int64) {
		for _, b := range buf {
			i <<= 8
			i |= int64(b)
		}
		return
	}

	for i := 0; i < len(buf) && j < len(xsd.Objects); i += xrefEntryLen {

		objNr := xsd.Objects[j]
		var c1 int64
		if i1 == 0 {
			// If the first element is zero, the type field shall not be present,
			// and shall default to type 1.
			c1 = 1
		} else {
			c1 = bufToInt64(buf[i : i+i1])
		}
		c2 := bufToInt64(buf[i+i1 : i+i1+i2])
		c3 := bufToInt64(buf[i+i1+i2 : i+i1+i2+i3])

		entry := createXRefTableEntryFromXRefStream(c1, objNr, c2, c3, offExtra, ctx.Read.ObjectStreams)

		if ctx.XRefTable.Exists(objNr) {
			if log.ReadEnabled() {
				log.Read.Printf("extractXRefTableEntriesFromXRefStream: Skip entry %d - already assigned\n", objNr)
			}
		} else {
			ctx.Table[objNr] = &entry
		}

		j++
	}

	if log.ReadEnabled() {
		log.Read.Println("extractXRefTableEntriesFromXRefStream: end")
	}

	return nil
}

func xRefStreamDict(c context.Context, ctx *model.Context, o types.Object, objNr int, streamOffset int64) (*types.XRefStreamDict, error) {
	d, ok := o.(types.Dict)
	if !ok {
		return nil, errors.New("pdfcpu: xRefStreamDict: no dict")
	}

	// Parse attributes for stream object.
	streamLength, streamLengthObjNr := d.Length()
	if streamLength == nil && streamLengthObjNr == nil {
		return nil, errors.New("pdfcpu: xRefStreamDict: no \"Length\" entry")
	}

	filterPipeline, err := pdfFilterPipeline(c, ctx, d)
	if err != nil {
		return nil, err
	}

	// We have a stream object.
	if log.ReadEnabled() {
		log.Read.Printf("xRefStreamDict: streamobject #%d\n", objNr)
	}
	sd := types.NewStreamDict(d, streamOffset, streamLength, streamLengthObjNr, filterPipeline)

	if err = loadEncodedStreamContent(c, ctx, &sd, false); err != nil {
		return nil, err
	}

	// Decode xrefstream content
	if err = saveDecodedStreamContent(nil, &sd, 0, 0, true); err != nil {
		return nil, errors.Wrapf(err, "xRefStreamDict: cannot decode stream for obj#:%d\n", objNr)
	}

	return model.ParseXRefStreamDict(&sd)
}

func processXRefStream(ctx *model.Context, xsd *types.XRefStreamDict, objNr, genNr *int, offset *int64, offExtra int64) (prevOffset *int64, err error) {
	if log.ReadEnabled() {
		log.Read.Println("processXRefStream: begin")
	}

	if err = parseTrailer(ctx.XRefTable, xsd.Dict); err != nil {
		return nil, err
	}

	// Parse xRefStream and create xRefTable entries for embedded objects.
	if err = extractXRefTableEntriesFromXRefStream(xsd.Content, offExtra, xsd, ctx); err != nil {
		return nil, err
	}

	*offset += offExtra

	if entry, ok := ctx.Table[*objNr]; ok && entry.Offset != nil && *entry.Offset == *offset {
		entry.Object = *xsd
	}

	//////////////////
	// entry :=
	// 	model.XRefTableEntry{
	// 		Free:       false,
	// 		Offset:     offset,
	// 		Generation: genNr,
	// 		Object:     *xsd}

	// if log.ReadEnabled() {
	// 	log.Read.Printf("processXRefStream: Insert new xRefTable entry for Object %d\n", *objNr)
	// }

	// ctx.Table[*objNr] = &entry
	// ctx.Read.XRefStreams[*objNr] = true
	///////////////////

	prevOffset = xsd.PreviousOffset

	if log.ReadEnabled() {
		log.Read.Println("processXRefStream: end")
	}

	return prevOffset, nil
}

// Parse xRef stream and setup xrefTable entries for all embedded objects and the xref stream dict.
func parseXRefStream(c context.Context, ctx *model.Context, rd io.Reader, offset *int64, offExtra int64) (prevOffset *int64, err error) {
	if log.ReadEnabled() {
		log.Read.Printf("parseXRefStream: begin at offset %d\n", *offset)
	}

	buf, endInd, streamInd, streamOffset, err := buffer(c, rd)
	if err != nil {
		return nil, err
	}

	if log.ReadEnabled() {
		log.Read.Printf("parseXRefStream: endInd=%[1]d(%[1]x) streamInd=%[2]d(%[2]x)\n", endInd, streamInd)
	}

	line := string(buf)

	// We expect a stream and therefore "stream" before "endobj" if "endobj" within buffer.
	// There is no guarantee that "endobj" is contained in this buffer for large streams!
	if streamInd < 0 || (endInd > 0 && endInd < streamInd) {
		return nil, errors.New("pdfcpu: parseXRefStream: corrupt pdf file")
	}

	// Init object parse buf.
	l := line[:streamInd]

	objNr, genNr, err := model.ParseObjectAttributes(&l)
	if err != nil {
		return nil, err
	}

	// parse this object
	if log.ReadEnabled() {
		log.Read.Printf("parseXRefStream: xrefstm obj#:%d gen:%d\n", *objNr, *genNr)
		log.Read.Printf("parseXRefStream: dereferencing object %d\n", *objNr)
	}

	o, err := model.ParseObjectContext(c, &l)
	if err != nil {
		return nil, errors.Wrapf(err, "parseXRefStream: no object")
	}

	if log.ReadEnabled() {
		log.Read.Printf("parseXRefStream: we have an object: %s\n", o)
	}

	streamOffset += *offset
	xsd, err := xRefStreamDict(c, ctx, o, *objNr, streamOffset)
	if err != nil {
		return nil, err
	}

	return processXRefStream(ctx, xsd, objNr, genNr, offset, offExtra)
}

// Parse an xRefStream for a hybrid PDF file.
func parseHybridXRefStream(c context.Context, ctx *model.Context, offset *int64, offExtra int64) error {
	if log.ReadEnabled() {
		log.Read.Println("parseHybridXRefStream: begin")
	}

	rd, err := newPositionedReader(ctx.Read.RS, offset)
	if err != nil {
		return err
	}

	if _, err = parseXRefStream(c, ctx, rd, offset, offExtra); err != nil {
		return err
	}

	if log.ReadEnabled() {
		log.Read.Println("parseHybridXRefStream: end")
	}

	return nil
}

func parseTrailerSize(xRefTable *model.XRefTable, d types.Dict) error {
	i := d.Size()
	if i == nil {
		return errors.New("pdfcpu: parseTrailerSize: missing entry \"Size\"")
	}
	// Not reliable!
	// Patched after all read in.
	xRefTable.Size = i
	return nil
}

func parseTrailerRoot(xRefTable *model.XRefTable, d types.Dict) error {
	indRef := d.IndirectRefEntry("Root")
	if indRef == nil {
		return errors.New("pdfcpu: parseTrailerRoot: missing entry \"Root\"")
	}
	xRefTable.Root = indRef
	if log.ReadEnabled() {
		log.Read.Printf("parseTrailerRoot: Root object: %s\n", *xRefTable.Root)
	}
	return nil
}

func parseTrailerInfo(xRefTable *model.XRefTable, d types.Dict) error {
	indRef := d.IndirectRefEntry("Info")
	if indRef != nil {
		xRefTable.Info = indRef
		if log.ReadEnabled() {
			log.Read.Printf("parseTrailerInfo: Info object: %s\n", *xRefTable.Info)
		}
	}
	return nil
}

func parseTrailerID(xRefTable *model.XRefTable, d types.Dict) error {
	arr := d.ArrayEntry("ID")
	if arr != nil {
		if len(arr) != 2 {
			if xRefTable.ValidationMode == model.ValidationStrict {
				return errors.New("pdfcpu: parseTrailerID: invalid entry \"ID\"")
			}
			if len(arr) != 1 {
				return errors.New("pdfcpu: parseTrailerID: invalid entry \"ID\"")
			}
			arr = append(arr, arr[0])
			model.ShowRepaired("trailer ID")
		}
		xRefTable.ID = arr
		if log.ReadEnabled() {
			log.Read.Printf("parseTrailerID: ID object: %s\n", xRefTable.ID)
		}
		return nil
	}

	if xRefTable.Encrypt != nil {
		return errors.New("pdfcpu: parseTrailerID: missing entry \"ID\"")
	}

	return nil
}

// Parse trailer dict and return any offset of a previous xref section.
func parseTrailer(xRefTable *model.XRefTable, d types.Dict) error {
	if log.ReadEnabled() {
		log.Read.Println("parseTrailer begin")
	}

	if indRef := d.IndirectRefEntry("Encrypt"); indRef != nil {
		xRefTable.Encrypt = indRef
		if log.ReadEnabled() {
			log.Read.Printf("parseTrailer: Encrypt object: %s\n", *xRefTable.Encrypt)
		}
	}

	if xRefTable.Size == nil {
		if err := parseTrailerSize(xRefTable, d); err != nil {
			return err
		}
	}

	if xRefTable.Root == nil {
		if err := parseTrailerRoot(xRefTable, d); err != nil {
			return err
		}
	}

	if xRefTable.Info == nil {
		if err := parseTrailerInfo(xRefTable, d); err != nil {
			return err
		}
	}

	if xRefTable.ID == nil {
		if err := parseTrailerID(xRefTable, d); err != nil {
			return err
		}
	}

	if log.ReadEnabled() {
		log.Read.Println("parseTrailerf end")
	}

	return nil
}

func scanForPreviousXref(ctx *model.Context, offset *int64) *int64 {
	var (
		prevBuf, workBuf []byte
		bufSize          int64 = 512
		off              int64
		match1           []byte = []byte("startxref")
		match2           []byte = []byte("xref")
	)

	m := match1

	for i := int64(1); ; i++ {
		off = *offset - i*bufSize
		rd, err := newPositionedReader(ctx.Read.RS, &off)
		if err != nil {
			return nil
		}

		curBuf := make([]byte, bufSize)

		n, err := fillBuffer(rd, curBuf)
		if err != nil {
			return nil
		}

		workBuf = curBuf
		if prevBuf != nil {
			workBuf = append(curBuf, prevBuf...)
		}

		j := bytes.LastIndex(workBuf, m)
		if j == -1 {
			if int64(n) < bufSize {
				return nil
			}
			prevBuf = curBuf
			continue
		}

		if bytes.Equal(m, match1) {
			m = match2
			continue
		}

		off += int64(j)
		break
	}

	return &off
}

func handleAdditionalStreams(trailerDict types.Dict, xRefTable *model.XRefTable) {
	arr := trailerDict.ArrayEntry("AdditionalStreams")
	if arr == nil {
		return
	}

	if log.ReadEnabled() {
		log.Read.Printf("found AdditionalStreams: %s\n", arr)
	}

	a := types.Array{}
	for _, value := range arr {
		if indRef, ok := value.(types.IndirectRef); ok {
			a = append(a, indRef)
		}
	}

	xRefTable.AdditionalStreams = &a
}

func offsetPrev(ctx *model.Context, trailerDict types.Dict, offCurXRef *int64) *int64 {
	offset := trailerDict.Prev()
	if offset != nil {
		if log.ReadEnabled() {
			log.Read.Printf("offsetPrev: previous xref table section offset:%d\n", *offset)
		}
		if *offset == 0 {
			offset = nil
			if offCurXRef != nil {
				if off := scanForPreviousXref(ctx, offCurXRef); off != nil {
					offset = off
				}
			}
		}
	}
	return offset
}

func parseTrailerDict(c context.Context, ctx *model.Context, trailerDict types.Dict, offCurXRef *int64, offExtra int64) (*int64, error) {
	if log.ReadEnabled() {
		log.Read.Println("parseTrailerDict begin")
	}

	xRefTable := ctx.XRefTable

	if err := parseTrailer(xRefTable, trailerDict); err != nil {
		return nil, err
	}

	handleAdditionalStreams(trailerDict, xRefTable)

	offset := offsetPrev(ctx, trailerDict, offCurXRef)

	offsetXRefStream := trailerDict.Int64Entry("XRefStm")
	if offsetXRefStream == nil {
		// No cross reference stream.
		if !ctx.Reader15 && xRefTable.Version() >= model.V14 && !ctx.Read.Hybrid {
			return nil, errors.Errorf("parseTrailerDict: PDF1.4 conformant reader: found incompatible version: %s", xRefTable.VersionString())
		}
		if log.ReadEnabled() {
			log.Read.Println("parseTrailerDict end")
		}
		// continue to parse previous xref section, if there is any.
		return offset, nil
	}

	// This file is using cross reference streams.

	if !ctx.Read.Hybrid {
		ctx.Read.Hybrid = true
		ctx.Read.UsingXRefStreams = true
	}

	// 1.5 conformant readers process hidden objects contained
	// in XRefStm before continuing to process any previous XRefSection.
	// Previous XRefSection is expected to have free entries for hidden entries.
	// May appear in XRefSections only.
	if ctx.Reader15 {
		if err := parseHybridXRefStream(c, ctx, offsetXRefStream, offExtra); err != nil {
			return nil, err
		}
	}

	if log.ReadEnabled() {
		log.Read.Println("parseTrailerDict end")
	}

	return offset, nil
}

func scanLineRaw(s *bufio.Scanner) (string, error) {
	if ok := s.Scan(); !ok {
		if s.Err() != nil {
			return "", s.Err()
		}
		return "", errors.New("pdfcpu: scanLineRaw: returning nothing")
	}
	return s.Text(), nil
}

func scanLine(s *bufio.Scanner) (s1 string, err error) {
	for i := 0; i <= 1; i++ {
		s1, err = scanLineRaw(s)
		if err != nil {
			return "", err
		}
		if len(s1) > 0 {
			break
		}
	}

	return s1, nil
}

func isDict(s string) (bool, error) {
	o, err := model.ParseObject(&s)
	if err != nil {
		return false, err
	}
	_, ok := o.(types.Dict)
	return ok, nil
}

func scanTrailerDictStart(s *bufio.Scanner, line *string) error {
	l := *line
	var err error
	for {
		i := strings.Index(l, "<<")
		if i >= 0 {
			*line = l[i:]
			return nil
		}
		l, err = scanLine(s)
		if log.ReadEnabled() {
			log.Read.Printf("line: <%s>\n", l)
		}
		if err != nil {
			return err
		}
	}
}

func scanTrailerDictRemainder(s *bufio.Scanner, line string, buf bytes.Buffer) (string, error) {
	var (
		i   int
		err error
	)

	for i = strings.Index(line, "startxref"); i < 0; {
		if log.ReadEnabled() {
			log.Read.Printf("line: <%s>\n", line)
		}
		buf.WriteString(line)
		buf.WriteString("\x0a")
		if line, err = scanLine(s); err != nil {
			return "", err
		}
		i = strings.Index(line, "startxref")
	}

	line = line[:i]
	if log.ReadEnabled() {
		log.Read.Printf("line: <%s>\n", line)
	}
	buf.WriteString(line[:i])
	buf.WriteString("\x0a")

	return buf.String(), nil
}

func scanTrailer(s *bufio.Scanner, line string) (string, error) {
	var buf bytes.Buffer
	if log.ReadEnabled() {
		log.Read.Printf("line: <%s>\n", line)
	}

	if err := scanTrailerDictStart(s, &line); err != nil {
		return "", err
	}

	return scanTrailerDictRemainder(s, line, buf)
}

func processTrailer(c context.Context, ctx *model.Context, s *bufio.Scanner, line string, offCurXRef *int64, offExtra int64) (*int64, error) {
	var trailerString string

	if line != "trailer" {
		trailerString = line[7:]
		if log.ReadEnabled() {
			log.Read.Printf("processTrailer: trailer leftover: <%s>\n", trailerString)
		}
	} else {
		if log.ReadEnabled() {
			log.Read.Printf("line (len %d) <%s>\n", len(line), line)
		}
	}

	trailerString, err := scanTrailer(s, trailerString)
	if err != nil {
		return nil, err
	}

	if log.ReadEnabled() {
		log.Read.Printf("processTrailer: trailerString: (len:%d) <%s>\n", len(trailerString), trailerString)
	}

	o, err := model.ParseObjectContext(c, &trailerString)
	if err != nil {
		return nil, err
	}

	trailerDict, ok := o.(types.Dict)
	if !ok {
		return nil, errors.New("pdfcpu: processTrailer: corrupt trailer dict")
	}

	if log.ReadEnabled() {
		log.Read.Printf("processTrailer: trailerDict:\n%s\n", trailerDict)
	}

	return parseTrailerDict(c, ctx, trailerDict, offCurXRef, offExtra)
}

// Parse xRef section into corresponding number of xRef table entries.
func parseXRefSection(c context.Context, ctx *model.Context, s *bufio.Scanner, fields []string, ssCount *int, offCurXRef *int64, offExtra int64, repairOff int) (*int64, error) {
	if log.ReadEnabled() {
		log.Read.Println("parseXRefSection begin")
	}

	var (
		line string
		err  error
	)

	if len(fields) == 0 {

		line, err = scanLine(s)
		if err != nil {
			return nil, err
		}

		if log.ReadEnabled() {
			log.Read.Printf("parseXRefSection: <%s>\n", line)
		}

		fields = strings.Fields(line)
	}

	// Process all sub sections of this xRef section.
	for !strings.HasPrefix(line, "trailer") && len(fields) == 2 {

		if err = parseXRefTableSubSection(ctx.XRefTable, s, fields, offExtra, repairOff); err != nil {
			return nil, err
		}
		*ssCount++

		// trailer or another xref table subsection ?
		if line, err = scanLine(s); err != nil {
			return nil, err
		}

		// if empty line try next line for trailer
		if len(line) == 0 {
			if line, err = scanLine(s); err != nil {
				return nil, err
			}
		}

		fields = strings.Fields(line)
	}

	if log.ReadEnabled() {
		log.Read.Println("parseXRefSection: All subsections read!")
	}

	if !strings.HasPrefix(line, "trailer") {
		return nil, errors.Errorf("xrefsection: missing trailer dict, line = <%s>", line)
	}

	if log.ReadEnabled() {
		log.Read.Println("parseXRefSection: parsing trailer dict..")
	}

	return processTrailer(c, ctx, s, line, offCurXRef, offExtra)
}

func scanForVersion(rs io.ReadSeeker, prefix string) ([]byte, int, error) {
	bufSize := 100

	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return nil, 0, err
	}

	buf := make([]byte, bufSize)
	var curBuf []byte

	off := 0
	found := false
	var buf2 []byte

	for !found {
		n, err := fillBuffer(rs, buf)
		if err != nil {
			return nil, 0, ErrCorruptHeader
		}
		curBuf = buf[:n]
		for {
			i := bytes.IndexByte(curBuf, '%')
			if i < 0 {
				// no match, check next block
				off += bufSize
				break
			}

			// Check all occurrences
			if i < len(curBuf)-18 {
				if !bytes.HasPrefix(curBuf[i:], []byte(prefix)) {
					// No match, keep checking
					curBuf = curBuf[i+1:]
					continue
				}
				off += i
				curBuf = curBuf[i:]
				found = true
				break
			}

			// Partial match, need 2nd buffer
			if len(buf2) == 0 {
				buf2 = make([]byte, bufSize)
			}
			n, err := fillBuffer(rs, buf2)
			if err != nil {
				return nil, 0, ErrCorruptHeader
			}
			buf3 := append(curBuf[i:], buf2[:n]...)
			if !bytes.HasPrefix(buf3, []byte(prefix)) {
				// No match, keep checking
				curBuf = buf2
				off += bufSize
				continue
			}
			off += i
			curBuf = buf3
			found = true
			break
		}
	}

	return curBuf, off, nil
}

// Get version from first line of file.
// Beginning with PDF 1.4, the Version entry in the document’s catalog dictionary
// (located via the Root entry in the file’s trailer, as described in 7.5.5, "File Trailer"),
// if present, shall be used instead of the version specified in the Header.
// The header version comes as the first line of the file.
// eolCount is the number of characters used for eol (1 or 2).
func headerVersion(rs io.ReadSeeker) (v *model.Version, eolCount int, offset int64, err error) {
	if log.ReadEnabled() {
		log.Read.Println("headerVersion begin")
	}

	prefix := "%PDF-"

	s, off, err := scanForVersion(rs, prefix)
	if err != nil {
		return nil, 0, 0, err
	}

	pdfVersion, err := model.PDFVersion(string(s[len(prefix) : len(prefix)+3]))
	if err != nil {
		return nil, 0, 0, errors.Wrapf(err, "headerVersion: unknown PDF Header Version")
	}

	s = s[8:]
	s = bytes.TrimLeft(s, "\t\f ")

	// Detect the used eol which should be 1 (0x00, 0x0D) or 2 chars (0x0D0A)long.
	// %PDF-1.x{whiteSpace}{text}{eol} or
	j := bytes.IndexAny(s, "\x0A\x0D")
	if j < 0 {
		return nil, 0, 0, ErrCorruptHeader
	}
	if s[j] == 0x0A {
		eolCount = 1
	} else if s[j] == 0x0D {
		eolCount = 1
		if (len(s) > j+1) && (s[j+1] == 0x0A) {
			eolCount = 2
		}
	}

	if log.ReadEnabled() {
		log.Read.Printf("headerVersion: end, found header version: %s\n", pdfVersion)
	}

	return &pdfVersion, eolCount, int64(off), nil
}

func parseAndLoad(c context.Context, ctx *model.Context, line string, offset *int64) error {
	l := line
	objNr, generation, err := model.ParseObjectAttributes(&l)
	if err != nil {
		return err
	}

	entry := model.XRefTableEntry{
		Free:       false,
		Offset:     offset,
		Generation: generation}

	if !ctx.XRefTable.Exists(*objNr) {
		ctx.Table[*objNr] = &entry
	}

	o, err := ParseObjectWithContext(c, ctx, *entry.Offset, *objNr, *entry.Generation)
	if err != nil {
		return err
	}

	entry.Object = o

	sd, ok := o.(types.StreamDict)
	if ok {
		if err = loadStreamDict(c, ctx, &sd, *objNr, *generation, true); err != nil {
			return err
		}
		entry.Object = sd
		*offset = sd.StreamOffset + *sd.StreamLength
		return nil
	}

	*offset += int64(len(line) + ctx.Read.EolCount)

	return nil
}

func processObject(c context.Context, ctx *model.Context, line string, offset *int64) (*bufio.Scanner, error) {
	if err := parseAndLoad(c, ctx, line, offset); err != nil {
		return nil, err
	}
	rd, err := newPositionedReader(ctx.Read.RS, offset)
	if err != nil {
		return nil, err
	}
	s := bufio.NewScanner(rd)
	s.Split(scan.Lines)
	return s, nil
}

// bypassXrefSection is a fix for digesting corrupt xref sections.
// It populates the xRefTable by reading in all indirect objects line by line
// and works on the assumption of a single xref section - meaning no incremental updates.
func bypassXrefSection(c context.Context, ctx *model.Context, offExtra int64, wasErr error) error {
	if log.ReadEnabled() {
		log.Read.Printf("bypassXRefSection after %v\n", wasErr)
	}

	var z int64
	g := types.FreeHeadGeneration
	ctx.Table[0] = &model.XRefTableEntry{
		Free:       true,
		Offset:     &z,
		Generation: &g}

	rs := ctx.Read.RS
	eolCount := ctx.Read.EolCount
	var offset int64

	rd, err := newPositionedReader(rs, &offset)
	if err != nil {
		return err
	}

	s := bufio.NewScanner(rd)
	s.Split(scan.Lines)

	bb := []byte{}
	var (
		withinXref    bool
		withinTrailer bool
	)

	for {
		line, err := scanLineRaw(s)
		if err != nil {
			break
		}
		if withinXref {
			offset += int64(len(line) + eolCount)
			if withinTrailer {
				bb = append(bb, '\n')
				bb = append(bb, line...)
				i := strings.Index(line, "startxref")
				if i >= 0 {
					_, err = processTrailer(c, ctx, s, string(bb), nil, offExtra)
					if err == nil {
						model.ShowRepaired("xreftable")
					}
					return err
				}
				continue
			}
			i := strings.Index(line, "trailer")
			if i >= 0 {
				bb = append(bb, line...)
				withinTrailer = true
			}
			continue
		}
		i := strings.Index(line, "xref")
		if i >= 0 {
			offset += int64(len(line) + eolCount)
			withinXref = true
			continue
		}
		i = strings.Index(line, "obj")
		if i >= 0 {
			if i > 2 && strings.Index(line, "endobj") != i-3 {
				s, err = processObject(c, ctx, line, &offset)
				if err != nil {
					return err
				}
				continue
			}
		}
		offset += int64(len(line) + eolCount)
		continue
	}
	return nil
}

func postProcess(ctx *model.Context, xrefSectionCount int) {
	// Ensure free object #0 if exactly one xref subsection
	// and in one of the following weird situations:
	if xrefSectionCount == 1 && !ctx.Exists(0) {
		// Fix for #250
		if *ctx.Size == len(ctx.Table)+1 {
			// Create free object 0 from scratch if the free list head is missing.
			g0 := types.FreeHeadGeneration
			ctx.Table[0] = &model.XRefTableEntry{Free: true, Offset: &zero, Generation: &g0}
		} else {
			// Create free object 0 by shifting down all objects by one.
			for i := 1; i <= *ctx.Size; i++ {
				ctx.Table[i-1] = ctx.Table[i]
			}
			delete(ctx.Table, *ctx.Size)
		}
		model.ShowRepaired("obj#0")
	}
}

func tryXRefSection(c context.Context, ctx *model.Context, rs io.ReadSeeker, offset *int64, offExtra int64, xrefSectionCount *int) (*int64, error) {
	rd, err := newPositionedReader(rs, offset)
	if err != nil {
		return nil, err
	}

	s := bufio.NewScanner(rd)
	buf := make([]byte, 0, 4096)
	s.Buffer(buf, 1024*1024)
	s.Split(scan.Lines)

	line, err := scanLine(s)
	if err != nil {
		return nil, err
	}
	if log.ReadEnabled() {
		log.Read.Printf("xref line 1: <%s>\n", line)
	}
	repairOff := len(line)

	if strings.TrimSpace(line) == "xref" {
		if log.ReadEnabled() {
			log.Read.Println("tryXRefSection: found xref section")
		}
		return parseXRefSection(c, ctx, s, nil, xrefSectionCount, offset, offExtra, 0)
	}

	// Repair fix for #823
	if strings.HasPrefix(line, "xref") {
		fields := strings.Fields(line)
		if len(fields) == 3 {
			return parseXRefSection(c, ctx, s, fields[1:], xrefSectionCount, offset, offExtra, 0)
		}
	}

	// Repair fix for #326
	if line, err = scanLine(s); err != nil {
		return nil, err
	}
	if log.ReadEnabled() {
		log.Read.Printf("xref line 2: <%s>\n", line)
	}

	i := strings.Index(line, "xref")
	if i >= 0 {
		if log.ReadEnabled() {
			log.Read.Println("tryXRefSection: found xref section")
		}
		repairOff += i
		if log.ReadEnabled() {
			log.Read.Printf("Repair offset: %d\n", repairOff)
		}
		return parseXRefSection(c, ctx, s, nil, xrefSectionCount, offset, offExtra, repairOff)
	}

	return &zero, nil
}

// Build XRefTable by reading XRef streams or XRef sections.
func buildXRefTableStartingAt(c context.Context, ctx *model.Context, offset *int64) error {
	if log.ReadEnabled() {
		log.Read.Println("buildXRefTableStartingAt: begin")
	}

	rs := ctx.Read.RS
	hv, eolCount, offExtra, err := headerVersion(rs)
	if err != nil {
		return err
	}
	*offset += offExtra

	ctx.HeaderVersion = hv
	ctx.Read.EolCount = eolCount
	offs := map[int64]bool{}
	xrefSectionCount := 0

	for offset != nil {

		if err := c.Err(); err != nil {
			return err
		}

		if offs[*offset] {
			if offset, err = offsetLastXRefSection(ctx, ctx.Read.FileSize-*offset); err != nil {
				return err
			}
			if offs[*offset] {
				return nil
			}
		}

		offs[*offset] = true

		off, err := tryXRefSection(c, ctx, rs, offset, offExtra, &xrefSectionCount)
		if err != nil {
			return err
		}

		if off == nil || *off != 0 {
			offset = off
			continue
		}

		if log.ReadEnabled() {
			log.Read.Println("buildXRefTableStartingAt: found xref stream")
		}
		ctx.Read.UsingXRefStreams = true
		rd, err := newPositionedReader(rs, offset)
		if err != nil {
			return err
		}

		if offset, err = parseXRefStream(c, ctx, rd, offset, offExtra); err != nil {
			// Try fix for corrupt single xref section.
			return bypassXrefSection(c, ctx, offExtra, err)
		}

	}

	postProcess(ctx, xrefSectionCount)

	if log.ReadEnabled() {
		log.Read.Println("buildXRefTableStartingAt: end")
	}

	return nil
}

// Populate the cross reference table for this PDF file.
// Goto offset of first xref table entry.
// Can be "xref" or indirect object reference eg. "34 0 obj"
// Keep digesting xref sections as long as there is a defined previous xref section
// and build up the xref table along the way.
func readXRefTable(c context.Context, ctx *model.Context) (err error) {
	if log.ReadEnabled() {
		log.Read.Println("readXRefTable: begin")
	}

	offset, err := offsetLastXRefSection(ctx, 0)
	if err != nil {
		return
	}

	ctx.Write.OffsetPrevXRef = offset

	err = buildXRefTableStartingAt(c, ctx, offset)
	if err == io.EOF {
		return errors.Wrap(err, "readXRefTable: unexpected eof")
	}
	if err != nil {
		return
	}

	//Log list of free objects (not the "free list").
	//log.Read.Printf("freelist: %v\n", ctx.freeObjects())

	// Note: Acrobat 6.0 and later do not use the free list to recycle object numbers - pdfcpu does.
	err = ctx.EnsureValidFreeList()

	if log.ReadEnabled() {
		log.Read.Println("readXRefTable: end")
	}

	return err
}

func growBufBy(buf []byte, size int, rd io.Reader) ([]byte, error) {
	b := make([]byte, size)

	if _, err := fillBuffer(rd, b); err != nil {
		return nil, err
	}
	//log.Read.Printf("growBufBy: Read %d bytes\n", n)

	return append(buf, b...), nil
}

func nextStreamOffset(line string, streamInd int) (off int) {
	off = streamInd + len("stream")

	// Skip optional blanks.
	// TODO Should we skip optional whitespace instead?
	for ; line[off] == 0x20; off++ {
	}

	// Skip 0A eol.
	if line[off] == '\n' {
		off++
		return
	}

	// Skip 0D eol.
	if line[off] == '\r' {
		off++
		// Skip 0D0A eol.
		if line[off] == '\n' {
			off++
		}
	}

	return
}

func lastStreamMarker(streamInd *int, endInd int, line string) {
	if *streamInd > len(line)-len("stream") {
		// No space for another stream marker.
		*streamInd = -1
		return
	}

	// We start searching after this stream marker.
	bufpos := *streamInd + len("stream")

	// Search for next stream marker.
	i := strings.Index(line[bufpos:], "stream")
	if i < 0 {
		// No stream marker within line buffer.
		*streamInd = -1
		return
	}

	// We found the next stream marker.
	*streamInd += len("stream") + i

	if endInd > 0 && *streamInd > endInd {
		// We found a stream marker of another object
		*streamInd = -1
	}

}

// Provide a PDF file buffer of sufficient size for parsing an object w/o stream.
func buffer(c context.Context, rd io.Reader) (buf []byte, endInd int, streamInd int, streamOffset int64, err error) {
	// process: # gen obj ... obj dict ... {stream ... data ... endstream} ... endobj
	//                                    streamInd                            endInd
	//                                  -1 if absent                        -1 if absent

	//log.Read.Println("buffer: begin")

	endInd, streamInd = -1, -1
	growSize := defaultBufSize

	for endInd < 0 && streamInd < 0 {
		if err := c.Err(); err != nil {
			return nil, 0, 0, 0, err
		}

		if buf, err = growBufBy(buf, growSize, rd); err != nil {
			return nil, 0, 0, 0, err
		}

		growSize = min(growSize*2, maximumBufSize)
		line := string(buf)

		endInd, streamInd, err = model.DetectKeywordsWithContext(c, line)
		if err != nil {
			return nil, 0, 0, 0, err
		}

		if endInd > 0 && (streamInd < 0 || streamInd > endInd) {
			// No stream marker in buf detected.
			break
		}

		// For very rare cases where "stream" also occurs within obj dict
		// we need to find the last "stream" marker before a possible end marker.
		for streamInd > 0 && !keywordStreamRightAfterEndOfDict(line, streamInd) {
			lastStreamMarker(&streamInd, endInd, line)
		}

		if log.ReadEnabled() {
			log.Read.Printf("buffer: endInd=%d streamInd=%d\n", endInd, streamInd)
		}

		if streamInd > 0 {

			// streamOffset ... the offset where the actual stream data begins.
			//                  is right after the eol after "stream".

			slack := 10 // for optional whitespace + eol (max 2 chars)
			need := streamInd + len("stream") + slack

			if len(line) < need {

				// to prevent buffer overflow.
				if buf, err = growBufBy(buf, need-len(line), rd); err != nil {
					return nil, 0, 0, 0, err
				}

				line = string(buf)
			}

			streamOffset = int64(nextStreamOffset(line, streamInd))
		}
	}

	//log.Read.Printf("buffer: end, returned bufsize=%d streamOffset=%d\n", len(buf), streamOffset)

	return buf, endInd, streamInd, streamOffset, nil
}

// return true if 'stream' follows end of dict: >>{whitespace}stream
func keywordStreamRightAfterEndOfDict(buf string, streamInd int) bool {
	//log.Read.Println("keywordStreamRightAfterEndOfDict: begin")

	// Get a slice of the chunk right in front of 'stream'.
	b := buf[:streamInd]

	// Look for last end of dict marker.
	eod := strings.LastIndex(b, ">>")
	if eod < 0 {
		// No end of dict in buf.
		return false
	}

	// We found the last >>. Return true if after end of dict only whitespace.
	ok := strings.TrimSpace(b[eod:]) == ">>"

	//log.Read.Printf("keywordStreamRightAfterEndOfDict: end, %v\n", ok)

	return ok
}

func buildFilterPipeline(c context.Context, ctx *model.Context, filterArray, decodeParmsArr types.Array) ([]types.PDFFilter, error) {
	var filterPipeline []types.PDFFilter

	for i, f := range filterArray {

		filterName, ok := f.(types.Name)
		if !ok {
			return nil, errors.New("pdfcpu: buildFilterPipeline: filterArray elements corrupt")
		}
		if decodeParmsArr == nil || decodeParmsArr[i] == nil {
			filterPipeline = append(filterPipeline, types.PDFFilter{Name: filterName.Value(), DecodeParms: nil})
			continue
		}

		dict, ok := decodeParmsArr[i].(types.Dict)
		if !ok {
			indRef, ok := decodeParmsArr[i].(types.IndirectRef)
			if !ok {
				return nil, errors.Errorf("buildFilterPipeline: corrupt Dict: %s\n", dict)
			}
			d, err := dereferencedDict(c, ctx, indRef.ObjectNumber.Value())
			if err != nil {
				return nil, err
			}
			dict = d
		}

		filterPipeline = append(filterPipeline, types.PDFFilter{Name: filterName.String(), DecodeParms: dict})
	}

	return filterPipeline, nil
}

func singleFilter(c context.Context, ctx *model.Context, filterName string, d types.Dict) ([]types.PDFFilter, error) {
	o, found := d.Find("DecodeParms")
	if !found {
		// w/o decode parameters.
		if log.ReadEnabled() {
			log.Read.Println("singleFilter: end w/o decode parms")
		}
		return []types.PDFFilter{{Name: filterName}}, nil
	}

	var err error
	d, ok := o.(types.Dict)
	if !ok {
		indRef, ok := o.(types.IndirectRef)
		if !ok {
			return nil, errors.Errorf("singleFilter: corrupt Dict: %s\n", o)
		}
		if d, err = dereferencedDict(c, ctx, indRef.ObjectNumber.Value()); err != nil {
			return nil, err
		}
	}

	// with decode parameters.
	if log.ReadEnabled() {
		log.Read.Println("singleFilter: end with decode parms")
	}

	return []types.PDFFilter{{Name: filterName, DecodeParms: d}}, nil
}

func filterArraySupportsDecodeParms(filters types.Array) bool {
	for _, obj := range filters {
		if name, ok := obj.(types.Name); ok {
			if filter.SupportsDecodeParms(name.String()) {
				return true
			}
		}
	}
	return false
}

// Return the filter pipeline associated with this stream dict.
func pdfFilterPipeline(c context.Context, ctx *model.Context, dict types.Dict) ([]types.PDFFilter, error) {
	if log.ReadEnabled() {
		log.Read.Println("pdfFilterPipeline: begin")
	}

	var err error

	o, found := dict.Find("Filter")
	if !found {
		// stream is not compressed.
		return nil, nil
	}

	// compressed stream.

	var filterPipeline []types.PDFFilter

	if indRef, ok := o.(types.IndirectRef); ok {
		if o, err = dereferencedObject(c, ctx, indRef.ObjectNumber.Value()); err != nil {
			return nil, err
		}
	}

	//fmt.Printf("dereferenced filter obj: %s\n", obj)

	if name, ok := o.(types.Name); ok {
		return singleFilter(c, ctx, name.String(), dict)
	}

	// filter pipeline.

	// Array of filternames
	filterArray, ok := o.(types.Array)
	if !ok {
		return nil, errors.Errorf("pdfFilterPipeline: Expected filterArray corrupt, %v %T", o, o)
	}

	// Optional array of decode parameter dicts.
	var decodeParmsArr types.Array
	decodeParms, found := dict.Find("DecodeParms")
	if found {
		if filterArraySupportsDecodeParms(filterArray) {
			decodeParmsArr, ok = decodeParms.(types.Array)
			if ok {
				if len(decodeParmsArr) != len(filterArray) {
					return nil, errors.New("pdfcpu: pdfFilterPipeline: expected decodeParms array corrupt")
				}
			}
		}
	}

	//fmt.Printf("decodeParmsArr: %s\n", decodeParmsArr)

	filterPipeline, err = buildFilterPipeline(c, ctx, filterArray, decodeParmsArr)

	if log.ReadEnabled() {
		log.Read.Println("pdfFilterPipeline: end")
	}

	return filterPipeline, err
}

func streamDictForObject(c context.Context, ctx *model.Context, d types.Dict, objNr, streamInd int, streamOffset, offset int64) (sd types.StreamDict, err error) {
	streamLength, streamLengthRef := d.Length()

	if streamInd <= 0 {
		return sd, errors.New("pdfcpu: streamDictForObject: stream object without streamOffset")
	}

	filterPipeline, err := pdfFilterPipeline(c, ctx, d)
	if err != nil {
		return sd, err
	}

	streamOffset += offset

	// We have a stream object.
	sd = types.NewStreamDict(d, streamOffset, streamLength, streamLengthRef, filterPipeline)

	if log.ReadEnabled() {
		log.Read.Printf("streamDictForObject: end, Streamobject #%d\n", objNr)
	}

	return sd, nil
}

func dict(ctx *model.Context, d1 types.Dict, objNr, genNr, endInd, streamInd int) (d2 types.Dict, err error) {
	if ctx.EncKey != nil {
		if _, err := decryptDeepObject(d1, objNr, genNr, ctx.EncKey, ctx.AES4Strings, ctx.E.R); err != nil {
			return nil, err
		}
	}

	if endInd >= 0 && (streamInd < 0 || streamInd > endInd) {
		if log.ReadEnabled() {
			log.Read.Printf("dict: end, #%d\n", objNr)
		}
		d2 = d1
	}

	return d2, nil
}

func object(c context.Context, ctx *model.Context, offset int64, objNr, genNr int) (o types.Object, endInd, streamInd int, streamOffset int64, err error) {
	var rd io.Reader

	if rd, err = newPositionedReader(ctx.Read.RS, &offset); err != nil {
		return nil, 0, 0, 0, err
	}

	//log.Read.Printf("object: seeked to offset:%d\n", offset)

	// process: # gen obj ... obj dict ... {stream ... data ... endstream} endobj
	//                                    streamInd                        endInd
	//                                  -1 if absent                    -1 if absent
	var buf []byte
	if buf, endInd, streamInd, streamOffset, err = buffer(c, rd); err != nil {
		return nil, 0, 0, 0, err
	}

	//log.Read.Printf("streamInd:%d(#%x) streamOffset:%d(#%x) endInd:%d(#%x)\n", streamInd, streamInd, streamOffset, streamOffset, endInd, endInd)
	//log.Read.Printf("buflen=%d\n%s", len(buf), hex.Dump(buf))

	line := string(buf)

	var l string

	if endInd < 0 { // && streamInd >= 0, streamdict
		// buf: # gen obj ... obj dict ... stream ... data
		// implies we detected no endobj and a stream starting at streamInd.
		// big stream, we parse object until "stream"
		if log.ReadEnabled() {
			log.Read.Println("object: big stream, we parse object until stream")
		}
		l = line[:streamInd]
	} else if streamInd < 0 { // dict
		// buf: # gen obj ... obj dict ... endobj
		// implies we detected endobj and no stream.
		// small object w/o stream, parse until "endobj"
		if log.ReadEnabled() {
			log.Read.Println("object: small object w/o stream, parse until endobj")
		}
		l = line[:endInd]
	} else if streamInd < endInd { // streamdict
		// buf: # gen obj ... obj dict ... stream ... data ... endstream endobj
		// implies we detected endobj and stream.
		// small stream within buffer, parse until "stream"
		if log.ReadEnabled() {
			log.Read.Println("object: small stream within buffer, parse until stream")
		}
		l = line[:streamInd]
	} else { // dict
		// buf: # gen obj ... obj dict ... endobj # gen obj ... obj dict ... stream
		// small obj w/o stream, parse until "endobj"
		// stream in buf belongs to subsequent object.
		if log.ReadEnabled() {
			log.Read.Println("object: small obj w/o stream, parse until endobj")
		}
		l = line[:endInd]
	}

	// Parse object number and object generation.
	var objectNr, generationNr *int
	if objectNr, generationNr, err = model.ParseObjectAttributes(&l); err != nil {
		return nil, 0, 0, 0, err
	}

	if objNr != *objectNr || genNr != *generationNr {
		// This is suspicious, but ok if two object numbers point to same offset and only one of them is used
		// (compare entry.RefCount) like for cases where the PDF Writer is MS Word 2013.
		if log.ReadEnabled() {
			log.Read.Printf("object %d: non matching objNr(%d) or generationNumber(%d) tags found.\n", objNr, *objectNr, *generationNr)
		}
	}

	l = strings.TrimSpace(l)
	if len(l) == 0 {
		// 7.3.9
		// Specifying the null object as the value of a dictionary entry (7.3.7, "Dictionary Objects")
		// shall be equivalent to omitting the entry entirely.
		return nil, endInd, streamInd, streamOffset, err
	}

	o, err = model.ParseObjectContext(c, &l)

	return o, endInd, streamInd, streamOffset, err
}

// ParseObject parses an object from file at given offset.
func ParseObject(ctx *model.Context, offset int64, objNr, genNr int) (types.Object, error) {
	return ParseObjectWithContext(context.Background(), ctx, offset, objNr, genNr)
}

func resolveObject(c context.Context, ctx *model.Context, obj types.Object, offset int64, objNr, genNr, endInd, streamInd int, streamOffset int64) (types.Object, error) {
	switch o := obj.(type) {

	case types.Dict:
		d, err := dict(ctx, o, objNr, genNr, endInd, streamInd)
		if err != nil || d != nil {
			// Dict
			return d, err
		}
		// StreamDict.
		return streamDictForObject(c, ctx, o, objNr, streamInd, streamOffset, offset)

	case types.Array:
		if ctx.EncKey != nil {
			if _, err := decryptDeepObject(o, objNr, genNr, ctx.EncKey, ctx.AES4Strings, ctx.E.R); err != nil {
				return nil, err
			}
		}
		return o, nil

	case types.StringLiteral:
		if ctx.EncKey != nil {
			sl, err := decryptStringLiteral(o, objNr, genNr, ctx.EncKey, ctx.AES4Strings, ctx.E.R)
			if err != nil {
				return nil, err
			}
			return *sl, nil
		}
		return o, nil

	case types.HexLiteral:
		if ctx.EncKey != nil {
			hl, err := decryptHexLiteral(o, objNr, genNr, ctx.EncKey, ctx.AES4Strings, ctx.E.R)
			if err != nil {
				return nil, err
			}
			return *hl, nil
		}
		return o, nil

	default:
		return o, nil
	}
}

func ParseObjectWithContext(c context.Context, ctx *model.Context, offset int64, objNr, genNr int) (types.Object, error) {
	if log.ReadEnabled() {
		log.Read.Printf("ParseObject: begin, obj#%d, offset:%d\n", objNr, offset)
	}

	obj, endInd, streamInd, streamOffset, err := object(c, ctx, offset, objNr, genNr)
	if err != nil {
		if ctx.XRefTable.ValidationMode == model.ValidationRelaxed {
			if err == io.EOF {
				err = nil
			}
		}
		return nil, err
	}

	return resolveObject(c, ctx, obj, offset, objNr, genNr, endInd, streamInd, streamOffset)
}

func dereferencedObject(c context.Context, ctx *model.Context, objNr int) (types.Object, error) {
	entry, ok := ctx.Find(objNr)
	if !ok {
		return nil, errors.Errorf("pdfcpu: dereferencedObject: unregistered object: %d", objNr)
	}

	if entry.Compressed {
		if err := decompressXRefTableEntry(ctx.XRefTable, objNr, entry); err != nil {
			return nil, err
		}
	}

	if entry.Object == nil {

		if log.ReadEnabled() {
			log.Read.Printf("dereferencedObject: dereferencing object %d\n", objNr)
		}

		if entry.Free {
			return nil, ErrReferenceDoesNotExist
		}

		o, err := ParseObjectWithContext(c, ctx, *entry.Offset, objNr, *entry.Generation)
		if err != nil {
			return nil, errors.Wrapf(err, "dereferencedObject: problem dereferencing object %d", objNr)
		}

		if o == nil {
			return nil, errors.New("pdfcpu: dereferencedObject: object is nil")
		}

		entry.Object = o
	} else if l, ok := entry.Object.(types.LazyObjectStreamObject); ok {
		o, err := l.DecodedObject(c)
		if err != nil {
			return nil, errors.Wrapf(err, "dereferencedObject: problem dereferencing object %d", objNr)
		}

		model.ProcessRefCounts(ctx.XRefTable, o)
		entry.Object = o
	}

	return entry.Object, nil
}

func dereferencedInteger(c context.Context, ctx *model.Context, objNr int) (*types.Integer, error) {
	o, err := dereferencedObject(c, ctx, objNr)
	if err != nil {
		return nil, err
	}

	i, ok := o.(types.Integer)
	if !ok {
		return nil, errors.New("pdfcpu: dereferencedInteger: corrupt integer")
	}

	return &i, nil
}

func dereferencedDict(c context.Context, ctx *model.Context, objNr int) (types.Dict, error) {
	o, err := dereferencedObject(c, ctx, objNr)
	if err != nil {
		return nil, err
	}

	d, ok := o.(types.Dict)
	if !ok {
		return nil, errors.New("pdfcpu: dereferencedDict: corrupt dict")
	}

	return d, nil
}

// dereference a Integer object representing an int64 value.
func int64Object(c context.Context, ctx *model.Context, objNr int) (*int64, error) {
	if log.ReadEnabled() {
		log.Read.Printf("int64Object begin: %d\n", objNr)
	}

	i, err := dereferencedInteger(c, ctx, objNr)
	if err != nil {
		return nil, err
	}

	i64 := int64(i.Value())

	if log.ReadEnabled() {
		log.Read.Printf("int64Object end: %d\n", objNr)
	}

	return &i64, nil

}

func readStreamContentBlindly(rd io.Reader) (buf []byte, err error) {
	// Weak heuristic for reading in stream data for cases where stream length is unknown.
	// ...data...{eol}endstream{eol}endobj

	growSize := defaultBufSize
	if buf, err = growBufBy(buf, growSize, rd); err != nil {
		return nil, err
	}

	i := bytes.Index(buf, []byte("endstream"))
	if i < 0 {
		for i = -1; i < 0; i = bytes.Index(buf, []byte("endstream")) {
			growSize = min(growSize*2, maximumBufSize)
			buf, err = growBufBy(buf, growSize, rd)
			if err != nil {
				return nil, err
			}
		}
	}

	buf = buf[:i]

	j := 0

	// Cut off trailing eol's.
	for i = len(buf) - 1; i >= 0 && (buf[i] == 0x0A || buf[i] == 0x0D); i-- {
		j++
	}

	if j > 0 {
		buf = buf[:len(buf)-j]
	}

	return buf, nil
}

// Reads and returns a file buffer with length = stream length using provided reader positioned at offset.
func readStreamContent(rd io.Reader, streamLength int) ([]byte, error) {
	if log.ReadEnabled() {
		log.Read.Printf("readStreamContent: begin streamLength:%d\n", streamLength)
	}

	if streamLength == 0 {
		// Read until "endstream" then fix "Length".
		return readStreamContentBlindly(rd)
	}

	buf := make([]byte, streamLength)

	for totalCount := 0; totalCount < streamLength; {
		count, err := fillBuffer(rd, buf[totalCount:])
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			// Weak heuristic to detect the actual end of this stream
			// once we have reached EOF due to incorrect streamLength.
			eob := bytes.Index(buf, []byte("endstream"))
			if eob < 0 {
				return nil, err
			}
			return buf[:eob], nil
		}

		if log.ReadEnabled() {
			log.Read.Printf("readStreamContent: count=%d, buflen=%d(%X)\n", count, len(buf), len(buf))
		}
		totalCount += count
	}

	if log.ReadEnabled() {
		log.Read.Printf("readStreamContent: end\n")
	}

	return buf, nil
}

func ensureStreamLength(sd *types.StreamDict, rawContent []byte, fixLength bool) {
	l := int64(len(rawContent))
	if fixLength || sd.StreamLength == nil || l != *sd.StreamLength {
		sd.StreamLength = &l
		sd.Dict["Length"] = types.Integer(l)
	}
}

// loadEncodedStreamContent loads the encoded stream content into sd.
func loadEncodedStreamContent(c context.Context, ctx *model.Context, sd *types.StreamDict, fixLength bool) error {
	if log.ReadEnabled() {
		log.Read.Printf("loadEncodedStreamContent: begin\n%v\n", sd)
	}

	var err error

	if sd.Raw != nil {
		if log.ReadEnabled() {
			log.Read.Println("loadEncodedStreamContent: end, already in memory.")
		}
		return nil
	}

	// Read stream content encoded at offset with stream length.

	// Dereference stream length if stream length is an indirect object.
	if !fixLength && sd.StreamLength == nil {
		if sd.StreamLengthObjNr == nil {
			return errors.New("pdfcpu: loadEncodedStreamContent: missing streamLength")
		}
		if sd.StreamLength, err = int64Object(c, ctx, *sd.StreamLengthObjNr); err != nil {
			if err != ErrReferenceDoesNotExist {
				return err
			}
		}
		if log.ReadEnabled() {
			log.Read.Printf("loadEncodedStreamContent: new indirect streamLength:%d\n", *sd.StreamLength)
		}
	}

	newOffset := sd.StreamOffset
	rd, err := newPositionedReader(ctx.Read.RS, &newOffset)
	if err != nil {
		return err
	}

	l1 := 0
	if !fixLength && sd.StreamLength != nil {
		l1 = int(*sd.StreamLength)
	}
	rawContent, err := readStreamContent(rd, l1)
	if err != nil {
		return err
	}

	ensureStreamLength(sd, rawContent, fixLength)

	sd.Raw = rawContent

	if log.ReadEnabled() {
		log.Read.Printf("loadEncodedStreamContent: end: len(streamDictRaw)=%d\n", len(sd.Raw))
	}

	return nil
}

// Decodes the raw encoded stream content and saves it to streamDict.Content.
func saveDecodedStreamContent(ctx *model.Context, sd *types.StreamDict, objNr, genNr int, decode bool) (err error) {
	if log.ReadEnabled() {
		log.Read.Printf("saveDecodedStreamContent: begin decode=%t\n", decode)
	}

	// If the "Identity" crypt filter is used we do not need to decrypt.
	if ctx != nil && ctx.EncKey != nil {
		if len(sd.FilterPipeline) == 1 && sd.FilterPipeline[0].Name == "Crypt" {
			sd.Content = sd.Raw
			return nil
		}
	}

	// Special case: If the length of the encoded data is 0, we do not need to decode anything.
	if len(sd.Raw) == 0 {
		sd.Content = sd.Raw
		return nil
	}

	// ctx gets created after XRefStream parsing.
	// XRefStreams are not encrypted.
	if ctx != nil && ctx.EncKey != nil {
		if sd.Raw, err = decryptStream(sd.Raw, objNr, genNr, ctx.EncKey, ctx.AES4Streams, ctx.E.R); err != nil {
			return err
		}
		l := int64(len(sd.Raw))
		sd.StreamLength = &l
	}

	if !decode {
		return nil
	}

	if sd.Image() {
		return nil
	}

	// Actual decoding of stream data.
	err = sd.Decode()
	if err == filter.ErrUnsupportedFilter {
		err = nil
	}
	if err != nil {
		return err
	}

	if log.ReadEnabled() {
		log.Read.Println("saveDecodedStreamContent: end")
	}

	return nil
}

// Resolve compressed xRefTableEntry
func decompressXRefTableEntry(xRefTable *model.XRefTable, objNr int, entry *model.XRefTableEntry) error {
	if log.ReadEnabled() {
		log.Read.Printf("decompressXRefTableEntry: compressed object %d at %d[%d]\n", objNr, *entry.ObjectStream, *entry.ObjectStreamInd)
	}

	// Resolve xRefTable entry of referenced object stream.
	objectStreamXRefTableEntry, ok := xRefTable.Find(*entry.ObjectStream)
	if !ok {
		return errors.Errorf("decompressXRefTableEntry: problem dereferencing object stream %d, no xref table entry", *entry.ObjectStream)
	}

	// Object of this entry has to be a ObjectStreamDict.
	sd, ok := objectStreamXRefTableEntry.Object.(types.ObjectStreamDict)
	if !ok {
		return errors.Errorf("decompressXRefTableEntry: problem dereferencing object stream %d, no object stream", *entry.ObjectStream)
	}

	// Get indexed object from ObjectStreamDict.
	o, err := sd.IndexedObject(*entry.ObjectStreamInd)
	if err != nil {
		return errors.Wrapf(err, "decompressXRefTableEntry: problem dereferencing object stream %d", *entry.ObjectStream)
	}

	// Save object to XRefRableEntry.
	g := 0
	entry.Object = o
	entry.Generation = &g
	entry.Compressed = false

	if log.ReadEnabled() {
		log.Read.Printf("decompressXRefTableEntry: end, Obj %d[%d]:\n<%s>\n", *entry.ObjectStream, *entry.ObjectStreamInd, o)
	}

	return nil
}

// Log interesting stream content.
func logStream(o types.Object) {
	if !log.ReadEnabled() {
		return
	}

	switch o := o.(type) {

	case types.StreamDict:

		if o.Content == nil {
			log.Read.Println("logStream: no stream content")
		}

		// if o.IsPageContent {
		// 	//log.Read.Printf("content <%s>\n", StreamDict.Content)
		// }

	case types.ObjectStreamDict:

		if o.Content == nil {
			log.Read.Println("logStream: no object stream content")
		} else {
			log.Read.Printf("logStream: objectStream content = %s\n", o.Content)
		}

		if o.ObjArray == nil {
			log.Read.Println("logStream: no object stream obj arr")
		} else {
			log.Read.Printf("logStream: objectStream objArr = %s\n", o.ObjArray)
		}

	default:
		log.Read.Println("logStream: no ObjectStreamDict")

	}

}

func decodeObjectStreamObjects(c context.Context, sd *types.StreamDict, objNr int) (*types.ObjectStreamDict, error) {
	osd, err := model.ObjectStreamDict(sd)
	if err != nil {
		return nil, errors.Wrapf(err, "decodeObjectStreamObjects: problem dereferencing object stream %d", objNr)
	}

	if log.ReadEnabled() {
		log.Read.Printf("decodeObjectStreamObjects: decoding object stream %d:\n", objNr)
	}

	// Parse all objects of this object stream and save them to ObjectStreamDict.ObjArray.
	if err = parseObjectStream(c, osd); err != nil {
		return nil, errors.Wrapf(err, "decodeObjectStreamObjects: problem decoding object stream %d\n", objNr)
	}

	if osd.ObjArray == nil {
		return nil, errors.Wrap(err, "decodeObjectStreamObjects: objArray should be set!")
	}

	if log.ReadEnabled() {
		log.Read.Printf("decodeObjectStreamObjects: decoded object stream %d:\n", objNr)
	}

	return osd, nil
}

func decodeObjectStream(c context.Context, ctx *model.Context, objNr int) error {
	entry := ctx.Table[objNr]
	if entry == nil {
		return errors.Errorf("decodeObjectStream: missing entry for obj#%d\n", objNr)
	}

	if log.ReadEnabled() {
		log.Read.Printf("decodeObjectStream: parsing object stream for obj#%d\n", objNr)
	}

	// Parse object stream from file.
	o, err := ParseObjectWithContext(c, ctx, *entry.Offset, objNr, *entry.Generation)
	if err != nil || o == nil {
		return errors.New("pdfcpu: decodeObjectStream: corrupt object stream")
	}

	// Ensure StreamDict
	sd, ok := o.(types.StreamDict)
	if !ok {
		return errors.New("pdfcpu: decodeObjectStream: corrupt object stream")
	}

	// Load encoded stream content to xRefTable.
	if err = loadEncodedStreamContent(c, ctx, &sd, false); err != nil {
		return errors.Wrapf(err, "decodeObjectStream: problem dereferencing object stream %d", objNr)
	}

	// Will only decrypt, the actual stream content is decoded later lazily.
	if err = saveDecodedStreamContent(ctx, &sd, objNr, *entry.Generation, false); err != nil {
		if log.ReadEnabled() {
			log.Read.Printf("obj %d: %s", objNr, err)
		}
		return err
	}

	// Ensure decoded objectArray for object stream dicts.
	if !sd.IsObjStm() {
		return errors.New("pdfcpu: decodeObjectStreams: corrupt object stream")
	}

	// We have an object stream.
	if log.ReadEnabled() {
		log.Read.Printf("decodeObjectStreams: object stream #%d\n", objNr)
	}

	ctx.Read.UsingObjectStreams = true

	osd, err := decodeObjectStreamObjects(c, &sd, objNr)
	if err != nil {
		return err
	}

	// Save object stream dict to xRefTableEntry.
	entry.Object = *osd

	return nil
}

// Decode all object streams so contained objects are ready to be used.
func decodeObjectStreams(c context.Context, ctx *model.Context) error {
	// Note:
	// Entry "Extends" intentionally left out.
	// No object stream collection validation necessary.

	if log.ReadEnabled() {
		log.Read.Println("decodeObjectStreams: begin")
	}

	// Get sorted slice of object numbers.
	var keys []int
	for k := range ctx.Read.ObjectStreams {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, objNr := range keys {

		if err := c.Err(); err != nil {
			return err
		}
		if err := decodeObjectStream(c, ctx, objNr); err != nil {
			return err
		}
	}

	if log.ReadEnabled() {
		log.Read.Println("decodeObjectStreams: end")
	}

	return nil
}

func handleLinearizationParmDict(ctx *model.Context, obj types.Object, objNr int) error {
	if ctx.Read.Linearized {
		// Linearization dict already processed.
		return nil
	}

	// handle linearization parm dict.
	if d, ok := obj.(types.Dict); ok && d.IsLinearizationParmDict() {

		ctx.Read.Linearized = true
		ctx.LinearizationObjs[objNr] = true
		if log.ReadEnabled() {
			log.Read.Printf("handleLinearizationParmDict: identified linearizationObj #%d\n", objNr)
		}

		a := d.ArrayEntry("H")

		if a == nil {
			return errors.Errorf("handleLinearizationParmDict: corrupt linearization dict at obj:%d - missing array entry H", objNr)
		}

		if len(a) != 2 && len(a) != 4 {
			return errors.Errorf("handleLinearizationParmDict: corrupt linearization dict at obj:%d - corrupt array entry H, needs length 2 or 4", objNr)
		}

		offset, ok := a[0].(types.Integer)
		if !ok {
			return errors.Errorf("handleLinearizationParmDict: corrupt linearization dict at obj:%d - corrupt array entry H, needs Integer values", objNr)
		}

		offset64 := int64(offset.Value())
		ctx.OffsetPrimaryHintTable = &offset64

		if len(a) == 4 {

			offset, ok := a[2].(types.Integer)
			if !ok {
				return errors.Errorf("handleLinearizationParmDict: corrupt linearization dict at obj:%d - corrupt array entry H, needs Integer values", objNr)
			}

			offset64 := int64(offset.Value())
			ctx.OffsetOverflowHintTable = &offset64
		}
	}

	return nil
}

func loadStreamDict(c context.Context, ctx *model.Context, sd *types.StreamDict, objNr, genNr int, fixLength bool) error {
	// Load encoded stream content for stream dicts into xRefTable entry.
	if err := loadEncodedStreamContent(c, ctx, sd, fixLength); err != nil {
		return errors.Wrapf(err, "dereferenceObject: problem dereferencing stream %d", objNr)
	}

	ctx.Read.BinaryTotalSize += *sd.StreamLength

	// Decode stream content.
	return saveDecodedStreamContent(ctx, sd, objNr, genNr, ctx.DecodeAllStreams)
}

func updateBinaryTotalSize(ctx *model.Context, o types.Object) {
	switch o := o.(type) {
	case types.StreamDict:
		ctx.Read.BinaryTotalSize += *o.StreamLength
	case types.ObjectStreamDict:
		ctx.Read.BinaryTotalSize += *o.StreamLength
	case types.XRefStreamDict:
		ctx.Read.BinaryTotalSize += *o.StreamLength
	}
}

func dereferenceAndLoad(c context.Context, ctx *model.Context, objNr int, entry *model.XRefTableEntry) error {
	if log.ReadEnabled() {
		log.Read.Printf("dereferenceAndLoad: dereferencing object %d\n", objNr)
	}

	// Parse object from ctx: anything goes dict, array, integer, float, streamdict...
	o, err := ParseObjectWithContext(c, ctx, *entry.Offset, objNr, *entry.Generation)
	if err != nil {
		return errors.Wrapf(err, "dereferenceAndLoad: problem dereferencing object %d", objNr)
	}
	if o == nil {
		return nil
	}

	entry.Object = o

	// Linearization dicts are validated and recorded for stats only.
	if err = handleLinearizationParmDict(ctx, o, objNr); err != nil {
		return err
	}

	// Handle stream dicts.

	if _, ok := o.(types.ObjectStreamDict); ok {
		return errors.Errorf("dereferenceAndLoad: object stream should already be dereferenced at obj:%d", objNr)
	}

	if _, ok := o.(types.XRefStreamDict); ok {
		return errors.Errorf("dereferenceAndLoad: xref stream should already be dereferenced at obj:%d", objNr)
	}

	if sd, ok := o.(types.StreamDict); ok {
		if err = loadStreamDict(c, ctx, &sd, objNr, *entry.Generation, false); err != nil {
			return err
		}
		entry.Object = sd
	}

	if log.ReadEnabled() {
		log.Read.Printf("dereferenceAndLoad: end obj %d of %d\n<%s>\n", objNr, len(ctx.Table), entry.Object)
	}

	return nil
}

func dereferenceObject(c context.Context, ctx *model.Context, objNr int) error {
	if log.ReadEnabled() {
		log.Read.Printf("dereferenceObject: begin, dereferencing object %d\n", objNr)
	}

	if objNr > ctx.MaxObjNr {
		ctx.MaxObjNr = objNr
	}

	entry := ctx.Table[objNr]

	if entry.Free {
		if log.ReadEnabled() {
			log.Read.Printf("free object %d\n", objNr)
		}
		return nil
	}

	if entry.Compressed {
		if err := decompressXRefTableEntry(ctx.XRefTable, objNr, entry); err != nil {
			return err
		}
		//log.Read.Printf("dereferenceObject: decompressed entry, Compressed=%v\n%s\n", entry.Compressed, entry.Object)
		return nil
	}

	// entry is in use.
	if log.ReadEnabled() {
		log.Read.Printf("in use object %d\n", objNr)
	}

	if entry.Offset == nil || *entry.Offset == 0 {
		if log.ReadEnabled() {
			log.Read.Printf("dereferenceObject: already decompressed or used object w/o offset -> ignored")
		}
		return nil
	}

	o := entry.Object

	if o != nil {
		// Already dereferenced.
		logStream(entry.Object)
		updateBinaryTotalSize(ctx, o)
		if log.ReadEnabled() {
			log.Read.Printf("dereferenceObject: using cached object %d of %d\n<%s>\n", objNr, ctx.MaxObjNr+1, entry.Object)
		}
		return nil
	}

	if err := dereferenceAndLoad(c, ctx, objNr, entry); err != nil {
		return err
	}

	logStream(entry.Object)

	return nil
}

func dereferenceObjectsSorted(c context.Context, ctx *model.Context) error {
	xRefTable := ctx.XRefTable
	var keys []int
	for k := range xRefTable.Table {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, objNr := range keys {
		if err := c.Err(); err != nil {
			return err
		}
		if err := dereferenceObject(c, ctx, objNr); err != nil {
			return err
		}
	}

	for _, objNr := range keys {
		entry := xRefTable.Table[objNr]
		if entry.Free || entry.Compressed {
			continue
		}
		if err := c.Err(); err != nil {
			return err
		}
		model.ProcessRefCounts(xRefTable, entry.Object)
	}

	return nil
}

func dereferenceObjectsRaw(c context.Context, ctx *model.Context) error {
	xRefTable := ctx.XRefTable
	for objNr := range xRefTable.Table {
		if err := c.Err(); err != nil {
			return err
		}
		if err := dereferenceObject(c, ctx, objNr); err != nil {
			return err
		}
	}

	for objNr := range xRefTable.Table {
		entry := xRefTable.Table[objNr]
		if entry.Free || entry.Compressed {
			continue
		}
		if err := c.Err(); err != nil {
			return err
		}
		model.ProcessRefCounts(xRefTable, entry.Object)
	}

	return nil
}

// Dereferences all objects including compressed objects from object streams.
func dereferenceObjects(c context.Context, ctx *model.Context) error {
	if log.ReadEnabled() {
		log.Read.Println("dereferenceObjects: begin")
	}

	var err error

	if log.StatsEnabled() {
		err = dereferenceObjectsSorted(c, ctx)
	} else {
		err = dereferenceObjectsRaw(c, ctx)
	}

	if err != nil {
		return err
	}

	if log.ReadEnabled() {
		log.Read.Println("dereferenceObjects: end")
	}

	return nil
}

// Locate a possible Version entry (since V1.4) in the catalog
// and record this as rootVersion (as opposed to headerVersion).
func identifyRootVersion(xRefTable *model.XRefTable) error {
	if log.ReadEnabled() {
		log.Read.Println("identifyRootVersion: begin")
	}

	// Try to get Version from Root.
	rootVersionStr, err := xRefTable.ParseRootVersion()
	if err != nil {
		return err
	}

	if rootVersionStr == nil {
		return nil
	}

	// Validate version and save corresponding constant to xRefTable.
	rootVersion, err := model.PDFVersion(*rootVersionStr)
	if err != nil {
		return errors.Wrapf(err, "identifyRootVersion: unknown PDF Root version: %s\n", *rootVersionStr)
	}

	xRefTable.RootVersion = &rootVersion

	// since V1.4 the header version may be overridden by a Version entry in the catalog.
	if *xRefTable.HeaderVersion < model.V14 {
		if log.InfoEnabled() {
			log.Info.Printf("identifyRootVersion: PDF version is %s - will ignore root version: %s\n", xRefTable.HeaderVersion, *rootVersionStr)
		}
	}

	if log.ReadEnabled() {
		log.Read.Println("identifyRootVersion: end")
	}

	return nil
}

// Parse all Objects including stream content from file and save to the corresponding xRefTableEntries.
// This includes processing of object streams and linearization dicts.
func dereferenceXRefTable(c context.Context, ctx *model.Context, conf *model.Configuration) error {
	if log.ReadEnabled() {
		log.Read.Println("dereferenceXRefTable: begin")
	}

	xRefTable := ctx.XRefTable

	// Note for encrypted files:
	// Mandatory provide userpw to open & display file.
	// Access may be restricted (Decode access privileges).
	// Optionally provide ownerpw in order to gain unrestricted access.
	if err := checkForEncryption(c, ctx); err != nil {
		return err
	}
	//fmt.Println("pw authenticated")

	// Prepare decompressed objects.
	if err := decodeObjectStreams(c, ctx); err != nil {
		return err
	}

	// For each xRefTableEntry assign a Object either by parsing from file or pointing to a decompressed object.
	if err := dereferenceObjects(c, ctx); err != nil {
		return err
	}

	// Identify an optional Version entry in the root object/catalog.
	if err := identifyRootVersion(xRefTable); err != nil {
		return err
	}

	if log.ReadEnabled() {
		log.Read.Println("dereferenceXRefTable: end")
	}

	return nil
}

func handleUnencryptedFile(ctx *model.Context) error {
	if ctx.Cmd == model.DECRYPT || ctx.Cmd == model.SETPERMISSIONS {
		return errors.New("pdfcpu: this file is not encrypted")
	}

	if ctx.Cmd != model.ENCRYPT {
		return nil
	}

	// Encrypt subcommand found.

	if ctx.OwnerPW == "" {
		return errors.New("pdfcpu: please provide owner password and optional user password")
	}

	return nil
}

func needsOwnerAndUserPassword(cmd model.CommandMode) bool {
	return cmd == model.CHANGEOPW || cmd == model.CHANGEUPW || cmd == model.SETPERMISSIONS
}

func handlePermissions(ctx *model.Context) error {
	// AES256 Validate permissions
	ok, err := validatePermissions(ctx)
	if err != nil {
		return err
	}

	if !ok {
		return errors.New("pdfcpu: corrupted permissions after upw ok")
	}

	if ctx.OwnerPW == "" && ctx.UserPW == "" {
		return nil
	}

	// Double check minimum permissions for pdfcpu processing.
	if !hasNeededPermissions(ctx.Cmd, ctx.E) {
		return errors.New("pdfcpu: operation restricted via pdfcpu's permission bits setting")
	}

	return nil
}

func setupEncryptionKey(ctx *model.Context, d types.Dict) (err error) {
	if ctx.E, err = supportedEncryption(ctx, d); err != nil {
		return err
	}

	if ctx.E.ID, err = ctx.IDFirstElement(); err != nil {
		return err
	}

	var ok bool

	//fmt.Printf("opw: <%s> upw: <%s> \n", ctx.OwnerPW, ctx.UserPW)

	// Validate the owner password aka. permissions/master password.
	if ok, err = validateOwnerPassword(ctx); err != nil {
		return err
	}

	// If the owner password does not match we generally move on if the user password is correct
	// unless we need to insist on a correct owner password due to the specific command in progress.
	if !ok && needsOwnerAndUserPassword(ctx.Cmd) {
		return errors.New("pdfcpu: please provide the owner password with -opw")
	}

	// Generally the owner password, which is also regarded as the master password or set permissions password
	// is sufficient for moving on. A password change is an exception since it requires both current passwords.
	if ok && !needsOwnerAndUserPassword(ctx.Cmd) {
		// AES256 Validate permissions
		if ok, err = validatePermissions(ctx); err != nil {
			return err
		}
		if !ok {
			return errors.New("pdfcpu: corrupted permissions after opw ok")
		}
		return nil
	}

	// Validate the user password aka. document open password.
	if ok, err = validateUserPassword(ctx); err != nil {
		return err
	}
	if !ok {
		return ErrWrongPassword
	}

	//fmt.Printf("upw ok: %t\n", ok)

	return handlePermissions(ctx)
}

func checkForEncryption(c context.Context, ctx *model.Context) error {
	indRef := ctx.Encrypt
	if indRef == nil {
		// This file is not encrypted.
		return handleUnencryptedFile(ctx)
	}

	// This file is encrypted.
	if log.ReadEnabled() {
		log.Read.Printf("Encryption: %v\n", indRef)
	}

	if ctx.Cmd == model.ENCRYPT {
		// We want to encrypt this file.
		return errors.New("pdfcpu: this file is already encrypted")
	}

	// Dereference encryptDict.
	d, err := dereferencedDict(c, ctx, indRef.ObjectNumber.Value())
	if err != nil {
		return err
	}

	if log.ReadEnabled() {
		log.Read.Printf("%s\n", d)
	}

	// We need to decrypt this file in order to read it.
	return setupEncryptionKey(ctx, d)
}
