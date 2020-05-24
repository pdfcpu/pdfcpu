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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/log"
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
		if err := decodeStream(sd); err != nil {
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

	d, err := xRefTable.NewFileSpecDict(filename, desc, *sd) // Supply description!
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

// KeywordsList returns a list of keywords as recorded in the document info dict.
func KeywordsList(xRefTable *XRefTable) ([]string, error) {
	ss := strings.FieldsFunc(xRefTable.Keywords, func(c rune) bool { return c == ',' || c == ';' || c == '\r' })
	for i, s := range ss {
		ss[i] = strings.TrimSpace(s)
	}
	return ss, nil
}

// KeywordsAdd adds keywords to the document info dict.
// Returns true if at least one keyword was added.
func KeywordsAdd(xRefTable *XRefTable, keywords []string) error {

	list, err := KeywordsList(xRefTable)
	if err != nil {
		return err
	}

	for _, s := range keywords {
		if !MemberOf(s, list) {
			xRefTable.Keywords += ", " + s
		}
	}

	d, err := xRefTable.DereferenceDict(*xRefTable.Info)
	if err != nil || d == nil {
		return err
	}

	d["Keywords"] = StringLiteral(xRefTable.Keywords)

	return nil
}

// KeywordsRemove deletes keywords from the document info dict.
// Returns true if at least one keyword was removed.
func KeywordsRemove(xRefTable *XRefTable, keywords []string) (bool, error) {
	// TODO Handle missing info dict.
	d, err := xRefTable.DereferenceDict(*xRefTable.Info)
	if err != nil || d == nil {
		return false, err
	}

	if len(keywords) == 0 {
		// Remove all keywords.
		delete(d, "Keywords")
		return true, nil
	}

	// Distil document keywords.
	ss := strings.FieldsFunc(xRefTable.Keywords, func(c rune) bool { return c == ',' || c == ';' || c == '\r' })

	xRefTable.Keywords = ""
	var removed bool
	first := true

	for _, s := range ss {
		s = strings.TrimSpace(s)
		if MemberOf(s, keywords) {
			removed = true
			continue
		}
		if first {
			xRefTable.Keywords = s
			first = false
			continue
		}
		xRefTable.Keywords += ", " + s
	}

	if removed {
		d["Keywords"] = StringLiteral(xRefTable.Keywords)
	}

	return removed, nil
}

// PropertiesList returns a list of document properties as recorded in the document info dict.
func PropertiesList(xRefTable *XRefTable) ([]string, error) {
	list := make([]string, 0, len(xRefTable.Properties))
	keys := make([]string, len(xRefTable.Properties))
	i := 0
	for k := range xRefTable.Properties {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := xRefTable.Properties[k]
		list = append(list, fmt.Sprintf("%s = %s", k, v))
	}
	return list, nil
}

// PropertiesAdd adds properties into the document info dict.
// Returns true if at least one property was added.
func PropertiesAdd(xRefTable *XRefTable, properties map[string]string) error {
	// TODO Handle missing info dict.
	d, err := xRefTable.DereferenceDict(*xRefTable.Info)
	if err != nil || d == nil {
		return err
	}
	for k, v := range properties {
		d[k] = StringLiteral(v)
		xRefTable.Properties[k] = v
	}
	return nil
}

// PropertiesRemove deletes specified properties.
// Returns true if at least one property was removed.
func PropertiesRemove(xRefTable *XRefTable, properties []string) (bool, error) {
	// TODO Handle missing info dict.
	d, err := xRefTable.DereferenceDict(*xRefTable.Info)
	if err != nil || d == nil {
		return false, err
	}

	if len(properties) == 0 {
		// Remove all properties.
		for k := range xRefTable.Properties {
			delete(d, k)
		}
		xRefTable.Properties = map[string]string{}
		return true, nil
	}

	var removed bool
	for _, k := range properties {
		_, ok := d[k]
		if ok && !removed {
			delete(d, k)
			delete(xRefTable.Properties, k)
			removed = true
		}
	}

	return removed, nil
}
