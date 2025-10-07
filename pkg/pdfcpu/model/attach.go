/*
Copyright 2021 The pdfcpu Authors.

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

package model

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// Attachment is a Reader representing a PDF attachment.
type Attachment struct {
	io.Reader            // attachment data
	ID        string     // id
	FileName  string     // filename
	Desc      string     // description
	ModTime   *time.Time // time of last modification (optional)
}

func (a Attachment) String() string {
	return fmt.Sprintf("Attachment: id:%s desc:%s modTime:%s", a.ID, a.Desc, a.ModTime)
}

func decodeFileSpecStreamDict(sd *types.StreamDict) error {
	fpl := sd.FilterPipeline

	if fpl == nil {
		sd.Content = sd.Raw
		return nil
	}

	return sd.Decode()
}

func fileSpecStreamFileName(xRefTable *XRefTable, d types.Dict) (string, error) {
	o, found := d.Find("UF")
	if found {
		return xRefTable.DereferenceStringOrHexLiteral(o, V10, nil)
	}

	o, found = d.Find("F")
	if found {
		return xRefTable.DereferenceStringOrHexLiteral(o, V10, nil)
	}

	return "", errors.New("fileSpecStream missing \"UF\",\"F\"")
}

func fileSpecStreamDict(xRefTable *XRefTable, d types.Dict) (*types.StreamDict, error) {
	// Entry EF is a dict holding a stream dict in entry F.
	o, found := d.Find("EF")
	if !found || o == nil {
		return nil, nil
	}

	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		return nil, err
	}

	// Entry F holds the embedded file's data.
	o, found = d.Find("F")
	if !found || o == nil {
		return nil, nil
	}

	sd, _, err := xRefTable.DereferenceStreamDict(o)
	return sd, err
}

// NewFileSpectDictForAttachment returns a fileSpecDict for a.
func (xRefTable *XRefTable) NewFileSpecDictForAttachment(a Attachment) (types.Dict, error) {
	modTime := time.Now()
	if a.ModTime != nil {
		modTime = *a.ModTime
	}
	sd, err := xRefTable.NewEmbeddedStreamDict(a, modTime)
	if err != nil {
		return nil, err
	}

	// TODO insert (escaped) reverse solidus before solidus between file name components.

	return xRefTable.NewFileSpecDict(a.ID, a.ID, a.Desc, *sd)
}

func getModDate(xRefTable *XRefTable, obj types.Object) (*time.Time, error) {
	errInvalidModDate := errors.New("pdfcpu: invalid date ModDate")
	o, err := xRefTable.Dereference(obj)
	if err != nil || o == nil {
		return nil, errInvalidModDate
	}
	sl, ok := o.(types.StringLiteral)
	if !ok {
		return nil, errInvalidModDate
	}
	s, err := types.StringLiteralToString(sl)
	if err != nil {
		return nil, errInvalidModDate
	}
	md, ok := types.DateTime(s, xRefTable.ValidationMode == ValidationRelaxed)
	if !ok {
		return nil, errInvalidModDate
	}
	return &md, nil
}

func fileSpecStreamDictInfo(xRefTable *XRefTable, id string, o types.Object, decode bool) (*types.StreamDict, string, string, *time.Time, error) {
	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return nil, "", "", nil, err
	}

	var desc string
	o, found := d.Find("Desc")
	if found {
		desc, err = xRefTable.DereferenceStringOrHexLiteral(o, V10, nil)
		if err != nil {
			return nil, "", "", nil, err
		}
	}

	fileName, err := fileSpecStreamFileName(xRefTable, d)
	if err != nil {
		return nil, "", "", nil, err
	}

	sd, err := fileSpecStreamDict(xRefTable, d)
	if err != nil {
		return nil, "", "", nil, err
	}

	var modDate *time.Time
	if d = sd.DictEntry("Params"); d != nil {
		obj, ok := d.Find("ModDate")
		if ok {
			modDate, err = getModDate(xRefTable, obj)
			if err != nil {
				return nil, desc, "", nil, err
			}
		}
	}

	err = decodeFileSpecStreamDict(sd)

	return sd, desc, fileName, modDate, err
}

// ListAttachments returns a slice of attachment stubs (attachment w/o data).
func (ctx *Context) ListAttachments() ([]Attachment, error) {
	xRefTable := ctx.XRefTable
	if !xRefTable.Valid {
		if err := xRefTable.LocateNameTree("EmbeddedFiles", false); err != nil {
			return nil, err
		}
	}
	if xRefTable.Names["EmbeddedFiles"] == nil {
		return nil, nil
	}

	aa := []Attachment{}

	createAttachmentStub := func(xRefTable *XRefTable, id string, o *types.Object) error {
		decode := false
		_, desc, fileName, modTime, err := fileSpecStreamDictInfo(xRefTable, id, *o, decode)
		if err != nil {
			return err
		}
		aa = append(aa, Attachment{nil, id, fileName, desc, modTime})
		return nil
	}

	// Extract stub info.
	if err := ctx.Names["EmbeddedFiles"].Process(xRefTable, createAttachmentStub); err != nil {
		return nil, err
	}

	return aa, nil
}

// AddAttachment adds a.
func (ctx *Context) AddAttachment(a Attachment, useCollection bool) error {
	xRefTable := ctx.XRefTable
	if err := xRefTable.LocateNameTree("EmbeddedFiles", true); err != nil {
		return err
	}

	if useCollection {
		// Ensure a Collection entry in the catalog.
		if err := xRefTable.EnsureCollection(); err != nil {
			return err
		}
	}

	d, err := xRefTable.NewFileSpecDictForAttachment(a)
	if err != nil {
		return err
	}

	ir, err := xRefTable.IndRefForNewObject(d)
	if err != nil {
		return err
	}

	m := NameMap{a.ID: []types.Dict{d}}

	return xRefTable.Names["EmbeddedFiles"].Add(xRefTable, a.ID, *ir, m, []string{"F", "UF"})
}

var errContentMatch = errors.New("name tree content match")

// SearchEmbeddedFilesNameTreeNodeByContent tries to identify a name tree by content.
func (ctx *Context) SearchEmbeddedFilesNameTreeNodeByContent(s string) (*string, types.Object, error) {

	var (
		k *string
		v types.Object
	)

	identifyAttachmentStub := func(xRefTable *XRefTable, id string, o *types.Object) error {
		decode := false
		_, desc, fileName, _, err := fileSpecStreamDictInfo(xRefTable, id, *o, decode)
		if err != nil {
			return err
		}
		if s == fileName || s == desc {
			k = &id
			v = *o
			return errContentMatch
		}
		return nil
	}

	if err := ctx.Names["EmbeddedFiles"].Process(ctx.XRefTable, identifyAttachmentStub); err != nil {
		if err != errContentMatch {
			return nil, nil, err
		}
		// Node identified.
		return k, v, nil
	}

	return nil, nil, nil
}

func (ctx *Context) removeAttachment(id string) (bool, error) {
	if log.CLIEnabled() {
		log.CLI.Printf("removing %s\n", id)
	}
	xRefTable := ctx.XRefTable
	// EmbeddedFiles name tree containing at least one key value pair.
	empty, ok, err := xRefTable.Names["EmbeddedFiles"].Remove(xRefTable, id)
	if err != nil {
		return false, err
	}
	if empty {
		// Delete name tree root object.
		if err := xRefTable.RemoveEmbeddedFilesNameTree(); err != nil {
			return false, err
		}
	}
	if !ok {
		// Try to identify name tree node by content.
		k, _, err := ctx.SearchEmbeddedFilesNameTreeNodeByContent(id)
		if err != nil {
			return false, err
		}
		if k == nil {
			if log.CLIEnabled() {
				log.CLI.Printf("attachment %s not found", id)
			}
			return false, nil
		}
		empty, _, err = xRefTable.Names["EmbeddedFiles"].Remove(xRefTable, *k)
		if err != nil {
			return false, err
		}
		if empty {
			// Delete name tree root object.
			if err := xRefTable.RemoveEmbeddedFilesNameTree(); err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

// RemoveAttachments removes attachments with given id and returns true if anything removed.
func (ctx *Context) RemoveAttachments(ids []string) (bool, error) {
	// Note: Any remove operation may be deleting the only key value pair of this name tree.
	xRefTable := ctx.XRefTable
	if !xRefTable.Valid {
		if err := xRefTable.LocateNameTree("EmbeddedFiles", false); err != nil {
			return false, err
		}
	}
	if xRefTable.Names["EmbeddedFiles"] == nil {
		return false, errors.Errorf("no attachments available.")
	}

	if len(ids) == 0 {
		// Remove all attachments - delete name tree root object.
		if log.CLIEnabled() {
			log.CLI.Println("removing all attachments")
		}
		if err := xRefTable.RemoveEmbeddedFilesNameTree(); err != nil {
			return false, err
		}
		return true, nil
	}

	for _, id := range ids {
		found, err := ctx.removeAttachment(id)
		if err != nil {
			return false, err
		}
		if !found {
			return false, nil
		}
	}

	return true, nil
}

// RemoveAttachment removes a and returns true on success.
func (ctx *Context) RemoveAttachment(a Attachment) (bool, error) {
	return ctx.RemoveAttachments([]string{a.ID})
}

// ExtractAttachments extracts attachments with id.
func (ctx *Context) ExtractAttachments(ids []string) ([]Attachment, error) {
	xRefTable := ctx.XRefTable
	if !xRefTable.Valid {
		if err := xRefTable.LocateNameTree("EmbeddedFiles", false); err != nil {
			return nil, err
		}
	}
	if xRefTable.Names["EmbeddedFiles"] == nil {
		return nil, errors.Errorf("no attachments available.")
	}

	aa := []Attachment{}

	createAttachment := func(xRefTable *XRefTable, id string, o *types.Object) error {
		decode := true
		sd, desc, fileName, modTime, err := fileSpecStreamDictInfo(xRefTable, id, *o, decode)
		if err != nil {
			return err
		}
		a := Attachment{Reader: bytes.NewReader(sd.Content), ID: id, FileName: fileName, Desc: desc, ModTime: modTime}
		aa = append(aa, a)
		return nil
	}

	// Search with UF,F,Desc
	if len(ids) > 0 {
		for _, id := range ids {
			v, ok := ctx.Names["EmbeddedFiles"].Value(id)
			if !ok {
				// Try to identify name tree node by content.
				k, o, err := ctx.SearchEmbeddedFilesNameTreeNodeByContent(id)
				if err != nil {
					return nil, err
				}
				if k == nil {
					if log.CLIEnabled() {
						log.CLI.Printf("attachment %s not found", id)
					}
					if log.InfoEnabled() {
						log.Info.Printf("pdfcpu: extractAttachments: %s not found", id)
					}
					continue
				}
				v = o
			}
			if err := createAttachment(ctx.XRefTable, id, &v); err != nil {
				return nil, err
			}
		}
		return aa, nil
	}

	// Extract all files.
	if err := ctx.Names["EmbeddedFiles"].Process(ctx.XRefTable, createAttachment); err != nil {
		return nil, err
	}

	return aa, nil
}

// ExtractAttachment extracts a fully populated attachment.
func (ctx *Context) ExtractAttachment(a Attachment) (*Attachment, error) {
	aa, err := ctx.ExtractAttachments([]string{a.ID})
	if err != nil || len(aa) == 0 {
		return nil, err
	}
	if len(aa) > 1 {
		return nil, errors.Errorf("pdfcpu: unexpected number of attachments: %d", len(aa))
	}
	return &aa[0], nil
}

func (ctx *Context) AddAttachmentsToInfoDigest(ss *[]string) error {
	aa, err := ctx.ListAttachments()
	if err != nil {
		return err
	}
	if len(aa) == 0 {
		return nil
	}

	var list []string
	for _, a := range aa {
		s := a.FileName
		if a.Desc != "" {
			s = fmt.Sprintf("%s (%s)", s, a.Desc)
		}
		list = append(list, s)
	}
	sort.Strings(list)

	for i, s := range list {
		if i == 0 {
			*ss = append(*ss, fmt.Sprintf("%20s: %s", "Attachments", s))
			continue
		}
		*ss = append(*ss, fmt.Sprintf("%20s  %s,", "", s))
	}

	return nil
}
