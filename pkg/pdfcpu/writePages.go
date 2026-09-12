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
	"context"
	"errors"
	"fmt"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// Write page entry to disk.
func writePageEntry(c context.Context, ctx *model.Context, d types.Dict, dictName, entryName string, statsAttr int) error {
	o, err := writeEntry(c, ctx, d, dictName, entryName)
	if err != nil {
		return err
	}

	if o != nil {
		ctx.Stats.AddPageAttr(statsAttr)
	}

	return nil
}

func writePageDict(c context.Context, ctx *model.Context, indRef *types.IndirectRef, pageDict types.Dict, pageNr int) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
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
		return errors.New("missing parent")
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
		if err := contextutil.Check(c); err != nil {
			return err
		}
		if err := writePageEntry(c, ctx, pageDict, dictName, e.entryName, e.statsAttr); err != nil {
			return err
		}
	}

	ctx.WritingPages = false

	if log.WriteEnabled() {
		log.Write.Printf("*** writePageDict end: obj#%d offset=%d ***\n", objNr, ctx.Write.Offset)
	}

	return nil
}

func pageNodeDict(ctx *model.Context, o types.Object) (types.Dict, *types.IndirectRef, *types.Name, error) {
	if o == nil {
		if log.WriteEnabled() {
			log.Write.Println("pageNodeDict: is nil")
		}
		return nil, nil, nil, nil
	}

	// Dereference next page node dict.
	indRef, ok := o.(types.IndirectRef)
	if !ok {
		return nil, nil, nil, errors.New("missing indirect reference")
	}
	if log.WriteEnabled() {
		log.Write.Printf("pageNodeDict: PageNode: %s\n", indRef)
	}

	d, err := ctx.DereferenceDict(indRef)
	if err != nil {
		return nil, nil, nil, errors.New("cannot dereference page node dict")
	}
	if d == nil {
		return nil, nil, nil, errors.New("page node dict is null")
	}

	dictType, _, err := ctx.DereferenceNameEntry(d, "Type")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("page node dict Type: %w", err)
	}
	if dictType == nil {
		return nil, nil, nil, errors.New("missing page node dict type")
	}

	return d, &indRef, dictType, nil
}

func writeSelectedPage(c context.Context, ctx *model.Context, ir *types.IndirectRef, d types.Dict, pageNr int) (bool, error) {
	if err := contextutil.Check(c); err != nil {
		return false, err
	}
	selected := len(ctx.Write.SelectedPages) > 0
	writePage := true
	if selected {
		writePage = ctx.Write.SelectedPages[pageNr]
		if ctx.Cmd == model.REMOVEPAGES {
			writePage = !writePage
		}
	}
	if !writePage {
		if log.WriteEnabled() {
			log.Write.Printf("writeKids: skipping page:%d\n", pageNr)
		}
		return false, nil
	}
	if log.WriteEnabled() {
		message := "writing page anyway"
		if selected {
			message = "writing page"
		}
		log.Write.Printf("writeKids: %s:%d\n", message, pageNr)
	}
	return true, writePageDict(c, ctx, ir, d, pageNr)
}

func writeKids(c context.Context, ctx *model.Context, a types.Array, pageNr *int, depth int, visit *model.PageTreeVisit) (types.Array, int, error) {
	kids := types.Array{}
	count := 0

	for _, o := range a {
		if err := contextutil.Check(c); err != nil {
			return nil, 0, err
		}

		d, ir, dictType, err := pageNodeDict(ctx, o)
		if err != nil {
			return nil, 0, err
		}
		if d == nil {
			continue
		}

		switch dictType.Value() {

		case "Pages":
			// Recurse over pagetree
			skip, writtenPages, err := writePagesDictDepth(c, ctx, ir, pageNr, depth+1, visit)
			if err != nil {
				return nil, 0, err
			}
			if !skip {
				kids = append(kids, o)
				count += writtenPages
			}

		case "Page":
			*pageNr++
			var written bool
			written, err = writeSelectedPage(c, ctx, ir, d, *pageNr)
			if written {
				kids = append(kids, o)
				count++
			}

		default:
			err = fmt.Errorf("unexpected dict type: %s", dictType.Value())

		}

		if err != nil {
			return nil, 0, err
		}

	}

	return kids, count, nil
}

func writePageEntries(c context.Context, ctx *model.Context, d types.Dict, dictName string) error {
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
		if err := contextutil.Check(c); err != nil {
			return err
		}
		if err := writePageEntry(c, ctx, d, dictName, e.entryName, e.statsAttr); err != nil {
			return err
		}
	}

	return nil
}

func writePagesDictDepth(c context.Context, ctx *model.Context, indRef *types.IndirectRef, pageNr *int, depth int, visit *model.PageTreeVisit) (skip bool, writtenPages int, err error) {
	if err := contextutil.Check(c); err != nil {
		return false, 0, err
	}
	if log.WriteEnabled() {
		log.Write.Printf("writePagesDict: begin pageNr=%d\n", *pageNr)
	}

	if err := ctx.XRefTable.CheckRecursionDepth("page tree", depth); err != nil {
		return false, 0, err
	}
	objNr := indRef.ObjectNumber.Value()
	if err := visit.Enter(objNr); err != nil {
		return false, 0, err
	}
	defer visit.Leave(objNr)

	dictName := "pagesDict"
	genNr := int(indRef.GenerationNumber)

	d, err := ctx.DereferenceDict(*indRef)
	if err != nil {
		return false, 0, fmt.Errorf("writePagesDict: unable to dereference indirect object #%d: %w", objNr, err)
	}

	// Push count, kids.
	countOrig, _ := d.Find("Count")
	kidsOrig := d.ArrayEntry("Kids")

	// Iterate over page tree.
	kidsArray := d.ArrayEntry("Kids")
	kidsNew, countNew, err := writeKids(c, ctx, kidsArray, pageNr, depth, visit)
	if err != nil {
		return false, 0, err
	}
	if err := contextutil.Check(c); err != nil {
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

	if err := writePageEntries(c, ctx, d, dictName); err != nil {
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

func writePagesDict(c context.Context, ctx *model.Context, indRef *types.IndirectRef, pageNr *int) (skip bool, writtenPages int, err error) {
	return writePagesDictDepth(c, ctx, indRef, pageNr, 0, model.NewPageTreeVisit())
}
