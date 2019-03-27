/*
	Copyright 2018 The pdfcpu Authors.
f
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

// Package api provides support for interacting with pdf.
package api

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hhrutter/pdfcpu/pkg/log"
	pdf "github.com/hhrutter/pdfcpu/pkg/pdfcpu"
	"github.com/hhrutter/pdfcpu/pkg/pdfcpu/validate"

	"github.com/pkg/errors"
)

func stringSet(slice []string) pdf.StringSet {

	strSet := pdf.StringSet{}

	if slice == nil {
		return strSet
	}

	for _, s := range slice {
		strSet[s] = true
	}

	return strSet
}

// ReadContext uses an io.Readseeker to build an internal structure holding its cross reference table aka the Context.
func ReadContext(rs io.ReadSeeker, fileIn string, fileSize int64, config *pdf.Configuration) (*pdf.Context, error) {
	return pdf.Read(rs, fileIn, fileSize, config)
}

// ValidateContext validates a PDF context.
func ValidateContext(ctx *pdf.Context) error {
	return validate.XRefTable(ctx.XRefTable)
}

// OptimizeContext optimizes a PDF context.
func OptimizeContext(ctx *pdf.Context) error {
	return pdf.OptimizeXRefTable(ctx)
}

// WriteContext writes a PDF context.
func WriteContext(ctx *pdf.Context, w io.Writer) error {
	ctx.Write.Writer = bufio.NewWriter(w)
	return pdf.Write(ctx)
}

// MergeContexts merges a sequence of PDF's represented by a slice of ReadSeekerCloser.
func MergeContexts(rsc []pdf.ReadSeekerCloser, config *pdf.Configuration) (*pdf.Context, error) {

	ctxDest, err := ReadContext(rsc[0], "", 0, config)
	if err != nil {
		return nil, err
	}

	err = ValidateContext(ctxDest)
	if err != nil {
		return nil, err
	}

	if ctxDest.XRefTable.Version() < pdf.V15 {
		v, _ := pdf.PDFVersion("1.5")
		ctxDest.XRefTable.RootVersion = &v
		log.Stats.Println("Ensure V1.5 for writing object & xref streams")
	}

	// Merge in all readSeekerWriters.
	for _, r := range rsc[1:] {

		ctxSource, err := ReadContext(r, "", 0, config)
		if err != nil {
			return nil, err
		}

		err = ValidateContext(ctxSource)
		if err != nil {
			return nil, err
		}

		// Merge the source context into the dest context.
		//log.API.Println("merging in another readSeekerCloser...")
		err = pdf.MergeXRefTables(ctxSource, ctxDest)
		if err != nil {
			return nil, err
		}

	}

	err = OptimizeContext(ctxDest)
	if err != nil {
		return nil, err
	}

	err = ValidateContext(ctxDest)

	return ctxDest, err
}

// ReadContextFromFile reads in a PDF file and builds an internal structure holding its cross reference table aka the Context.
func ReadContextFromFile(fileIn string, config *pdf.Configuration) (*pdf.Context, error) {
	return pdf.ReadFile(fileIn, config)
}

// Validate validates a PDF file against ISO-32000-1:2008.
func Validate(cmd *Command) ([]string, error) {

	config := cmd.Config
	fileIn := *cmd.InFile

	from1 := time.Now()

	log.API.Printf("validating(mode=%s) %s ...\n", config.ValidationModeString(), fileIn)
	//logInfoAPI.Printf("validating(mode=%s) %s..\n", config.ValidationModeString(), fileIn)

	ctx, err := ReadContextFromFile(fileIn, config)
	if err != nil {
		return nil, err
	}

	dur1 := time.Since(from1).Seconds()

	from2 := time.Now()

	err = ValidateContext(ctx)
	if err != nil {

		s := ""
		if config.ValidationMode == pdf.ValidationStrict {
			s = " (try -mode=relaxed)"
		}

		err = errors.Wrap(err, "validation error"+s)
	} else {
		log.API.Println("validation ok")
		//logInfoAPI.Println("validation ok")
	}

	dur2 := time.Since(from2).Seconds()
	dur := time.Since(from1).Seconds()

	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	pdf.ValidationTimingStats(dur1, dur2, dur)
	// at this stage: no binary breakup available!
	ctx.Read.LogStats(ctx.Optimized)

	return nil, err
}

// Write generates a PDF file for a given Context.
func Write(ctx *pdf.Context) error {

	log.API.Printf("writing %s ...\n", ctx.Write.DirName+ctx.Write.FileName)
	//logInfoAPI.Printf("writing to %s..\n", fileName)

	err := pdf.Write(ctx)
	if err != nil {
		return errors.Wrap(err, "Write failed.")
	}

	// For the Optimize command only.
	if ctx.StatsFileName != "" {
		err = pdf.AppendStatsFile(ctx)
		if err != nil {
			return errors.Wrap(err, "Write stats failed.")
		}
	}

	return nil
}

// singlePageFileName generates a filename for a Context and a specific page number.
func singlePageFileName(ctx *pdf.Context, pageNr int) string {

	baseFileName := filepath.Base(ctx.Read.FileName)
	fileName := strings.TrimSuffix(baseFileName, ".pdf")
	return fileName + "_" + strconv.Itoa(pageNr) + ".pdf"
}

func writeSinglePagePDF(ctx *pdf.Context, pageNr int, dirOut string) error {

	ctx.ResetWriteContext()

	w := ctx.Write
	w.SelectedPages[pageNr] = true
	w.DirName = dirOut + "/"
	w.FileName = singlePageFileName(ctx, pageNr)
	log.API.Printf("writing %s ...\n", w.DirName+w.FileName)

	return pdf.Write(ctx)
}

func writeSinglePagePDFs(ctx *pdf.Context, selectedPages pdf.IntSet, dirOut string) error {

	ensureSelectedPages(ctx, &selectedPages)

	for i, v := range selectedPages {
		if v {
			err := writeSinglePagePDF(ctx, i, dirOut)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func readAndValidate(fileIn string, config *pdf.Configuration, from1 time.Time) (ctx *pdf.Context, dur1, dur2 float64, err error) {

	ctx, err = ReadContextFromFile(fileIn, config)
	if err != nil {
		return nil, 0, 0, err
	}
	dur1 = time.Since(from1).Seconds()

	from2 := time.Now()
	//log.API.Printf("validating %s ...\n", fileIn)
	err = validate.XRefTable(ctx.XRefTable)
	if err != nil {
		return nil, 0, 0, err
	}
	dur2 = time.Since(from2).Seconds()

	return ctx, dur1, dur2, nil
}

func readValidateAndOptimize(fileIn string, config *pdf.Configuration, from1 time.Time) (ctx *pdf.Context, dur1, dur2, dur3 float64, err error) {

	ctx, dur1, dur2, err = readAndValidate(fileIn, config, from1)
	if err != nil {
		return nil, 0, 0, 0, err
	}

	from3 := time.Now()
	//log.API.Printf("optimizing %s ...\n", fileIn)
	err = OptimizeContext(ctx)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	dur3 = time.Since(from3).Seconds()

	return ctx, dur1, dur2, dur3, nil
}

func logOperationStats(ctx *pdf.Context, op string, durRead, durVal, durOpt, durWrite, durTotal float64) {
	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	pdf.TimingStats(op, durRead, durVal, durOpt, durWrite, durTotal)
	ctx.Read.LogStats(ctx.Optimized)
	ctx.Write.LogStats()
}

// Optimize reads in fileIn, does validation, optimization and writes the result to fileOut.
func Optimize(cmd *Command) ([]string, error) {

	fileIn := *cmd.InFile
	fileOut := *cmd.OutFile
	config := cmd.Config

	fromStart := time.Now()

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return nil, err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	fromWrite := time.Now()

	dirName, fileName := filepath.Split(fileOut)
	ctx.Write.DirName = dirName
	ctx.Write.FileName = fileName

	err = Write(ctx)
	if err != nil {
		return nil, err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil, nil
}

func selectedPageRange(from, thru int) pdf.IntSet {
	s := pdf.IntSet{}
	for i := from; i <= thru; i++ {
		s[i] = true
	}
	return s
}

func pageRangeFileName(ctx *pdf.Context, from, thru int) string {
	if from == thru {
		return singlePageFileName(ctx, from)
	}
	baseFileName := filepath.Base(ctx.Read.FileName)
	fileName := strings.TrimSuffix(baseFileName, ".pdf")
	return fileName + "_" + strconv.Itoa(from) + "-" + strconv.Itoa(thru) + ".pdf"
}

func writeSpan(ctx *pdf.Context, from, thru int, dirOut string) error {
	ctx.ResetWriteContext()
	w := ctx.Write
	w.SelectedPages = selectedPageRange(from, thru)
	w.DirName = dirOut + "/"
	w.FileName = pageRangeFileName(ctx, from, thru)
	log.API.Printf("writing %s ...\n", w.DirName+w.FileName)
	return pdf.Write(ctx)
}

func writePDFSequence(ctx *pdf.Context, span int, dirOut string) error {

	for i := 0; i < ctx.PageCount/span; i++ {

		start := i * span
		from := start + 1
		thru := start + span

		err := writeSpan(ctx, from, thru, dirOut)
		if err != nil {
			return err
		}

	}

	if ctx.PageCount%span > 0 {

		start := (ctx.PageCount / span) * span
		from := start + 1
		thru := start + ctx.PageCount%span

		err := writeSpan(ctx, from, thru, dirOut)
		if err != nil {
			return err
		}

	}

	return nil
}

// Split generates a sequence of PDF files in dirOut obeying given split span.
// The default span 1 creates a sequence of single page PDFs.
func Split(cmd *Command) ([]string, error) {

	fileIn := *cmd.InFile
	dirOut := *cmd.OutDir
	config := cmd.Config
	span := cmd.Span

	fromStart := time.Now()

	log.API.Printf("splitting %s into %s (span=%d)...\n", fileIn, dirOut, span)

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return nil, err
	}

	fromWrite := time.Now()

	err = writePDFSequence(ctx, span, dirOut)
	if err != nil {
		return nil, err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "split", durRead, durVal, durOpt, durWrite, durTotal)

	return nil, nil
}

// appendTo appends fileIn to ctxDest's page tree.
func appendTo(fileIn string, ctxDest *pdf.Context) error {

	log.Stats.Printf("appendTo: appending %s to %s\n", fileIn, ctxDest.Read.FileName)

	// Build a Context for fileIn.
	ctxSource, _, _, err := readAndValidate(fileIn, ctxDest.Configuration, time.Now())
	if err != nil {
		return err
	}

	// Merge the source context into the dest context.
	log.API.Printf("merging in %s ...\n", fileIn)
	return pdf.MergeXRefTables(ctxSource, ctxDest)
}

// Merge some PDF files together and write the result to fileOut.
// This corresponds to concatenating these files in the order specified by filesIn.
// The first entry of filesIn serves as the destination xRefTable where all the remaining files gets merged into.
func Merge(cmd *Command) ([]string, error) {

	filesIn := cmd.InFiles
	fileOut := *cmd.OutFile
	config := cmd.Config

	log.API.Printf("merging into %s: %v\n", fileOut, filesIn)

	ctxDest, _, _, err := readAndValidate(filesIn[0], config, time.Now())
	if err != nil {
		return nil, err
	}

	if ctxDest.XRefTable.Version() < pdf.V15 {
		v, _ := pdf.PDFVersion("1.5")
		ctxDest.XRefTable.RootVersion = &v
		log.Stats.Println("Ensure V1.5 for writing object & xref streams")
	}

	// Repeatedly merge files into fileDest's xref table.
	for _, f := range filesIn[1:] {
		err = appendTo(f, ctxDest)
		if err != nil {
			return nil, err
		}
	}

	err = OptimizeContext(ctxDest)
	if err != nil {
		return nil, err
	}

	err = ValidateContext(ctxDest)
	if err != nil {
		return nil, err
	}

	dirName, fileName := filepath.Split(fileOut)
	ctxDest.Write.DirName = dirName
	ctxDest.Write.FileName = fileName

	err = Write(ctxDest)
	if err != nil {
		return nil, err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctxDest)

	return nil, nil
}

func imageObjNrs(ctx *pdf.Context, page int) []int {

	// TODO Exclude SMask image objects.

	o := []int{}

	for k, v := range ctx.Optimize.PageImages[page-1] {
		if v {
			o = append(o, k)
		}
	}

	return o
}

func imageFilenameWithoutExtension(dir, resID string, pageNr, objNr int) string {
	return filepath.Join(dir, fmt.Sprintf("%s_%d_%d", resID, pageNr, objNr))
}

func doExtractImages(ctx *pdf.Context, selectedPages pdf.IntSet) error {

	visited := pdf.IntSet{}

	for pageNr, v := range selectedPages {

		if v {

			log.Info.Printf("writing images for page %d\n", pageNr)

			for _, objNr := range imageObjNrs(ctx, pageNr) {

				if visited[objNr] {
					continue
				}

				visited[objNr] = true

				io, err := pdf.ExtractImageData(ctx, objNr)
				if err != nil {
					return err
				}

				if io == nil {
					continue
				}

				filename := imageFilenameWithoutExtension(ctx.Write.DirName, io.ResourceNames[0], pageNr, objNr)

				_, err = pdf.WriteImage(ctx.XRefTable, filename, io.ImageDict, objNr)
				if err != nil {
					return err
				}

			}

		}

	}

	return nil
}

// ExtractImages dumps embedded image resources from fileIn into dirOut for selected pages.
func ExtractImages(cmd *Command) ([]string, error) {

	fileIn := *cmd.InFile
	dirOut := *cmd.OutDir
	pageSelection := cmd.PageSelection
	config := cmd.Config

	fromStart := time.Now()

	log.API.Printf("extracting images from %s into %s ...\n", fileIn, dirOut)

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return nil, err
	}

	fromWrite := time.Now()

	pages, err := pagesForPageSelection(ctx.PageCount, pageSelection)
	if err != nil {
		return nil, err
	}

	ensureSelectedPages(ctx, &pages)

	ctx.Write.DirName = dirOut
	err = doExtractImages(ctx, pages)
	if err != nil {
		return nil, err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	pdf.TimingStats("write images", durRead, durVal, durOpt, durWrite, durTotal)

	return nil, nil
}

func fontObjNrs(ctx *pdf.Context, page int) []int {

	o := []int{}

	for k, v := range ctx.Optimize.PageFonts[page-1] {
		if v {
			o = append(o, k)
		}
	}

	return o
}

func doExtractFonts(ctx *pdf.Context, selectedPages pdf.IntSet) error {

	visited := pdf.IntSet{}

	for p, v := range selectedPages {

		if v {

			log.Info.Printf("writing fonts for page %d\n", p)

			for _, objNr := range fontObjNrs(ctx, p) {

				if visited[objNr] {
					continue
				}

				visited[objNr] = true

				fo, err := pdf.ExtractFontData(ctx, objNr)
				if err != nil {
					return err
				}

				if fo == nil {
					continue
				}

				fileName := fmt.Sprintf("%s/%s_%d_%d.%s", ctx.Write.DirName, fo.ResourceNames[0], p, objNr, fo.Extension)

				err = ioutil.WriteFile(fileName, fo.Data, os.ModePerm)
				if err != nil {
					return err
				}

			}

		}

	}

	return nil
}

// ExtractFonts dumps embedded fontfiles from fileIn into dirOut for selected pages.
func ExtractFonts(cmd *Command) ([]string, error) {

	fileIn := *cmd.InFile
	dirOut := *cmd.OutDir
	pageSelection := cmd.PageSelection
	config := cmd.Config

	fromStart := time.Now()

	log.API.Printf("extracting fonts from %s into %s ...\n", fileIn, dirOut)

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return nil, err
	}

	fromWrite := time.Now()

	pages, err := pagesForPageSelection(ctx.PageCount, pageSelection)
	if err != nil {
		return nil, err
	}

	ensureSelectedPages(ctx, &pages)

	ctx.Write.DirName = dirOut
	err = doExtractFonts(ctx, pages)
	if err != nil {
		return nil, err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	pdf.TimingStats("write fonts", durRead, durVal, durOpt, durWrite, durTotal)

	return nil, nil
}

// ExtractPages generates single page PDF files from fileIn in dirOut for selected pages.
func ExtractPages(cmd *Command) ([]string, error) {

	fileIn := *cmd.InFile
	dirOut := *cmd.OutDir
	pageSelection := cmd.PageSelection
	config := cmd.Config

	fromStart := time.Now()

	log.API.Printf("extracting pages from %s into %s ...\n", fileIn, dirOut)

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return nil, err
	}

	fromWrite := time.Now()

	pages, err := pagesForPageSelection(ctx.PageCount, pageSelection)
	if err != nil {
		return nil, err
	}

	err = writeSinglePagePDFs(ctx, pages, dirOut)
	if err != nil {
		return nil, err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	pdf.TimingStats("write PDFs", durRead, durVal, durOpt, durWrite, durTotal)

	return nil, nil
}

func contentObjNrs(ctx *pdf.Context, page int) ([]int, error) {

	objNrs := []int{}

	d, _, err := ctx.PageDict(page)
	if err != nil {
		return nil, err
	}

	o, found := d.Find("Contents")
	if !found || o == nil {
		return nil, nil
	}

	var objNr int

	ir, ok := o.(pdf.IndirectRef)
	if ok {
		objNr = ir.ObjectNumber.Value()
	}

	o, err = ctx.Dereference(o)
	if err != nil {
		return nil, err
	}

	if o == nil {
		return nil, nil
	}

	switch o := o.(type) {

	case pdf.StreamDict:

		objNrs = append(objNrs, objNr)

	case pdf.Array:

		for _, o := range o {

			ir, ok := o.(pdf.IndirectRef)
			if !ok {
				return nil, errors.Errorf("missing indref for page tree dict content no page %d", page)
			}

			sd, err := ctx.DereferenceStreamDict(ir)
			if err != nil {
				return nil, err
			}

			if sd == nil {
				continue
			}

			objNrs = append(objNrs, ir.ObjectNumber.Value())

		}

	}

	return objNrs, nil
}

func doExtractContent(ctx *pdf.Context, selectedPages pdf.IntSet) error {

	visited := pdf.IntSet{}

	for p, v := range selectedPages {

		if v {

			log.Info.Printf("writing content for page %d\n", p)

			objNrs, err := contentObjNrs(ctx, p)
			if err != nil {
				return err
			}

			if objNrs == nil {
				continue
			}

			for _, objNr := range objNrs {

				if visited[objNr] {
					continue
				}

				visited[objNr] = true

				b, err := pdf.ExtractStreamData(ctx, objNr)
				if err != nil {
					return err
				}

				if b == nil {
					continue
				}

				fileName := fmt.Sprintf("%s/%d_%d.txt", ctx.Write.DirName, p, objNr)

				err = ioutil.WriteFile(fileName, b, os.ModePerm)
				if err != nil {
					return err
				}

			}

		}

	}

	return nil
}

// ExtractContent dumps "PDF source" files from fileIn into dirOut for selected pages.
func ExtractContent(cmd *Command) ([]string, error) {

	fileIn := *cmd.InFile
	dirOut := *cmd.OutDir
	pageSelection := cmd.PageSelection
	config := cmd.Config

	fromStart := time.Now()

	log.API.Printf("extracting content from %s into %s ...\n", fileIn, dirOut)

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return nil, err
	}

	fromWrite := time.Now()

	pages, err := pagesForPageSelection(ctx.PageCount, pageSelection)
	if err != nil {
		return nil, err
	}

	ensureSelectedPages(ctx, &pages)

	ctx.Write.DirName = dirOut
	err = doExtractContent(ctx, pages)
	if err != nil {
		return nil, err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	pdf.TimingStats("write content", durRead, durVal, durOpt, durWrite, durTotal)

	return nil, nil
}

func extractMetadataStream(ctx *pdf.Context, obj pdf.Object, objNr int, dt string) error {

	ir, _ := obj.(pdf.IndirectRef)
	sObjNr := ir.ObjectNumber.Value()
	b, err := pdf.ExtractStreamData(ctx, sObjNr)
	if err != nil {
		return err
	}

	if b == nil {
		return nil
	}

	fileName := fmt.Sprintf("%s/%d_%s.txt", ctx.Write.DirName, objNr, dt)

	return ioutil.WriteFile(fileName, b, os.ModePerm)
}

func doExtractMetadata(ctx *pdf.Context, selectedPages pdf.IntSet) error {

	for k, v := range ctx.XRefTable.Table {
		if v.Free || v.Compressed {
			continue
		}
		switch d := v.Object.(type) {

		case pdf.Dict:

			o, found := d.Find("Metadata")
			if !found || o == nil {
				continue
			}

			dt := "unknown"
			if d.Type() != nil {
				dt = *d.Type()
			}

			err := extractMetadataStream(ctx, o, k, dt)
			if err != nil {
				return err
			}

		case pdf.StreamDict:

			o, found := d.Find("Metadata")
			if !found || o == nil {
				continue
			}

			dt := "unknown"
			if d.Type() != nil {
				dt = *d.Type()
			}

			err := extractMetadataStream(ctx, o, k, dt)
			if err != nil {
				return err
			}

		}
	}

	return nil
}

// ExtractMetadata dumps all metadata dict entries for fileIn into dirOut.
func ExtractMetadata(cmd *Command) ([]string, error) {

	fileIn := *cmd.InFile
	dirOut := *cmd.OutDir
	pageSelection := cmd.PageSelection
	config := cmd.Config

	fromStart := time.Now()

	log.API.Printf("extracting metadata from %s into %s ...\n", fileIn, dirOut)

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return nil, err
	}

	fromWrite := time.Now()

	pages, err := pagesForPageSelection(ctx.PageCount, pageSelection)
	if err != nil {
		return nil, err
	}

	ensureSelectedPages(ctx, &pages)

	ctx.Write.DirName = dirOut
	err = doExtractMetadata(ctx, pages)
	if err != nil {
		return nil, err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	pdf.TimingStats("write metadata", durRead, durVal, durOpt, durWrite, durTotal)

	return nil, nil
}

// Trim generates a trimmed version of fileIn containing all pages selected.
func Trim(cmd *Command) ([]string, error) {

	fileIn := *cmd.InFile
	fileOut := *cmd.OutFile
	pageSelection := cmd.PageSelection
	config := cmd.Config

	// pageSelection points to an empty slice if flag pages was omitted.

	fromStart := time.Now()

	log.API.Printf("trimming %s ...\n", fileIn)

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return nil, err
	}

	fromWrite := time.Now()

	pages, err := pagesForPageSelection(ctx.PageCount, pageSelection)
	if err != nil {
		return nil, err
	}

	ctx.Write.SelectedPages = pages

	dirName, fileName := filepath.Split(fileOut)
	ctx.Write.DirName = dirName
	ctx.Write.FileName = fileName

	err = Write(ctx)
	if err != nil {
		return nil, err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "trim, write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil, nil
}

// Encrypt fileIn and write result to fileOut.
func Encrypt(cmd *Command) ([]string, error) {
	return Optimize(cmd)
}

// Decrypt fileIn and write result to fileOut.
func Decrypt(cmd *Command) ([]string, error) {
	return Optimize(cmd)
}

// ChangeUserPassword of fileIn and write result to fileOut.
func ChangeUserPassword(cmd *Command) ([]string, error) {
	cmd.Config.UserPW = *cmd.PWOld
	cmd.Config.UserPWNew = cmd.PWNew
	return Optimize(cmd)
}

// ChangeOwnerPassword of fileIn and write result to fileOut.
func ChangeOwnerPassword(cmd *Command) ([]string, error) {
	cmd.Config.OwnerPW = *cmd.PWOld
	cmd.Config.OwnerPWNew = cmd.PWNew
	return Optimize(cmd)
}

// ListAttachments returns a list of embedded file attachments.
func ListAttachments(fileIn string, config *pdf.Configuration) ([]string, error) {

	fromStart := time.Now()

	//log.API.Println("Attachments:")

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return nil, err
	}

	fromWrite := time.Now()

	list, err := pdf.AttachList(ctx.XRefTable)
	if err != nil {
		return nil, err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	pdf.TimingStats("list files", durRead, durVal, durOpt, durWrite, durTotal)

	return list, nil
}

// AddAttachments embeds files into a PDF.
func AddAttachments(fileIn string, files []string, config *pdf.Configuration) error {

	fromStart := time.Now()

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return err
	}

	log.API.Printf("adding %d attachments to %s ...\n", len(files), fileIn)

	from := time.Now()
	var ok bool

	ok, err = pdf.AttachAdd(ctx.XRefTable, stringSet(files))
	if err != nil {
		return err
	}
	if !ok {
		log.API.Println("no attachment added.")
		return nil
	}

	durAdd := time.Since(from).Seconds()

	fromWrite := time.Now()

	fileOut := fileIn
	dirName, fileName := filepath.Split(fileOut)
	ctx.Write.DirName = dirName
	ctx.Write.FileName = fileName

	err = Write(ctx)
	if err != nil {
		return err
	}

	durWrite := durAdd + time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "add attachment, write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// RemoveAttachments deletes embedded files from a PDF.
func RemoveAttachments(fileIn string, files []string, config *pdf.Configuration) error {

	fromStart := time.Now()

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return err
	}

	if len(files) > 0 {
		log.API.Printf("removing %d attachments from %s ...\n", len(files), fileIn)
	} else {
		log.API.Printf("removing all attachments from %s ...\n", fileIn)
	}

	from := time.Now()

	var ok bool
	ok, err = pdf.AttachRemove(ctx.XRefTable, stringSet(files))
	if err != nil {
		return err
	}
	if !ok {
		log.API.Println("no attachment removed.")
		return nil
	}

	durRemove := time.Since(from).Seconds()

	fromWrite := time.Now()

	fileOut := fileIn
	dirName, fileName := filepath.Split(fileOut)
	ctx.Write.DirName = dirName
	ctx.Write.FileName = fileName

	err = Write(ctx)
	if err != nil {
		return err
	}

	durWrite := durRemove + time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "remove att, write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// ExtractAttachments extracts embedded files from a PDF.
func ExtractAttachments(fileIn, dirOut string, files []string, config *pdf.Configuration) error {

	fromStart := time.Now()

	log.API.Printf("extracting attachments from %s into %s ...\n", fileIn, dirOut)

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return err
	}

	fromWrite := time.Now()

	ctx.Write.DirName = dirOut
	err = pdf.AttachExtract(ctx, stringSet(files))
	if err != nil {
		return err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	pdf.TimingStats("write files", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// ListPermissions returns a list of user access permissions.
func ListPermissions(fileIn string, config *pdf.Configuration) ([]string, error) {

	fromStart := time.Now()

	//log.API.Println("User access permissions:")

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
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

// AddPermissions sets the user access permissions.
func AddPermissions(fileIn string, config *pdf.Configuration) error {

	fromStart := time.Now()

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return err
	}

	log.API.Printf("adding permissions to %s ...\n", fileIn)

	fromWrite := time.Now()

	fileOut := fileIn
	dirName, fileName := filepath.Split(fileOut)
	ctx.Write.DirName = dirName
	ctx.Write.FileName = fileName

	err = Write(ctx)
	if err != nil {
		return err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}

// AddWatermarks adds watermarks to all pages selected.
func AddWatermarks(cmd *Command) ([]string, error) {

	fileIn := *cmd.InFile
	fileOut := *cmd.OutFile
	pageSelection := cmd.PageSelection
	wm := cmd.Watermark
	config := cmd.Config

	fromStart := time.Now()

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return nil, err
	}

	log.API.Printf("%sing %s ...\n", wm.OnTopString(), fileIn)

	from := time.Now()

	pages, err := pagesForPageSelection(ctx.PageCount, pageSelection)
	if err != nil {
		return nil, err
	}

	ensureSelectedPages(ctx, &pages)

	err = pdf.AddWatermarks(ctx, pages, wm)
	if err != nil {
		return nil, err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	durStamp := time.Since(from).Seconds()

	fromWrite := time.Now()

	dirName, fileName := filepath.Split(fileOut)
	ctx.Write.DirName = dirName
	ctx.Write.FileName = fileName

	err = Write(ctx)
	if err != nil {
		return nil, err
	}

	durWrite := durStamp + time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "watermark, write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil, nil
}

func fileExists(filename string) bool {
	f, err := os.Open(filename)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

// ImportImages turns image files into a page sequence and writes the result to outFile.
// In its simplest form this operation converts an image into a PDF.
func ImportImages(cmd *Command) ([]string, error) {

	config := cmd.Config
	fileOut := *cmd.OutFile
	filesIn := cmd.InFiles
	imp := cmd.Import

	log.API.Printf("importing images into %s: %v\n%s", fileOut, filesIn, imp)
	//fmt.Printf("importing images into %s: %v\n%s\n", fileOut, filesIn, imp)

	var (
		ctx *pdf.Context
		err error
	)

	if fileExists(fileOut) {
		//fmt.Printf("%s already exists..\n", fileOut)
		ctx, _, _, err = readAndValidate(fileOut, config, time.Now())
	} else {
		//fmt.Printf("%s will be created\n", fileOut)
		ctx, err = pdf.CreateContextWithXRefTable(config, imp.PageDim)
	}
	if err != nil {
		return nil, err
	}

	pagesIndRef, err := ctx.Pages()
	if err != nil {
		return nil, err
	}

	// This is the page tree root.
	pagesDict, err := ctx.DereferenceDict(*pagesIndRef)
	if err != nil {
		return nil, err
	}

	for _, imgFilename := range filesIn {

		indRef, err := pdf.NewPageForImage(ctx.XRefTable, imgFilename, pagesIndRef, imp)
		if err != nil {
			return nil, err
		}

		err = pdf.AppendPageTree(indRef, 1, &pagesDict)
		if err != nil {
			return nil, err
		}

		ctx.PageCount++
	}

	err = ValidateContext(ctx)
	if err != nil {
		return nil, err
	}

	dirName, fileName := filepath.Split(fileOut)
	ctx.Write.DirName = dirName
	ctx.Write.FileName = fileName

	err = Write(ctx)
	if err != nil {
		return nil, err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	return nil, nil
}

// InsertPages inserts a blank page at every page selected.
func InsertPages(cmd *Command) ([]string, error) {

	fileIn := *cmd.InFile
	fileOut := *cmd.OutFile
	pageSelection := cmd.PageSelection
	config := cmd.Config

	fromStart := time.Now()

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return nil, err
	}

	log.API.Printf("inserting pages into %s ...\n", fileIn)

	from := time.Now()

	pages, err := pagesForPageSelection(ctx.PageCount, pageSelection)
	if err != nil {
		return nil, err
	}

	ensureSelectedPages(ctx, &pages)

	err = ctx.InsertPages(pages)
	if err != nil {
		return nil, err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	durStamp := time.Since(from).Seconds()

	fromWrite := time.Now()

	dirName, fileName := filepath.Split(fileOut)
	ctx.Write.DirName = dirName
	ctx.Write.FileName = fileName

	err = Write(ctx)
	if err != nil {
		return nil, err
	}

	durWrite := durStamp + time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "insert pages, write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil, nil
}

// RemovePages removes selected pages.
func RemovePages(cmd *Command) ([]string, error) {

	fileIn := *cmd.InFile
	fileOut := *cmd.OutFile
	pageSelection := cmd.PageSelection
	config := cmd.Config

	fromStart := time.Now()

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return nil, err
	}

	log.API.Printf("removing pages from %s ...\n", fileIn)

	fromWrite := time.Now()

	pages, err := pagesForPageSelection(ctx.PageCount, pageSelection)
	if err != nil {
		return nil, err
	}

	ctx.Write.SelectedPages = pages

	dirName, fileName := filepath.Split(fileOut)
	ctx.Write.DirName = dirName
	ctx.Write.FileName = fileName

	err = Write(ctx)
	if err != nil {
		return nil, err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "remove pages, write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil, nil
}

// Rotate rotates selected pages clockwise.
func Rotate(cmd *Command) ([]string, error) {

	fileIn := *cmd.InFile
	pageSelection := cmd.PageSelection
	config := cmd.Config

	fromStart := time.Now()

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return nil, err
	}

	log.API.Printf("rotating %s ...\n", fileIn)

	from := time.Now()

	pages, err := pagesForPageSelection(ctx.PageCount, pageSelection)
	if err != nil {
		return nil, err
	}

	ensureSelectedPages(ctx, &pages)

	err = pdf.RotatePages(ctx, pages, cmd.Rotation)
	if err != nil {
		return nil, err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	durStamp := time.Since(from).Seconds()

	fromWrite := time.Now()

	dirName, fileName := filepath.Split(fileIn)
	ctx.Write.DirName = dirName
	ctx.Write.FileName = fileName

	err = Write(ctx)
	if err != nil {
		return nil, err
	}

	durWrite := durStamp + time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	logOperationStats(ctx, "rotate, write", durRead, durVal, durOpt, durWrite, durTotal)

	return nil, nil
}

// NUp rearranges pages or images into page grids.
func NUp(cmd *Command) ([]string, error) {

	filesIn := cmd.InFiles
	fileOut := *cmd.OutFile
	pageSelection := cmd.PageSelection
	config := cmd.Config
	nup := cmd.NUp

	log.Info.Printf("%s", nup)

	var (
		ctx *pdf.Context
		err error
	)

	if nup.ImgInputFile {

		ctx, err = pdf.NUpFromImage(config, filesIn, nup)
		if err != nil {
			return nil, err
		}

	} else {

		ctx, _, _, err = readAndValidate(filesIn[0], config, time.Now())
		if err != nil {
			return nil, err
		}

		pages, err := pagesForPageSelection(ctx.PageCount, pageSelection)
		if err != nil {
			return nil, err
		}

		ensureSelectedPages(ctx, &pages)

		// New pages get added to ctx while old pages get deleted.
		// This way we avoid migrating objects between contexts.
		err = pdf.NUpFromPDF(ctx, pages, nup)
		if err != nil {
			return nil, err
		}

	}

	// Optional
	err = ValidateContext(ctx)
	if err != nil {
		return nil, err
	}

	dirName, fileName := filepath.Split(fileOut)
	ctx.Write.DirName = dirName
	ctx.Write.FileName = fileName

	err = Write(ctx)
	if err != nil {
		return nil, err
	}

	log.Stats.Printf("XRefTable:\n%s\n", ctx)

	return nil, nil
}
