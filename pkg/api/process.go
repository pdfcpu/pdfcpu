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
	"github.com/hhrutter/pdfcpu/pkg/pdfcpu"
	"github.com/pkg/errors"
)

// Command represents an execution context.
type Command struct {
	Mode          pdfcpu.CommandMode    // VALIDATE  OPTIMIZE  SPLIT  MERGE  EXTRACT  TRIM  LISTATT ADDATT REMATT EXTATT  ENCRYPT  DECRYPT  CHANGEUPW  CHANGEOPW LISTP ADDP  WATERMARK
	InFile        *string               //    *         *        *      -       *      *      *       *       *      *       *        *         *          *       *     *       *
	InFiles       []string              //    -         -        -      *       -      -      -       *       *      *       -        -         -          -       -     -       -
	InDir         *string               //    -         -        -      -       -      -      -       -       -      -       -        -         -          -       -     -       -
	OutFile       *string               //    -         *        -      *       -      *      -       -       -      -       *        *         *          *       -     -       *
	OutDir        *string               //    -         -        *      -       *      -      -       -       -      *       -        -         -          -       -     -       -
	PageSelection []string              //    -         -        -      -       *      *      -       -       -      -       -        -         -          -       -     -       *
	Config        *pdfcpu.Configuration //    *         *        *      *       *      *      *       *       *      *       *        *         *          *       *     *       *
	PWOld         *string               //    -         -        -      -       -      -      -       -       -      -       -        -         *          *       -     -       -
	PWNew         *string               //    -         -        -      -       -      -      -       -       -      -       -        -         *          *       -     -       -
	Watermark     *pdfcpu.Watermark     //    -         -        -      -       -      -      -       -       -      -       -        -         -          -       -     -       -
}

// Process executes a pdfcpu command.
func Process(cmd *Command) (out []string, err error) {

	defer func() {
		if r := recover(); r != nil {
			//fmt.Println(r)
			err = errors.Errorf("unexpected panic attack: %v\n", r)
		}
	}()

	cmd.Config.Mode = cmd.Mode

	for k, v := range map[pdfcpu.CommandMode]func(cmd *Command) ([]string, error){
		pdfcpu.VALIDATE:           Validate,
		pdfcpu.OPTIMIZE:           Optimize,
		pdfcpu.SPLIT:              Split,
		pdfcpu.MERGE:              Merge,
		pdfcpu.EXTRACTIMAGES:      ExtractImages,
		pdfcpu.EXTRACTFONTS:       ExtractFonts,
		pdfcpu.EXTRACTPAGES:       ExtractPages,
		pdfcpu.EXTRACTCONTENT:     ExtractContent,
		pdfcpu.TRIM:               Trim,
		pdfcpu.ADDWATERMARKS:      AddWatermarks,
		pdfcpu.LISTATTACHMENTS:    processAttachments,
		pdfcpu.ADDATTACHMENTS:     processAttachments,
		pdfcpu.REMOVEATTACHMENTS:  processAttachments,
		pdfcpu.EXTRACTATTACHMENTS: processAttachments,
		pdfcpu.ENCRYPT:            processEncryption,
		pdfcpu.DECRYPT:            processEncryption,
		pdfcpu.CHANGEUPW:          processEncryption,
		pdfcpu.CHANGEOPW:          processEncryption,
		pdfcpu.LISTPERMISSIONS:    processPermissions,
		pdfcpu.ADDPERMISSIONS:     processPermissions,
	} {
		if cmd.Mode == k {
			return v(cmd)
		}
	}

	return nil, errors.Errorf("Process: Unknown command mode %d\n", cmd.Mode)
}

// ValidateCommand creates a new command to validate a file.
func ValidateCommand(pdfFileName string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:   pdfcpu.VALIDATE,
		InFile: &pdfFileName,
		Config: config}
}

// OptimizeCommand creates a new command to optimize a file.
func OptimizeCommand(pdfFileNameIn, pdfFileNameOut string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:    pdfcpu.OPTIMIZE,
		InFile:  &pdfFileNameIn,
		OutFile: &pdfFileNameOut,
		Config:  config}
}

// SplitCommand creates a new command to split a file into single page file.
func SplitCommand(pdfFileNameIn, dirNameOut string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:   pdfcpu.SPLIT,
		InFile: &pdfFileNameIn,
		OutDir: &dirNameOut,
		Config: config}
}

// MergeCommand creates a new command to merge files.
func MergeCommand(pdfFileNamesIn []string, pdfFileNameOut string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode: pdfcpu.MERGE,
		//InFile:  &pdfFileNameIn,
		InFiles: pdfFileNamesIn,
		OutFile: &pdfFileNameOut,
		Config:  config}
}

// ExtractImagesCommand creates a new command to extract embedded images.
// (experimental
func ExtractImagesCommand(pdfFileNameIn, dirNameOut string, pageSelection []string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:          pdfcpu.EXTRACTIMAGES,
		InFile:        &pdfFileNameIn,
		OutDir:        &dirNameOut,
		PageSelection: pageSelection,
		Config:        config}
}

// ExtractFontsCommand creates a new command to extract embedded fonts.
// (experimental)
func ExtractFontsCommand(pdfFileNameIn, dirNameOut string, pageSelection []string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:          pdfcpu.EXTRACTFONTS,
		InFile:        &pdfFileNameIn,
		OutDir:        &dirNameOut,
		PageSelection: pageSelection,
		Config:        config}
}

// ExtractPagesCommand creates a new command to extract specific pages of a file.
func ExtractPagesCommand(pdfFileNameIn, dirNameOut string, pageSelection []string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:          pdfcpu.EXTRACTPAGES,
		InFile:        &pdfFileNameIn,
		OutDir:        &dirNameOut,
		PageSelection: pageSelection,
		Config:        config}
}

// ExtractContentCommand creates a new command to extract page content streams.
func ExtractContentCommand(pdfFileNameIn, dirNameOut string, pageSelection []string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:          pdfcpu.EXTRACTCONTENT,
		InFile:        &pdfFileNameIn,
		OutDir:        &dirNameOut,
		PageSelection: pageSelection,
		Config:        config}
}

// TrimCommand creates a new command to trim the pages of a file.
func TrimCommand(pdfFileNameIn, pdfFileNameOut string, pageSelection []string, config *pdfcpu.Configuration) *Command {
	// A slice parameter may be called with nil => empty slice.
	return &Command{
		Mode:          pdfcpu.TRIM,
		InFile:        &pdfFileNameIn,
		OutFile:       &pdfFileNameOut,
		PageSelection: pageSelection,
		Config:        config}
}

// ListAttachmentsCommand create a new command to list attachments.
func ListAttachmentsCommand(pdfFileNameIn string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:   pdfcpu.LISTATTACHMENTS,
		InFile: &pdfFileNameIn,
		Config: config}
}

// AddAttachmentsCommand creates a new command to add attachments.
func AddAttachmentsCommand(pdfFileNameIn string, fileNamesIn []string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:    pdfcpu.ADDATTACHMENTS,
		InFile:  &pdfFileNameIn,
		InFiles: fileNamesIn,
		Config:  config}
}

// RemoveAttachmentsCommand creates a new command to remove attachments.
func RemoveAttachmentsCommand(pdfFileNameIn string, fileNamesIn []string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:    pdfcpu.REMOVEATTACHMENTS,
		InFile:  &pdfFileNameIn,
		InFiles: fileNamesIn,
		Config:  config}
}

// ExtractAttachmentsCommand creates a new command to extract attachments.
func ExtractAttachmentsCommand(pdfFileNameIn, dirNameOut string, fileNamesIn []string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:    pdfcpu.EXTRACTATTACHMENTS,
		InFile:  &pdfFileNameIn,
		OutDir:  &dirNameOut,
		InFiles: fileNamesIn,
		Config:  config}
}

// EncryptCommand creates a new command to encrypt a file.
func EncryptCommand(pdfFileNameIn, pdfFileNameOut string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:    pdfcpu.ENCRYPT,
		InFile:  &pdfFileNameIn,
		OutFile: &pdfFileNameOut,
		Config:  config}
}

// DecryptCommand creates a new command to decrypt a file.
func DecryptCommand(pdfFileNameIn, pdfFileNameOut string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:    pdfcpu.DECRYPT,
		InFile:  &pdfFileNameIn,
		OutFile: &pdfFileNameOut,
		Config:  config}
}

// ChangeUserPWCommand creates a new command to change the user password.
func ChangeUserPWCommand(pdfFileNameIn, pdfFileNameOut string, config *pdfcpu.Configuration, pwOld, pwNew *string) *Command {
	return &Command{
		Mode:    pdfcpu.CHANGEUPW,
		InFile:  &pdfFileNameIn,
		OutFile: &pdfFileNameOut,
		Config:  config,
		PWOld:   pwOld,
		PWNew:   pwNew}
}

// ChangeOwnerPWCommand creates a new command to change the owner password.
func ChangeOwnerPWCommand(pdfFileNameIn, pdfFileNameOut string, config *pdfcpu.Configuration, pwOld, pwNew *string) *Command {
	return &Command{
		Mode:    pdfcpu.CHANGEOPW,
		InFile:  &pdfFileNameIn,
		OutFile: &pdfFileNameOut,
		Config:  config,
		PWOld:   pwOld,
		PWNew:   pwNew}
}

// ListPermissionsCommand create a new command to list permissions.
func ListPermissionsCommand(pdfFileNameIn string, config *pdfcpu.Configuration) *Command {
	return &Command{
		Mode:   pdfcpu.LISTPERMISSIONS,
		InFile: &pdfFileNameIn,
		Config: config}
}

// AddPermissionsCommand creates a new command to add permissions.
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

func processEncryption(cmd *Command) (out []string, err error) {

	switch cmd.Mode {

	case pdfcpu.ENCRYPT:
		return Encrypt(cmd)

	case pdfcpu.DECRYPT:
		return Decrypt(cmd)

	case pdfcpu.CHANGEUPW:
		return ChangeUserPassword(cmd)

	case pdfcpu.CHANGEOPW:
		return ChangeOwnerPassword(cmd)
	}

	return nil, nil
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

// AddWatermarksCommand creates a new command to add Watermarks to a file.
func AddWatermarksCommand(pdfFileNameIn, pdfFileNameOut string, pageSelection []string, wm *pdfcpu.Watermark, config *pdfcpu.Configuration) *Command {

	return &Command{
		Mode:          pdfcpu.ADDWATERMARKS,
		InFile:        &pdfFileNameIn,
		OutFile:       &pdfFileNameOut,
		PageSelection: pageSelection,
		Watermark:     wm,
		Config:        config}
}
