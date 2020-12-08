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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
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

func processAttachments(cmd *Command) (out []string, err error) {
	switch cmd.Mode {

	case pdfcpu.LISTATTACHMENTS:
		out, err = ListAttachments(cmd)

	case pdfcpu.ADDATTACHMENTS, pdfcpu.ADDATTACHMENTSPORTFOLIO:
		out, err = AddAttachments(cmd)

	case pdfcpu.REMOVEATTACHMENTS:
		out, err = RemoveAttachments(cmd)

	case pdfcpu.EXTRACTATTACHMENTS:
		out, err = ExtractAttachments(cmd)
	}

	return out, err
}

func processKeywords(cmd *Command) (out []string, err error) {
	switch cmd.Mode {

	case pdfcpu.LISTKEYWORDS:
		out, err = ListKeywords(cmd)

	case pdfcpu.ADDKEYWORDS:
		out, err = AddKeywords(cmd)

	case pdfcpu.REMOVEKEYWORDS:
		out, err = RemoveKeywords(cmd)

	}

	return out, err
}

func processProperties(cmd *Command) (out []string, err error) {
	switch cmd.Mode {

	case pdfcpu.LISTPROPERTIES:
		out, err = ListProperties(cmd)

	case pdfcpu.ADDPROPERTIES:
		out, err = AddProperties(cmd)

	case pdfcpu.REMOVEPROPERTIES:
		out, err = RemoveProperties(cmd)

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
		return ListPermissions(cmd)

	case pdfcpu.SETPERMISSIONS:
		return SetPermissions(cmd)
	}

	return nil, nil
}

func processPages(cmd *Command) (out []string, err error) {
	switch cmd.Mode {

	case pdfcpu.INSERTPAGESBEFORE, pdfcpu.INSERTPAGESAFTER:
		return InsertPages(cmd)

	case pdfcpu.REMOVEPAGES:
		return RemovePages(cmd)
	}

	return nil, nil
}

func processPageBoundaries(cmd *Command) (out []string, err error) {
	switch cmd.Mode {

	case pdfcpu.LISTBOXES:
		out, err = ListBoxes(cmd)

	case pdfcpu.ADDBOXES:
		out, err = AddBoxes(cmd)

	case pdfcpu.REMOVEBOXES:
		out, err = RemoveBoxes(cmd)

	case pdfcpu.CROP:
		out, err = Crop(cmd)

	}

	return out, err
}
