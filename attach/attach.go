// Package attach provides management code for file attachments / embedded files.
package attach

import (
	"io/ioutil"
	"os"

	"github.com/hhrutter/pdfcpu/filter"
	"github.com/hhrutter/pdfcpu/log"
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

func decodedFileSpecStreamDict(xRefTable *types.XRefTable, fileName string, o types.PDFObject) (*types.PDFStreamDict, error) {

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
		if fpl[0].Name != "FlateDecode" {
			log.Debug.Printf("writeFile: ignore %s, %s filter unsupported.\n", fileName, fpl[0].Name)
			return nil, nil
		}

		// Decode streamDict for supported filters only.
		err = filter.DecodeStream(sd)
		if err != nil {
			return nil, err
		}

	}

	return sd, nil
}

func extractAttachedFiles(ctx *types.PDFContext, files types.StringSet) error {

	writeFile := func(xRefTable *types.XRefTable, fileName string, o types.PDFObject) error {

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

func fileSpectDict(xRefTable *types.XRefTable, filename string) (*types.PDFIndirectRef, error) {

	sd, err := xRefTable.NewEmbeddedFileStreamDict(filename)
	if err != nil {
		return nil, err
	}

	err = filter.EncodeStream(sd)
	if err != nil {
		return nil, err
	}

	indRef, err := xRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	dict, err := xRefTable.NewFileSpecDict(filename, *indRef)
	if err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*dict)
}

// ok returns true if at least one attachment was added.
func addAttachedFiles(xRefTable *types.XRefTable, files types.StringSet) (ok bool, err error) {

	// Ensure a Collection entry in the catalog.
	err = xRefTable.EnsureCollection()
	if err != nil {
		return false, err
	}

	for fileName := range files {

		indRef, err := fileSpectDict(xRefTable, fileName)
		if err != nil {
			return false, err
		}

		xRefTable.Names["EmbeddedFiles"].Add(xRefTable, fileName, *indRef)
		if err != nil {
			return false, err
		}

		ok = true

	}

	return ok, nil
}

// ok returns true if at least one attachment was removed.
func removeAttachedFiles(xRefTable *types.XRefTable, files types.StringSet) (ok bool, err error) {

	log.Debug.Println("removeAttachedFiles begin")

	if len(files) > 0 {

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

	// If no files specified, remove all embedded files.
	err = xRefTable.RemoveEmbeddedFilesNameTree()
	if err != nil {
		return false, err
	}

	return true, nil
}

// List returns a list of embedded files.
func List(xRefTable *types.XRefTable) (list []string, err error) {

	log.Debug.Println("List begin")

	if !xRefTable.Valid && xRefTable.Names["EmbeddedFiles"] == nil {
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

// Extract exports specified embedded files.
// If no files specified extract all embedded files.
func Extract(ctx *types.PDFContext, files types.StringSet) (err error) {

	log.Debug.Println("Extract begin")

	if !ctx.Valid && ctx.Names["EmbeddedFiles"] == nil {
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

// Add embeds specified files.
// Existing attachments are replaced.
// ok returns true if at least one attachment was added.
func Add(xRefTable *types.XRefTable, files types.StringSet) (ok bool, err error) {

	log.Debug.Println("Add begin")

	if xRefTable.Names["EmbeddedFiles"] == nil {
		err := xRefTable.LocateNameTree("EmbeddedFiles", true)
		if err != nil {
			return false, err
		}
	}

	ok, err = addAttachedFiles(xRefTable, files)

	log.Debug.Println("Add end")

	return ok, err
}

// Remove deletes specified embedded files.
// ok returns true if at least one attachment could be removed.
func Remove(xRefTable *types.XRefTable, files types.StringSet) (ok bool, err error) {

	log.Debug.Println("Remove begin")

	if !xRefTable.Valid && xRefTable.Names["EmbeddedFiles"] == nil {
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
