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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pkg/errors"
)

// Process executes a pdfcpu command.
func Process(cmd *Command) (out []string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("unexpected panic attack: %v\n", r)
		}
	}()

	cmd.Conf.Cmd = cmd.Mode

	if f, ok := cmdMap[cmd.Mode]; ok {
		return f(cmd)
	}

	return nil, errors.Errorf("pdfcpu: process: Unknown command mode %d\n", cmd.Mode)
}

func processPageAnnotations(cmd *Command) (out []string, err error) {
	switch cmd.Mode {

	case model.LISTANNOTATIONS:
		out, err = ListAnnotations(cmd)

	case model.REMOVEANNOTATIONS:
		out, err = RemoveAnnotations(cmd)
	}

	return out, err
}

func processAttachments(cmd *Command) (out []string, err error) {
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

func processKeywords(cmd *Command) (out []string, err error) {
	switch cmd.Mode {

	case model.LISTKEYWORDS:
		out, err = ListKeywords(cmd)

	case model.ADDKEYWORDS:
		out, err = AddKeywords(cmd)

	case model.REMOVEKEYWORDS:
		out, err = RemoveKeywords(cmd)

	}

	return out, err
}

func processProperties(cmd *Command) (out []string, err error) {
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

func processEncryption(cmd *Command) (out []string, err error) {
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

func processPermissions(cmd *Command) (out []string, err error) {
	switch cmd.Mode {

	case model.LISTPERMISSIONS:
		return ListPermissions(cmd)

	case model.SETPERMISSIONS:
		return SetPermissions(cmd)
	}

	return nil, nil
}

func processPages(cmd *Command) (out []string, err error) {
	switch cmd.Mode {

	case model.INSERTPAGESBEFORE, model.INSERTPAGESAFTER:
		return InsertPages(cmd)

	case model.REMOVEPAGES:
		return RemovePages(cmd)
	}

	return nil, nil
}

func processPageBoundaries(cmd *Command) (out []string, err error) {
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

func processImages(cmd *Command) (out []string, err error) {
	switch cmd.Mode {

	case model.LISTIMAGES:
		return ListImages(cmd)
	}

	return nil, nil
}

func processForm(cmd *Command) (out []string, err error) {
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

func processBookmarks(cmd *Command) (out []string, err error) {
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
