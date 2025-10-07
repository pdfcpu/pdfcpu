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
	"github.com/angel-one/pdfcpu/pkg/log"
	"github.com/angel-one/pdfcpu/pkg/pdfcpu/model"
	"github.com/angel-one/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// Write page entry to disk.
func writePageEntry(ctx *model.Context, d types.Dict, dictName, entryName string, statsAttr int) error {
	o, err := writeEntry(ctx, d, dictName, entryName)
	if err != nil {
		return err
	}

	if o != nil {
		ctx.Stats.AddPageAttr(statsAttr)
	}

	return nil
}

func writePageDict(ctx *model.Context, indRef *types.IndirectRef, pageDict types.Dict, pageNr int) error {
	objNr := indRef.ObjectNumber.Value()
	genNr := indRef.GenerationNumber.Value()

	if ctx.Write.HasWriteOffset(objNr) {
		if log.WriteEnabled() {
			log.Write.Printf("writePageDict: object #%d already written.\n", objNr)
		}
		return nil
	}

	if log.WriteEnabled() {
		log.Write.Printf("writePageDict: logical pageNr=%d object #%d gets writeoffset: %d\n", pageNr, objNr, ctx.Write.Offset)
	}

	dictName := "pageDict"

	if err := writeDictObject(ctx, objNr, genNr, pageDict); err != nil {
		return err
	}

	if log.WriteEnabled() {
		log.Write.Printf("writePageDict: new offset = %d\n", ctx.Write.Offset)
	}

	if indRef := pageDict.IndirectRefEntry("Parent"); indRef == nil {
		return errors.New("pdfcpu: writePageDict: missing parent")
	}

	ctx.WritingPages = true

	for _, e := range []struct {
		entryName string
		statsAttr int
	}{
		{"Contents", model.PageContents},
		{"Resources", model.PageResources},
		{"MediaBox", model.PageMediaBox},
		{"CropBox", model.PageCropBox},
		{"BleedBox", model.PageBleedBox},
		{"TrimBox", model.PageTrimBox},
		{"ArtBox", model.PageArtBox},
		{"BoxColorInfo", model.PageBoxColorInfo},
		{"PieceInfo", model.PagePieceInfo},
		{"LastModified", model.PageLastModified},
		{"Rotate", model.PageRotate},
		{"Group", model.PageGroup},
		{"Annots", model.PageAnnots},
		{"Thumb", model.PageThumb},
		{"B", model.PageB},
		{"Dur", model.PageDur},
		{"Trans", model.PageTrans},
		{"AA", model.PageAA},
		{"Metadata", model.PageMetadata},
		{"StructParents", model.PageStructParents},
		{"ID", model.PageID},
		{"PZ", model.PagePZ},
		{"SeparationInfo", model.PageSeparationInfo},
		{"Tabs", model.PageTabs},
		{"TemplateInstantiated", model.PageTemplateInstantiated},
		{"PresSteps", model.PagePresSteps},
		{"UserUnit", model.PageUserUnit},
		{"VP", model.PageVP},
	} {
		if err := writePageEntry(ctx, pageDict, dictName, e.entryName, e.statsAttr); err != nil {
			return err
		}
	}

	ctx.WritingPages = false

	if log.WriteEnabled() {
		log.Write.Printf("*** writePageDict end: obj#%d offset=%d ***\n", objNr, ctx.Write.Offset)
	}

	return nil
}

func pageNodeDict(ctx *model.Context, o types.Object) (types.Dict, *types.IndirectRef, error) {
	if o == nil {
		if log.WriteEnabled() {
			log.Write.Println("pageNodeDict: is nil")
		}
		return nil, nil, nil
	}

	// Dereference next page node dict.
	indRef, ok := o.(types.IndirectRef)
	if !ok {
		return nil, nil, errors.New("pdfcpu: pageNodeDict: missing indirect reference")
	}
	if log.WriteEnabled() {
		log.Write.Printf("pageNodeDict: PageNode: %s\n", indRef)
	}

	d, err := ctx.DereferenceDict(indRef)
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

	return d, &indRef, nil
}

func writeKids(ctx *model.Context, a types.Array, pageNr *int) (types.Array, int, error) {
	kids := types.Array{}
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
				// if log.WriteEnabled() {
				// 	log.Write.Printf("selectedPages: %v\n", ctx.Write.SelectedPages)
				// }
				writePage := ctx.Write.SelectedPages[*pageNr]
				if ctx.Cmd == model.REMOVEPAGES {
					writePage = !writePage
				}
				if writePage {
					if log.WriteEnabled() {
						log.Write.Printf("writeKids: writing page:%d\n", *pageNr)
					}
					err = writePageDict(ctx, ir, d, *pageNr)
					kids = append(kids, o)
					count++
				} else {
					if log.WriteEnabled() {
						log.Write.Printf("writeKids: skipping page:%d\n", *pageNr)
					}
				}
			} else {
				if log.WriteEnabled() {
					log.Write.Printf("writeKids: writing page anyway:%d\n", *pageNr)
				}
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

func writePageEntries(ctx *model.Context, d types.Dict, dictName string) error {
	// TODO Check inheritance rules.
	for _, e := range []struct {
		entryName string
		statsAttr int
	}{
		{"Resources", model.PageResources},
		{"MediaBox", model.PageMediaBox},
		{"CropBox", model.PageCropBox},
		{"Rotate", model.PageRotate},
	} {
		if err := writePageEntry(ctx, d, dictName, e.entryName, e.statsAttr); err != nil {
			return err
		}
	}

	return nil
}

func writePagesDict(ctx *model.Context, indRef *types.IndirectRef, pageNr *int) (skip bool, writtenPages int, err error) {
	if log.WriteEnabled() {
		log.Write.Printf("writePagesDict: begin pageNr=%d\n", *pageNr)
	}

	dictName := "pagesDict"
	objNr := int(indRef.ObjectNumber)
	genNr := int(indRef.GenerationNumber)

	d, err := ctx.DereferenceDict(*indRef)
	if err != nil {
		return false, 0, errors.Wrapf(err, "writePagesDict: unable to dereference indirect object #%d", objNr)
	}

	// Push count, kids.
	countOrig, _ := d.Find("Count")
	kidsOrig := d.ArrayEntry("Kids")

	// Iterate over page tree.
	kidsArray := d.ArrayEntry("Kids")
	kidsNew, countNew, err := writeKids(ctx, kidsArray, pageNr)
	if err != nil {
		return false, 0, err
	}

	d.Update("Kids", kidsNew)
	d.Update("Count", types.Integer(countNew))
	if log.WriteEnabled() {
		log.Write.Printf("writePagesDict: writing pageDict for obj=%d page=%d\n%s", objNr, *pageNr, d)
	}

	if err = writeDictObject(ctx, objNr, genNr, d); err != nil {
		return false, 0, err
	}

	if err := writePageEntries(ctx, d, dictName); err != nil {
		return false, 0, err
	}

	// Pop kids, count.
	d.Update("Kids", kidsOrig)
	d.Update("Count", countOrig)

	if log.WriteEnabled() {
		log.Write.Printf("writePagesDict: end pageNr=%d\n", *pageNr)
	}

	return false, countNew, nil
}
