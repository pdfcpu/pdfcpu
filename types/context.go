package types

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
)

// PDFContext represents the context for processing PDF files.
type PDFContext struct {
	*Configuration
	*XRefTable
	Read     *ReadContext
	Optimize *OptimizationContext
	Write    *WriteContext
}

// NewPDFContext initializes a new PDFContext.
func NewPDFContext(fileName string, file *os.File, config *Configuration) (*PDFContext, error) {

	if config == nil {
		config = NewDefaultConfiguration()
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	ctx := &PDFContext{
		config,
		newXRefTable(config.ValidationMode),
		newReadContext(fileName, file, fileInfo.Size()),
		newOptimizationContext(),
		NewWriteContext(config.Eol),
	}

	return ctx, nil
}

// ResetWriteContext prepares an existing WriteContext for a new file to be written.
func (ctx *PDFContext) ResetWriteContext() {

	ctx.Write = NewWriteContext(ctx.Write.Eol)
}

func (ctx *PDFContext) String() string {

	var logStr []string

	logStr = append(logStr, "*************************************************************************************************\n")
	logStr = append(logStr, fmt.Sprintf("HeaderVersion: %s\n", VersionString(*ctx.HeaderVersion)))

	if ctx.RootVersion != nil {
		logStr = append(logStr, fmt.Sprintf("RootVersion: %s\n", VersionString(*ctx.RootVersion)))
	}

	logStr = append(logStr, fmt.Sprintf("has %d pages\n", ctx.PageCount))

	if ctx.Read.UsingObjectStreams {
		logStr = append(logStr, "using object streams\n")
	}

	if ctx.Read.UsingXRefStreams {
		logStr = append(logStr, "using xref streams\n")
	}

	if ctx.Read.Linearized {
		logStr = append(logStr, "is linearized file\n")
	}

	if ctx.Read.Hybrid {
		logStr = append(logStr, "is hybrid reference file\n")
	}

	if ctx.Tagged {
		logStr = append(logStr, "is tagged file\n")
	}

	logStr = append(logStr, "XRefTable:\n")
	logStr = append(logStr, fmt.Sprintf("                     Size: %d\n", *ctx.XRefTable.Size))
	logStr = append(logStr, fmt.Sprintf("              Root object: %s\n", *ctx.Root))

	if ctx.Info != nil {
		logStr = append(logStr, fmt.Sprintf("              Info object: %s\n", *ctx.Info))
	}

	if ctx.ID != nil {
		logStr = append(logStr, fmt.Sprintf("                ID object: %s\n", *ctx.ID))
	}

	if ctx.Encrypt != nil {
		logStr = append(logStr, fmt.Sprintf("           Encrypt object: %s\n", *ctx.Encrypt))
	}

	if ctx.AdditionalStreams != nil && len(*ctx.AdditionalStreams) > 0 {

		var objectNumbers []string
		for _, k := range *ctx.AdditionalStreams {
			indRef, _ := k.(PDFIndirectRef)
			objectNumbers = append(objectNumbers, fmt.Sprintf("%d", int(indRef.ObjectNumber)))
		}
		sort.Strings(objectNumbers)

		logStr = append(logStr, fmt.Sprintf("        AdditionalStreams: %s\n\n", strings.Join(objectNumbers, ",")))
	}

	logStr = append(logStr, fmt.Sprintf("XRefTable with %d entres:\n", len(ctx.Table)))

	// Print sorted object list.
	logStr = ctx.list(logStr)

	// Print free list.
	logStr, err := ctx.freeList(logStr)
	if err != nil {
		log.Fatal(err)
	}

	// Print list of any missing objects.
	if len(ctx.XRefTable.Table) != *ctx.XRefTable.Size {
		missing, s := ctx.MissingObjects()
		logStr = append(logStr, fmt.Sprintf("%d missing objects: %s\n", missing, *s))
	}

	logStr = append(logStr, fmt.Sprintf("\nTotal pages: %d\n", ctx.PageCount))

	logStr = ctx.Optimize.collectFontInfo(logStr)
	logStr = ctx.Optimize.collectImageInfo(logStr)

	logStr = append(logStr, "\n")

	return strings.Join(logStr, "")
}
