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
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pkg/errors"
)

func fileSpecStreamDictInfo(xRefTable *XRefTable, id string, o Object) (string, *time.Time, error) {
	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return "", nil, err
	}

	// Entry EF is a dict holding a stream dict in entry F.
	o, found := d.Find("EF")
	if !found || o == nil {
		return "", nil, nil
	}

	d, err = xRefTable.DereferenceDict(o)
	if err != nil || o == nil {
		return "", nil, err
	}

	var desc string
	if d := d.StringEntry("Desc"); d != nil {
		desc = *d
	}

	// Entry F holds the embedded file's data.
	o, found = d.Find("F")
	if !found || o == nil {
		return desc, nil, nil
	}

	sd, err := xRefTable.DereferenceStreamDict(o)
	if err != nil || sd == nil {
		return desc, nil, err
	}

	//modDate := sd.StringEntry("ModDate")
	// Parse into time.Now()
	modTime := time.Now()

	return desc, &modTime, nil
}

func decodedFileSpecStreamDict(xRefTable *XRefTable, id string, o Object) (*StreamDict, string, *time.Time, error) {
	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return nil, "", nil, err
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

	var desc string
	if d := d.StringEntry("Desc"); d != nil {
		desc = *d
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

	//modDate := sd.StringEntry("ModDate")
	// Parse into time.Now()
	modTime := time.Now()

	fpl := sd.FilterPipeline

	if fpl == nil {

		sd.Content = sd.Raw

	} else {

		// Ignore filter chains with length > 1
		if len(fpl) > 1 {
			log.Debug.Printf("decodedFileSpecStreamDict: ignore %s, more than 1 filter.\n", id)
			return nil, "", nil, nil
		}

		// Only FlateDecode supported.
		if fpl[0].Name != filter.Flate {
			log.Debug.Printf("decodedFileSpecStreamDict: ignore %s, %s filter unsupported.\n", id, fpl[0].Name)
			return nil, "", nil, nil
		}

		// Decode streamDict for supported filters only.
		if err := decodeStream(sd); err != nil {
			return nil, "", nil, err
		}

	}

	return sd, desc, &modTime, nil
}

func extractAttachedFiles(ctx *Context, files StringSet) error {

	writeFile := func(xRefTable *XRefTable, fileName string, o Object) error {

		path := ctx.Write.DirName + "/" + fileName

		log.Debug.Printf("writeFile begin: %s\n", path)

		sd, _, _, err := decodedFileSpecStreamDict(xRefTable, fileName, o)
		if err != nil {
			return err
		}

		log.Info.Printf("writing %s\n", path)

		// TODO Refactor into returning only stream object numbers for files to be extracted.
		// No writing to file in library!
		if err := ioutil.WriteFile(path, sd.Content, os.ModePerm); err != nil {
			return err
		}

		log.Debug.Printf("writeFile end: %s \n", path)

		return nil
	}

	if len(files) > 0 {

		for fileName := range files {

			v, ok := ctx.Names["EmbeddedFiles"].Value(fileName)
			if !ok {
				log.Info.Printf("extractAttachedFiles: %s not found", fileName)
				continue
			}

			if err := writeFile(ctx.XRefTable, fileName, v); err != nil {
				return err
			}

		}

		return nil
	}

	// Extract all files.
	return ctx.Names["EmbeddedFiles"].Process(ctx.XRefTable, writeFile)
}

func fileSpectDict(xRefTable *XRefTable, filename, desc string) (*IndirectRef, error) {
	sd, err := xRefTable.NewEmbeddedFileStreamDict(filename)
	if err != nil {
		return nil, err
	}

	_, fn := filepath.Split(filename)
	d, err := xRefTable.NewFileSpecDict(fn, desc, *sd)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(d)
}

// ok returns true if at least one attachment was added.
func addAttachedFiles(xRefTable *XRefTable, files StringSet, coll bool) (bool, error) {

	if coll {
		// Ensure a Collection entry in the catalog.
		if err := xRefTable.EnsureCollection(); err != nil {
			return false, err
		}
	}

	var ok bool
	for f := range files {

		s := strings.Split(f, ",")
		if len(s) == 0 || len(s) > 2 {
			continue
		}

		fileName := s[0]
		desc := ""
		if len(s) == 2 {
			desc = s[1]
		}

		ir, err := fileSpectDict(xRefTable, fileName, desc)
		if err != nil {
			return false, err
		}

		_, fn := filepath.Split(fileName)
		if err := xRefTable.Names["EmbeddedFiles"].Add(xRefTable, fn, *ir); err != nil {
			return false, err
		}

		ok = true
	}

	return ok, nil
}

// removeAttachedFiles returns true if at least one attachment was removed.
func removeAttachedFiles(xRefTable *XRefTable, files StringSet) (bool, error) {

	// If no files specified, remove all embedded files.
	if len(files) == 0 {
		// Delete name tree root object.
		if err := xRefTable.RemoveEmbeddedFilesNameTree(); err != nil {
			return false, err
		}
		return true, nil
	}

	var removed bool

	for fileName := range files {

		log.Debug.Printf("removeAttachedFiles: removing %s\n", fileName)

		// Any remove operation may be deleting the only key value pair of this name tree.
		if xRefTable.Names["EmbeddedFiles"] == nil {
			//logErrorAttach.Printf("removeAttachedFiles: no attachments, can't remove %s\n", fileName)
			continue
		}

		// EmbeddedFiles name tree containing at least one key value pair.

		empty, ok, err := xRefTable.Names["EmbeddedFiles"].Remove(xRefTable, fileName)
		if err != nil {
			return false, err
		}

		if !ok {
			log.Info.Printf("removeAttachedFiles: %s not found\n", fileName)
			continue
		}

		log.Debug.Printf("removeAttachedFiles: removed key value pair for %s - empty=%t\n", fileName, empty)

		if empty {
			// Delete name tree root object.
			if err := xRefTable.RemoveEmbeddedFilesNameTree(); err != nil {
				return false, err
			}
		}

		removed = true
	}

	return removed, nil
}

// AttachList returns a list of embedded files.
func AttachList(xRefTable *XRefTable) ([]string, error) {

	if !xRefTable.Valid {
		if err := xRefTable.LocateNameTree("EmbeddedFiles", false); err != nil {
			return nil, err
		}
	}

	if xRefTable.Names["EmbeddedFiles"] == nil {
		return nil, nil
	}

	return xRefTable.Names["EmbeddedFiles"].KeyList()
}

// AttachExtract exports specified embedded files.
// If no files specified extract all embedded files.
func AttachExtract(ctx *Context, files StringSet) error {
	if !ctx.Valid {
		if err := ctx.LocateNameTree("EmbeddedFiles", false); err != nil {
			return err
		}
	}
	if ctx.Names["EmbeddedFiles"] == nil {
		return errors.Errorf("no attachments available.")
	}
	return extractAttachedFiles(ctx, files)
}

// AttachAdd embeds specified files.
// Existing attachments are replaced.
// Ensures collection for portfolios.
// Returns true if at least one attachment was added.
func AttachAdd(xRefTable *XRefTable, files StringSet, coll bool) (bool, error) {
	if err := xRefTable.LocateNameTree("EmbeddedFiles", true); err != nil {
		return false, err
	}
	return addAttachedFiles(xRefTable, files, coll)
}

// AttachRemove deletes specified embedded files.
// Returns true if at least one attachment could be removed.
func AttachRemove(xRefTable *XRefTable, files StringSet) (bool, error) {
	if !xRefTable.Valid {
		if err := xRefTable.LocateNameTree("EmbeddedFiles", false); err != nil {
			return false, err
		}
	}
	if xRefTable.Names["EmbeddedFiles"] == nil {
		return false, errors.Errorf("no attachments available.")
	}
	return removeAttachedFiles(xRefTable, files)
}

// Attachment represents a PDF attachment.
type Attachment struct {
	r       io.Reader  // reader, used for add and extract only.
	ID      string     // id
	Desc    string     // description
	ModTime *time.Time // time of last modification
}

// Bytes returns this attachments data.
func (a *Attachment) Bytes() ([]byte, error) {
	if a.r == nil {
		return nil, nil
	}
	return ioutil.ReadAll(a.r)
}

// NewAttachment returns a new attachment.
func NewAttachment(r io.Reader, id string, desc string, modTime *time.Time) *Attachment {
	return &Attachment{
		r:       r,
		ID:      id,
		Desc:    desc,
		ModTime: modTime,
	}
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
		desc, modTime, err := fileSpecStreamDictInfo(xRefTable, id, o)
		if err != nil {
			return err
		}
		a := NewAttachment(nil, id, desc, modTime)
		aa = append(aa, *a)
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

// RemoveAttachments removes attachments with id.
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

	if len(ids) == 0 {
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
		sd, desc, modTime, err := decodedFileSpecStreamDict(xRefTable, id, o)
		if err != nil {
			return err
		}
		a := NewAttachment(bytes.NewReader(sd.Content), id, desc, modTime)
		aa = append(aa, *a)
		return nil
	}

	if len(ids) > 0 {
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
