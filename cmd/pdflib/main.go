package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hhrutter/pdflib"
	"github.com/hhrutter/pdflib/extract"
	"github.com/hhrutter/pdflib/merge"
	"github.com/hhrutter/pdflib/optimize"
	"github.com/hhrutter/pdflib/read"
	"github.com/hhrutter/pdflib/types"
	"github.com/hhrutter/pdflib/validate"
	"github.com/hhrutter/pdflib/write"
)

const (
	usageValidate = "usage: pdflib validate [-verbose] [-mode strict|relaxed] inFile"
	usageOptimize = "usage: pdflib optimize [-verbose] [-stats csvFile] inFile [outFile]"
	usageSplit    = "usage: pdflib split [-verbose] inFile outDir"
	usageMerge    = "usage: pdflib merge [-verbose] outFile inFile1 inFile2 ..."
	usageExtract  = "usage: pdflib extract [-verbose] -mode image|font|content|page [-pages pageSelection] inFile outDir"
	usageTrim     = "usage: pdflib trim [-verbose] -pages pageSelection inFile outFile"
	usageVersion  = "usage: pdflib version"
)

var (
	fileStats, mode, pageSelection string
	in, out                        string
	verbose                        bool
	logInfo                        *log.Logger

	needStackTrace = true
)

func init() {

	flag.StringVar(&fileStats, "stats", "", "optimize: a csv file for stats appending")
	flag.StringVar(&fileStats, "s", "", "optimize: a csv file for stats appending")

	flag.StringVar(&mode, "mode", "", "validate: strict|relaxed; extract: image|font|content|page")
	flag.StringVar(&mode, "m", "", "validate: strict|relaxed; extract: image|font|content|page")

	flag.StringVar(&pageSelection, "pages", "", "a comma separated list of pages or page ranges, see pdflib help split/extract")
	flag.StringVar(&pageSelection, "p", "", "a comma separated list of pages or page ranges, see pdflib help split/extract")

	flag.BoolVar(&verbose, "verbose", false, "")
	flag.BoolVar(&verbose, "v", false, "")

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

func usage() {
	usageStr := "pdflib is a tool for PDF manipulation written in Go.\n\n" +
		"Usage:\n\n" +
		"	pdflib command [arguments]\n\n" +
		"The commands are:\n\n" +
		"	validate	validate PDF against PDF 32000-1:2008 (PDF 1.7)\n" +
		"	optimize	optimize PDF by getting rid of redundant page resources\n" +
		"	split		split multi-page PDF into several single-page PDFs\n" +
		"	merge		concatenate 2 or more PDFs\n" +
		"	extract		extract images, fonts, content, pages out of a PDF\n" +
		"	trim		create trimmed version of a PDF\n" +
		"	version		print pdflib version\n\n" +
		"	Single-letter Unix-style supported for commands and flags.\n\n" +
		"Use \"pdflib help [command]\" for more information about a command."
	fmt.Println(usageStr)
}

func version() {
	fmt.Printf("pdflib version %s\n", write.PdflibVersion)
}

func usagePageSelection() {
	fmt.Printf("pageSelection selects pages for processing and is a comma separated list of expressions:\n\n")
	fmt.Printf("Valid expressions are:\n\n")
	fmt.Printf("  # ... include page #               #-# ... include page range\n")
	fmt.Printf(" !# ... exclude page #              !#-# ... exclude page range\n")
	fmt.Printf(" n# ... exclude page #              n#-# ... exclude page range\n\n")
	fmt.Printf(" #- ... include page # - last page    -# ... include first page - page #\n")
	fmt.Printf("!#- ... exclude page # - last page   !-# ... exclude first page - page #\n")
	fmt.Printf("n#- ... exclude page # - last page   n-# ... exclude first page - page #\n\n")
	fmt.Printf("n serves as an alternative for !, since ! needs to be escaped with single quotes on the cmd line.\n\n")
	fmt.Printf("Valid pageSelections e.g. \"-3,5,7-\" or \"4-7,!6\" or \"1-,!5\"\n\n")
	fmt.Printf("A missing pageSelection means all pages are selected for generation.\n")

}

func help() {

	if len(flag.Args()) == 0 {
		usage()
		return
	}

	if len(flag.Args()) > 1 {
		fmt.Printf("usage: pdflib help command\n\nToo many arguments given.\n")
		return
	}

	topic := flag.Arg(0)

	switch topic {

	case "validate":
		fmt.Printf("%s\n\n", usageValidate)
		fmt.Printf("Validate checks the inFile for specification compliance.\n\n")
		fmt.Printf("verbose ... extensive log output\n")
		fmt.Printf("   mode ... validation mode\n")
		fmt.Printf(" inFile ... input pdf file\n\n")
		fmt.Printf("The validation modes are:\n\n")
		fmt.Printf(" strict ... (default) validates against PDF 32000-1:2008 (PDF 1.7)\n")
		fmt.Printf("relaxed ... like strict but doesn't complain about common seen spec violations.\n")

	case "optimize":
		fmt.Printf("%s\n\n", usageOptimize)
		fmt.Printf("Optimize reads inFile, removes redundant page resources")
		fmt.Printf(" like embedded fonts and images and writes the result to outFile.\n\n")
		fmt.Printf("verbose ... extensive log output\n")
		fmt.Printf("  stats ... Appends a stats line to a csv file with information about the usage of root and page entries.\n")
		fmt.Printf("            Useful for batch optimization and debugging PDFs.\n")
		fmt.Printf(" inFile ... input pdf file\n")
		fmt.Printf("outFile ... output pdf file (default: inputPdfFile-opt.pdf)\n")

	case "split":
		fmt.Printf("%s\n\n", usageSplit)
		fmt.Printf("Split generates a set of single page PDFs for the input file in outDir.\n\n")
		fmt.Printf("verbose ... extensive log output\n")
		fmt.Printf(" inFile ... input pdf file\n")
		fmt.Printf(" outDir ... output directory\n\n")

	case "merge":
		fmt.Printf("%s\n\n", usageMerge)
		fmt.Printf("Merge concatenates a sequence of PDFs/inFiles to outFile.\n\n")
		fmt.Printf("verbose ... extensive log output\n")
		fmt.Printf("outFile	... output pdf file\n")
		fmt.Printf("inFiles ... a list of at least 2 pdf files subject to concatenation.\n")

	case "extract":
		fmt.Printf("%s\n\n", usageExtract)
		fmt.Printf("Extract exports inFile's images, fonts, content or pages into outDir.\n\n")
		fmt.Printf("verbose ... extensive log output\n")
		fmt.Printf("   mode ... extraction mode\n")
		fmt.Printf("  pages ... page selection\n")
		fmt.Printf(" inFile ... input pdf file\n")
		fmt.Printf(" outDir ... output directory\n\n")
		fmt.Printf("The extraction modes are:\n\n")
		fmt.Printf("  image ... extract images (supported PDF filters: DCTDecode, JPXDecode)\n")
		fmt.Printf("   font ... extract font files (supported font types: TrueType)\n")
		fmt.Printf("content ... extract raw page content\n")
		fmt.Printf("   page ... extract single page PDFs\n\n")
		usagePageSelection()

	case "trim":
		fmt.Printf("%s\n\n", usageTrim)
		fmt.Printf("Trim generates a trimmed version of inFile for selectedPages.\n\n")
		fmt.Printf("verbose ... extensive log output\n")
		fmt.Printf("  pages ... page selection\n")
		fmt.Printf(" inFile ... input pdf file\n")
		fmt.Printf("outFile ... output pdf file, the trimmed version of inFile\n\n")
		usagePageSelection()

	case "version":
		fmt.Printf("usage: pdflib version\n\n")
		fmt.Printf("Version prints the pdflib version\n.")

	default:
		fmt.Printf("Unknown help topic `%s`.  Run 'pdflib help'.\n", topic)

	}
}

func setupLogging(verbose bool) {

	types.Verbose(verbose)
	read.Verbose(verbose)
	validate.Verbose(verbose)
	optimize.Verbose(verbose)
	write.Verbose(verbose)
	extract.Verbose(verbose)
	merge.Verbose(verbose)
	pdflib.Verbose(verbose)

	needStackTrace = verbose
}

func main() {

	if len(os.Args) == 1 {
		usage()
		return
	}

	// the first argument is the pdflib command, start flag processing with 2nd argument.
	flag.CommandLine.Parse(os.Args[2:])

	setupLogging(verbose)

	config := types.NewDefaultConfiguration()

	var cmd pdflib.Command

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

		cmd = pdflib.ValidateCommand(filenameIn, config)

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
			//logInfo.Printf("stats will be appended to %s\n", fileStats)
		}

		// Always write using 0x0A end of line sequence.
		cmd = pdflib.OptimizeCommand(filenameIn, filenameOut, config)

	case "split", "s":

		if len(flag.Args()) != 2 || pageSelection != "" {
			fmt.Printf("%s\n\n", usageSplit)
			return
		}

		filenameIn := flag.Arg(0)
		ensurePdfExtension(filenameIn)

		dirnameOut := flag.Arg(1)

		cmd = pdflib.SplitCommand(filenameIn, dirnameOut, config)

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

		cmd = pdflib.MergeCommand(filenamesIn, filenameOut, config)

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

		pages, err := pdflib.ParsePageSelection(pageSelection)
		if err != nil {
			log.Fatalf("extract: problem with flag pageSelection: %v", err)
		}

		switch mode {
		case "image", "i":
			cmd = pdflib.ExtractImagesCommand(filenameIn, dirnameOut, pages, config)
		case "font", "f":
			cmd = pdflib.ExtractFontsCommand(filenameIn, dirnameOut, pages, config)
		case "page", "p":
			cmd = pdflib.ExtractPagesCommand(filenameIn, dirnameOut, pages, config)
		case "content", "c":
			cmd = pdflib.ExtractContentCommand(filenameIn, dirnameOut, pages, config)
		}

	case "trim", "t":

		if len(flag.Args()) == 0 || len(flag.Args()) > 2 || pageSelection == "" {
			fmt.Printf("%s\n\n", usageTrim)
			return
		}

		pages, err := pdflib.ParsePageSelection(pageSelection)
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

		cmd = pdflib.TrimCommand(filenameIn, filenameOut, pages, config)

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

		fmt.Printf("pdflib unknown subcommand \"%s\"\n", command)
		fmt.Println("Run 'pdflib help' for usage.")
		return

	}

	err := pdflib.Process(&cmd)
	if err != nil {
		if needStackTrace {
			fmt.Printf("Fatal: %+v\n", err)
		} else {
			fmt.Printf("%v\n", err)
		}
		os.Exit(1)
	}

}
