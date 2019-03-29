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

package api

import (
	pdf "github.com/hhrutter/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

// Command represents an execution context.
type Command struct {
	Mode          pdf.CommandMode    // VALIDATE  OPTIMIZE  SPLIT  MERGE  EXTRACT  TRIM  LISTATT ADDATT REMATT EXTATT  ENCRYPT  DECRYPT  CHANGEUPW  CHANGEOPW LISTP ADDP  WATERMARK  IMPORT  INSERTP REMOVEP ROTATE  NUP
	InFile        *string            //    *         *        *      -       *      *      *       *       *      *       *        *         *          *       *     *       *         -       *       *       *     -
	InFiles       []string           //    -         -        -      *       -      -      -       *       *      *       -        -         -          -       -     -       -         *       -       -       -     *
	InDir         *string            //    -         -        -      -       -      -      -       -       -      -       -        -         -          -       -     -       -         -       -       -       -     -
	OutFile       *string            //    -         *        -      *       -      *      -       -       -      -       *        *         *          *       -     -       *         *       *       *       -     *
	OutDir        *string            //    -         -        *      -       *      -      -       -       -      *       -        -         -          -       -     -       -         -       -       -       -     -
	PageSelection []string           //    -         -        -      -       *      *      -       -       -      -       -        -         -          -       -     -       *         -       *       *       -     *
	Config        *pdf.Configuration //    *         *        *      *       *      *      *       *       *      *       *        *         *          *       *     *       *         *       *       *       *     *
	PWOld         *string            //    -         -        -      -       -      -      -       -       -      -       -        -         *          *       -     -       -         -       -       -       -     -
	PWNew         *string            //    -         -        -      -       -      -      -       -       -      -       -        -         *          *       -     -       -         -       -       -       -     -
	Watermark     *pdf.Watermark     //    -         -        -      -       -      -      -       -       -      -       -        -         -          -       -     -       -         -       -       -       -     -
	Span          int                //    -         -        *      -       -      -      -       -       -      -       -        -         -          -       -     -       -         -       -       -       -     -
	Import        *pdf.Import        //    -         -        -      -       -      -      -       -       -      -       -        -         -          -       -     -       -         *       -       -       -     -
	Rotation      int                //    -         -        -      -       -      -      -       -       -      -       -        -         -          -       -     -       -         -       -       -       *     -
	NUp           *pdf.NUp           //    -         -        -      -       -      -      -       -       -      -       -        -         -          -       -     -       -         -       -       -       -     *
}

// Process executes a pdfcpu command.
func Process(cmd *Command) (out []string, err error) {

	defer func() {
		if r := recover(); r != nil {
			//fmt.Println(r)
			err = errors.Errorf("unexpected panic attack: %v\n", r)
		}
	}()

	cmd.Config.Cmd = cmd.Mode

	for k, v := range map[pdf.CommandMode]func(cmd *Command) ([]string, error){
		pdf.VALIDATE:           Validate,
		pdf.OPTIMIZE:           Optimize,
		pdf.SPLIT:              Split,
		pdf.MERGE:              Merge,
		pdf.EXTRACTIMAGES:      ExtractImages,
		pdf.EXTRACTFONTS:       ExtractFonts,
		pdf.EXTRACTPAGES:       ExtractPages,
		pdf.EXTRACTCONTENT:     ExtractContent,
		pdf.EXTRACTMETADATA:    ExtractMetadata,
		pdf.TRIM:               Trim,
		pdf.ADDWATERMARKS:      AddWatermarks,
		pdf.LISTATTACHMENTS:    processAttachments,
		pdf.ADDATTACHMENTS:     processAttachments,
		pdf.REMOVEATTACHMENTS:  processAttachments,
		pdf.EXTRACTATTACHMENTS: processAttachments,
		pdf.ENCRYPT:            processEncryption,
		pdf.DECRYPT:            processEncryption,
		pdf.CHANGEUPW:          processEncryption,
		pdf.CHANGEOPW:          processEncryption,
		pdf.LISTPERMISSIONS:    processPermissions,
		pdf.ADDPERMISSIONS:     processPermissions,
		pdf.IMPORTIMAGES:       ImportImages,
		pdf.INSERTPAGES:        InsertPages,
		pdf.REMOVEPAGES:        RemovePages,
		pdf.ROTATE:             Rotate,
		pdf.NUP:                NUp,
	} {
		if cmd.Mode == k {
			return v(cmd)
		}
	}

	return nil, errors.Errorf("Process: Unknown command mode %d\n", cmd.Mode)
}

// ValidateCommand creates a new command to validate a file.
func ValidateCommand(pdfFileName string, config *pdf.Configuration) *Command {
	return &Command{
		Mode:   pdf.VALIDATE,
		InFile: &pdfFileName,
		Config: config}
}

// OptimizeCommand creates a new command to optimize a file.
func OptimizeCommand(pdfFileNameIn, pdfFileNameOut string, config *pdf.Configuration) *Command {
	return &Command{
		Mode:    pdf.OPTIMIZE,
		InFile:  &pdfFileNameIn,
		OutFile: &pdfFileNameOut,
		Config:  config}
}

// SplitCommand creates a new command to split a file into single page file.
func SplitCommand(pdfFileNameIn, dirNameOut string, span int, config *pdf.Configuration) *Command {
	return &Command{
		Mode:   pdf.SPLIT,
		InFile: &pdfFileNameIn,
		OutDir: &dirNameOut,
		Span:   span,
		Config: config}
}

// MergeCommand creates a new command to merge files.
func MergeCommand(pdfFileNamesIn []string, pdfFileNameOut string, config *pdf.Configuration) *Command {
	return &Command{
		Mode:    pdf.MERGE,
		InFiles: pdfFileNamesIn,
		OutFile: &pdfFileNameOut,
		Config:  config}
}

// ExtractImagesCommand creates a new command to extract embedded images.
// (experimental
func ExtractImagesCommand(pdfFileNameIn, dirNameOut string, pageSelection []string, config *pdf.Configuration) *Command {
	return &Command{
		Mode:          pdf.EXTRACTIMAGES,
		InFile:        &pdfFileNameIn,
		OutDir:        &dirNameOut,
		PageSelection: pageSelection,
		Config:        config}
}

// ExtractFontsCommand creates a new command to extract embedded fonts.
// (experimental)
func ExtractFontsCommand(pdfFileNameIn, dirNameOut string, pageSelection []string, config *pdf.Configuration) *Command {
	return &Command{
		Mode:          pdf.EXTRACTFONTS,
		InFile:        &pdfFileNameIn,
		OutDir:        &dirNameOut,
		PageSelection: pageSelection,
		Config:        config}
}

// ExtractPagesCommand creates a new command to extract specific pages of a file.
func ExtractPagesCommand(pdfFileNameIn, dirNameOut string, pageSelection []string, config *pdf.Configuration) *Command {
	return &Command{
		Mode:          pdf.EXTRACTPAGES,
		InFile:        &pdfFileNameIn,
		OutDir:        &dirNameOut,
		PageSelection: pageSelection,
		Config:        config}
}

// ExtractContentCommand creates a new command to extract page content streams.
func ExtractContentCommand(pdfFileNameIn, dirNameOut string, pageSelection []string, config *pdf.Configuration) *Command {
	return &Command{
		Mode:          pdf.EXTRACTCONTENT,
		InFile:        &pdfFileNameIn,
		OutDir:        &dirNameOut,
		PageSelection: pageSelection,
		Config:        config}
}

// ExtractMetadataCommand creates a new command to extract metadata streams.
func ExtractMetadataCommand(pdfFileNameIn, dirNameOut string, config *pdf.Configuration) *Command {
	return &Command{
		Mode:   pdf.EXTRACTMETADATA,
		InFile: &pdfFileNameIn,
		OutDir: &dirNameOut,
		Config: config}
}

// TrimCommand creates a new command to trim the pages of a file.
func TrimCommand(pdfFileNameIn, pdfFileNameOut string, pageSelection []string, config *pdf.Configuration) *Command {
	// A slice parameter may be called with nil => empty slice.
	return &Command{
		Mode:          pdf.TRIM,
		InFile:        &pdfFileNameIn,
		OutFile:       &pdfFileNameOut,
		PageSelection: pageSelection,
		Config:        config}
}

// ListAttachmentsCommand create a new command to list attachments.
func ListAttachmentsCommand(pdfFileNameIn string, config *pdf.Configuration) *Command {
	return &Command{
		Mode:   pdf.LISTATTACHMENTS,
		InFile: &pdfFileNameIn,
		Config: config}
}

// AddAttachmentsCommand creates a new command to add attachments.
func AddAttachmentsCommand(pdfFileNameIn string, fileNamesIn []string, config *pdf.Configuration) *Command {
	return &Command{
		Mode:    pdf.ADDATTACHMENTS,
		InFile:  &pdfFileNameIn,
		InFiles: fileNamesIn,
		Config:  config}
}

// RemoveAttachmentsCommand creates a new command to remove attachments.
func RemoveAttachmentsCommand(pdfFileNameIn string, fileNamesIn []string, config *pdf.Configuration) *Command {
	return &Command{
		Mode:    pdf.REMOVEATTACHMENTS,
		InFile:  &pdfFileNameIn,
		InFiles: fileNamesIn,
		Config:  config}
}

// ExtractAttachmentsCommand creates a new command to extract attachments.
func ExtractAttachmentsCommand(pdfFileNameIn, dirNameOut string, fileNamesIn []string, config *pdf.Configuration) *Command {
	return &Command{
		Mode:    pdf.EXTRACTATTACHMENTS,
		InFile:  &pdfFileNameIn,
		OutDir:  &dirNameOut,
		InFiles: fileNamesIn,
		Config:  config}
}

// EncryptCommand creates a new command to encrypt a file.
func EncryptCommand(pdfFileNameIn, pdfFileNameOut string, config *pdf.Configuration) *Command {
	return &Command{
		Mode:    pdf.ENCRYPT,
		InFile:  &pdfFileNameIn,
		OutFile: &pdfFileNameOut,
		Config:  config}
}

// DecryptCommand creates a new command to decrypt a file.
func DecryptCommand(pdfFileNameIn, pdfFileNameOut string, config *pdf.Configuration) *Command {
	return &Command{
		Mode:    pdf.DECRYPT,
		InFile:  &pdfFileNameIn,
		OutFile: &pdfFileNameOut,
		Config:  config}
}

// ChangeUserPWCommand creates a new command to change the user password.
func ChangeUserPWCommand(pdfFileNameIn, pdfFileNameOut string, config *pdf.Configuration, pwOld, pwNew *string) *Command {
	return &Command{
		Mode:    pdf.CHANGEUPW,
		InFile:  &pdfFileNameIn,
		OutFile: &pdfFileNameOut,
		Config:  config,
		PWOld:   pwOld,
		PWNew:   pwNew}
}

// ChangeOwnerPWCommand creates a new command to change the owner password.
func ChangeOwnerPWCommand(pdfFileNameIn, pdfFileNameOut string, config *pdf.Configuration, pwOld, pwNew *string) *Command {
	return &Command{
		Mode:    pdf.CHANGEOPW,
		InFile:  &pdfFileNameIn,
		OutFile: &pdfFileNameOut,
		Config:  config,
		PWOld:   pwOld,
		PWNew:   pwNew}
}

// ListPermissionsCommand create a new command to list permissions.
func ListPermissionsCommand(pdfFileNameIn string, config *pdf.Configuration) *Command {
	return &Command{
		Mode:   pdf.LISTPERMISSIONS,
		InFile: &pdfFileNameIn,
		Config: config}
}

// AddPermissionsCommand creates a new command to add permissions.
func AddPermissionsCommand(pdfFileNameIn string, config *pdf.Configuration) *Command {
	return &Command{
		Mode:   pdf.ADDPERMISSIONS,
		InFile: &pdfFileNameIn,
		Config: config}
}

func processAttachments(cmd *Command) (out []string, err error) {

	switch cmd.Mode {

	case pdf.LISTATTACHMENTS:
		out, err = ListAttachments(*cmd.InFile, cmd.Config)

	case pdf.ADDATTACHMENTS:
		err = AddAttachments(*cmd.InFile, cmd.InFiles, cmd.Config)

	case pdf.REMOVEATTACHMENTS:
		err = RemoveAttachments(*cmd.InFile, cmd.InFiles, cmd.Config)

	case pdf.EXTRACTATTACHMENTS:
		err = ExtractAttachments(*cmd.InFile, *cmd.OutDir, cmd.InFiles, cmd.Config)
	}

	return out, err
}

func processEncryption(cmd *Command) (out []string, err error) {

	switch cmd.Mode {

	case pdf.ENCRYPT:
		return Encrypt(cmd)

	case pdf.DECRYPT:
		return Decrypt(cmd)

	case pdf.CHANGEUPW:
		return ChangeUserPassword(cmd)

	case pdf.CHANGEOPW:
		return ChangeOwnerPassword(cmd)
	}

	return nil, nil
}

func processPermissions(cmd *Command) (out []string, err error) {

	switch cmd.Mode {

	case pdf.LISTPERMISSIONS:
		out, err = ListPermissions(*cmd.InFile, cmd.Config)

	case pdf.ADDPERMISSIONS:
		err = AddPermissions(*cmd.InFile, cmd.Config)
	}

	return out, err
}

// AddWatermarksCommand creates a new command to add Watermarks to a file.
func AddWatermarksCommand(pdfFileNameIn, pdfFileNameOut string, pageSelection []string, wm *pdf.Watermark, config *pdf.Configuration) *Command {
	return &Command{
		Mode:          pdf.ADDWATERMARKS,
		InFile:        &pdfFileNameIn,
		OutFile:       &pdfFileNameOut,
		PageSelection: pageSelection,
		Watermark:     wm,
		Config:        config}
}

// ImportImagesCommand creates a new command to import images.
func ImportImagesCommand(imageFileNamesIn []string, pdfFileNameOut string, imp *pdf.Import, config *pdf.Configuration) *Command {
	return &Command{
		Mode:    pdf.IMPORTIMAGES,
		InFiles: imageFileNamesIn,
		OutFile: &pdfFileNameOut,
		Import:  imp,
		Config:  config}
}

// InsertPagesCommand creates a new command to insert a blank page before selected pages.
func InsertPagesCommand(pdfFileNameIn, pdfFileNameOut string, pageSelection []string, config *pdf.Configuration) *Command {
	return &Command{
		Mode:          pdf.INSERTPAGES,
		InFile:        &pdfFileNameIn,
		OutFile:       &pdfFileNameOut,
		PageSelection: pageSelection,
		Config:        config}
}

// RemovePagesCommand creates a new command to remove selected pages.
func RemovePagesCommand(pdfFileNameIn, pdfFileNameOut string, pageSelection []string, config *pdf.Configuration) *Command {
	return &Command{
		Mode:          pdf.REMOVEPAGES,
		InFile:        &pdfFileNameIn,
		OutFile:       &pdfFileNameOut,
		PageSelection: pageSelection,
		Config:        config}
}

// RotateCommand creates a new command to rotate pages.
func RotateCommand(pdfFileNameIn string, rotation int, pageSelection []string, config *pdf.Configuration) *Command {
	return &Command{
		Mode:          pdf.ROTATE,
		InFile:        &pdfFileNameIn,
		PageSelection: pageSelection,
		Rotation:      rotation,
		Config:        config}
}

// NUpCommand creates a new command to render PDFs or image files in n-up fashion.
func NUpCommand(fileNamesIn []string, pdfFileNameOut string, pageSelection []string, nUp *pdf.NUp, config *pdf.Configuration) *Command {
	return &Command{
		Mode:          pdf.NUP,
		InFiles:       fileNamesIn,
		OutFile:       &pdfFileNameOut,
		PageSelection: pageSelection,
		NUp:           nUp,
		Config:        config}
}
