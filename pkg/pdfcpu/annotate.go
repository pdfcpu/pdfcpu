/*
Copyright 2021 The pdfcpu Authors.

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
	"time"

	"github.com/pkg/errors"
)

type AnnotationRenderer interface {
	RenderDict(pageIndRef IndirectRef) Dict
	ID() string
}

type Annotation struct {
	SubType  string       // The type of annotation that this dictionary describes.
	Rect     Rectangle    // The annotation rectangle, defining the location of the annotation on the page in default user space units.
	Contents string       // Text that shall be displayed for the annotation.
	P        *IndirectRef // An indirect reference to the page object with which this annotation is associated.
	NM       string       // (Since V1.4) The annotation name, a text string uniquely identifying it among all the annotations on its page.
	ModDate  string       // The date and time when the annotation was most recently modified.
	F        int          // A set of flags specifying various characteristics of the annotation.
	C        *SimpleColor // The background color of the annotation’s icon when closed.
}

func NewAnnotation(
	subType string,
	rect Rectangle,
	contents string,
	pageIndRef *IndirectRef,
	nm string,
	f int,
	backgrCol *SimpleColor) Annotation {

	return Annotation{
		SubType:  subType,
		Rect:     rect,
		Contents: contents,
		P:        pageIndRef,
		NM:       nm,
		F:        f,
		C:        backgrCol}
}

func (ann Annotation) ID() string {
	return ann.NM
}

type MarkupAnnotation struct {
	Annotation
	T            string       // The text label that shall be displayed in the title bar of the annotation’s pop-up window when open and active. This entry shall identify the user who added the annotation.
	PopupIndRef  *IndirectRef // An indirect reference to a pop-up annotation for entering or editing the text associated with this annotation.
	CA           *float64     // (Default: 1.0) The constant opacity value that shall be used in painting the annotation.
	RC           string       // A rich text string that shall be displayed in the pop-up window when the annotation is opened.
	CreationDate string       // The date and time when the annotation was created.
	Subj         string       // Text representing a short description of the subject being addressed by the annotation.
}

func NewMarkupAnnotation(
	subType string,
	rect Rectangle,
	pageIndRef *IndirectRef,
	contents, id, title string,
	f int,
	backgrCol *SimpleColor,
	popupIndRef *IndirectRef,
	ca *float64,
	rc, subject string) MarkupAnnotation {

	ann := NewAnnotation(subType, rect, contents, pageIndRef, id, f, backgrCol)

	return MarkupAnnotation{
		Annotation:   ann,
		T:            title,
		PopupIndRef:  popupIndRef,
		CreationDate: DateString(time.Now()),
		CA:           ca,
		RC:           rc,
		Subj:         subject}
}

// Sticky Note
type TextAnnotation struct {
	MarkupAnnotation        // SubType = Text
	Open             bool   // A flag specifying whether the annotation shall initially be displayed open.
	Name             string // The name of an icon that shall be used in displaying the annotation. Comment, Key, (Note), Help, NewParagraph, Paragraph, Insert
}

func NewTextAnnotation(
	rect Rectangle,
	contents, id, title string,
	f int,
	backgrCol *SimpleColor,
	ca *float64,
	rc, subj string,
	open bool,
	name string) TextAnnotation {

	ma := NewMarkupAnnotation("Text", rect, nil, contents, id, title, f, backgrCol, nil, ca, rc, subj)

	return TextAnnotation{
		MarkupAnnotation: ma,
		Open:             open,
		Name:             name,
	}
}

func (ann TextAnnotation) RenderDict(pageIndRef IndirectRef) Dict {
	subject := "Sticky Note"
	if ann.Subj != "" {
		subject = ann.Subj
	}
	d := Dict(map[string]Object{
		"Type":         Name("Annot"),
		"Subtype":      Name(ann.SubType),
		"Rect":         ann.Rect.Array(),
		"P":            pageIndRef,
		"F":            Integer(ann.F), // 24 = NoRotate + NoZoom
		"CreationDate": StringLiteral(ann.CreationDate),
		"Subj":         StringLiteral(subject),
		"Open":         Boolean(ann.Open),
	})
	if ann.CA != nil {
		d.Insert("CA", Float(*ann.CA))
	}
	if ann.PopupIndRef != nil {
		d.Insert("Popup", *ann.PopupIndRef)
	}
	if ann.RC != "" {
		d.InsertString("RC", ann.RC)
	}
	if ann.Name != "" {
		d.InsertName("Name", ann.Name)
	}
	if ann.Contents != "" {
		d.InsertString("Contents", ann.Contents)
	}
	if ann.NM != "" {
		d.InsertString("NM", ann.NM) // check for uniqueness across annotations on this page
	} else {
		// new UUID
	}
	if ann.T != "" {
		d.InsertString("T", ann.T)
	}
	if ann.C != nil {
		d.Insert("C", NewNumberArray(float64(ann.C.R), float64(ann.C.G), float64(ann.C.B)))
	}
	return d
}

type LinkAnnotation struct {
	Annotation // SubType = Link
	URI        string
}

func NewLinkAnnotation(
	rect Rectangle,
	uri, id string,
	f int,
	backgrCol *SimpleColor) LinkAnnotation {

	ann := NewAnnotation("Link", rect, "", nil, id, f, backgrCol)

	return LinkAnnotation{
		Annotation: ann,
		URI:        uri,
	}
}

func (ann LinkAnnotation) RenderDict(pageIndRef IndirectRef) Dict {

	// <A, <<
	// 	<S, URI>
	// 	<Type, Action>
	// 	<URI, (http://www.acme.org)>
	// 	>>
	// >

	actionDict := Dict(map[string]Object{
		"Type": Name("Action"),
		"S":    Name("URI"),
		"URI":  StringLiteral(ann.URI),
	})

	d := Dict(map[string]Object{
		"Type":    Name("Annot"),
		"Subtype": Name(ann.SubType),
		"Rect":    ann.Rect.Array(),
		"P":       pageIndRef,
		"F":       Integer(24), // = NoRotate + NoZoom
		"Border":  NewIntegerArray(0, 0, 1),
		"H":       Name("I"), // = Default
		"A":       actionDict,
	})

	if ann.NM != "" {
		d.InsertString("NM", ann.NM) // check for uniqueness across annotations on this page
	} else {
		// new UUID
	}
	if ann.C != nil {
		d.Insert("C", NewNumberArray(float64(ann.C.R), float64(ann.C.G), float64(ann.C.B)))
	}
	return d
}

func (ctx *Context) findAnnot(id string, annots Array) (int, error) {
	for i, o := range annots {
		d, err := ctx.DereferenceDict(o)
		if err != nil {
			return -1, err
		}
		s := d.StringEntry("NM")
		if s == nil {
			continue
		}
		if *s == id {
			return i, nil
		}
	}
	return -1, nil
}

func (ctx *Context) createAnnot(ar AnnotationRenderer, pageIndRef *IndirectRef) (*IndirectRef, error) {
	d := ar.RenderDict(*pageIndRef)
	return ctx.IndRefForNewObject(d)
}

func (ctx *Context) AddAnnotations(selectedPages IntSet, ar AnnotationRenderer, incr bool) error {
	if incr {
		ctx.Write.Increment = true
		ctx.Write.Offset = ctx.Read.FileSize
	}
	for k, v := range selectedPages {
		if !v {
			continue
		}
		if k > ctx.PageCount {
			return errors.Errorf("pdfcpu: invalid page number: %d", k)
		}

		pageDictIndRef, err := ctx.PageDictIndRef(k)
		if err != nil {
			return err
		}

		d, err := ctx.DereferenceDict(*pageDictIndRef)
		if err != nil {
			return err
		}

		annotIndRef, err := ctx.createAnnot(ar, pageDictIndRef)
		if err != nil {
			return err
		}
		if incr {
			// Mark new annotaton dict obj for incremental writing.
			ctx.Write.IncrementWithObjNr(annotIndRef.ObjectNumber.Value())
		}

		obj, found := d.Find("Annots")
		if !found {
			d.Insert("Annots", Array{*annotIndRef})
			// Alternatively create a separate array object:
			// ir, err := ctx.IndRefForNewObject(Array{*annotIndRef})
			// if err != nil {
			// 	return err
			// }
			// if incr {
			// 	// Mark new Annot array obj for incremental writing.
			// 	ctx.Write.IncrementWithObjNr(ir.ObjectNumber.Value())
			// 	// Mark page dict obj for incremental writing.
			// 	ctx.Write.IncrementWithObjNr(pageDictIndRef.ObjectNumber.Value())
			// }
			// d.Insert("Annots", *ir)
			continue
		}

		ir, ok := obj.(IndirectRef)
		if !ok {
			annots, _ := obj.(Array)
			i, err := ctx.findAnnot(ar.ID(), annots)
			if err != nil {
				return err
			}
			if i >= 0 {
				return errors.Errorf("page %d: duplicate annotation with id:%s\n", k, ar.ID())
			}
			d.Update("Annots", append(annots, *annotIndRef))
			if incr {
				// Mark page dict obj for incremental writing.
				ctx.Write.IncrementWithObjNr(pageDictIndRef.ObjectNumber.Value())
			}
			continue
		}

		// Annots array is an IndirectReference.

		o, err := ctx.Dereference(ir)
		if err != nil || o == nil {
			return err
		}

		annots, _ := o.(Array)
		i, err := ctx.findAnnot(ar.ID(), annots)
		if err != nil {
			return err
		}
		if i >= 0 {
			return errors.Errorf("page %d: duplicate annotation with id:%s\n", k, ar.ID())
		}

		entry, ok := ctx.FindTableEntryForIndRef(&ir)
		if !ok {
			return errors.Errorf("page %d: can't dereference Annots indirect reference(obj#:%d)\n", k, ir.ObjectNumber)
		}
		entry.Object = append(annots, *annotIndRef)
		if incr {
			// Mark Annot array obj for incremental writing.
			ctx.Write.IncrementWithObjNr(ir.ObjectNumber.Value())
		}
	}

	//ctx.EnsureVersionForWriting()
	return nil
}

func (ctx *Context) RemoveAnnotations(selectedPages IntSet, id string, incr bool) (bool, error) {
	// Based on the assumptions that annotations don't use indRefs other than for P.
	// TODO: Handle popUpIndRef for markup annotations.
	ok := false
	if incr {
		ctx.Write.Increment = true
		ctx.Write.Offset = ctx.Read.FileSize
	}
	for k, v := range selectedPages {
		if !v {
			continue
		}
		if k > ctx.PageCount {
			return false, errors.Errorf("pdfcpu: invalid page number: %d", k)
		}

		pageDictIndRef, err := ctx.PageDictIndRef(k)
		if err != nil {
			return false, err
		}

		d, err := ctx.DereferenceDict(*pageDictIndRef)
		if err != nil {
			return false, err
		}

		obj, found := d.Find("Annots")
		if !found {
			continue
		}

		ir, ok1 := obj.(IndirectRef)
		if !ok1 {
			annots, _ := obj.(Array)
			i, err := ctx.findAnnot(id, annots)
			if err != nil {
				return false, err
			}
			if i < 0 {
				// annotation not found.
				continue
			}
			if incr {
				// Mark page dict obj for incremental writing.
				ctx.Write.IncrementWithObjNr(pageDictIndRef.ObjectNumber.Value())
				// Mark annotation dict as free.
				ir, _ := annots[i].(IndirectRef)
				if err = ctx.markAsFree(ir); err != nil {
					return false, err
				}
				// Mark annotation dict obj for incremental writing.
				ctx.Write.IncrementWithObjNr(ir.ObjectNumber.Value())
			}
			// Remove annotation indRef from Annots array.
			// TODO (arr Array) func Delete(i int)
			if len(annots) == 1 {
				d.Delete("Annots")
				ok = true
				continue
			}
			d.Update("Annots", append(annots[:i], annots[i+1:]...))
			ok = true
			continue
		}

		// Annots array is an IndirectReference.
		o, err := ctx.Dereference(ir)
		if err != nil || o == nil {
			return false, err
		}
		annots, _ := o.(Array)
		i, err := ctx.findAnnot(id, annots)
		if err != nil {
			return false, err
		}
		if i < 0 {
			// annotation not found.
			continue
		}
		objNr := ir.ObjectNumber.Value()
		genNr := ir.GenerationNumber.Value()
		if incr {
			// Mark annotation dict as free.
			ir, _ := annots[i].(IndirectRef)
			if err = ctx.markAsFree(ir); err != nil {
				return false, err
			}
			// Mark annotation dict obj for incremental writing.
			ctx.Write.IncrementWithObjNr(ir.ObjectNumber.Value())
			// Modify Annots array obj for incremental writing.
			ctx.Write.IncrementWithObjNr(objNr)
		}
		entry, _ := ctx.FindTableEntry(objNr, genNr)
		// Remove annotation indRef from Annots array.
		if len(annots) == 1 {
			d.Delete("Annots")
			if incr {
				// Mark Annots array as free.
				if err = ctx.markAsFree(ir); err != nil {
					return false, err
				}
				// Mark page dict obj for incremental writing.
				ctx.Write.IncrementWithObjNr(pageDictIndRef.ObjectNumber.Value())
			}
			ok = true
			continue
		}
		entry.Object = append(annots[:i], annots[i+1:]...)
		ok = true
	}

	//ctx.EnsureVersionForWriting()
	return ok, nil
}
