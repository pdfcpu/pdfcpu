/*
Copyright 2019 The pdfcpu Authors.

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
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

var (
	// ErrMissingCommand signals a missing CLI command.
	ErrMissingCommand = errors.New("missing command")

	// ErrUnsupportedCommandMode signals a command mode unsupported by a dispatcher.
	ErrUnsupportedCommandMode = errors.New("unsupported command mode")
)

type dispatchFunc func(*Command) ([]string, error)

var dispatchTable = map[model.CommandMode]dispatchFunc{
	// Document commands.
	model.VALIDATE:       Validate,
	model.OPTIMIZE:       Optimize,
	model.LISTINFO:       ListInfo,
	model.DUMP:           Dump,
	model.CREATE:         Create,
	model.MERGECREATE:    MergeCreate,
	model.MERGECREATEZIP: MergeCreateZip,
	model.MERGEAPPEND:    MergeAppend,
	model.SPLIT:          Split,
	model.SPLITBYPAGENR:  SplitByPageNr,
	model.TRIM:           Trim,
	model.COLLECT:        Collect,

	// Page commands.
	model.INSERTPAGESBEFORE: dispatchPages,
	model.INSERTPAGESAFTER:  dispatchPages,
	model.REMOVEPAGES:       dispatchPages,
	model.ROTATE:            Rotate,
	model.NUP:               NUp,
	model.GRID:              Grid,
	model.BOOKLET:           Booklet,
	model.RESIZE:            Resize,
	model.POSTER:            Poster,
	model.NDOWN:             NDown,
	model.CUT:               Cut,
	model.CROP:              dispatchPageBoundaries,
	model.ZOOM:              Zoom,

	// Content commands.
	model.ADDWATERMARKS:          AddWatermarks,
	model.REMOVEWATERMARKS:       RemoveWatermarks,
	model.LISTANNOTATIONS:        dispatchPageAnnotations,
	model.REMOVEANNOTATIONS:      dispatchPageAnnotations,
	model.LISTBOOKMARKS:          dispatchBookmarks,
	model.EXPORTBOOKMARKS:        dispatchBookmarks,
	model.IMPORTBOOKMARKS:        dispatchBookmarks,
	model.REMOVEBOOKMARKS:        dispatchBookmarks,
	model.LISTPAGEMODE:           dispatchPageMode,
	model.SETPAGEMODE:            dispatchPageMode,
	model.RESETPAGEMODE:          dispatchPageMode,
	model.LISTPAGELAYOUT:         dispatchPageLayout,
	model.SETPAGELAYOUT:          dispatchPageLayout,
	model.RESETPAGELAYOUT:        dispatchPageLayout,
	model.LISTVIEWERPREFERENCES:  dispatchViewerPreferences,
	model.SETVIEWERPREFERENCES:   dispatchViewerPreferences,
	model.RESETVIEWERPREFERENCES: dispatchViewerPreferences,

	// Resource commands.
	model.IMPORTIMAGES:            ImportImages,
	model.CHEATSHEETSFONTS:        CreateCheatSheetsFonts,
	model.INSTALLFONTS:            InstallFonts,
	model.LISTFONTS:               ListFonts,
	model.LISTIMAGES:              dispatchImages,
	model.UPDATEIMAGES:            dispatchImages,
	model.LISTATTACHMENTS:         dispatchAttachments,
	model.ADDATTACHMENTS:          dispatchAttachments,
	model.ADDATTACHMENTSPORTFOLIO: dispatchAttachments,
	model.REMOVEATTACHMENTS:       dispatchAttachments,
	model.EXTRACTATTACHMENTS:      dispatchAttachments,
	model.LISTKEYWORDS:            dispatchKeywords,
	model.ADDKEYWORDS:             dispatchKeywords,
	model.REMOVEKEYWORDS:          dispatchKeywords,
	model.LISTPROPERTIES:          dispatchProperties,
	model.ADDPROPERTIES:           dispatchProperties,
	model.REMOVEPROPERTIES:        dispatchProperties,
	model.LISTBOXES:               dispatchPageBoundaries,
	model.ADDBOXES:                dispatchPageBoundaries,
	model.REMOVEBOXES:             dispatchPageBoundaries,

	// Extract commands.
	model.EXTRACTIMAGES:   ExtractImages,
	model.EXTRACTFONTS:    ExtractFonts,
	model.EXTRACTPAGES:    ExtractPages,
	model.EXTRACTCONTENT:  ExtractContent,
	model.EXTRACTMETADATA: ExtractMetadata,

	// Form commands.
	model.LISTFORMFIELDS:      dispatchForm,
	model.REMOVEFORMFIELDS:    dispatchForm,
	model.LOCKFORMFIELDS:      dispatchForm,
	model.UNLOCKFORMFIELDS:    dispatchForm,
	model.RESETFORMFIELDS:     dispatchForm,
	model.EXPORTFORMFIELDS:    dispatchForm,
	model.FILLFORMFIELDS:      dispatchForm,
	model.MULTIFILLFORMFIELDS: dispatchForm,

	// Security commands.
	model.ENCRYPT:         dispatchEncryption,
	model.DECRYPT:         dispatchEncryption,
	model.CHANGEUPW:       dispatchEncryption,
	model.CHANGEOPW:       dispatchEncryption,
	model.LISTPERMISSIONS: dispatchPermissions,
	model.SETPERMISSIONS:  dispatchPermissions,

	// Trust and signature commands.
	model.LISTCERTIFICATES:    dispatchCertificates,
	model.INSPECTCERTIFICATES: dispatchCertificates,
	model.IMPORTCERTIFICATES:  dispatchCertificates,
	model.VALIDATESIGNATURES:  dispatchSignatures,
	model.REMOVESIGNATURES:    dispatchSignatures,
	model.ADDSIGNATURE:        dispatchSignatures,
}

// Dispatch executes a pdfcpu command.
func Dispatch(cmd *Command) (out []string, err error) {
	defer func() {
		if r := recover(); r != nil {
			if p, ok := r.(fault.Panic); ok {
				err = p
				return
			}
			err = fault.Panic{
				Err:   fmt.Errorf("unexpected panic attack: %v", r),
				Stack: debug.Stack(),
			}
		}
	}()

	if cmd == nil {
		return nil, fmt.Errorf("pdfcpu: dispatch: %w", ErrMissingCommand)
	}
	if cmd.Conf == nil {
		cmd.Conf = model.NewDefaultConfiguration()
	}
	cmd.Conf.Cmd = cmd.Mode

	if f, ok := dispatchTable[cmd.Mode]; ok {
		return f(cmd)
	}

	return nil, fmt.Errorf("pdfcpu: dispatch: mode %d: %w", cmd.Mode, ErrUnsupportedCommandMode)
}

func dispatchAttachments(cmd *Command) (out []string, err error) {
	switch cmd.Mode {

	case model.LISTATTACHMENTS:
		out, err = ListAttachments(cmd)

	case model.ADDATTACHMENTS, model.ADDATTACHMENTSPORTFOLIO:
		out, err = AddAttachments(cmd)

	case model.REMOVEATTACHMENTS:
		out, err = RemoveAttachments(cmd)

	case model.EXTRACTATTACHMENTS:
		out, err = ExtractAttachments(cmd)
	}

	return out, err
}

func dispatchBookmarks(cmd *Command) (out []string, err error) {
	switch cmd.Mode {

	case model.LISTBOOKMARKS:
		return ListBookmarks(cmd)

	case model.EXPORTBOOKMARKS:
		return ExportBookmarks(cmd)

	case model.IMPORTBOOKMARKS:
		return ImportBookmarks(cmd)

	case model.REMOVEBOOKMARKS:
		return RemoveBookmarks(cmd)
	}

	return nil, nil
}

func dispatchEncryption(cmd *Command) (out []string, err error) {
	switch cmd.Mode {

	case model.ENCRYPT:
		return Encrypt(cmd)

	case model.DECRYPT:
		return Decrypt(cmd)

	case model.CHANGEUPW:
		return ChangeUserPassword(cmd)

	case model.CHANGEOPW:
		return ChangeOwnerPassword(cmd)
	}

	return nil, nil
}

func dispatchForm(cmd *Command) (out []string, err error) {
	switch cmd.Mode {

	case model.LISTFORMFIELDS:
		return ListFormFields(cmd)

	case model.REMOVEFORMFIELDS:
		return RemoveFormFields(cmd)

	case model.LOCKFORMFIELDS:
		return LockFormFields(cmd)

	case model.UNLOCKFORMFIELDS:
		return UnlockFormFields(cmd)

	case model.RESETFORMFIELDS:
		return ResetFormFields(cmd)

	case model.EXPORTFORMFIELDS:
		return ExportFormFields(cmd)

	case model.FILLFORMFIELDS:
		return FillFormFields(cmd)

	case model.MULTIFILLFORMFIELDS:
		return MultiFillFormFields(cmd)
	}

	return nil, nil
}

func dispatchImages(cmd *Command) (out []string, err error) {
	switch cmd.Mode {

	case model.LISTIMAGES:
		return ListImages(cmd)

	case model.UPDATEIMAGES:
		return UpdateImages(cmd)
	}

	return nil, nil
}

func dispatchKeywords(cmd *Command) (out []string, err error) {
	if cmd == nil {
		return nil, fmt.Errorf("dispatch keywords: %w", ErrMissingCommand)
	}
	switch cmd.Mode {

	case model.LISTKEYWORDS:
		out, err = ListKeywords(cmd)

	case model.ADDKEYWORDS:
		out, err = AddKeywords(cmd)

	case model.REMOVEKEYWORDS:
		out, err = RemoveKeywords(cmd)

	default:
		return nil, fmt.Errorf("dispatch keywords: mode %d: %w", cmd.Mode, ErrUnsupportedCommandMode)
	}

	return out, err
}

func dispatchPageAnnotations(cmd *Command) (out []string, err error) {
	switch cmd.Mode {

	case model.LISTANNOTATIONS:
		out, err = ListAnnotations(cmd)

	case model.REMOVEANNOTATIONS:
		out, err = RemoveAnnotations(cmd)
	}

	return out, err
}

func dispatchPageBoundaries(cmd *Command) (out []string, err error) {
	switch cmd.Mode {

	case model.LISTBOXES:
		return ListBoxes(cmd)

	case model.ADDBOXES:
		return AddBoxes(cmd)

	case model.REMOVEBOXES:
		return RemoveBoxes(cmd)

	case model.CROP:
		return Crop(cmd)
	}

	return nil, nil
}

func dispatchPageLayout(cmd *Command) (out []string, err error) {
	switch cmd.Mode {

	case model.LISTPAGELAYOUT:
		return ListPageLayout(cmd)

	case model.SETPAGELAYOUT:
		return SetPageLayout(cmd)

	case model.RESETPAGELAYOUT:
		return ResetPageLayout(cmd)
	}

	return nil, nil
}

func dispatchPageMode(cmd *Command) (out []string, err error) {
	switch cmd.Mode {

	case model.LISTPAGEMODE:
		return ListPageMode(cmd)

	case model.SETPAGEMODE:
		return SetPageMode(cmd)

	case model.RESETPAGEMODE:
		return ResetPageMode(cmd)
	}

	return nil, nil
}

func dispatchPages(cmd *Command) (out []string, err error) {
	switch cmd.Mode {

	case model.INSERTPAGESBEFORE, model.INSERTPAGESAFTER:
		return InsertPages(cmd)

	case model.REMOVEPAGES:
		return RemovePages(cmd)
	}

	return nil, nil
}

func dispatchPermissions(cmd *Command) (out []string, err error) {
	switch cmd.Mode {

	case model.LISTPERMISSIONS:
		return ListPermissions(cmd)

	case model.SETPERMISSIONS:
		return SetPermissions(cmd)
	}

	return nil, nil
}

func dispatchProperties(cmd *Command) (out []string, err error) {
	switch cmd.Mode {

	case model.LISTPROPERTIES:
		out, err = ListProperties(cmd)

	case model.ADDPROPERTIES:
		out, err = AddProperties(cmd)

	case model.REMOVEPROPERTIES:
		out, err = RemoveProperties(cmd)

	}

	return out, err
}

func dispatchViewerPreferences(cmd *Command) (out []string, err error) {
	switch cmd.Mode {

	case model.LISTVIEWERPREFERENCES:
		return ListViewerPreferences(cmd)

	case model.SETVIEWERPREFERENCES:
		return SetViewerPreferences(cmd)

	case model.RESETVIEWERPREFERENCES:
		return ResetViewerPreferences(cmd)
	}

	return nil, nil
}

func dispatchCertificates(cmd *Command) (out []string, err error) {
	switch cmd.Mode {

	case model.LISTCERTIFICATES:
		return ListCertificates(cmd)

	case model.INSPECTCERTIFICATES:
		return InspectCertificates(cmd)

	case model.IMPORTCERTIFICATES:
		return ImportCertificates(cmd)

	}

	return nil, nil
}

func dispatchSignatures(cmd *Command) (out []string, err error) {
	switch cmd.Mode {

	case model.VALIDATESIGNATURES:
		return ValidateSignatures(cmd)

	case model.REMOVESIGNATURES:
		return RemoveSignatures(cmd)
	}

	return nil, nil
}
