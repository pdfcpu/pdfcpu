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

package pdfcpu

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pkg/errors"
)

func decodeFileSpecStreamDict(sd *StreamDict, id string) error {
	fpl := sd.FilterPipeline

	if fpl == nil {
		sd.Content = sd.Raw
		return nil
	}

	// Ignore filter chains with length > 1
	if len(fpl) > 1 {
		log.Debug.Printf("decodedFileSpecStreamDict: ignore %s, more than 1 filter.\n", id)
		return nil
	}

	// Only FlateDecode supported.
	if fpl[0].Name != filter.Flate {
		log.Debug.Printf("decodedFileSpecStreamDict: ignore %s, %s filter unsupported.\n", id, fpl[0].Name)
		return nil
	}

	// Decode streamDict for supported filters only.
	return sd.Decode()
}

func fileSpecStreamDictInfo(xRefTable *XRefTable, id string, o Object, decode bool) (*StreamDict, string, *time.Time, error) {
	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return nil, "", nil, err
	}

	var desc string
	if s := d.StringEntry("Desc"); s != nil {
		desc = *s
	}

	// Entry EF is a dict holding a stream dict in entry F.
	o, found := d.Find("EF")
	if !found || o == nil {
		return nil, "", nil, nil
	}

	d, err = xRefTable.DereferenceDict(o)
	if err != nil || o == nil {
		return nil, "", nil, err
	}

	// Entry F holds the embedded file's data.
	o, found = d.Find("F")
	if !found || o == nil {
		return nil, desc, nil, nil
	}

	sd, err := xRefTable.DereferenceStreamDict(o)
	if err != nil || sd == nil {
		return nil, desc, nil, err
	}

	if d = sd.DictEntry("Params"); d == nil {
		return sd, desc, nil, nil
	}

	var modDate *time.Time
	if s := d.StringEntry("ModDate"); s != nil {
		dt, ok := DateTime(*s)
		if !ok {
			return nil, desc, nil, errors.New("pdfcpu: invalid date ModDate")
		}
		modDate = &dt
	}

	err = decodeFileSpecStreamDict(sd, id)

	return sd, desc, modDate, err
}

// Attachment is a Reader representing a PDF attachment.
type Attachment struct {
	io.Reader            // attachment data
	ID        string     // id
	Desc      string     // description
	ModTime   *time.Time // time of last modification (optional)
}

func (a Attachment) String() string {
	return fmt.Sprintf("Attachment: id:%s desc:%s modTime:%s", a.ID, a.Desc, a.ModTime)
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

	createAttachmentStub := func(xRefTable *XRefTable, id string, o Object) error {
		decode := false
		_, desc, modTime, err := fileSpecStreamDictInfo(xRefTable, id, o, decode)
		if err != nil {
			return err
		}
		aa = append(aa, Attachment{nil, id, desc, modTime})
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

	ir, err := xRefTable.NewFileSpectDictForAttachment(a)
	if err != nil {
		return err
	}

	return xRefTable.Names["EmbeddedFiles"].Add(xRefTable, a.ID, *ir)
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
		return false, errors.Errorf("pdfcpu: no attachments available.")
	}

	if ids == nil || len(ids) == 0 {
		// Remove all attachments - delete name tree root object.
		if err := xRefTable.RemoveEmbeddedFilesNameTree(); err != nil {
			return false, err
		}
		return true, nil
	}

	for _, id := range ids {
		// EmbeddedFiles name tree containing at least one key value pair.
		empty, ok, err := xRefTable.Names["EmbeddedFiles"].Remove(xRefTable, id)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
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

// ExtractAttachments extracts attachments with id.
func (ctx *Context) ExtractAttachments(ids []string) ([]Attachment, error) {
	xRefTable := ctx.XRefTable
	if !xRefTable.Valid {
		if err := xRefTable.LocateNameTree("EmbeddedFiles", false); err != nil {
			return nil, err
		}
	}
	if xRefTable.Names["EmbeddedFiles"] == nil {
		return nil, errors.Errorf("pdfcpu: no attachments available.")
	}

	aa := []Attachment{}

	createAttachment := func(xRefTable *XRefTable, id string, o Object) error {
		decode := true
		sd, desc, modTime, err := fileSpecStreamDictInfo(xRefTable, id, o, decode)
		if err != nil {
			return err
		}
		a := Attachment{Reader: bytes.NewReader(sd.Content), ID: id, Desc: desc, ModTime: modTime}
		aa = append(aa, a)
		return nil
	}

	if ids != nil && len(ids) > 0 {
		for _, id := range ids {
			v, ok := ctx.Names["EmbeddedFiles"].Value(id)
			if !ok {
				log.Info.Printf("pdfcpu: extractAttachments: %s not found", id)
				continue
			}
			if err := createAttachment(ctx.XRefTable, id, v); err != nil {
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
