package pdfcpu

import (
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

// Command represents an execution context.
type Command struct {
	Mode          types.CommandMode    // VALIDATE  OPTIMIZE  SPLIT  MERGE  EXTRACT  TRIM  LISTATT ADDATT REMATT EXTATT  ENCRYPT  DECRYPT  CHANGEUPW  CHANGEOPW LISTP ADDP
	InFile        *string              //    *         *        *      -       *      *      *       *       *      *       *        *         *          *       *     *
	InFiles       []string             //    -         -        -      *       -      -      -       *       *      *       -        -         -          -       -     -
	InDir         *string              //    -         -        -      -       -      -      -       -       -      -       -        -         -          -       -     -
	OutFile       *string              //    -         *        -      *       -      *      -       -       -      -       *        *         *          *       -     -
	OutDir        *string              //    -         -        *      -       *      -      -       -       -      *       -        -         -          -       -     -
	PageSelection []string             //    -         -        -      -       *      *      -       -       -      -       -        -         -          -       -     -
	Config        *types.Configuration //    *         *        *      *       *      *      *       *       *      *       *        *         *          *       *     *
	PWOld         *string              //    -         -        -      -       -      -      -       -       -      -       -        -         *          *       -     -
	PWNew         *string              //    -         -        -      -       -      -      -       -       -      -       -        -         *          *       -     -
}

// Process executes a pdfcpu command.
func Process(cmd *Command) (out []string, err error) {

	cmd.Config.Mode = cmd.Mode

	switch cmd.Mode {

	case types.VALIDATE:
		err = Validate(*cmd.InFile, cmd.Config)

	case types.OPTIMIZE:
		err = Optimize(*cmd.InFile, *cmd.OutFile, cmd.Config)

	case types.SPLIT:
		err = Split(*cmd.InFile, *cmd.OutDir, cmd.Config)

	case types.MERGE:
		err = Merge(cmd.InFiles, *cmd.OutFile, cmd.Config)

	case types.EXTRACTIMAGES:
		err = ExtractImages(*cmd.InFile, *cmd.OutDir, cmd.PageSelection, cmd.Config)

	case types.EXTRACTFONTS:
		err = ExtractFonts(*cmd.InFile, *cmd.OutDir, cmd.PageSelection, cmd.Config)

	case types.EXTRACTPAGES:
		err = ExtractPages(*cmd.InFile, *cmd.OutDir, cmd.PageSelection, cmd.Config)

	case types.EXTRACTCONTENT:
		err = ExtractContent(*cmd.InFile, *cmd.OutDir, cmd.PageSelection, cmd.Config)

	case types.TRIM:
		err = Trim(*cmd.InFile, *cmd.OutFile, cmd.PageSelection, cmd.Config)

	case types.LISTATTACHMENTS, types.ADDATTACHMENTS, types.REMOVEATTACHMENTS, types.EXTRACTATTACHMENTS:
		out, err = processAttachments(cmd)

	case types.ENCRYPT, types.DECRYPT, types.CHANGEUPW, types.CHANGEOPW:
		err = processEncryption(cmd)

	case types.LISTPERMISSIONS, types.ADDPERMISSIONS:
		out, err = processPermissions(cmd)

	default:
		err = errors.Errorf("Process: Unknown command mode %d\n", cmd.Mode)
	}

	return out, err
}

// ValidateCommand creates a new ValidateCommand.
func ValidateCommand(pdfFileName string, config *types.Configuration) *Command {
	return &Command{
		Mode:   types.VALIDATE,
		InFile: &pdfFileName,
		Config: config}
}

// OptimizeCommand creates a new OptimizeCommand.
func OptimizeCommand(pdfFileNameIn, pdfFileNameOut string, config *types.Configuration) *Command {
	return &Command{
		Mode:    types.OPTIMIZE,
		InFile:  &pdfFileNameIn,
		OutFile: &pdfFileNameOut,
		Config:  config}
}

// SplitCommand creates a new SplitCommand.
func SplitCommand(pdfFileNameIn, dirNameOut string, config *types.Configuration) *Command {
	return &Command{
		Mode:   types.SPLIT,
		InFile: &pdfFileNameIn,
		OutDir: &dirNameOut,
		Config: config}
}

// MergeCommand creates a new MergeCommand.
func MergeCommand(pdfFileNamesIn []string, pdfFileNameOut string, config *types.Configuration) *Command {
	return &Command{
		Mode: types.MERGE,
		//InFile:  &pdfFileNameIn,
		InFiles: pdfFileNamesIn,
		OutFile: &pdfFileNameOut,
		Config:  config}
}

// ExtractImagesCommand creates a new ExtractImagesCommand.
// (experimental)
func ExtractImagesCommand(pdfFileNameIn, dirNameOut string, pageSelection []string, config *types.Configuration) *Command {
	return &Command{
		Mode:          types.EXTRACTIMAGES,
		InFile:        &pdfFileNameIn,
		OutDir:        &dirNameOut,
		PageSelection: pageSelection,
		Config:        config}
}

// ExtractFontsCommand creates a new ExtractFontsCommand.
// (experimental)
func ExtractFontsCommand(pdfFileNameIn, dirNameOut string, pageSelection []string, config *types.Configuration) *Command {
	return &Command{
		Mode:          types.EXTRACTFONTS,
		InFile:        &pdfFileNameIn,
		OutDir:        &dirNameOut,
		PageSelection: pageSelection,
		Config:        config}
}

// ExtractPagesCommand creates a new ExtractPagesCommand.
func ExtractPagesCommand(pdfFileNameIn, dirNameOut string, pageSelection []string, config *types.Configuration) *Command {
	return &Command{
		Mode:          types.EXTRACTPAGES,
		InFile:        &pdfFileNameIn,
		OutDir:        &dirNameOut,
		PageSelection: pageSelection,
		Config:        config}
}

// ExtractContentCommand creates a new ExtractContentCommand.
func ExtractContentCommand(pdfFileNameIn, dirNameOut string, pageSelection []string, config *types.Configuration) *Command {
	return &Command{
		Mode:          types.EXTRACTCONTENT,
		InFile:        &pdfFileNameIn,
		OutDir:        &dirNameOut,
		PageSelection: pageSelection,
		Config:        config}
}

// TrimCommand creates a new TrimCommand.
func TrimCommand(pdfFileNameIn, pdfFileNameOut string, pageSelection []string, config *types.Configuration) *Command {
	// A slice parameter may be called with nil => empty slice.
	return &Command{
		Mode:          types.TRIM,
		InFile:        &pdfFileNameIn,
		OutFile:       &pdfFileNameOut,
		PageSelection: pageSelection,
		Config:        config}
}

// ListAttachmentsCommand create a new ListAttachmentsCommand.
func ListAttachmentsCommand(pdfFileNameIn string, config *types.Configuration) *Command {
	return &Command{
		Mode:   types.LISTATTACHMENTS,
		InFile: &pdfFileNameIn,
		Config: config}
}

// AddAttachmentsCommand creates a new AddAttachmentsCommand.
func AddAttachmentsCommand(pdfFileNameIn string, fileNamesIn []string, config *types.Configuration) *Command {
	return &Command{
		Mode:    types.ADDATTACHMENTS,
		InFile:  &pdfFileNameIn,
		InFiles: fileNamesIn,
		Config:  config}
}

// RemoveAttachmentsCommand creates a new RemoveAttachmentsCommand.
func RemoveAttachmentsCommand(pdfFileNameIn string, fileNamesIn []string, config *types.Configuration) *Command {
	return &Command{
		Mode:    types.REMOVEATTACHMENTS,
		InFile:  &pdfFileNameIn,
		InFiles: fileNamesIn,
		Config:  config}
}

// ExtractAttachmentsCommand creates a new ExtractAttachmentsCommand.
func ExtractAttachmentsCommand(pdfFileNameIn, dirNameOut string, fileNamesIn []string, config *types.Configuration) *Command {
	return &Command{
		Mode:    types.EXTRACTATTACHMENTS,
		InFile:  &pdfFileNameIn,
		OutDir:  &dirNameOut,
		InFiles: fileNamesIn,
		Config:  config}
}

// EncryptCommand creates a new EncryptCommand.
func EncryptCommand(pdfFileNameIn, pdfFileNameOut string, config *types.Configuration) *Command {
	return &Command{
		Mode:    types.ENCRYPT,
		InFile:  &pdfFileNameIn,
		OutFile: &pdfFileNameOut,
		Config:  config}
}

// DecryptCommand creates a new DecryptCommand.
func DecryptCommand(pdfFileNameIn, pdfFileNameOut string, config *types.Configuration) *Command {
	return &Command{
		Mode:    types.DECRYPT,
		InFile:  &pdfFileNameIn,
		OutFile: &pdfFileNameOut,
		Config:  config}
}

// ChangeUserPWCommand creates a new ChangeUserPWCommand.
func ChangeUserPWCommand(pdfFileNameIn, pdfFileNameOut string, config *types.Configuration, pwOld, pwNew *string) *Command {
	return &Command{
		Mode:    types.CHANGEUPW,
		InFile:  &pdfFileNameIn,
		OutFile: &pdfFileNameOut,
		Config:  config,
		PWOld:   pwOld,
		PWNew:   pwNew}
}

// ChangeOwnerPWCommand creates a new ChangeOwnerPWCommand.
func ChangeOwnerPWCommand(pdfFileNameIn, pdfFileNameOut string, config *types.Configuration, pwOld, pwNew *string) *Command {
	return &Command{
		Mode:    types.CHANGEOPW,
		InFile:  &pdfFileNameIn,
		OutFile: &pdfFileNameOut,
		Config:  config,
		PWOld:   pwOld,
		PWNew:   pwNew}
}

// ListPermissionsCommand create a new ListPermissionsCommand.
func ListPermissionsCommand(pdfFileNameIn string, config *types.Configuration) *Command {
	return &Command{
		Mode:   types.LISTPERMISSIONS,
		InFile: &pdfFileNameIn,
		Config: config}
}

// AddPermissionsCommand creates a new AddPermissionsCommand.
func AddPermissionsCommand(pdfFileNameIn string, config *types.Configuration) *Command {
	return &Command{
		Mode:   types.ADDPERMISSIONS,
		InFile: &pdfFileNameIn,
		Config: config}
}

func processAttachments(cmd *Command) (out []string, err error) {

	switch cmd.Mode {

	case types.LISTATTACHMENTS:
		out, err = ListAttachments(*cmd.InFile, cmd.Config)

	case types.ADDATTACHMENTS:
		err = AddAttachments(*cmd.InFile, cmd.InFiles, cmd.Config)

	case types.REMOVEATTACHMENTS:
		err = RemoveAttachments(*cmd.InFile, cmd.InFiles, cmd.Config)

	case types.EXTRACTATTACHMENTS:
		err = ExtractAttachments(*cmd.InFile, *cmd.OutDir, cmd.InFiles, cmd.Config)
	}

	return out, err
}

func processEncryption(cmd *Command) (err error) {

	switch cmd.Mode {

	case types.ENCRYPT:
		err = Encrypt(*cmd.InFile, *cmd.OutFile, cmd.Config)

	case types.DECRYPT:
		err = Decrypt(*cmd.InFile, *cmd.OutFile, cmd.Config)

	case types.CHANGEUPW:
		err = ChangeUserPassword(*cmd.InFile, *cmd.OutFile, cmd.Config, cmd.PWOld, cmd.PWNew)

	case types.CHANGEOPW:
		err = ChangeOwnerPassword(*cmd.InFile, *cmd.OutFile, cmd.Config, cmd.PWOld, cmd.PWNew)
	}

	return err
}

func processPermissions(cmd *Command) (out []string, err error) {

	switch cmd.Mode {

	case types.LISTPERMISSIONS:
		out, err = ListPermissions(*cmd.InFile, cmd.Config)

	case types.ADDPERMISSIONS:
		err = AddPermissions(*cmd.InFile, cmd.Config)
	}

	return out, err
}
