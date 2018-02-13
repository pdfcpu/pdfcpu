// Package attach provides management code for file attachments / embedded files.
package attach

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/hhrutter/pdfcpu/filter"
	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

var logDebugAttach, logInfoAttach, logErrorAttach *log.Logger

func init() {
	logDebugAttach = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	//logDebugAttach = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	logInfoAttach = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	logErrorAttach = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// Verbose controls logging output.
func Verbose(verbose bool) {
	if verbose {
		logDebugAttach = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
		//logInfoAttach = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	} else {
		logDebugAttach = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
		//logInfoAttach = log.New(ioutil.Discard, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	}
}

func processFileSpecDict(ctx *types.PDFContext, dict *types.PDFDict, processor func(string, *types.PDFStreamDict) error) error {

	logDebugAttach.Println("processFileSpecDict begin")

	// Entry F holds the filename.
	obj, found := dict.Find("F")
	if !found || obj == nil {
		return nil
	}
	obj, err := ctx.Dereference(obj)
	if err != nil || obj == nil {
		return err
	}
	fileName, _ := obj.(types.PDFStringLiteral)

	// Entry EF is a dict holding a stream dict in entry F.
	obj, found = dict.Find("EF")
	if !found || obj == nil {
		return nil
	}
	var d *types.PDFDict
	d, err = ctx.DereferenceDict(obj)
	if err != nil || obj == nil {
		return err
	}

	// Entry F holds the embedded file's data.
	obj, found = d.Find("F")
	if !found || obj == nil {
		return nil
	}
	var sd *types.PDFStreamDict
	sd, err = ctx.DereferenceStreamDict(obj)
	if err != nil || sd == nil {
		return err
	}

	err = processor(fileName.Value(), sd)
	if err != nil {
		return err
	}

	logDebugAttach.Println("processFileSpecDict end")

	return nil
}

func processNamesEntry(ctx *types.PDFContext, dict *types.PDFDict, processor func(string, *types.PDFStreamDict) error) error {

	logDebugAttach.Println("processNamesEntry begin")

	// Names: array of the form [key1 value1 key2 value2 ... key n value n]
	obj, found := dict.Find("Names")
	if !found {
		return errors.Errorf("processNamesEntry: missing \"Names\" entry.")
	}

	arr, err := ctx.DereferenceArray(obj)
	if err != nil {
		return err
	}
	if arr == nil {
		return errors.Errorf("processNamesEntry: missing \"Names\" array.")
	}

	// arr length needs to be even because of contained key value pairs.
	if len(*arr)%2 == 1 {
		return errors.Errorf("processNamesEntry: Names array entry length needs to be even, length=%d\n", len(*arr))
	}

	for i, obj := range *arr {

		if i%2 == 0 {
			continue
		}

		logDebugAttach.Printf("processNamesEntry: Names array value: %v\n", obj)

		if obj == nil {
			continue
		}

		d, err := ctx.DereferenceDict(obj)
		if err != nil {
			return err
		}

		if d == nil {
			continue
		}

		logDebugAttach.Printf("processNamesEntry: file spec dict: %v\n", d)
		err = processFileSpecDict(ctx, d, processor)
		if err != nil {
			return err
		}

	}

	logDebugAttach.Println("processNamesEntry end")

	return nil
}

func processNameTree(ctx *types.PDFContext, nameTree types.PDFNameTree, processor func(string, *types.PDFStreamDict) error) error {

	logDebugAttach.Println("processNameTree begin")

	dict, err := ctx.DereferenceDict(nameTree.PDFIndirectRef)
	if err != nil {
		return err
	}

	if obj, found := dict.Find("Kids"); found {

		var arr *types.PDFArray

		arr, err = ctx.DereferenceArray(obj)
		if err != nil {
			return err
		}

		if arr == nil {
			return errors.New("processNameTree: missing \"Kids\" array")
		}

		for _, obj := range *arr {

			logDebugAttach.Printf("processNameTree: processing kid: %v\n", obj)

			kid, ok := obj.(types.PDFIndirectRef)
			if !ok {
				return errors.New("processNameTree: corrupt kid, should be indirect reference")
			}

			err = processNameTree(ctx, *types.NewNameTree(kid), processor)
			if err != nil {
				return err
			}
		}

		logDebugAttach.Println("processNameTree end")

		return nil
	}

	err = processNamesEntry(ctx, dict, processor)
	if err != nil {
		return err
	}

	logDebugAttach.Println("processNameTree end")

	return nil
}

func listAttachedFiles(ctx *types.PDFContext) (list []string, err error) {

	collectFileNames := func(fileName string, sd *types.PDFStreamDict) error {
		list = append(list, fileName)
		return nil
	}

	err = processNameTree(ctx, *ctx.EmbeddedFiles, collectFileNames)

	return list, nil
}

func extractAttachedFiles(ctx *types.PDFContext, files types.StringSet) error {

	writeFile := func(fileName string, sd *types.PDFStreamDict) (err error) {

		path := ctx.Write.DirName + "/" + fileName

		logDebugAttach.Printf("writeFile begin: %s\n", path)

		fpl := sd.FilterPipeline

		if fpl == nil {

			sd.Content = sd.Raw

		} else {

			// Ignore filter chains with length > 1
			if len(fpl) > 1 {
				logDebugAttach.Printf("writeFile end: ignore %s, more than 1 filter.\n", fileName)
				return nil
			}

			// Only FlateDecode supported.
			if fpl[0].Name != "FlateDecode" {
				logDebugAttach.Printf("writeFile: ignore %s, %s filter unsupported.\n", fileName, fpl[0].Name)
				return nil
			}

			// Decode streamDict for supported filters only.
			err = filter.DecodeStream(sd)
			if err != nil {
				return err
			}

		}

		logInfoAttach.Printf("writing %s\n", path)
		err = ioutil.WriteFile(path, sd.Content, os.ModePerm)
		if err != nil {
			return err
		}

		logDebugAttach.Printf("writeFile end: %s \n", path)

		return nil
	}

	nameTree := ctx.EmbeddedFiles

	if len(files) > 0 {

		for fileName := range files {

			// Locate value for name tree key=fileName - the corresponding fileSpecDict.
			indRef, err := nameTree.Value(ctx.XRefTable, fileName)
			if indRef == nil {
				logErrorAttach.Printf("%s not found", fileName)
				continue
			}

			d, err := ctx.DereferenceDict(*indRef)
			if err != nil {
				return err
			}

			if d == nil {
				continue
			}

			// Apply the writeFile processor to this fileSpecDict.
			err = processFileSpecDict(ctx, d, writeFile)
			if err != nil {
				return err
			}

		}

		return nil
	}

	// Extract all files.
	return processNameTree(ctx, *nameTree, writeFile)
}

// ok returns true if at least one attachment was added.
func addAttachedFiles(ctx *types.PDFContext, files types.StringSet) (ok bool, err error) {

	// Ensure a Collection entry in the catalog.
	err = ctx.EnsureCollection()
	if err != nil {
		return false, err
	}

	for filename := range files {

		sd, err := ctx.NewEmbeddedFileStreamDict(filename)
		if err != nil {
			return false, err
		}

		err = filter.EncodeStream(sd)
		if err != nil {
			return false, err
		}

		indRef, err := ctx.IndRefForNewObject(*sd)
		if err != nil {
			return false, err
		}

		dict, err := ctx.NewFileSpecDict(filename, *indRef)
		if err != nil {
			return false, err
		}

		indRef, err = ctx.IndRefForNewObject(*dict)
		if err != nil {
			return false, err
		}

		err = ctx.EmbeddedFiles.SetValue(ctx.XRefTable, filename, *indRef)
		if err != nil {
			return false, err
		}

		ok = true

	}

	return ok, nil
}

// ok returns true if at least one attachment was removed.
func removeAttachedFiles(ctx *types.PDFContext, files types.StringSet) (ok bool, err error) {

	logDebugAttach.Println("removeAttachedFiles begin")

	if ctx.EmbeddedFiles == nil {
		return false, nil
	}

	if len(files) > 0 {

		for fileName := range files {

			logDebugAttach.Printf("removeAttachedFiles: removing %s\n", fileName)

			// Any remove operation may be deleting the only key value pair of this name tree.
			if ctx.EmbeddedFiles != nil {

				// There is an embeddedFiles name tree containing at least one key value pair.
				var found bool
				var deadKid bool

				var root = true
				_, found, deadKid, err = ctx.EmbeddedFiles.RemoveKeyValuePair(ctx.XRefTable, root, fileName)
				if err != nil {
					return false, err
				}

				if !found {
					logErrorAttach.Printf("removeAttachedFiles: %s not found\n", fileName)
					continue
				}

				logDebugAttach.Printf("removeAttachedFiles: removed key value pair for %s - deadKid=%t\n", fileName, deadKid)

				ok = true

				if deadKid {

					// Delete name tree root object.
					// Clean up root.Names entry and delete if EmbeddedFiles was the only Names entry.
					err = ctx.RemoveEmbeddedFilesNameTree()
					if err != nil {
						return false, err
					}

					ctx.EmbeddedFiles = nil
				}

			} else {

				logErrorAttach.Printf("removeAttachedFiles: no attachments, can't remove %s\n", fileName)

			}

		}

		return ok, nil
	}

	// If no files specified, remove all embedded files.
	ok = true

	// Delete name tree root object.

	err = ctx.RemoveEmbeddedFilesNameTree()
	if err != nil {
		return false, err
	}

	ctx.EmbeddedFiles = nil

	return ok, nil
}

// List returns a list of embedded files.
func List(ctx *types.PDFContext) (list []string, err error) {

	logDebugAttach.Println("List begin")

	if !ctx.Valid && ctx.EmbeddedFiles == nil {
		ctx.EmbeddedFiles, err = ctx.LocateNameTree("EmbeddedFiles", false)
		if err != nil {
			return nil, err
		}
	}

	if ctx.EmbeddedFiles == nil {
		return nil, nil
	}

	list, err = listAttachedFiles(ctx)
	if err != nil {
		return nil, err
	}

	logDebugAttach.Println("List end")

	return list, nil
}

// Extract exports specified embedded files.
// If no files specified extract all embedded files.
func Extract(ctx *types.PDFContext, files types.StringSet) (err error) {

	logDebugAttach.Println("Extract begin")

	if !ctx.Valid && ctx.EmbeddedFiles == nil {
		ctx.EmbeddedFiles, err = ctx.LocateNameTree("EmbeddedFiles", false)
		if err != nil {
			return err
		}
	}

	if ctx.EmbeddedFiles == nil {
		return errors.Errorf("no attachments available.")
	}

	err = extractAttachedFiles(ctx, files)
	if err != nil {
		return err
	}

	logDebugAttach.Println("Extract end")

	return nil
}

// Add embeds specified files.
// Existing attachments are replaced.
// ok returns true if at least one attachment was added.
func Add(ctx *types.PDFContext, files types.StringSet) (ok bool, err error) {

	logDebugAttach.Println("Add begin")

	if ctx.EmbeddedFiles == nil {
		ctx.EmbeddedFiles, err = ctx.LocateNameTree("EmbeddedFiles", true)
		if err != nil {
			return false, err
		}
	}

	ok, err = addAttachedFiles(ctx, files)

	logDebugAttach.Println("Add end")

	return ok, err
}

// Remove deletes specified embedded files.
// ok returns true if at least one attachment could be removed.
func Remove(ctx *types.PDFContext, files types.StringSet) (ok bool, err error) {

	logDebugAttach.Println("Remove begin")

	if !ctx.Valid && ctx.EmbeddedFiles == nil {
		ctx.EmbeddedFiles, err = ctx.LocateNameTree("EmbeddedFiles", false)
		if err != nil {
			return false, err
		}
	}

	if ctx.EmbeddedFiles == nil {
		return false, errors.Errorf("no attachments available.")
	}

	ok, err = removeAttachedFiles(ctx, files)

	logDebugAttach.Println("Remove end")

	return ok, err
}
