package pdflib

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
	Mode           commandMode // VALIDATE  OPTIMIZE  SPLIT  MERGE  EXTRACT  TRIM
	InFile         *string     //    *         *        *      -       *      *
	InFiles        *[]string   //    -         -        -      *       -      -
	OutFile        *string     //    -         *        -      *       -      *
	OutDir         *string     //    -         -        *      -       *      -
	PageSelection  *[]string   //    -         -        -      -       *      *
	StatsFile      *string     //    -         *        -      -       -      -
	ValidationMode *string     //    *         -        -      -       -      -
}

// ValidateCommand creates a new ValidateCommand.
func ValidateCommand(pdfFileName, validationMode string) Command {
	vMode := "strict"
	if len(validationMode) > 0 {
		vMode = validationMode
	}
	return Command{
		Mode:           VALIDATE,
		InFile:         &pdfFileName,
		ValidationMode: &vMode}
}

// OptimizeCommand creates a new OptimizeCommand.
func OptimizeCommand(pdfFileNameIn, pdfFileNameOut, statsFileName string) Command {
	return Command{
		Mode:      OPTIMIZE,
		InFile:    &pdfFileNameIn,
		OutFile:   &pdfFileNameOut,
		StatsFile: &statsFileName}
}

// SplitCommand creates a new SplitCommand.
func SplitCommand(pdfFileNameIn, dirNameOut string) Command {
	return Command{
		Mode:   SPLIT,
		InFile: &pdfFileNameIn,
		OutDir: &dirNameOut}
}

// MergeCommand creates a new MergeCommand.
func MergeCommand(pdfFileNamesIn []string, pdfFileNameOut string) Command {
	return Command{
		Mode:    MERGE,
		InFiles: &pdfFileNamesIn,
		OutFile: &pdfFileNameOut}
}

// ExtractImagesCommand creates a new ExtractImagesCommand.
// (experimental)
func ExtractImagesCommand(pdfFileNameIn, dirNameOut string, pageSelection []string) Command {
	return Command{
		Mode:          EXTRACTIMAGES,
		InFile:        &pdfFileNameIn,
		OutDir:        &dirNameOut,
		PageSelection: &pageSelection}
}

// ExtractFontsCommand creates a new ExtractFontsCommand.
// (experimental)
func ExtractFontsCommand(pdfFileNameIn, dirNameOut string, pageSelection []string) Command {
	return Command{
		Mode:          EXTRACTFONTS,
		InFile:        &pdfFileNameIn,
		OutDir:        &dirNameOut,
		PageSelection: &pageSelection}
}

// ExtractPagesCommand creates a new ExtractPagesCommand.
func ExtractPagesCommand(pdfFileNameIn, dirNameOut string, pageSelection []string) Command {
	return Command{
		Mode:          EXTRACTPAGES,
		InFile:        &pdfFileNameIn,
		OutDir:        &dirNameOut,
		PageSelection: &pageSelection}
}

// ExtractContentCommand creates a new ExtractContentCommand.
func ExtractContentCommand(pdfFileNameIn, dirNameOut string, pageSelection []string) Command {
	return Command{
		Mode:          EXTRACTCONTENT,
		InFile:        &pdfFileNameIn,
		OutDir:        &dirNameOut,
		PageSelection: &pageSelection}
}

// TrimCommand creates a new TrimCommand.
func TrimCommand(pdfFileNameIn, pdfFileNameOut string, pageSelection []string) Command {
	// A slice parameter can be called with nil => empty slice.
	return Command{
		Mode:          TRIM,
		InFile:        &pdfFileNameIn,
		OutFile:       &pdfFileNameOut,
		PageSelection: &pageSelection}
}

// Process executes a pdflib command.
func Process(cmd *Command) (err error) {

	switch cmd.Mode {

	case VALIDATE:
		err = Validate(*cmd.InFile, *cmd.ValidationMode)

	case OPTIMIZE:
		err = Optimize(*cmd.InFile, *cmd.OutFile, *cmd.StatsFile)

	case SPLIT:
		err = Split(*cmd.InFile, *cmd.OutDir)

	case MERGE:
		err = Merge(*cmd.InFiles, *cmd.OutFile)

	case EXTRACTIMAGES:
		err = ExtractImages(*cmd.InFile, *cmd.OutDir, cmd.PageSelection)

	case EXTRACTFONTS:
		err = ExtractFonts(*cmd.InFile, *cmd.OutDir, cmd.PageSelection)

	case EXTRACTPAGES:
		err = ExtractPages(*cmd.InFile, *cmd.OutDir, cmd.PageSelection)

	case EXTRACTCONTENT:
		err = ExtractContent(*cmd.InFile, *cmd.OutDir, cmd.PageSelection)

	case TRIM:
		err = Trim(*cmd.InFile, *cmd.OutFile, cmd.PageSelection)
	}

	return
}
