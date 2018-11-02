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

// ReaderContext uses an io.Readseeker to build an internal structure holding its cross reference table aka the Context.
func ReaderContext(rs io.ReadSeeker, fileIn string, fileSize int64, config *pdf.Configuration) (*pdf.Context, error) {
	return pdf.ReadPDFFile(rs, fileIn, fileSize, config)
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
	return pdf.WritePDFFile(ctx)
}

// ReadContext reads in a PDF file and builds an internal structure holding its cross reference table aka the Context.
func ReadContext(fileIn string, config *pdf.Configuration) (*pdf.Context, error) {

	//logInfoAPI.Printf("reading %s..\n", fileIn)

	file, err := os.Open(fileIn)
	if err != nil {
		return nil, errors.Wrapf(err, "can't open %q", fileIn)
	}

	defer func() {
		file.Close()
	}()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	return ReaderContext(file, fileIn, fileInfo.Size(), config)
}

// Validate validates a PDF file against ISO-32000-1:2008.
func Validate(cmd *Command) ([]string, error) {

	config := cmd.Config
	fileIn := *cmd.InFile

	from1 := time.Now()

	fmt.Printf("validating(mode=%s) %s ...\n", config.ValidationModeString(), fileIn)
	//logInfoAPI.Printf("validating(mode=%s) %s..\n", config.ValidationModeString(), fileIn)

	ctx, err := ReadContext(fileIn, config)
	if err != nil {
		return nil, err
	}

	dur1 := time.Since(from1).Seconds()

	from2 := time.Now()

	err = ValidateContext(ctx)
	if err != nil {
		err = errors.Wrap(err, "validation error (try -mode=relaxed)")
	} else {
		fmt.Println("validation ok")
		//logInfoAPI.Println("validation ok")
	}

	dur2 := time.Since(from2).Seconds()
	dur := time.Since(from1).Seconds()

	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	log.Stats.Println("Timing:")
	log.Stats.Printf("read                 : %6.3fs  %4.1f%%\n", dur1, dur1/dur*100)
	log.Stats.Printf("validate             : %6.3fs  %4.1f%%\n", dur2, dur2/dur*100)
	log.Stats.Printf("total processing time: %6.3fs\n\n", dur)
	// at this stage: no binary breakup available!
	ctx.Read.LogStats(ctx.Optimized)

	return nil, err
}

// Write generates a PDF file for a given Context.
func Write(ctx *pdf.Context) error {

	fmt.Printf("writing %s ...\n", ctx.Write.DirName+ctx.Write.FileName)
	//logInfoAPI.Printf("writing to %s..\n", fileName)

	err := pdf.WritePDFFile(ctx)
	if err != nil {
		return errors.Wrap(err, "Write failed.")
	}

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
	w.Command = "Split"
	w.ExtractPageNr = pageNr
	w.DirName = dirOut + "/"
	w.FileName = singlePageFileName(ctx, pageNr)
	fmt.Printf("writing %s ...\n", w.DirName+w.FileName)

	return pdf.WritePDFFile(ctx)
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

	ctx, err = ReadContext(fileIn, config)
	if err != nil {
		return nil, 0, 0, err
	}
	dur1 = time.Since(from1).Seconds()

	from2 := time.Now()
	//fmt.Printf("validating %s ...\n", fileIn)
	//logInfoAPI.Printf("validating %s..\n", fileIn)
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
	//fmt.Printf("optimizing %s ...\n", fileIn)
	err = OptimizeContext(ctx)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	dur3 = time.Since(from3).Seconds()

	return ctx, dur1, dur2, dur3, nil
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

	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	log.Stats.Println("Timing:")
	log.Stats.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	log.Stats.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	log.Stats.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	log.Stats.Printf("write                : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	log.Stats.Printf("total processing time: %6.3fs\n\n", durTotal)
	ctx.Read.LogStats(ctx.Optimized)
	ctx.Write.LogStats()

	return nil, nil
}

// Split generates a sequence of single page PDF files in dirOut creating one file for every page of inFile.
func Split(cmd *Command) ([]string, error) {

	fileIn := *cmd.InFile
	dirOut := *cmd.OutDir
	config := cmd.Config

	fromStart := time.Now()

	fmt.Printf("splitting %s into %s ...\n", fileIn, dirOut)

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return nil, err
	}

	fromWrite := time.Now()

	err = writeSinglePagePDFs(ctx, nil, dirOut)
	if err != nil {
		return nil, err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()

	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	log.Stats.Println("Timing:")
	log.Stats.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	log.Stats.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	log.Stats.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	log.Stats.Printf("split                : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	log.Stats.Printf("total processing time: %6.3fs\n\n", durTotal)
	ctx.Read.LogStats(ctx.Optimized)
	ctx.Write.LogStats()

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
	fmt.Printf("merging in %s ...\n", fileIn)
	return pdf.MergeXRefTables(ctxSource, ctxDest)
}

// Merge some PDF files together and write the result to fileOut.
// This corresponds to concatenating these files in the order specified by filesIn.
// The first entry of filesIn serves as the destination xRefTable where all the remaining files gets merged into.
func Merge(cmd *Command) ([]string, error) {

	filesIn := cmd.InFiles
	fileOut := *cmd.OutFile
	config := cmd.Config

	fmt.Printf("merging into %s: %v\n", fileOut, filesIn)
	//logErrorAPI.Printf("Merge: filesIn: %v\n", filesIn)

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

	err = pdf.OptimizeXRefTable(ctxDest)
	if err != nil {
		return nil, err
	}

	err = validate.XRefTable(ctxDest.XRefTable)
	if err != nil {
		return nil, err
	}

	ctxDest.Write.Command = "Merge"

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

	fmt.Printf("extracting images from %s into %s ...\n", fileIn, dirOut)

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
	log.Stats.Println("Timing:")
	log.Stats.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	log.Stats.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	log.Stats.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	log.Stats.Printf("write images         : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	log.Stats.Printf("total processing time: %6.3fs\n\n", durTotal)

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

	fmt.Printf("extracting fonts from %s into %s ...\n", fileIn, dirOut)

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
	log.Stats.Println("Timing:")
	log.Stats.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	log.Stats.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	log.Stats.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	log.Stats.Printf("write fonts          : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	log.Stats.Printf("total processing time: %6.3fs\n\n", durTotal)

	return nil, nil
}

// ExtractPages generates single page PDF files from fileIn in dirOut for selected pages.
func ExtractPages(cmd *Command) ([]string, error) {

	fileIn := *cmd.InFile
	dirOut := *cmd.OutDir
	pageSelection := cmd.PageSelection
	config := cmd.Config

	fromStart := time.Now()

	fmt.Printf("extracting pages from %s into %s ...\n", fileIn, dirOut)

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
	log.Stats.Println("Timing:")
	log.Stats.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	log.Stats.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	log.Stats.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	log.Stats.Printf("write PDFs           : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	log.Stats.Printf("total processing time: %6.3fs\n\n", durTotal)
	ctx.Read.LogStats(ctx.Optimized)
	ctx.Write.LogStats()

	return nil, nil
}

func contentObjNrs(ctx *pdf.Context, page int) ([]int, error) {

	objNrs := []int{}

	d, _, err := ctx.PageDict(page)
	if err != nil {
		return nil, err
	}

	obj, found := d.Find("Contents")
	if !found || obj == nil {
		return nil, nil
	}

	var objNr int

	indRef, ok := obj.(pdf.IndirectRef)
	if ok {
		objNr = indRef.ObjectNumber.Value()
	}

	obj, err = ctx.Dereference(obj)
	if err != nil {
		return nil, err
	}

	if obj == nil {
		return nil, nil
	}

	switch obj := obj.(type) {

	case pdf.StreamDict:

		objNrs = append(objNrs, objNr)

	case pdf.Array:

		for _, obj := range obj {

			indRef, ok := obj.(pdf.IndirectRef)
			if !ok {
				return nil, errors.Errorf("missing indref for page tree dict content no page %d", page)
			}

			sd, err := ctx.DereferenceStreamDict(obj)
			if err != nil {
				return nil, err
			}

			if sd == nil {
				continue
			}

			objNrs = append(objNrs, indRef.ObjectNumber.Value())

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

	fmt.Printf("extracting content from %s into %s ...\n", fileIn, dirOut)

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
	log.Stats.Println("Timing:")
	log.Stats.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	log.Stats.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	log.Stats.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	log.Stats.Printf("write content        : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	log.Stats.Printf("total processing time: %6.3fs\n\n", durTotal)

	return nil, nil
}

func extractMetadataStream(ctx *pdf.Context, obj pdf.Object, objNr int, dt string) error {

	indRef, _ := obj.(pdf.IndirectRef)
	sObjNr := indRef.ObjectNumber.Value()
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

			obj, found := d.Find("Metadata")
			if !found || obj == nil {
				continue
			}

			dt := "unknown"
			if d.Type() != nil {
				dt = *d.Type()
			}

			err := extractMetadataStream(ctx, obj, k, dt)
			if err != nil {
				return err
			}

		case pdf.StreamDict:

			obj, found := d.Find("Metadata")
			if !found || obj == nil {
				continue
			}

			dt := "unknown"
			if d.Type() != nil {
				dt = *d.Type()
			}

			err := extractMetadataStream(ctx, obj, k, dt)
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

	fmt.Printf("extracting metadata from %s into %s ...\n", fileIn, dirOut)

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
	log.Stats.Println("Timing:")
	log.Stats.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	log.Stats.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	log.Stats.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	log.Stats.Printf("write metadata       : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	log.Stats.Printf("total processing time: %6.3fs\n\n", durTotal)

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

	fmt.Printf("trimming %s ...\n", fileIn)

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return nil, err
	}

	fromWrite := time.Now()

	pages, err := pagesForPageSelection(ctx.PageCount, pageSelection)
	if err != nil {
		return nil, err
	}

	ctx.Write.Command = "Trim"
	ctx.Write.ExtractPages = pages

	dirName, fileName := filepath.Split(fileOut)
	ctx.Write.DirName = dirName
	ctx.Write.FileName = fileName

	err = Write(ctx)
	if err != nil {
		return nil, err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()

	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	log.Stats.Println("Timing:")
	log.Stats.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	log.Stats.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	log.Stats.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	log.Stats.Printf("write PDF            : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	log.Stats.Printf("total processing time: %6.3fs\n\n", durTotal)
	ctx.Read.LogStats(ctx.Optimized)
	ctx.Write.LogStats()

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

	//fmt.Println("Attachments:")

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
	log.Stats.Println("Timing:")
	log.Stats.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	log.Stats.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	log.Stats.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	log.Stats.Printf("list files           : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	log.Stats.Printf("total processing time: %6.3fs\n\n", durTotal)

	return list, nil
}

// AddAttachments embeds files into a PDF.
func AddAttachments(fileIn string, files []string, config *pdf.Configuration) error {

	fromStart := time.Now()

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return err
	}

	fmt.Printf("adding %d attachments to %s ...\n", len(files), fileIn)

	from := time.Now()
	var ok bool

	ok, err = pdf.AttachAdd(ctx.XRefTable, stringSet(files))
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("no attachment added.")
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

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()

	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	log.Stats.Println("Timing:")
	log.Stats.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	log.Stats.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	log.Stats.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	log.Stats.Printf("add attachment       : %6.3fs  %4.1f%%\n", durAdd, durAdd/durTotal*100)
	log.Stats.Printf("write                : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	log.Stats.Printf("total processing time: %6.3fs\n\n", durTotal)
	ctx.Read.LogStats(ctx.Optimized)
	ctx.Write.LogStats()

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
		fmt.Printf("removing %d attachments from %s ...\n", len(files), fileIn)
	} else {
		fmt.Printf("removing all attachments from %s ...\n", fileIn)
	}

	from := time.Now()

	var ok bool
	ok, err = pdf.AttachRemove(ctx.XRefTable, stringSet(files))
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("no attachment removed.")
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

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()

	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	log.Stats.Println("Timing:")
	log.Stats.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	log.Stats.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	log.Stats.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	log.Stats.Printf("add attachment       : %6.3fs  %4.1f%%\n", durAdd, durAdd/durTotal*100)
	log.Stats.Printf("write                : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	log.Stats.Printf("total processing time: %6.3fs\n\n", durTotal)
	ctx.Read.LogStats(ctx.Optimized)
	ctx.Write.LogStats()

	return nil
}

// ExtractAttachments extracts embedded files from a PDF.
func ExtractAttachments(fileIn, dirOut string, files []string, config *pdf.Configuration) error {

	fromStart := time.Now()

	fmt.Printf("extracting attachments from %s into %s ...\n", fileIn, dirOut)

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
	log.Stats.Println("Timing:")
	log.Stats.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	log.Stats.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	log.Stats.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	log.Stats.Printf("write files          : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	log.Stats.Printf("total processing time: %6.3fs\n\n", durTotal)

	return nil
}

// ListPermissions returns a list of user access permissions.
func ListPermissions(fileIn string, config *pdf.Configuration) ([]string, error) {

	fromStart := time.Now()

	//fmt.Println("User access permissions:")

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return nil, err
	}

	fromList := time.Now()
	list := pdf.Permissions(ctx)
	durList := time.Since(fromList).Seconds()

	durTotal := time.Since(fromStart).Seconds()

	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	log.Stats.Println("Timing:")
	log.Stats.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	log.Stats.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	log.Stats.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	log.Stats.Printf("list permissions     : %6.3fs  %4.1f%%\n", durList, durList/durTotal*100)
	log.Stats.Printf("total processing time: %6.3fs\n\n", durTotal)

	return list, nil
}

// AddPermissions sets the user access permissions.
func AddPermissions(fileIn string, config *pdf.Configuration) error {

	fromStart := time.Now()

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return err
	}

	fmt.Printf("adding permissions to %s ...\n", fileIn)

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

	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	log.Stats.Println("Timing:")
	log.Stats.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	log.Stats.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	log.Stats.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	log.Stats.Printf("write                : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	log.Stats.Printf("total processing time: %6.3fs\n\n", durTotal)
	ctx.Read.LogStats(ctx.Optimized)
	ctx.Write.LogStats()

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

	fmt.Printf("%sing %s ...\n", wm.OnTopString(), fileIn)

	from := time.Now()

	pages, err := pagesForPageSelection(ctx.PageCount, pageSelection)
	if err != nil {
		return nil, err
	}

	ensureSelectedPages(ctx, &pages)

	err = pdf.AddWatermarks(ctx.XRefTable, pages, wm)
	if err != nil {
		return nil, err
	}

	durStamp := time.Since(from).Seconds()

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

	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	log.Stats.Println("Timing:")
	log.Stats.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	log.Stats.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	log.Stats.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	log.Stats.Printf("watermark            : %6.3fs  %4.1f%%\n", durStamp, durStamp/durTotal*100)
	log.Stats.Printf("write                : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	log.Stats.Printf("total processing time: %6.3fs\n\n", durTotal)
	ctx.Read.LogStats(ctx.Optimized)
	ctx.Write.LogStats()

	return nil, nil
}
