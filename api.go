package pdfcpu

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hhrutter/pdfcpu/crypto"

	"github.com/hhrutter/pdfcpu/attach"
	"github.com/hhrutter/pdfcpu/extract"
	"github.com/hhrutter/pdfcpu/merge"
	"github.com/hhrutter/pdfcpu/optimize"
	"github.com/hhrutter/pdfcpu/read"
	"github.com/hhrutter/pdfcpu/types"
	"github.com/hhrutter/pdfcpu/validate"
	"github.com/hhrutter/pdfcpu/write"

	"github.com/pkg/errors"
)

var (
	logInfoAPI  *log.Logger
	logStatsAPI *log.Logger
	logErrorAPI *log.Logger

	selectedPagesRegExp *regexp.Regexp
)

func stringSet(slice []string) types.StringSet {

	strSet := types.StringSet{}

	if slice == nil {
		return strSet
	}

	for _, s := range slice {
		strSet[s] = true
	}

	return strSet
}

func setupRegExpForPageSelection() *regexp.Regexp {

	e := "[!n]?((-\\d+)|(\\d+(-(\\d+)?)?))"

	exp := "^" + e + "(," + e + ")*$"

	re, _ := regexp.Compile(exp)

	return re
}

func init() {
	logInfoAPI = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	logStatsAPI = log.New(ioutil.Discard, "STATS: ", log.Ldate|log.Ltime|log.Lshortfile)
	logErrorAPI = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	selectedPagesRegExp = setupRegExpForPageSelection()
}

// Verbose controls logging output.
func Verbose(verbose bool) {
	if verbose {
		logStatsAPI = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	} else {
		logStatsAPI = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	}
}

// Read reads in a PDF file and builds an internal structure holding its cross reference table aka the PDFContext.
func Read(fileIn string, config *types.Configuration) (*types.PDFContext, error) {

	//logInfoAPI.Printf("reading %s..\n", fileIn)

	ctx, err := read.PDFFile(fileIn, config)
	if err != nil {
		return nil, errors.Wrap(err, "Read failed.")
	}

	return ctx, nil
}

// Validate validates a PDF file against ISO-32000-1:2008.
func Validate(fileIn string, config *types.Configuration) error {

	from1 := time.Now()

	fmt.Printf("validating(mode=%s) %s ...\n", config.ValidationModeString(), fileIn)
	//logInfoAPI.Printf("validating(mode=%s) %s..\n", config.ValidationModeString(), fileIn)

	ctx, err := Read(fileIn, config)
	if err != nil {
		return err
	}

	dur1 := time.Since(from1).Seconds()

	from2 := time.Now()

	err = validate.XRefTable(ctx.XRefTable)
	if err != nil {
		err = errors.Wrap(err, "validation error (try -mode=relaxed)")
	} else {
		fmt.Println("validation ok")
		//logInfoAPI.Println("validation ok")
	}

	dur2 := time.Since(from2).Seconds()
	dur := time.Since(from1).Seconds()

	logStatsAPI.Printf("XRefTable:\n%s\n", ctx)
	logStatsAPI.Println("Timing:")
	logStatsAPI.Printf("read                 : %6.3fs  %4.1f%%\n", dur1, dur1/dur*100)
	logStatsAPI.Printf("validate             : %6.3fs  %4.1f%%\n", dur2, dur2/dur*100)
	logStatsAPI.Printf("total processing time: %6.3fs\n\n", dur)
	// at this stage: no binary breakup available!
	ctx.Read.LogStats(logStatsAPI, ctx.Optimized)

	return err
}

// Write generates a PDF file for a given PDFContext.
func Write(ctx *types.PDFContext) error {

	fmt.Printf("writing %s ...\n", ctx.Write.DirName+ctx.Write.FileName)
	//logInfoAPI.Printf("writing to %s..\n", fileName)

	err := write.PDFFile(ctx)
	if err != nil {
		return errors.Wrap(err, "Write failed.")
	}

	if ctx.StatsFileName != "" {
		err = write.AppendStatsFile(ctx)
		if err != nil {
			return errors.Wrap(err, "Write stats failed.")
		}
	}

	return nil
}

// singlePageFileName generates a filename for a PDFContext and a specific page number.
func singlePageFileName(ctx *types.PDFContext, pageNr int) string {

	baseFileName := filepath.Base(ctx.Read.FileName)
	fileName := strings.TrimSuffix(baseFileName, ".pdf")
	return fileName + "_" + strconv.Itoa(pageNr) + ".pdf"
}

func writeSinglePagePDF(ctx *types.PDFContext, pageNr int, dirOut string) error {

	ctx.ResetWriteContext()

	w := ctx.Write
	w.Command = "Split"
	w.ExtractPageNr = pageNr
	w.DirName = dirOut + "/"
	w.FileName = singlePageFileName(ctx, pageNr)
	fmt.Printf("writing %s ...\n", w.DirName+w.FileName)

	return write.PDFFile(ctx)
}

func writeSinglePagePDFs(ctx *types.PDFContext, selectedPages types.IntSet, dirOut string) error {

	if selectedPages == nil {
		selectedPages = types.IntSet{}
	}

	if len(selectedPages) == 0 {
		for i := 1; i <= ctx.PageCount; i++ {
			selectedPages[i] = true
		}
	}

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

func readAndValidate(fileIn string, config *types.Configuration, from1 time.Time) (ctx *types.PDFContext, dur1, dur2 float64, err error) {

	ctx, err = Read(fileIn, config)
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

func readValidateAndOptimize(fileIn string, config *types.Configuration, from1 time.Time) (ctx *types.PDFContext, dur1, dur2, dur3 float64, err error) {

	ctx, dur1, dur2, err = readAndValidate(fileIn, config, from1)
	if err != nil {
		return nil, 0, 0, 0, err
	}

	from3 := time.Now()
	//fmt.Printf("optimizing %s ...\n", fileIn)
	//logInfoAPI.Printf("optimizing %s..\n", fileIn)
	err = optimize.XRefTable(ctx)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	dur3 = time.Since(from3).Seconds()

	return ctx, dur1, dur2, dur3, nil
}

// Optimize reads in fileIn, does validation, optimization and writes the result to fileOut.
func Optimize(fileIn, fileOut string, config *types.Configuration) error {

	fromStart := time.Now()

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return err
	}

	logStatsAPI.Printf("XRefTable:\n%s\n", ctx)

	fromWrite := time.Now()

	dirName, fileName := filepath.Split(fileOut)
	ctx.Write.DirName = dirName
	ctx.Write.FileName = fileName

	err = Write(ctx)
	if err != nil {
		return err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()

	logStatsAPI.Printf("XRefTable:\n%s\n", ctx)
	logStatsAPI.Println("Timing:")
	logStatsAPI.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	logStatsAPI.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	logStatsAPI.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	logStatsAPI.Printf("write                : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	logStatsAPI.Printf("total processing time: %6.3fs\n\n", durTotal)
	ctx.Read.LogStats(logStatsAPI, ctx.Optimized)
	ctx.Write.LogStats(logStatsAPI)

	return nil
}

// ParsePageSelection ensures a correct page selection expression.
func ParsePageSelection(s string) ([]string, error) {

	if s == "" {
		return nil, nil
	}

	// Ensure valid comma separated expression of: {!}{-}# or {!}#-{#}
	//
	// Negated expressions:
	// '!' negates an expression
	// since '!' needs to be part of a single quoted string in bash
	// as an alternative also 'n' works instead of "!"
	//
	// Extract all but page 4 may be expressed as: "1-,!4" or "1-,n4"
	//
	// The pageSelection is evaluated strictly from left to right!
	// e.g. "!3,1-5" extracts pages 1-5 whereas "1-5,!3" extracts pages 1,2,4,5
	//

	if !selectedPagesRegExp.MatchString(s) {
		return nil, errors.Errorf("-pages \"%s\" => syntax error\n", s)
	}

	//fmt.Printf("pageSelection: <%s>\n", pageSelection)

	return strings.Split(s, ","), nil
}

func handlePrefix(v string, negated bool, pageCount int, selectedPages types.IntSet) error {

	i, err := strconv.Atoi(v)
	if err != nil {
		return err
	}

	// Handle overflow gracefully
	if i > pageCount {
		i = pageCount
	}

	// identified
	// -# ... select all pages up to and including #
	// or !-# ... deselect all pages up to and including #
	for j := 1; j <= i; j++ {
		selectedPages[j] = !negated
	}

	return nil
}

func handleSuffix(v string, negated bool, pageCount int, selectedPages types.IntSet) error {

	// must be #- ... select all pages from here until the end.
	// or !#- ... deselect all pages from here until the end.

	i, err := strconv.Atoi(v)
	if err != nil {
		return err
	}

	// Handle overflow gracefully
	if i > pageCount {
		return nil
	}

	for j := i; j <= pageCount; j++ {
		selectedPages[j] = !negated
	}

	return nil
}

func handleSpecificPage(s string, negated bool, pageCount int, selectedPages types.IntSet) error {

	// must be # ... select a specific page
	// or !# ... deselect a specific page

	i, err := strconv.Atoi(s)
	if err != nil {
		return err
	}

	// Handle overflow gracefully
	if i > pageCount {
		return nil
	}

	selectedPages[i] = !negated

	return nil
}

func negation(c byte) bool {
	return c == '!' || c == 'n'
}

func setPageSelection(pageCount int, pageSelection []string) (selectedPages types.IntSet, err error) {

	selectedPages = types.IntSet{}

	for _, v := range pageSelection {

		//logStatsAPI.Printf("pageExp: <%s>\n", v)

		var negated bool
		if negation(v[0]) {
			negated = true
			//logInfoAPI.Printf("is a negated exp\n")
			v = v[1:]
		}

		if v[0] == '-' {

			v = v[1:]

			err = handlePrefix(v, negated, pageCount, selectedPages)
			if err != nil {
				return nil, err
			}

			continue
		}

		if strings.HasSuffix(v, "-") {

			err = handleSuffix(v[:len(v)-1], negated, pageCount, selectedPages)
			if err != nil {
				return nil, err
			}

			continue
		}

		// if v contains '-' somewhere in the middle
		// this must be #-# ... select a page range
		// or !#-# ... deselect a page range

		pr := strings.Split(v, "-")
		if len(pr) == 2 {

			from, err := strconv.Atoi(pr[0])
			if err != nil {
				return nil, err
			}

			// Handle overflow gracefully
			if from > pageCount {
				continue
			}

			thru, err := strconv.Atoi(pr[1])
			if err != nil {
				return nil, err
			}

			// Handle overflow gracefully
			if thru < from {
				continue
			}

			if thru > pageCount {
				thru = pageCount
			}

			for i := from; i <= thru; i++ {
				selectedPages[i] = !negated
			}

			continue
		}

		err = handleSpecificPage(pr[0], negated, pageCount, selectedPages)
		if err != nil {
			return nil, err
		}

	}

	return selectedPages, nil
}

func pagesForPageSelection(pageCount int, pageSelection []string) (selectedPages types.IntSet, err error) {

	if pageSelection == nil {
		logInfoAPI.Println("pagesForPageSelection: pageSelection is nil")
		return nil, nil
	}

	if len(pageSelection) == 0 {
		logStatsAPI.Println("pagesForPageSelection: pageSelection is empty")
	}

	return setPageSelection(pageCount, pageSelection)
}

// Split generates a sequence of single page PDF files in dirOut creating one file for every page of inFile.
func Split(fileIn, dirOut string, config *types.Configuration) error {

	fromStart := time.Now()

	fmt.Printf("splitting %s into %s ...\n", fileIn, dirOut)

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return err
	}

	fromWrite := time.Now()

	err = writeSinglePagePDFs(ctx, nil, dirOut)
	if err != nil {
		return err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()

	logStatsAPI.Printf("XRefTable:\n%s\n", ctx)
	logStatsAPI.Println("Timing:")
	logStatsAPI.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	logStatsAPI.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	logStatsAPI.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	logStatsAPI.Printf("split                : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	logStatsAPI.Printf("total processing time: %6.3fs\n\n", durTotal)
	ctx.Read.LogStats(logStatsAPI, ctx.Optimized)
	ctx.Write.LogStats(logStatsAPI)

	return nil
}

// appendTo appends fileIn to ctxDest's page tree.
func appendTo(fileIn string, ctxDest *types.PDFContext) error {

	logStatsAPI.Printf("appendTo: appending %s to %s\n", fileIn, ctxDest.Read.FileName)

	// Build a PDFContext for fileIn.
	ctxSource, _, _, err := readAndValidate(fileIn, ctxDest.Configuration, time.Now())
	if err != nil {
		return err
	}

	// Merge the source context into the dest context.
	fmt.Printf("merging in %s ...\n", fileIn)
	return merge.XRefTables(ctxSource, ctxDest)
}

// Merge some PDF files together and write the result to fileOut.
// This corresponds to concatenating these files in the order specified by filesIn.
// The first entry of filesIn serves as the destination xRefTable where all the remaining files gets merged into.
func Merge(filesIn []string, fileOut string, config *types.Configuration) error {

	fmt.Printf("merging into %s: %v\n", fileOut, filesIn)
	//logErrorAPI.Printf("Merge: filesIn: %v\n", filesIn)

	ctxDest, _, _, err := readAndValidate(filesIn[0], config, time.Now())
	if err != nil {
		return err
	}

	if ctxDest.XRefTable.Version() < types.V15 {
		v, _ := types.Version("1.5")
		ctxDest.XRefTable.RootVersion = &v
		logStatsAPI.Println("Ensure V1.5 for writing object & xref streams")
	}

	// Repeatedly merge files into fileDest's xref table.
	for f := range stringSet(filesIn[1:]) {
		err = appendTo(f, ctxDest)
		if err != nil {
			return err
		}
	}

	err = optimize.XRefTable(ctxDest)
	if err != nil {
		return err
	}

	err = validate.XRefTable(ctxDest.XRefTable)
	if err != nil {
		return err
	}

	ctxDest.Write.Command = "Merge"

	dirName, fileName := filepath.Split(fileOut)
	ctxDest.Write.DirName = dirName
	ctxDest.Write.FileName = fileName

	err = Write(ctxDest)
	if err != nil {
		return err
	}

	logStatsAPI.Printf("XRefTable:\n%s\n", ctxDest)

	return nil
}

// ExtractImages dumps embedded image resources from fileIn into dirOut for selected pages.
func ExtractImages(fileIn, dirOut string, pageSelection []string, config *types.Configuration) error {

	fromStart := time.Now()

	fmt.Printf("extracting images from %s into %s ...\n", fileIn, dirOut)

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return err
	}

	fromWrite := time.Now()

	pages, err := pagesForPageSelection(ctx.PageCount, pageSelection)
	if err != nil {
		return err
	}

	ctx.Write.DirName = dirOut
	err = extract.Images(ctx, pages)
	if err != nil {
		return err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()

	logStatsAPI.Printf("XRefTable:\n%s\n", ctx)
	logStatsAPI.Println("Timing:")
	logStatsAPI.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	logStatsAPI.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	logStatsAPI.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	logStatsAPI.Printf("write images         : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	logStatsAPI.Printf("total processing time: %6.3fs\n\n", durTotal)

	return nil
}

// ExtractFonts dumps embedded fontfiles from fileIn into dirOut for selected pages.
func ExtractFonts(fileIn, dirOut string, pageSelection []string, config *types.Configuration) error {

	fromStart := time.Now()

	fmt.Printf("extracting fonts from %s into %s ...\n", fileIn, dirOut)

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return err
	}

	fromWrite := time.Now()

	pages, err := pagesForPageSelection(ctx.PageCount, pageSelection)
	if err != nil {
		return err
	}

	ctx.Write.DirName = dirOut
	err = extract.Fonts(ctx, pages)
	if err != nil {
		return err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()

	logStatsAPI.Printf("XRefTable:\n%s\n", ctx)
	logStatsAPI.Println("Timing:")
	logStatsAPI.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	logStatsAPI.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	logStatsAPI.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	logStatsAPI.Printf("write fonts          : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	logStatsAPI.Printf("total processing time: %6.3fs\n\n", durTotal)

	return nil
}

// ExtractPages generates single page PDF files from fileIn in dirOut for selected pages.
func ExtractPages(fileIn, dirOut string, pageSelection []string, config *types.Configuration) error {

	fromStart := time.Now()

	fmt.Printf("extracting pages from %s into %s ...\n", fileIn, dirOut)

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return err
	}

	fromWrite := time.Now()

	pages, err := pagesForPageSelection(ctx.PageCount, pageSelection)
	if err != nil {
		return err
	}

	err = writeSinglePagePDFs(ctx, pages, dirOut)
	if err != nil {
		return err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()

	logStatsAPI.Printf("XRefTable:\n%s\n", ctx)
	logStatsAPI.Println("Timing:")
	logStatsAPI.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	logStatsAPI.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	logStatsAPI.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	logStatsAPI.Printf("write PDFs           : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	logStatsAPI.Printf("total processing time: %6.3fs\n\n", durTotal)
	ctx.Read.LogStats(logStatsAPI, ctx.Optimized)
	ctx.Write.LogStats(logStatsAPI)

	return nil
}

// ExtractContent dumps "PDF source" files from fileIn into dirOut for selected pages.
func ExtractContent(fileIn, dirOut string, pageSelection []string, config *types.Configuration) error {

	fromStart := time.Now()

	fmt.Printf("extracting content from %s into %s ...\n", fileIn, dirOut)

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return err
	}

	fromWrite := time.Now()

	pages, err := pagesForPageSelection(ctx.PageCount, pageSelection)
	if err != nil {
		return err
	}

	ctx.Write.DirName = dirOut
	err = extract.Content(ctx, pages)
	if err != nil {
		return err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()

	logStatsAPI.Printf("XRefTable:\n%s\n", ctx)
	logStatsAPI.Println("Timing:")
	logStatsAPI.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	logStatsAPI.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	logStatsAPI.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	logStatsAPI.Printf("write content        : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	logStatsAPI.Printf("total processing time: %6.3fs\n\n", durTotal)

	return nil
}

// Trim generates a trimmed version of fileIn containing all pages selected.
func Trim(fileIn, fileOut string, pageSelection []string, config *types.Configuration) error {

	// pageSelection points to an empty slice if flag pages was omitted.

	fromStart := time.Now()

	fmt.Printf("trimming %s ...\n", fileIn)

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return err
	}

	fromWrite := time.Now()

	pages, err := pagesForPageSelection(ctx.PageCount, pageSelection)
	if err != nil {
		return err
	}

	ctx.Write.Command = "Trim"
	ctx.Write.ExtractPages = pages

	dirName, fileName := filepath.Split(fileOut)
	ctx.Write.DirName = dirName
	ctx.Write.FileName = fileName

	err = Write(ctx)
	if err != nil {
		return err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()

	logStatsAPI.Printf("XRefTable:\n%s\n", ctx)
	logStatsAPI.Println("Timing:")
	logStatsAPI.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	logStatsAPI.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	logStatsAPI.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	logStatsAPI.Printf("write PDF            : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	logStatsAPI.Printf("total processing time: %6.3fs\n\n", durTotal)
	ctx.Read.LogStats(logStatsAPI, ctx.Optimized)
	ctx.Write.LogStats(logStatsAPI)

	return nil
}

// Encrypt fileIn and write result to fileOut.
func Encrypt(fileIn, fileOut string, config *types.Configuration) error {
	return Optimize(fileIn, fileOut, config)
}

// Decrypt fileIn and write result to fileOut.
func Decrypt(fileIn, fileOut string, config *types.Configuration) error {
	return Optimize(fileIn, fileOut, config)
}

// ChangeUserPassword of fileIn and write result to fileOut.
func ChangeUserPassword(fileIn, fileOut string, config *types.Configuration, pwOld, pwNew *string) error {
	config.UserPW = *pwOld
	config.UserPWNew = pwNew
	return Optimize(fileIn, fileOut, config)
}

// ChangeOwnerPassword of fileIn and write result to fileOut.
func ChangeOwnerPassword(fileIn, fileOut string, config *types.Configuration, pwOld, pwNew *string) error {
	config.OwnerPW = *pwOld
	config.OwnerPWNew = pwNew
	return Optimize(fileIn, fileOut, config)
}

// ListAttachments returns a list of embedded file attachments.
func ListAttachments(fileIn string, config *types.Configuration) ([]string, error) {

	fromStart := time.Now()

	//fmt.Println("Attachments:")

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return nil, err
	}

	fromWrite := time.Now()

	list, err := attach.List(ctx.XRefTable)
	if err != nil {
		return nil, err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()

	logStatsAPI.Printf("XRefTable:\n%s\n", ctx)
	logStatsAPI.Println("Timing:")
	logStatsAPI.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	logStatsAPI.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	logStatsAPI.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	logStatsAPI.Printf("list files           : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	logStatsAPI.Printf("total processing time: %6.3fs\n\n", durTotal)

	return list, nil
}

// AddAttachments embeds files into a PDF.
func AddAttachments(fileIn string, files []string, config *types.Configuration) error {

	fromStart := time.Now()

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return err
	}

	fmt.Printf("adding %d attachments to %s ...\n", len(files), fileIn)

	from := time.Now()
	var ok bool

	ok, err = attach.Add(ctx.XRefTable, stringSet(files))
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

	logStatsAPI.Printf("XRefTable:\n%s\n", ctx)
	logStatsAPI.Println("Timing:")
	logStatsAPI.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	logStatsAPI.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	logStatsAPI.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	logStatsAPI.Printf("add attachment       : %6.3fs  %4.1f%%\n", durAdd, durAdd/durTotal*100)
	logStatsAPI.Printf("write                : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	logStatsAPI.Printf("total processing time: %6.3fs\n\n", durTotal)
	ctx.Read.LogStats(logStatsAPI, ctx.Optimized)
	ctx.Write.LogStats(logStatsAPI)

	return nil
}

// RemoveAttachments deletes embedded files from a PDF.
func RemoveAttachments(fileIn string, files []string, config *types.Configuration) error {

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
	ok, err = attach.Remove(ctx.XRefTable, stringSet(files))
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

	logStatsAPI.Printf("XRefTable:\n%s\n", ctx)
	logStatsAPI.Println("Timing:")
	logStatsAPI.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	logStatsAPI.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	logStatsAPI.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	logStatsAPI.Printf("add attachment       : %6.3fs  %4.1f%%\n", durAdd, durAdd/durTotal*100)
	logStatsAPI.Printf("write                : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	logStatsAPI.Printf("total processing time: %6.3fs\n\n", durTotal)
	ctx.Read.LogStats(logStatsAPI, ctx.Optimized)
	ctx.Write.LogStats(logStatsAPI)

	return nil
}

// ExtractAttachments extracts embedded files from a PDF.
func ExtractAttachments(fileIn, dirOut string, files []string, config *types.Configuration) error {

	fromStart := time.Now()

	fmt.Printf("extracting attachments from %s into %s ...\n", fileIn, dirOut)

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return err
	}

	fromWrite := time.Now()

	ctx.Write.DirName = dirOut
	err = attach.Extract(ctx, stringSet(files))
	if err != nil {
		return err
	}

	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()

	logStatsAPI.Printf("XRefTable:\n%s\n", ctx)
	logStatsAPI.Println("Timing:")
	logStatsAPI.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	logStatsAPI.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	logStatsAPI.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	logStatsAPI.Printf("write files          : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	logStatsAPI.Printf("total processing time: %6.3fs\n\n", durTotal)

	return nil
}

// ListPermissions returns a list of user access permissions.
func ListPermissions(fileIn string, config *types.Configuration) ([]string, error) {

	fromStart := time.Now()

	//fmt.Println("User access permissions:")

	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(fileIn, config, fromStart)
	if err != nil {
		return nil, err
	}

	fromList := time.Now()
	list := crypto.ListPermissions(ctx)
	durList := time.Since(fromList).Seconds()

	durTotal := time.Since(fromStart).Seconds()

	logStatsAPI.Printf("XRefTable:\n%s\n", ctx)
	logStatsAPI.Println("Timing:")
	logStatsAPI.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	logStatsAPI.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	logStatsAPI.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	logStatsAPI.Printf("list permissions     : %6.3fs  %4.1f%%\n", durList, durList/durTotal*100)
	logStatsAPI.Printf("total processing time: %6.3fs\n\n", durTotal)

	return list, nil
}

// AddPermissions sets the user access permissions.
func AddPermissions(fileIn string, config *types.Configuration) error {

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

	logStatsAPI.Printf("XRefTable:\n%s\n", ctx)
	logStatsAPI.Println("Timing:")
	logStatsAPI.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	logStatsAPI.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	logStatsAPI.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	logStatsAPI.Printf("write                : %6.3fs  %4.1f%%\n", durWrite, durWrite/durTotal*100)
	logStatsAPI.Printf("total processing time: %6.3fs\n\n", durTotal)
	ctx.Read.LogStats(logStatsAPI, ctx.Optimized)
	ctx.Write.LogStats(logStatsAPI)

	return nil
}
