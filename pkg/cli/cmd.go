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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// Command represents an execution context.
type Command struct {
	Mode              model.CommandMode
	InFile            *string
	InFileJSON        *string
	InFiles           []string
	InDir             *string
	OutFile           *string
	OutFileJSON       *string
	OutDir            *string
	PageSelection     []string
	PWOld             *string
	PWNew             *string
	StringVal         string
	IntVal            int
	BoolVal1          bool
	BoolVal2          bool
	IntVals           []int
	StringVals        []string
	StringMap         map[string]string
	Input             io.ReadSeeker
	Inputs            []io.ReadSeeker
	Output            io.Writer
	Box               *model.Box
	Import            *pdfcpu.Import
	NUp               *model.NUp
	Cut               *model.Cut
	PageBoundaries    *model.PageBoundaries
	Resize            *model.Resize
	Zoom              *model.Zoom
	Watermark         *model.Watermark
	ViewerPreferences *model.ViewerPreferences
	Conf              *model.Configuration
}

var cmdMap = map[model.CommandMode]func(cmd *Command) ([]string, error){
	model.VALIDATE:                Validate,
	model.OPTIMIZE:                Optimize,
	model.SPLIT:                   Split,
	model.SPLITBYPAGENR:           SplitByPageNr,
	model.MERGECREATE:             MergeCreate,
	model.MERGECREATEZIP:          MergeCreateZip,
	model.MERGEAPPEND:             MergeAppend,
	model.EXTRACTIMAGES:           ExtractImages,
	model.EXTRACTFONTS:            ExtractFonts,
	model.EXTRACTPAGES:            ExtractPages,
	model.EXTRACTCONTENT:          ExtractContent,
	model.EXTRACTMETADATA:         ExtractMetadata,
	model.TRIM:                    Trim,
	model.ADDWATERMARKS:           AddWatermarks,
	model.REMOVEWATERMARKS:        RemoveWatermarks,
	model.LISTATTACHMENTS:         processAttachments,
	model.ADDATTACHMENTS:          processAttachments,
	model.ADDATTACHMENTSPORTFOLIO: processAttachments,
	model.REMOVEATTACHMENTS:       processAttachments,
	model.EXTRACTATTACHMENTS:      processAttachments,
	model.ENCRYPT:                 processEncryption,
	model.DECRYPT:                 processEncryption,
	model.CHANGEUPW:               processEncryption,
	model.CHANGEOPW:               processEncryption,
	model.LISTPERMISSIONS:         processPermissions,
	model.SETPERMISSIONS:          processPermissions,
	model.IMPORTIMAGES:            ImportImages,
	model.INSERTPAGESBEFORE:       processPages,
	model.INSERTPAGESAFTER:        processPages,
	model.REMOVEPAGES:             processPages,
	model.ROTATE:                  Rotate,
	model.NUP:                     NUp,
	model.BOOKLET:                 Booklet,
	model.LISTINFO:                ListInfo,
	model.CHEATSHEETSFONTS:        CreateCheatSheetsFonts,
	model.INSTALLFONTS:            InstallFonts,
	model.LISTFONTS:               ListFonts,
	model.LISTKEYWORDS:            processKeywords,
	model.ADDKEYWORDS:             processKeywords,
	model.REMOVEKEYWORDS:          processKeywords,
	model.LISTPROPERTIES:          processProperties,
	model.ADDPROPERTIES:           processProperties,
	model.REMOVEPROPERTIES:        processProperties,
	model.COLLECT:                 Collect,
	model.LISTBOXES:               processPageBoundaries,
	model.ADDBOXES:                processPageBoundaries,
	model.REMOVEBOXES:             processPageBoundaries,
	model.CROP:                    processPageBoundaries,
	model.LISTANNOTATIONS:         processPageAnnotations,
	model.REMOVEANNOTATIONS:       processPageAnnotations,
	model.LISTIMAGES:              processImages,
	model.DUMP:                    Dump,
	model.CREATE:                  Create,
	model.LISTFORMFIELDS:          processForm,
	model.REMOVEFORMFIELDS:        processForm,
	model.LOCKFORMFIELDS:          processForm,
	model.UNLOCKFORMFIELDS:        processForm,
	model.RESETFORMFIELDS:         processForm,
	model.EXPORTFORMFIELDS:        processForm,
	model.FILLFORMFIELDS:          processForm,
	model.MULTIFILLFORMFIELDS:     processForm,
	model.RESIZE:                  Resize,
	model.POSTER:                  Poster,
	model.NDOWN:                   NDown,
	model.CUT:                     Cut,
	model.LISTBOOKMARKS:           processBookmarks,
	model.EXPORTBOOKMARKS:         processBookmarks,
	model.IMPORTBOOKMARKS:         processBookmarks,
	model.REMOVEBOOKMARKS:         processBookmarks,
	model.LISTPAGEMODE:            processPageMode,
	model.SETPAGEMODE:             processPageMode,
	model.RESETPAGEMODE:           processPageMode,
	model.LISTPAGELAYOUT:          processPageLayout,
	model.SETPAGELAYOUT:           processPageLayout,
	model.RESETPAGELAYOUT:         processPageLayout,
	model.LISTVIEWERPREFERENCES:   processViewerPreferences,
	model.SETVIEWERPREFERENCES:    processViewerPreferences,
	model.RESETVIEWERPREFERENCES:  processViewerPreferences,
	model.ZOOM:                    Zoom,
}

// ValidateCommand creates a new command to validate a file.
func ValidateCommand(inFiles []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.VALIDATE
	return &Command{
		Mode:    model.VALIDATE,
		InFiles: inFiles,
		Conf:    conf}
}

// OptimizeCommand creates a new command to optimize a file.
func OptimizeCommand(inFile, outFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.OPTIMIZE
	return &Command{
		Mode:    model.OPTIMIZE,
		InFile:  &inFile,
		OutFile: &outFile,
		Conf:    conf}
}

// SplitCommand creates a new command to split a file according to span or along bookmarks..
func SplitCommand(inFile, dirNameOut string, span int, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.SPLIT
	return &Command{
		Mode:   model.SPLIT,
		InFile: &inFile,
		OutDir: &dirNameOut,
		IntVal: span,
		Conf:   conf}
}

// SplitByPageNrCommand creates a new command to split a file into files along given pages.
func SplitByPageNrCommand(inFile, dirNameOut string, pageNrs []int, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.SPLITBYPAGENR
	return &Command{
		Mode:    model.SPLITBYPAGENR,
		InFile:  &inFile,
		OutDir:  &dirNameOut,
		IntVals: pageNrs,
		Conf:    conf}
}

// MergeCreateCommand creates a new command to merge files.
// Outfile will be created. An existing outFile will be overwritten.
func MergeCreateCommand(inFiles []string, outFile string, dividerPage bool, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.MERGECREATE
	return &Command{
		Mode:     model.MERGECREATE,
		InFiles:  inFiles,
		OutFile:  &outFile,
		BoolVal1: dividerPage,
		Conf:     conf}
}

// MergeCreateZipCommand creates a new command to zip merge 2 files.
// Outfile will be created. An existing outFile will be overwritten.
func MergeCreateZipCommand(inFiles []string, outFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.MERGECREATEZIP
	return &Command{
		Mode:    model.MERGECREATEZIP,
		InFiles: inFiles,
		OutFile: &outFile,
		Conf:    conf}
}

// MergeAppendCommand creates a new command to merge files.
// Any existing outFile PDF content will be preserved and serves as the beginning of the merge result.
func MergeAppendCommand(inFiles []string, outFile string, dividerPage bool, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.MERGEAPPEND
	return &Command{
		Mode:     model.MERGEAPPEND,
		InFiles:  inFiles,
		OutFile:  &outFile,
		BoolVal1: dividerPage,
		Conf:     conf}
}

// ExtractImagesCommand creates a new command to extract embedded images.
// (experimental)
func ExtractImagesCommand(inFile string, outDir string, pageSelection []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXTRACTIMAGES
	return &Command{
		Mode:          model.EXTRACTIMAGES,
		InFile:        &inFile,
		OutDir:        &outDir,
		PageSelection: pageSelection,
		Conf:          conf}
}

// ExtractFontsCommand creates a new command to extract embedded fonts.
// (experimental)
func ExtractFontsCommand(inFile string, outDir string, pageSelection []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXTRACTFONTS
	return &Command{
		Mode:          model.EXTRACTFONTS,
		InFile:        &inFile,
		OutDir:        &outDir,
		PageSelection: pageSelection,
		Conf:          conf}
}

// ExtractPagesCommand creates a new command to extract specific pages of a file.
func ExtractPagesCommand(inFile string, outDir string, pageSelection []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXTRACTPAGES
	return &Command{
		Mode:          model.EXTRACTPAGES,
		InFile:        &inFile,
		OutDir:        &outDir,
		PageSelection: pageSelection,
		Conf:          conf}
}

// ExtractContentCommand creates a new command to extract page content streams.
func ExtractContentCommand(inFile string, outDir string, pageSelection []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXTRACTCONTENT
	return &Command{
		Mode:          model.EXTRACTCONTENT,
		InFile:        &inFile,
		OutDir:        &outDir,
		PageSelection: pageSelection,
		Conf:          conf}
}

// ExtractMetadataCommand creates a new command to extract metadata streams.
func ExtractMetadataCommand(inFile string, outDir string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXTRACTMETADATA
	return &Command{
		Mode:   model.EXTRACTMETADATA,
		InFile: &inFile,
		OutDir: &outDir,
		Conf:   conf}
}

// TrimCommand creates a new command to trim the pages of a file.
func TrimCommand(inFile, outFile string, pageSelection []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.TRIM
	return &Command{
		Mode:          model.TRIM,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Conf:          conf}
}

// ListAttachmentsCommand create a new command to list attachments.
func ListAttachmentsCommand(inFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTATTACHMENTS
	return &Command{
		Mode:   model.LISTATTACHMENTS,
		InFile: &inFile,
		Conf:   conf}
}

// AddAttachmentsCommand creates a new command to add attachments.
func AddAttachmentsCommand(inFile, outFile string, fileNames []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ADDATTACHMENTS
	return &Command{
		Mode:    model.ADDATTACHMENTS,
		InFile:  &inFile,
		OutFile: &outFile,
		InFiles: fileNames,
		Conf:    conf}
}

// AddAttachmentsPortfolioCommand creates a new command to add attachments to a portfolio.
func AddAttachmentsPortfolioCommand(inFile, outFile string, fileNames []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ADDATTACHMENTSPORTFOLIO
	return &Command{
		Mode:    model.ADDATTACHMENTSPORTFOLIO,
		InFile:  &inFile,
		OutFile: &outFile,
		InFiles: fileNames,
		Conf:    conf}
}

// RemoveAttachmentsCommand creates a new command to remove attachments.
func RemoveAttachmentsCommand(inFile, outFile string, fileNames []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVEATTACHMENTS
	return &Command{
		Mode:    model.REMOVEATTACHMENTS,
		InFile:  &inFile,
		OutFile: &outFile,
		InFiles: fileNames,
		Conf:    conf}
}

// ExtractAttachmentsCommand creates a new command to extract attachments.
func ExtractAttachmentsCommand(inFile string, outDir string, fileNames []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXTRACTATTACHMENTS
	return &Command{
		Mode:    model.EXTRACTATTACHMENTS,
		InFile:  &inFile,
		OutDir:  &outDir,
		InFiles: fileNames,
		Conf:    conf}
}

// EncryptCommand creates a new command to encrypt a file.
func EncryptCommand(inFile, outFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ENCRYPT
	return &Command{
		Mode:    model.ENCRYPT,
		InFile:  &inFile,
		OutFile: &outFile,
		Conf:    conf}
}

// DecryptCommand creates a new command to decrypt a file.
func DecryptCommand(inFile, outFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.DECRYPT
	return &Command{
		Mode:    model.DECRYPT,
		InFile:  &inFile,
		OutFile: &outFile,
		Conf:    conf}
}

// ChangeUserPWCommand creates a new command to change the user password.
func ChangeUserPWCommand(inFile, outFile string, pwOld, pwNew *string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.CHANGEUPW
	return &Command{
		Mode:    model.CHANGEUPW,
		InFile:  &inFile,
		OutFile: &outFile,
		PWOld:   pwOld,
		PWNew:   pwNew,
		Conf:    conf}
}

// ChangeOwnerPWCommand creates a new command to change the owner password.
func ChangeOwnerPWCommand(inFile, outFile string, pwOld, pwNew *string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.CHANGEOPW
	return &Command{
		Mode:    model.CHANGEOPW,
		InFile:  &inFile,
		OutFile: &outFile,
		PWOld:   pwOld,
		PWNew:   pwNew,
		Conf:    conf}
}

// ListPermissionsCommand create a new command to list permissions.
func ListPermissionsCommand(inFiles []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTPERMISSIONS
	return &Command{
		Mode:    model.LISTPERMISSIONS,
		InFiles: inFiles,
		Conf:    conf}
}

// SetPermissionsCommand creates a new command to add permissions.
func SetPermissionsCommand(inFile, outFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.SETPERMISSIONS
	return &Command{
		Mode:    model.SETPERMISSIONS,
		InFile:  &inFile,
		OutFile: &outFile,
		Conf:    conf}
}

// AddWatermarksCommand creates a new command to add Watermarks to a file.
func AddWatermarksCommand(inFile, outFile string, pageSelection []string, wm *model.Watermark, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ADDWATERMARKS
	return &Command{
		Mode:          model.ADDWATERMARKS,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Watermark:     wm,
		Conf:          conf}
}

// RemoveWatermarksCommand creates a new command to remove Watermarks from a file.
func RemoveWatermarksCommand(inFile, outFile string, pageSelection []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVEWATERMARKS
	return &Command{
		Mode:          model.REMOVEWATERMARKS,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Conf:          conf}
}

// ImportImagesCommand creates a new command to import images.
func ImportImagesCommand(imageFiles []string, outFile string, imp *pdfcpu.Import, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.IMPORTIMAGES
	return &Command{
		Mode:    model.IMPORTIMAGES,
		InFiles: imageFiles,
		OutFile: &outFile,
		Import:  imp,
		Conf:    conf}
}

// InsertPagesCommand creates a new command to insert a blank page before or after selected pages.
func InsertPagesCommand(inFile, outFile string, pageSelection []string, conf *model.Configuration, mode string) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	cmdMode := model.INSERTPAGESBEFORE
	if mode == "after" {
		cmdMode = model.INSERTPAGESAFTER
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
func RemovePagesCommand(inFile, outFile string, pageSelection []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVEPAGES
	return &Command{
		Mode:          model.REMOVEPAGES,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Conf:          conf}
}

// RotateCommand creates a new command to rotate pages.
func RotateCommand(inFile, outFile string, rotation int, pageSelection []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ROTATE
	return &Command{
		Mode:          model.ROTATE,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		IntVal:        rotation,
		Conf:          conf}
}

// NUpCommand creates a new command to render PDFs or image files in n-up fashion.
func NUpCommand(inFiles []string, outFile string, pageSelection []string, nUp *model.NUp, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.NUP
	return &Command{
		Mode:          model.NUP,
		InFiles:       inFiles,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		NUp:           nUp,
		Conf:          conf}
}

// BookletCommand creates a new command to render PDFs or image files in booklet fashion.
func BookletCommand(inFiles []string, outFile string, pageSelection []string, nup *model.NUp, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.BOOKLET
	return &Command{
		Mode:          model.BOOKLET,
		InFiles:       inFiles,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		NUp:           nup,
		Conf:          conf}
}

// InfoCommand creates a new command to output information about inFile.
func InfoCommand(inFiles []string, pageSelection []string, json bool, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTINFO
	return &Command{
		Mode:          model.LISTINFO,
		InFiles:       inFiles,
		PageSelection: pageSelection,
		BoolVal1:      json,
		Conf:          conf}
}

// ListFontsCommand returns a list of supported fonts.
func ListFontsCommand(conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTFONTS
	return &Command{
		Mode: model.LISTFONTS,
		Conf: conf}
}

// InstallFontsCommand installs true type fonts for embedding.
func InstallFontsCommand(fontFiles []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.INSTALLFONTS
	return &Command{
		Mode:    model.INSTALLFONTS,
		InFiles: fontFiles,
		Conf:    conf}
}

// CreateCheatSheetsFontsCommand creates single page PDF cheat sheets in current dir.
func CreateCheatSheetsFontsCommand(fontFiles []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.CHEATSHEETSFONTS
	return &Command{
		Mode:    model.CHEATSHEETSFONTS,
		InFiles: fontFiles,
		Conf:    conf}
}

// ListKeywordsCommand create a new command to list keywords.
func ListKeywordsCommand(inFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTKEYWORDS
	return &Command{
		Mode:   model.LISTKEYWORDS,
		InFile: &inFile,
		Conf:   conf}
}

// AddKeywordsCommand creates a new command to add keywords.
func AddKeywordsCommand(inFile, outFile string, keywords []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ADDKEYWORDS
	return &Command{
		Mode:       model.ADDKEYWORDS,
		InFile:     &inFile,
		OutFile:    &outFile,
		StringVals: keywords,
		Conf:       conf}
}

// RemoveKeywordsCommand creates a new command to remove keywords.
func RemoveKeywordsCommand(inFile, outFile string, keywords []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVEKEYWORDS
	return &Command{
		Mode:       model.REMOVEKEYWORDS,
		InFile:     &inFile,
		OutFile:    &outFile,
		StringVals: keywords,
		Conf:       conf}
}

// ListPropertiesCommand creates a new command to list document properties.
func ListPropertiesCommand(inFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTPROPERTIES
	return &Command{
		Mode:   model.LISTPROPERTIES,
		InFile: &inFile,
		Conf:   conf}
}

// AddPropertiesCommand creates a new command to add document properties.
func AddPropertiesCommand(inFile, outFile string, properties map[string]string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ADDPROPERTIES
	return &Command{
		Mode:      model.ADDPROPERTIES,
		InFile:    &inFile,
		OutFile:   &outFile,
		StringMap: properties,
		Conf:      conf}
}

// RemovePropertiesCommand creates a new command to remove document properties.
func RemovePropertiesCommand(inFile, outFile string, propKeys []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVEPROPERTIES
	return &Command{
		Mode:       model.REMOVEPROPERTIES,
		InFile:     &inFile,
		OutFile:    &outFile,
		StringVals: propKeys,
		Conf:       conf}
}

// CollectCommand creates a new command to create a custom PDF page sequence.
func CollectCommand(inFile, outFile string, pageSelection []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.COLLECT
	return &Command{
		Mode:          model.COLLECT,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Conf:          conf}
}

// ListBoxesCommand creates a new command to list page boundaries for selected pages.
func ListBoxesCommand(inFile string, pageSelection []string, pb *model.PageBoundaries, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTBOXES
	return &Command{
		Mode:           model.LISTBOXES,
		InFile:         &inFile,
		PageSelection:  pageSelection,
		PageBoundaries: pb,
		Conf:           conf}
}

// AddBoxesCommand creates a new command to add page boundaries for selected pages.
func AddBoxesCommand(inFile, outFile string, pageSelection []string, pb *model.PageBoundaries, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ADDBOXES
	return &Command{
		Mode:           model.ADDBOXES,
		InFile:         &inFile,
		OutFile:        &outFile,
		PageSelection:  pageSelection,
		PageBoundaries: pb,
		Conf:           conf}
}

// RemoveBoxesCommand creates a new command to remove page boundaries for selected pages.
func RemoveBoxesCommand(inFile, outFile string, pageSelection []string, pb *model.PageBoundaries, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVEBOXES
	return &Command{
		Mode:           model.REMOVEBOXES,
		InFile:         &inFile,
		OutFile:        &outFile,
		PageSelection:  pageSelection,
		PageBoundaries: pb,
		Conf:           conf}
}

// CropCommand creates a new command to apply a cropBox to selected pages.
func CropCommand(inFile, outFile string, pageSelection []string, box *model.Box, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.CROP
	return &Command{
		Mode:          model.CROP,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Box:           box,
		Conf:          conf}
}

// ListAnnotationsCommand creates a new command to list annotations for selected pages.
func ListAnnotationsCommand(inFile string, pageSelection []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTANNOTATIONS
	return &Command{
		Mode:          model.LISTANNOTATIONS,
		InFile:        &inFile,
		PageSelection: pageSelection,
		Conf:          conf}
}

// RemoveAnnotationsCommand creates a new command to remove annotations for selected pages.
func RemoveAnnotationsCommand(inFile, outFile string, pageSelection []string, idsAndTypes []string, objNrs []int, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVEANNOTATIONS
	return &Command{
		Mode:          model.REMOVEANNOTATIONS,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		StringVals:    idsAndTypes,
		IntVals:       objNrs,
		Conf:          conf}
}

// ListImagesCommand creates a new command to list annotations for selected pages.
func ListImagesCommand(inFiles []string, pageSelection []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTIMAGES
	return &Command{
		Mode:          model.LISTIMAGES,
		InFiles:       inFiles,
		PageSelection: pageSelection,
		Conf:          conf}
}

// DumpCommand creates a new command to dump objects on stdout.
func DumpCommand(inFilePDF string, vals []int, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.DUMP
	return &Command{
		Mode:    model.DUMP,
		InFile:  &inFilePDF,
		IntVals: vals,
		Conf:    conf}
}

// CreateCommand creates a new command to create a PDF file.
func CreateCommand(inFilePDF, inFileJSON, outFilePDF string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.CREATE
	return &Command{
		Mode:       model.CREATE,
		InFile:     &inFilePDF,
		InFileJSON: &inFileJSON,
		OutFile:    &outFilePDF,
		Conf:       conf}
}

// ListFormFieldsCommand creates a new command to list the field ids from a PDF form.
func ListFormFieldsCommand(inFiles []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTFORMFIELDS
	return &Command{
		Mode:    model.LISTFORMFIELDS,
		InFiles: inFiles,
		Conf:    conf}
}

// RemoveFormFieldsCommand creates a new command to remove fields from a PDF form.
func RemoveFormFieldsCommand(inFile, outFile string, fieldIDs []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVEFORMFIELDS
	return &Command{
		Mode:       model.REMOVEFORMFIELDS,
		InFile:     &inFile,
		OutFile:    &outFile,
		StringVals: fieldIDs,
		Conf:       conf}
}

// LockFormCommand creates a new command to lock PDF form fields.
func LockFormCommand(inFile, outFile string, fieldIDs []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LOCKFORMFIELDS
	return &Command{
		Mode:       model.LOCKFORMFIELDS,
		InFile:     &inFile,
		OutFile:    &outFile,
		StringVals: fieldIDs,
		Conf:       conf}
}

// UnlockFormCommand creates a new command to unlock PDF form fields.
func UnlockFormCommand(inFile, outFile string, fieldIDs []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.UNLOCKFORMFIELDS
	return &Command{
		Mode:       model.UNLOCKFORMFIELDS,
		InFile:     &inFile,
		OutFile:    &outFile,
		StringVals: fieldIDs,
		Conf:       conf}
}

// ResetFormCommand creates a new command to lock PDF form fields.
func ResetFormCommand(inFile, outFile string, fieldIDs []string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.RESETFORMFIELDS
	return &Command{
		Mode:       model.RESETFORMFIELDS,
		InFile:     &inFile,
		OutFile:    &outFile,
		StringVals: fieldIDs,
		Conf:       conf}
}

// ExportFormCommand creates a new command to export a PDF form.
func ExportFormCommand(inFilePDF, outFileJSON string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXPORTFORMFIELDS
	return &Command{
		Mode:        model.EXPORTFORMFIELDS,
		InFile:      &inFilePDF,
		OutFileJSON: &outFileJSON,
		Conf:        conf}
}

// FillFormCommand creates a new command to fill a PDF form with data.
func FillFormCommand(inFilePDF, inFileJSON, outFilePDF string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.FILLFORMFIELDS
	return &Command{
		Mode:       model.FILLFORMFIELDS,
		InFile:     &inFilePDF,
		InFileJSON: &inFileJSON,
		OutFile:    &outFilePDF,
		Conf:       conf}
}

// MultiFillFormCommand creates a new command to fill multiple PDF forms with JSON or CSV data.
func MultiFillFormCommand(inFilePDF, inFileData, outDir, outFilePDF string, merge bool, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.MULTIFILLFORMFIELDS
	return &Command{
		Mode:       model.MULTIFILLFORMFIELDS,
		InFile:     &inFilePDF,
		InFileJSON: &inFileData, // TODO Fix name clash.
		OutDir:     &outDir,
		OutFile:    &outFilePDF,
		BoolVal1:   merge,
		Conf:       conf}
}

// ResizeCommand creates a new command to scale selected pages.
func ResizeCommand(inFile, outFile string, pageSelection []string, resize *model.Resize, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.RESIZE
	return &Command{
		Mode:          model.RESIZE,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Resize:        resize,
		Conf:          conf}
}

// PosterCommand creates a new command to cut and slice pages horizontally or vertically.
func PosterCommand(inFile, outDir, outFile string, pageSelection []string, cut *model.Cut, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.POSTER
	return &Command{
		Mode:          model.POSTER,
		InFile:        &inFile,
		OutDir:        &outDir,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Cut:           cut,
		Conf:          conf}
}

// NDownCommand creates a new command to cut and slice pages horizontally or vertically.
func NDownCommand(inFile, outDir, outFile string, pageSelection []string, n int, cut *model.Cut, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.NDOWN
	return &Command{
		Mode:          model.NDOWN,
		InFile:        &inFile,
		OutDir:        &outDir,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		IntVal:        n,
		Cut:           cut,
		Conf:          conf}
}

// CutCommand creates a new command to cut and slice pages horizontally or vertically.
func CutCommand(inFile, outDir, outFile string, pageSelection []string, cut *model.Cut, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.CUT
	return &Command{
		Mode:          model.CUT,
		InFile:        &inFile,
		OutDir:        &outDir,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Cut:           cut,
		Conf:          conf}
}

// ListBookmarksCommand creates a new command to list bookmarks of inFile.
func ListBookmarksCommand(inFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTBOOKMARKS
	return &Command{
		Mode:   model.LISTBOOKMARKS,
		InFile: &inFile,
		Conf:   conf}
}

// ExportBookmarksCommand creates a new command to export bookmarks of inFile.
func ExportBookmarksCommand(inFile, outFileJSON string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.EXPORTBOOKMARKS
	return &Command{
		Mode:        model.EXPORTBOOKMARKS,
		InFile:      &inFile,
		OutFileJSON: &outFileJSON,
		Conf:        conf}
}

// ImportBookmarksCommand creates a new command to import bookmarks to inFile.
func ImportBookmarksCommand(inFile, inFileJSON, outFile string, replace bool, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.IMPORTBOOKMARKS
	return &Command{
		Mode:       model.IMPORTBOOKMARKS,
		BoolVal1:   replace,
		InFile:     &inFile,
		InFileJSON: &inFileJSON,
		OutFile:    &outFile,
		Conf:       conf}
}

// RemoveBookmarksCommand creates a new command to remove all bookmarks from inFile.
func RemoveBookmarksCommand(inFile, outFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.REMOVEBOOKMARKS
	return &Command{
		Mode:    model.REMOVEBOOKMARKS,
		InFile:  &inFile,
		OutFile: &outFile,
		Conf:    conf}
}

// ListPageLayoutCommand creates a new command to list the document page layout.
func ListPageLayoutCommand(inFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTPAGELAYOUT
	return &Command{
		Mode:   model.LISTPAGELAYOUT,
		InFile: &inFile,
		Conf:   conf}
}

// SetPageLayoutCommand creates a new command to set the document page layout.
func SetPageLayoutCommand(inFile, outFile, value string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.SETPAGELAYOUT
	return &Command{
		Mode:      model.SETPAGELAYOUT,
		InFile:    &inFile,
		OutFile:   &outFile,
		StringVal: value,
		Conf:      conf}
}

// ResetPageLayoutCommand creates a new command to reset the document page layout.
func ResetPageLayoutCommand(inFile, outFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.RESETPAGELAYOUT
	return &Command{
		Mode:    model.RESETPAGELAYOUT,
		InFile:  &inFile,
		OutFile: &outFile,
		Conf:    conf}
}

// ListPageModeCommand creates a new command to list the document page mode.
func ListPageModeCommand(inFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTPAGEMODE
	return &Command{
		Mode:   model.LISTPAGEMODE,
		InFile: &inFile,
		Conf:   conf}
}

// SetPageModeCommand creates a new command to set the document page mode.
func SetPageModeCommand(inFile, outFile, value string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.SETPAGEMODE
	return &Command{
		Mode:      model.SETPAGEMODE,
		InFile:    &inFile,
		OutFile:   &outFile,
		StringVal: value,
		Conf:      conf}
}

// ResetPageModeCommand creates a new command to reset the document page mode.
func ResetPageModeCommand(inFile, outFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.RESETPAGEMODE
	return &Command{
		Mode:    model.RESETPAGEMODE,
		InFile:  &inFile,
		OutFile: &outFile,
		Conf:    conf}
}

// ListViewerPreferencesCommand creates a new command to list the viewer preferences.
func ListViewerPreferencesCommand(inFile string, all, json bool, conf *model.Configuration) *Command {

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.LISTVIEWERPREFERENCES
	return &Command{
		Mode:     model.LISTVIEWERPREFERENCES,
		InFile:   &inFile,
		BoolVal1: all,
		BoolVal2: json,
		Conf:     conf}
}

// SetViewerPreferencesCommand creates a new command to set the viewer preferences.
func SetViewerPreferencesCommand(inFilePDF, inFileJSON, outFilePDF, stringJSON string, conf *model.Configuration) *Command {

	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.SETVIEWERPREFERENCES
	return &Command{
		Mode:       model.SETVIEWERPREFERENCES,
		InFile:     &inFilePDF,
		InFileJSON: &inFileJSON,
		OutFile:    &outFilePDF,
		StringVal:  stringJSON,
		Conf:       conf}
}

// ResetViewerPreferencesCommand creates a new command to reset the viewer preferences.
func ResetViewerPreferencesCommand(inFile, outFile string, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.RESETVIEWERPREFERENCES
	return &Command{
		Mode:    model.RESETVIEWERPREFERENCES,
		InFile:  &inFile,
		OutFile: &outFile,
		Conf:    conf}
}

// ZoomCommand creates a new command to zoom in/out of selected pages.
func ZoomCommand(inFile, outFile string, pageSelection []string, zoom *model.Zoom, conf *model.Configuration) *Command {
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	}
	conf.Cmd = model.ZOOM
	return &Command{
		Mode:          model.ZOOM,
		InFile:        &inFile,
		OutFile:       &outFile,
		PageSelection: pageSelection,
		Zoom:          zoom,
		Conf:          conf}
}
