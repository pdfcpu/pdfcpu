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
	"context"
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

var (
	// ErrMissingCommand signals a missing CLI command.
	ErrMissingCommand = errors.New("missing command")

	// ErrInvalidCommandArguments signals an incomplete or structurally invalid CLI command.
	ErrInvalidCommandArguments = errors.New("invalid command arguments")

	// ErrMissingContext signals a missing required Go context.
	ErrMissingContext = contextutil.ErrMissingContext

	// ErrUnsupportedCommandMode signals a command mode unsupported by a dispatcher.
	ErrUnsupportedCommandMode = errors.New("unsupported command mode")
)

type dispatchFunc func(context.Context, *Command) ([]string, error)

var dispatchTable = map[model.CommandMode]dispatchFunc{
	model.VALIDATE:                validateCommand,
	model.OPTIMIZE:                optimize,
	model.LISTINFO:                listInfoCommand,
	model.DUMP:                    dump,
	model.CREATE:                  create,
	model.MERGECREATE:             mergeCreate,
	model.MERGEAPPEND:             mergeAppend,
	model.MERGECREATEZIP:          mergeCreateZip,
	model.SPLIT:                   split,
	model.SPLITBYPAGENR:           splitByPageNr,
	model.TRIM:                    trim,
	model.COLLECT:                 collect,
	model.ROTATE:                  rotate,
	model.INSERTPAGESBEFORE:       insertPages,
	model.INSERTPAGESAFTER:        insertPages,
	model.REMOVEPAGES:             removePages,
	model.CROP:                    crop,
	model.ZOOM:                    zoom,
	model.RESIZE:                  resize,
	model.NUP:                     nUp,
	model.GRID:                    grid,
	model.BOOKLET:                 booklet,
	model.POSTER:                  poster,
	model.NDOWN:                   nDown,
	model.CUT:                     cut,
	model.ADDWATERMARKS:           addWatermarks,
	model.REMOVEWATERMARKS:        removeWatermarks,
	model.LISTANNOTATIONS:         listAnnotationsForCommand,
	model.REMOVEANNOTATIONS:       removeAnnotations,
	model.LISTBOOKMARKS:           listBookmarks,
	model.EXPORTBOOKMARKS:         exportBookmarks,
	model.IMPORTBOOKMARKS:         importBookmarks,
	model.REMOVEBOOKMARKS:         removeBookmarks,
	model.LISTPAGEMODE:            listPageMode,
	model.SETPAGEMODE:             setPageMode,
	model.RESETPAGEMODE:           resetPageMode,
	model.LISTPAGELAYOUT:          listPageLayout,
	model.SETPAGELAYOUT:           setPageLayout,
	model.RESETPAGELAYOUT:         resetPageLayout,
	model.LISTVIEWERPREFERENCES:   listViewerPreferences,
	model.SETVIEWERPREFERENCES:    setViewerPreferences,
	model.RESETVIEWERPREFERENCES:  resetViewerPreferences,
	model.IMPORTIMAGES:            importImages,
	model.LISTIMAGES:              listImages,
	model.UPDATEIMAGES:            updateImages,
	model.EXTRACTIMAGES:           extractImages,
	model.EXTRACTFONTS:            extractFonts,
	model.EXTRACTPAGES:            extractPages,
	model.EXTRACTCONTENT:          extractContent,
	model.EXTRACTMETADATA:         extractMetadata,
	model.LISTATTACHMENTS:         listAttachmentsCommand,
	model.ADDATTACHMENTS:          addAttachments,
	model.ADDATTACHMENTSPORTFOLIO: addAttachments,
	model.REMOVEATTACHMENTS:       removeAttachments,
	model.EXTRACTATTACHMENTS:      extractAttachments,
	model.LISTKEYWORDS:            listKeywords,
	model.ADDKEYWORDS:             addKeywords,
	model.REMOVEKEYWORDS:          removeKeywords,
	model.LISTPROPERTIES:          listPropertiesCommand,
	model.ADDPROPERTIES:           addProperties,
	model.REMOVEPROPERTIES:        removeProperties,
	model.ENCRYPT:                 encrypt,
	model.DECRYPT:                 decrypt,
	model.CHANGEUPW:               changeUserPassword,
	model.CHANGEOPW:               changeOwnerPassword,
	model.LISTFORMFIELDS:          listFormFieldsForCommand,
	model.REMOVEFORMFIELDS:        removeFormFields,
	model.LOCKFORMFIELDS:          lockFormFields,
	model.UNLOCKFORMFIELDS:        unlockFormFields,
	model.RESETFORMFIELDS:         resetFormFields,
	model.EXPORTFORMFIELDS:        exportFormFields,
	model.FILLFORMFIELDS:          fillFormFields,
	model.MULTIFILLFORMFIELDS:     multiFillFormFields,
	model.LISTPERMISSIONS:         listPermissionsCommand,
	model.SETPERMISSIONS:          setPermissions,
	model.LISTBOXES:               listBoxes,
	model.ADDBOXES:                addBoxes,
	model.REMOVEBOXES:             removeBoxes,
	model.CHEATSHEETSFONTS:        createCheatSheetsFonts,
	model.INSTALLFONTS:            installFonts,
	model.LISTFONTS:               listFonts,
	model.LISTCERTIFICATES:        listCertificates,
	model.INSPECTCERTIFICATES:     inspectCertificates,
	model.IMPORTCERTIFICATES:      importCertificates,
	model.VALIDATESIGNATURES:      validateSignaturesCommand,
	model.REMOVESIGNATURES:        removeSignatures,
}

func configurationForMode(conf *model.Configuration, mode model.CommandMode) *model.Configuration {
	if conf != nil && conf.Cmd == mode {
		return conf
	}
	if conf == nil {
		conf = model.NewDefaultConfiguration()
	} else {
		conf = conf.Clone()
	}
	conf.Cmd = mode
	return conf
}

// Dispatch executes a pdfcpu command with a caller-owned Go context.
func Dispatch(c context.Context, cmd *Command) (out []string, err error) {
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

	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if cmd == nil {
		return nil, fmt.Errorf("pdfcpu: dispatch: %w", ErrMissingCommand)
	}
	f, ok := dispatchTable[cmd.Mode]
	if !ok {
		return nil, fmt.Errorf("pdfcpu: dispatch: mode %d: %w", cmd.Mode, ErrUnsupportedCommandMode)
	}

	execution := *cmd
	if cmd.Conf == nil {
		execution.Conf = model.NewDefaultConfiguration()
	} else {
		execution.Conf = cmd.Conf.Clone()
	}
	execution.Conf.Cmd = execution.Mode

	return f(c, &execution)
}
