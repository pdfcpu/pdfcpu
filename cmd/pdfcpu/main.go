package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hhrutter/pdfcpu"
	"github.com/hhrutter/pdfcpu/crypto"
	"github.com/hhrutter/pdfcpu/extract"
	"github.com/hhrutter/pdfcpu/merge"
	"github.com/hhrutter/pdfcpu/optimize"
	"github.com/hhrutter/pdfcpu/read"
	"github.com/hhrutter/pdfcpu/types"
	"github.com/hhrutter/pdfcpu/validate"
	"github.com/hhrutter/pdfcpu/write"
)

const (
	usage = `pdfcpu is a tool for PDF manipulation written in Go.
	
Usage:
	
	pdfcpu command [arguments]
		
The commands are:
	
	validate	validate PDF against PDF 32000-1:2008 (PDF 1.7)
	optimize	optimize PDF by getting rid of redundant page resources
	split		split multi-page PDF into several single-page PDFs
	merge		concatenate 2 or more PDFs
	extract		extract images, fonts, content, pages out of a PDF
	trim		create trimmed version of a PDF
	version		print pdfcpu version

	Single-letter Unix-style supported for commands and flags.

Use "pdfcpu help [command]" for more information about a command.`

	usageValidate     = "usage: pdfcpu validate [-verbose] [-mode strict|relaxed] [-upw userpw] [-opw ownerpw] inFile"
	usageLongValidate = `Validate checks inFile for specification compliance.

verbose ... extensive log output
   mode ... validation mode
    upw ... user password
    opw ... owner password
 inFile ... input pdf file
		
The validation modes are:

 strict ... (default) validates against PDF 32000-1:2008 (PDF 1.7)
relaxed ... like strict but doesn't complain about common seen spec violations.`

	usageOptimize     = "usage: pdfcpu optimize [-verbose] [-stats csvFile] [-upw userpw] [-opw ownerpw] inFile [outFile]"
	usageLongOptimize = `Optimize reads inFile, removes redundant page resources like embedded fonts and images and writes the result to outFile.

verbose ... extensive log output
  stats ... appends a stats line to a csv file with information about the usage of root and page entries.
            useful for batch optimization and debugging PDFs.
    upw ... user password
    opw ... owner password
 inFile ... input pdf file
outFile ... output pdf file (default: inputPdfFile-opt.pdf)`

	usageSplit     = "usage: pdfcpu split [-verbose] [-upw userpw] [-opw ownerpw] inFile outDir"
	usageLongSplit = `Split generates a set of single page PDFs for the input file in outDir.

verbose ... extensive log output
    upw ... user password
    opw ... owner password
 inFile ... input pdf file
 outDir ... output directory`

	usageMerge     = "usage: pdfcpu merge [-verbose] outFile inFile1 inFile2 ..."
	usageLongMerge = `Merge concatenates a sequence of PDFs/inFiles to outFile.

verbose ... extensive log output
outFile	... output pdf file
inFiles ... a list of at least 2 pdf files subject to concatenation.`

	usageExtract     = "usage: pdfcpu extract [-verbose] -mode image|font|content|page [-pages pageSelection] [-upw userpw] [-opw ownerpw] inFile outDir"
	usageLongExtract = `Extract exports inFile's images, fonts, content or pages into outDir.

verbose ... extensive log output
   mode ... extraction mode
  pages ... page selection
    upw ... user password
    opw ... owner password
 inFile ... input pdf file
 outDir ... output directory

 The extraction modes are:

  image ... extract images (supported PDF filters: DCTDecode, JPXDecode)
   font ... extract font files (supported font types: TrueType)
content ... extract raw page content
   page ... extract single page PDFs`

	usageTrim     = "usage: pdfcpu trim [-verbose] -pages pageSelection [-upw userpw] [-opw ownerpw] inFile outFile"
	usageLongTrim = `Trim generates a trimmed version of inFile for selectedPages.

verbose ... extensive log output
  pages ... page selection
    upw ... user password
    opw ... owner password
 inFile ... input pdf file 
outFile ... output pdf file, the trimmed version of inFile`

	usagePageSelection = `pageSelection selects pages for processing and is a comma separated list of expressions:

Valid expressions are:

  # ... include page #               #-# ... include page range
 !# ... exclude page #              !#-# ... exclude page range
 n# ... exclude page #              n#-# ... exclude page range

 #- ... include page # - last page    -# ... include first page - page #
!#- ... exclude page # - last page   !-# ... exclude first page - page #
n#- ... exclude page # - last page   n-# ... exclude first page - page #

n serves as an alternative for !, since ! needs to be escaped with single quotes on the cmd line.

Valid pageSelections e.g. -3,5,7- or 4-7,!6 or 1-,!5

A missing pageSelection means all pages are selected for generation.`

	usageVersion     = "usage: pdfcpu version"
	usageLongVersion = "Version prints the pdfcpu version"
)

var (
	fileStats, mode, pageSelection string
	in, out                        string
	upw, opw                       string
	verbose                        bool
	logInfo                        *log.Logger

	needStackTrace = true
)

func init() {

	flag.StringVar(&fileStats, "stats", "", "optimize: a csv file for stats appending")
	flag.StringVar(&fileStats, "s", "", "optimize: a csv file for stats appending")

	flag.StringVar(&mode, "mode", "", "validate: strict|relaxed; extract: image|font|content|page")
	flag.StringVar(&mode, "m", "", "validate: strict|relaxed; extract: image|font|content|page")

	flag.StringVar(&pageSelection, "pages", "", "a comma separated list of pages or page ranges, see pdfcpu help split/extract")
	flag.StringVar(&pageSelection, "p", "", "a comma separated list of pages or page ranges, see pdfcpu help split/extract")

	flag.BoolVar(&verbose, "verbose", false, "")
	flag.BoolVar(&verbose, "v", false, "")

	flag.StringVar(&upw, "upw", "", "user password")
	flag.StringVar(&opw, "opw", "", "owner password")

	logInfo = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func ensurePdfExtension(fileName string) {
	if !strings.HasSuffix(fileName, ".pdf") {
		log.Fatalf("%s needs extension \".pdf\".", fileName)
	}
}

func defaultFilenameOut(fileName string) string {
	ensurePdfExtension(fileName)
	return strings.TrimSuffix(fileName, ".pdf") + "_new.pdf"
}

func version() {
	fmt.Printf("pdfcpu version %s\n", write.PdfcpuVersion)
}

func help() {

	if len(flag.Args()) == 0 {
		fmt.Println(usage)
		return
	}

	if len(flag.Args()) > 1 {
		fmt.Printf("usage: pdfcpu help command\n\nToo many arguments given.\n")
		return
	}

	topic := flag.Arg(0)

	switch topic {

	case "validate":
		fmt.Printf("%s\n\n%s\n", usageValidate, usageLongValidate)

	case "optimize":
		fmt.Printf("%s\n\n%s\n", usageOptimize, usageLongOptimize)

	case "split":
		fmt.Printf("%s\n\n%s\n", usageSplit, usageLongSplit)

	case "merge":
		fmt.Printf("%s\n\n%s\n", usageMerge, usageLongMerge)

	case "extract":
		fmt.Printf("%s\n\n%s\n\n%s\n", usageExtract, usageLongExtract, usagePageSelection)

	case "trim":
		fmt.Printf("%s\n\n%s\n\n%s\n", usageTrim, usageLongTrim, usagePageSelection)

	case "version":
		fmt.Printf("%s\n\n%s\n", usageVersion, usageLongVersion)

	default:
		fmt.Printf("Unknown help topic `%s`.  Run 'pdfcpu help'.\n", topic)

	}
}

func setupLogging(verbose bool) {

	types.Verbose(verbose)
	crypto.Verbose(verbose)
	read.Verbose(verbose)
	validate.Verbose(verbose)
	optimize.Verbose(verbose)
	write.Verbose(verbose)
	extract.Verbose(verbose)
	merge.Verbose(verbose)
	pdfcpu.Verbose(verbose)

	needStackTrace = verbose
}

func main() {

	if len(os.Args) == 1 {
		fmt.Println(usage)
		return
	}

	// the first argument is the pdfcpu command, start flag processing with 2nd argument.
	flag.CommandLine.Parse(os.Args[2:])

	setupLogging(verbose)

	config := types.NewDefaultConfiguration()
	config.UserPW = upw
	config.OwnerPW = opw

	var cmd pdfcpu.Command

	command := os.Args[1]

	switch command {

	case "validate", "v":

		if command == "v" && len(flag.Args()) == 0 {
			version()
			return
		}

		if len(flag.Args()) == 0 || len(flag.Args()) > 1 || pageSelection != "" {
			fmt.Printf("%s\n\n", usageValidate)
			return
		}

		filenameIn := flag.Arg(0)
		ensurePdfExtension(filenameIn)

		if mode != "" && mode != "strict" && mode != "s" && mode != "relaxed" && mode != "r" {
			fmt.Printf("%s\n\n", usageValidate)
			return
		}

		switch mode {
		case "strict", "s":
			config.SetValidationStrict()
		case "relaxed", "r":
			config.SetValidationRelaxed()
		}

		cmd = pdfcpu.ValidateCommand(filenameIn, config)

	case "optimize", "o":

		if len(flag.Args()) == 0 || len(flag.Args()) > 2 || pageSelection != "" {
			fmt.Printf("%s\n\n", usageOptimize)
			return
		}

		filenameIn := flag.Arg(0)
		ensurePdfExtension(filenameIn)

		filenameOut := defaultFilenameOut(filenameIn)
		if len(flag.Args()) == 2 {
			filenameOut = flag.Arg(1)
			ensurePdfExtension(filenameOut)
		}

		config.StatsFileName = fileStats
		if len(fileStats) > 0 {
			fmt.Printf("stats will be appended to %s\n", fileStats)
		}

		// Always write using 0x0A end of line sequence.
		cmd = pdfcpu.OptimizeCommand(filenameIn, filenameOut, config)

	case "split", "s":

		if len(flag.Args()) != 2 || pageSelection != "" {
			fmt.Printf("%s\n\n", usageSplit)
			return
		}

		filenameIn := flag.Arg(0)
		ensurePdfExtension(filenameIn)

		dirnameOut := flag.Arg(1)

		cmd = pdfcpu.SplitCommand(filenameIn, dirnameOut, config)

	case "merge", "m":

		if len(flag.Args()) < 3 || pageSelection != "" {
			fmt.Printf("%s\n\n", usageMerge)
			return
		}

		filenameOut := ""
		filenamesIn := []string{}
		for i, arg := range flag.Args() {
			if i == 0 {
				filenameOut = arg
				ensurePdfExtension(filenameOut)
				continue
			}
			ensurePdfExtension(arg)
			filenamesIn = append(filenamesIn, arg)
		}

		cmd = pdfcpu.MergeCommand(filenamesIn, filenameOut, config)

	case "extract", "e":

		if len(flag.Args()) != 2 || mode == "" ||
			(mode != "image" && mode != "font" && mode != "page" && mode != "content") &&
				(mode != "i" && mode != "f" && mode != "p" && mode != "c") {
			fmt.Printf("%s\n\n", usageExtract)
			return
		}

		filenameIn := flag.Arg(0)
		ensurePdfExtension(filenameIn)

		dirnameOut := flag.Arg(1)

		pages, err := pdfcpu.ParsePageSelection(pageSelection)
		if err != nil {
			log.Fatalf("extract: problem with flag pageSelection: %v", err)
		}

		switch mode {
		case "image", "i":
			cmd = pdfcpu.ExtractImagesCommand(filenameIn, dirnameOut, pages, config)
		case "font", "f":
			cmd = pdfcpu.ExtractFontsCommand(filenameIn, dirnameOut, pages, config)
		case "page", "p":
			cmd = pdfcpu.ExtractPagesCommand(filenameIn, dirnameOut, pages, config)
		case "content", "c":
			cmd = pdfcpu.ExtractContentCommand(filenameIn, dirnameOut, pages, config)
		}

	case "trim", "t":

		if len(flag.Args()) == 0 || len(flag.Args()) > 2 || pageSelection == "" {
			fmt.Printf("%s\n\n", usageTrim)
			return
		}

		pages, err := pdfcpu.ParsePageSelection(pageSelection)
		if err != nil {
			log.Fatalf("trim: problem with flag pageSelection: %v", err)
		}

		filenameIn := flag.Arg(0)
		ensurePdfExtension(filenameIn)

		filenameOut := defaultFilenameOut(filenameIn)
		if len(flag.Args()) == 2 {
			filenameOut = flag.Arg(1)
			ensurePdfExtension(filenameOut)
		}

		cmd = pdfcpu.TrimCommand(filenameIn, filenameOut, pages, config)

	case "version":

		if len(flag.Args()) != 0 {
			fmt.Printf("%s\n\n", usageVersion)
			return
		}

		version()
		return

	case "help", "h":

		help()
		return

	default:

		fmt.Printf("pdfcpu unknown subcommand \"%s\"\n", command)
		fmt.Println("Run 'pdfcpu help' for usage.")
		return

	}

	err := pdfcpu.Process(&cmd)
	if err != nil {
		if needStackTrace {
			fmt.Printf("Fatal: %+v\n", err)
		} else {
			fmt.Printf("%v\n", err)
		}
		os.Exit(1)
	}

}
