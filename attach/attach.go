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

func processFileSpecDict(ctx *types.PDFContext, dict *types.PDFDict, processor func(string, *types.PDFStreamDict) error) (err error) {

	logDebugAttach.Println("processFileSpecDict begin")

	// Entry F holds the filename.
	obj, found := dict.Find("F")
	if !found || obj == nil {
		return
	}
	obj, err = ctx.Dereference(obj)
	if err != nil || obj == nil {
		return
	}
	fileName, _ := obj.(types.PDFStringLiteral)

	// Entry EF is a dict holding a stream dict in entry F.
	obj, found = dict.Find("EF")
	if !found || obj == nil {
		return
	}
	var d *types.PDFDict
	d, err = ctx.DereferenceDict(obj)
	if err != nil {
		return
	}
	if obj == nil {
		return
	}

	// Entry F holds the embedded file's data.
	obj, found = d.Find("F")
	if !found || obj == nil {
		return
	}
	var sd *types.PDFStreamDict
	sd, err = ctx.DereferenceStreamDict(obj)
	if err != nil {
		return
	}
	if sd == nil {
		return
	}

	err = processor(fileName.Value(), sd)
	if err != nil {
		return
	}

	logDebugAttach.Println("processFileSpecDict end")

	return
}

func processNamesEntry(ctx *types.PDFContext, dict *types.PDFDict, processor func(string, *types.PDFStreamDict) error) (err error) {

	logDebugAttach.Println("processNamesEntry begin")

	// Names: array of the form [key1 value1 key2 value2 ... key n value n]
	obj, found := dict.Find("Names")
	if !found {
		return errors.Errorf("processNamesEntry: missing \"Names\" entry.")
	}

	var arr *types.PDFArray
	var d *types.PDFDict

	arr, err = ctx.DereferenceArray(obj)
	if err != nil {
		return
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

		d, err = ctx.DereferenceDict(obj)
		if err != nil {
			return err
		}

		if d == nil {
			continue
		}

		logDebugAttach.Printf("processNamesEntry: file spec dict: %v\n", d)
		err = processFileSpecDict(ctx, d, processor)
		if err != nil {
			return
		}

	}

	logDebugAttach.Println("processNamesEntry end")

	return
}

func processNameTree(ctx *types.PDFContext, nameTree types.PDFNameTree, processor func(string, *types.PDFStreamDict) error) (err error) {

	logDebugAttach.Println("processNameTree begin")

	var dict *types.PDFDict

	dict, err = ctx.DereferenceDict(nameTree.PDFIndirectRef)
	if err != nil {
		return
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

		return
	}

	err = processNamesEntry(ctx, dict, processor)
	if err != nil {
		return
	}

	logDebugAttach.Println("processNameTree end")

	return
}

func listAttachedFiles(ctx *types.PDFContext) (list []string, err error) {

	collectFileNames := func(fileName string, sd *types.PDFStreamDict) (err error) {
		list = append(list, fileName)
		return
	}

	err = processNameTree(ctx, *ctx.EmbeddedFiles, collectFileNames)

	return
}

func extractAttachedFiles(ctx *types.PDFContext, files types.StringSet) (err error) {

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
				return
			}

			// Only FlateDecode supported.
			if fpl[0].Name != "FlateDecode" {
				logDebugAttach.Printf("writeFile: ignore %s, %s filter unsupported.\n", fileName, fpl[0].Name)
				return
			}

			// Decode streamDict for supported filters only.
			err = filter.DecodeStream(sd)
			if err != nil {
				return
			}

		}

		logInfoAttach.Printf("writing %s\n", path)
		err = ioutil.WriteFile(path, sd.Content, os.ModePerm)
		if err != nil {
			return
		}

		logDebugAttach.Printf("writeFile end: %s \n", path)

		return
	}

	nameTree := ctx.EmbeddedFiles

	if len(files) > 0 {

		for fileName := range files {

			// Locate value for name tree key=fileName - the corresponding fileSpecDict.
			var indRef *types.PDFIndirectRef
			indRef, err = nameTree.Value(ctx.XRefTable, fileName)
			if indRef == nil {
				logErrorAttach.Printf("%s not found", fileName)
				continue
			}

			var d *types.PDFDict
			d, err = ctx.DereferenceDict(*indRef)
			if err != nil {
				return
			}

			if d == nil {
				continue
			}

			// Apply the writeFile processor to this fileSpecDict.
			err = processFileSpecDict(ctx, d, writeFile)
			if err != nil {
				return
			}

		}

		return
	}

	// Extract all files.
	return processNameTree(ctx, *nameTree, writeFile)
}

func createFileSpecDict(ctx *types.PDFContext, filename string, indRefStreamDict types.PDFIndirectRef) (dict *types.PDFDict, err error) {

	d := types.NewPDFDict()
	d.Insert("Type", types.PDFName("Filespec"))
	d.Insert("F", types.PDFStringLiteral(filename))
	d.Insert("UF", types.PDFStringLiteral(filename))
	// TODO d.Insert("UF", utf16.Encode([]rune(filename)))

	efDict := types.NewPDFDict()
	efDict.Insert("F", indRefStreamDict)
	efDict.Insert("UF", indRefStreamDict)
	d.Insert("EF", efDict)

	d.Insert("Desc", types.PDFStringLiteral("attached by "+types.PDFCPULongVersion))

	return &d, nil
}

// ok returns true if at least one attachment was added.
func addAttachedFiles(ctx *types.PDFContext, files types.StringSet) (ok bool, err error) {

	for filename := range files {

		buf, err := ioutil.ReadFile(filename)
		if err != nil {
			logErrorAttach.Printf("%s: %s\n", filename, err)
			continue
		}

		sd, err := ctx.InsertPDFStreamDict(buf)
		if err != nil {
			return false, err
		}

		err = filter.EncodeStream(sd)
		if err != nil {
			return false, err
		}

		entry := types.NewXRefTableEntryGen0(*sd)
		objNr := ctx.InsertNew(*entry)

		indRefSD := types.NewPDFIndirectRef(objNr, 0)
		dict, err := createFileSpecDict(ctx, filename, indRefSD)
		if err != nil {
			return false, err
		}

		objNr, err = ctx.InsertObject(*dict)
		if err != nil {
			return false, err
		}

		indRefFS := types.NewPDFIndirectRef(objNr, 0)
		//logErrorAttach.Printf("indRefSD:%s indRefFS:%s\n", indRefSD, indRefFS)

		err = ctx.EmbeddedFiles.SetValue(ctx.XRefTable, filename, indRefFS)
		if err != nil {
			return false, err
		}

		ok = true

	}

	return
}

func removeEmbeddedFilesNameTree(ctx *types.PDFContext) (err error) {

	logDebugAttach.Println("removeEmbeddedFilesNameTree begin")

	if ctx.EmbeddedFiles != nil {
		// Remove the object graph of ctx.EmbeddedFiles
		logDebugAttach.Println("removeEmbeddedFilesNameTree: deleting object graph")
		err = ctx.DeleteObjectGraph(ctx.EmbeddedFiles.PDFIndirectRef)
		if err != nil {
			return
		}
		ctx.EmbeddedFiles = nil
	}

	rootDict, err := ctx.DereferenceDict(*ctx.Root)
	if err != nil {
		return err
	}

	obj, found := rootDict.Find("Names")
	if !found {
		return errors.New("removeEmbeddedFilesNameTree: \"Names\" root entry missing")
	}

	namesDict, err := ctx.DereferenceDict(obj)
	if err != nil {
		return err
	}

	if namesDict == nil {
		return errors.New("removeEmbeddedFilesNameTree: \"Names\" root entry corrupt")
	}

	namesDict.Delete("EmbeddedFiles")
	if namesDict.Len() > 0 {
		// Stop here, if EmbeddedFiles was not the only Names entry.
		return
	}

	// Remove NamesDict.
	if namesIndRef := rootDict.IndirectRefEntry("Names"); namesIndRef != nil {
		ctx.DeleteObject(namesIndRef.ObjectNumber.Value())
	}
	rootDict.Delete("Names")

	logDebugAttach.Println("removeEmbeddedFilesNameTree end")

	return
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
					return
				}

				if !found {
					logErrorAttach.Printf("removeAttachedFiles: %s not found\n", fileName)
					continue
				}

				logDebugAttach.Printf("removeAttachedFiles: removed key value pair for %s - deadKid=%t\n", fileName, deadKid)

				ok = true

				if deadKid {

					// Delete name tree root object.
					indRef := ctx.EmbeddedFiles.PDFIndirectRef
					err = ctx.DeleteObject(indRef.ObjectNumber.Value())
					if err != nil {
						return
					}

					// Clean up root.Names entry and delete if EmbeddedFiles was the only Names entry.
					ctx.EmbeddedFiles = nil
					err = removeEmbeddedFilesNameTree(ctx)
					if err != nil {
						return
					}
				}

			} else {

				logErrorAttach.Printf("removeAttachedFiles: no attachments, can't remove %s\n", fileName)

			}

		}

		return
	}

	// If no files specified, remove all embedded files.
	return true, removeEmbeddedFilesNameTree(ctx)
}

// List returns a list of embedded files.
func List(ctx *types.PDFContext) (list []string, err error) {

	logDebugAttach.Println("List begin")

	if !ctx.Valid && ctx.EmbeddedFiles == nil {
		ctx.EmbeddedFiles, err = ctx.LocateNameTree("EmbeddedFiles", false)
		if err != nil {
			return
		}
	}

	if ctx.EmbeddedFiles == nil {
		return
	}

	list, err = listAttachedFiles(ctx)
	if err != nil {
		return
	}

	logDebugAttach.Println("List end")

	return
}

// Extract exports specified embedded files.
// If no files specified extract all embedded files.
func Extract(ctx *types.PDFContext, files types.StringSet) (err error) {

	logDebugAttach.Println("Extract begin")

	if !ctx.Valid && ctx.EmbeddedFiles == nil {
		ctx.EmbeddedFiles, err = ctx.LocateNameTree("EmbeddedFiles", false)
		if err != nil {
			return
		}
	}

	if ctx.EmbeddedFiles == nil {
		return errors.Errorf("no attachments available.")
	}

	err = extractAttachedFiles(ctx, files)
	if err != nil {
		return
	}

	logDebugAttach.Println("Extract end")

	return
}

// Add embeds specified files.
// Existing attachments are replaced.
// ok returns true if at least one attachment was added.
func Add(ctx *types.PDFContext, files types.StringSet) (ok bool, err error) {

	logDebugAttach.Println("Add begin")

	if ctx.EmbeddedFiles == nil {
		ctx.EmbeddedFiles, err = ctx.LocateNameTree("EmbeddedFiles", true)
		if err != nil {
			return
		}
	}

	ok, err = addAttachedFiles(ctx, files)
	if err != nil {
		return
	}

	logDebugAttach.Println("Add end")

	return
}

// Remove deletes specified embedded files.
// ok returns true if at least one attachment could be removed.
func Remove(ctx *types.PDFContext, files types.StringSet) (ok bool, err error) {

	logDebugAttach.Println("Remove begin")

	if !ctx.Valid && ctx.EmbeddedFiles == nil {
		ctx.EmbeddedFiles, err = ctx.LocateNameTree("EmbeddedFiles", false)
		if err != nil {
			return
		}
	}

	if ctx.EmbeddedFiles == nil {
		return false, errors.Errorf("no attachments available.")
	}

	ok, err = removeAttachedFiles(ctx, files)
	if err != nil {
		return
	}

	logDebugAttach.Println("Remove end")

	return
}
