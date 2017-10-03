// Package read provides methods for parsing PDF files into memory.
//
// The in memory representation of a PDF file is called a PDFContext.
//
// The PDFContext is a container for the PDF cross reference table and stats.
package read

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/hhrutter/pdfcpu/filter"
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

const (
	defaultBufSize   = 1024
	unknownDelimiter = byte(0)
)

var logDebugReader, logInfoReader, logWarningReader, logErrorReader *log.Logger

func init() {
	logDebugReader = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	logInfoReader = log.New(ioutil.Discard, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	logWarningReader = log.New(os.Stdout, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	logErrorReader = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// Verbose controls logging output.
func Verbose(verbose bool) {
	out := ioutil.Discard
	if verbose {
		out = os.Stdout
	}
	logInfoReader = log.New(out, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	logDebugReader = log.New(out, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// ScanLines is a split function for a Scanner that returns each line of
// text, stripped of any trailing end-of-line marker. The returned line may
// be empty. The end-of-line marker is one carriage return followed
// by one newline or one carriage return or one newline.
// The last non-empty line of input will be returned even if it has no newline.
func ScanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {

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

	logDebugReader.Printf("newPositionedReader: positioned to offset: %d\n", *offset)

	return bufio.NewReader(rs), nil
}

// Get the file offset of the last XRefSection.
// Go to end of file and search backwards for the first occurrence of startxref {offset} %%EOF
func getOffsetLastXRefSection(ra io.ReaderAt, fileSize int64) (*int64, error) {

	var bufSize int64 = defaultBufSize

	off := fileSize - defaultBufSize
	if off < 0 {
		off = 0
		bufSize = fileSize
	}
	buf := make([]byte, bufSize)

	logDebugReader.Printf("getOffsetLastXRefSection at %d\n", off)

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

	logDebugReader.Printf("Offset last xrefsection: %d\n", offset)

	return &offset, nil
}

// Read next subsection entry and generate corresponding xref table entry.
func parseXRefTableEntry(s *bufio.Scanner, xRefTable *types.XRefTable, objectNumber int) error {

	logDebugReader.Println("parseXRefTableEntry: begin")

	line, err := scanLine(s)
	if err != nil {
		return err
	}

	if xRefTable.Exists(objectNumber) {
		logDebugReader.Printf("parseXRefTableEntry: end - Skip entry %d - already assigned\n", objectNumber)
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

		logDebugReader.Printf("parseXRefTableEntry: Object #%d is in use at offset=%d, generation=%d\n", objectNumber, offset, generation)

		if offset == 0 {
			logInfoReader.Printf("parseXRefTableEntry: Skip entry for in use object #%d with offset 0\n", objectNumber)
			return nil
		}

		xRefTableEntry =
			types.XRefTableEntry{
				Free:       false,
				Offset:     &offset,
				Generation: &generation}

	} else {

		// free object

		logDebugReader.Printf("parseXRefTableEntry: Object #%d is unused, next free is object#%d, generation=%d\n", objectNumber, offset, generation)

		xRefTableEntry =
			types.XRefTableEntry{
				Free:       true,
				Offset:     &offset,
				Generation: &generation}

	}

	logDebugReader.Printf("parseXRefTableEntry: Insert new xreftable entry for Object %d\n", objectNumber)

	if !xRefTable.Insert(objectNumber, xRefTableEntry) {
		return errors.Errorf("parseXRefTableEntry: Problem inserting entry for %d", objectNumber)
	}

	logDebugReader.Println("parseXRefTableEntry: end")

	return nil
}

// Process xRef table subsection and create corrresponding xRef table entries.
func parseXRefTableSubSection(s *bufio.Scanner, xRefTable *types.XRefTable, fields []string) error {

	logDebugReader.Println("parseXRefTableSubSection: begin")

	startObjNumber, err := strconv.Atoi(fields[0])
	if err != nil {
		return err
	}

	objCount, err := strconv.Atoi(fields[1])
	if err != nil {
		return err
	}

	logDebugReader.Printf("detected xref subsection, startObj=%d length=%d\n", startObjNumber, objCount)

	// Process all entries of this subsection into xRefTable entries.
	for i := 0; i < objCount; i++ {
		if err = parseXRefTableEntry(s, xRefTable, startObjNumber+i); err != nil {
			return err
		}
	}

	logDebugReader.Println("parseXRefTableSubSection: end")

	return nil
}

// Parse compressed object.
func getCompressedObject(s string) (interface{}, error) {

	logDebugReader.Println("getCompressedObject: begin")

	pdfObject, err := parseObject(&s)
	if err != nil {
		return nil, err
	}

	pdfDict, ok := pdfObject.(types.PDFDict)
	if !ok {
		// return trivial PDFObject: PDFInteger, PDFArray, etc.
		logDebugReader.Println("getCompressedObject: end, any other than dict")
		return pdfObject, nil
	}

	streamLength, streamLengthRef := pdfDict.Length()
	if streamLength == nil && streamLengthRef == nil {
		// return PDFDict
		logDebugReader.Println("getCompressedObject: end, dict")
		return pdfDict, nil
	}

	return nil, errors.New("getCompressedObject: Stream objects are not to be stored in an object stream")
}

// Parse all objects of an object stream and save them into objectStreamDict.ObjArray.
func parseObjectStream(objectStreamDict *types.PDFObjectStreamDict) error {

	logDebugReader.Printf("parseObjectStream begin: decoding %d objects.\n", objectStreamDict.ObjCount)

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
			logDebugReader.Printf("parseObjectStream: objString = %s\n", dstr)
			pdfObject, err := getCompressedObject(dstr)
			if err != nil {
				return err
			}

			logDebugReader.Printf("parseObjectStream: [%d] = obj %s:\n%s\n", i/2-1, objs[i-2], pdfObject)
			objArray = append(objArray, pdfObject)
		}

		if i == len(objs)-2 {
			dstr := string(decodedContent[offset:])
			logDebugReader.Printf("parseObjectStream: objString = %s\n", dstr)
			pdfObject, err := getCompressedObject(dstr)
			if err != nil {
				return err
			}

			logDebugReader.Printf("parseObjectStream: [%d] = obj %s:\n%s\n", i/2, objs[i], pdfObject)
			objArray = append(objArray, pdfObject)
		}

		offsetOld = offset
	}

	objectStreamDict.ObjArray = objArray

	logDebugReader.Println("parseObjectStream end")

	return nil
}

// For each object embedded in this xRefStream create the corresponding xRef table entry.
func extractXRefTableEntriesFromXRefStream(buf []byte, xRefStreamDict types.PDFXRefStreamDict, ctx *types.PDFContext) error {

	logDebugReader.Printf("extractXRefTableEntriesFromXRefStream begin")

	// Note:
	// A value of zero for an element in the W array indicates that the corresponding field shall not be present in the stream,
	// and the default value shall be used, if there is one.
	// If the first element is zero, the type field shall not be present, and shall default to type 1.

	i1 := xRefStreamDict.W[0]
	i2 := xRefStreamDict.W[1]
	i3 := xRefStreamDict.W[2]

	xrefEntryLen := i1 + i2 + i3
	logDebugReader.Printf("extractXRefTableEntriesFromXRefStream: begin xrefEntryLen = %d\n", xrefEntryLen)

	if len(buf)%xrefEntryLen > 0 {
		return errors.New("extractXRefTableEntriesFromXRefStream: corrupt xrefstream")
	}

	objCount := len(xRefStreamDict.Objects)
	logDebugReader.Printf("extractXRefTableEntriesFromXRefStream: objCount:%d %v\n", objCount, xRefStreamDict.Objects)

	logDebugReader.Printf("extractXRefTableEntriesFromXRefStream: len(buf):%d objCount*xrefEntryLen:%d\n", len(buf), objCount*xrefEntryLen)
	if len(buf) != objCount*xrefEntryLen {
		return errors.New("extractXRefTableEntriesFromXRefStream: corrupt xrefstream")
	}

	j := 0

	// getInt interprets the content of buf as an int64.
	getInt := func(buf []byte) (i int64) {

		for _, b := range buf {
			i <<= 8
			i |= int64(b)
		}

		return
	}

	for i := 0; i < len(buf) && j < len(xRefStreamDict.Objects); i += xrefEntryLen {

		objectNumber := xRefStreamDict.Objects[j]

		i2Start := i + i1
		c2 := getInt(buf[i2Start : i2Start+i2])
		c3 := getInt(buf[i2Start+i2 : i2Start+i2+i3])

		var xRefTableEntry types.XRefTableEntry

		switch buf[i] {

		case 0x00:
			// free object
			logDebugReader.Printf("extractXRefTableEntriesFromXRefStream: Object #%d is unused, next free is object#%d, generation=%d\n", objectNumber, c2, c3)
			g := int(c3)

			xRefTableEntry =
				types.XRefTableEntry{
					Free:       true,
					Compressed: false,
					Offset:     &c2,
					Generation: &g}

		case 0x01:
			// in use object
			logDebugReader.Printf("extractXRefTableEntriesFromXRefStream: Object #%d is in use at offset=%d, generation=%d\n", objectNumber, c2, c3)
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
			logDebugReader.Printf("extractXRefTableEntriesFromXRefStream: Object #%d is compressed at obj %5d[%d]\n", objectNumber, c2, c3)
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
			logDebugReader.Printf("extractXRefTableEntriesFromXRefStream: Skip entry %d - already assigned\n", objectNumber)
		} else {

			//logDebugReader.Printf("Insert new xreftable entry for Object %d\n", objectNumber)
			if !ctx.XRefTable.Insert(objectNumber, xRefTableEntry) {
				return errors.Errorf("extractXRefTableEntriesFromXRefStream: Problem inserting entry for %d", objectNumber)
			}

		}

		j++
	}

	logDebugReader.Println("extractXRefTableEntriesFromXRefStream: end")

	return nil
}

// Parse xRef stream and setup xrefTable entries for all embedded objects and the xref stream dict.
func parseXRefStream(rd io.Reader, offset *int64, ctx *types.PDFContext) (prevOffset *int64, err error) {

	logDebugReader.Printf("parseXRefStream: begin at offset %d\n", *offset)

	buf, endInd, streamInd, streamOffset, err := getBuffer(rd)
	if err != nil {
		return nil, err
	}

	logDebugReader.Printf("parseXRefStream: endInd=%[1]d(%[1]x) streamInd=%[2]d(%[2]x)\n", endInd, streamInd)

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
	logDebugReader.Printf("parseXRefStream: xrefstm obj#:%d gen:%d\n", *objectNumber, *generationNumber)
	logDebugReader.Printf("parseXRefStream: dereferencing object %d\n", *objectNumber)
	pdfObject, err := parseObject(&l)
	if err != nil {
		return nil, errors.Wrapf(err, "parseXRefStream: no pdfObject")
	}

	logDebugReader.Printf("parseXRefStream: we have a pdfObject: %s\n", pdfObject)

	// must be pdfDict
	pdfDict, ok := pdfObject.(types.PDFDict)
	if !ok {
		return nil, errors.New("parseXRefStream: no pdfDict")
	}

	// Parse attributes for stream object.
	streamLength, streamLengthObjNr := pdfDict.Length()
	if streamLength == nil && streamLengthObjNr == nil {
		return nil, errors.New("parseXRefStream: no \"Length\" entry")
	}

	filterPipeline, err := getPDFFilterPipeline(ctx, pdfDict)
	if err != nil {
		return nil, err
	}

	streamOffset += *offset

	// We have a stream object.
	logDebugReader.Printf("parseXRefStream: streamobject #%d\n", *objectNumber)
	pdfStreamDict := types.NewPDFStreamDict(pdfDict, streamOffset, streamLength, streamLengthObjNr, filterPipeline)

	if _, err = GetEncodedStreamContent(ctx, &pdfStreamDict); err != nil {
		return nil, err
	}

	// Decode xrefstream content
	if err = setDecodedStreamContent(nil, &pdfStreamDict, 0, 0, true); err != nil {
		return nil, errors.Wrapf(err, "parseXRefStream: cannot decode stream for obj#:%d\n", *objectNumber)
	}

	pdfXRefStreamDict, err := parseXRefStreamDict(pdfStreamDict)
	if err != nil {
		return nil, err
	}
	// We have an xref stream object

	err = parseTrailerInfo(pdfXRefStreamDict.PDFDict, ctx.XRefTable)
	if err != nil {
		return nil, err
	}

	// Parse xRefStream and create xRefTable entries for embedded objects.
	err = extractXRefTableEntriesFromXRefStream(pdfStreamDict.Content, *pdfXRefStreamDict, ctx)
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

	logDebugReader.Printf("parseXRefStream: Insert new xRefTable entry for Object %d\n", *objectNumber)

	if !ctx.XRefTable.Insert(*objectNumber, entry) {
		logDebugReader.Printf("parseXRefStream: Problem inserting entry for %d\n", *objectNumber)
		return nil, errors.New("parseXRefStreams: can't insert entry into xRefTable")
	}

	ctx.Read.XRefStreams[*objectNumber] = true

	prevOffset = pdfXRefStreamDict.PreviousOffset

	logDebugReader.Println("parseXRefStream: end")

	return
}

// Parse an xRefStream for a hybrid PDF file.
func parseHybridXRefStream(offset *int64, ctx *types.PDFContext) error {

	logDebugReader.Println("parseHybridXRefStream: begin")

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

	logDebugReader.Println("parseHybridXRefStream: end")

	return nil
}

// Parse trailer dict and return any offset of a previous xref section.
func parseTrailerInfo(dict types.PDFDict, xRefTable *types.XRefTable) error {

	logDebugReader.Println("parseTrailerInfo begin")

	// Encrypted files are not supported.
	if _, found := dict.Find("Encrypt"); found {
		encryptObjRef := dict.IndirectRefEntry("Encrypt")
		if encryptObjRef != nil {
			xRefTable.Encrypt = encryptObjRef
			logDebugReader.Printf("parseTrailerInfo: Encrypt object: %s\n", *xRefTable.Encrypt)
		}
		//return errors.New("parseTrailerInfo: unsupported entry \"Encrypt\"")
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
		logDebugReader.Printf("parseTrailerInfo: Root object: %s\n", *xRefTable.Root)
	}

	if xRefTable.Info == nil {
		infoObjRef := dict.IndirectRefEntry("Info")
		if infoObjRef != nil {
			xRefTable.Info = infoObjRef
			logDebugReader.Printf("parseTrailerInfo: Info object: %s\n", *xRefTable.Info)
		}
	}

	if xRefTable.ID == nil {
		idArray := dict.PDFArrayEntry("ID")
		if idArray != nil {
			xRefTable.ID = idArray
			logDebugReader.Printf("parseTrailerInfo: ID object: %s\n", *xRefTable.ID)
		}
	}

	logDebugReader.Println("parseTrailerInfo end")

	return nil
}

func parseTrailerDict(trailerDict types.PDFDict, ctx *types.PDFContext) (*int64, error) {

	logDebugReader.Println("parseTrailerDict begin")

	xRefTable := ctx.XRefTable

	err := parseTrailerInfo(trailerDict, xRefTable)
	if err != nil {
		return nil, err
	}

	if arr := trailerDict.PDFArrayEntry("AdditionalStreams"); arr != nil {
		logDebugReader.Printf("parseTrailerInfo: found AdditionalStreams: %s\n", arr)
		for _, value := range *arr {
			if indRef, ok := value.(types.PDFIndirectRef); ok {
				xRefTable.AdditionalStreams = append(xRefTable.AdditionalStreams, indRef)
			}
		}
	}

	offset := trailerDict.Prev()
	if offset != nil {
		logDebugReader.Printf("parseTrailerDict: previous xref table section offset:%d\n", *offset)
	}

	offsetXRefStream := trailerDict.Int64Entry("XRefStm")
	if offsetXRefStream == nil {
		// No cross reference stream.
		if !ctx.Reader15 && xRefTable.Version() >= types.V14 && !ctx.Read.Hybrid {
			return nil, errors.Errorf("parseTrailerDict: PDF1.4 conformant reader: found incompatible version: %s", xRefTable.VersionString())
		}
		logDebugReader.Println("parseTrailerDict end")
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

	logDebugReader.Println("parseTrailerDict end")

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

	logDebugReader.Println("parseXRefSection begin")

	line, err := scanLine(s)
	if err != nil {
		return nil, err
	}

	logDebugReader.Printf("parseXRefSection: <%s>\n", line)

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

	logDebugReader.Println("parseXRefSection: All subsections read!")

	if !strings.HasPrefix(line, "trailer") {
		return nil, errors.Errorf("xrefsection: missing trailer dict, line = <%s>", line)
	}

	logDebugReader.Println("parseXRefSection: parsing trailer dict..")

	var trailerString string

	if line != "trailer" {
		trailerString = line[7:]
		logDebugReader.Printf("parseXRefSection: trailer leftover: <%s>\n", trailerString)
	} else {
		logDebugReader.Printf("line (len %d) <%s>\n", len(line), line)
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

	logDebugReader.Printf("parseXRefSection: trailerString: (len:%d) <%s>\n", len(trailerString), trailerString)

	pdfObject, err := parseObject(&trailerString)
	if err != nil {
		return nil, err
	}

	trailerDict, ok := pdfObject.(types.PDFDict)
	if !ok {
		return nil, errors.New("parseXRefSection: corrupt trailer dict")
	}

	logDebugReader.Printf("parseXRefSection: trailerDict:\n%s\n", trailerDict)

	offset, err := parseTrailerDict(trailerDict, ctx)
	if err != nil {
		return nil, err
	}

	logDebugReader.Println("parseXRefSection end")

	return offset, nil
}

// Get version from first line of file.
// Beginning with PDF 1.4, the Version entry in the document’s catalog dictionary
// (located via the Root entry in the file’s trailer, as described in 7.5.5, "File Trailer"),
// if present, shall be used instead of the version specified in the Header.
// Save PDF Version from header to xRefTable.
// The header version comes as the first line of the file.
func getHeaderVersion(ra io.ReaderAt) (*types.PDFVersion, error) {

	logDebugReader.Println("getHeaderVersion begin")

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
		return nil, errors.New("getHeaderVersion: corrupt pfd file - no header version available")
	}

	pdfVersion, err := types.Version(s[len(prefix) : len(prefix)+3])
	if err != nil {
		return nil, errors.Wrapf(err, "getHeaderVersion: unknown PDF Header Version")
	}

	logDebugReader.Printf("getHeaderVersion: end, found header version: %s\n", types.VersionString(pdfVersion))

	return &pdfVersion, nil
}

// Build XRefTable by reading XRef streams or XRef sections.
func buildXRefTableStartingAt(ctx *types.PDFContext, offset *int64) error {

	logDebugReader.Println("buildXRefTableStartingAt: begin")

	file := ctx.Read.File

	hv, err := getHeaderVersion(file)
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
		s.Split(ScanLines)

		line, err := scanLine(s)
		if err != nil {
			return err
		}

		logDebugReader.Printf("line: <%s>\n", line)

		if line != "xref" {

			logDebugReader.Println("buildXRefTableStartingAt: found xref stream")
			ctx.Read.UsingXRefStreams = true
			rd, err = newPositionedReader(file, offset)
			if err != nil {
				return err
			}

			if offset, err = parseXRefStream(rd, offset, ctx); err != nil {
				return err
			}

		} else {

			logDebugReader.Println("buildXRefTableStartingAt: found xref section")
			if offset, err = parseXRefSection(s, ctx); err != nil {
				return err
			}

		}
	}

	logDebugReader.Println("buildXRefTableStartingAt: end")

	return nil
}

// Populate the cross reference table for this PDF file.
// Goto offset of first xref table entry.
// Can be "xref" or indirect object reference eg. "34 0 obj"
// Keep digesting xref sections as long as there is a defined previous xref section
// and build up the xref table along the way.
func readXRefTable(ctx *types.PDFContext) (err error) {

	logDebugReader.Println("readXRefTable: begin")

	offset, err := getOffsetLastXRefSection(ctx.Read.File, ctx.Read.FileSize)
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
	//logDebugReader.Printf("freelist: %v\n", ctx.FreeObjects)

	// Ensure valid freelist of objects.
	err = ctx.EnsureValidFreeList()
	if err != nil {
		return
	}

	logDebugReader.Println("readXRefTable: end")

	return
}

func growBufBy(buf []byte, size int, rd io.Reader) ([]byte, error) {

	b := make([]byte, size)

	n, err := rd.Read(b)
	if err != nil {
		return nil, err
	}
	logDebugReader.Printf("growBufBy: Read %d bytes\n", n)

	return append(buf, b...), nil
}

func getStreamOffset(line string, streamInd int) (off int) {

	off = streamInd + len("stream")

	// Skip eol char.
	if line[off] == '\n' || line[off] == '\r' {
		off++
	}

	// Skip eol char.
	if line[off] == '\n' || line[off] == '\r' {
		off++
	}

	return
}

// Provide a PDF file buffer of sufficient size for parsing an object w/o stream.
func getBuffer(rd io.Reader) (buf []byte, endInd int, streamInd int, streamOffset int64, err error) {

	// process: # gen obj ... obj dict ... {stream ... data ... endstream} ... endobj
	//                                    streamInd                            endInd
	//                                  -1 if absent                        -1 if absent

	logDebugReader.Println("getBuffer: begin")

	endInd, streamInd = -1, -1

	for endInd < 0 && streamInd < 0 {

		buf, err = growBufBy(buf, defaultBufSize, rd)
		if err != nil {
			return
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

			if streamInd > len(line)-len("stream") {
				// No space for another stream marker.
				streamInd = -1
				break
			}

			// We start searching after this stream marker.
			bufpos := streamInd + len("stream")

			// Search for next stream marker.
			i := strings.Index(line[bufpos:], "stream")
			if i < 0 {
				// No stream marker within line buffer.
				streamInd = -1
				break
			}

			// We found the next stream marker.
			streamInd += len("stream") + i

			if endInd > 0 && streamInd > endInd {
				// We found a stream marker of another object
				streamInd = -1
				break
			}
		}

		logDebugReader.Printf("getBuffer: endInd=%d streamInd=%d\n", endInd, streamInd)

		if streamInd > 0 {

			// streamOffset ... the offset where the actual stream data begins.
			//                  is right after the eol after "stream".
			need := streamInd + len("stream") + 2 // 2 = maxLen(eol)

			if len(line) < need {

				// to prevent buffer overflow.
				buf, err = growBufBy(buf, need-len(line), rd)
				if err != nil {
					return
				}

				line = string(buf)
			}

			streamOffset = int64(getStreamOffset(line, streamInd))
		}
	}

	logDebugReader.Printf("getBuffer: end, returned bufsize=%d\n", len(buf))

	return
}

// return true if 'stream' follows end of dict: >>{whitespace}stream
func keywordStreamRightAfterEndOfDict(buf string, streamInd int) bool {

	logDebugReader.Println("keywordStreamRightAfterEndOfDict: begin")

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

	logDebugReader.Printf("keywordStreamRightAfterEndOfDict: end, %v\n", ok)

	return ok
}

// Return the filter pipeline associated with this stream dict.
func getPDFFilterPipeline(ctx *types.PDFContext, pdfDict types.PDFDict) ([]types.PDFFilter, error) {

	logDebugReader.Println("getPDFFilterPipeline: begin")

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
			logDebugReader.Println("getPDFFilterPipeline: end w/o decode parms")
			return append(filterPipeline, types.PDFFilter{Name: filterName, DecodeParms: nil}), nil
		}

		dict, ok := obj.(types.PDFDict)
		if !ok {
			return nil, errors.New("getPDFFilterPipeline: DecodeParms corrupt")
		}

		// with decode parameters.
		logDebugReader.Println("getPDFFilterPipeline: end with decode parms")
		return append(filterPipeline, types.PDFFilter{Name: filterName, DecodeParms: &dict}), nil
	}

	// filter pipeline.

	// Array of filternames
	filterArray, ok := obj.(types.PDFArray)
	if !ok {
		return nil, errors.Errorf("getPDFFilterPipeline: Expected filterArray corrupt, %v %T", obj, obj)
	}

	// Optional array of decode parameter dicts.
	var decodeParmsArr types.PDFArray
	decodeParms, found := pdfDict.Find("DecodeParms")
	if found {
		decodeParmsArr, ok = decodeParms.(types.PDFArray)
		if !ok {
			return nil, errors.New("getPDFFilterPipeline: Expected DecodeParms Array corrupt")
		}
	}

	for i, f := range filterArray {
		filterName, ok := f.(types.PDFName)
		if !ok {
			return nil, errors.New("getPDFFilterPipeline: FilterArray elements corrupt")
		}
		if decodeParms == nil || decodeParmsArr[i] == nil {
			filterPipeline = append(filterPipeline, types.PDFFilter{Name: filterName.String(), DecodeParms: nil})
			continue
		}

		decodeParmsDict, ok := decodeParmsArr[i].(types.PDFDict) // can be NULL if there are no DecodeParms!
		if !ok {
			return nil, errors.New("getPDFFilterPipeline: Expected DecodeParms Array corrupt")
		}
		filterPipeline = append(filterPipeline, types.PDFFilter{Name: filterName.String(), DecodeParms: &decodeParmsDict})
	}

	logDebugReader.Println("getPDFFilterPipeline: end")

	return filterPipeline, nil
}

// Parses an object from file at given offset.
func getObject(ctx *types.PDFContext, offset int64, objectNumber int, generationNumber int) (interface{}, error) {

	logDebugReader.Printf("getObject: begin, obj#%d, offset:%d\n", objectNumber, offset)

	rd, err := newPositionedReader(ctx.Read.File, &offset)
	if err != nil {
		return nil, err
	}

	logDebugReader.Printf("getObject: seeked to offset:%d\n", offset)

	// process: # gen obj ... obj dict ... {stream ... data ... endstream} endobj
	//                                    streamInd                        endInd
	//                                  -1 if absent                    -1 if absent
	buf, endInd, streamInd, streamOffset, err := getBuffer(rd)
	if err != nil {
		return nil, err
	}

	//logDebugReader.Printf("streamInd:%d(#%x) streamOffset:%d(#%x) endInd:%d(#%x)\n", streamInd, streamInd, streamOffset, streamOffset, endInd, endInd)
	//logDebugReader.Printf("buflen=%d\n%s", len(buf), hex.Dump(buf))

	line := string(buf)

	var l string

	if endInd < 0 { // && streamInd >= 0, streamdict
		// buf: # gen obj ... obj dict ... stream ... data
		// implies we detected no endobj and a stream starting at streamInd.
		// big stream, we parse object until "stream"
		logDebugReader.Println("getObject: big stream, we parse object until stream")
		l = line[:streamInd]
	} else if streamInd < 0 { // dict
		// buf: # gen obj ... obj dict ... endobj
		// implies we detected endobj and no stream.
		// small object w/o stream, parse until "endobj"
		logDebugReader.Println("getObject: small object w/o stream, parse until endobj")
		l = line[:endInd]
	} else if streamInd < endInd { // streamdict
		// buf: # gen obj ... obj dict ... stream ... data ... endstream endobj
		// implies we detected endobj and stream.
		// small stream within buffer, parse until "stream"
		logDebugReader.Println("getObject: small stream within buffer, parse until stream")
		l = line[:streamInd]
	} else { // dict
		// buf: # gen obj ... obj dict ... endobj # gen obj ... obj dict ... stream
		// small obj w/o stream, parse until "endobj"
		// stream in buf belongs to subsequent object.
		logDebugReader.Println("getObject: small obj w/o stream, parse until endobj")
		l = line[:endInd]
	}

	// Parse object number and object generation.
	objNr, genNr, err := parseObjectAttributes(&l)
	if err != nil {
		return nil, err
	}

	if *objNr != objectNumber || *genNr != generationNumber {
		return nil, errors.Errorf("getObject: non matching objNr(%d) or generationNumber(%d) tags found.", *objNr, *genNr)
	}

	pdfObject, err := parseObject(&l)
	if err != nil {
		return nil, err
	}

	switch o := pdfObject.(type) {

	case types.PDFDict:

	case types.PDFArray:
		if ctx.EncKey != nil {
			if _, err = decryptDeepObject(o, *objNr, *genNr, ctx.EncKey, ctx.AES4Strings); err != nil {
				return nil, err
			}
		}
		return o, nil

	case types.PDFStringLiteral:
		if ctx.EncKey != nil {
			s1, err := decryptString(o.Value(), *objNr, *genNr, ctx.EncKey, ctx.AES4Strings)
			if err != nil {
				return nil, err
			}
			return types.PDFStringLiteral(*s1), nil
		}
		return o, nil

	default:
		return o, nil
	}

	pdfDict := pdfObject.(types.PDFDict)

	if ctx.EncKey != nil {
		_, err = decryptDeepObject(pdfDict, *objNr, *genNr, ctx.EncKey, ctx.AES4Strings)
		if err != nil {
			return nil, err
		}
	}

	streamLength, streamLengthRef := pdfDict.Length()

	if endInd >= 0 && (streamInd < 0 || streamInd > endInd) {
		logDebugReader.Printf("getObject: end, #%d\n", objectNumber)
		return pdfDict, nil
	}

	// Parse associated stream data into a PDFStreamDict.

	if streamInd <= 0 {
		return nil, errors.New("getObject: stream object without streamOffset")
	}

	filterPipeline, err := getPDFFilterPipeline(ctx, pdfDict)
	if err != nil {
		return nil, err
	}

	streamOffset += offset

	// We have a stream object.
	pdfStreamDict := types.NewPDFStreamDict(pdfDict, streamOffset, streamLength, streamLengthRef, filterPipeline)

	logDebugReader.Printf("getObject: end, Streamobject #%d\n", objectNumber)

	return pdfStreamDict, nil
}

func dereferencedObject(ctx *types.PDFContext, objectNumber int) (interface{}, error) {

	entry, ok := ctx.Find(objectNumber)
	if !ok {
		return nil, errors.New("dereferencedObject: object not registered in xRefTable")
	}

	if entry.Compressed {
		decompressXRefTableEntry(ctx.XRefTable, objectNumber, entry)
	}

	if entry.Object == nil {

		// dereference this object!

		logDebugReader.Printf("dereferencedObject: dereferencing object %d\n", objectNumber)

		obj, err := getObject(ctx, *entry.Offset, objectNumber, *entry.Generation)
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
func getInt64Object(ctx *types.PDFContext, objectNumber int) (*int64, error) {

	logDebugReader.Printf("getInt64Object begin: %d\n", objectNumber)

	obj, err := dereferencedObject(ctx, objectNumber)
	if err != nil {
		return nil, err
	}

	i, ok := obj.(types.PDFInteger)
	if !ok {
		return nil, errors.New("getInt64Object: object is not PDFInteger")
	}

	i64 := int64(i.Value())

	logDebugReader.Printf("getInt64Object end: %d\n", objectNumber)

	return &i64, nil

}

// Reads and returns a file buffer with length = stream length using provided reader positioned at offset.
func readContentStream(rd io.Reader, streamLength int) (buf []byte, err error) {

	logDebugReader.Printf("readContentStream: begin streamLength:%d\n", streamLength)

	buf = make([]byte, streamLength)

	for totalCount := 0; totalCount < streamLength; {
		count, err := rd.Read(buf[totalCount:])
		if err != nil {
			return nil, err
		}
		logDebugReader.Printf("readContentStream: count=%d, buflen=%d(%X)\n", count, len(buf), len(buf))
		totalCount += count
	}

	logDebugReader.Printf("readContentStream: end\n")

	return
}

// GetEncodedStreamContent loads the encoded stream content from file into PDFStreamDict.
func GetEncodedStreamContent(ctx *types.PDFContext, streamDict *types.PDFStreamDict) ([]byte, error) {

	logDebugReader.Printf("GetEncodedStreamContent: begin\n%v\n", streamDict)

	var err error

	// Return saved decoded content.
	if streamDict.Raw != nil {
		logDebugReader.Println("GetEncodedStreamContent: end, already in memory.")
		return streamDict.Raw, nil
	}

	// Read stream content encoded at offset with stream length.

	// Dereference stream length if stream length is an indirect object.
	if streamDict.StreamLength == nil {
		if streamDict.StreamLengthObjNr == nil {
			return nil, errors.New("GetEncodedStreamContent: missing streamLength")
		}
		// Get stream length from indirect object
		streamDict.StreamLength, err = getInt64Object(ctx, *streamDict.StreamLengthObjNr)
		if err != nil {
			return nil, err
		}
		logDebugReader.Printf("GetEncodedStreamContent: new indirect streamLength:%d\n", *streamDict.StreamLength)
	}

	newOffset := streamDict.StreamOffset
	rd, err := newPositionedReader(ctx.Read.File, &newOffset)
	if err != nil {
		return nil, err
	}

	logDebugReader.Printf("GetEncodedStreamContent: seeked to offset:%d\n", newOffset)

	// Buffer stream contents.
	// Read content from disk.
	rawContent, err := readContentStream(rd, int(*streamDict.StreamLength))
	if err != nil {
		return nil, err
	}

	//logDebugReader.Printf("rawContent buflen=%d(#%x)\n%s", len(rawContent), len(rawContent), hex.Dump(rawContent))

	// Save encoded content.
	streamDict.Raw = rawContent

	logDebugReader.Println("GetEncodedStreamContent: end")

	// Return encoded content.
	return rawContent, nil
}

// Decodes the raw encoded stream content and saves it to streamDict.Content.
func setDecodedStreamContent(ctx *types.PDFContext, streamDict *types.PDFStreamDict, objNr, genNr int, decode bool) (err error) {

	logDebugReader.Println("setDecodedStreamContent: begin")

	if ctx != nil && ctx.EncKey != nil &&
		len(streamDict.FilterPipeline) == 1 && streamDict.FilterPipeline[0].Name == "Crypt" {
		streamDict.Content = streamDict.Raw
		return
	}

	// ctx gets created after XRefStream parsing.
	// XRefStreams are not encrypted.
	if ctx != nil && ctx.EncKey != nil {
		streamDict.Raw, err = decryptStream(streamDict.Raw, objNr, genNr, ctx.EncKey, ctx.AES4Streams)
		if err != nil {
			return
		}
		l := int64(len(streamDict.Raw))
		streamDict.StreamLength = &l
	}

	if !decode {
		return
	}

	// Actual decoding of content stream.
	err = filter.DecodeStream(streamDict)
	if err != nil {
		return
	}

	logDebugReader.Println("setDecodedStreamContent: end")

	return
}

// Resolve compressed xRefTableEntry
func decompressXRefTableEntry(xRefTable *types.XRefTable, objectNumber int, entry *types.XRefTableEntry) error {

	logDebugReader.Printf("decompressXRefTableEntry: compressed object %d at %d[%d]\n", objectNumber, *entry.ObjectStream, *entry.ObjectStreamInd)

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
	pdfObject, err := pdfObjectStreamDict.GetIndexedObject(*entry.ObjectStreamInd)
	if err != nil {
		return errors.Wrapf(err, "decompressXRefTableEntry: problem dereferencing object stream %d", *entry.ObjectStream)
	}

	// Save object to XRefRableEntry.
	g := 0
	entry.Object = pdfObject
	entry.Generation = &g
	entry.Compressed = false

	logDebugReader.Printf("decompressXRefTableEntry: end, Obj %d[%d]:\n<%s>\n", *entry.ObjectStream, *entry.ObjectStreamInd, pdfObject)

	return nil
}

// Log interesting stream content.
func logStream(obj interface{}) {

	switch obj := obj.(type) {

	case types.PDFStreamDict:

		if obj.Content == nil {
			logDebugReader.Println("logStream: no stream content")
		}

		if obj.IsPageContent {
			//logDebugReader.Printf("content <%s>\n", pdfStreamDict.Content)
		}

	case types.PDFObjectStreamDict:

		if obj.Content == nil {
			logDebugReader.Println("logStream: no object stream content")
		} else {
			logDebugReader.Printf("logStream: objectStream content = %s\n", obj.Content)
		}

		if obj.ObjArray == nil {
			logDebugReader.Println("logStream: no object stream obj arr")
		} else {
			logDebugReader.Printf("logStream: objectStream objArr = %s\n", obj.ObjArray)
		}

	default:
		logDebugReader.Println("logStream: no pdfObjectStreamDict")

	}

}

// Decode all object streams so contained objects are ready to be used.
func decodeObjectStreams(ctx *types.PDFContext) (err error) {

	// Note:
	// Entry "Extends" intentionally left out.
	// No object stream collection validation necessary.

	logDebugReader.Println("decodeObjectStreams: begin")

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

		logDebugReader.Printf("decodeObjectStreams: parsing object stream for obj#%d\n", objectNumber)

		// Parse object stream from file.
		obj, err := getObject(ctx, *entry.Offset, objectNumber, *entry.Generation)
		if err != nil || obj == nil {
			return errors.New("decodeObjectStreams: corrupt object stream")
		}

		// Ensure PDFStreamDict
		pdfStreamDict, ok := obj.(types.PDFStreamDict)
		if !ok {
			return errors.New("decodeObjectStreams: corrupt object stream")
		}

		// Save encoded stream content to xRefTable.
		if _, err = GetEncodedStreamContent(ctx, &pdfStreamDict); err != nil {
			return errors.Wrapf(err, "decodeObjectStreams: problem dereferencing object stream %d", objectNumber)
		}

		// Save decoded stream content to xRefTable.
		if err = setDecodedStreamContent(ctx, &pdfStreamDict, objectNumber, *entry.Generation, true); err != nil {
			logErrorReader.Printf("obj %d: %s", objectNumber, err)
			return err
		}

		// Ensure decoded objectArray for object stream dicts.
		if !pdfStreamDict.IsObjStm() {
			return errors.New("decodeObjectStreams: corrupt object stream")
		}

		// We have an object stream.
		logDebugReader.Printf("decodeObjectStreams: object stream #%d\n", objectNumber)

		ctx.Read.UsingObjectStreams = true

		// Create new object stream dict.
		pdfObjectStreamDict, err := objectStreamDict(pdfStreamDict)
		if err != nil {
			return errors.Wrapf(err, "decodeObjectStreams: problem dereferencing object stream %d", objectNumber)
		}

		logDebugReader.Printf("decodeObjectStreams: decoding object stream %d:\n", objectNumber)

		// Parse all objects of this object stream and save them to pdfObjectStreamDict.ObjArray.
		if err = parseObjectStream(pdfObjectStreamDict); err != nil {
			return errors.Wrapf(err, "decodeObjectStreams: problem decoding object stream %d\n", objectNumber)
		}

		if pdfObjectStreamDict.ObjArray == nil {
			return errors.Wrap(err, "decodeObjectStreams: objArray should be set!")
		}

		logDebugReader.Printf("decodeObjectStreams: decoded object stream %d:\n", objectNumber)

		// Save object stream dict to xRefTableEntry.
		entry.Object = *pdfObjectStreamDict
	}

	logDebugReader.Println("decodeObjectStreams: end")

	return
}

// Dereferences all objects including compressed objects from object streams.
func dereferenceObjects(ctx *types.PDFContext) error {

	xRefTable := ctx.XRefTable

	logDebugReader.Println("dereferenceObjects: begin")

	xRefTableSize := len(xRefTable.Table)

	// Get sorted slice of object numbers.
	var keys []int
	for k := range xRefTable.Table {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, objectNumber := range keys {

		logDebugReader.Printf("dereferenceObjects: dereferencing object %d\n", objectNumber)

		entry := xRefTable.Table[objectNumber]

		if entry.Free {
			//logDebugReader.Printf("free object %d\n", objectNumber)
			continue
		}

		if entry.Compressed {
			err := decompressXRefTableEntry(xRefTable, objectNumber, entry)
			if err != nil {
				return err
			}
			logDebugReader.Printf("dereferenceObjects: decompressed entry, Compressed=%v\n%s\n", entry.Compressed, entry.Object)
			continue
		}

		// entry is in use.

		if entry.Offset == nil || *entry.Offset == 0 {
			logDebugReader.Printf("dereferenceObjects: already decompressed or used object w/o offset -> ignored")
			continue
		}

		obj := entry.Object
		var err error

		// Already dereferenced stream dict.
		if obj != nil {

			logStream(entry.Object)

			switch obj := obj.(type) {
			case types.PDFStreamDict:
				ctx.Read.BinaryTotalSize += *obj.StreamLength
			case types.PDFObjectStreamDict:
				ctx.Read.BinaryTotalSize += *obj.StreamLength
			case types.PDFXRefStreamDict:
				ctx.Read.BinaryTotalSize += *obj.StreamLength
			}

			logDebugReader.Printf("dereferenceObjects: using cached object %d of %d\n<%s>\n", objectNumber, xRefTableSize, entry.Object)

			continue
		}

		// Dereference (load from disk into memory).

		logDebugReader.Printf("dereferenceObjects: dereferencing object %d\n", objectNumber)

		// Parse object from file: anything goes dict,array,integer,float,streamdicts..
		obj, err = getObject(ctx, *entry.Offset, objectNumber, *entry.Generation)
		if err != nil {
			return errors.Wrapf(err, "dereferenceObjects: problem dereferencing object %d", objectNumber)
		}

		entry.Object = obj

		// Linearization dicts are only validated and recorded for stats.
		if !ctx.Read.Linearized {

			// handle linearization parm dict.
			if pdfDict, ok := obj.(types.PDFDict); ok && pdfDict.IsLinearizationParmDict() {

				ctx.Read.Linearized = true
				xRefTable.LinearizationObjs[objectNumber] = true
				logDebugReader.Printf("dereferenceObjects: identified linearizationObj #%d\n", objectNumber)

				arr := pdfDict.PDFArrayEntry("H")

				if arr == nil {
					return errors.Errorf("dereferenceObjects: corrupt linearization dict at obj:%d - missing array entry H", objectNumber)
				}

				if len(*arr) != 2 && len(*arr) != 4 {
					return errors.Errorf("dereferenceObjects: corrupt linearization dict at obj:%d - corrupt array entry H, needs length 2 or 4", objectNumber)
				}

				offset, ok := (*arr)[0].(types.PDFInteger)
				if !ok {
					return errors.Errorf("dereferenceObjects: corrupt linearization dict at obj:%d - corrupt array entry H, needs Integer values", objectNumber)
				}

				offset64 := int64(offset.Value())
				xRefTable.OffsetPrimaryHintTable = &offset64

				if len(*arr) == 4 {

					offset, ok := (*arr)[2].(types.PDFInteger)
					if !ok {
						return errors.Errorf("dereferenceObjects: corrupt linearization dict at obj:%d - corrupt array entry H, needs Integer values", objectNumber)
					}

					offset64 := int64(offset.Value())
					xRefTable.OffsetOverflowHintTable = &offset64
				}
			}
		}

		// Handle stream dicts.

		if _, ok := obj.(types.PDFObjectStreamDict); ok {
			return errors.Errorf("dereferenceObjects: object stream should already be dereferenced at obj:%d", objectNumber)
		}

		if _, ok := obj.(types.PDFXRefStreamDict); ok {
			return errors.Errorf("dereferenceObjects: xref stream should already be dereferenced at obj:%d", objectNumber)
		}

		// Save encoded stream content for stream dicts into xRefTable entry.
		if pdfStreamDict, ok := obj.(types.PDFStreamDict); ok {

			// fontfiles, images, ?

			if _, err = GetEncodedStreamContent(ctx, &pdfStreamDict); err != nil {
				return errors.Wrapf(err, "dereferenceObjects: problem dereferencing stream %d", objectNumber)
			}

			ctx.Read.BinaryTotalSize += *pdfStreamDict.StreamLength

			err = setDecodedStreamContent(ctx, &pdfStreamDict, objectNumber, *entry.Generation, ctx.DecodeAllStreams)
			if err != nil {
				return err
			}

			entry.Object = pdfStreamDict
		}

		logDebugReader.Printf("dereferenceObjects: end obj %d of %d\n<%s>\n", objectNumber, xRefTableSize, entry.Object)

		logStream(entry.Object)
	}

	logDebugReader.Println("dereferenceObjects: end")

	return nil
}

// Locate a possible Version entry (since V1.4) in the catalog
// and record this as rootVersion (as opposed to headerVersion).
func identifyRootVersion(xRefTable *types.XRefTable) (err error) {

	logDebugReader.Println("identifyRootVersion: begin")

	// Try to get Version from Root.
	rootVersionStr, err := xRefTable.ParseRootVersion()
	if err != nil {
		return
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
		logInfoReader.Printf("identifyRootVersion: PDF version is %s - will ignore root version: %s\n",
			types.VersionString(*xRefTable.HeaderVersion), *rootVersionStr)
	}

	logDebugReader.Println("identifyRootVersion: end")

	return
}

// Parse all PDFObjects including stream content from file and save to the corresponding xRefTableEntries.
// This includes processing of object streams and linearization dicts.
func dereferenceXRefTable(ctx *types.PDFContext, config *types.Configuration) (err error) {

	logDebugReader.Println("dereferenceXRefTable: begin")

	xRefTable := ctx.XRefTable

	// Note for encrypted files:
	// Mandatory supply userpw to open & display file.
	// Access may be restricted (Decode access privileges).
	// Optionally supply ownerpw in order to gain unrestricted access.
	err = checkForEncryption(ctx, config.UserPW, config.OwnerPW)
	if err != nil {
		return
	}
	//logErrorReader.Println("pw authenticated")

	// Prepare decompressed objects.
	err = decodeObjectStreams(ctx)
	if err != nil {
		return
	}

	// For each xRefTableEntry assign a PDFObject either by parsing from file or pointing to a decompressed object.
	err = dereferenceObjects(ctx)
	if err != nil {
		return
	}

	// Identify an optional Version entry in the root object/catalog.
	err = identifyRootVersion(xRefTable)
	if err != nil {
		return
	}

	logDebugReader.Println("dereferenceXRefTable: end")

	return
}

// PDFFile reads in a PDFFile and generates a PDFContext, an in-memory representation containing a cross reference table.
func PDFFile(fileName string, config *types.Configuration) (ctx *types.PDFContext, err error) {

	logDebugReader.Println("PDFFile: begin")

	file, err := os.Open(fileName)
	if err != nil {
		return nil, errors.Wrapf(err, "can't open %q", fileName)
	}

	defer func() {
		file.Close()
	}()

	ctx, err = types.NewPDFContext(fileName, file, config)
	if err != nil {
		return
	}

	if ctx.Reader15 {
		logInfoReader.Println("PDF Version 1.5 conforming reader")
	} else {
		logErrorReader.Println("PDF Version 1.4 conforming reader - no object streams and xrefstreams allowed")
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
		return
	}

	logDebugReader.Println("PDFFile: end")

	return
}
