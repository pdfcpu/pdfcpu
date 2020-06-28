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

// Package api lets you integrate pdfcpu's operations into your Go backend.
//
// There are two api layers supporting all pdfcpu operations:
//  1) The file based layer (used by pdfcpu's cli)
//  2) The io.ReadSeeker/io.Writer based layer for backend integration.
//
// For any pdfcpu command there are two functions.
//
// The file based function always calls the io.ReadSeeker/io.Writer based function:
//  func CommandFile(inFile, outFile string, conf *pdf.Configuration) error
//  func Command(rs io.ReadSeeker, w io.Writer, conf *pdf.Configuration) error
//
// eg. for optimization:
//  func OptimizeFile(inFile, outFile string, conf *pdf.Configuration) error
//  func Optimize(rs io.ReadSeeker, w io.Writer, conf *pdf.Configuration) error
package api

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/validate"
	"github.com/pkg/errors"
)

func logOperationStats(ctx *pdf.Context, op string, durRead, durVal, durOpt, durWrite, durTotal float64) {
	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	pdf.TimingStats(op, durRead, durVal, durOpt, durWrite, durTotal)
	if ctx.Read.FileSize > 0 {
		ctx.Read.LogStats(ctx.Optimized)
		ctx.Write.LogStats()
	}
}

// ReadContext uses an io.ReadSeeker to build an internal structure holding its cross reference table aka the Context.
func ReadContext(rs io.ReadSeeker, conf *pdf.Configuration) (*pdf.Context, error) {
	return pdf.Read(rs, conf)
}

// ReadContextFile returns inFile's validated context.
func ReadContextFile(inFile string) (*pdf.Context, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	ctx, err := ReadContext(f, pdf.NewDefaultConfiguration())
	if err != nil {
		return nil, err
	}
	if err = validate.XRefTable(ctx.XRefTable); err != nil {
		return nil, err
	}
	return ctx, err
}

// PageCount returns rs's page count.
func PageCount(rs io.ReadSeeker, conf *pdf.Configuration) (int, error) {
	ctx, err := ReadContext(rs, conf)
	if err != nil {
		return 0, err
	}
	if err := ValidateContext(ctx); err != nil {
		return 0, err
	}
	return ctx.PageCount, nil
}

// PageCountFile returns inFile's page count.
func PageCountFile(inFile string) (int, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	return PageCount(f, pdf.NewDefaultConfiguration())
}

// PageDims returns a sorted slice of mediaBox dimensions for rs.
func PageDims(rs io.ReadSeeker, conf *pdf.Configuration) ([]pdf.Dim, error) {
	ctx, err := ReadContext(rs, conf)
	if err != nil {
		return nil, err
	}

	pd, err := ctx.PageDims()
	if err != nil {
		return nil, err
	}
	if len(pd) != ctx.PageCount {
		return nil, errors.New("pdfcpu: corrupt page dimensions")
	}

	return pd, nil
}

// PageDimsFile returns a sorted slice of mediaBox dimensions for inFile.
func PageDimsFile(inFile string) ([]pdf.Dim, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return PageDims(f, pdf.NewDefaultConfiguration())
}

// ValidateContext validates a PDF context.
func ValidateContext(ctx *pdf.Context) error {
	return validate.XRefTable(ctx.XRefTable)
}

// OptimizeContext optimizes a PDF context.
func OptimizeContext(ctx *pdf.Context) error {
	return pdf.OptimizeXRefTable(ctx)
}

// WriteContext writes a PDF context to w.
func WriteContext(ctx *pdf.Context, w io.Writer) error {
	if f, ok := w.(*os.File); ok {
		ctx.Write.Fp = f
	}
	ctx.Write.Writer = bufio.NewWriter(w)
	return pdf.Write(ctx)
}

// WriteContextFile writes a PDF context to outFile.
func WriteContextFile(ctx *pdf.Context, outFile string) error {
	f, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer f.Close()
	return WriteContext(ctx, f)
}

func readAndValidate(rs io.ReadSeeker, conf *pdf.Configuration, from1 time.Time) (ctx *pdf.Context, dur1, dur2 float64, err error) {
	if ctx, err = ReadContext(rs, conf); err != nil {
		return nil, 0, 0, err
	}

	dur1 = time.Since(from1).Seconds()

	if conf.ValidationMode == pdf.ValidationNone {
		// Bypass validation
		return ctx, 0, 0, nil
	}

	from2 := time.Now()

	if err = validate.XRefTable(ctx.XRefTable); err != nil {
		return nil, 0, 0, err
	}

	dur2 = time.Since(from2).Seconds()

	return ctx, dur1, dur2, nil
}

func readValidateAndOptimize(rs io.ReadSeeker, conf *pdf.Configuration, from1 time.Time) (ctx *pdf.Context, dur1, dur2, dur3 float64, err error) {
	ctx, dur1, dur2, err = readAndValidate(rs, conf, from1)
	if err != nil {
		return nil, 0, 0, 0, err
	}

	from3 := time.Now()

	if err = OptimizeContext(ctx); err != nil {
		return nil, 0, 0, 0, err
	}

	dur3 = time.Since(from3).Seconds()

	return ctx, dur1, dur2, dur3, nil
}

// Validate validates a PDF stream read from rs.
func Validate(rs io.ReadSeeker, conf *pdf.Configuration) error {
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	}
	conf.Cmd = pdf.VALIDATE

	if conf.ValidationMode == pdf.ValidationNone {
		return errors.New("pdfcpu: validate: mode ValidationNone not allowed")
	}

	from1 := time.Now()

	ctx, err := ReadContext(rs, conf)
	if err != nil {
		return err
	}

	dur1 := time.Since(from1).Seconds()
	from2 := time.Now()

	if err = ValidateContext(ctx); err != nil {
		s := ""
		if conf.ValidationMode == pdf.ValidationStrict {
			s = " (try -mode=relaxed)"
		}
		err = errors.Wrap(err, "validation error"+s)
	}

	dur2 := time.Since(from2).Seconds()
	dur := time.Since(from1).Seconds()

	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	pdf.ValidationTimingStats(dur1, dur2, dur)

	// at this stage: no binary breakup available!
	if ctx.Read.FileSize > 0 {
		ctx.Read.LogStats(ctx.Optimized)
	}

	return err
}

// ValidateFile validates inFile.
func ValidateFile(inFile string, conf *pdf.Configuration) error {
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	}

	if conf != nil && conf.ValidationMode == pdf.ValidationNone {
		return nil
	}

	log.CLI.Printf("validating(mode=%s) %s ...\n", conf.ValidationModeString(), inFile)

	f, err := os.Open(inFile)
	if err != nil {
		return err
	}

	defer f.Close()

	if err = Validate(f, conf); err != nil {
		return err
	}

	log.CLI.Println("validation ok")

	return nil
}

// Optimize reads a PDF stream from rs and writes the optimized PDF stream to w.
func Optimize(rs io.ReadSeeker, w io.Writer, conf *pdf.Configuration) error {
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
		conf.Cmd = pdf.OPTIMIZE
	}

	fromStart := time.Now()

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	fromWrite := time.Now()

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "write", durRead, durVal, durOpt, durWrite, durTotal)

	// For Optimize only.
	if ctx.StatsFileName != "" {
		err = pdf.AppendStatsFile(ctx)
		if err != nil {
			return errors.Wrap(err, "Write stats failed.")
		}
	}

	return nil
}

// OptimizeFile reads inFile and writes the optimized PDF to outFile.
// If outFile is not provided then inFile gets overwritten
// which leads to the same result as when inFile equals outFile.
func OptimizeFile(inFile, outFile string, conf *pdf.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		log.CLI.Printf("writing %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", inFile)
	}

	if f2, err = os.Create(tmpFile); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			if err = os.Rename(tmpFile, inFile); err != nil {
				return
			}
		}
	}()

	return Optimize(f1, f2, conf)
}

// EncryptFile encrypts inFile and writes the result to outFile.
// A configuration containing the current passwords is required.
func EncryptFile(inFile, outFile string, conf *pdf.Configuration) error {
	if conf == nil {
		return errors.New("pdfcpu: missing configuration for encryption")
	}
	conf.Cmd = pdf.ENCRYPT
	return OptimizeFile(inFile, outFile, conf)
}

// DecryptFile decrypts inFile and writes the result to outFile.
// A configuration containing the current passwords is required.
func DecryptFile(inFile, outFile string, conf *pdf.Configuration) error {
	if conf == nil {
		return errors.New("pdfcpu: missing configuration for decryption")
	}
	conf.Cmd = pdf.DECRYPT
	return OptimizeFile(inFile, outFile, conf)
}

// ChangeUserPasswordFile reads inFile, changes the user password and writes the result to outFile.
// A configuration containing the current passwords is required.
func ChangeUserPasswordFile(inFile, outFile string, pwOld, pwNew string, conf *pdf.Configuration) error {
	if conf == nil {
		return errors.New("pdfcpu: missing configuration for change user password")
	}
	conf.Cmd = pdf.CHANGEUPW
	conf.UserPW = pwOld
	conf.UserPWNew = &pwNew
	return OptimizeFile(inFile, outFile, conf)
}

// ChangeOwnerPasswordFile reads inFile, changes the user password and writes the result to outFile.
// A configuration containing the current passwords is required.
func ChangeOwnerPasswordFile(inFile, outFile string, pwOld, pwNew string, conf *pdf.Configuration) error {
	if conf == nil {
		return errors.New("pdfcpu: missing configuration for change owner password")
	}
	conf.Cmd = pdf.CHANGEOPW
	conf.OwnerPW = pwOld
	conf.OwnerPWNew = &pwNew
	return OptimizeFile(inFile, outFile, conf)
}

// ListPermissions returns a list of user access permissions.
func ListPermissions(rs io.ReadSeeker, conf *pdf.Configuration) ([]string, error) {
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	}
	conf.Cmd = pdf.LISTPERMISSIONS

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return nil, err
	}

	fromList := time.Now()
	list := pdf.Permissions(ctx)

	durList := time.Since(fromList).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	pdf.TimingStats("list permissions", durRead, durVal, durOpt, durList, durTotal)

	return list, nil
}

// ListPermissionsFile returns a list of user access permissions for inFile.
func ListPermissionsFile(inFile string, conf *pdf.Configuration) ([]string, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}

	defer func() {
		f.Close()
	}()

	return ListPermissions(f, conf)
}

// SetPermissions sets user access permissions.
// inFile has to be encrypted.
// A configuration containing the current passwords is required.
func SetPermissions(rs io.ReadSeeker, w io.Writer, conf *pdf.Configuration) error {
	if conf == nil {
		return errors.New("pdfcpu: missing configuration for setting permissions")
	}
	conf.Cmd = pdf.SETPERMISSIONS

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	fromWrite := time.Now()
	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// SetPermissionsFile sets inFile's user access permissions.
// inFile has to be encrypted.
// A configuration containing the current passwords is required.
func SetPermissionsFile(inFile, outFile string, conf *pdf.Configuration) (err error) {
	if conf == nil {
		return errors.New("pdfcpu: missing configuration for setting permissions")
	}

	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		log.CLI.Printf("writing %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", inFile)
	}
	if f2, err = os.Create(tmpFile); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			if err = os.Rename(tmpFile, inFile); err != nil {
				return
			}
		}
	}()

	return SetPermissions(f1, f2, conf)
}

func selectedPageRange(from, thru int) []int {
	s := make([]int, thru-from+1)
	for i := 0; i < len(s); i++ {
		s[i] = from + i
	}
	return s
}

func spanFileName(fileName string, from, thru int) string {
	baseFileName := filepath.Base(fileName)
	fn := strings.TrimSuffix(baseFileName, ".pdf")
	fn = fn + "_" + strconv.Itoa(from)
	if from == thru {
		return fn + ".pdf"
	}
	return fn + "-" + strconv.Itoa(thru) + ".pdf"
}

func writeSpan(ctx *pdf.Context, from, thru int, outDir, fileName string, forBookmark bool) error {

	selectedPages := selectedPageRange(from, thru)

	ctxDest, err := pdf.CreateContextWithXRefTable(nil, pdf.PaperSize["A4"])
	if err != nil {
		return err
	}

	usePgCache := false
	if err := pdf.AddPages(ctx, ctxDest, selectedPages, usePgCache); err != nil {
		return err
	}

	w := ctxDest.Write
	w.DirName = outDir
	w.FileName = fileName + ".pdf"
	if !forBookmark {
		w.FileName = spanFileName(fileName, from, thru)
		//log.CLI.Printf("writing to: <%s>\n", w.FileName)
	}

	return pdf.Write(ctxDest)
}

type bookmark struct {
	title    string
	pageFrom int
	pageThru int // We assume, pageThru has to be at least pageFrom and reaches until before pageFrom of the next bookmark.
}

func dereferenceDestinationArray(ctx *pdf.Context, key string) (pdf.Array, error) {
	o, ok := ctx.Names["Dests"].Value(key)
	if !ok {
		return nil, errors.New("Corrupt named destination")
	}
	return ctx.DereferenceArray(o)
}

func positionToOutlineTreeLevel(ctx *pdf.Context) (pdf.Dict, *pdf.IndirectRef, error) {

	// Load Dests nametree.
	if err := ctx.LocateNameTree("Dests", false); err != nil {
		return nil, nil, err
	}

	ir, err := ctx.Outlines()
	if err != nil {
		return nil, nil, err
	}
	if ir == nil {
		return nil, nil, errors.New("No bookmarks available")
	}

	d, err := ctx.DereferenceDict(*ir)
	if err != nil {
		return nil, nil, err
	}
	if d == nil {
		return nil, nil, errors.New("No bookmarks available")
	}

	first := d.IndirectRefEntry("First")
	last := d.IndirectRefEntry("Last")

	// We consider Bookmarks at level 1 or 2 only.
	for *first == *last {
		//fmt.Println("first == last")
		if d, err = ctx.DereferenceDict(*first); err != nil {
			return nil, nil, err
		}
		first = d.IndirectRefEntry("First")
		last = d.IndirectRefEntry("Last")
	}

	return d, first, nil
}

func bookmarksForOutlineLevel1(ctx *pdf.Context) ([]bookmark, error) {

	d, first, err := positionToOutlineTreeLevel(ctx)
	if err != nil {
		return nil, err
	}

	bms := []bookmark{}

	// Process linked list of outline items.
	for ir := first; ir != nil; ir = d.IndirectRefEntry("Next") {

		//objNr := ir.ObjectNumber.Value()
		if d, err = ctx.DereferenceDict(*ir); err != nil {
			return nil, err
		}

		title, _ := pdf.Text(d["Title"])
		//fmt.Printf("bookmark obj:%d title:%s\n", objNr, title)

		dest, found := d["Dest"]
		if !found {
			return nil, errors.New("No destination based bookmarks available")
		}

		var pageIndRef pdf.IndirectRef

		dest, _ = ctx.Dereference(dest)

		switch dest := dest.(type) {

		case pdf.Name:
			//fmt.Printf("dest is Name: %s\n", dest.Value())
			arr, err := dereferenceDestinationArray(ctx, dest.Value())
			if err != nil {
				return nil, err
			}
			pageIndRef = arr[0].(pdf.IndirectRef)

		case pdf.StringLiteral:
			//fmt.Printf("dest is StringLiteral: %s\n", dest.Value())
			arr, err := dereferenceDestinationArray(ctx, dest.Value())
			if err != nil {
				return nil, err
			}
			pageIndRef = arr[0].(pdf.IndirectRef)

		case pdf.HexLiteral:
			//fmt.Printf("dest is HexLiteral: %s\n", dest.Value())
			arr, err := dereferenceDestinationArray(ctx, dest.Value())
			if err != nil {
				return nil, err
			}
			pageIndRef = arr[0].(pdf.IndirectRef)

		case pdf.Array:
			pageIndRef = dest[0].(pdf.IndirectRef)

		}

		pageFrom, err := ctx.PageNumber(pageIndRef.ObjectNumber.Value())
		if err != nil {
			return nil, err
		}

		if len(bms) > 0 {
			if pageFrom > bms[len(bms)-1].pageFrom {
				bms[len(bms)-1].pageThru = pageFrom - 1
			} else {
				bms[len(bms)-1].pageThru = bms[len(bms)-1].pageFrom
			}
		}
		bms = append(bms, bookmark{title: title, pageFrom: pageFrom})
	}

	return bms, nil
}

func writePDFSequenceSplitAlongBookmarks(ctx *pdf.Context, outDir string) error {

	bms, err := bookmarksForOutlineLevel1(ctx)
	if err != nil {
		return err
	}

	for _, bm := range bms {
		fileName := bm.title
		from := bm.pageFrom
		thru := bm.pageThru
		if thru == 0 {
			thru = ctx.PageCount
		}
		forBookmark := true
		if err := writeSpan(ctx, from, thru, outDir, fileName, forBookmark); err != nil {
			return err
		}
	}

	return nil
}

func writePDFSequence(ctx *pdf.Context, span int, outDir, fileName string) error {

	if span == 0 {
		return writePDFSequenceSplitAlongBookmarks(ctx, outDir)
	}

	forBookmark := false

	for i := 0; i < ctx.PageCount/span; i++ {

		start := i * span
		from := start + 1
		thru := start + span

		if err := writeSpan(ctx, from, thru, outDir, fileName, forBookmark); err != nil {
			return err
		}

	}

	// A possible last file that has less than span pages.
	if ctx.PageCount%span > 0 {

		start := (ctx.PageCount / span) * span
		from := start + 1
		thru := ctx.PageCount

		if err := writeSpan(ctx, from, thru, outDir, fileName, forBookmark); err != nil {
			return err
		}

	}

	return nil
}

// Split generates a sequence of PDF files in outDir for the PDF stream read from rs obeying given split span.
// If span == 1 splitting results in single page PDFs.
// If span == 0 we split along given bookmarks (level 1 only).
func Split(rs io.ReadSeeker, outDir, fileName string, span int, conf *pdf.Configuration) error {
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	}
	conf.Cmd = pdf.SPLIT

	fromStart := time.Now()

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	fromWrite := time.Now()

	if err = writePDFSequence(ctx, span, outDir, fileName); err != nil {
		return err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "split", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// SplitFile generates a sequence of PDF files in outDir for inFile obeying given split span.
// The default span 1 creates a sequence of single page PDFs.
func SplitFile(inFile, outDir string, span int, conf *pdf.Configuration) error {
	f, err := os.Open(inFile)
	if err != nil {
		return err
	}
	log.CLI.Printf("splitting %s to %s/...\n", inFile, outDir)

	defer func() {
		if err != nil {
			f.Close()
			return
		}
		err = f.Close()
	}()

	return Split(f, outDir, filepath.Base(inFile), span, conf)
}

// Trim generates a trimmed version of rs
// containing all selected pages and writes the result to w.
func Trim(rs io.ReadSeeker, w io.Writer, selectedPages []string, conf *pdf.Configuration) error {
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	}
	conf.Cmd = pdf.TRIM

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	fromWrite := time.Now()

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, false)
	if err != nil {
		return err
	}

	ctx.Write.SelectedPages = pages
	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "trim, write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// TrimFile generates a trimmed version of inFile
// containing all selected pages and writes the result to outFile.
func TrimFile(inFile, outFile string, selectedPages []string, conf *pdf.Configuration) (err error) {
	// if conf == nil {
	// 	conf = pdf.NewDefaultConfiguration()
	// }

	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		log.CLI.Printf("writing %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", inFile)
	}
	if f2, err = os.Create(tmpFile); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			if err = os.Rename(tmpFile, inFile); err != nil {
				return
			}
		}
	}()

	return Trim(f1, f2, selectedPages, conf)
}

// Rotate rotates selected pages of rs clockwise by rotation degrees and writes the result to w.
func Rotate(rs io.ReadSeeker, w io.Writer, rotation int, selectedPages []string, conf *pdf.Configuration) error {
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	}
	conf.Cmd = pdf.ROTATE

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	from := time.Now()
	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return err
	}

	if err = pdf.RotatePages(ctx, pages, rotation); err != nil {
		return err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	durStamp := time.Since(from).Seconds()
	fromWrite := time.Now()

	if conf.ValidationMode != pdf.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	durWrite := durStamp + time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "rotate, write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// RotateFile rotates selected pages of inFile clockwise by rotation degrees and writes the result to outFile.
func RotateFile(inFile, outFile string, rotation int, selectedPages []string, conf *pdf.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		log.CLI.Printf("writing %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", inFile)
	}
	if f2, err = os.Create(tmpFile); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			if err = os.Rename(tmpFile, inFile); err != nil {
				return
			}
		}
	}()

	return Rotate(f1, f2, rotation, selectedPages, conf)
}

// WatermarkContext applies a watermark for selected pages,
func WatermarkContext(ctx *pdf.Context, selectedPages pdf.IntSet, wm *pdf.Watermark) error {
	return pdf.AddWatermarks(ctx, selectedPages, wm)
}

// AddWatermarks adds watermarks to all pages selected in rs and writes the result to w.
// Called by AddWatermarksFile or manually by passing in wm created by
// calling TextWatermark, ImageWatermark or PDFWatermark.
func AddWatermarks(rs io.ReadSeeker, w io.Writer, selectedPages []string, wm *pdf.Watermark, conf *pdf.Configuration) error {
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	}
	conf.Cmd = pdf.ADDWATERMARKS

	if wm == nil {
		return errors.New("pdfcpu: missing watermark configuration")
	}

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	from := time.Now()
	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return err
	}

	if err = pdf.AddWatermarks(ctx, pages, wm); err != nil {
		return err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != pdf.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	durStamp := time.Since(from).Seconds()
	fromWrite := time.Now()

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	durWrite := durStamp + time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "watermark, write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// AddWatermarksFile adds watermarks to all selected pages of inFile and writes the result to outFile.
// Called by:
// AddTextWatermarksFile, AddImageWatermarksFile, AddPDFWatermarksFile
// UpdateTextWatermarksFile, UpdateImageWatermarksFile, UpdatePDFWatermarksFile
func AddWatermarksFile(inFile, outFile string, selectedPages []string, wm *pdf.Watermark, conf *pdf.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		log.CLI.Printf("writing %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", inFile)
	}
	if f2, err = os.Create(tmpFile); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			if err = os.Rename(tmpFile, inFile); err != nil {
				return
			}
		}
	}()

	return AddWatermarks(f1, f2, selectedPages, wm, conf)
}

// RemoveWatermarks removes watermarks from all pages selected in rs and writes the result to w.
func RemoveWatermarks(rs io.ReadSeeker, w io.Writer, selectedPages []string, conf *pdf.Configuration) error {
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	}
	conf.Cmd = pdf.ADDWATERMARKS

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	from := time.Now()
	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return err
	}

	if err = pdf.RemoveWatermarks(ctx, pages); err != nil {
		return err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != pdf.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	durStamp := time.Since(from).Seconds()
	fromWrite := time.Now()

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	durWrite := durStamp + time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "watermark, write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// RemoveWatermarksFile removes watermarks from all selected pages of inFile and writes the result to outFile.
func RemoveWatermarksFile(inFile, outFile string, selectedPages []string, conf *pdf.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		log.CLI.Printf("writing %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", inFile)
	}
	if f2, err = os.Create(tmpFile); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			if err = os.Rename(tmpFile, inFile); err != nil {
				return
			}
		}
	}()

	return RemoveWatermarks(f1, f2, selectedPages, conf)
}

// HasWatermarks checks rs for watermarks.
func HasWatermarks(rs io.ReadSeeker, conf *pdf.Configuration) (bool, error) {
	ctx, err := ReadContext(rs, conf)
	if err != nil {
		return false, err
	}
	if err := pdf.DetectWatermarks(ctx); err != nil {
		return false, err
	}

	return ctx.Watermarked, nil
}

// HasWatermarksFile checks inFile for watermarks.
func HasWatermarksFile(inFile string, conf *pdf.Configuration) (bool, error) {
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	}

	f, err := os.Open(inFile)
	if err != nil {
		return false, err
	}

	defer f.Close()

	return HasWatermarks(f, conf)
}

// NUp rearranges PDF pages or images into page grids and writes the result to w.
// Either rs or imgFiles will be used.
func NUp(rs io.ReadSeeker, w io.Writer, imgFiles, selectedPages []string, nup *pdf.NUp, conf *pdf.Configuration) error {
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	}
	conf.Cmd = pdf.NUP

	log.Info.Printf("%s", nup)

	var (
		ctx *pdf.Context
		err error
	)

	if nup.ImgInputFile {

		if ctx, err = pdf.NUpFromImage(conf, imgFiles, nup); err != nil {
			return err
		}

	} else {

		if ctx, _, _, err = readAndValidate(rs, conf, time.Now()); err != nil {
			return err
		}

		if err := ctx.EnsurePageCount(); err != nil {
			return err
		}

		pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true)
		if err != nil {
			return err
		}

		// New pages get added to ctx while old pages get deleted.
		// This way we avoid migrating objects between contexts.
		if err = pdf.NUpFromPDF(ctx, pages, nup); err != nil {
			return err
		}

	}

	if conf.ValidationMode != pdf.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	return nil
}

// NUpFile rearranges PDF pages or images into page grids and writes the result to outFile.
func NUpFile(inFiles []string, outFile string, selectedPages []string, nup *pdf.NUp, conf *pdf.Configuration) (err error) {
	var f1, f2 *os.File

	if !nup.ImgInputFile {
		// Nup from a PDF page.
		if f1, err = os.Open(inFiles[0]); err != nil {
			return err
		}
	}

	if f2, err = os.Create(outFile); err != nil {
		return err
	}
	log.CLI.Printf("writing %s...\n", outFile)

	defer func() {
		if err != nil {
			if f1 != nil {
				f1.Close()
			}
			f2.Close()
			return
		}
		if f1 != nil {
			if err = f1.Close(); err != nil {
				return
			}
		}
		err = f2.Close()
		return

	}()

	return NUp(f1, f2, inFiles, selectedPages, nup, conf)
}

// ImportImages appends PDF pages containing images to rs and writes the result to w.
// If rs == nil a new PDF file will be written to w.
func ImportImages(rs io.ReadSeeker, w io.Writer, imgs []io.Reader, imp *pdf.Import, conf *pdf.Configuration) error {
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	}
	conf.Cmd = pdf.IMPORTIMAGES

	if imp == nil {
		imp = pdf.DefaultImportConfig()
	}

	var (
		ctx *pdf.Context
		err error
	)

	if rs != nil {
		ctx, _, _, err = readAndValidate(rs, conf, time.Now())
	} else {
		ctx, err = pdf.CreateContextWithXRefTable(conf, imp.PageDim)
	}
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	pagesIndRef, err := ctx.Pages()
	if err != nil {
		return err
	}

	// This is the page tree root.
	pagesDict, err := ctx.DereferenceDict(*pagesIndRef)
	if err != nil {
		return err
	}

	for _, r := range imgs {

		indRef, err := pdf.NewPageForImage(ctx.XRefTable, r, pagesIndRef, imp)
		if err != nil {
			return err
		}

		if err = pdf.AppendPageTree(indRef, 1, pagesDict); err != nil {
			return err
		}

		ctx.PageCount++
	}

	if conf.ValidationMode != pdf.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	return nil
}

func fileExists(filename string) bool {
	f, err := os.Open(filename)
	defer f.Close()
	return err == nil
}

// ImportImagesFile appends PDF pages containing images to outFile which will be created if necessary.
func ImportImagesFile(imgFiles []string, outFile string, imp *pdf.Import, conf *pdf.Configuration) (err error) {
	var f1, f2 *os.File

	rs := io.ReadSeeker(nil)
	f1 = nil
	tmpFile := outFile
	if fileExists(outFile) {
		if f1, err = os.Open(outFile); err != nil {
			return err
		}
		rs = f1
		tmpFile += ".tmp"
		log.CLI.Printf("appending to %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", outFile)
	}

	rc := make([]io.ReadCloser, len(imgFiles))
	rr := make([]io.Reader, len(imgFiles))
	for i, fn := range imgFiles {
		f, err := os.Open(fn)
		if err != nil {
			return err
		}
		rc[i] = f
		rr[i] = bufio.NewReader(f)
	}

	if f2, err = os.Create(tmpFile); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			if f1 != nil {
				f1.Close()
				os.Remove(tmpFile)
			}
			for _, f := range rc {
				f.Close()
			}
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if f1 != nil {
			if err = f1.Close(); err != nil {
				return
			}
			if err = os.Rename(tmpFile, outFile); err != nil {
				return
			}
		}
		for _, f := range rc {
			if err := f.Close(); err != nil {
				return
			}
		}
	}()

	return ImportImages(rs, f2, rr, imp, conf)
}

// InsertPages inserts a blank page before or after every page selected of rs and writes the result to w.
func InsertPages(rs io.ReadSeeker, w io.Writer, selectedPages []string, before bool, conf *pdf.Configuration) error {
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	}
	conf.Cmd = pdf.INSERTPAGESAFTER
	if before {
		conf.Cmd = pdf.INSERTPAGESBEFORE
	}

	fromStart := time.Now()
	ctx, _, _, _, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return err
	}

	if err = ctx.InsertBlankPages(pages, before); err != nil {
		return err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	if conf.ValidationMode != pdf.ValidationNone {
		if err = ValidateContext(ctx); err != nil {
			return err
		}
	}

	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	return nil
}

// InsertPagesFile inserts a blank page before or after every inFile page selected and writes the result to w.
func InsertPagesFile(inFile, outFile string, selectedPages []string, before bool, conf *pdf.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		log.CLI.Printf("writing %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", inFile)
	}
	if f2, err = os.Create(tmpFile); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			if err = os.Rename(tmpFile, inFile); err != nil {
				return
			}
		}
	}()

	return InsertPages(f1, f2, selectedPages, before, conf)
}

// RemovePages removes selected pages from rs and writes the result to w.
func RemovePages(rs io.ReadSeeker, w io.Writer, selectedPages []string, conf *pdf.Configuration) error {
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	}
	conf.Cmd = pdf.REMOVEPAGES

	fromStart := time.Now()
	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	fromWrite := time.Now()

	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, false)
	if err != nil {
		return err
	}

	// ctx.Pagecount gets set during validation.
	if len(pages) >= ctx.PageCount {
		return errors.New("pdfcpu: operation invalid")
	}

	ctx.Write.SelectedPages = pages
	if err = WriteContext(ctx, w); err != nil {
		return err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "remove pages, write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// RemovePagesFile removes selected inFile pages and writes the result to outFile..
func RemovePagesFile(inFile, outFile string, selectedPages []string, conf *pdf.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		log.CLI.Printf("writing %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", inFile)
	}
	if f2, err = os.Create(tmpFile); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			if err = os.Rename(tmpFile, inFile); err != nil {
				return
			}
		}
	}()

	return RemovePages(f1, f2, selectedPages, conf)
}

// appendTo appends inFile to ctxDest's page tree.
func appendTo(rs io.ReadSeeker, ctxDest *pdf.Context) error {
	ctxSource, _, _, err := readAndValidate(rs, ctxDest.Configuration, time.Now())
	if err != nil {
		return err
	}

	// Merge the source context into the dest context.
	return pdf.MergeXRefTables(ctxSource, ctxDest)
}

// ReadSeekerCloser combines io.ReadSeeker and io.Closer
type ReadSeekerCloser interface {
	io.ReadSeeker
	io.Closer
}

// Merge merges a sequence of PDF streams and writes the result to w.
func Merge(rsc []io.ReadSeeker, w io.Writer, conf *pdf.Configuration) error {
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	}
	conf.Cmd = pdf.MERGECREATE

	ctxDest, _, _, err := readAndValidate(rsc[0], conf, time.Now())
	if err != nil {
		return err
	}

	ctxDest.EnsureVersionForWriting()

	// Repeatedly merge files into fileDest's xref table.
	for _, f := range rsc[1:] {
		err = appendTo(f, ctxDest)
		if err != nil {
			return err
		}
	}

	if err = OptimizeContext(ctxDest); err != nil {
		return err
	}

	if conf.ValidationMode != pdf.ValidationNone {
		if err = ValidateContext(ctxDest); err != nil {
			return err
		}
	}

	return WriteContext(ctxDest, w)
}

// MergeCreateFile merges a sequence of inFiles and writes the result to outFile.
// This operation corresponds to file concatenation in the order specified by inFiles.
// The first entry of inFiles serves as the destination context where all remaining files get merged into.
func MergeCreateFile(inFiles []string, outFile string, conf *pdf.Configuration) error {
	ff := []*os.File(nil)
	for _, f := range inFiles {
		log.CLI.Println(f)
		f, err := os.Open(f)
		if err != nil {
			return err
		}
		ff = append(ff, f)
	}
	f, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			f.Close()
			for _, f := range ff {
				f.Close()
			}
		}
		if err = f.Close(); err != nil {
			return
		}
		for _, f := range ff {
			if err = f.Close(); err != nil {
				return
			}
		}
	}()

	rs := make([]io.ReadSeeker, len(ff))
	for i, f := range ff {
		rs[i] = f
	}

	log.CLI.Printf("writing %s...\n", outFile)
	return Merge(rs, f, conf)
}

// MergeAppendFile merges a sequence of inFiles and writes the result to outFile.
// This operation corresponds to file concatenation in the order specified by inFiles.
// If outFile already exists, inFiles will be appended.
func MergeAppendFile(inFiles []string, outFile string, conf *pdf.Configuration) (err error) {
	var f1, f2 *os.File
	tmpFile := outFile
	if fileExists(outFile) {
		if f1, err = os.Open(outFile); err != nil {
			return err
		}
		tmpFile += ".tmp"
		log.CLI.Printf("appending to %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", outFile)
	}

	if f2, err = os.Create(tmpFile); err != nil {
		return err
	}

	ff := []*os.File(nil)
	if f1 != nil {
		ff = append(ff, f1)
	}
	for _, f := range inFiles {
		log.CLI.Println(f)
		f, err := os.Open(f)
		if err != nil {
			return err
		}
		ff = append(ff, f)
	}

	defer func() {
		if err != nil {
			f2.Close()
			if f1 != nil {
				os.Remove(tmpFile)
			}
			for _, f := range ff {
				f.Close()
			}
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if f1 != nil {
			if err = os.Rename(tmpFile, outFile); err != nil {
				return
			}
		}
		for _, f := range ff {
			if err = f.Close(); err != nil {
				return
			}
		}
	}()

	rss := make([]io.ReadSeeker, len(ff))
	for i, f := range ff {
		rss[i] = f
	}

	return Merge(rss, f2, conf)
}

// Info returns information about rs.
func Info(rs io.ReadSeeker, conf *pdf.Configuration) ([]string, error) {
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	}
	ctx, _, _, err := readAndValidate(rs, conf, time.Now())
	if err != nil {
		return nil, err
	}
	if err := pdf.DetectWatermarks(ctx); err != nil {
		return nil, err
	}
	return ctx.InfoDigest()
}

// InfoFile returns information about inFile.
func InfoFile(inFile string, conf *pdf.Configuration) ([]string, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Info(f, conf)
}

func isSupportedFontFile(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".gob")
}

// ListFonts returns a list of supported fonts.
func ListFonts() ([]string, error) {

	// Get list of PDF core fonts.
	coreFonts := font.CoreFontNames()
	for i, s := range coreFonts {
		coreFonts[i] = "  " + s
	}
	sort.Strings(coreFonts)

	sscf := []string{"Corefonts:"}
	sscf = append(sscf, coreFonts...)

	// Get installed fonts from pdfcpu config dir in users home dir
	userFonts := font.UserFontNames()
	for i, s := range userFonts {
		userFonts[i] = "  " + s
	}
	sort.Strings(userFonts)
	ssuf := []string{"Userfonts:"}
	ssuf = append(ssuf, userFonts...)

	sscf = append(sscf, "")
	return append(sscf, ssuf...), nil
}

// InstallFonts installs true type fonts for embedding.
func InstallFonts(fileNames []string) error {
	fontDir, err := font.Dir()
	if err != nil {
		return err
	}
	log.CLI.Printf("installing to %s...", fontDir)
	for _, fn := range fileNames {
		switch filepath.Ext(fn) {
		case ".ttf":
			log.CLI.Println(filepath.Base(fn))
			if err := font.InstallTrueTypeFont(fontDir, fn); err != nil {
				return err
			}
		}
	}
	return nil
}

// Collect creates a custom PDF page sequence for selected pages of rs and writes the result to w.
func Collect(rs io.ReadSeeker, w io.Writer, selectedPages []string, conf *pdf.Configuration) error {
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	}
	conf.Cmd = pdf.COLLECT

	fromStart := time.Now()
	ctxSource, _, _, _, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	if err := ctxSource.EnsurePageCount(); err != nil {
		return err
	}

	pages, err := PagesForPageCollection(ctxSource.PageCount, selectedPages)
	if err != nil {
		return err
	}

	ctxDest, err := pdf.CollectPages(ctxSource, pages)
	if err != nil {
		return err
	}

	if conf.ValidationMode != pdf.ValidationNone {
		if err = ValidateContext(ctxDest); err != nil {
			return err
		}
	}

	return WriteContext(ctxDest, w)
}

// CollectFile creates a custom PDF page sequence for inFile and writes the result to outFile.
func CollectFile(inFile, outFile string, selectedPages []string, conf *pdf.Configuration) (err error) {
	var f1, f2 *os.File

	if f1, err = os.Open(inFile); err != nil {
		return err
	}

	tmpFile := inFile + ".tmp"
	if outFile != "" && inFile != outFile {
		tmpFile = outFile
		log.CLI.Printf("writing %s...\n", outFile)
	} else {
		log.CLI.Printf("writing %s...\n", inFile)
	}
	if f2, err = os.Create(tmpFile); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			f2.Close()
			f1.Close()
			os.Remove(tmpFile)
			return
		}
		if err = f2.Close(); err != nil {
			return
		}
		if err = f1.Close(); err != nil {
			return
		}
		if outFile == "" || inFile == outFile {
			if err = os.Rename(tmpFile, inFile); err != nil {
				return
			}
		}
	}()

	return Collect(f1, f2, selectedPages, conf)
}

// GetPermissions returns the permissions for rs.
func GetPermissions(rs io.ReadSeeker, conf *pdf.Configuration) (*int16, error) {
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	}
	ctx, _, _, err := readAndValidate(rs, conf, time.Now())
	if err != nil {
		return nil, err
	}
	if ctx.E == nil {
		// Full access - permissions don't apply.
		return nil, nil
	}
	p := int16(ctx.E.P)
	return &p, nil
}

// GetPermissionsFile returns the permissions for inFile.
func GetPermissionsFile(inFile string, conf *pdf.Configuration) (*int16, error) {
	f, err := os.Open(inFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return GetPermissions(f, conf)
}
