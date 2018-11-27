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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/hhrutter/pdfcpu/pkg/filter"
	"github.com/hhrutter/pdfcpu/pkg/log"
	"github.com/pkg/errors"
)

func decodedFileSpecStreamDict(xRefTable *XRefTable, fileName string, o Object) (*StreamDict, error) {

	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return nil, err
	}

	// Entry EF is a dict holding a stream dict in entry F.
	o, found := d.Find("EF")
	if !found || o == nil {
		return nil, nil
	}

	d, err = xRefTable.DereferenceDict(o)
	if err != nil || o == nil {
		return nil, err
	}

	// Entry F holds the embedded file's data.
	o, found = d.Find("F")
	if !found || o == nil {
		return nil, nil
	}

	sd, err := xRefTable.DereferenceStreamDict(o)
	if err != nil || sd == nil {
		return nil, err
	}

	fpl := sd.FilterPipeline

	if fpl == nil {

		sd.Content = sd.Raw

	} else {

		// Ignore filter chains with length > 1
		if len(fpl) > 1 {
			log.Debug.Printf("writeFile end: ignore %s, more than 1 filter.\n", fileName)
			return nil, nil
		}

		// Only FlateDecode supported.
		if fpl[0].Name != filter.Flate {
			log.Debug.Printf("writeFile: ignore %s, %s filter unsupported.\n", fileName, fpl[0].Name)
			return nil, nil
		}

		// Decode streamDict for supported filters only.
		err = decodeStream(sd)
		if err != nil {
			return nil, err
		}

	}

	return sd, nil
}

func extractAttachedFiles(ctx *Context, files StringSet) error {

	writeFile := func(xRefTable *XRefTable, fileName string, o Object) error {

		path := ctx.Write.DirName + "/" + fileName

		log.Debug.Printf("writeFile begin: %s\n", path)

		sd, err := decodedFileSpecStreamDict(xRefTable, fileName, o)
		if err != nil {
			return err
		}

		log.Info.Printf("writing %s\n", path)

		// TODO Refactor into returning only stream object numbers for files to be extracted.
		// No writing to file in library!
		err = ioutil.WriteFile(path, sd.Content, os.ModePerm)
		if err != nil {
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

			err := writeFile(ctx.XRefTable, fileName, v)
			if err != nil {
				return err
			}

		}

		return nil
	}

	// Extract all files.
	return ctx.Names["EmbeddedFiles"].Process(ctx.XRefTable, writeFile)
}

func fileSpectDict(xRefTable *XRefTable, filename string) (*IndirectRef, error) {

	sd, err := xRefTable.NewEmbeddedFileStreamDict(filename)
	if err != nil {
		return nil, err
	}

	err = encodeStream(sd)
	if err != nil {
		return nil, err
	}

	ir, err := xRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	d, err := xRefTable.NewFileSpecDict(filename, *ir)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(d)
}

// ok returns true if at least one attachment was added.
func addAttachedFiles(xRefTable *XRefTable, files StringSet) (ok bool, err error) {

	// Ensure a Collection entry in the catalog.
	err = xRefTable.EnsureCollection()
	if err != nil {
		return false, err
	}

	for fileName := range files {

		ir, err := fileSpectDict(xRefTable, fileName)
		if err != nil {
			return false, err
		}

		_, fn := filepath.Split(fileName)

		xRefTable.Names["EmbeddedFiles"].Add(xRefTable, fn, *ir)
		if err != nil {
			return false, err
		}

		ok = true

	}

	return ok, nil
}

// ok returns true if at least one attachment was removed.
func removeAttachedFiles(xRefTable *XRefTable, files StringSet) (ok bool, err error) {

	log.Debug.Println("removeAttachedFiles begin")

	// If no files specified, remove all embedded files.
	if len(files) == 0 {
		err = xRefTable.RemoveEmbeddedFilesNameTree()
		if err != nil {
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
			// Clean up root.Names entry and delete if EmbeddedFiles was the only Names entry.
			err = xRefTable.RemoveEmbeddedFilesNameTree()
			if err != nil {
				return false, err
			}

		}

		removed = true
	}

	return removed, nil
}

// AttachList returns a list of embedded files.
func AttachList(xRefTable *XRefTable) (list []string, err error) {

	log.Debug.Println("List begin")

	if !xRefTable.Valid {
		err = xRefTable.LocateNameTree("EmbeddedFiles", false)
		if err != nil {
			return nil, err
		}
	}

	if xRefTable.Names["EmbeddedFiles"] == nil {
		return nil, nil
	}

	list, err = xRefTable.Names["EmbeddedFiles"].KeyList()
	if err != nil {
		return nil, err
	}

	log.Debug.Println("List end")

	return list, nil
}

// AttachExtract exports specified embedded files.
// If no files specified extract all embedded files.
func AttachExtract(ctx *Context, files StringSet) (err error) {

	log.Debug.Println("Extract begin")

	if !ctx.Valid {
		err = ctx.LocateNameTree("EmbeddedFiles", false)
		if err != nil {
			return err
		}
	}

	if ctx.Names["EmbeddedFiles"] == nil {
		return errors.Errorf("no attachments available.")
	}

	err = extractAttachedFiles(ctx, files)
	if err != nil {
		return err
	}

	log.Debug.Println("Extract end")

	return nil
}

// AttachAdd embeds specified files.
// Existing attachments are replaced.
// ok returns true if at least one attachment was added.
func AttachAdd(xRefTable *XRefTable, files StringSet) (ok bool, err error) {

	log.Debug.Println("Add begin")

	err = xRefTable.LocateNameTree("EmbeddedFiles", true)
	if err != nil {
		return false, err
	}

	ok, err = addAttachedFiles(xRefTable, files)

	log.Debug.Println("Add end")

	return ok, err
}

// AttachRemove deletes specified embedded files.
// ok returns true if at least one attachment could be removed.
func AttachRemove(xRefTable *XRefTable, files StringSet) (ok bool, err error) {

	log.Debug.Println("Remove begin")

	if !xRefTable.Valid {
		err = xRefTable.LocateNameTree("EmbeddedFiles", false)
		if err != nil {
			return false, err
		}
	}

	if xRefTable.Names["EmbeddedFiles"] == nil {
		return false, errors.Errorf("no attachments available.")
	}

	ok, err = removeAttachedFiles(xRefTable, files)

	log.Debug.Println("Remove end")

	return ok, err
}
