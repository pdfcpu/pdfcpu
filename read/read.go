// Package read provides for parsing a PDF file into memory.
//
// The in memory representation of a PDF file is called a PDFContext.
//
// The PDFContext is a container for the PDF cross reference table and stats.
package read

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/hhrutter/pdfcpu/crypto"
	"github.com/hhrutter/pdfcpu/filter"
	"github.com/hhrutter/pdfcpu/log"
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

const (
	defaultBufSize   = 1024
	unknownDelimiter = byte(0)
)

// PDFFile reads in a PDFFile and generates a PDFContext, an in-memory representation containing a cross reference table.
func PDFFile(fileName string, config *types.Configuration) (*types.PDFContext, error) {

	log.Debug.Println("PDFFile: begin")

	file, err := os.Open(fileName)
	if err != nil {
		return nil, errors.Wrapf(err, "can't open %q", fileName)
	}

	defer func() {
		file.Close()
	}()

	ctx, err := types.NewPDFContext(fileName, file, config)
	if err != nil {
		return nil, err
	}

	if ctx.Reader15 {
		log.Info.Println("PDF Version 1.5 conforming reader")
	} else {
		log.Info.Println("PDF Version 1.4 conforming reader - no object streams or xrefstreams allowed")
	}

	// Populate xRefTable.
	err = readXRefTable(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "xRefTable failed")
	}

	// Make all objects explicitly available (load into memory) in corresponding xRefTable entries.
	// Also decode any involved object streams.
	err = dereferenceXRefTable(ctx, config)
	if err != nil {
		return nil, err
	}

	log.Debug.Println("PDFFile: end")

	return ctx, nil
}

// ScanLines is a split function for a Scanner that returns each line of
// text, stripped of any trailing end-of-line marker. The returned line may
// be empty. The end-of-line marker is one carriage return followed
// by one newline or one carriage return or one newline.
// The last non-empty line of input will be returned even if it has no newline.
func scanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {

	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	indCR := bytes.IndexByte(data, '\r')
	indLF := bytes.IndexByte(data, '\n')

	switch {

	case indCR >= 0 && indLF >= 0:
		if indCR < indLF {
			if indLF == indCR+1 {
				// 0x0D0A
				return indLF + 1, data[0:indCR], nil
			}
			// 0x0D ... 0x0A
			return indCR + 1, data[0:indCR], nil
		}
		// 0x0A ... 0x0D
		return indLF + 1, data[0:indLF], nil

	case indCR >= 0:
		// We have a full carriage return terminated line.
		return indCR + 1, data[0:indCR], nil

	case indLF >= 0:
		// We have a full newline-terminated line.
		return indLF + 1, data[0:indLF], nil

	}

	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}

	// Request more data.
	return 0, nil, nil
}

func newPositionedReader(rs io.ReadSeeker, offset *int64) (*bufio.Reader, error) {

	if _, err := rs.Seek(*offset, 0); err != nil {
		return nil, err
	}

	log.Debug.Printf("newPositionedReader: positioned to offset: %d\n", *offset)

	return bufio.NewReader(rs), nil
}

// Get the file offset of the last XRefSection.
// Go to end of file and search backwards for the first occurrence of startxref {offset} %%EOF
func offsetLastXRefSection(ra io.ReaderAt, fileSize int64) (*int64, error) {

	var bufSize int64 = defaultBufSize

	off := fileSize - defaultBufSize
	if off < 0 {
		off = 0
		bufSize = fileSize
	}
	buf := make([]byte, bufSize)

	log.Debug.Printf("offsetLastXRefSection at %d\n", off)

	if _, err := ra.ReadAt(buf, off); err != nil {
		return nil, err
	}

	i := strings.LastIndex(string(buf), "startxref")
	if i == -1 {
		return nil, errors.New("cannot find last xrefsection pointer")
	}

	buf = buf[i+len("startxref"):]
	posEOF := strings.Index(string(buf), "%%EOF")
	if posEOF == -1 {
		return nil, errors.New("no matching %%EOF for startxref")
	}

	buf = buf[:posEOF]
	offset, err := strconv.ParseInt(strings.TrimSpace(string(buf)), 10, 64)
	if err != nil {
		return nil, errors.New("corrupted xref section")
	}

	log.Debug.Printf("Offset last xrefsection: %d\n", offset)

	return &offset, nil
}

// Read next subsection entry and generate corresponding xref table entry.
func parseXRefTableEntry(s *bufio.Scanner, xRefTable *types.XRefTable, objectNumber int) error {

	log.Debug.Println("parseXRefTableEntry: begin")

	line, err := scanLine(s)
	if err != nil {
		return err
	}

	if xRefTable.Exists(objectNumber) {
		log.Debug.Printf("parseXRefTableEntry: end - Skip entry %d - already assigned\n", objectNumber)
		return nil
	}

	fields := strings.Fields(line)
	if len(fields) != 3 ||
		len(fields[0]) != 10 || len(fields[1]) != 5 || len(fields[2]) != 1 {
		return errors.New("parseXRefTableEntry: corrupt xref subsection header")
	}

	offset, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return err
	}

	generation, err := strconv.Atoi(fields[1])
	if err != nil {
		return err
	}

	entryType := fields[2]
	if entryType != "f" && entryType != "n" {
		return errors.New("parseXRefTableEntry: corrupt xref subsection entry")
	}

	var xRefTableEntry types.XRefTableEntry

	if entryType == "n" {

		// in use object

		log.Debug.Printf("parseXRefTableEntry: Object #%d is in use at offset=%d, generation=%d\n", objectNumber, offset, generation)

		if offset == 0 {
			log.Info.Printf("parseXRefTableEntry: Skip entry for in use object #%d with offset 0\n", objectNumber)
			return nil
		}

		xRefTableEntry =
			types.XRefTableEntry{
				Free:       false,
				Offset:     &offset,
				Generation: &generation}

	} else {

		// free object

		log.Debug.Printf("parseXRefTableEntry: Object #%d is unused, next free is object#%d, generation=%d\n", objectNumber, offset, generation)

		xRefTableEntry =
			types.XRefTableEntry{
				Free:       true,
				Offset:     &offset,
				Generation: &generation}

	}

	log.Debug.Printf("parseXRefTableEntry: Insert new xreftable entry for Object %d\n", objectNumber)

	xRefTable.Table[objectNumber] = &xRefTableEntry

	log.Debug.Println("parseXRefTableEntry: end")

	return nil
}

// Process xRef table subsection and create corrresponding xRef table entries.
func parseXRefTableSubSection(s *bufio.Scanner, xRefTable *types.XRefTable, fields []string) error {

	log.Debug.Println("parseXRefTableSubSection: begin")

	startObjNumber, err := strconv.Atoi(fields[0])
	if err != nil {
		return err
	}

	objCount, err := strconv.Atoi(fields[1])
	if err != nil {
		return err
	}

	log.Debug.Printf("detected xref subsection, startObj=%d length=%d\n", startObjNumber, objCount)

	// Process all entries of this subsection into xRefTable entries.
	for i := 0; i < objCount; i++ {
		if err = parseXRefTableEntry(s, xRefTable, startObjNumber+i); err != nil {
			return err
		}
	}

	log.Debug.Println("parseXRefTableSubSection: end")

	return nil
}

// Parse compressed object.
func compressedObject(s string) (types.PDFObject, error) {

	log.Debug.Println("compressedObject: begin")

	pdfObject, err := parseObject(&s)
	if err != nil {
		return nil, err
	}

	pdfDict, ok := pdfObject.(types.PDFDict)
	if !ok {
		// return trivial PDFObject: PDFInteger, PDFArray, etc.
		log.Debug.Println("compressedObject: end, any other than dict")
		return pdfObject, nil
	}

	streamLength, streamLengthRef := pdfDict.Length()
	if streamLength == nil && streamLengthRef == nil {
		// return PDFDict
		log.Debug.Println("compressedObject: end, dict")
		return pdfDict, nil
	}

	return nil, errors.New("compressedObject: Stream objects are not to be stored in an object stream")
}

// Parse all objects of an object stream and save them into objectStreamDict.ObjArray.
func parseObjectStream(objectStreamDict *types.PDFObjectStreamDict) error {

	log.Debug.Printf("parseObjectStream begin: decoding %d objects.\n", objectStreamDict.ObjCount)

	decodedContent := objectStreamDict.Content
	prolog := decodedContent[:objectStreamDict.FirstObjOffset]

	objs := strings.Fields(string(prolog))
	if len(objs)%2 > 0 {
		return errors.New("parseObjectStream: corrupt object stream dict")
	}

	// e.g., 10 0 11 25 = 2 Objects: #10 @ offset 0, #11 @ offset 25

	var objArray types.PDFArray

	var offsetOld int

	for i := 0; i < len(objs); i += 2 {

		offset, err := strconv.Atoi(objs[i+1])
		if err != nil {
			return err
		}

		offset += objectStreamDict.FirstObjOffset

		if i > 0 {
			dstr := string(decodedContent[offsetOld:offset])
			log.Debug.Printf("parseObjectStream: objString = %s\n", dstr)
			pdfObject, err := compressedObject(dstr)
			if err != nil {
				return err
			}

			log.Debug.Printf("parseObjectStream: [%d] = obj %s:\n%s\n", i/2-1, objs[i-2], pdfObject)
			objArray = append(objArray, pdfObject)
		}

		if i == len(objs)-2 {
			dstr := string(decodedContent[offset:])
			log.Debug.Printf("parseObjectStream: objString = %s\n", dstr)
			pdfObject, err := compressedObject(dstr)
			if err != nil {
				return err
			}

			log.Debug.Printf("parseObjectStream: [%d] = obj %s:\n%s\n", i/2, objs[i], pdfObject)
			objArray = append(objArray, pdfObject)
		}

		offsetOld = offset
	}

	objectStreamDict.ObjArray = objArray

	log.Debug.Println("parseObjectStream end")

	return nil
}

// For each object embedded in this xRefStream create the corresponding xRef table entry.
func extractXRefTableEntriesFromXRefStream(buf []byte, xRefStreamDict types.PDFXRefStreamDict, ctx *types.PDFContext) error {

	log.Debug.Printf("extractXRefTableEntriesFromXRefStream begin")

	// Note:
	// A value of zero for an element in the W array indicates that the corresponding field shall not be present in the stream,
	// and the default value shall be used, if there is one.
	// If the first element is zero, the type field shall not be present, and shall default to type 1.

	i1 := xRefStreamDict.W[0]
	i2 := xRefStreamDict.W[1]
	i3 := xRefStreamDict.W[2]

	xrefEntryLen := i1 + i2 + i3
	log.Debug.Printf("extractXRefTableEntriesFromXRefStream: begin xrefEntryLen = %d\n", xrefEntryLen)

	if len(buf)%xrefEntryLen > 0 {
		return errors.New("extractXRefTableEntriesFromXRefStream: corrupt xrefstream")
	}

	objCount := len(xRefStreamDict.Objects)
	log.Debug.Printf("extractXRefTableEntriesFromXRefStream: objCount:%d %v\n", objCount, xRefStreamDict.Objects)

	log.Debug.Printf("extractXRefTableEntriesFromXRefStream: len(buf):%d objCount*xrefEntryLen:%d\n", len(buf), objCount*xrefEntryLen)
	if len(buf) != objCount*xrefEntryLen {
		return errors.New("extractXRefTableEntriesFromXRefStream: corrupt xrefstream")
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

	for i := 0; i < len(buf) && j < len(xRefStreamDict.Objects); i += xrefEntryLen {

		objectNumber := xRefStreamDict.Objects[j]

		i2Start := i + i1
		c2 := bufToInt64(buf[i2Start : i2Start+i2])
		c3 := bufToInt64(buf[i2Start+i2 : i2Start+i2+i3])

		var xRefTableEntry types.XRefTableEntry

		switch buf[i] {

		case 0x00:
			// free object
			log.Debug.Printf("extractXRefTableEntriesFromXRefStream: Object #%d is unused, next free is object#%d, generation=%d\n", objectNumber, c2, c3)
			g := int(c3)

			xRefTableEntry =
				types.XRefTableEntry{
					Free:       true,
					Compressed: false,
					Offset:     &c2,
					Generation: &g}

		case 0x01:
			// in use object
			log.Debug.Printf("extractXRefTableEntriesFromXRefStream: Object #%d is in use at offset=%d, generation=%d\n", objectNumber, c2, c3)
			g := int(c3)

			xRefTableEntry =
				types.XRefTableEntry{
					Free:       false,
					Compressed: false,
					Offset:     &c2,
					Generation: &g}

		case 0x02:
			// compressed object
			// generation always 0.
			log.Debug.Printf("extractXRefTableEntriesFromXRefStream: Object #%d is compressed at obj %5d[%d]\n", objectNumber, c2, c3)
			objNumberRef := int(c2)
			objIndex := int(c3)

			xRefTableEntry =
				types.XRefTableEntry{
					Free:            false,
					Compressed:      true,
					ObjectStream:    &objNumberRef,
					ObjectStreamInd: &objIndex}

			ctx.Read.ObjectStreams[objNumberRef] = true

		}

		if ctx.XRefTable.Exists(objectNumber) {
			log.Debug.Printf("extractXRefTableEntriesFromXRefStream: Skip entry %d - already assigned\n", objectNumber)
		} else {
			ctx.Table[objectNumber] = &xRefTableEntry
		}

		j++
	}

	log.Debug.Println("extractXRefTableEntriesFromXRefStream: end")

	return nil
}

func xRefStreamDict(ctx *types.PDFContext, o types.PDFObject, objNr int, streamOffset int64) (*types.PDFXRefStreamDict, error) {

	// must be pdfDict
	pdfDict, ok := o.(types.PDFDict)
	if !ok {
		return nil, errors.New("xRefStreamDict: no pdfDict")
	}

	// Parse attributes for stream object.
	streamLength, streamLengthObjNr := pdfDict.Length()
	if streamLength == nil && streamLengthObjNr == nil {
		return nil, errors.New("xRefStreamDict: no \"Length\" entry")
	}

	filterPipeline, err := pdfFilterPipeline(ctx, pdfDict)
	if err != nil {
		return nil, err
	}

	// We have a stream object.
	log.Debug.Printf("xRefStreamDict: streamobject #%d\n", objNr)
	pdfStreamDict := types.NewPDFStreamDict(pdfDict, streamOffset, streamLength, streamLengthObjNr, filterPipeline)

	if _, err = LoadEncodedStreamContent(ctx, &pdfStreamDict); err != nil {
		return nil, err
	}

	// Decode xrefstream content
	if err = setDecodedStreamContent(nil, &pdfStreamDict, 0, 0, true); err != nil {
		return nil, errors.Wrapf(err, "xRefStreamDict: cannot decode stream for obj#:%d\n", objNr)
	}

	return parseXRefStreamDict(pdfStreamDict)
}

// Parse xRef stream and setup xrefTable entries for all embedded objects and the xref stream dict.
func parseXRefStream(rd io.Reader, offset *int64, ctx *types.PDFContext) (prevOffset *int64, err error) {

	log.Debug.Printf("parseXRefStream: begin at offset %d\n", *offset)

	buf, endInd, streamInd, streamOffset, err := buffer(rd)
	if err != nil {
		return nil, err
	}

	log.Debug.Printf("parseXRefStream: endInd=%[1]d(%[1]x) streamInd=%[2]d(%[2]x)\n", endInd, streamInd)

	line := string(buf)

	// We expect a stream and therefore "stream" before "endobj" if "endobj" within buffer.
	// There is no guarantee that "endobj" is contained in this buffer for large streams!
	if streamInd < 0 || (endInd > 0 && endInd < streamInd) {
		return nil, errors.New("parseXRefStream: corrupt pdf file")
	}

	// Init object parse buf.
	l := line[:streamInd]

	objectNumber, generationNumber, err := parseObjectAttributes(&l)
	if err != nil {
		return nil, err
	}

	// parse this object
	log.Debug.Printf("parseXRefStream: xrefstm obj#:%d gen:%d\n", *objectNumber, *generationNumber)
	log.Debug.Printf("parseXRefStream: dereferencing object %d\n", *objectNumber)
	pdfObject, err := parseObject(&l)
	if err != nil {
		return nil, errors.Wrapf(err, "parseXRefStream: no pdfObject")
	}

	log.Debug.Printf("parseXRefStream: we have a pdfObject: %s\n", pdfObject)

	streamOffset += *offset
	pdfXRefStreamDict, err := xRefStreamDict(ctx, pdfObject, *objectNumber, streamOffset)
	if err != nil {
		return nil, err
	}
	// We have an xref stream object

	err = parseTrailerInfo(pdfXRefStreamDict.PDFDict, ctx.XRefTable)
	if err != nil {
		return nil, err
	}

	// Parse xRefStream and create xRefTable entries for embedded objects.
	err = extractXRefTableEntriesFromXRefStream(pdfXRefStreamDict.Content, *pdfXRefStreamDict, ctx)
	if err != nil {
		return nil, err
	}

	// Create xRefTableEntry for XRefStreamDict.
	entry :=
		types.XRefTableEntry{
			Free:       false,
			Offset:     offset,
			Generation: generationNumber,
			Object:     *pdfXRefStreamDict}

	log.Debug.Printf("parseXRefStream: Insert new xRefTable entry for Object %d\n", *objectNumber)

	ctx.Table[*objectNumber] = &entry
	ctx.Read.XRefStreams[*objectNumber] = true
	prevOffset = pdfXRefStreamDict.PreviousOffset

	log.Debug.Println("parseXRefStream: end")

	return prevOffset, nil
}

// Parse an xRefStream for a hybrid PDF file.
func parseHybridXRefStream(offset *int64, ctx *types.PDFContext) error {

	log.Debug.Println("parseHybridXRefStream: begin")

	rd, err := newPositionedReader(ctx.Read.File, offset)
	if err != nil {
		return err
	}

	prevOffset, err := parseXRefStream(rd, offset, ctx)
	if err != nil {
		return err
	}

	if prevOffset != nil {
		return errors.New("parseHybridXRefStream: previous xref stream not allowed")
	}

	log.Debug.Println("parseHybridXRefStream: end")

	return nil
}

// Parse trailer dict and return any offset of a previous xref section.
func parseTrailerInfo(dict types.PDFDict, xRefTable *types.XRefTable) error {

	log.Debug.Println("parseTrailerInfo begin")

	if _, found := dict.Find("Encrypt"); found {
		encryptObjRef := dict.IndirectRefEntry("Encrypt")
		if encryptObjRef != nil {
			xRefTable.Encrypt = encryptObjRef
			log.Debug.Printf("parseTrailerInfo: Encrypt object: %s\n", *xRefTable.Encrypt)
		}
	}

	if xRefTable.Size == nil {
		size := dict.Size()
		if size == nil {
			return errors.New("parseTrailerInfo: missing entry \"Size\"")
		}
		xRefTable.Size = size
	}

	if xRefTable.Root == nil {
		rootObjRef := dict.IndirectRefEntry("Root")
		if rootObjRef == nil {
			return errors.New("parseTrailerInfo: missing entry \"Root\"")
		}
		xRefTable.Root = rootObjRef
		log.Debug.Printf("parseTrailerInfo: Root object: %s\n", *xRefTable.Root)
	}

	if xRefTable.Info == nil {
		infoObjRef := dict.IndirectRefEntry("Info")
		if infoObjRef != nil {
			xRefTable.Info = infoObjRef
			log.Debug.Printf("parseTrailerInfo: Info object: %s\n", *xRefTable.Info)
		}
	}

	if xRefTable.ID == nil {
		idArray := dict.PDFArrayEntry("ID")
		if idArray != nil {
			xRefTable.ID = idArray
			log.Debug.Printf("parseTrailerInfo: ID object: %s\n", *xRefTable.ID)
		} else if xRefTable.Encrypt != nil {
			return errors.New("parseTrailerInfo: missing entry \"ID\"")
		}
	}

	log.Debug.Println("parseTrailerInfo end")

	return nil
}

func parseTrailerDict(trailerDict types.PDFDict, ctx *types.PDFContext) (*int64, error) {

	log.Debug.Println("parseTrailerDict begin")

	xRefTable := ctx.XRefTable

	err := parseTrailerInfo(trailerDict, xRefTable)
	if err != nil {
		return nil, err
	}

	if arr := trailerDict.PDFArrayEntry("AdditionalStreams"); arr != nil {
		log.Debug.Printf("parseTrailerInfo: found AdditionalStreams: %s\n", arr)
		a := types.PDFArray{}
		for _, value := range *arr {
			if indRef, ok := value.(types.PDFIndirectRef); ok {
				a = append(a, indRef)
			}
		}
		xRefTable.AdditionalStreams = &a
	}

	offset := trailerDict.Prev()
	if offset != nil {
		log.Debug.Printf("parseTrailerDict: previous xref table section offset:%d\n", *offset)
	}

	offsetXRefStream := trailerDict.Int64Entry("XRefStm")
	if offsetXRefStream == nil {
		// No cross reference stream.
		if !ctx.Reader15 && xRefTable.Version() >= types.V14 && !ctx.Read.Hybrid {
			return nil, errors.Errorf("parseTrailerDict: PDF1.4 conformant reader: found incompatible version: %s", xRefTable.VersionString())
		}
		log.Debug.Println("parseTrailerDict end")
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
		if err := parseHybridXRefStream(offsetXRefStream, ctx); err != nil {
			return nil, err
		}
	}

	log.Debug.Println("parseTrailerDict end")

	return offset, nil
}

func scanLine(s *bufio.Scanner) (string, error) {
	for i := 0; i <= 1; i++ {
		if ok := s.Scan(); !ok {
			err := s.Err()
			if err != nil {
				return "", err
			}
			return "", errors.New("scanner: returning nothing")
		}
		if len(s.Text()) > 0 {
			break
		}
	}
	return s.Text(), nil
}

func scanTrailerDict(s *bufio.Scanner, startTag bool) (string, error) {

	var buf bytes.Buffer
	var line string
	var err error

	if !startTag {
		// scan for dict start tag <<
		for strings.Index(line, "<<") < 0 {
			line, err = scanLine(s)
			if err != nil {
				return "", err
			}
			buf.WriteString(line)
			buf.WriteString(" ")
		}
	}

	// scan for dict end tag >>
	for strings.Index(line, ">>") < 0 {
		line, err = scanLine(s)
		if err != nil {
			return "", err
		}
		buf.WriteString(line)
		buf.WriteString(" ")
	}

	return buf.String(), nil
}

// Parse xRef section into corresponding number of xRef table entries.
func parseXRefSection(s *bufio.Scanner, ctx *types.PDFContext) (*int64, error) {

	log.Debug.Println("parseXRefSection begin")

	line, err := scanLine(s)
	if err != nil {
		return nil, err
	}

	log.Debug.Printf("parseXRefSection: <%s>\n", line)

	fields := strings.Fields(line)

	// Process all sub sections of this xRef section.
	for !strings.HasPrefix(line, "trailer") && len(fields) == 2 {

		if err = parseXRefTableSubSection(s, ctx.XRefTable, fields); err != nil {
			return nil, err
		}

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

	log.Debug.Println("parseXRefSection: All subsections read!")

	if !strings.HasPrefix(line, "trailer") {
		return nil, errors.Errorf("xrefsection: missing trailer dict, line = <%s>", line)
	}

	log.Debug.Println("parseXRefSection: parsing trailer dict..")

	var trailerString string

	if line != "trailer" {
		trailerString = line[7:]
		log.Debug.Printf("parseXRefSection: trailer leftover: <%s>\n", trailerString)
	} else {
		log.Debug.Printf("line (len %d) <%s>\n", len(line), line)
	}

	// Unless trailerDict already scanned into trailerString
	if strings.Index(trailerString, ">>") == -1 {

		// scan lines until we have the complete trailer dict:  << ... >>
		trailerDictString, err := scanTrailerDict(s, strings.Index(trailerString, "<<") > 0)
		if err != nil {
			return nil, err
		}

		trailerString += trailerDictString
	}

	log.Debug.Printf("parseXRefSection: trailerString: (len:%d) <%s>\n", len(trailerString), trailerString)

	pdfObject, err := parseObject(&trailerString)
	if err != nil {
		return nil, err
	}

	trailerDict, ok := pdfObject.(types.PDFDict)
	if !ok {
		return nil, errors.New("parseXRefSection: corrupt trailer dict")
	}

	log.Debug.Printf("parseXRefSection: trailerDict:\n%s\n", trailerDict)

	offset, err := parseTrailerDict(trailerDict, ctx)
	if err != nil {
		return nil, err
	}

	log.Debug.Println("parseXRefSection end")

	return offset, nil
}

// Get version from first line of file.
// Beginning with PDF 1.4, the Version entry in the document’s catalog dictionary
// (located via the Root entry in the file’s trailer, as described in 7.5.5, "File Trailer"),
// if present, shall be used instead of the version specified in the Header.
// Save PDF Version from header to xRefTable.
// The header version comes as the first line of the file.
func headerVersion(ra io.ReaderAt) (*types.PDFVersion, error) {

	log.Debug.Println("headerVersion begin")

	// Get first line of file which holds the version of this PDFFile.
	// We call this the header version.

	buf := make([]byte, 10)
	if _, err := ra.ReadAt(buf, 0); err != nil {
		return nil, err
	}

	// Parse the PDF-Version.

	prefix := "%PDF-"

	s := strings.TrimSpace(string(buf))

	if len(s) < 8 || !strings.HasPrefix(s, prefix) {
		return nil, errors.New("headerVersion: corrupt pfd file - no header version available")
	}

	pdfVersion, err := types.Version(s[len(prefix) : len(prefix)+3])
	if err != nil {
		return nil, errors.Wrapf(err, "headerVersion: unknown PDF Header Version")
	}

	log.Debug.Printf("headerVersion: end, found header version: %s\n", types.VersionString(pdfVersion))

	return &pdfVersion, nil
}

// Build XRefTable by reading XRef streams or XRef sections.
func buildXRefTableStartingAt(ctx *types.PDFContext, offset *int64) error {

	log.Debug.Println("buildXRefTableStartingAt: begin")

	file := ctx.Read.File

	hv, err := headerVersion(file)
	if err != nil {
		return err
	}

	ctx.HeaderVersion = hv

	for offset != nil {

		rd, err := newPositionedReader(file, offset)
		if err != nil {
			return err
		}

		s := bufio.NewScanner(rd)
		s.Split(scanLines)

		line, err := scanLine(s)
		if err != nil {
			return err
		}

		log.Debug.Printf("line: <%s>\n", line)

		if line != "xref" {

			log.Debug.Println("buildXRefTableStartingAt: found xref stream")
			ctx.Read.UsingXRefStreams = true
			rd, err = newPositionedReader(file, offset)
			if err != nil {
				return err
			}

			if offset, err = parseXRefStream(rd, offset, ctx); err != nil {
				return err
			}

		} else {

			log.Debug.Println("buildXRefTableStartingAt: found xref section")
			if offset, err = parseXRefSection(s, ctx); err != nil {
				return err
			}

		}
	}

	log.Debug.Println("buildXRefTableStartingAt: end")

	return nil
}

// Populate the cross reference table for this PDF file.
// Goto offset of first xref table entry.
// Can be "xref" or indirect object reference eg. "34 0 obj"
// Keep digesting xref sections as long as there is a defined previous xref section
// and build up the xref table along the way.
func readXRefTable(ctx *types.PDFContext) (err error) {

	log.Debug.Println("readXRefTable: begin")

	offset, err := offsetLastXRefSection(ctx.Read.File, ctx.Read.FileSize)
	if err != nil {
		return
	}

	err = buildXRefTableStartingAt(ctx, offset)
	if err == io.EOF {
		return errors.Wrap(err, "readXRefTable: unexpected eof")
	}
	if err != nil {
		return
	}

	// Log list of free objects (not the "free list").
	//log.Debug.Printf("freelist: %v\n", ctx.FreeObjects)

	// Ensure valid freelist of objects.
	err = ctx.EnsureValidFreeList()
	if err != nil {
		return
	}

	log.Debug.Println("readXRefTable: end")

	return
}

func growBufBy(buf []byte, size int, rd io.Reader) ([]byte, error) {

	b := make([]byte, size)

	n, err := rd.Read(b)
	if err != nil {
		return nil, err
	}
	log.Debug.Printf("growBufBy: Read %d bytes\n", n)

	return append(buf, b...), nil
}

func nextStreamOffset(line string, streamInd int) (off int) {

	off = streamInd + len("stream")

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
func buffer(rd io.Reader) (buf []byte, endInd int, streamInd int, streamOffset int64, err error) {

	// process: # gen obj ... obj dict ... {stream ... data ... endstream} ... endobj
	//                                    streamInd                            endInd
	//                                  -1 if absent                        -1 if absent

	log.Debug.Println(" buffer: begin")

	endInd, streamInd = -1, -1

	for endInd < 0 && streamInd < 0 {

		buf, err = growBufBy(buf, defaultBufSize, rd)
		if err != nil {
			return nil, 0, 0, 0, err
		}

		line := string(buf)
		endInd = strings.Index(line, "endobj")
		streamInd = strings.Index(line, "stream")

		if endInd > 0 && (streamInd < 0 || streamInd > endInd) {
			// No stream marker in buf detected.
			break
		}

		// For very rare cases where "stream" also occurs within obj dict
		// we need to find the last "stream" marker before a possible end marker.
		for streamInd > 0 && !keywordStreamRightAfterEndOfDict(line, streamInd) {
			lastStreamMarker(&streamInd, endInd, line)
		}

		log.Debug.Printf("buffer: endInd=%d streamInd=%d\n", endInd, streamInd)

		if streamInd > 0 {

			// streamOffset ... the offset where the actual stream data begins.
			//                  is right after the eol after "stream".
			need := streamInd + len("stream") + 2 // 2 = maxLen(eol)

			if len(line) < need {

				// to prevent buffer overflow.
				buf, err = growBufBy(buf, need-len(line), rd)
				if err != nil {
					return nil, 0, 0, 0, err
				}

				line = string(buf)
			}

			streamOffset = int64(nextStreamOffset(line, streamInd))
		}
	}

	log.Debug.Printf("buffer: end, returned bufsize=%d streamOffset=%d\n", len(buf), streamOffset)

	return buf, endInd, streamInd, streamOffset, nil
}

// return true if 'stream' follows end of dict: >>{whitespace}stream
func keywordStreamRightAfterEndOfDict(buf string, streamInd int) bool {

	log.Debug.Println("keywordStreamRightAfterEndOfDict: begin")

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

	log.Debug.Printf("keywordStreamRightAfterEndOfDict: end, %v\n", ok)

	return ok
}

// Return the filter pipeline associated with this stream dict.
func pdfFilterPipeline(ctx *types.PDFContext, pdfDict types.PDFDict) ([]types.PDFFilter, error) {

	log.Debug.Println("pdfFilterPipeline: begin")

	obj, found := pdfDict.Find("Filter")
	if !found {
		// stream is not compressed.
		return nil, nil
	}

	// compressed stream.

	var filterPipeline []types.PDFFilter

	if indRef, ok := obj.(types.PDFIndirectRef); ok {
		var err error
		obj, err = dereferencedObject(ctx, indRef.ObjectNumber.Value())
		if err != nil {
			return nil, err
		}
	}

	if name, ok := obj.(types.PDFName); ok {

		// single filter.

		filterName := name.String()

		obj, found := pdfDict.Find("DecodeParms")
		if !found {
			// w/o decode parameters.
			log.Debug.Println("pdfFilterPipeline: end w/o decode parms")
			return append(filterPipeline, types.PDFFilter{Name: filterName, DecodeParms: nil}), nil
		}

		dict, ok := obj.(types.PDFDict)
		if !ok {
			return nil, errors.New("pdfFilterPipeline: DecodeParms corrupt")
		}

		// with decode parameters.
		log.Debug.Println("pdfFilterPipeline: end with decode parms")
		return append(filterPipeline, types.PDFFilter{Name: filterName, DecodeParms: &dict}), nil
	}

	// filter pipeline.

	// Array of filternames
	filterArray, ok := obj.(types.PDFArray)
	if !ok {
		return nil, errors.Errorf("pdfFilterPipeline: Expected filterArray corrupt, %v %T", obj, obj)
	}

	// Optional array of decode parameter dicts.
	var decodeParmsArr types.PDFArray
	decodeParms, found := pdfDict.Find("DecodeParms")
	if found {
		decodeParmsArr, ok = decodeParms.(types.PDFArray)
		if !ok {
			return nil, errors.New("pdfFilterPipeline: Expected DecodeParms Array corrupt")
		}
	}

	for i, f := range filterArray {
		filterName, ok := f.(types.PDFName)
		if !ok {
			return nil, errors.New("pdfFilterPipeline: FilterArray elements corrupt")
		}
		if decodeParms == nil || decodeParmsArr[i] == nil {
			filterPipeline = append(filterPipeline, types.PDFFilter{Name: filterName.String(), DecodeParms: nil})
			continue
		}

		decodeParmsDict, ok := decodeParmsArr[i].(types.PDFDict) // can be NULL if there are no DecodeParms!
		if !ok {
			return nil, errors.New("pdfFilterPipeline: Expected DecodeParms Array corrupt")
		}
		filterPipeline = append(filterPipeline, types.PDFFilter{Name: filterName.String(), DecodeParms: &decodeParmsDict})
	}

	log.Debug.Println("pdfFilterPipeline: end")

	return filterPipeline, nil
}

func streamDict(ctx *types.PDFContext, pdfDict types.PDFDict, objNr, streamInd int, streamOffset, offset int64) (sd types.PDFStreamDict, err error) {

	streamLength, streamLengthRef := pdfDict.Length()

	if streamInd <= 0 {
		return sd, errors.New("streamDict: stream object without streamOffset")
	}

	filterPipeline, err := pdfFilterPipeline(ctx, pdfDict)
	if err != nil {
		return sd, err
	}

	streamOffset += offset

	// We have a stream object.
	sd = types.NewPDFStreamDict(pdfDict, streamOffset, streamLength, streamLengthRef, filterPipeline)

	log.Debug.Printf("streamDict: end, Streamobject #%d\n", objNr)

	return sd, nil
}

func dict(ctx *types.PDFContext, pdfDict types.PDFDict, objNr, genNr, endInd, streamInd int) (d *types.PDFDict, err error) {

	if ctx.EncKey != nil {
		_, err := crypto.DecryptDeepObject(pdfDict, objNr, genNr, ctx.EncKey, ctx.AES4Strings)
		if err != nil {
			return nil, err
		}
	}

	if endInd >= 0 && (streamInd < 0 || streamInd > endInd) {
		log.Debug.Printf("dict: end, #%d\n", objNr)
		d = &pdfDict
	}

	return d, nil
}

func object(ctx *types.PDFContext, offset int64, objNr, genNr int) (o types.PDFObject, endInd, streamInd int, streamOffset int64, err error) {

	var rd io.Reader
	rd, err = newPositionedReader(ctx.Read.File, &offset)
	if err != nil {
		return nil, 0, 0, 0, err
	}

	log.Debug.Printf("object: seeked to offset:%d\n", offset)

	// process: # gen obj ... obj dict ... {stream ... data ... endstream} endobj
	//                                    streamInd                        endInd
	//                                  -1 if absent                    -1 if absent
	var buf []byte
	buf, endInd, streamInd, streamOffset, err = buffer(rd)
	if err != nil {
		return nil, 0, 0, 0, err
	}

	//log.Debug.Printf("streamInd:%d(#%x) streamOffset:%d(#%x) endInd:%d(#%x)\n", streamInd, streamInd, streamOffset, streamOffset, endInd, endInd)
	//log.Debug.Printf("buflen=%d\n%s", len(buf), hex.Dump(buf))

	line := string(buf)

	var l string

	if endInd < 0 { // && streamInd >= 0, streamdict
		// buf: # gen obj ... obj dict ... stream ... data
		// implies we detected no endobj and a stream starting at streamInd.
		// big stream, we parse object until "stream"
		log.Debug.Println("object: big stream, we parse object until stream")
		l = line[:streamInd]
	} else if streamInd < 0 { // dict
		// buf: # gen obj ... obj dict ... endobj
		// implies we detected endobj and no stream.
		// small object w/o stream, parse until "endobj"
		log.Debug.Println("object: small object w/o stream, parse until endobj")
		l = line[:endInd]
	} else if streamInd < endInd { // streamdict
		// buf: # gen obj ... obj dict ... stream ... data ... endstream endobj
		// implies we detected endobj and stream.
		// small stream within buffer, parse until "stream"
		log.Debug.Println("object: small stream within buffer, parse until stream")
		l = line[:streamInd]
	} else { // dict
		// buf: # gen obj ... obj dict ... endobj # gen obj ... obj dict ... stream
		// small obj w/o stream, parse until "endobj"
		// stream in buf belongs to subsequent object.
		log.Debug.Println("object: small obj w/o stream, parse until endobj")
		l = line[:endInd]
	}

	// Parse object number and object generation.
	var objectNr, generationNr *int
	objectNr, generationNr, err = parseObjectAttributes(&l)
	if err != nil {
		return nil, 0, 0, 0, err
	}

	if objNr != *objectNr || genNr != *generationNr {
		return nil, 0, 0, 0, errors.Errorf("object: non matching objNr(%d) or generationNumber(%d) tags found.", *objectNr, *generationNr)
	}

	o, err = parseObject(&l)

	return o, endInd, streamInd, streamOffset, err
}

// Parses an object from file at given offset.
func pdfObject(ctx *types.PDFContext, offset int64, objNr, genNr int) (types.PDFObject, error) {

	log.Debug.Printf("pdfObject: begin, obj#%d, offset:%d\n", objNr, offset)

	pdfObject, endInd, streamInd, streamOffset, err := object(ctx, offset, objNr, genNr)
	if err != nil {
		return nil, err
	}

	switch o := pdfObject.(type) {

	case types.PDFDict:
		d, err := dict(ctx, o, objNr, genNr, endInd, streamInd)
		if err != nil || d != nil {
			return *d, err
		}
		// Parse associated stream data into a PDFStreamDict.
		return streamDict(ctx, o, objNr, streamInd, streamOffset, offset)

	case types.PDFArray:
		if ctx.EncKey != nil {
			if _, err = crypto.DecryptDeepObject(o, objNr, genNr, ctx.EncKey, ctx.AES4Strings); err != nil {
				return nil, err
			}
		}
		return o, nil

	case types.PDFStringLiteral:
		if ctx.EncKey != nil {
			s1, err := crypto.DecryptString(ctx.AES4Strings, o.Value(), objNr, genNr, ctx.EncKey)
			if err != nil {
				return nil, err
			}
			return types.PDFStringLiteral(*s1), nil
		}
		return o, nil

	case types.PDFHexLiteral:
		if ctx.EncKey != nil {
			s1, err := crypto.DecryptString(ctx.AES4Strings, o.Value(), objNr, genNr, ctx.EncKey)
			if err != nil {
				return nil, err
			}
			return types.PDFHexLiteral(*s1), nil
		}

	default:
		return o, nil
	}

	return nil, nil
}

func dereferencedObject(ctx *types.PDFContext, objectNumber int) (types.PDFObject, error) {

	entry, ok := ctx.Find(objectNumber)
	if !ok {
		return nil, errors.New("dereferencedObject: object not registered in xRefTable")
	}

	if entry.Compressed {
		decompressXRefTableEntry(ctx.XRefTable, objectNumber, entry)
	}

	if entry.Object == nil {

		// dereference this object!

		log.Debug.Printf("dereferencedObject: dereferencing object %d\n", objectNumber)

		obj, err := pdfObject(ctx, *entry.Offset, objectNumber, *entry.Generation)
		if err != nil {
			return nil, errors.Wrapf(err, "dereferencedObject: problem dereferencing object %d", objectNumber)
		}

		if obj == nil {
			return nil, errors.New("dereferencedObject: object is nil")
		}

		entry.Object = obj
	}

	return entry.Object, nil
}

// dereference a PDFInteger object representing an int64 value.
func int64Object(ctx *types.PDFContext, objectNumber int) (*int64, error) {

	log.Debug.Printf("int64Object begin: %d\n", objectNumber)

	obj, err := dereferencedObject(ctx, objectNumber)
	if err != nil {
		return nil, err
	}

	i, ok := obj.(types.PDFInteger)
	if !ok {
		return nil, errors.New("int64Object: object is not PDFInteger")
	}

	i64 := int64(i.Value())

	log.Debug.Printf("int64Object end: %d\n", objectNumber)

	return &i64, nil

}

// Reads and returns a file buffer with length = stream length using provided reader positioned at offset.
func readContentStream(rd io.Reader, streamLength int) ([]byte, error) {

	log.Debug.Printf("readContentStream: begin streamLength:%d\n", streamLength)

	buf := make([]byte, streamLength)

	for totalCount := 0; totalCount < streamLength; {
		count, err := rd.Read(buf[totalCount:])
		if err != nil {
			return nil, err
		}
		log.Debug.Printf("readContentStream: count=%d, buflen=%d(%X)\n", count, len(buf), len(buf))
		totalCount += count
	}

	log.Debug.Printf("readContentStream: end\n")

	return buf, nil
}

// LoadEncodedStreamContent loads the encoded stream content from file into PDFStreamDict.
func LoadEncodedStreamContent(ctx *types.PDFContext, streamDict *types.PDFStreamDict) ([]byte, error) {

	log.Debug.Printf("LoadEncodedStreamContent: begin\n%v\n", streamDict)

	var err error

	// Return saved decoded content.
	if streamDict.Raw != nil {
		log.Debug.Println("LoadEncodedStreamContent: end, already in memory.")
		return streamDict.Raw, nil
	}

	// Read stream content encoded at offset with stream length.

	// Dereference stream length if stream length is an indirect object.
	if streamDict.StreamLength == nil {
		if streamDict.StreamLengthObjNr == nil {
			return nil, errors.New("LoadEncodedStreamContent: missing streamLength")
		}
		// Get stream length from indirect object
		streamDict.StreamLength, err = int64Object(ctx, *streamDict.StreamLengthObjNr)
		if err != nil {
			return nil, err
		}
		log.Debug.Printf("LoadEncodedStreamContent: new indirect streamLength:%d\n", *streamDict.StreamLength)
	}

	newOffset := streamDict.StreamOffset
	rd, err := newPositionedReader(ctx.Read.File, &newOffset)
	if err != nil {
		return nil, err
	}

	log.Debug.Printf("LoadEncodedStreamContent: seeked to offset:%d\n", newOffset)

	// Buffer stream contents.
	// Read content from disk.
	rawContent, err := readContentStream(rd, int(*streamDict.StreamLength))
	if err != nil {
		return nil, err
	}

	//log.Debug.Printf("rawContent buflen=%d(#%x)\n%s", len(rawContent), len(rawContent), hex.Dump(rawContent))

	// Save encoded content.
	streamDict.Raw = rawContent

	log.Debug.Printf("LoadEncodedStreamContent: end: len(streamDictRaw)=%d\n", len(streamDict.Raw))

	// Return encoded content.
	return rawContent, nil
}

// Decodes the raw encoded stream content and saves it to streamDict.Content.
func setDecodedStreamContent(ctx *types.PDFContext, streamDict *types.PDFStreamDict, objNr, genNr int, decode bool) (err error) {

	log.Debug.Printf("setDecodedStreamContent: begin decode=%t\n", decode)

	//  If the "Identity" crypt filter is used we do not need to decrypt.
	if ctx != nil && ctx.EncKey != nil {
		if len(streamDict.FilterPipeline) == 1 && streamDict.FilterPipeline[0].Name == "Crypt" {
			streamDict.Content = streamDict.Raw
			return nil
		}
	}

	// ctx gets created after XRefStream parsing.
	// XRefStreams are not encrypted.
	if ctx != nil && ctx.EncKey != nil {
		streamDict.Raw, err = crypto.DecryptStream(ctx.AES4Streams, streamDict.Raw, objNr, genNr, ctx.EncKey)
		if err != nil {
			return err
		}
		l := int64(len(streamDict.Raw))
		streamDict.StreamLength = &l
	}

	if !decode {
		return nil
	}

	// Actual decoding of content stream.
	err = filter.DecodeStream(streamDict)
	if err != nil {
		return err
	}

	log.Debug.Println("setDecodedStreamContent: end")

	return nil
}

// Resolve compressed xRefTableEntry
func decompressXRefTableEntry(xRefTable *types.XRefTable, objectNumber int, entry *types.XRefTableEntry) error {

	log.Debug.Printf("decompressXRefTableEntry: compressed object %d at %d[%d]\n", objectNumber, *entry.ObjectStream, *entry.ObjectStreamInd)

	// Resolve xRefTable entry of referenced object stream.
	objectStreamXRefTableEntry, ok := xRefTable.Find(*entry.ObjectStream)
	if !ok {
		return errors.Errorf("decompressXRefTableEntry: problem dereferencing object stream %d, no xref table entry", *entry.ObjectStream)
	}

	// Object of this entry has to be a PDFObjectStreamDict.
	pdfObjectStreamDict, ok := objectStreamXRefTableEntry.Object.(types.PDFObjectStreamDict)
	if !ok {
		return errors.Errorf("decompressXRefTableEntry: problem dereferencing object stream %d, no object stream", *entry.ObjectStream)
	}

	// Get indexed object from PDFObjectStreamDict.
	pdfObject, err := pdfObjectStreamDict.IndexedObject(*entry.ObjectStreamInd)
	if err != nil {
		return errors.Wrapf(err, "decompressXRefTableEntry: problem dereferencing object stream %d", *entry.ObjectStream)
	}

	// Save object to XRefRableEntry.
	g := 0
	entry.Object = pdfObject
	entry.Generation = &g
	entry.Compressed = false

	log.Debug.Printf("decompressXRefTableEntry: end, Obj %d[%d]:\n<%s>\n", *entry.ObjectStream, *entry.ObjectStreamInd, pdfObject)

	return nil
}

// Log interesting stream content.
func logStream(obj types.PDFObject) {

	switch obj := obj.(type) {

	case types.PDFStreamDict:

		if obj.Content == nil {
			log.Debug.Println("logStream: no stream content")
		}

		if obj.IsPageContent {
			//log.Debug.Printf("content <%s>\n", pdfStreamDict.Content)
		}

	case types.PDFObjectStreamDict:

		if obj.Content == nil {
			log.Debug.Println("logStream: no object stream content")
		} else {
			log.Debug.Printf("logStream: objectStream content = %s\n", obj.Content)
		}

		if obj.ObjArray == nil {
			log.Debug.Println("logStream: no object stream obj arr")
		} else {
			log.Debug.Printf("logStream: objectStream objArr = %s\n", obj.ObjArray)
		}

	default:
		log.Debug.Println("logStream: no pdfObjectStreamDict")

	}

}

// Decode all object streams so contained objects are ready to be used.
func decodeObjectStreams(ctx *types.PDFContext) error {

	// Note:
	// Entry "Extends" intentionally left out.
	// No object stream collection validation necessary.

	log.Debug.Println("decodeObjectStreams: begin")

	// Get sorted slice of object numbers.
	var keys []int
	for k := range ctx.Read.ObjectStreams {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, objectNumber := range keys {

		// Get XRefTableEntry.
		entry := ctx.XRefTable.Table[objectNumber]
		if entry == nil {
			return errors.Errorf("decodeObjectStream: missing entry for obj#%d\n", objectNumber)
		}

		log.Debug.Printf("decodeObjectStreams: parsing object stream for obj#%d\n", objectNumber)

		// Parse object stream from file.
		obj, err := pdfObject(ctx, *entry.Offset, objectNumber, *entry.Generation)
		if err != nil || obj == nil {
			return errors.New("decodeObjectStreams: corrupt object stream")
		}

		// Ensure PDFStreamDict
		pdfStreamDict, ok := obj.(types.PDFStreamDict)
		if !ok {
			return errors.New("decodeObjectStreams: corrupt object stream")
		}

		// Load encoded stream content to xRefTable.
		if _, err = LoadEncodedStreamContent(ctx, &pdfStreamDict); err != nil {
			return errors.Wrapf(err, "decodeObjectStreams: problem dereferencing object stream %d", objectNumber)
		}

		// Save decoded stream content to xRefTable.
		if err = setDecodedStreamContent(ctx, &pdfStreamDict, objectNumber, *entry.Generation, true); err != nil {
			log.Debug.Printf("obj %d: %s", objectNumber, err)
			return err
		}

		// Ensure decoded objectArray for object stream dicts.
		if !pdfStreamDict.IsObjStm() {
			return errors.New("decodeObjectStreams: corrupt object stream")
		}

		// We have an object stream.
		log.Debug.Printf("decodeObjectStreams: object stream #%d\n", objectNumber)

		ctx.Read.UsingObjectStreams = true

		// Create new object stream dict.
		pdfObjectStreamDict, err := objectStreamDict(pdfStreamDict)
		if err != nil {
			return errors.Wrapf(err, "decodeObjectStreams: problem dereferencing object stream %d", objectNumber)
		}

		log.Debug.Printf("decodeObjectStreams: decoding object stream %d:\n", objectNumber)

		// Parse all objects of this object stream and save them to pdfObjectStreamDict.ObjArray.
		if err = parseObjectStream(pdfObjectStreamDict); err != nil {
			return errors.Wrapf(err, "decodeObjectStreams: problem decoding object stream %d\n", objectNumber)
		}

		if pdfObjectStreamDict.ObjArray == nil {
			return errors.Wrap(err, "decodeObjectStreams: objArray should be set!")
		}

		log.Debug.Printf("decodeObjectStreams: decoded object stream %d:\n", objectNumber)

		// Save object stream dict to xRefTableEntry.
		entry.Object = *pdfObjectStreamDict
	}

	log.Debug.Println("decodeObjectStreams: end")

	return nil
}

func handleLinearizationParmDict(ctx *types.PDFContext, obj types.PDFObject, objNr int) error {

	if ctx.Read.Linearized {
		// Linearization dict already processed.
		return nil
	}

	// handle linearization parm dict.
	if pdfDict, ok := obj.(types.PDFDict); ok && pdfDict.IsLinearizationParmDict() {

		ctx.Read.Linearized = true
		ctx.LinearizationObjs[objNr] = true
		log.Debug.Printf("handleLinearizationParmDict: identified linearizationObj #%d\n", objNr)

		arr := pdfDict.PDFArrayEntry("H")

		if arr == nil {
			return errors.Errorf("handleLinearizationParmDict: corrupt linearization dict at obj:%d - missing array entry H", objNr)
		}

		if len(*arr) != 2 && len(*arr) != 4 {
			return errors.Errorf("handleLinearizationParmDict: corrupt linearization dict at obj:%d - corrupt array entry H, needs length 2 or 4", objNr)
		}

		offset, ok := (*arr)[0].(types.PDFInteger)
		if !ok {
			return errors.Errorf("handleLinearizationParmDict: corrupt linearization dict at obj:%d - corrupt array entry H, needs Integer values", objNr)
		}

		offset64 := int64(offset.Value())
		ctx.OffsetPrimaryHintTable = &offset64

		if len(*arr) == 4 {

			offset, ok := (*arr)[2].(types.PDFInteger)
			if !ok {
				return errors.Errorf("handleLinearizationParmDict: corrupt linearization dict at obj:%d - corrupt array entry H, needs Integer values", objNr)
			}

			offset64 := int64(offset.Value())
			ctx.OffsetOverflowHintTable = &offset64
		}
	}

	return nil
}

func loadPDFStreamDict(ctx *types.PDFContext, sd *types.PDFStreamDict, objNr, genNr int) error {

	// Load encoded stream content for stream dicts into xRefTable entry.
	if _, err := LoadEncodedStreamContent(ctx, sd); err != nil {
		return errors.Wrapf(err, "dereferenceObject: problem dereferencing stream %d", objNr)
	}

	ctx.Read.BinaryTotalSize += *sd.StreamLength

	// Decode stream content.
	return setDecodedStreamContent(ctx, sd, objNr, genNr, ctx.DecodeAllStreams)
}

func updateBinaryTotalSize(ctx *types.PDFContext, o types.PDFObject) {

	switch obj := o.(type) {

	case types.PDFStreamDict:
		ctx.Read.BinaryTotalSize += *obj.StreamLength

	case types.PDFObjectStreamDict:
		ctx.Read.BinaryTotalSize += *obj.StreamLength

	case types.PDFXRefStreamDict:
		ctx.Read.BinaryTotalSize += *obj.StreamLength

	}

}

func dereferenceObject(ctx *types.PDFContext, objNr int) error {

	xRefTable := ctx.XRefTable

	log.Debug.Println("dereferenceObject: begin")

	xRefTableSize := len(xRefTable.Table)

	log.Debug.Printf("dereferenceObject: dereferencing object %d\n", objNr)

	entry := xRefTable.Table[objNr]

	if entry.Free {
		//log.Debug.Printf("free object %d\n", objectNumber)
		return nil
	}

	if entry.Compressed {
		err := decompressXRefTableEntry(xRefTable, objNr, entry)
		if err != nil {
			return err
		}
		log.Debug.Printf("dereferenceObject: decompressed entry, Compressed=%v\n%s\n", entry.Compressed, entry.Object)
		return nil
	}

	// entry is in use.

	if entry.Offset == nil || *entry.Offset == 0 {
		log.Debug.Printf("dereferenceObject: already decompressed or used object w/o offset -> ignored")
		return nil
	}

	obj := entry.Object

	// Already dereferenced stream dict.
	if obj != nil {
		logStream(entry.Object)
		updateBinaryTotalSize(ctx, obj)
		log.Debug.Printf("handleCachedStreamDict: using cached object %d of %d\n<%s>\n", objNr, xRefTableSize, entry.Object)
		return nil
	}

	// Dereference (load from disk into memory).

	log.Debug.Printf("dereferenceObject: dereferencing object %d\n", objNr)

	// Parse object from file: anything goes dict,array,integer,float,streamdicts..
	obj, err := pdfObject(ctx, *entry.Offset, objNr, *entry.Generation)
	if err != nil {
		return errors.Wrapf(err, "dereferenceObject: problem dereferencing object %d", objNr)
	}

	entry.Object = obj

	// Linearization dicts are validated and recorded for stats only.
	err = handleLinearizationParmDict(ctx, obj, objNr)
	if err != nil {
		return err
	}

	// Handle stream dicts.

	if _, ok := obj.(types.PDFObjectStreamDict); ok {
		return errors.Errorf("dereferenceObject: object stream should already be dereferenced at obj:%d", objNr)
	}

	if _, ok := obj.(types.PDFXRefStreamDict); ok {
		return errors.Errorf("dereferenceObject: xref stream should already be dereferenced at obj:%d", objNr)
	}

	if pdfStreamDict, ok := obj.(types.PDFStreamDict); ok {

		err = loadPDFStreamDict(ctx, &pdfStreamDict, objNr, *entry.Generation)
		if err != nil {
			return err
		}

		entry.Object = pdfStreamDict
	}

	log.Debug.Printf("dereferenceObject: end obj %d of %d\n<%s>\n", objNr, xRefTableSize, entry.Object)

	logStream(entry.Object)

	return nil
}

// Dereferences all objects including compressed objects from object streams.
func dereferenceObjects(ctx *types.PDFContext) error {

	log.Debug.Println("dereferenceObjects: begin")

	// Get sorted slice of object numbers.
	var keys []int
	for k := range ctx.XRefTable.Table {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, objNr := range keys {
		err := dereferenceObject(ctx, objNr)
		if err != nil {
			return err
		}
	}

	log.Debug.Println("dereferenceObjects: end")

	return nil
}

// Locate a possible Version entry (since V1.4) in the catalog
// and record this as rootVersion (as opposed to headerVersion).
func identifyRootVersion(xRefTable *types.XRefTable) error {

	log.Debug.Println("identifyRootVersion: begin")

	// Try to get Version from Root.
	rootVersionStr, err := xRefTable.ParseRootVersion()
	if err != nil {
		return err
	}

	if rootVersionStr == nil {
		return nil
	}

	// Validate version and save corresponding constant to xRefTable.
	rootVersion, err := types.Version(*rootVersionStr)
	if err != nil {
		return errors.Wrapf(err, "identifyRootVersion: unknown PDF Root version: %s\n", *rootVersionStr)
	}

	xRefTable.RootVersion = &rootVersion

	// since V1.4 the header version may be overridden by a Version entry in the catalog.
	if *xRefTable.HeaderVersion < types.V14 {
		log.Info.Printf("identifyRootVersion: PDF version is %s - will ignore root version: %s\n",
			types.VersionString(*xRefTable.HeaderVersion), *rootVersionStr)
	}

	log.Debug.Println("identifyRootVersion: end")

	return nil
}

// Parse all PDFObjects including stream content from file and save to the corresponding xRefTableEntries.
// This includes processing of object streams and linearization dicts.
func dereferenceXRefTable(ctx *types.PDFContext, config *types.Configuration) error {

	log.Debug.Println("dereferenceXRefTable: begin")

	xRefTable := ctx.XRefTable

	// Note for encrypted files:
	// Mandatory supply userpw to open & display file.
	// Access may be restricted (Decode access privileges).
	// Optionally supply ownerpw in order to gain unrestricted access.
	err := checkForEncryption(ctx)
	if err != nil {
		return err
	}
	//logErrorReader.Println("pw authenticated")

	// Prepare decompressed objects.
	err = decodeObjectStreams(ctx)
	if err != nil {
		return err
	}

	// For each xRefTableEntry assign a PDFObject either by parsing from file or pointing to a decompressed object.
	err = dereferenceObjects(ctx)
	if err != nil {
		return err
	}

	// Identify an optional Version entry in the root object/catalog.
	err = identifyRootVersion(xRefTable)
	if err != nil {
		return err
	}

	log.Debug.Println("dereferenceXRefTable: end")

	return nil
}

func handleUnencryptedFile(ctx *types.PDFContext) error {

	if ctx.Mode == types.DECRYPT || ctx.Mode == types.ADDPERMISSIONS {
		return errors.New("decrypt: this file is not encrypted")
	}

	if ctx.Mode != types.ENCRYPT {
		return nil
	}

	// Encrypt subcommand found.

	if len(ctx.UserPW) == 0 && len(ctx.OwnerPW) == 0 {
		return errors.New("encrypt: user or/and owner password missing")
	}

	// Ensure ctx.ID
	if ctx.ID == nil {
		ctx.ID = crypto.ID(ctx)
	}

	// Must be array of length 2
	arr := *ctx.ID

	if len(arr) != 2 {
		return errors.New("encrypt: ID must be array with 2 elements")
	}

	return nil
}

func dereferenceEncryptDict(ctx *types.PDFContext, encryptDictObjNr int) (*types.PDFDict, error) {

	obj, err := dereferencedObject(ctx, encryptDictObjNr)
	if err != nil {
		return nil, err
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		return nil, errors.New("corrupt encrypt dict")
	}

	return &dict, nil
}

func idBytes(ctx *types.PDFContext) (id []byte, err error) {

	if ctx.ID == nil {
		return nil, errors.New("missing ID entry")
	}

	hl, ok := ((*ctx.ID)[0]).(types.PDFHexLiteral)
	if ok {
		id, err = hl.Bytes()
		if err != nil {
			return nil, err
		}
	} else {
		sl, ok := ((*ctx.ID)[0]).(types.PDFStringLiteral)
		if !ok {
			return nil, errors.New("encryption: ID must contain PDFHexLiterals or PDFStringLiterals")
		}
		id, err = types.Unescape(sl.Value())
		if err != nil {
			return nil, err
		}
	}

	return id, nil
}

func needsOwnerAndUserPassword(cmd types.CommandMode) bool {

	return cmd == types.CHANGEOPW || cmd == types.CHANGEUPW || cmd == types.ADDPERMISSIONS
}

func setupEncryptionKey(ctx *types.PDFContext, encryptDictObjNr int) error {

	// Dereference encryptDict.
	encryptDict, err := dereferenceEncryptDict(ctx, encryptDictObjNr)
	if err != nil {
		return err
	}

	log.Debug.Printf("%s\n", encryptDict)

	enc, err := crypto.SupportedEncryption(ctx, encryptDict)
	if err != nil {
		return err
	}
	if enc == nil {
		return errors.New("This encryption is not supported")
	}

	ctx.E = enc
	//fmt.Printf("read: O = %0X\n", enc.O)
	//fmt.Printf("read: U = %0X\n", enc.U)

	enc.ID, err = idBytes(ctx)
	if err != nil {
		return err
	}

	var ok bool

	//fmt.Println("checking opw")
	ok, ctx.EncKey, err = crypto.ValidateOwnerPassword(ctx)
	if err != nil {
		return err
	}

	// If the owner password does not match we generally move on if the user password is correct
	// unless we need to insist on a correct owner password.
	if !ok && needsOwnerAndUserPassword(ctx.Mode) {
		return errors.New("owner password authentication error")
	}

	// Generally the owner password, which is also regarded as the master password or set permissions password
	// is sufficient for moving on. A password change is an exception since it requires both passwords authenticated.
	if ok && !needsOwnerAndUserPassword(ctx.Mode) {
		return nil
	}

	//fmt.Println("checking upw")
	ok, ctx.EncKey, err = crypto.ValidateUserPassword(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("user password authentication error")
	}

	if !crypto.HasNeededPermissions(ctx.Mode, ctx.E) {
		return errors.New("Insufficient access permissions")
	}

	return nil
}

func checkForEncryption(ctx *types.PDFContext) error {

	indRef := ctx.Encrypt

	if indRef == nil {
		// This file is not encrypted.
		return handleUnencryptedFile(ctx)
	}

	// This file is encrypted.
	log.Info.Printf("Encryption: %v\n", indRef)

	if ctx.Mode == types.ENCRYPT {
		// We want to encrypt this file.
		return errors.New("encrypt: This file is already encrypted")
	}

	// We need to decrypt this file in order to read it.
	return setupEncryptionKey(ctx, indRef.ObjectNumber.Value())
}
