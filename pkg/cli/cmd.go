/*
Copyright 2020 The pdfcpu Authors.

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

package cli

import (
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

// Command represents an execution context.
type Command struct {
	Mode           pdfcpu.CommandMode
	InFile         *string
	InFiles        []string
	InDir          *string
	OutFile        *string
	OutDir         *string
	PageSelection  []string
	Conf           *pdfcpu.Configuration
	PWOld          *string
	PWNew          *string
	Watermark      *pdfcpu.Watermark
	Span           int
	Import         *pdfcpu.Import
	Rotation       int
	NUp            *pdfcpu.NUp
	Input          io.ReadSeeker
	Inputs         []io.ReadSeeker
	Output         io.Writer
	StringMap      map[string]string
	Box            *pdfcpu.Box
	PageBoundaries *pdfcpu.PageBoundaries
}

var cmdMap = map[pdfcpu.CommandMode]func(cmd *Command) ([]string, error){
	pdfcpu.VALIDATE:                Validate,
	pdfcpu.OPTIMIZE:                Optimize,
	pdfcpu.SPLIT:                   Split,
	pdfcpu.MERGECREATE:             MergeCreate,
	pdfcpu.MERGEAPPEND:             MergeAppend,
	pdfcpu.EXTRACTIMAGES:           ExtractImages,
	pdfcpu.EXTRACTFONTS:            ExtractFonts,
	pdfcpu.EXTRACTPAGES:            ExtractPages,
	pdfcpu.EXTRACTCONTENT:          ExtractContent,
	pdfcpu.EXTRACTMETADATA:         ExtractMetadata,
	pdfcpu.TRIM:                    Trim,
	pdfcpu.ADDWATERMARKS:           AddWatermarks,
	pdfcpu.REMOVEWATERMARKS:        RemoveWatermarks,
	pdfcpu.LISTATTACHMENTS:         processAttachments,
	pdfcpu.ADDATTACHMENTS:          processAttachments,
	pdfcpu.ADDATTACHMENTSPORTFOLIO: processAttachments,
	pdfcpu.REMOVEATTACHMENTS:       processAttachments,
	pdfcpu.EXTRACTATTACHMENTS:      processAttachments,
	pdfcpu.ENCRYPT:                 processEncryption,
	pdfcpu.DECRYPT:                 processEncryption,
	pdfcpu.CHANGEUPW:               processEncryption,
	pdfcpu.CHANGEOPW:               processEncryption,
	pdfcpu.LISTPERMISSIONS:         processPermissions,
	pdfcpu.SETPERMISSIONS:          processPermissions,
	pdfcpu.IMPORTIMAGES:            ImportImages,
	pdfcpu.INSERTPAGESBEFORE:       processPages,
	pdfcpu.INSERTPAGESAFTER:        processPages,
	pdfcpu.REMOVEPAGES:             processPages,
	pdfcpu.ROTATE:                  Rotate,
	pdfcpu.NUP:                     NUp,
	pdfcpu.BOOKLET:                 Booklet,
	pdfcpu.INFO:                    Info,
	pdfcpu.CHEATSHEETSFONTS:        CreateCheatSheetsFonts,
	pdfcpu.INSTALLFONTS:            InstallFonts,
	pdfcpu.LISTFONTS:               ListFonts,
	pdfcpu.LISTKEYWORDS:            processKeywords,
	pdfcpu.ADDKEYWORDS:             processKeywords,
	pdfcpu.REMOVEKEYWORDS:          processKeywords,
	pdfcpu.LISTPROPERTIES:          processProperties,
	pdfcpu.ADDPROPERTIES:           processProperties,
	pdfcpu.REMOVEPROPERTIES:        processProperties,
	pdfcpu.COLLECT:                 Collect,
	pdfcpu.LISTBOXES:               processPageBoundaries,
	pdfcpu.ADDBOXES:                processPageBoundaries,
	pdfcpu.REMOVEBOXES:             processPageBoundaries,
	pdfcpu.CROP:                    processPageBoundaries,
}

// ValidateCommand creates a new command to validate a file.
func ValidateCommand(inFile string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.VALIDATE
	return &Command{
		Mode:   pdfcpu.VALIDATE,
		InFile: &inFile,
		Conf:   conf}
}

// OptimizeCommand creates a new command to optimize a file.
func OptimizeCommand(inFile, outFile string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.OPTIMIZE
	return &Command{
		Mode:    pdfcpu.OPTIMIZE,
		InFile:  &inFile,
		OutFile: &outFile,
		Conf:    conf}
}

// SplitCommand creates a new command to split a file into single page files.
func SplitCommand(inFile, dirNameOut string, span int, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.SPLIT
	return &Command{
		Mode:   pdfcpu.SPLIT,
		InFile: &inFile,
		OutDir: &dirNameOut,
		Span:   span,
		Conf:   conf}
}

// MergeCreateCommand creates a new command to merge files.
// Outfile will be created. An existing outFile will be overwritten.
func MergeCreateCommand(inFiles []string, outFile string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.MERGECREATE
	return &Command{
		Mode:    pdfcpu.MERGECREATE,
		InFiles: inFiles,
		OutFile: &outFile,
		Conf:    conf}
}

// MergeAppendCommand creates a new command to merge files.
// Any existing outFile PDF content will be preserved and serves as the beginning of the merge result.
func MergeAppendCommand(inFiles []string, outFile string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.MERGEAPPEND
	return &Command{
		Mode:    pdfcpu.MERGEAPPEND,
		InFiles: inFiles,
		OutFile: &outFile,
		Conf:    conf}
}

// ExtractImagesCommand creates a new command to extract embedded images.
// (experimental)
func ExtractImagesCommand(inFile string, outDir string, pageSelection []string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.EXTRACTIMAGES
	return &Command{
		Mode:          pdfcpu.EXTRACTIMAGES,
		InFile:        &inFile,
		OutDir:        &outDir,
		PageSelection: pageSelection,
		Conf:          conf}
}

// ExtractFontsCommand creates a new command to extract embedded fonts.
// (experimental)
func ExtractFontsCommand(inFile string, outDir string, pageSelection []string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.EXTRACTFONTS
	return &Command{
		Mode:          pdfcpu.EXTRACTFONTS,
		InFile:        &inFile,
		OutDir:        &outDir,
		PageSelection: pageSelection,
		Conf:          conf}
}

// ExtractPagesCommand creates a new command to extract specific pages of a file.
func ExtractPagesCommand(inFile string, outDir string, pageSelection []string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.EXTRACTPAGES
	return &Command{
		Mode:          pdfcpu.EXTRACTPAGES,
		InFile:        &inFile,
		OutDir:        &outDir,
		PageSelection: pageSelection,
		Conf:          conf}
}

// ExtractContentCommand creates a new command to extract page content streams.
func ExtractContentCommand(inFile string, outDir string, pageSelection []string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.EXTRACTCONTENT
	return &Command{
		Mode:          pdfcpu.EXTRACTCONTENT,
		InFile:        &inFile,
		OutDir:        &outDir,
		PageSelection: pageSelection,
		Conf:          conf}
}

// ExtractMetadataCommand creates a new command to extract metadata streams.
func ExtractMetadataCommand(inFile string, outDir string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.EXTRACTMETADATA
	return &Command{
		Mode:   pdfcpu.EXTRACTMETADATA,
		InFile: &inFile,
		OutDir: &outDir,
		Conf:   conf}
}

// TrimCommand creates a new command to trim the pages of a file.
func TrimCommand(inFile, outFile string, pageSelection []string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.TRIM
	return &Command{
		Mode:          pdfcpu.TRIM,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Conf:          conf}
}

// ListAttachmentsCommand create a new command to list attachments.
func ListAttachmentsCommand(inFile string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.LISTATTACHMENTS
	return &Command{
		Mode:   pdfcpu.LISTATTACHMENTS,
		InFile: &inFile,
		Conf:   conf}
}

// AddAttachmentsCommand creates a new command to add attachments.
func AddAttachmentsCommand(inFile, outFile string, fileNames []string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.ADDATTACHMENTS
	return &Command{
		Mode:    pdfcpu.ADDATTACHMENTS,
		InFile:  &inFile,
		OutFile: &outFile,
		InFiles: fileNames,
		Conf:    conf}
}

// AddAttachmentsPortfolioCommand creates a new command to add attachments to a portfolio.
func AddAttachmentsPortfolioCommand(inFile, outFile string, fileNames []string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.ADDATTACHMENTSPORTFOLIO
	return &Command{
		Mode:    pdfcpu.ADDATTACHMENTSPORTFOLIO,
		InFile:  &inFile,
		OutFile: &outFile,
		InFiles: fileNames,
		Conf:    conf}
}

// RemoveAttachmentsCommand creates a new command to remove attachments.
func RemoveAttachmentsCommand(inFile, outFile string, fileNames []string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.REMOVEATTACHMENTS
	return &Command{
		Mode:    pdfcpu.REMOVEATTACHMENTS,
		InFile:  &inFile,
		OutFile: &outFile,
		InFiles: fileNames,
		Conf:    conf}
}

// ExtractAttachmentsCommand creates a new command to extract attachments.
func ExtractAttachmentsCommand(inFile string, outDir string, fileNames []string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.EXTRACTATTACHMENTS
	return &Command{
		Mode:    pdfcpu.EXTRACTATTACHMENTS,
		InFile:  &inFile,
		OutDir:  &outDir,
		InFiles: fileNames,
		Conf:    conf}
}

// EncryptCommand creates a new command to encrypt a file.
func EncryptCommand(inFile, outFile string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.ENCRYPT
	return &Command{
		Mode:    pdfcpu.ENCRYPT,
		InFile:  &inFile,
		OutFile: &outFile,
		Conf:    conf}
}

// DecryptCommand creates a new command to decrypt a file.
func DecryptCommand(inFile, outFile string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.DECRYPT
	return &Command{
		Mode:    pdfcpu.DECRYPT,
		InFile:  &inFile,
		OutFile: &outFile,
		Conf:    conf}
}

// ChangeUserPWCommand creates a new command to change the user password.
func ChangeUserPWCommand(inFile, outFile string, pwOld, pwNew *string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.CHANGEUPW
	return &Command{
		Mode:    pdfcpu.CHANGEUPW,
		InFile:  &inFile,
		OutFile: &outFile,
		PWOld:   pwOld,
		PWNew:   pwNew,
		Conf:    conf}
}

// ChangeOwnerPWCommand creates a new command to change the owner password.
func ChangeOwnerPWCommand(inFile, outFile string, pwOld, pwNew *string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.CHANGEOPW
	return &Command{
		Mode:    pdfcpu.CHANGEOPW,
		InFile:  &inFile,
		OutFile: &outFile,
		PWOld:   pwOld,
		PWNew:   pwNew,
		Conf:    conf}
}

// ListPermissionsCommand create a new command to list permissions.
func ListPermissionsCommand(inFile string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.LISTPERMISSIONS
	return &Command{
		Mode:   pdfcpu.LISTPERMISSIONS,
		InFile: &inFile,
		Conf:   conf}
}

// SetPermissionsCommand creates a new command to add permissions.
func SetPermissionsCommand(inFile, outFile string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.SETPERMISSIONS
	return &Command{
		Mode:    pdfcpu.SETPERMISSIONS,
		InFile:  &inFile,
		OutFile: &outFile,
		Conf:    conf}
}

// AddWatermarksCommand creates a new command to add Watermarks to a file.
func AddWatermarksCommand(inFile, outFile string, pageSelection []string, wm *pdfcpu.Watermark, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.ADDWATERMARKS
	return &Command{
		Mode:          pdfcpu.ADDWATERMARKS,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Watermark:     wm,
		Conf:          conf}
}

// RemoveWatermarksCommand creates a new command to remove Watermarks from a file.
func RemoveWatermarksCommand(inFile, outFile string, pageSelection []string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.REMOVEWATERMARKS
	return &Command{
		Mode:          pdfcpu.REMOVEWATERMARKS,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Conf:          conf}
}

// ImportImagesCommand creates a new command to import images.
func ImportImagesCommand(imageFiles []string, outFile string, imp *pdfcpu.Import, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.IMPORTIMAGES
	return &Command{
		Mode:    pdfcpu.IMPORTIMAGES,
		InFiles: imageFiles,
		OutFile: &outFile,
		Import:  imp,
		Conf:    conf}
}

// InsertPagesCommand creates a new command to insert a blank page before or after selected pages.
func InsertPagesCommand(inFile, outFile string, pageSelection []string, conf *pdfcpu.Configuration, mode string) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	cmdMode := pdfcpu.INSERTPAGESBEFORE
	if mode == "after" {
		cmdMode = pdfcpu.INSERTPAGESAFTER
	}
	conf.Cmd = cmdMode
	return &Command{
		Mode:          cmdMode,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Conf:          conf}
}

// RemovePagesCommand creates a new command to remove selected pages.
func RemovePagesCommand(inFile, outFile string, pageSelection []string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.REMOVEPAGES
	return &Command{
		Mode:          pdfcpu.REMOVEPAGES,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Conf:          conf}
}

// RotateCommand creates a new command to rotate pages.
func RotateCommand(inFile, outFile string, rotation int, pageSelection []string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.ROTATE
	return &Command{
		Mode:          pdfcpu.ROTATE,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Rotation:      rotation,
		Conf:          conf}
}

// NUpCommand creates a new command to render PDFs or image files in n-up fashion.
func NUpCommand(inFiles []string, outFile string, pageSelection []string, nUp *pdfcpu.NUp, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.NUP
	return &Command{
		Mode:          pdfcpu.NUP,
		InFiles:       inFiles,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		NUp:           nUp,
		Conf:          conf}
}

// BookletCommand creates a new command to render PDFs or image files in booklet fashion.
func BookletCommand(inFiles []string, outFile string, pageSelection []string, nup *pdfcpu.NUp, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.BOOKLET
	return &Command{
		Mode:          pdfcpu.BOOKLET,
		InFiles:       inFiles,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		NUp:           nup,
		Conf:          conf}
}

// InfoCommand creates a new command to output information about inFile.
func InfoCommand(inFile string, pageSelection []string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.INFO
	return &Command{
		Mode:          pdfcpu.INFO,
		InFile:        &inFile,
		PageSelection: pageSelection,
		Conf:          conf}
}

// ListFontsCommand returns a list of supported fonts.
func ListFontsCommand(conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.LISTFONTS
	return &Command{
		Mode: pdfcpu.LISTFONTS,
		Conf: conf}
}

// InstallFontsCommand installs true type fonts for embedding.
func InstallFontsCommand(fontFiles []string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.INSTALLFONTS
	return &Command{
		Mode:    pdfcpu.INSTALLFONTS,
		InFiles: fontFiles,
		Conf:    conf}
}

// CreateCheatSheetsFontsCommand creates single page PDF cheat sheets in current dir.
func CreateCheatSheetsFontsCommand(fontFiles []string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.CHEATSHEETSFONTS
	return &Command{
		Mode:    pdfcpu.CHEATSHEETSFONTS,
		InFiles: fontFiles,
		Conf:    conf}
}

// ListKeywordsCommand create a new command to list keywords.
func ListKeywordsCommand(inFile string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.LISTKEYWORDS
	return &Command{
		Mode:   pdfcpu.LISTKEYWORDS,
		InFile: &inFile,
		Conf:   conf}
}

// AddKeywordsCommand creates a new command to add keywords.
func AddKeywordsCommand(inFile, outFile string, keywords []string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.ADDKEYWORDS
	return &Command{
		Mode:    pdfcpu.ADDKEYWORDS,
		InFile:  &inFile,
		OutFile: &outFile,
		InFiles: keywords,
		Conf:    conf}
}

// RemoveKeywordsCommand creates a new command to remove keywords.
func RemoveKeywordsCommand(inFile, outFile string, keywords []string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.REMOVEKEYWORDS
	return &Command{
		Mode:    pdfcpu.REMOVEKEYWORDS,
		InFile:  &inFile,
		OutFile: &outFile,
		InFiles: keywords,
		Conf:    conf}
}

// ListPropertiesCommand creates a new command to list document properties.
func ListPropertiesCommand(inFile string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.LISTPROPERTIES
	return &Command{
		Mode:   pdfcpu.LISTPROPERTIES,
		InFile: &inFile,
		Conf:   conf}
}

// AddPropertiesCommand creates a new command to add document properties.
func AddPropertiesCommand(inFile, outFile string, properties map[string]string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.ADDPROPERTIES
	return &Command{
		Mode:      pdfcpu.ADDPROPERTIES,
		InFile:    &inFile,
		OutFile:   &outFile,
		StringMap: properties,
		Conf:      conf}
}

// RemovePropertiesCommand creates a new command to remove document properties.
func RemovePropertiesCommand(inFile, outFile string, propKeys []string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.REMOVEPROPERTIES
	return &Command{
		Mode:    pdfcpu.REMOVEPROPERTIES,
		InFile:  &inFile,
		OutFile: &outFile,
		InFiles: propKeys,
		Conf:    conf}
}

// CollectCommand creates a new command to create a custom PDF page sequence.
func CollectCommand(inFile, outFile string, pageSelection []string, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.COLLECT
	return &Command{
		Mode:          pdfcpu.COLLECT,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Conf:          conf}
}

// ListBoxesCommand creates a new command to list page boundaries for selected pages.
func ListBoxesCommand(inFile string, pageSelection []string, pb *pdfcpu.PageBoundaries, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.LISTBOXES
	return &Command{
		Mode:           pdfcpu.LISTBOXES,
		InFile:         &inFile,
		PageSelection:  pageSelection,
		PageBoundaries: pb,
		Conf:           conf}
}

// AddBoxesCommand creates a new command to add page boundaries for selected pages.
func AddBoxesCommand(inFile, outFile string, pageSelection []string, pb *pdfcpu.PageBoundaries, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.ADDBOXES
	return &Command{
		Mode:           pdfcpu.ADDBOXES,
		InFile:         &inFile,
		OutFile:        &outFile,
		PageSelection:  pageSelection,
		PageBoundaries: pb,
		Conf:           conf}
}

// RemoveBoxesCommand creates a new command to remove page boundaries for selected pages.
func RemoveBoxesCommand(inFile, outFile string, pageSelection []string, pb *pdfcpu.PageBoundaries, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.REMOVEBOXES
	return &Command{
		Mode:           pdfcpu.REMOVEBOXES,
		InFile:         &inFile,
		OutFile:        &outFile,
		PageSelection:  pageSelection,
		PageBoundaries: pb,
		Conf:           conf}
}

// CropCommand creates a new command to apply a cropBox to selected pages.
func CropCommand(inFile, outFile string, pageSelection []string, box *pdfcpu.Box, conf *pdfcpu.Configuration) *Command {
	if conf == nil {
		conf = pdfcpu.NewDefaultConfiguration()
	}
	conf.Cmd = pdfcpu.CROP
	return &Command{
		Mode:          pdfcpu.CROP,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Box:           box,
		Conf:          conf}
}
