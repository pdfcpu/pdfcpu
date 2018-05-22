package api

import (
	"github.com/hhrutter/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

// Command represents an execution context.
type Command struct {
	Mode          pdfcpu.CommandMode    // VALIDATE  OPTIMIZE  SPLIT  MERGE  EXTRACT  TRIM  LISTATT ADDATT REMATT EXTATT  ENCRYPT  DECRYPT  CHANGEUPW  CHANGEOPW LISTP ADDP
	InFile        *string               //    *         *        *      -       *      *      *       *       *      *       *        *         *          *       *     *
	InFiles       []string              //    -         -        -      *       -      -      -       *       *      *       -        -         -          -       -     -
	InDir         *string               //    -         -        -      -       -      -      -       -       -      -       -        -         -          -       -     -
	OutFile       *string               //    -         *        -      *       -      *      -       -       -      -       *        *         *          *       -     -
	OutDir        *string               //    -         -        *      -       *      -      -       -       -      *       -        -         -          -       -     -
	PageSelection []string              //    -         -        -      -       *      *      -       -       -      -       -        -         -          -       -     -
	Config        *pdfcpu.Configuration //    *         *        *      *       *      *      *       *       *      *       *        *         *          *       *     *
	PWOld         *string               //    -         -        -      -       -      -      -       -       -      -       -        -         *          *       -     -
	PWNew         *string               //    -         -        -      -       -      -      -       -       -      -       -        -         *          *       -     -
}

// Process executes a pdfcpu command.
func Process(cmd *Command) (out []string, err error) {

	cmd.Config.Mode = cmd.Mode

	switch cmd.Mode {

	case pdfcpu.VALIDATE:
		err = Validate(*cmd.InFile, cmd.Config)

	case pdfcpu.OPTIMIZE:
		err = Optimize(*cmd.InFile, *cmd.OutFile, cmd.Config)

	case pdfcpu.SPLIT:
		err = Split(*cmd.InFile, *cmd.OutDir, cmd.Config)

	case pdfcpu.MERGE:
		err = Merge(cmd.InFiles, *cmd.OutFile, cmd.Config)

	case pdfcpu.EXTRACTIMAGES:
		err = ExtractImages(*cmd.InFile, *cmd.OutDir, cmd.PageSelection, cmd.Config)

	case pdfcpu.EXTRACTFONTS:
		err = ExtractFonts(*cmd.InFile, *cmd.OutDir, cmd.PageSelection, cmd.Config)

	case pdfcpu.EXTRACTPAGES:
		err = ExtractPages(*cmd.InFile, *cmd.OutDir, cmd.PageSelection, cmd.Config)

	case pdfcpu.EXTRACTCONTENT:
		err = ExtractContent(*cmd.InFile, *cmd.OutDir, cmd.PageSelection, cmd.Config)

	case pdfcpu.TRIM:
		err = Trim(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Config)

	case pdfcpu.LISTATTACHMENTS, pdfcpu.ADDATTACHMENTS, pdfcpu.REMOVEATTACHMENTS, pdfcpu.EXTRACTATTACHMENTS:
		out, err = processAttachments(cmd)

	case pdfcpu.ENCRYPT, pdfcpu.DECRYPT, pdfcpu.CHANGEUPW, pdfcpu.CHANGEOPW:
		err = processEncryption(cmd)

	case pdfcpu.LISTPERMISSIONS, pdfcpu.ADDPERMISSIONS:
		out, err = processPermissions(cmd)

	default:
		err = errors.Errorf("Process: Unknown command mode %d\n", cmd.Mode)
	}

	return out, err
}

// ValidateCommand creates a new ValidateCommand.
func ValidateCommand(pdfFileName string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:   pdfcpu.VALIDATE,
		InFile: &pdfFileName,
		Config: config}
}

// OptimizeCommand creates a new OptimizeCommand.
func OptimizeCommand(pdfFileNameIn, pdfFileNameOut string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:    pdfcpu.OPTIMIZE,
		InFile:  &pdfFileNameIn,
		OutFile: &pdfFileNameOut,
		Config:  config}
}

// SplitCommand creates a new SplitCommand.
func SplitCommand(pdfFileNameIn, dirNameOut string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:   pdfcpu.SPLIT,
		InFile: &pdfFileNameIn,
		OutDir: &dirNameOut,
		Config: config}
}

// MergeCommand creates a new MergeCommand.
func MergeCommand(pdfFileNamesIn []string, pdfFileNameOut string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode: pdfcpu.MERGE,
		//InFile:  &pdfFileNameIn,
		InFiles: pdfFileNamesIn,
		OutFile: &pdfFileNameOut,
		Config:  config}
}

// ExtractImagesCommand creates a new ExtractImagesCommand.
// (experimental)
func ExtractImagesCommand(pdfFileNameIn, dirNameOut string, pageSelection []string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:          pdfcpu.EXTRACTIMAGES,
		InFile:        &pdfFileNameIn,
		OutDir:        &dirNameOut,
		PageSelection: pageSelection,
		Config:        config}
}

// ExtractFontsCommand creates a new ExtractFontsCommand.
// (experimental)
func ExtractFontsCommand(pdfFileNameIn, dirNameOut string, pageSelection []string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:          pdfcpu.EXTRACTFONTS,
		InFile:        &pdfFileNameIn,
		OutDir:        &dirNameOut,
		PageSelection: pageSelection,
		Config:        config}
}

// ExtractPagesCommand creates a new ExtractPagesCommand.
func ExtractPagesCommand(pdfFileNameIn, dirNameOut string, pageSelection []string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:          pdfcpu.EXTRACTPAGES,
		InFile:        &pdfFileNameIn,
		OutDir:        &dirNameOut,
		PageSelection: pageSelection,
		Config:        config}
}

// ExtractContentCommand creates a new ExtractContentCommand.
func ExtractContentCommand(pdfFileNameIn, dirNameOut string, pageSelection []string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:          pdfcpu.EXTRACTCONTENT,
		InFile:        &pdfFileNameIn,
		OutDir:        &dirNameOut,
		PageSelection: pageSelection,
		Config:        config}
}

// TrimCommand creates a new TrimCommand.
func TrimCommand(pdfFileNameIn, pdfFileNameOut string, pageSelection []string, config *pdfcpu.Configuration) *Command {
	// A slice parameter may be called with nil => empty slice.
	return &Command{
		Mode:          pdfcpu.TRIM,
		InFile:        &pdfFileNameIn,
		OutFile:       &pdfFileNameOut,
		PageSelection: pageSelection,
		Config:        config}
}

// ListAttachmentsCommand create a new ListAttachmentsCommand.
func ListAttachmentsCommand(pdfFileNameIn string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:   pdfcpu.LISTATTACHMENTS,
		InFile: &pdfFileNameIn,
		Config: config}
}

// AddAttachmentsCommand creates a new AddAttachmentsCommand.
func AddAttachmentsCommand(pdfFileNameIn string, fileNamesIn []string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:    pdfcpu.ADDATTACHMENTS,
		InFile:  &pdfFileNameIn,
		InFiles: fileNamesIn,
		Config:  config}
}

// RemoveAttachmentsCommand creates a new RemoveAttachmentsCommand.
func RemoveAttachmentsCommand(pdfFileNameIn string, fileNamesIn []string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:    pdfcpu.REMOVEATTACHMENTS,
		InFile:  &pdfFileNameIn,
		InFiles: fileNamesIn,
		Config:  config}
}

// ExtractAttachmentsCommand creates a new ExtractAttachmentsCommand.
func ExtractAttachmentsCommand(pdfFileNameIn, dirNameOut string, fileNamesIn []string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:    pdfcpu.EXTRACTATTACHMENTS,
		InFile:  &pdfFileNameIn,
		OutDir:  &dirNameOut,
		InFiles: fileNamesIn,
		Config:  config}
}

// EncryptCommand creates a new EncryptCommand.
func EncryptCommand(pdfFileNameIn, pdfFileNameOut string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:    pdfcpu.ENCRYPT,
		InFile:  &pdfFileNameIn,
		OutFile: &pdfFileNameOut,
		Config:  config}
}

// DecryptCommand creates a new DecryptCommand.
func DecryptCommand(pdfFileNameIn, pdfFileNameOut string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:    pdfcpu.DECRYPT,
		InFile:  &pdfFileNameIn,
		OutFile: &pdfFileNameOut,
		Config:  config}
}

// ChangeUserPWCommand creates a new ChangeUserPWCommand.
func ChangeUserPWCommand(pdfFileNameIn, pdfFileNameOut string, config *pdfcpu.Configuration, pwOld, pwNew *string) *Command {
	return &Command{
		Mode:    pdfcpu.CHANGEUPW,
		InFile:  &pdfFileNameIn,
		OutFile: &pdfFileNameOut,
		Config:  config,
		PWOld:   pwOld,
		PWNew:   pwNew}
}

// ChangeOwnerPWCommand creates a new ChangeOwnerPWCommand.
func ChangeOwnerPWCommand(pdfFileNameIn, pdfFileNameOut string, config *pdfcpu.Configuration, pwOld, pwNew *string) *Command {
	return &Command{
		Mode:    pdfcpu.CHANGEOPW,
		InFile:  &pdfFileNameIn,
		OutFile: &pdfFileNameOut,
		Config:  config,
		PWOld:   pwOld,
		PWNew:   pwNew}
}

// ListPermissionsCommand create a new ListPermissionsCommand.
func ListPermissionsCommand(pdfFileNameIn string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:   pdfcpu.LISTPERMISSIONS,
		InFile: &pdfFileNameIn,
		Config: config}
}

// AddPermissionsCommand creates a new AddPermissionsCommand.
func AddPermissionsCommand(pdfFileNameIn string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:   pdfcpu.ADDPERMISSIONS,
		InFile: &pdfFileNameIn,
		Config: config}
}

func processAttachments(cmd *Command) (out []string, err error) {

	switch cmd.Mode {

	case pdfcpu.LISTATTACHMENTS:
		out, err = ListAttachments(*cmd.InFile, cmd.Config)

	case pdfcpu.ADDATTACHMENTS:
		err = AddAttachments(*cmd.InFile, cmd.InFiles, cmd.Config)

	case pdfcpu.REMOVEATTACHMENTS:
		err = RemoveAttachments(*cmd.InFile, cmd.InFiles, cmd.Config)

	case pdfcpu.EXTRACTATTACHMENTS:
		err = ExtractAttachments(*cmd.InFile, *cmd.OutDir, cmd.InFiles, cmd.Config)
	}

	return out, err
}

func processEncryption(cmd *Command) (err error) {

	switch cmd.Mode {

	case pdfcpu.ENCRYPT:
		err = Encrypt(*cmd.InFile, *cmd.OutFile, cmd.Config)

	case pdfcpu.DECRYPT:
		err = Decrypt(*cmd.InFile, *cmd.OutFile, cmd.Config)

	case pdfcpu.CHANGEUPW:
		err = ChangeUserPassword(*cmd.InFile, *cmd.OutFile, cmd.Config, cmd.PWOld, cmd.PWNew)

	case pdfcpu.CHANGEOPW:
		err = ChangeOwnerPassword(*cmd.InFile, *cmd.OutFile, cmd.Config, cmd.PWOld, cmd.PWNew)
	}

	return err
}

func processPermissions(cmd *Command) (out []string, err error) {

	switch cmd.Mode {

	case pdfcpu.LISTPERMISSIONS:
		out, err = ListPermissions(*cmd.InFile, cmd.Config)

	case pdfcpu.ADDPERMISSIONS:
		err = AddPermissions(*cmd.InFile, cmd.Config)
	}

	return out, err
}
