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
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/umats/pdfcpu/internal/contextutil"
	"github.com/umats/pdfcpu/pkg/filter"
	"github.com/umats/pdfcpu/pkg/log"
	"github.com/umats/pdfcpu/pkg/pdfcpu/model"
	"github.com/umats/pdfcpu/pkg/pdfcpu/safemath"
	"github.com/umats/pdfcpu/pkg/pdfcpu/scan"
	"github.com/umats/pdfcpu/pkg/pdfcpu/types"
)

const (
	defaultBufSize     = 1 << 10 // 1 KiB
	maxBufSize         = 1 << 20 // 1 MiB
	maxObjectBufferLen = 64 << 20
)

var (
	// ErrCorruptHeader reports a missing or malformed PDF header.
	ErrCorruptHeader = errors.New("no header version available")

	// ErrEmptyInput reports an empty input stream passed to the PDF reader.
	ErrEmptyInput = errors.New("the file could not be opened because it is empty")

	// ErrEncrypted reports an operation that cannot process encrypted PDFs.
	ErrEncrypted = errors.New("this file is encrypted")

	// ErrMissingXRefSection reports a PDF without a detectable final xref section.
	ErrMissingXRefSection = errors.New("can't detect last xref section")

	// ErrNotEncrypted reports an operation requiring encryption on an unencrypted PDF.
	ErrNotEncrypted = errors.New("this file is not encrypted")

	// ErrOwnerPasswordRequired reports missing owner credentials for an encryption operation.
	ErrOwnerPasswordRequired = errors.New("please provide owner password and optional user password")

	// ErrPermissionDenied reports a PDF operation blocked by encryption permission bits.
	ErrPermissionDenied = errors.New("operation restricted via permission bits setting")

	// ErrPostScriptInput reports a PostScript input stream passed to the PDF reader.
	ErrPostScriptInput = errors.New("PostScript input is not supported; convert to PDF before processing")

	// ErrReferenceDoesNotExist reports a reference to a missing indirect object.
	ErrReferenceDoesNotExist = errors.New("referenced object does not exist")

	// ErrWrongPassword reports supplied PDF credentials that fail password authentication.
	ErrWrongPassword = errors.New("please provide the correct password")

	errCorruptDecodeParms           = errors.New("corrupt decode parameters")
	errCorruptDictObject            = errors.New("corrupt dict object")
	errCorruptFilterArray           = errors.New("corrupt filter array")
	errCorruptIntegerObject         = errors.New("corrupt integer object")
	errCorruptObjectStream          = errors.New("corrupt object stream")
	errCorruptObjectStreamDict      = errors.New("corrupt object stream dict")
	errCorruptStreamDict            = errors.New("corrupt stream dict")
	errCorruptStreamMarker          = errors.New("corrupt stream marker")
	errCorruptTrailerDict           = errors.New("corrupt trailer dict")
	errCorruptXRefStream            = errors.New("corrupt xref stream")
	errCorruptXRefSubsection        = errors.New("corrupt xref subsection")
	errIncompleteXRefSubsection     = errors.New("incomplete xref subsection")
	errInvalidLastXRefSection       = errors.New("invalid last xref section")
	errInvalidPermissions           = errors.New("invalid permissions")
	errInvalidTrailerID             = errors.New("invalid trailer ID")
	errInvalidXRefStreamWArray      = errors.New("invalid xref stream W array")
	errMissingObjectStream          = errors.New("missing object stream")
	errMissingObjectStreamEntry     = errors.New("missing object stream xref entry")
	errMissingObjectStreamObjects   = errors.New("missing object stream objects")
	errMissingReadSeeker            = errors.New("missing PDF read seeker")
	errMissingScannerLine           = errors.New("missing scanner line")
	errMissingStreamLength          = errors.New("missing stream length")
	errMissingStreamOffset          = errors.New("missing stream offset")
	errMissingTrailerDict           = errors.New("missing trailer dict")
	errMissingTrailerID             = errors.New("missing trailer ID")
	errMissingTrailerRoot           = errors.New("missing trailer Root")
	errMissingTrailerSize           = errors.New("missing trailer Size")
	errMissingXRefEOF               = errors.New("no matching %%EOF for startxref")
	errMissingXRefStreamDict        = errors.New("missing xref stream dictionary")
	errMissingXRefStreamLength      = errors.New("missing xref stream Length")
	errNilDereferencedObject        = errors.New("nil dereferenced object")
	errObjectBufferLimit            = errors.New("object buffer limit exceeded")
	errObjectStreamContainsStream   = errors.New("object stream contains stream object")
	errObjectStreamObjectCountLimit = errors.New("object stream object count limit exceeded")
	errTruncatedStreamMarker        = errors.New("truncated stream marker")
	errUnregisteredObject           = errors.New("unregistered object")

	zero int64 = 0
)

func hasPostScriptHeader(buf []byte) bool {
	buf = bytes.TrimPrefix(buf, []byte{0xEF, 0xBB, 0xBF})
	buf = bytes.TrimLeft(buf, "\x00\t\n\f\r ")
	if !bytes.HasPrefix(buf, []byte("%!")) {
		return false
	}
	i := bytes.IndexAny(buf, "\x0A\x0D")
	if i < 0 {
		i = len(buf)
	}
	return bytes.Contains(buf[:i], []byte("PS-Adobe"))
}

// ReadFile reads a PDF file and builds the internal model context holding its cross reference table.
// If the passed Go context is cancelled, reading will be interrupted.
func ReadFile(c context.Context, inFile string, conf *model.Configuration) (*model.Context, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}

	if log.InfoEnabled() {
		log.Info.Printf("reading %s..\n", inFile)
	}

	f, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf("can't open %q: %w", inFile, err)
	}

	defer func() {
		f.Close()
	}()

	return Read(c, f, conf)
}

// Read takes a readSeeker and generates a PDF model context,
// an in-memory representation containing a cross reference table.
// If the passed Go context is cancelled, reading will be interrupted.
func Read(c context.Context, rs io.ReadSeeker, conf *model.Configuration) (*model.Context, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}

	if log.ReadEnabled() {
		log.Read.Println("Read: begin")
	}

	if rs == nil {
		return nil, errMissingReadSeeker
	}

	ctx, err := model.NewContext(rs, conf)
	if err != nil {
		return nil, err
	}

	if ctx.Read.FileSize == 0 {
		return nil, ErrEmptyInput
	}

	if log.InfoEnabled() {
		if ctx.Reader15 {
			log.Info.Println("PDF Version 1.5 conforming reader")
		} else {
			log.Info.Println("PDF Version 1.4 conforming reader - no object streams or xrefstreams allowed")
		}
	}

	// Preserve ordinary parsing semantics while retaining ambiguity from
	// every trailer/object origin, including later lazy materialization.
	c = contextutil.WithParseProvenance(c, &contextutil.ParseProvenance{})

	// Populate xRefTable.
	if err = readXRefTable(c, ctx); err != nil {
		return nil, fmt.Errorf("xref table: %w", err)
	}

	// Detect effective trailer Size repair before hint authorization. Keep
	// the existing Size correction after ordinary object loading below.
	if ctx.XRefTable.Size == nil || *ctx.XRefTable.Size != maxObjectNumber(ctx.Table)+1 {
		ctx.Read.RepairedXRef = true
	}

	// Make all objects explicitly available (load into memory) in corresponding xRefTable entries.
	// Also decode any involved object streams.
	if err = dereferenceXRefTable(c, ctx); err != nil {
		return nil, err
	}

	// Some PDFWriters write an incorrect Size into trailer.
	if ctx.XRefTable.Size == nil || *ctx.XRefTable.Size != ctx.MaxObjNr+1 {
		maxObjNr := ctx.MaxObjNr + 1
		ctx.XRefTable.Size = &maxObjNr
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

func incrEpilogIndex(s string) int {
	suffixes := []string{"%%EO", "%%E", "%%", "%"}
	for _, suf := range suffixes {
		if strings.HasSuffix(s, suf) {
			return len(s) - len(suf)
		}
	}
	return -1
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
			return nil, ErrMissingXRefSection
		}

		if log.ReadEnabled() {
			log.Read.Printf("scanning for offsetLastXRefSection starting at %d\n", off)
		}

		curBuf := make([]byte, bufSize)

		if _, err = fillBuffer(rs, curBuf); err != nil {
			return nil, fmt.Errorf("scan for startxref: %w", err)
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

		start := j + len("startxref")
		if start >= len(workBuf) {
			return nil, errMissingXRefEOF
		}

		// Substring discovery must not treat a marker inside a comment or
		// discarded same-line prefix as exact epilogue framing.
		lineStart := strings.LastIndexAny(string(workBuf[:j]), "\r\n") + 1
		for _, b := range workBuf[lineStart:j] {
			if !contextutil.PDFWhitespace(rune(b)) {
				ctx.Read.RepairedXRef = true
			}
		}
		if !contextutil.PDFWhitespace(rune(workBuf[start])) ||
			(j > 0 && !contextutil.PDFTokenBoundary(string(workBuf), j-1)) {
			ctx.Read.RepairedXRef = true
		}
		p := workBuf[start+1:]
		posEOF := strings.Index(string(p), "%%EOF")
		if posEOF < 0 {
			ctx.Read.RepairedXRef = true
			posEOF = incrEpilogIndex(string(p))
			if posEOF < 0 {
				return nil, errMissingXRefEOF
			}
		}

		p = p[:posEOF]
		auditXRefWhitespace(string(p), &ctx.Read.RepairedXRef)
		s := strings.TrimSpace(string(p))
		if strings.HasSuffix(s, "%") {
			ctx.Read.RepairedXRef = true
		}
		s = strings.TrimSpace(strings.TrimRight(s, "%"))
		offset, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, errInvalidLastXRefSection
		}
		if offset == 0 || offset >= ctx.Read.FileSize {
			ctx.Read.RepairedXRef = true
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

func createXRefTableEntry(entryType string, objNr int, offset, offExtra int64, generation, incr int) (model.XRefTableEntry, bool) {
	entry := model.XRefTableEntry{Offset: &offset, Generation: &generation, Incr: incr}

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

		if offset < 9 {
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

func decodeSubsection(fields []string) (int64, int, string, error) {
	s := fields[0]
	for len(s) > 0 && !unicode.IsDigit(rune(s[0])) {
		s = s[1:]
	}

	offset, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, 0, "", err
	}

	s = "00000"
	// model.ShowRepaired
	if len(fields[1]) <= 5 {
		s = fields[1]
	}
	generation, err := strconv.Atoi(s)
	if err != nil {
		return 0, 0, "", err
	}

	entryType := fields[2]
	if entryType != "f" && entryType != "n" {
		return 0, 0, "", errCorruptXRefSubsection
	}

	return offset, generation, entryType, nil
}

// Read next subsection entry and generate corresponding xref table entry.
func parseXRefTableEntry(xRefTable *model.XRefTable, fields []string, objNr int, offExtra int64, incr int, unreliable ...*bool) error {
	if log.ReadEnabled() {
		log.Read.Println("parseXRefTableEntry: begin")
	}

	offset, generation, entryType, err := decodeSubsection(fields)
	if err != nil {
		return err
	}

	if len(unreliable) > 0 && (len(fields[0]) != 10 || len(fields[1]) != 5 ||
		strings.Trim(fields[0], "0123456789") != "" || strings.Trim(fields[1], "0123456789") != "" ||
		(entryType == "n" && offset < 9)) {
		*unreliable[0] = true
	}
	if entryType == "n" && objNr > 0 && offset == 0 {
		err := model.WithValidationErrorObject(errors.New("in-use xref entry has offset 0"), objNr)
		if xRefTable.ValidationMode == model.ValidationStrict {
			return fmt.Errorf("object %d: %w", objNr, err)
		}
		xRefTable.AddValidationNotice(model.NewValidationNotice(model.NoticePhaseRead, model.NoticeSkipped, err.Error(), err))
		return nil
	}

	entry, ok := createXRefTableEntry(entryType, objNr, offset, offExtra, generation, incr)
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

func parseXRefTableSubsectionHeader(fields []string, entryCount int, limits model.ResourceLimits) (int, int, error) {
	startObjNumber, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, 0, err
	}

	objCount, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, 0, err
	}

	if startObjNumber < 0 || objCount < 0 {
		return 0, 0, errCorruptXRefSubsection
	}
	if objCount > limits.MaxObjectCount || startObjNumber > limits.MaxObjectCount-objCount {
		return 0, 0, fmt.Errorf("xref subsection object range exceeds limit %d: %w", limits.MaxObjectCount, errCorruptXRefSubsection)
	}
	if entryCount > limits.MaxXRefEntries || objCount > limits.MaxXRefEntries-entryCount {
		return 0, 0, fmt.Errorf("xref entry count exceeds limit %d", limits.MaxXRefEntries)
	}

	return startObjNumber, objCount, nil
}

func xRefTableObjectNumber(startObjNumber, index, entryCount int, limits model.ResourceLimits) (int, error) {
	if entryCount >= limits.MaxXRefEntries {
		return 0, fmt.Errorf("xref entry count exceeds limit %d", limits.MaxXRefEntries)
	}
	if index >= limits.MaxObjectCount-startObjNumber {
		return 0, fmt.Errorf("xref subsection object range exceeds limit %d: %w", limits.MaxObjectCount, errCorruptXRefSubsection)
	}
	return startObjNumber + index, nil
}

// auditXRefWhitespace examines raw consumed separators, before Fields/TrimSpace
// erases them. Call only on xref framing, never dictionary/string/comment payload.
func auditXRefWhitespace(line string, unreliable ...*bool) {
	if len(unreliable) == 0 {
		return
	}
	for _, r := range line {
		if unicode.IsSpace(r) && !contextutil.PDFWhitespace(r) {
			*unreliable[0] = true
		}
	}
}

// Process xRef table subsection and create corresponding xRef table entries.
func parseXRefTableSubSection(xRefTable *model.XRefTable, limits model.ResourceLimits, s *bufio.Scanner, fields []string, offExtra int64, incr int, unreliable ...*bool) (string, error) {
	var line string
	trailer := false
	entryCount := 0
	// This call consumes one section, including all its subsections. Older
	// revisions may repeat numbers; assignments within this section may not
	// establish trustworthy hint provenance.
	seen := types.IntSet{}

	for !trailer && len(fields) == 2 {

		startObjNumber, objCount, err := parseXRefTableSubsectionHeader(fields, entryCount, limits)
		if err != nil {
			return "", err
		}

		if log.ReadEnabled() {
			log.Read.Printf("detected xref subsection, startObj=%d length=%d\n", startObjNumber, objCount)
		}

		// Process all entries of this subsection into xRefTable entries.
		for i := 0; ; i++ {
			line, err = scanLine(s)
			if err != nil {
				return "", err
			}

			if !strings.HasPrefix(strings.TrimSpace(line), "trailer") {
				auditXRefWhitespace(line, unreliable...)
			}
			fields = strings.Fields(line)
			if len(fields) != 3 {
				if i < objCount {
					return "", errIncompleteXRefSubsection
				}
				trailer = strings.Contains(line, "trailer")
				break
			}

			objNr, err := xRefTableObjectNumber(startObjNumber, i, entryCount, limits)
			if err != nil {
				return "", err
			}
			entryCount++
			if seen[objNr] && len(unreliable) > 0 {
				*unreliable[0] = true
			}
			seen[objNr] = true
			// Audit each section's records before applying older-revision
			// precedence. Ignored origins still affect authorization reliability.
			if len(unreliable) > 0 && (i >= objCount || len(fields[0]) != 10 || len(fields[1]) != 5 ||
				strings.Trim(fields[0], "0123456789") != "" || strings.Trim(fields[1], "0123456789") != "" ||
				(fields[2] != "n" && fields[2] != "f") ||
				(fields[2] == "n" && fields[0] < "0000000009")) {
				*unreliable[0] = true
			}

			if xRefTable.Exists(objNr) {
				if log.ReadEnabled() {
					log.Read.Printf("parseXRefTableEntry: end - Skip entry %d - already assigned\n", objNr)
				}
				// Add incr!
				continue
			}

			if len(fields[0]) == 1 { // should be 10
				if len(unreliable) > 0 {
					*unreliable[0] = true
				}
				continue
			}

			if i >= objCount {
				if len(unreliable) > 0 {
					*unreliable[0] = true
				}
				model.ShowMsg(fmt.Sprintf("digesting extra XrefTable entry for obj#%d", objNr))
			}
			if err = parseXRefTableEntry(xRefTable, fields, objNr, offExtra, incr, unreliable...); err != nil {
				return "", err
			}
		}
	}

	return line, nil
}

// markObjectRemainder applies only at complete structural boundaries, never
// the public parse-next-value API. Preserve acceptance but do not authorize
// hints when discarded syntax could contain invisible incoming references.
func markObjectRemainder(c context.Context, s string) {
	for len(s) > 0 {
		if strings.ContainsRune("\x00\t\n\f\r ", rune(s[0])) {
			s = s[1:]
			continue
		}
		if s[0] != '%' {
			contextutil.ParseProvenanceFromContext(c).MarkRepair()
			return
		}
		i := strings.IndexAny(s, "\r\n")
		if i < 0 {
			return
		}
		s = s[i:]
	}
}

// Parse compressed object.
func compressedObject(c context.Context, s string, maxDepth ...int) (types.Object, error) {
	if log.ReadEnabled() {
		log.Read.Println("compressedObject: begin")
	}

	o, err := model.ParseObject(c, &s, 0, maxDepth...)
	if err != nil {
		return nil, err
	}

	markObjectRemainder(c, s)
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

	return nil, errObjectStreamContainsStream
}

// lexicalCutState tracks only framing state, not PDF values. It is used on
// already bounded decoded origins so a slice EOF cannot terminate a token,
// comment, string, or open container. Each audit scans only its bounded slice.
func lexicalCutState(c context.Context, b []byte, start, end int) (byte, error) {
	var mode byte
	depth, containers := 0, 0
	escaped := false
	for i := start; i < end; i++ {
		if (i-start)%4096 == 0 {
			if err := c.Err(); err != nil {
				return 0, err
			}
		}
		ch := b[i]
		switch mode {
		case 'c':
			if ch == '\r' || ch == '\n' {
				mode = 0
			}
			continue
		case 's':
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '(' {
				depth++
			}
			if ch == ')' {
				depth--
				if depth == 0 {
					mode = 0
				}
			}
			continue
		case 'h':
			if ch == '>' {
				mode = 0
			}
			continue
		case 'r', 'n':
			if !contextutil.PDFTokenBoundary(string(b[i:i+1]), 0) {
				continue
			}
			mode = 0
		}
		switch ch {
		case '%':
			mode = 'c'
		case '/':
			mode = 'n'
		case '(':
			mode, depth = 's', 1
		case '<':
			if i+1 < end && b[i+1] == '<' {
				containers++
				i++
			} else {
				mode = 'h'
			}
		case '>':
			if i+1 < end && b[i+1] == '>' {
				containers--
				i++
			}
		case '[':
			containers++
		case ']':
			containers--
		default:
			if !contextutil.PDFWhitespace(rune(ch)) {
				mode = 'r'
			}
		}
	}
	if containers != 0 {
		return 'u', nil
	}
	return mode, c.Err()
}

func auditObjectStreamCut(c context.Context, b []byte, start, end int) error {
	if start < 0 || end < start || end > len(b) {
		contextutil.ParseProvenanceFromContext(c).MarkRepair()
		return nil
	}
	mode, err := lexicalCutState(c, b, start, end)
	if err != nil {
		return err
	}
	valid := mode == 0
	if mode == 'c' {
		valid = end == len(b) || b[end] == '\r' || b[end] == '\n'
	}
	if mode == 'r' || mode == 'n' {
		valid = end == len(b) || contextutil.PDFTokenBoundary(string(b[end:end+1]), 0)
	}
	if !valid {
		contextutil.ParseProvenanceFromContext(c).MarkRepair()
	}
	return nil
}

// objectStreamOffset checks PDF-controlled offsets before conversion or slicing.
func objectStreamOffset(relative, first int) (int, error) {
	if relative < 0 || first < 0 {
		return 0, errCorruptObjectStreamDict
	}
	if relative > int(^uint(0)>>1)-first {
		return 0, fmt.Errorf("object stream offset overflow: %w", filter.ErrDecodeLimitExceeded)
	}
	return relative + first, nil
}

// Private partial decodes must enforce the budget themselves: a positive-length
// Flate request does not apply the filter's full-decode limit.
func decodeObjectStreamPrefix(osd *types.ObjectStreamDict, offset, lookahead int, budget int64) ([]byte, error) {
	length, err := objectStreamOffset(lookahead, offset)
	if err != nil {
		return nil, err
	}
	if budget == 0 {
		budget = filter.DefaultMaxDecodeBytes
	}
	if budget > 0 && int64(length) > budget {
		return nil, fmt.Errorf("object stream prefix length %d exceeds limit %d: %w", length, budget, filter.ErrDecodeLimitExceeded)
	}
	return osd.DecodeLengthWithLimit(int64(length), budget)
}

func buildObjectArrayForObjectStream(
	c context.Context,
	osd *types.ObjectStreamDict,
	objs []string,
	fullContent bool,
	decodedContent []byte, maxDepth int) (types.Array, error) {
	var objArray types.Array
	var offsetOld int
	provenance := contextutil.ParseProvenanceFromContext(c)
	decode := func(c context.Context, s string) (types.Object, error) {
		// Structural lookups can use a different Go context. Retain this
		// compressed origin's read-local provenance without replacing it.
		return compressedObject(contextutil.WithParseProvenance(c, provenance), s, maxDepth)
	}

	decodeAt := func(start, end int) types.DecodeLazyObjectStreamObjectFunc {
		return func(c context.Context, s string) (types.Object, error) {
			// Member-index EOF must not manufacture a lexical boundary.
			// Lazy materialization has already decoded the actual container.
			c = contextutil.WithParseProvenance(c, provenance)
			if err := auditObjectStreamCut(c, osd.Content, start, end); err != nil {
				return nil, err
			}
			return decode(c, s)
		}
	}

	for i := 0; i < len(objs); i += 2 {

		if err := c.Err(); err != nil {
			return nil, err
		}

		offset, err := strconv.Atoi(objs[i+1])
		if err != nil {
			return nil, fmt.Errorf("object stream dict offset %d: %w", i/2, err)
		}

		offset, err = objectStreamOffset(offset, osd.FirstObjOffset)
		if err != nil {
			return nil, err
		}
		if fullContent && offset > len(decodedContent) {
			return nil, errCorruptObjectStreamDict
		}
		if i > 0 && offset < offsetOld {
			return nil, errCorruptObjectStreamDict
		}

		if i == 0 && offset > osd.FirstObjOffset {
			prefixContent := decodedContent
			if !fullContent {
				prefixContent, err = decodeObjectStreamPrefix(osd, offset, 1, osd.MaxDecodeBytes)
				if err != nil {
					if terminalHintError(err) || c.Err() != nil {
						return nil, err
					}
					provenance.MarkRepair()
				}
			}
			if len(prefixContent) >= offset {
				if err := auditObjectStreamCut(c, prefixContent, osd.FirstObjOffset, offset); err != nil {
					return nil, err
				}
				markObjectRemainder(c, string(prefixContent[osd.FirstObjOffset:offset]))
			} else {
				provenance.MarkRepair()
			}
		}
		if i > 0 && offset == offsetOld {
			provenance.MarkRepair()
		}

		if i > 0 {
			o := types.NewLazyObjectStreamObject(osd, offsetOld, offset, decodeAt(offsetOld, offset))
			objArray = append(objArray, o)
		}

		if i == len(objs)-2 {
			o := types.NewLazyObjectStreamObject(osd, offset, -1, decode)
			objArray = append(objArray, o)
		}

		offsetOld = offset
	}

	return objArray, nil
}

// Parse all objects of an object stream and save them into objectStreamDict.ObjArray.
func parseObjectStream(c context.Context, osd *types.ObjectStreamDict, limits model.ResourceLimits, auditIndex ...func([]string) error) error {
	if log.ReadEnabled() {
		log.Read.Printf("parseObjectStream begin: decoding %d objects.\n", osd.ObjCount)
	}

	decodedContent := osd.Content
	fullContent := decodedContent != nil
	if decodedContent == nil {
		// The actual content will be decoded lazily, only decode the prolog here.
		var err error
		decodedContent, err = decodeObjectStreamPrefix(osd, osd.FirstObjOffset, 0, limits.MaxDecodeBytes)
		if err != nil {
			return fmt.Errorf("decode object stream prolog: %w", err)
		}
	}
	if osd.FirstObjOffset > len(decodedContent) {
		return errCorruptObjectStreamDict
	}
	cutContent := decodedContent
	if !fullContent {
		var err error
		cutContent, err = decodeObjectStreamPrefix(osd, osd.FirstObjOffset, 1, limits.MaxDecodeBytes)
		if err != nil {
			if terminalHintError(err) || c.Err() != nil {
				return err
			}
			contextutil.ParseProvenanceFromContext(c).MarkRepair()
		}
	}
	if err := auditObjectStreamCut(c, cutContent, 0, osd.FirstObjOffset); err != nil {
		return err
	}
	prolog := decodedContent[:osd.FirstObjOffset]

	// Remove inline comment.
	if i := bytes.Index(prolog, []byte("%")); i != -1 {
		// Preserve the historical truncated index, but audit all discarded
		// lines: a PDF comment ends at EOL, not at the end of the prolog.
		markObjectRemainder(c, string(prolog[i:]))
		prolog = prolog[:i]
	}

	for _, r := range string(prolog) {
		if unicode.IsSpace(r) && !strings.ContainsRune("\x00\t\n\f\r ", r) {
			contextutil.ParseProvenanceFromContext(c).MarkRepair()
		}
	}

	// The separator used in the prolog shall be white space
	// but some PDF writers use 0x00.
	prolog = bytes.ReplaceAll(prolog, []byte{0x00}, []byte{0x20})

	objs := strings.Fields(string(prolog))
	if len(objs)%2 > 0 {
		return errCorruptObjectStreamDict
	}
	entryCount := len(objs) / 2
	if entryCount > limits.MaxObjectStreamCount {
		return fmt.Errorf(
			"object stream object count %d exceeds limit %d: %w",
			entryCount,
			limits.MaxObjectStreamCount,
			errObjectStreamObjectCountLimit,
		)
	}
	if entryCount != osd.ObjCount {
		return fmt.Errorf(
			"object stream index contains %d entries, expected %d: %w",
			entryCount,
			osd.ObjCount,
			errCorruptObjectStreamDict,
		)
	}

	seen := types.IntSet{}
	for i := 0; i < len(objs); i += 2 {
		nr, err := strconv.Atoi(objs[i])
		if err != nil || nr <= 0 || seen[nr] || strings.Trim(objs[i], "0123456789") != "" || strings.Trim(objs[i+1], "0123456789") != "" {
			contextutil.ParseProvenanceFromContext(c).MarkRepair()
		}
		seen[nr] = true
	}
	for _, audit := range auditIndex {
		if err := audit(objs); err != nil {
			return err
		}
	}

	// e.g., 10 0 11 25 = 2 Objects: #10 @ offset 0, #11 @ offset 25

	objArray, err := buildObjectArrayForObjectStream(c, osd, objs, fullContent, decodedContent, limits.MaxRecursionDepth)
	if err != nil {
		return fmt.Errorf("build object stream object array: %w", err)
	}

	osd.ObjArray = objArray

	if log.ReadEnabled() {
		log.Read.Println("parseObjectStream end")
	}

	return nil
}

func createXRefTableEntryFromXRefStream(entryType int64, objNr int, c2, c3, offExtra int64, objStreams types.IntSet, incr int) model.XRefTableEntry {
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
				Generation: &g,
				Incr:       incr}

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
				Generation: &g,
				Incr:       incr}

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
				ObjectStreamInd: &objIndex,
				Incr:            incr}

		objStreams[objNumberRef] = true
	}

	return xRefTableEntry
}

func xRefStreamEntryLen(w [3]int) (int, error) {
	n, err := safemath.AddInt(w[0], w[1])
	if err != nil {
		return 0, errInvalidXRefStreamWArray
	}
	n, err = safemath.AddInt(n, w[2])
	if err != nil || n == 0 {
		return 0, errInvalidXRefStreamWArray
	}
	return n, nil
}

// For each object embedded in this xRefStream create the corresponding xRef table entry.
func extractXRefTableEntriesFromXRefStream(buf []byte, offExtra int64, xsd *types.XRefStreamDict, ctx *model.Context, incr int) error {
	if log.ReadEnabled() {
		log.Read.Printf("extractXRefTableEntriesFromXRefStream begin")
	}

	// Note:
	// A value of zero for an element in the W array indicates that the corresponding field shall not be present in the
	// stream,
	// and the default value shall be used, if there is one.
	// If the first element is zero, the type field shall not be present, and shall default to type 1.

	i1 := xsd.W[0]
	i2 := xsd.W[1]
	i3 := xsd.W[2]

	xrefEntryLen, err := xRefStreamEntryLen(xsd.W)
	if err != nil {
		return err
	}
	if log.ReadEnabled() {
		log.Read.Printf("extractXRefTableEntriesFromXRefStream: begin xrefEntryLen = %d\n", xrefEntryLen)
	}

	if len(buf)%xrefEntryLen > 0 {
		return errCorruptXRefStream
	}

	objCount := len(xsd.Objects)
	if log.ReadEnabled() {
		log.Read.Printf("extractXRefTableEntriesFromXRefStream: objCount:%d %v\n", objCount, xsd.Objects)
		log.Read.Printf(
			"extractXRefTableEntriesFromXRefStream: len(buf):%d availableEntries:%d\n",
			len(buf), len(buf)/xrefEntryLen,
		)
	}
	if objCount > len(buf)/xrefEntryLen {
		return errCorruptXRefStream
	}
	if objCount < len(buf)/xrefEntryLen {
		// Preserve normal acceptance, but omitted origins cannot authorize
		// hint compatibility or deletion.
		ctx.Read.RepairedXRef = true
	}

	j := 0

	// bufToInt64 interprets the content of buf as an int64.
	bufToInt64 := func(buf []byte) (i int64) {
		// Preserve historical low-word conversion, but unsigned xref
		// values outside int64 cannot supply trustworthy live identities.
		for j, b := range buf {
			if (j < len(buf)-8 && b != 0) || (j == len(buf)-8 && b&0x80 != 0) {
				ctx.Read.RepairedXRef = true
			}
		}
		for _, b := range buf {
			i <<= 8
			i |= int64(b)
		}
		return
	}

	seen := types.IntSet{}
	for i := 0; i < len(buf) && j < len(xsd.Objects); i += xrefEntryLen {

		objNr := xsd.Objects[j]
		if seen[objNr] {
			ctx.Read.RepairedXRef = true
		}
		seen[objNr] = true

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

		// These unsigned fields are narrowed to native int by entry creation.
		// Audit before revision precedence, including ignored older origins.
		if ((c1 == 0 || c1 == 1 || c1 == 2) && int64(int(c3)) != c3) ||
			(c1 == 2 && int64(int(c2)) != c2) {
			ctx.Read.RepairedXRef = true
		}

		// Audit even older entries that revision precedence discards.
		if ctx.XRefTable.Exists(objNr) {
			if log.ReadEnabled() {
				log.Read.Printf("Skip entry %d - already assigned\n", objNr)
			}
			j++
			continue
		}

		entry := createXRefTableEntryFromXRefStream(c1, objNr, c2, c3, offExtra, ctx.Read.ObjectStreams, incr)
		ctx.Table[objNr] = &entry
		j++
	}

	if log.ReadEnabled() {
		log.Read.Println("extractXRefTableEntriesFromXRefStream: end")
	}

	return nil
}

func validateXRefStreamDirectEntries(d types.Dict, objNr int) error {
	for _, key := range []string{"Type", "Size", "Index", "Prev", "W"} {
		if _, ok := d[key].(types.IndirectRef); ok {
			return fmt.Errorf("xref stream obj#%d entry %s: value must be direct", objNr, key)
		}
	}
	if _, encoded := d["Filter"]; !encoded {
		return nil
	}
	for _, key := range []string{"Filter", "DecodeParms"} {
		if _, ok := d[key].(types.IndirectRef); ok {
			return fmt.Errorf("xref stream obj#%d entry %s: value must be direct", objNr, key)
		}
	}
	return nil
}

func xRefStreamDict(c context.Context, ctx *model.Context, o types.Object, objNr int, streamOffset int64) (*types.XRefStreamDict, error) {
	d, ok := o.(types.Dict)
	if !ok {
		return nil, fmt.Errorf("xref stream obj#%d: %w", objNr, errMissingXRefStreamDict)
	}
	if err := validateXRefStreamDirectEntries(d, objNr); err != nil {
		return nil, err
	}

	// Parse attributes for stream object.
	streamLength, streamLengthObjNr := d.Length()
	if streamLength == nil && streamLengthObjNr == nil {
		return nil, fmt.Errorf("xref stream obj#%d: %w", objNr, errMissingXRefStreamLength)
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
		return nil, fmt.Errorf("xRefStreamDict: cannot decode stream for obj#:%d: %w", objNr, err)
	}

	declaredSize := sd.Size()
	var xsd *types.XRefStreamDict
	if ctx.Configuration.ValidationMode == model.ValidationRelaxed {
		xsd, err = model.ParseXRefStreamDictRelaxedWithLimits(&sd, ctx.Configuration.Limits)
	} else {
		xsd, err = model.ParseXRefStreamDictWithLimits(&sd, ctx.Configuration.Limits)
	}
	if err != nil {
		return nil, err
	}
	if declaredSize != nil && *declaredSize != xsd.Size {
		ctx.Read.RepairedXRef = true
		xsd.Dict.Update("Size", types.Integer(xsd.Size))
		model.ShowRepaired(fmt.Sprintf("xref stream obj#%d Size from %d to %d", objNr, *declaredSize, xsd.Size))
	}
	return xsd, nil
}

func processXRefStream(c context.Context, ctx *model.Context, xsd *types.XRefStreamDict, objNr *int, offset *int64, offExtra int64, incr int) (prevOffset *int64, err error) {
	if log.ReadEnabled() {
		log.Read.Println("processXRefStream: begin")
	}

	ctx.Read.TrailerDicts = append(ctx.Read.TrailerDicts, xsd.Dict)
	if err = c.Err(); err != nil {
		return nil, err
	}
	if err = parseTrailer(ctx.XRefTable, xsd.Dict, c); err != nil {
		return nil, err
	}

	// Parse xRefStream and create xRefTable entries for embedded objects.
	if err = extractXRefTableEntriesFromXRefStream(xsd.Content, offExtra, xsd, ctx, incr); err != nil {
		return nil, err
	}

	*offset += offExtra
	if offExtra == 0 {
		contextutil.ParseProvenanceFromContext(c).RecordXRefOrigin(*offset)
	}

	if entry, ok := ctx.Table[*objNr]; ok && entry.Offset != nil && *entry.Offset == *offset {
		entry.Object = *xsd
	}

	prevOffset = xsd.PreviousOffset

	if log.ReadEnabled() {
		log.Read.Println("processXRefStream: end")
	}

	return prevOffset, nil
}

// Parse xRef stream and setup xrefTable entries for all embedded objects and the xref stream dict.
func parseXRefStream(c context.Context, ctx *model.Context, rd io.Reader, offset *int64, offExtra int64, incr int) (prevOffset *int64, err error) {
	if log.ReadEnabled() {
		log.Read.Printf("parseXRefStream: begin at offset %d\n", *offset)
	}

	buf, endInd, streamInd, streamOffset, err := buffer(c, rd, objectBufferLimit(ctx))
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
		return nil, fmt.Errorf("xref stream object: %w", errCorruptXRefStream)
	}

	// Init object parse buf.
	l := line[:streamInd]

	objNr, genNr, err := model.ParseObjectAttributes(&l)
	if err != nil {
		return nil, err
	}

	if !literalObjectHeader([]byte(line[:min(128, streamInd)]), *objNr, *genNr) {
		ctx.Read.RepairedXRef = true
	}

	// parse this object
	if log.ReadEnabled() {
		log.Read.Printf("parseXRefStream: xrefstm obj#:%d gen:%d\n", *objNr, *genNr)
		log.Read.Printf("parseXRefStream: dereferencing object %d\n", *objNr)
	}

	o, err := model.ParseObject(c, &l, 0, recursionLimit(ctx))
	if err != nil {
		return nil, fmt.Errorf("parseXRefStream: no object: %w", err)
	}

	if log.ReadEnabled() {
		log.Read.Printf("parseXRefStream: we have an object: %s\n", o)
	}

	markObjectRemainder(c, l)
	streamOffset += *offset
	xsd, err := xRefStreamDict(c, ctx, o, *objNr, streamOffset)
	if err != nil {
		return nil, err
	}

	return processXRefStream(c, ctx, xsd, objNr, offset, offExtra, incr)
}

// Parse an xRefStream for a hybrid PDF file.
func parseHybridXRefStream(c context.Context, ctx *model.Context, offset *int64, offExtra int64, incr int) error {
	if log.ReadEnabled() {
		log.Read.Println("parseHybridXRefStream: begin")
	}

	rd, err := newPositionedReader(ctx.Read.RS, offset)
	if err != nil {
		return err
	}

	if _, err = parseXRefStream(c, ctx, rd, offset, offExtra, incr); err != nil {
		return err
	}

	if log.ReadEnabled() {
		log.Read.Println("parseHybridXRefStream: end")
	}

	return nil
}

func parseTrailerSize(xRefTable *model.XRefTable, d types.Dict) error {
	i, err := directTrailerIntegerEntry(d, "Size")
	if err != nil {
		return err
	}
	if i == nil {
		return errMissingTrailerSize
	}
	// Not reliable!
	// Patched after all read in.
	v := i.Value()
	if v <= 0 {
		return fmt.Errorf("invalid trailer Size %d: %w", v, errCorruptTrailerDict)
	}
	limit := model.DefaultResourceLimits().MaxObjectCount
	if xRefTable.Conf != nil {
		limit = xRefTable.Conf.Limits.MaxObjectCount
	}
	if v > limit {
		return fmt.Errorf("trailer Size %d exceeds limit %d: %w", v, limit, errCorruptTrailerDict)
	}
	xRefTable.Size = &v
	return nil
}

func directTrailerIntegerEntry(d types.Dict, key string) (*types.Integer, error) {
	o, found := d.Find(key)
	if !found || o == nil {
		return nil, nil
	}
	i, ok := o.(types.Integer)
	if !ok {
		return nil, fmt.Errorf("%w: trailer entry %s must be a direct integer", errCorruptTrailerDict, key)
	}
	return &i, nil
}

func parseTrailerRoot(xRefTable *model.XRefTable, d types.Dict) error {
	indRef := d.IndirectRefEntry("Root")
	if indRef == nil {
		return errMissingTrailerRoot
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
		return nil
	}

	infoDict := d.DictEntry("Info")
	if infoDict == nil {
		return nil
	}
	xRefTable.DirectInfoDict = infoDict
	if log.ReadEnabled() {
		log.Read.Printf("parseTrailerInfo: direct Info dictionary pending PDF version identification\n")
	}
	return nil
}

func materializeDirectInfoDict(xRefTable *model.XRefTable) error {
	if xRefTable.DirectInfoDict == nil {
		return nil
	}
	if !xRefTable.PDF20() {
		xRefTable.DirectInfoDict = nil
		return nil
	}
	if xRefTable.Size == nil {
		return errMissingTrailerSize
	}
	if xRefTable.Table == nil {
		xRefTable.Table = map[int]*model.XRefTableEntry{}
	}

	objNr := *xRefTable.Size
	for {
		if _, found := xRefTable.Table[objNr]; !found {
			break
		}
		objNr++
	}
	entry := model.NewXRefTableEntryGen0(xRefTable.DirectInfoDict)
	entry.RefCount = 1
	xRefTable.Table[objNr] = entry
	*xRefTable.Size = objNr + 1
	if objNr > xRefTable.MaxObjNr {
		xRefTable.MaxObjNr = objNr
	}
	xRefTable.Info = types.NewIndirectRef(objNr, 0)
	xRefTable.DirectInfoDict = nil
	if log.ReadEnabled() {
		log.Read.Printf("materializeDirectInfoDict: PDF 2.0 Info dictionary: %s\n", *xRefTable.Info)
	}
	return nil
}

func parseTrailerID(xRefTable *model.XRefTable, d types.Dict, contexts ...context.Context) error {
	arr := d.ArrayEntry("ID")
	if arr != nil {
		if len(arr) != 2 {
			if xRefTable.ValidationMode == model.ValidationStrict {
				return errInvalidTrailerID
			}
			if len(arr) != 1 {
				return errInvalidTrailerID
			}
			if len(contexts) > 0 {
				contextutil.ParseProvenanceFromContext(contexts[0]).MarkRepair()
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

	if xRefTable.Encrypt == nil {
		return nil
	}

	if xRefTable.ValidationMode == model.ValidationStrict {
		return errMissingTrailerID
	}

	return nil
}

// Parse trailer dict and return any offset of a previous xref section.
func parseTrailer(xRefTable *model.XRefTable, d types.Dict, contexts ...context.Context) error {
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

	if xRefTable.Info == nil && xRefTable.DirectInfoDict == nil {
		if err := parseTrailerInfo(xRefTable, d); err != nil {
			return err
		}
	}

	if xRefTable.ID == nil {
		if err := parseTrailerID(xRefTable, d, contexts...); err != nil {
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

func offsetPrev(ctx *model.Context, trailerDict types.Dict, offCurXRef *int64) (*int64, error) {
	i, err := directTrailerIntegerEntry(trailerDict, "Prev")
	if err != nil {
		return nil, err
	}
	var offset *int64
	if i != nil {
		v := int64(i.Value())
		offset = &v
	}
	if offset != nil {
		if log.ReadEnabled() {
			log.Read.Printf("offsetPrev: previous xref table section offset:%d\n", *offset)
		}
		if *offset == 0 {
			if ctx.Read != nil {
				ctx.Read.RepairedXRef = true
			}
			offset = nil
			if offCurXRef != nil {
				if off := scanForPreviousXref(ctx, offCurXRef); off != nil {
					offset = off
				}
			}
		}
	}
	return offset, nil
}

func xrefStreamOffset(c context.Context, ctx *model.Context, trailerDict types.Dict, repairing bool) (*int64, error) {
	if repairing {
		return nil, nil
	}
	i, _, err := bootstrapIntegerEntry(c, ctx, trailerDict, "XRefStm")
	if err != nil {
		return nil, fmt.Errorf("parse trailer XRefStm: %w", err)
	}
	if i == nil {
		return nil, nil
	}
	v := int64(i.Value())
	return &v, nil
}

func parseTrailerDict(c context.Context, ctx *model.Context, trailerDict types.Dict, offCurXRef *int64, offExtra int64, incr int, repairing bool) (*int64, error) {
	if log.ReadEnabled() {
		log.Read.Println("parseTrailerDict begin")
	}

	xRefTable := ctx.XRefTable

	ctx.Read.TrailerDicts = append(ctx.Read.TrailerDicts, trailerDict)
	if err := parseTrailer(xRefTable, trailerDict, c); err != nil {
		return nil, err
	}
	if !repairing && offExtra == 0 && offCurXRef != nil {
		contextutil.ParseProvenanceFromContext(c).RecordXRefOrigin(*offCurXRef)
	}

	handleAdditionalStreams(trailerDict, xRefTable)

	offset, err := offsetPrev(ctx, trailerDict, offCurXRef)
	if err != nil {
		return nil, err
	}

	offsetXRefStream, err := xrefStreamOffset(c, ctx, trailerDict, repairing)
	if err != nil {
		return nil, err
	}
	if offsetXRefStream == nil {
		// No cross reference stream.
		if !ctx.Reader15 && xRefTable.Version() >= model.V15 && !ctx.Read.Hybrid {
			return nil, fmt.Errorf("parseTrailerDict: PDF1.4 conformant reader: found incompatible version: %s", xRefTable.VersionString())
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
		if err := parseHybridXRefStream(c, ctx, offsetXRefStream, offExtra, incr); err != nil {
			if err := bypassXrefSection(c, ctx, offExtra, err, incr); err != nil {
				return nil, err
			}
			return nil, nil
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
		return "", errMissingScannerLine
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

func scanTrailerDictStart(s *bufio.Scanner, line *string, contexts ...context.Context) error {
	l := *line
	var err error
	for {
		i := strings.Index(l, "<<")
		if len(contexts) > 0 {
			prefix := l
			if i >= 0 {
				prefix = l[:i]
			}
			markObjectRemainder(contexts[0], prefix)
			if i >= 0 && strings.Contains(prefix, "%") {
				contextutil.ParseProvenanceFromContext(contexts[0]).MarkRepair()
			}
		}
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

// auditTrailerTerminator checks the selected cut and follows counterfeit/commented cuts.
// Its bounded continuation is provenance-only; the historical returned trailer
// bytes stay unchanged, and unreadable/overlong continuation denies authority.
func auditTrailerTerminator(c context.Context, s *bufio.Scanner, line string, cut int, prefix string) error {
	p := contextutil.ParseProvenanceFromContext(c)
	if p == nil {
		return nil
	}
	mode, err := lexicalCutState(c, []byte(prefix+line), 0, len(prefix)+cut)
	if err != nil {
		return err
	}
	end := cut + len("startxref")
	canonical := mode == 0 && (end == len(line) || contextutil.PDFWhitespace(rune(line[end])))
	if canonical {
		markObjectRemainder(c, line[end:]+"\n")
		// A syntactically terminated marker alone is not a complete epilogue.
		// Audit the discarded offset/EOF lines without changing trailer values.
		for _, offsetLine := range []bool{true, false} {
			if err := c.Err(); err != nil {
				return err
			}
			next, err := scanLine(s)
			if err != nil {
				p.MarkRepair()
				return c.Err()
			}
			if len(next) > 1024 {
				p.MarkRepair()
				return nil
			}
			unreliable := false
			auditXRefWhitespace(next, &unreliable)
			trimmed := strings.TrimSpace(next)
			if unreliable || (offsetLine && (trimmed == "" || strings.Trim(trimmed, "0123456789") != "")) ||
				(!offsetLine && trimmed != "%%EOF") {
				p.MarkRepair()
			}
		}
		return nil
	}
	// A marker wholly inside a comment is harmless if all continuation up
	// to the real marker is also whitespace/comments. Other fake markers
	// discard syntax and must always invalidate provenance.
	if mode != 'c' {
		p.MarkRepair()
	}
	budget := 1024
	for {
		if err := c.Err(); err != nil {
			return err
		}
		next, err := scanLine(s)
		if err != nil {
			p.MarkRepair()
			return c.Err()
		}
		budget -= len(next) + 1
		if budget < 0 {
			p.MarkRepair()
			return nil
		}
		trimmed := strings.TrimLeftFunc(next, contextutil.PDFWhitespace)
		if strings.HasPrefix(trimmed, "startxref") {
			end := len("startxref")
			if end == len(trimmed) || contextutil.PDFWhitespace(rune(trimmed[end])) {
				return auditTrailerTerminator(c, s, next, len(next)-len(trimmed), "")
			}
		}
		markObjectRemainder(c, next+"\n")
	}
}

func scanTrailerDictRemainder(s *bufio.Scanner, line string, buf bytes.Buffer, contexts ...context.Context) (string, error) {
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

	if len(contexts) > 0 {
		if err := auditTrailerTerminator(contexts[0], s, line, i, buf.String()); err != nil {
			return "", err
		}
	}
	line = line[:i]
	if log.ReadEnabled() {
		log.Read.Printf("line: <%s>\n", line)
	}
	buf.WriteString(line[:i])
	buf.WriteString("\x0a")

	return buf.String(), nil
}

func scanTrailer(s *bufio.Scanner, line string, contexts ...context.Context) (string, error) {
	var buf bytes.Buffer
	if log.ReadEnabled() {
		log.Read.Printf("line: <%s>\n", line)
	}

	if err := scanTrailerDictStart(s, &line, contexts...); err != nil {
		return "", err
	}

	return scanTrailerDictRemainder(s, line, buf, contexts...)
}

func processTrailer(c context.Context, ctx *model.Context, s *bufio.Scanner, line string, offCurXRef *int64, offExtra int64, incr int, repairing bool) (*int64, error) {
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

	trailerString, err := scanTrailer(s, trailerString, c)
	if err != nil {
		return nil, err
	}

	if log.ReadEnabled() {
		log.Read.Printf("processTrailer: trailerString: (len:%d) <%s>\n", len(trailerString), trailerString)
	}

	o, err := model.ParseObject(c, &trailerString, 0, recursionLimit(ctx))
	if err != nil {
		return nil, err
	}

	markObjectRemainder(c, trailerString)
	trailerDict, ok := o.(types.Dict)
	if !ok {
		return nil, errCorruptTrailerDict
	}

	if log.ReadEnabled() {
		log.Read.Printf("processTrailer: trailerDict:\n%s\n", trailerDict)
	}

	return parseTrailerDict(c, ctx, trailerDict, offCurXRef, offExtra, incr, repairing)
}

// Parse xRef section into corresponding number of xRef table entries.
func parseXRefSection(c context.Context, ctx *model.Context, s *bufio.Scanner, fields []string, ssCount *int, offCurXRef *int64, offExtra int64, incr int) (*int64, error) {
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

		if !strings.HasPrefix(strings.TrimSpace(line), "trailer") {
			auditXRefWhitespace(line, &ctx.Read.RepairedXRef)
		}
		fields = strings.Fields(line)
	}

	line = strings.TrimLeft(line, " ")
	if !strings.HasPrefix(line, "trailer") {
		// Process all sub sections of this xRef section.
		var unreliable []*bool
		if ctx.Read != nil {
			unreliable = []*bool{&ctx.Read.RepairedXRef}
		}
		if line, err = parseXRefTableSubSection(ctx.XRefTable, ctx.Configuration.Limits, s, fields, offExtra, incr, unreliable...); err != nil {
			return nil, err
		}
		*ssCount++
		line = strings.TrimLeft(line, " ")
	}

	if log.ReadEnabled() {
		log.Read.Println("parseXRefSection: All subsections read!")
	}

	if !strings.HasPrefix(line, "trailer") {
		return nil, fmt.Errorf("xrefsection: %w, line = <%s>", errMissingTrailerDict, line)
	}

	if log.ReadEnabled() {
		log.Read.Println("parseXRefSection: parsing trailer dict..")
	}

	return processTrailer(c, ctx, s, line, offCurXRef, offExtra, incr, false)
}

func scanForVersion(rs io.ReadSeeker, prefix string) ([]byte, int, error) {
	bufSize := 100

	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return nil, 0, fmt.Errorf("seek header: %w", err)
	}

	buf := make([]byte, bufSize)
	var curBuf []byte

	off := 0
	found := false
	var buf2 []byte

	for !found {
		n, err := fillBuffer(rs, buf)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return nil, 0, fmt.Errorf("read header: %w", err)
			}
			return nil, 0, ErrCorruptHeader
		}
		curBuf = buf[:n]
		if off == 0 && hasPostScriptHeader(curBuf) {
			return nil, 0, ErrPostScriptInput
		}
		for {
			i := bytes.IndexByte(curBuf, '%')
			if i < 0 {
				// no match, check next block
				off += len(curBuf)
				break
			}

			// Check all occurrences
			if i < len(curBuf)-18 {
				if !bytes.HasPrefix(curBuf[i:], []byte(prefix)) {
					// No match, keep checking
					off += i + 1
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
				if !errors.Is(err, io.EOF) {
					return nil, 0, fmt.Errorf("read header: %w", err)
				}
				return nil, 0, ErrCorruptHeader
			}
			buf3 := append(curBuf[i:], buf2[:n]...)
			if !bytes.HasPrefix(buf3, []byte(prefix)) {
				// No match, keep checking
				off += len(curBuf)
				curBuf = buf2
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
		return nil, 0, 0, fmt.Errorf("unknown PDF header version: %w", err)
	}

	s = s[8:]
	s = bytes.TrimLeft(s, "\t\f ")

	// Detect the used eol which should be 1 (0x00, 0x0D) or 2 chars (0x0D0A)long.
	// %PDF-1.x{whiteSpace}{text}{eol} or
	j := bytes.IndexAny(s, "\x0A\x0D")
	if j < 0 {
		return nil, 0, 0, ErrCorruptHeader
	}

	switch s[j] {
	case 0x0A:
		eolCount = 1
	case 0x0D:
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

func parseAndLoad(c context.Context, ctx *model.Context, line string, offset *int64, incr int, offsetPrev *int64) (int, error) {
	l := line
	objNr, generation, err := model.ParseObjectAttributes(&l)
	if err != nil {
		return 0, err
	}

	if *objNr == 0 {
		return 0, nil
	}

	off := *offset

	obj, endInd, streamInd, streamOffset, err := object(c, ctx, off, *objNr, *generation)
	if err != nil {
		return endInd, err
	}
	obj, err = resolveObject(c, ctx, obj, off, *objNr, *generation, endInd, streamInd, streamOffset)
	if err != nil {
		return endInd, err
	}

	if d, ok := obj.(types.Dict); ok {
		catalog, err := isCatalogForRepair(c, ctx, d)
		if err != nil {
			return endInd, fmt.Errorf("repair catalog candidate obj#%d Type: %w", *objNr, err)
		}
		if catalog {
			ctx.RootDict = d
			ctx.Root = types.NewIndirectRef(*objNr, *generation)
			model.ShowRepaired("catalog")
		}
	}

	*offset += int64(len(line))

	sd, ok := obj.(types.StreamDict)
	if ok {
		if err = loadStreamDict(c, ctx, &sd, *objNr, *generation, true); err != nil {
			return endInd, err
		}
		obj = sd
		*offset = sd.StreamOffset + *sd.StreamLength
	}

	e, found := ctx.Table[*objNr]
	if !found {
		entry := model.XRefTableEntry{
			Free:       false,
			Generation: generation,
			Offset:     &off,
			Object:     obj,
			Incr:       incr,
		}
		ctx.Table[*objNr] = &entry
		return endInd, nil
	}

	e.Offset = &off
	e.Object = obj
	e.Incr = incr

	return endInd, nil
}

func isCatalogForRepair(c context.Context, ctx *model.Context, d types.Dict) (bool, error) {
	o, found := d.Find("Type")
	if !found || o == nil {
		return false, nil
	}

	if ir, ok := o.(types.IndirectRef); ok {
		entry, found := ctx.FindTableEntryForIndRef(&ir)
		if !found || entry == nil || entry.Free || entry.Compressed {
			return false, nil
		}
		if entry.Object == nil && entry.Offset != nil {
			if _, err := dereferencedObject(c, ctx, ir.ObjectNumber.Value()); err != nil {
				if errors.Is(err, errNilDereferencedObject) {
					return false, nil
				}
				return false, model.WithValidationErrorObject(err, ir.ObjectNumber.Value())
			}
		}
	} else if _, ok := o.(types.Name); !ok {
		return false, nil
	}

	t, _, err := ctx.DereferenceNameEntry(d, "Type")
	if err != nil {
		return false, err
	}
	return t != nil && t.Value() == "Catalog", nil
}

func processObject(c context.Context, ctx *model.Context, line string, offset *int64, incr int, offsetPrev *int64) (*bufio.Scanner, error) {
	endInd, err := parseAndLoad(c, ctx, line, offset, incr, offsetPrev)
	if err != nil {
		if ctx.XRefTable.ValidationMode != model.ValidationRelaxed || endInd < 0 {
			return nil, err
		}
		model.ShowSkipped(fmt.Sprintf("malformed object at offset %d: %v", *offset, err))
		*offset += int64(endInd + len("endobj"))
	}
	rd, err := newPositionedReader(ctx.Read.RS, offset)
	if err != nil {
		return nil, err
	}
	s := bufio.NewScanner(rd)
	s.Buffer(make([]byte, 0, defaultBufSize), maxObjectBufferLen)
	s.Split(scan.LinesSingleEOL)
	return s, nil
}

func objCandidate(withinObj bool, line string) bool {
	if withinObj {
		return false
	}
	i := strings.Index(line, "obj")
	return i > 2 && strings.Index(line, "endobj") != i-3
}

func checkEndObj(withinObj *bool, line *string) {
	if *withinObj {
		i := strings.Index(*line, "endobj")
		if i >= 0 {
			*line = (*line)[i:]
			*withinObj = false
		}
	}
}

func ensureNoStartXRef(line string, i int) bool {
	return i == 0 || i > 0 && line[i-1] != 't'
}

func registerXRefEntry(ctx *model.Context, line string, offset int64, incr int) bool {
	objNr, generation, err := model.ParseObjectAttributes(&line)
	if err != nil || *objNr == 0 {
		return false
	}

	ctx.Table[*objNr] = &model.XRefTableEntry{
		Generation: generation,
		Offset:     &offset,
		Incr:       incr,
	}
	return true
}

func registerXRefObjects(ctx *model.Context, incr int) error {
	var offset int64
	rd, err := newPositionedReader(ctx.Read.RS, &offset)
	if err != nil {
		return err
	}

	s := bufio.NewScanner(rd)
	s.Buffer(make([]byte, 0, defaultBufSize), maxObjectBufferLen)
	s.Split(scan.LinesSingleEOL)

	withinObj := false
	for s.Scan() {
		rawLine := s.Text()
		line := types.TrimLeadingComment(rawLine)
		checkEndObj(&withinObj, &line)
		if objCandidate(withinObj, line) && registerXRefEntry(ctx, line, offset, incr) {
			withinObj = !strings.Contains(line, "endobj")
		}
		offset += int64(len(rawLine) + 1)
	}
	if err := s.Err(); err != nil {
		return fmt.Errorf("repair xref table: register objects: %w", err)
	}
	return nil
}

type xrefRepairState struct {
	scanner       *bufio.Scanner
	offset        int64
	withinObj     bool
	withinXref    bool
	withinTrailer bool
	bb            []byte
	offsetPrev    *int64
}

func processXRefRepairLine(c context.Context, ctx *model.Context, state *xrefRepairState, line string, length, eolCount int, offExtra int64, incr int) (bool, error) {
	if !state.withinXref {
		return false, nil
	}

	state.offset += int64(length + eolCount)
	if !state.withinTrailer {
		if strings.Contains(line, "trailer") {
			state.bb = append([]byte{}, line...)
			state.withinTrailer = true
		}
		return true, nil
	}
	if length == 0 {
		return true, nil
	}

	state.bb = append(state.bb, '\n')
	state.bb = append(state.bb, line...)
	if !strings.HasPrefix(line, "startxref") {
		return true, nil
	}

	offsetPrev, err := processTrailer(c, ctx, state.scanner, string(state.bb), nil, offExtra, incr, true)
	if err != nil {
		return false, err
	}
	state.offsetPrev = offsetPrev
	state.withinXref = false
	state.withinTrailer = false
	model.ShowRepaired("xreftable")
	return true, nil
}

func discardPreEncryptionObjects(ctx *model.Context) {
	if ctx.Encrypt == nil || ctx.EncKey != nil {
		return
	}

	for _, entry := range ctx.Table {
		if entry == nil || entry.Free || entry.Compressed || entry.Offset == nil {
			continue
		}
		entry.Object = nil
	}

	ctx.RootDict = nil
	ctx.Read.BinaryTotalSize = 0
}

// bypassXrefSection is a fix for digesting corrupt xref sections.
// It populates the xRefTable by reading in all indirect objects line by line
// and works on the assumption of a single xref section - meaning no incremental updates.
func bypassXrefSection(c context.Context, ctx *model.Context, offExtra int64, wasErr error, incr int) error {
	ctx.Read.RepairedXRef = true
	ctx.Table = make(map[int]*model.XRefTableEntry)

	var z int64
	g := types.FreeHeadGeneration
	ctx.Table[0] = &model.XRefTableEntry{
		Free:       true,
		Offset:     &z,
		Generation: &g}
	if err := registerXRefObjects(ctx, incr); err != nil {
		return err
	}

	rs := ctx.Read.RS
	var offset int64

	rd, err := newPositionedReader(rs, &offset)
	if err != nil {
		return err
	}

	s := bufio.NewScanner(rd)
	s.Buffer(make([]byte, 0, defaultBufSize), maxObjectBufferLen)
	s.Split(scan.LinesSingleEOL)
	eolCount := 1

	state := xrefRepairState{scanner: s}
	var prevLine string

	for {
		line, err := scanLineRaw(state.scanner)
		if err != nil {
			if errors.Is(err, errMissingScannerLine) {
				break
			}
			return fmt.Errorf("repair xref table: scan input: %w", err)
		}
		length := len(line)
		line = types.TrimLeadingComment(line)
		if len(prevLine) > 0 {
			line = prevLine + line
			prevLine = ""
		}
		handled, err := processXRefRepairLine(c, ctx, &state, line, length, eolCount, offExtra, incr)
		if err != nil {
			return err
		}
		if handled {
			continue
		}
		i := strings.Index(line, "xref")
		if ensureNoStartXRef(line, i) {
			state.offset += int64(length + eolCount)
			state.withinXref = true
			continue
		}
		checkEndObj(&state.withinObj, &line)
		if objCandidate(state.withinObj, line) {
			if !strings.HasSuffix(line, "obj") {
				state.withinObj = true
				state.scanner, err = processObject(c, ctx, line, &state.offset, incr, state.offsetPrev)
				if err != nil {
					return err
				}
				continue
			}
			prevLine = line
			continue
		}

		state.offset += int64(length + eolCount)
		continue
	}
	discardPreEncryptionObjects(ctx)
	return nil
}

func postProcess(ctx *model.Context, xrefSectionCount int) {
	// Ensure free object #0 if exactly one xref subsection
	// and in one of the following weird situations:
	if xrefSectionCount == 1 && !ctx.Exists(0) {
		if ctx.Read != nil {
			ctx.Read.RepairedXRef = true
		}
		// Fix for #250
		e := ctx.Table[1]
		if e != nil && e.Free {
			// Create free object 0 by shifting down parsed objects by one.
			for _, objNr := range slices.Sorted(maps.Keys(ctx.Table)) {
				if objNr == 0 {
					continue
				}
				entry := ctx.Table[objNr]
				delete(ctx.Table, objNr)
				if entry != nil {
					ctx.Table[objNr-1] = entry
				}
			}
		} else {
			// Create free object 0 from scratch if the free list head is missing.
			g0 := types.FreeHeadGeneration
			ctx.Table[0] = &model.XRefTableEntry{Free: true, Offset: &zero, Generation: &g0}
		}
		model.ShowRepaired("obj#0")
	}
}

func tryXRefSection(c context.Context, ctx *model.Context, rs io.ReadSeeker, offset *int64, offExtra int64, xrefSectionCount *int, incr int) (*int64, error) {
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
		auditXRefWhitespace(line, &ctx.Read.RepairedXRef)
		if log.ReadEnabled() {
			log.Read.Println("tryXRefSection: found xref section")
		}
		return parseXRefSection(c, ctx, s, nil, xrefSectionCount, offset, offExtra, incr)
	}

	// Repair fix for #823
	if strings.HasPrefix(line, "xref") {
		fields := strings.Fields(line)
		if len(fields) == 3 {
			ctx.Read.RepairedXRef = true
			return parseXRefSection(c, ctx, s, fields[1:], xrefSectionCount, offset, offExtra, incr)
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
	if i == 0 || (i > 0 && line[i-1] != 't') { // Don't confuse with "startxref".
		if log.ReadEnabled() {
			log.Read.Println("tryXRefSection: found xref section")
		}
		repairOff += i
		if log.ReadEnabled() {
			log.Read.Printf("Repair offset: %d\n", repairOff)
		}
		ctx.Read.RepairedXRef = true
		ctx.Read.RepairOffset = int64(repairOff)
		return parseXRefSection(c, ctx, s, nil, xrefSectionCount, offset, offExtra, incr)
	}

	return &zero, nil
}

func parseXRefStreamOrRepair(c context.Context, ctx *model.Context, rs io.ReadSeeker, offset *int64, offExtra int64, incr int) (*int64, bool, error) {
	if log.ReadEnabled() {
		log.Read.Println("buildXRefTableStartingAt: found xref stream")
	}
	ctx.Read.UsingXRefStreams = true
	rd, err := newPositionedReader(rs, offset)
	if err != nil {
		return nil, false, err
	}

	next, err := parseXRefStream(c, ctx, rd, offset, offExtra, incr)
	if err == nil {
		return next, false, nil
	}
	if ctx.Configuration.ValidationMode == model.ValidationStrict && errors.Is(err, model.ErrXRefStreamIndexSizeMismatch) {
		return nil, false, err
	}

	// Try fix for corrupt single xref section.
	return nil, true, bypassXrefSection(c, ctx, offExtra, err, incr)
}

func repairCorruptXRefSection(c context.Context, ctx *model.Context, offExtra int64, err error, incr int) error {
	if ctx.Configuration.ValidationMode != model.ValidationRelaxed || incr != 1 || !errors.Is(err, errMissingTrailerDict) {
		return err
	}
	return bypassXrefSection(c, ctx, offExtra, err, incr)
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
	incr := 0

	for offset != nil {

		incr++
		//fmt.Printf("Incr: %d\n", incr)

		if err := c.Err(); err != nil {
			return err
		}

		if offs[*offset] {
			ctx.Read.RepairedXRef = true
			if offset, err = offsetLastXRefSection(ctx, ctx.Read.FileSize-*offset); err != nil {
				return err
			}
			if offs[*offset] {
				return nil
			}
		}

		offs[*offset] = true

		off, err := tryXRefSection(c, ctx, rs, offset, offExtra, &xrefSectionCount, incr)
		if err != nil {
			return repairCorruptXRefSection(c, ctx, offExtra, err, incr)
		}

		if off == nil || *off != 0 {
			offset = off
			continue
		}

		repaired := false
		offset, repaired, err = parseXRefStreamOrRepair(c, ctx, rs, offset, offExtra, incr)
		if err != nil {
			return err
		}
		if repaired {
			return nil
		}
	}

	postProcess(ctx, xrefSectionCount)

	if log.ReadEnabled() {
		log.Read.Println("buildXRefTableStartingAt: end")
	}

	return nil
}

type freeHeadIdentity struct {
	free, compressed bool
	offset           int64
	generation       int
	objectStream     int
	objectStreamInd  int
}

func freeHeadEntryIdentity(table map[int]*model.XRefTableEntry, nr int) (freeHeadIdentity, bool) {
	e := table[nr]
	if nr < 0 || e == nil {
		return freeHeadIdentity{}, false
	}
	if !e.Compressed {
		if e.Offset == nil || e.Generation == nil || e.ObjectStream != nil || e.ObjectStreamInd != nil ||
			(nr != 0 && (e.Free || *e.Offset <= 0)) {
			return freeHeadIdentity{}, false
		}
		return freeHeadIdentity{free: e.Free, offset: *e.Offset, generation: *e.Generation}, true
	}
	if nr == 0 || e.Free || e.Offset != nil || e.Generation != nil || e.ObjectStream == nil || e.ObjectStreamInd == nil ||
		*e.ObjectStream <= 0 || *e.ObjectStream == nr || *e.ObjectStreamInd < 0 {
		return freeHeadIdentity{}, false
	}
	container := table[*e.ObjectStream]
	if container == nil || container.Free || container.Compressed || container.Offset == nil || *container.Offset <= 0 ||
		container.Generation == nil || *container.Generation != 0 || container.ObjectStream != nil || container.ObjectStreamInd != nil {
		return freeHeadIdentity{}, false
	}
	return freeHeadIdentity{compressed: true, objectStream: *e.ObjectStream, objectStreamInd: *e.ObjectStreamInd}, true
}

// emptyFreeHeadSnapshot permits only generation-0 empty-list canonicalization.
// Copy compressed member identities as scalars too; no live origin may change.
func emptyFreeHeadSnapshot(table map[int]*model.XRefTableEntry) map[int]freeHeadIdentity {
	head := table[0]
	if head == nil || !head.Free || head.Compressed || head.Offset == nil || head.Generation == nil ||
		*head.Offset != 0 || *head.Generation != 0 {
		return nil
	}
	before := make(map[int]freeHeadIdentity, len(table))
	members := map[[2]int]bool{}
	for nr := range table {
		identity, ok := freeHeadEntryIdentity(table, nr)
		if !ok {
			return nil
		}
		if identity.compressed {
			member := [2]int{identity.objectStream, identity.objectStreamInd}
			if members[member] {
				return nil
			}
			members[member] = true
		}
		before[nr] = identity
	}
	return before
}

func unchangedEmptyFreeHead(table map[int]*model.XRefTableEntry, before map[int]freeHeadIdentity) bool {
	if before == nil || len(table) != len(before) {
		return false
	}
	for nr, old := range before {
		identity, ok := freeHeadEntryIdentity(table, nr)
		if !ok {
			return false
		}
		if nr == 0 {
			old.generation = types.FreeHeadGeneration
		}
		if old != identity {
			return false
		}
	}
	return true
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
		if !repairableLastXRefSection(ctx, err) {
			return err
		}
		zero := int64(0)
		offset = &zero
	}

	ctx.Write.OffsetPrevXRef = offset

	err = buildXRefTableStartingAt(c, ctx, offset)
	if errors.Is(err, io.EOF) {
		return fmt.Errorf("xref table: unexpected eof: %w", err)
	}
	if err != nil {
		return
	}

	//Log list of free objects (not the "free list").
	//log.Read.Printf("freelist: %v\n", ctx.freeObjects())

	// Note: Acrobat 6.0 and later do not use the free list to recycle object numbers - pdfcpu does.
	// Track actual normalization, including head reconstruction/generation,
	// dangling free entries and invalid links. Do not change ordinary repair
	// acceptance or the model API used by writers.
	type freeIdentity struct {
		offset     int64
		generation int
	}
	before := map[int]freeIdentity{}
	for nr, e := range ctx.Table {
		if e != nil && (e.Free || nr == 0) && e.Offset != nil && e.Generation != nil {
			before[nr] = freeIdentity{*e.Offset, *e.Generation}
		}
	}
	emptyHeadBefore := emptyFreeHeadSnapshot(ctx.Table)
	err = ctx.EnsureValidFreeList()
	// This sole free-head generation canonicalization cannot change live origins.
	// It does not clear earlier repairs or authorize a hint by itself.
	benignHead := err == nil && unchangedEmptyFreeHead(ctx.Table, emptyHeadBefore)
	for nr, e := range ctx.Table {
		if e != nil && (e.Free || nr == 0) && e.Offset != nil && e.Generation != nil {
			old, found := before[nr]
			if (!found || old != (freeIdentity{*e.Offset, *e.Generation})) && !(nr == 0 && benignHead) {
				ctx.Read.RepairedXRef = true
			}
		}
	}

	if log.ReadEnabled() {
		log.Read.Println("readXRefTable: end")
	}

	return err
}

func isMissingXRefSection(err error) bool {
	return errors.Is(err, ErrMissingXRefSection)
}

func repairableLastXRefSection(ctx *model.Context, err error) bool {
	if isMissingXRefSection(err) {
		return true
	}
	return ctx.XRefTable.ValidationMode == model.ValidationRelaxed && errors.Is(err, errMissingXRefEOF)
}

func growBufBy(buf []byte, size int, rd io.Reader) ([]byte, error) {
	b := make([]byte, size)

	if _, err := fillBuffer(rd, b); err != nil {
		return nil, err
	}
	//log.Read.Printf("growBufBy: Read %d bytes\n", n)

	return append(buf, b...), nil
}

func nextStreamOffset(line string, streamInd int) (int, error) {
	const marker = "stream"
	if streamInd < 0 || streamInd > len(line)-len(marker) || line[streamInd:streamInd+len(marker)] != marker {
		return 0, errCorruptStreamMarker
	}
	off := streamInd + len(marker)

	// Relaxed compatibility: tolerate optional ASCII spaces before the required EOL.
	for off < len(line) && line[off] == 0x20 {
		off++
	}
	if off == len(line) {
		return 0, errTruncatedStreamMarker
	}

	// Skip 0A eol.
	if line[off] == '\n' {
		return off + 1, nil
	}

	// Skip 0D eol.
	if line[off] == '\r' {
		off++
		// Skip 0D0A eol.
		if off < len(line) && line[off] == '\n' {
			off++
		}
	}

	return off, nil
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

func objectBufferLimit(ctx *model.Context) int64 {
	if ctx.Configuration == nil || ctx.Configuration.Limits.MaxObjectBytes == 0 {
		return model.DefaultResourceLimits().MaxObjectBytes
	}
	return ctx.Configuration.Limits.MaxObjectBytes
}

func growObjectBuffer(buf []byte, growSize int, rd io.Reader, maxObjectBytes int64) ([]byte, error) {
	remaining := maxObjectBytes - int64(len(buf))
	if remaining <= 0 {
		return nil, fmt.Errorf("maximum length %d: %w", maxObjectBytes, errObjectBufferLimit)
	}
	if int64(growSize) > remaining {
		growSize = int(remaining)
	}
	return growBufBy(buf, growSize, rd)
}

func streamOffsetForBuffer(buf []byte, line string, streamInd int, rd io.Reader, maxObjectBytes int64) ([]byte, int64, error) {
	for {
		off, err := nextStreamOffset(line, streamInd)
		if err == nil {
			return buf, int64(off), nil
		}
		if !errors.Is(err, errTruncatedStreamMarker) {
			return nil, 0, err
		}

		buf, err = growObjectBuffer(buf, defaultBufSize, rd, maxObjectBytes)
		if err != nil {
			return nil, 0, err
		}
		line = string(buf)
	}
}

// Provide a PDF file buffer of sufficient size for parsing an object w/o stream.
func buffer(c context.Context, rd io.Reader, maxObjectBytes int64) (buf []byte, endInd int, streamInd int, streamOffset int64, err error) {
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

		if buf, err = growObjectBuffer(buf, growSize, rd, maxObjectBytes); err != nil {
			return nil, 0, 0, 0, err
		}

		growSize = min(growSize*2, maxBufSize)
		line := string(buf)

		endInd, streamInd, err = model.DetectKeywords(c, line)
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
			// Recovery searches substrings, not lexical origins. Audit the FINAL
			// marker while its entire prefix is still available, for ordinary
			// objects and xref streams alike. Retain historical parsing, but
			// never let a comment/name/string/container confer hint authority.
			if p := contextutil.ParseProvenanceFromContext(c); p != nil {
				mode, auditErr := lexicalCutState(c, buf, 0, streamInd)
				if auditErr != nil {
					return nil, 0, 0, 0, auditErr
				}
				if mode != 0 {
					p.MarkRepair()
				}
			}

			// streamOffset ... the offset where the actual stream data begins.
			//                  is right after the eol after "stream".
			buf, streamOffset, err = streamOffsetForBuffer(buf, line, streamInd, rd, maxObjectBytes)
			if err != nil {
				return nil, 0, 0, 0, err
			}
			// Historical marker recovery is retained, but cannot establish
			// an exact physical envelope for hint authorization.
			markerEnd := streamInd + len("stream")
			if markerEnd >= len(buf) || (buf[markerEnd] != '\n' && buf[markerEnd] != '\r') {
				contextutil.ParseProvenanceFromContext(c).MarkRepair()
			}
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

// inlineImageFilterAliases are standardized for inline images only.
// Relaxed validation also accepts them for regular streams for reader compatibility.
var inlineImageFilterAliases = map[string]string{
	"AHx": filter.ASCIIHex,
	"A85": filter.ASCII85,
	"LZW": filter.LZW,
	"Fl":  filter.Flate,
	"RL":  filter.RunLength,
	"CCF": filter.CCITTFax,
	"DCT": filter.DCT,
}

func streamFilterName(ctx *model.Context, name string) string {
	if ctx.XRefTable.ValidationMode == model.ValidationRelaxed {
		if canonicalName, ok := inlineImageFilterAliases[name]; ok {
			return canonicalName
		}
	}
	return name
}

func normalizeStreamFilterName(ctx *model.Context, dict types.Dict, name string) string {
	filterName := streamFilterName(ctx, name)
	if filterName == name {
		return name
	}
	dict.Update("Filter", types.Name(filterName))
	model.ShowRepaired(fmt.Sprintf("stream filter %s", name))
	return filterName
}

func normalizeStreamFilterArray(ctx *model.Context, dict types.Dict, filters types.Array, names []string) []string {
	if ctx.XRefTable.ValidationMode != model.ValidationRelaxed {
		return names
	}
	canonicalFilters := filters.Clone().(types.Array)
	repaired := false
	for i, name := range names {
		filterName := streamFilterName(ctx, name)
		if filterName != name {
			canonicalFilters[i] = types.Name(filterName)
			names[i] = filterName
			repaired = true
		}
	}
	if !repaired {
		return names
	}
	dict.Update("Filter", canonicalFilters)
	model.ShowRepaired("stream filter pipeline")
	return names
}

func filterNames(c context.Context, ctx *model.Context, filterArray types.Array) ([]string, error) {
	names := make([]string, len(filterArray))
	for i, o := range filterArray {
		if ir, ok := o.(types.IndirectRef); ok {
			var err error
			o, err = dereferencedObject(c, ctx, ir.ObjectNumber.Value())
			if err != nil {
				return nil, fmt.Errorf("filter pipeline entry %d obj#%d: %w", i, ir.ObjectNumber.Value(), err)
			}
		}
		name, ok := o.(types.Name)
		if !ok {
			return nil, fmt.Errorf("filter pipeline entry %d: %w, got %T", i, errCorruptFilterArray, o)
		}
		names[i] = name.Value()
	}
	return names, nil
}

func buildFilterPipeline(c context.Context, ctx *model.Context, filterNames []string, decodeParmsArr types.Array) ([]types.PDFFilter, error) {
	var filterPipeline []types.PDFFilter

	for i, filterName := range filterNames {
		name := streamFilterName(ctx, filterName)
		if decodeParmsArr == nil || decodeParmsArr[i] == nil {
			filterPipeline = append(filterPipeline, types.PDFFilter{Name: name, DecodeParms: nil})
			continue
		}

		dict, ok := decodeParmsArr[i].(types.Dict)
		if !ok {
			indRef, ok := decodeParmsArr[i].(types.IndirectRef)
			if !ok {
				return nil, fmt.Errorf("filter pipeline DecodeParms entry %d: %w", i, errCorruptDecodeParms)
			}
			d, err := dereferencedDict(c, ctx, indRef.ObjectNumber.Value())
			if err != nil {
				return nil, fmt.Errorf("filter pipeline DecodeParms entry %d: %w", i, err)
			}
			dict = d
		}

		filterPipeline = append(filterPipeline, types.PDFFilter{Name: name, DecodeParms: dict})
	}

	return filterPipeline, nil
}

func singleFilter(c context.Context, ctx *model.Context, filterName string, d types.Dict) ([]types.PDFFilter, error) {
	filterName = streamFilterName(ctx, filterName)
	obj, found := d.Find("DecodeParms")
	if !found {
		// w/o decode parameters.
		if log.ReadEnabled() {
			log.Read.Println("singleFilter: end w/o decode parms")
		}
		return []types.PDFFilter{{Name: filterName}}, nil
	}

	var err error

	if indRef, ok := obj.(types.IndirectRef); ok {
		obj, err = dereferencedObject(c, ctx, indRef.ObjectNumber.Value())
		if err != nil {
			return nil, fmt.Errorf("single filter DecodeParms: %w", err)
		}
	}

	if d, ok := obj.(types.Dict); ok {
		if len(d) == 0 {
			d = nil
		}
		return []types.PDFFilter{{Name: filterName, DecodeParms: d}}, nil
	}

	if arr, ok := obj.(types.Array); ok {
		if len(arr) > 1 {
			return nil, fmt.Errorf("single filter %s DecodeParms: %w", filterName, errCorruptDecodeParms)
		}
		if len(arr) == 0 || arr[0] == nil {
			return []types.PDFFilter{{Name: filterName}}, nil
		}
		d, ok := arr[0].(types.Dict)
		if !ok {
			return nil, fmt.Errorf("single filter %s DecodeParms: %w", filterName, errCorruptDecodeParms)
		}
		if len(d) == 0 {
			d = nil
		}
		return []types.PDFFilter{{Name: filterName, DecodeParms: d}}, nil
	}

	return nil, fmt.Errorf("single filter %s DecodeParms: %w", filterName, errCorruptDecodeParms)
}

func filterArraySupportsDecodeParms(ctx *model.Context, names []string) bool {
	for _, name := range names {
		if filter.SupportsDecodeParms(streamFilterName(ctx, name)) {
			return true
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
			return nil, fmt.Errorf("filter pipeline Filter: %w", err)
		}
	}

	//fmt.Printf("dereferenced filter obj: %s\n", obj)

	if name, ok := o.(types.Name); ok {
		filterName := normalizeStreamFilterName(ctx, dict, name.String())
		return singleFilter(c, ctx, filterName, dict)
	}

	// filter pipeline.

	// Array of filternames
	filterArray, ok := o.(types.Array)
	if !ok {
		return nil, fmt.Errorf("filter pipeline: %w", errCorruptFilterArray)
	}
	names, err := filterNames(c, ctx, filterArray)
	if err != nil {
		return nil, err
	}

	// Optional array of decode parameter dicts.
	var decodeParmsArr types.Array
	decodeParms, found := dict.Find("DecodeParms")
	if found {
		if filterArraySupportsDecodeParms(ctx, names) {
			decodeParmsArr, ok = decodeParms.(types.Array)
			if ok {
				if len(decodeParmsArr) != len(filterArray) {
					return nil, fmt.Errorf("filter pipeline DecodeParms: %w", errCorruptDecodeParms)
				}
			}
		}
	}

	names = normalizeStreamFilterArray(ctx, dict, filterArray, names)

	//fmt.Printf("decodeParmsArr: %s\n", decodeParmsArr)

	filterPipeline, err = buildFilterPipeline(c, ctx, names, decodeParmsArr)

	if log.ReadEnabled() {
		log.Read.Println("pdfFilterPipeline: end")
	}

	return filterPipeline, err
}

func streamDictForObject(c context.Context, ctx *model.Context, d types.Dict, objNr, streamInd int, streamOffset, offset int64) (sd types.StreamDict, err error) {
	streamLength, streamLengthRef := d.Length()

	if streamInd <= 0 {
		return sd, fmt.Errorf("stream obj#%d: %w", objNr, errMissingStreamOffset)
	}

	filterPipeline, err := pdfFilterPipeline(c, ctx, d)
	if err != nil {
		return sd, fmt.Errorf("stream obj#%d filter pipeline: %w", objNr, err)
	}

	streamOffset += offset

	// We have a stream object.
	sd = types.NewStreamDict(d, streamOffset, streamLength, streamLengthRef, filterPipeline)

	if log.ReadEnabled() {
		log.Read.Printf("streamDictForObject: end, Streamobject #%d\n", objNr)
	}

	return sd, nil
}

func dict(c context.Context, ctx *model.Context, d1 types.Dict, objNr, genNr, endInd, streamInd int) (d2 types.Dict, err error) {
	if ctx.EncKey != nil {
		if _, err := decryptDeepObject(c, d1, objNr, genNr, ctx.EncKey, ctx.AES4Strings, ctx.E.R); err != nil {
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

func parseObjectWithValidationPolicy(c context.Context, ctx *model.Context, line *string, objNr int) (types.Object, error) {
	result, err := model.ParseObjectWithPolicy(c, line, 0, ctx.XRefTable.ValidationMode, recursionLimit(ctx))
	if result.StrictFailure == nil {
		return result.Object, err
	}
	if err != nil {
		return nil, model.WithValidationErrorObject(err, objNr)
	}

	cause := model.WithValidationErrorObject(result.StrictFailure, objNr)
	notice := model.NewValidationNotice(
		model.NoticePhaseParse,
		model.NoticeDigested,
		"object dictionary contains a non-name key token",
		cause,
	)
	ctx.AddValidationNotice(notice)
	return result.Object, nil
}

func object(c context.Context, ctx *model.Context, offset int64, objNr, genNr int, validationMode ...int) (o types.Object, endInd, streamInd int, streamOffset int64, err error) {
	var rd io.Reader

	if rd, err = newPositionedReader(ctx.Read.RS, &offset); err != nil {
		return nil, 0, 0, 0, err
	}

	//log.Read.Printf("object: seeked to offset:%d\n", offset)

	// process: # gen obj ... obj dict ... {stream ... data ... endstream} endobj
	//                                    streamInd                        endInd
	//                                  -1 if absent                    -1 if absent
	var buf []byte
	if buf, endInd, streamInd, streamOffset, err = buffer(c, rd, objectBufferLimit(ctx)); err != nil {
		return nil, 0, 0, 0, err
	}

	//log.Read.Printf("buflen=%d\n%s", len(buf), hex.Dump(buf))

	line := string(buf)

	if endInd >= 0 && (streamInd < 0 || endInd < streamInd) {
		if p := contextutil.ParseProvenanceFromContext(c); p != nil {
			mode, auditErr := lexicalCutState(c, buf, 0, endInd)
			if auditErr != nil {
				return nil, 0, 0, 0, auditErr
			}
			if mode != 0 {
				p.MarkRepair()
			}
		}
		// The permissive detector may recover junk/non-PDF terminators.
		// Audit the actual bytes before slicing the marker away below.
		end := endInd + len("endobj")
		if end >= len(line) || !strings.HasPrefix(line[endInd:], "endobj") ||
			!contextutil.PDFTokenBoundary(line, end) ||
			(endInd > 0 && !contextutil.PDFWhitespace(rune(line[endInd-1])) &&
				!strings.ContainsRune(")>]", rune(line[endInd-1]))) {
			contextutil.ParseProvenanceFromContext(c).MarkRepair()
		}
	}

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

	if !literalObjectHeader([]byte(line[:min(128, len(line))]), *objectNr, *generationNr) || objNr != *objectNr || genNr != *generationNr {
		contextutil.ParseProvenanceFromContext(c).MarkRepair()
	}
	if objNr != *objectNr || genNr != *generationNr {
		// This is suspicious, but ok if two object numbers point to same offset and only one of them is used
		// (compare entry.RefCount) like for cases where the PDF Writer is MS Word 2013.
		if log.ReadEnabled() {
			log.Read.Printf("object %d: non matching objNr(%d) or generationNumber(%d) tags found.\n", objNr, *objectNr, *generationNr)
		}
	}

	// Audit leading separators before the historical TrimSpace removes
	// them. Do not scan string or comment payload as structural whitespace.
	leading := strings.TrimLeftFunc(l, unicode.IsSpace)
	markObjectRemainder(c, l[:len(l)-len(leading)])
	l = leading
	if len(strings.TrimSpace(l)) == 0 {
		// 7.3.9
		// Specifying the null object as the value of a dictionary entry (7.3.7, "Dictionary Objects")
		// shall be equivalent to omitting the entry entirely.
		return nil, endInd, streamInd, streamOffset, err
	}

	if len(validationMode) > 0 {
		// Structural hint probes must not repair syntax or emit duplicate
		// notices before the ordinary parser processes the same object.
		result, parseErr := model.ParseObjectWithPolicy(c, &l, 0, validationMode[0], recursionLimit(ctx))
		o, err = result.Object, parseErr
	} else {
		o, err = parseObjectWithValidationPolicy(c, ctx, &l, objNr)
	}

	if err == nil {
		markObjectRemainder(c, l)
	}
	return o, endInd, streamInd, streamOffset, err
}

func resolveObject(c context.Context, ctx *model.Context, obj types.Object, offset int64, objNr, genNr, endInd, streamInd int, streamOffset int64) (types.Object, error) {
	switch o := obj.(type) {

	case types.Dict:
		d, err := dict(c, ctx, o, objNr, genNr, endInd, streamInd)
		if err != nil || d != nil {
			// Dict
			return d, err
		}
		// StreamDict.
		return streamDictForObject(c, ctx, o, objNr, streamInd, streamOffset, offset)

	case types.Array:
		if ctx.EncKey != nil {
			if _, err := decryptDeepObject(c, o, objNr, genNr, ctx.EncKey, ctx.AES4Strings, ctx.E.R); err != nil {
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

// ParseObject parses an object at offset using c for cancellation.
func ParseObject(c context.Context, ctx *model.Context, offset int64, objNr, genNr int) (types.Object, error) {
	if log.ReadEnabled() {
		log.Read.Printf("ParseObject: begin, obj#%d, offset:%d\n", objNr, offset)
	}

	obj, endInd, streamInd, streamOffset, err := object(c, ctx, offset, objNr, genNr)
	if err != nil {
		if ctx.XRefTable.ValidationMode == model.ValidationRelaxed {
			if errors.Is(err, io.EOF) {
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
		return nil, fmt.Errorf("object %d: %w", objNr, errUnregisteredObject)
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

		o, err := ParseObject(c, ctx, *entry.Offset, objNr, *entry.Generation)
		if err != nil {
			return nil, fmt.Errorf("problem dereferencing object %d: %w", objNr, err)
		}

		if o == nil {
			return nil, fmt.Errorf("object %d: %w", objNr, errNilDereferencedObject)
		}

		entry.Object = o
	} else if l, ok := entry.Object.(types.LazyObjectStreamObject); ok {
		o, err := l.DecodedObject(c)
		if err != nil {
			return nil, fmt.Errorf("problem dereferencing object %d: %w", objNr, err)
		}

		if err := model.ProcessRefCountsWithError(ctx.XRefTable, o); err != nil {
			return nil, err
		}
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
		return nil, fmt.Errorf("object %d: %w", objNr, errCorruptIntegerObject)
	}

	return &i, nil
}

func bootstrapIntegerEntry(c context.Context, ctx *model.Context, d types.Dict, key string) (*types.Integer, bool, error) {
	o, found := d.Find(key)
	if !found || o == nil {
		return nil, found, nil
	}

	if i, ok := o.(types.Integer); ok {
		return &i, true, nil
	}

	indRef, ok := o.(types.IndirectRef)
	if !ok {
		return nil, true, fmt.Errorf("entry=%s: expected integer, got %T", key, o)
	}

	objNr := indRef.ObjectNumber.Value()
	entry, found := ctx.XRefTable.FindTableEntryForIndRef(&indRef)
	if !found || entry == nil || entry.Free {
		err := fmt.Errorf("entry=%s: missing indirect target", key)
		return nil, true, model.WithValidationErrorObject(err, objNr)
	}
	// Bootstrap values cannot come from object streams because those streams are not available yet.
	if entry.Compressed {
		err := fmt.Errorf("entry=%s: indirect target is compressed during bootstrap", key)
		return nil, true, model.WithValidationErrorObject(err, objNr)
	}
	if entry.Object == nil && entry.Offset == nil {
		return nil, true, nil
	}

	i, err := dereferencedInteger(c, ctx, objNr)
	if err != nil {
		err = fmt.Errorf("entry=%s: %w", key, err)
		return nil, true, model.WithValidationErrorObject(err, objNr)
	}
	return i, true, nil
}

func materializeBootstrapIntegerEntry(c context.Context, ctx *model.Context, d types.Dict, key string) error {
	o, found := d.Find(key)
	if !found {
		return nil
	}
	if _, ok := o.(types.IndirectRef); !ok {
		return nil
	}
	_, _, err := bootstrapIntegerEntry(c, ctx, d, key)
	return err
}

func materializeEncryptionIntegers(c context.Context, ctx *model.Context, d types.Dict) error {
	for _, key := range []string{"V", "Length", "R", "P"} {
		if err := materializeBootstrapIntegerEntry(c, ctx, d, key); err != nil {
			return fmt.Errorf("encrypt dict: %w", err)
		}
	}

	cf := d.DictEntry("CF")
	for _, name := range slices.Sorted(maps.Keys(cf)) {
		o := cf[name]
		d, ok := o.(types.Dict)
		if !ok {
			continue
		}
		if err := materializeBootstrapIntegerEntry(c, ctx, d, "Length"); err != nil {
			return fmt.Errorf("crypt filter %s: %w", name, err)
		}
	}
	return nil
}

func dereferencedDict(c context.Context, ctx *model.Context, objNr int) (types.Dict, error) {
	o, err := dereferencedObject(c, ctx, objNr)
	if err != nil {
		return nil, err
	}

	d, ok := o.(types.Dict)
	if !ok {
		return nil, fmt.Errorf("object %d: %w", objNr, errCorruptDictObject)
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

func readStreamContentBlindly(rd io.Reader, maxStreamBytes int64) (buf []byte, err error) {
	// Weak heuristic for reading in stream data for cases where stream length is unknown.
	// ...data...{eol}endstream{eol}endobj

	growSize := defaultBufSize
	if int64(growSize) > maxStreamBytes {
		growSize = int(maxStreamBytes)
	}

	if buf, err = growBufBy(buf, growSize, rd); err != nil {
		return nil, err
	}

	i := bytes.Index(buf, []byte("endstream"))
	for i < 0 {
		if int64(len(buf)) >= maxStreamBytes {
			return nil, fmt.Errorf(
				"stream content exceeds maximum blind read length %d",
				maxStreamBytes,
			)
		}

		growSize = min(growSize*2, maxBufSize)

		remaining := maxStreamBytes - int64(len(buf))
		if int64(growSize) > remaining {
			growSize = int(remaining)
		}

		buf, err = growBufBy(buf, growSize, rd)
		if err != nil {
			return nil, err
		}

		i = bytes.Index(buf, []byte("endstream"))
	}

	buf = buf[:i]

	// Cut off trailing eol's.
	for len(buf) > 0 {
		last := buf[len(buf)-1]
		if last != 0x0A && last != 0x0D {
			break
		}
		buf = buf[:len(buf)-1]
	}

	return buf, nil
}

// Reads and returns a file buffer with length = stream length using provided reader positioned at offset.
func readStreamContent(rd io.Reader, streamLength int, maxStreamBytes int64) ([]byte, error) {
	if log.ReadEnabled() {
		log.Read.Printf("readStreamContent: begin streamLength:%d\n", streamLength)
	}

	if streamLength <= 0 {
		// Read until "endstream" then fix "Length", but do not allow
		// unbounded memory growth for missing or corrupt markers.
		return readStreamContentBlindly(rd, maxStreamBytes)
	}

	if int64(streamLength) > maxStreamBytes {
		return nil, fmt.Errorf(
			"stream length %d exceeds maximum supported length %d",
			streamLength,
			maxStreamBytes,
		)
	}

	buf := make([]byte, streamLength)

	totalCount := 0
	for totalCount < streamLength {
		count, err := fillBuffer(rd, buf[totalCount:])
		if count < 0 || count > streamLength-totalCount {
			return nil, fmt.Errorf("invalid stream read count=%d", count)
		}
		if count > 0 {
			totalCount += count

			if log.ReadEnabled() {
				log.Read.Printf("count=%d, buflen=%d(%X)\n", count, len(buf), len(buf))
			}
		}

		if err != nil {
			if !errors.Is(err, io.EOF) {
				return nil, err
			}

			// Search only the bytes actually read. Searching the whole
			// preallocated buffer includes zero-filled unread bytes.
			eob := bytes.Index(buf[:totalCount], []byte("endstream"))
			if eob < 0 {
				return nil, err
			}
			return buf[:eob], nil
		}

		if count == 0 {
			return nil, io.ErrNoProgress
		}
	}

	if log.ReadEnabled() {
		log.Read.Printf("readStreamContent: end\n")
	}

	return buf, nil
}

func ensureStreamLength(sd *types.StreamDict, fixLength bool) error {
	if sd == nil || sd.Dict == nil {
		return errCorruptStreamDict
	}

	l := int64(len(sd.Raw))
	if fixLength || sd.StreamLength == nil || l != *sd.StreamLength {
		sd.StreamLength = &l
		sd.Dict["Length"] = types.Integer(l)
	}

	return nil
}

func decodeLimit(ctx *model.Context) int64 {
	if ctx == nil || ctx.Configuration == nil {
		return model.DefaultResourceLimits().MaxDecodeBytes
	}
	return ctx.Configuration.Limits.MaxDecodeBytes
}

func streamLimit(ctx *model.Context) int64 {
	if ctx == nil || ctx.Configuration == nil {
		return model.DefaultResourceLimits().MaxStreamBytes
	}
	return ctx.Configuration.Limits.MaxStreamBytes
}

func recursionLimit(ctx *model.Context) int {
	if ctx == nil || ctx.Configuration == nil {
		return model.DefaultResourceLimits().MaxRecursionDepth
	}
	return ctx.Configuration.Limits.MaxRecursionDepth
}

func ensureIndirectStreamLength(c context.Context, ctx *model.Context, sd *types.StreamDict, fixLength bool) error {
	if fixLength || sd.StreamLength != nil {
		return nil
	}
	if sd.StreamLengthObjNr == nil {
		if ctx.XRefTable.ValidationMode == model.ValidationStrict {
			return errMissingStreamLength
		}
		model.ShowSkipped("missing stream length")
		return nil
	}

	streamLength, err := int64Object(c, ctx, *sd.StreamLengthObjNr)
	if err != nil && !errors.Is(err, ErrReferenceDoesNotExist) {
		return err
	}
	sd.StreamLength = streamLength

	return nil
}

// auditStreamPostlude inspects the actual encoded-input boundary, before
// decryption or length normalization. It is provenance-only: an unreadable or
// overlong envelope disqualifies hints without changing ordinary acceptance.
func auditStreamPostlude(c context.Context, ctx *model.Context, sd *types.StreamDict) error {
	p := contextutil.ParseProvenanceFromContext(c)
	if p == nil {
		return nil
	}
	if err := c.Err(); err != nil {
		return err
	}
	start := sd.StreamOffset + int64(len(sd.Raw))
	rd, err := newPositionedReader(ctx.Read.RS, &start)
	if err != nil {
		p.MarkRepair()
		return nil
	}
	bb, err := io.ReadAll(io.LimitReader(rd, min(int64(1025), objectBufferLimit(ctx))))
	if err != nil {
		p.MarkRepair()
		return c.Err()
	}
	if err := c.Err(); err != nil {
		return err
	}
	pos := 0
	skip := func() {
		for pos < len(bb) {
			if strings.ContainsRune("\x00\t\n\f\r ", rune(bb[pos])) {
				pos++
				continue
			}
			if bb[pos] != '%' {
				return
			}
			for pos < len(bb) && bb[pos] != '\n' && bb[pos] != '\r' {
				pos++
			}
		}
	}
	for _, marker := range []string{"endstream", "endobj"} {
		skip()
		if !bytes.HasPrefix(bb[pos:], []byte(marker)) {
			p.MarkRepair()
			return nil
		}
		pos += len(marker)
		if pos == len(bb) {
			// Only physical file EOF can terminate the marker at buffer EOF.
			if start+int64(pos) != ctx.Read.FileSize {
				p.MarkRepair()
			}
		} else if !strings.ContainsRune("\x00\t\n\f\r <>[]()/%", rune(bb[pos])) {
			p.MarkRepair()
			return nil
		}
	}
	return nil
}

// loadEncodedStreamContent loads the encoded stream content into sd.
func loadEncodedStreamContent(c context.Context, ctx *model.Context, sd *types.StreamDict, fixLength bool) error {
	if sd.Raw != nil {
		return nil
	}

	if log.ReadEnabled() {
		log.Read.Printf("loadEncodedStreamContent: begin\n%v\n", sd)
	}

	// Read stream content encoded at offset with stream length.

	// Dereference stream length if stream length is an indirect object.
	if err := ensureIndirectStreamLength(c, ctx, sd, fixLength); err != nil {
		return fmt.Errorf("stream length: %w", err)
	}

	rd, err := newPositionedReader(ctx.Read.RS, &sd.StreamOffset)
	if err != nil {
		return fmt.Errorf("position stream reader: %w", err)
	}

	l1 := 0
	if !fixLength && sd.StreamLength != nil {
		l1 = int(*sd.StreamLength)
	}
	sd.Raw, err = readStreamContent(rd, l1, streamLimit(ctx))
	if err != nil {
		return fmt.Errorf("read stream content: %w", err)
	}

	if err := auditStreamPostlude(c, ctx, sd); err != nil {
		return fmt.Errorf("audit stream postlude: %w", err)
	}

	// This is encoded input, before decryption. Length reconstruction here
	// is a reader repair; later plaintext-length adjustment is not.
	if sd.StreamLength == nil || int64(len(sd.Raw)) != *sd.StreamLength {
		contextutil.ParseProvenanceFromContext(c).MarkRepair()
	}
	if err := ensureStreamLength(sd, fixLength); err != nil {
		return fmt.Errorf("stream length: %w", err)
	}

	if log.ReadEnabled() {
		log.Read.Printf("loadEncodedStreamContent: end: len(streamDictRaw)=%d\n", len(sd.Raw))
	}

	return nil
}

func metadataStream(ctx *model.Context, sd *types.StreamDict) (bool, error) {
	if sd == nil {
		return false, nil
	}
	if ctx == nil || ctx.XRefTable == nil {
		t := sd.Dict.Type()
		return t != nil && *t == "Metadata", nil
	}

	t, _, err := ctx.DereferenceNameEntry(sd.Dict, "Type")
	if err != nil {
		return false, fmt.Errorf("metadata stream Type: %w", err)
	}

	return t != nil && t.Value() == "Metadata", nil
}

func unencryptedMetadata(ctx *model.Context, metadata bool) bool {
	return ctx != nil && ctx.XRefTable != nil && ctx.E != nil && !ctx.E.Emd && metadata
}

func repairablePlaintextMetadata(ctx *model.Context, metadata bool, err error) bool {
	if ctx == nil || ctx.XRefTable == nil || ctx.E == nil || ctx.XRefTable.ValidationMode != model.ValidationRelaxed {
		return false
	}
	if !ctx.E.Emd || !ctx.AES4Streams || !metadata {
		return false
	}
	return errors.Is(err, errAESCiphertextTooShort) || errors.Is(err, errAESCiphertextUnaligned)
}

func validPlaintextMetadata(ctx *model.Context, sd *types.StreamDict) bool {
	probe := *sd
	probe.Raw = bytes.Clone(sd.Raw)
	probe.Content = nil
	if err := probe.DecodeWithLimit(decodeLimit(ctx)); err != nil {
		return false
	}

	var x model.XMPMeta
	return xml.Unmarshal(probe.Content, &x) == nil
}

func decryptStreamContent(ctx *model.Context, sd *types.StreamDict, objNr, genNr int) error {
	metadata, err := metadataStream(ctx, sd)
	if err != nil {
		return err
	}
	if ctx == nil || ctx.EncKey == nil || unencryptedMetadata(ctx, metadata) {
		return nil
	}

	raw, err := decryptStream(sd.Raw, objNr, genNr, ctx.EncKey, ctx.AES4Streams, ctx.E.R)
	if err != nil {
		if repairablePlaintextMetadata(ctx, metadata, err) && validPlaintextMetadata(ctx, sd) {
			model.ShowRepaired(fmt.Sprintf("plaintext metadata obj#%d", objNr))
			return nil
		}
		return err
	}
	sd.Raw = raw

	return ensureStreamLength(sd, true)
}

func isImageStreamDict(ctx *model.Context, sd *types.StreamDict) (bool, error) {
	if ctx == nil {
		return sd.Image(), nil
	}

	t, _, err := ctx.DereferenceNameEntry(sd.Dict, "Type")
	if err != nil {
		return false, fmt.Errorf("stream dict Type: %w", err)
	}
	if t == nil || t.Value() != "XObject" {
		return false, nil
	}

	subtype, _, err := ctx.DereferenceNameEntry(sd.Dict, "Subtype")
	if err != nil {
		return false, fmt.Errorf("stream dict Subtype: %w", err)
	}

	return subtype != nil && subtype.Value() == "Image", nil
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
	if err := decryptStreamContent(ctx, sd, objNr, genNr); err != nil {
		return fmt.Errorf("decrypt stream content: %w", err)
	}

	if !decode {
		return nil
	}

	image, err := isImageStreamDict(ctx, sd)
	if err != nil {
		return fmt.Errorf("classify image stream: %w", err)
	}
	if image {
		return nil
	}

	// Actual decoding of stream data.
	err = sd.DecodeWithLimit(decodeLimit(ctx))
	if errors.Is(err, filter.ErrUnsupportedFilter) {
		err = nil
	}
	if err != nil {
		return fmt.Errorf("decode stream content: %w", err)
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
		return fmt.Errorf("object stream %d: %w", *entry.ObjectStream, errMissingObjectStreamEntry)
	}

	// Object of this entry has to be a ObjectStreamDict.
	sd, ok := objectStreamXRefTableEntry.Object.(types.ObjectStreamDict)
	if !ok {
		return fmt.Errorf("object stream %d: %w", *entry.ObjectStream, errMissingObjectStream)
	}

	// Get indexed object from ObjectStreamDict.
	o, err := sd.IndexedObject(*entry.ObjectStreamInd)
	if err != nil {
		return fmt.Errorf("decompressXRefTableEntry: problem dereferencing object stream %d: %w", *entry.ObjectStream, err)
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

// compressedClaim retains the effective xref identity for one container member.
// The index is private to a single read and never survives xref reconstruction.
type compressedClaim struct {
	nr    int
	entry *model.XRefTableEntry
}

func indexCompressedClaims(c context.Context, table map[int]*model.XRefTableEntry) (map[int][]compressedClaim, error) {
	claims := map[int][]compressedClaim{}
	for nr, e := range table {
		if err := c.Err(); err != nil {
			return nil, err
		}
		if e != nil && e.Compressed && e.ObjectStream != nil {
			claims[*e.ObjectStream] = append(claims[*e.ObjectStream], compressedClaim{nr, e})
		}
	}
	return claims, nil
}

func auditCompressedClaims(c context.Context, claims []compressedClaim, objs []string) error {
	for _, claim := range claims {
		if err := c.Err(); err != nil {
			return err
		}
		e := claim.entry
		if e.ObjectStreamInd == nil || *e.ObjectStreamInd < 0 || *e.ObjectStreamInd >= len(objs)/2 {
			contextutil.ParseProvenanceFromContext(c).MarkRepair()
			continue
		}
		claimed, err := strconv.Atoi(objs[2*(*e.ObjectStreamInd)])
		if err != nil || claimed != claim.nr {
			contextutil.ParseProvenanceFromContext(c).MarkRepair()
		}
	}
	return nil
}

func decodeObjectStreamObjects(c context.Context, ctx *model.Context, sd *types.StreamDict, objNr int, claims []compressedClaim) (*types.ObjectStreamDict, error) {
	first, _, err := bootstrapIntegerEntry(c, ctx, sd.Dict, "First")
	if err != nil {
		return nil, fmt.Errorf("object stream obj#%d: parse dictionary: object stream entry First: %w", objNr, err)
	}
	n, _, err := bootstrapIntegerEntry(c, ctx, sd.Dict, "N")
	if err != nil {
		return nil, fmt.Errorf("object stream obj#%d: parse dictionary: object stream entry N: %w", objNr, err)
	}
	osd, err := model.ObjectStreamDictWithResolvedIntegers(sd, ctx.Configuration.Limits, n, first)
	if err != nil {
		return nil, fmt.Errorf("object stream obj#%d: parse dictionary: %w", objNr, err)
	}

	if log.ReadEnabled() {
		log.Read.Printf("decodeObjectStreamObjects: decoding object stream %d:\n", objNr)
	}

	// Parse all objects of this object stream and save them to ObjectStreamDict.ObjArray.
	audit := func(objs []string) error {
		return auditCompressedClaims(c, claims, objs)
	}
	if err = parseObjectStream(c, osd, ctx.Configuration.Limits, audit); err != nil {
		return nil, fmt.Errorf("object stream obj#%d: parse objects: %w", objNr, err)
	}

	if osd.ObjArray == nil {
		return nil, fmt.Errorf("object stream obj#%d: %w", objNr, errMissingObjectStreamObjects)
	}

	if log.ReadEnabled() {
		log.Read.Printf("decodeObjectStreamObjects: decoded object stream %d:\n", objNr)
	}

	return osd, nil
}

func decodeObjectStream(c context.Context, ctx *model.Context, objNr int, indexes ...map[int][]compressedClaim) error {
	var claims map[int][]compressedClaim
	if len(indexes) > 0 {
		claims = indexes[0]
	} else {
		var err error
		claims, err = indexCompressedClaims(c, ctx.Table)
		if err != nil {
			return err
		}
	}
	entry := ctx.Table[objNr]
	if entry == nil {
		return fmt.Errorf("object stream obj#%d: %w", objNr, errMissingObjectStreamEntry)
	}

	if log.ReadEnabled() {
		log.Read.Printf("decodeObjectStream: parsing object stream for obj#%d\n", objNr)
	}

	// Parse object stream from file.
	o, err := ParseObject(c, ctx, *entry.Offset, objNr, *entry.Generation)
	if err != nil {
		return fmt.Errorf("object stream obj#%d: parse object: %w", objNr, err)
	}
	if o == nil {
		return fmt.Errorf("object stream obj#%d: %w", objNr, errCorruptObjectStream)
	}

	// Ensure StreamDict
	sd, ok := o.(types.StreamDict)
	if !ok {
		return fmt.Errorf("object stream obj#%d: %w", objNr, errCorruptObjectStream)
	}

	// Load encoded stream content to xRefTable.
	if err = loadEncodedStreamContent(c, ctx, &sd, false); err != nil {
		return fmt.Errorf("object stream obj#%d: load encoded content: %w", objNr, err)
	}

	// Will only decrypt, the actual stream content is decoded later lazily.
	if err = saveDecodedStreamContent(ctx, &sd, objNr, *entry.Generation, false); err != nil {
		if log.ReadEnabled() {
			log.Read.Printf("obj %d: %s", objNr, err)
		}
		return fmt.Errorf("object stream obj#%d: decode content: %w", objNr, err)
	}

	// We have an object stream.
	if log.ReadEnabled() {
		log.Read.Printf("decodeObjectStream: object stream #%d\n", objNr)
	}

	ctx.Read.UsingObjectStreams = true

	osd, err := decodeObjectStreamObjects(c, ctx, &sd, objNr, claims[objNr])
	if err != nil {
		return err
	}

	// Save object stream dict to xRefTableEntry.
	entry.Object = *osd

	return nil
}

func materializeObjectStreamType(c context.Context, ctx *model.Context, o types.Object, objNr int) error {
	ir, ok := o.(types.IndirectRef)
	if !ok {
		return nil
	}
	target, found := ctx.FindTableEntryForIndRef(&ir)
	if !found || target == nil || target.Free {
		return nil
	}
	if !target.Compressed && (target.Object != nil || target.Offset == nil) {
		return nil
	}
	if _, err := dereferencedObject(c, ctx, ir.ObjectNumber.Value()); err != nil {
		err = fmt.Errorf("object stream obj#%d Type: entry=Type: %w", objNr, err)
		return model.WithValidationErrorObject(err, ir.ObjectNumber.Value())
	}
	return nil
}

func validateObjectStreamType(c context.Context, ctx *model.Context, objNr int) error {
	entry := ctx.Table[objNr]
	if entry == nil {
		return fmt.Errorf("object stream obj#%d: %w", objNr, errMissingObjectStreamEntry)
	}
	osd, ok := entry.Object.(types.ObjectStreamDict)
	if !ok {
		return fmt.Errorf("object stream obj#%d: %w", objNr, errMissingObjectStream)
	}

	o := osd.Dict["Type"]
	if err := materializeObjectStreamType(c, ctx, o, objNr); err != nil {
		return err
	}

	t, found, err := ctx.DereferenceNameEntry(osd.Dict, "Type")
	if err != nil {
		return fmt.Errorf("object stream obj#%d Type: %w", objNr, err)
	}
	if found && t != nil && t.Value() == "ObjStm" {
		return nil
	}
	err = fmt.Errorf("object stream obj#%d: %w", objNr, errCorruptObjectStream)
	if ir, ok := o.(types.IndirectRef); ok {
		return model.WithValidationErrorObject(err, ir.ObjectNumber.Value())
	}
	return err
}

// Decode all object streams so contained objects are ready to be used.
func decodeObjectStreams(c context.Context, ctx *model.Context) error {
	// Note:
	// Entry "Extends" intentionally left out.
	// No object stream collection validation necessary.

	if log.ReadEnabled() {
		log.Read.Println("decodeObjectStreams: begin")
	}

	claims, err := indexCompressedClaims(c, ctx.Table)
	if err != nil {
		return fmt.Errorf("index compressed claims: %w", err)
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
		if err := decodeObjectStream(c, ctx, objNr, claims); err != nil {
			return err
		}
	}
	for _, objNr := range keys {
		if err := validateObjectStreamType(c, ctx, objNr); err != nil {
			return err
		}
	}

	if log.ReadEnabled() {
		log.Read.Println("decodeObjectStreams: end")
	}

	return nil
}

func validateLinearizationDirectEntries(d types.Dict, objNr int) error {
	for _, key := range slices.Sorted(maps.Keys(d)) {
		o := d[key]
		if _, ok := o.(types.IndirectRef); ok {
			return fmt.Errorf("linearization dict obj#%d entry %s: value must be direct", objNr, key)
		}
	}
	return nil
}

func handleLinearizationParmDict(ctx *model.Context, obj types.Object, objNr int) error {
	if ctx.Read.Linearized {
		// Linearization dict already processed.
		return nil
	}

	d, ok := obj.(types.Dict)
	if !ok {
		return nil
	}
	marker, found := d.Find("Linearized")
	if !found {
		return nil
	}
	if _, ok := marker.(types.IndirectRef); ok {
		return fmt.Errorf("linearization dict obj#%d entry Linearized: value must be direct", objNr)
	}
	if !d.IsLinearizationParmDict() {
		return nil
	}
	if err := validateLinearizationDirectEntries(d, objNr); err != nil {
		return err
	}

	ctx.Read.Linearized = true
	if log.ReadEnabled() {
		log.Read.Printf("handleLinearizationParmDict: identified linearizationObj #%d\n", objNr)
	}

	a := d.ArrayEntry("H")

	if a == nil {
		return fmt.Errorf("handleLinearizationParmDict: corrupt linearization dict at obj:%d - missing array entry H", objNr)
	}

	if len(a) != 2 && len(a) != 4 {
		return fmt.Errorf("handleLinearizationParmDict: corrupt linearization dict at obj:%d - corrupt array entry H, needs length 2 or 4", objNr)
	}

	offset, ok := a[0].(types.Integer)
	if !ok {
		return fmt.Errorf("handleLinearizationParmDict: corrupt linearization dict at obj:%d - corrupt array entry H, needs Integer values", objNr)
	}

	offset64 := int64(offset.Value())
	ctx.OffsetPrimaryHintTable = &offset64

	if len(a) == 4 {

		offset, ok := a[2].(types.Integer)
		if !ok {
			return fmt.Errorf("handleLinearizationParmDict: corrupt linearization dict at obj:%d - corrupt array entry H, needs Integer values", objNr)
		}

		offset64 := int64(offset.Value())
		ctx.OffsetOverflowHintTable = &offset64
	}

	return nil
}

func loadStreamDict(c context.Context, ctx *model.Context, sd *types.StreamDict, objNr, genNr int, fixLength bool) error {
	// Load encoded stream content for stream dicts into xRefTable entry.
	if err := loadEncodedStreamContent(c, ctx, sd, fixLength); err != nil {
		return fmt.Errorf("stream obj#%d: load encoded content: %w", objNr, err)
	}

	// Decode stream content.
	if err := saveDecodedStreamContent(ctx, sd, objNr, genNr, ctx.DecodeAllStreams); err != nil {
		return fmt.Errorf("stream obj#%d: decode content: %w", objNr, err)
	}

	ctx.Read.BinaryTotalSize += *sd.StreamLength

	return nil
}

func updateBinaryTotalSize(ctx *model.Context, o types.Object) {
	var streamLength *int64
	switch o := o.(type) {
	case types.StreamDict:
		streamLength = o.StreamLength
	case types.ObjectStreamDict:
		streamLength = o.StreamLength
	case types.XRefStreamDict:
		streamLength = o.StreamLength
	}
	if streamLength != nil {
		ctx.Read.BinaryTotalSize += *streamLength
	}
}

func dereferenceAndLoad(c context.Context, ctx *model.Context, objNr int, entry *model.XRefTableEntry, hints ...*hintReadState) error {
	if log.ReadEnabled() {
		log.Read.Printf("dereferenceAndLoad: dereferencing object %d\n", objNr)
	}

	// Parse object from ctx: anything goes dict, array, integer, float, streamdict...
	o, err := ParseObject(c, ctx, *entry.Offset, objNr, *entry.Generation)
	if err != nil {
		if ctx.XRefTable.ValidationMode == model.ValidationStrict {
			return fmt.Errorf("dereferenceAndLoad: problem dereferencing object %d: %w", objNr, err)
		}
		if ctx.Read.RepairOffset > 0 {
			o, err = ParseObject(c, ctx, *entry.Offset+ctx.Read.RepairOffset, objNr, *entry.Generation)
		}
		if err != nil {
			model.ShowSkipped(fmt.Sprintf("obj #%d reason: %v", objNr, err))
		}
		if errors.Is(err, model.ErrCorruptObjectOffset) || errors.Is(err, errUnregisteredObject) {
			return err
		}
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
		return fmt.Errorf("object stream should already be dereferenced at obj:%d", objNr)
	}

	if _, ok := o.(types.XRefStreamDict); ok {
		return fmt.Errorf("xref stream should already be dereferenced at obj:%d", objNr)
	}

	if sd, ok := o.(types.StreamDict); ok {
		if hintCandidate(hints, objNr) {
			if err := loadEncodedStreamContent(c, ctx, &sd, false); err != nil {
				return fmt.Errorf("hint stream obj#%d: load encoded content: %w", objNr, err)
			}
			entry.Object = sd
			return nil
		}
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

func dereferenceObject(c context.Context, ctx *model.Context, objNr int, hints ...*hintReadState) error {
	return dereferenceObjectEntry(c, ctx, objNr, ctx.Table[objNr], hints...)
}

func dereferenceObjectEntry(c context.Context, ctx *model.Context, objNr int, entry *model.XRefTableEntry, hints ...*hintReadState) error {
	if entry == nil {
		return fmt.Errorf("xref entry for object %d is nil: %w", objNr, errCorruptXRefSubsection)
	}
	if log.ReadEnabled() {
		log.Read.Printf("dereferenceObject: begin, dereferencing object %d\n", objNr)
	}

	if objNr > ctx.MaxObjNr {
		ctx.MaxObjNr = objNr
	}

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
		// Candidate dictionaries may have been cached by structural lookups;
		// cached does not mean their encoded stream has been loaded/finalized.
		if hintCandidate(hints, objNr) {
			if sd, ok := o.(types.StreamDict); ok {
				if err := loadEncodedStreamContent(c, ctx, &sd, false); err != nil {
					return fmt.Errorf("hint stream obj#%d: load cached content: %w", objNr, err)
				}
				entry.Object = sd
				return nil
			}
		}
		// Already dereferenced.
		logStream(entry.Object)
		updateBinaryTotalSize(ctx, o)
		if log.ReadEnabled() {
			log.Read.Printf("dereferenceObject: using cached object %d of %d\n<%s>\n", objNr, ctx.MaxObjNr+1, entry.Object)
		}
		return nil
	}

	if err := dereferenceAndLoad(c, ctx, objNr, entry, hints...); err != nil {
		return err
	}

	logStream(entry.Object)

	return nil
}

func dereferenceObjectsSorted(c context.Context, ctx *model.Context, hints ...*hintReadState) error {
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
		if err := dereferenceObject(c, ctx, objNr, hints...); err != nil {
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
		if err := model.ProcessRefCountsWithError(xRefTable, entry.Object); err != nil {
			return err
		}
	}

	return nil
}

func maxObjectNumber(table map[int]*model.XRefTableEntry) int {
	maxObjNr := -1
	for objNr := range table {
		if objNr > maxObjNr {
			maxObjNr = objNr
		}
	}
	return maxObjNr
}

// denseXRefTable reports whether at least half of the object-number range through maxObjNr is populated.
func denseXRefTable(table map[int]*model.XRefTableEntry, maxObjNr int) bool {
	return len(table) > 0 && maxObjNr >= 0 && maxObjNr/2 < len(table)
}

func dereferenceObjectsAscending(c context.Context, ctx *model.Context, hints ...*hintReadState) error {
	xRefTable := ctx.XRefTable
	maxObjNr := maxObjectNumber(xRefTable.Table)
	if !denseXRefTable(xRefTable.Table, maxObjNr) {
		return dereferenceObjectsSorted(c, ctx, hints...)
	}

	for objNr := 0; objNr <= maxObjNr; objNr++ {
		entry, found := xRefTable.Table[objNr]
		if !found {
			continue
		}
		if err := c.Err(); err != nil {
			return err
		}
		if err := dereferenceObjectEntry(c, ctx, objNr, entry, hints...); err != nil {
			return err
		}
	}

	for objNr := 0; objNr <= maxObjNr; objNr++ {
		entry, found := xRefTable.Table[objNr]
		if !found {
			continue
		}
		if entry.Free || entry.Compressed {
			continue
		}
		if err := c.Err(); err != nil {
			return err
		}
		if err := model.ProcessRefCountsWithError(xRefTable, entry.Object); err != nil {
			return err
		}
	}

	return nil
}

func validateHintStreamObject(ctx *model.Context, o types.Object, path string, depth int) error {
	if err := ctx.XRefTable.CheckRecursionDepth("hint stream dictionary", depth); err != nil {
		return err
	}
	switch o := o.(type) {
	case types.IndirectRef:
		return fmt.Errorf("%s: indirect reference not permitted", path)
	case types.Array:
		for i, o := range o {
			if err := validateHintStreamObject(ctx, o, fmt.Sprintf("%s array index %d", path, i), depth+1); err != nil {
				return err
			}
		}
	case types.Dict:
		for _, key := range slices.Sorted(maps.Keys(o)) {
			if err := validateHintStreamObject(ctx, o[key], fmt.Sprintf("%s key %s", path, key), depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateHintStreamDict(ctx *model.Context, d types.Dict, objNr int) error {
	for _, key := range slices.Sorted(maps.Keys(d)) {
		if key == "Length" {
			continue
		}
		if err := validateHintStreamObject(ctx, d[key], fmt.Sprintf("hint stream obj#%d entry %s", objNr, key), 0); err != nil {
			return err
		}
	}
	return nil
}

func hintStreamEntryForOffset(ctx *model.Context, offset int64) (int, *model.XRefTableEntry) {
	objNr := -1
	var selected *model.XRefTableEntry
	for candidate, entry := range ctx.Table {
		if entry == nil || entry.Offset == nil || *entry.Offset != offset {
			continue
		}
		if objNr < 0 || candidate < objNr {
			objNr = candidate
			selected = entry
		}
	}
	return objNr, selected
}

func validateHintStreamDictionaries(ctx *model.Context) error {
	if ctx.Read == nil || !ctx.Read.Linearized {
		return nil
	}
	for _, offset := range []*int64{ctx.OffsetPrimaryHintTable, ctx.OffsetOverflowHintTable} {
		if offset == nil {
			continue
		}
		objNr, entry := hintStreamEntryForOffset(ctx, *offset)
		if entry == nil {
			continue
		}
		sd, ok := entry.Object.(types.StreamDict)
		if ok {
			if err := validateHintStreamDict(ctx, sd.Dict, objNr); err != nil {
				return err
			}
		}
	}
	return nil
}

// Dereferences all objects including compressed objects from object streams.
func dereferenceObjects(c context.Context, ctx *model.Context, hints ...*hintReadState) error {
	if log.ReadEnabled() {
		log.Read.Println("dereferenceObjects: begin")
	}

	f := dereferenceObjectsAscending
	if log.StatsEnabled() {
		f = dereferenceObjectsSorted
	}

	if err := f(c, ctx, hints...); err != nil {
		repairOffset := errors.Is(err, model.ErrCorruptObjectOffset)
		repairMissing := ctx.XRefTable.ValidationMode == model.ValidationRelaxed && errors.Is(err, errUnregisteredObject)
		if !repairOffset && !repairMissing {
			return err
		}
		// Rebuilding replaces candidates with normally loaded/decrypted
		// entries. The old deferred ciphertext inventory no longer applies.
		hints = nil
		if err := bypassXrefSection(c, ctx, 0, err, 1); err != nil {
			return err
		}
		if err = f(c, ctx, hints...); err != nil {
			return err
		}
	}
	if len(hints) > 0 && hints[0] != nil {
		if err := finalizeHintStreams(c, ctx, hints[0]); err != nil {
			return err
		}
	}
	if err := validateHintStreamDictionaries(ctx); err != nil {
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
		defer log.Read.Println("identifyRootVersion: end")
	}

	// Try to get Version from Root.
	rootVersionStr, err := xRefTable.ParseRootVersion()
	if err != nil {
		return fmt.Errorf("root version: %w", err)
	}

	if rootVersionStr == nil {
		return nil
	}

	// Since PDF 1.4 the header version may be overridden by a Version entry in the catalog.
	if *xRefTable.HeaderVersion < model.V14 {
		if log.InfoEnabled() {
			log.Info.Printf("identifyRootVersion: PDF version is %s - will ignore root version: %s\n", xRefTable.HeaderVersion, *rootVersionStr)
		}
		return nil
	}

	// Validate version and save corresponding constant to xRefTable.
	rootVersion, err := model.PDFVersion(*rootVersionStr)
	if err != nil {
		if xRefTable.ValidationMode == model.ValidationStrict {
			return nil
		}
		rootVersion, err = model.PDFVersionRelaxed(*rootVersionStr)
		if err != nil {
			return nil
		}
	}

	xRefTable.RootVersion = &rootVersion

	return nil
}

// Parse all Objects including stream content from file and save to the corresponding xRefTableEntries.
// This includes processing of object streams and linearization dicts.
func dereferenceXRefTable(c context.Context, ctx *model.Context) error {
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

	hints, err := discoverHintStreams(c, ctx)
	if err != nil {
		return fmt.Errorf("identify linearization hints: %w", err)
	}

	// Prepare decompressed objects.
	if err := decodeObjectStreams(c, ctx); err != nil {
		return err
	}

	// For each xRefTableEntry assign a Object either by parsing from file or pointing to a decompressed object.
	if err := dereferenceObjects(c, ctx, hints); err != nil {
		return err
	}

	// Resolve the Catalog version after all objects are available as well.
	if err := identifyRootVersion(xRefTable); err != nil {
		return err
	}
	if err := materializeDirectInfoDict(xRefTable); err != nil {
		return err
	}

	if log.ReadEnabled() {
		log.Read.Println("dereferenceXRefTable: end")
	}

	return nil
}

func handleUnencryptedFile(ctx *model.Context) error {
	if ctx.Cmd == model.DECRYPT || ctx.Cmd == model.SETPERMISSIONS {
		return ErrNotEncrypted
	}

	if ctx.Cmd != model.ENCRYPT {
		return nil
	}

	// Encrypt subcommand found.

	if ctx.OwnerPW == "" {
		return ErrOwnerPasswordRequired
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
		return fmt.Errorf("validate permissions: %w", err)
	}

	if !ok {
		return fmt.Errorf("user password permissions: %w", errInvalidPermissions)
	}

	if ctx.OwnerPW == "" && ctx.UserPW == "" {
		return nil
	}

	// Double check minimum permissions for pdfcpu processing.
	if !hasNeededPermissions(ctx.Cmd, ctx.E) {
		return ErrPermissionDenied
	}

	return nil
}

func setupEncryptionKey(ctx *model.Context, d types.Dict) (err error) {
	if ctx.E, err = supportedEncryption(ctx, d); err != nil {
		return fmt.Errorf("parse encryption dictionary: %w", err)
	}

	if len(ctx.ID) == 0 {
		if ctx.XRefTable.ValidationMode == model.ValidationStrict {
			return fmt.Errorf("read encryption ID: %w", errMissingTrailerID)
		}
	} else {
		if ctx.E.ID, err = ctx.IDFirstElement(); err != nil {
			return fmt.Errorf("read encryption ID: %w", err)
		}
	}

	var ok bool

	//fmt.Printf("opw: <%s> upw: <%s> \n", ctx.OwnerPW, ctx.UserPW)

	// Validate the owner password aka. permissions/master password.
	if ok, err = validateOwnerPassword(ctx); err != nil {
		return fmt.Errorf("validate owner password: %w", err)
	}

	// If the owner password does not match we generally move on if the user password is correct
	// unless we need to insist on a correct owner password due to the specific command in progress.
	if !ok && needsOwnerAndUserPassword(ctx.Cmd) {
		return fmt.Errorf("%w with --opw", ErrOwnerPasswordRequired)
	}

	// Generally the owner password, which is also regarded as the master password or set permissions password
	// is sufficient for moving on. A password change is an exception since it requires both current passwords.
	if ok && !needsOwnerAndUserPassword(ctx.Cmd) {
		// AES256 Validate permissions
		if ok, err = validatePermissions(ctx); err != nil {
			return fmt.Errorf("validate permissions: %w", err)
		}
		if !ok {
			return fmt.Errorf("owner password permissions: %w", errInvalidPermissions)
		}
		return nil
	}

	// Validate the user password aka. document open password.
	if ok, err = validateUserPassword(ctx); err != nil {
		return fmt.Errorf("validate user password: %w", err)
	}
	if !ok {
		return ErrWrongPassword
	}

	//fmt.Printf("upw ok: %t\n", ok)

	return handlePermissions(ctx)
}

// bootstrapCatalogVersion reads only Catalog syntax before the encryption key
// exists. Failed reads are left to the normal dereference/xref-repair path.
func bootstrapCatalogVersion(c context.Context, ctx *model.Context) error {
	if ctx.Root == nil || ctx.RootVersion != nil {
		return nil
	}
	ref := ctx.Root
	entry, ok := ctx.Find(ref.ObjectNumber.Value())
	if !ok || entry == nil || entry.Offset == nil || entry.Generation == nil {
		return nil
	}
	root, end, stream, _, err := object(c, ctx, *entry.Offset, ref.ObjectNumber.Value(), *entry.Generation)
	if err != nil || stream >= 0 && (end < 0 || stream < end) {
		return nil // Preserve the normal parser and xref-repair error path.
	}
	d, ok := root.(types.Dict)
	if !ok {
		return nil
	}
	if version, ok := d.Find("Version"); ok {
		if ref, ok := version.(types.IndirectRef); ok {
			entry, found := ctx.XRefTable.FindTableEntryForIndRef(&ref)
			if found && entry != nil && entry.Offset != nil && entry.Generation != nil {
				name, end, stream, _, err := object(c, ctx, *entry.Offset, ref.ObjectNumber.Value(), *entry.Generation)
				if err == nil && (stream < 0 || end >= 0 && stream > end) {
					if name, ok := name.(types.Name); ok {
						d = types.Dict{"Version": name}
					}
				}
			}
		}
	}
	previous := ctx.RootDict
	ctx.RootDict = d
	err = identifyRootVersion(ctx.XRefTable)
	ctx.RootDict = previous
	return err
}

func checkForEncryption(c context.Context, ctx *model.Context) error {
	if err := requireContextWithXRefTable(ctx); err != nil {
		return err
	}
	indRef := ctx.Encrypt
	if indRef == nil {
		// This file is not encrypted.
		if err := handleUnencryptedFile(ctx); err != nil {
			return fmt.Errorf("encryption status: %w", err)
		}
		return nil
	}

	// This file is encrypted.

	if log.ReadEnabled() {
		log.Read.Printf("Encryption: %v\n", indRef)
	}

	if ctx.Cmd == model.BOOKLET ||
		ctx.Cmd == model.ENCRYPT ||
		ctx.Cmd == model.MERGEAPPEND ||
		ctx.Cmd == model.MERGECREATE ||
		ctx.Cmd == model.MERGECREATEZIP ||
		ctx.Cmd == model.ADDSIGNATURE {
		return fmt.Errorf("encryption status: %w", ErrEncrypted)
	}

	// Dereference encryptDict.
	objNr := indRef.ObjectNumber.Value()
	d, err := dereferencedDict(c, ctx, objNr)
	if err != nil {
		return fmt.Errorf("encryption dictionary obj#%d: %w", objNr, classifyEncryptionDictionaryError(err))
	}

	if log.ReadEnabled() {
		log.Read.Printf("%s\n", d)
	}
	if err := materializeEncryptionIntegers(c, ctx, d); err != nil {
		return fmt.Errorf("encryption dictionary obj#%d: %w", objNr, err)
	}

	// Resolve the Catalog Version before checking crypt-filter Length, without
	// caching undecrypted objects or resolving untrusted stream filters.
	if err := bootstrapCatalogVersion(c, ctx); err != nil {
		return err
	}

	// We need to decrypt this file in order to read it.
	if err := setupEncryptionKey(ctx, d); err != nil {
		return fmt.Errorf("encryption setup: %w", err)
	}

	return nil
}
