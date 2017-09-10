package pdfcpu

import "github.com/hhrutter/pdfcpu/types"

type commandMode int

// The available commands for the CLI.
const (
	VALIDATE commandMode = iota
	OPTIMIZE
	SPLIT
	MERGE
	EXTRACTIMAGES
	EXTRACTFONTS
	EXTRACTPAGES
	EXTRACTCONTENT
	TRIM
)

// Command represents an execution context.
type Command struct {
	Mode          commandMode          // VALIDATE  OPTIMIZE  SPLIT  MERGE  EXTRACT  TRIM
	InFile        *string              //    *         *        *      -       *      *
	InFiles       *[]string            //    -         -        -      *       -      -
	OutFile       *string              //    -         *        -      *       -      *
	OutDir        *string              //    -         -        *      -       *      -
	PageSelection *[]string            //    -         -        -      -       *      *
	Config        *types.Configuration //    *         *        *      *       *      *
}

// ValidateCommand creates a new ValidateCommand.
func ValidateCommand(pdfFileName string, config *types.Configuration) Command {
	return Command{
		Mode:   VALIDATE,
		InFile: &pdfFileName,
		Config: config}
}

// OptimizeCommand creates a new OptimizeCommand.
func OptimizeCommand(pdfFileNameIn, pdfFileNameOut string, config *types.Configuration) Command {
	return Command{
		Mode:    OPTIMIZE,
		InFile:  &pdfFileNameIn,
		OutFile: &pdfFileNameOut,
		Config:  config}
}

// SplitCommand creates a new SplitCommand.
func SplitCommand(pdfFileNameIn, dirNameOut string, config *types.Configuration) Command {
	return Command{
		Mode:   SPLIT,
		InFile: &pdfFileNameIn,
		OutDir: &dirNameOut,
		Config: config}
}

// MergeCommand creates a new MergeCommand.
func MergeCommand(pdfFileNamesIn []string, pdfFileNameOut string, config *types.Configuration) Command {
	return Command{
		Mode:    MERGE,
		InFiles: &pdfFileNamesIn,
		OutFile: &pdfFileNameOut,
		Config:  config}
}

// ExtractImagesCommand creates a new ExtractImagesCommand.
// (experimental)
func ExtractImagesCommand(pdfFileNameIn, dirNameOut string, pageSelection []string, config *types.Configuration) Command {
	return Command{
		Mode:          EXTRACTIMAGES,
		InFile:        &pdfFileNameIn,
		OutDir:        &dirNameOut,
		PageSelection: &pageSelection,
		Config:        config}
}

// ExtractFontsCommand creates a new ExtractFontsCommand.
// (experimental)
func ExtractFontsCommand(pdfFileNameIn, dirNameOut string, pageSelection []string, config *types.Configuration) Command {
	return Command{
		Mode:          EXTRACTFONTS,
		InFile:        &pdfFileNameIn,
		OutDir:        &dirNameOut,
		PageSelection: &pageSelection,
		Config:        config}
}

// ExtractPagesCommand creates a new ExtractPagesCommand.
func ExtractPagesCommand(pdfFileNameIn, dirNameOut string, pageSelection []string, config *types.Configuration) Command {
	return Command{
		Mode:          EXTRACTPAGES,
		InFile:        &pdfFileNameIn,
		OutDir:        &dirNameOut,
		PageSelection: &pageSelection,
		Config:        config}
}

// ExtractContentCommand creates a new ExtractContentCommand.
func ExtractContentCommand(pdfFileNameIn, dirNameOut string, pageSelection []string, config *types.Configuration) Command {
	return Command{
		Mode:          EXTRACTCONTENT,
		InFile:        &pdfFileNameIn,
		OutDir:        &dirNameOut,
		PageSelection: &pageSelection,
		Config:        config}
}

// TrimCommand creates a new TrimCommand.
func TrimCommand(pdfFileNameIn, pdfFileNameOut string, pageSelection []string, config *types.Configuration) Command {
	// A slice parameter can be called with nil => empty slice.
	return Command{
		Mode:          TRIM,
		InFile:        &pdfFileNameIn,
		OutFile:       &pdfFileNameOut,
		PageSelection: &pageSelection,
		Config:        config}
}

// Process executes a pdfcpu command.
func Process(cmd *Command) (err error) {

	switch cmd.Mode {

	case VALIDATE:
		err = Validate(*cmd.InFile, cmd.Config)

	case OPTIMIZE:
		err = Optimize(*cmd.InFile, *cmd.OutFile, cmd.Config)

	case SPLIT:
		err = Split(*cmd.InFile, *cmd.OutDir, cmd.Config)

	case MERGE:
		err = Merge(*cmd.InFiles, *cmd.OutFile, cmd.Config)

	case EXTRACTIMAGES:
		err = ExtractImages(*cmd.InFile, *cmd.OutDir, cmd.PageSelection, cmd.Config)

	case EXTRACTFONTS:
		err = ExtractFonts(*cmd.InFile, *cmd.OutDir, cmd.PageSelection, cmd.Config)

	case EXTRACTPAGES:
		err = ExtractPages(*cmd.InFile, *cmd.OutDir, cmd.PageSelection, cmd.Config)

	case EXTRACTCONTENT:
		err = ExtractContent(*cmd.InFile, *cmd.OutDir, cmd.PageSelection, cmd.Config)

	case TRIM:
		err = Trim(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Config)
	}

	return
}
