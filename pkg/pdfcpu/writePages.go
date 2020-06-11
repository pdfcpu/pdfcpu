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
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pkg/errors"
)

// Write page entry to disk.
func writePageEntry(ctx *Context, d Dict, dictName, entryName string, statsAttr int) error {

	o, err := writeEntry(ctx, d, dictName, entryName)
	if err != nil {
		return err
	}

	if o != nil {
		ctx.Stats.AddPageAttr(statsAttr)
	}

	return nil
}

func writePageDict(ctx *Context, ir *IndirectRef, pageDict Dict, pageNr int) error {

	objNr := ir.ObjectNumber.Value()
	genNr := ir.GenerationNumber.Value()

	if ctx.Write.HasWriteOffset(objNr) {
		log.Write.Printf("writePageDict: object #%d already written.\n", objNr)
		return nil
	}

	log.Write.Printf("writePageDict: logical pageNr=%d object #%d gets writeoffset: %d\n", pageNr, objNr, ctx.Write.Offset)

	dictName := "pageDict"

	if err := writeDictObject(ctx, objNr, genNr, pageDict); err != nil {
		return err
	}

	log.Write.Printf("writePageDict: new offset = %d\n", ctx.Write.Offset)

	if ir := pageDict.IndirectRefEntry("Parent"); ir == nil {
		return errors.New("pdfcpu: writePageDict: missing parent")
	}

	ctx.writingPages = true

	for _, e := range []struct {
		entryName string
		statsAttr int
	}{
		{"Contents", PageContents},
		{"Resources", PageResources},
		{"MediaBox", PageMediaBox},
		{"CropBox", PageCropBox},
		{"BleedBox", PageBleedBox},
		{"TrimBox", PageTrimBox},
		{"ArtBox", PageArtBox},
		{"BoxColorInfo", PageBoxColorInfo},
		{"PieceInfo", PagePieceInfo},
		{"LastModified", PageLastModified},
		{"Rotate", PageRotate},
		{"Group", PageGroup},
		{"Annots", PageAnnots},
		{"Thumb", PageThumb},
		{"B", PageB},
		{"Dur", PageDur},
		{"Trans", PageTrans},
		{"AA", PageAA},
		{"Metadata", PageMetadata},
		{"StructParents", PageStructParents},
		{"ID", PageID},
		{"PZ", PagePZ},
		{"SeparationInfo", PageSeparationInfo},
		{"Tabs", PageTabs},
		{"TemplateInstantiated", PageTemplateInstantiated},
		{"PresSteps", PagePresSteps},
		{"UserUnit", PageUserUnit},
		{"VP", PageVP},
	} {
		if err := writePageEntry(ctx, pageDict, dictName, e.entryName, e.statsAttr); err != nil {
			return err
		}
	}

	ctx.writingPages = false

	log.Write.Printf("*** writePageDict end: obj#%d offset=%d ***\n", objNr, ctx.Write.Offset)

	return nil
}

func pageNodeDict(ctx *Context, o Object) (d Dict, indRef *IndirectRef, err error) {

	if o == nil {
		log.Write.Println("pageNodeDict: is nil")
		return nil, nil, nil
	}

	// Dereference next page node dict.
	ir, ok := o.(IndirectRef)
	if !ok {
		return nil, nil, errors.New("pdfcpu: pageNodeDict: missing indirect reference")
	}
	log.Write.Printf("pageNodeDict: PageNode: %s\n", ir)

	d, err = ctx.DereferenceDict(ir)
	if err != nil {
		return nil, nil, errors.New("pdfcpu: pageNodeDict: cannot dereference, pageNodeDict")
	}
	if d == nil {
		return nil, nil, errors.New("pdfcpu: pageNodeDict: pageNodeDict is null")
	}

	dictType := d.Type()
	if dictType == nil {
		return nil, nil, errors.New("pdfcpu: pageNodeDict: missing pageNodeDict type")
	}

	return d, &ir, nil
}

func writeKids(ctx *Context, a Array, pageNr *int) (Array, int, error) {

	kids := Array{}
	count := 0

	for _, o := range a {

		d, ir, err := pageNodeDict(ctx, o)
		if err != nil {
			return nil, 0, err
		}
		if d == nil {
			continue
		}

		switch *d.Type() {

		case "Pages":
			// Recurse over pagetree
			skip, c, err := writePagesDict(ctx, ir, pageNr)
			if err != nil {
				return nil, 0, err
			}
			if !skip {
				kids = append(kids, o)
				count += c
			}

		case "Page":
			*pageNr++
			if len(ctx.Write.SelectedPages) > 0 {
				log.Write.Printf("selectedPages: %v\n", ctx.Write.SelectedPages)
				writePage := ctx.Write.SelectedPages[*pageNr]
				if ctx.Cmd == REMOVEPAGES {
					writePage = !writePage
				}
				if writePage {
					log.Write.Printf("writeKids: writing page:%d\n", *pageNr)
					err = writePageDict(ctx, ir, d, *pageNr)
					kids = append(kids, o)
					count++
				} else {
					log.Write.Printf("writeKids: skipping page:%d\n", *pageNr)
				}
			} else {
				log.Write.Printf("writeKids: writing page anyway:%d\n", *pageNr)
				err = writePageDict(ctx, ir, d, *pageNr)
				kids = append(kids, o)
				count++
			}

		default:
			err = errors.Errorf("pdfcpu: writeKids: Unexpected dict type: %s", *d.Type())

		}

		if err != nil {
			return nil, 0, err
		}

	}

	return kids, count, nil
}

func containsSelectedPages(ctx *Context, from, thru int) bool {
	for i := from; i <= thru; i++ {
		if ctx.Write.SelectedPages[i] {
			return true
		}
	}
	return false
}

func writePagesDict(ctx *Context, ir *IndirectRef, pageNr *int) (skip bool, writtenPages int, err error) {

	log.Write.Printf("writePagesDict: begin pageNr=%d\n", *pageNr)

	dictName := "pagesDict"
	objNr := int(ir.ObjectNumber)
	genNr := int(ir.GenerationNumber)

	d, err := ctx.DereferenceDict(*ir)
	if err != nil {
		return false, 0, errors.Wrapf(err, "writePagesDict: unable to dereference indirect object #%d", objNr)
	}

	// Push count, kids.
	countOrig, _ := d.Find("Count")
	kidsOrig := d.ArrayEntry("Kids")

	// TRIM, REMOVEPAGES are the only commands where we modify the page tree during writing.
	// In these cases the selected pages to be written or to be removed are defined in ctx.Write.SelectedPages.
	if len(ctx.Write.SelectedPages) > 0 {
		c := int(countOrig.(Integer))
		log.Write.Printf("writePagesDict: checking page range %d - %d \n", *pageNr+1, *pageNr+c)
		if ctx.Cmd == REMOVEPAGES ||
			((ctx.Cmd == TRIM) && containsSelectedPages(ctx, *pageNr+1, *pageNr+c)) {
			log.Write.Println("writePagesDict: process this subtree")
		} else {
			log.Write.Println("writePagesDict: skip this subtree")
			*pageNr += c
			return true, 0, nil
		}
	}

	// Iterate over page tree.
	kidsArray := d.ArrayEntry("Kids")
	kidsNew, countNew, err := writeKids(ctx, kidsArray, pageNr)
	if err != nil {
		return false, 0, err
	}

	d.Update("Kids", kidsNew)
	d.Update("Count", Integer(countNew))
	log.Write.Printf("writePagesDict: writing pageDict for obj=%d page=%d\n%s", objNr, *pageNr, d)

	if err = writeDictObject(ctx, objNr, genNr, d); err != nil {
		return false, 0, err
	}

	// TODO Check inheritance rules.
	for _, e := range []struct {
		entryName string
		statsAttr int
	}{
		{"Resources", PageResources},
		{"MediaBox", PageMediaBox},
		{"CropBox", PageCropBox},
		{"Rotate", PageRotate},
	} {
		if err = writePageEntry(ctx, d, dictName, e.entryName, e.statsAttr); err != nil {
			return false, 0, err
		}
	}

	// Pop kids, count.
	d.Update("Kids", kidsOrig)
	d.Update("Count", countOrig)

	log.Write.Printf("writePagesDict: end pageNr=%d\n", *pageNr)

	return false, countNew, nil
}
