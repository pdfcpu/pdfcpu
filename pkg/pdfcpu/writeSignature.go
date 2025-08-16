/*
Copyright 2024 The pdfcpu Authors.

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
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

func sigDictPDFString(d types.Dict) string {
	s := []string{}
	s = append(s, "<<")
	s = append(s, fmt.Sprintf("/ByteRange%-62v", d["ByteRange"].PDFString()))
	s = append(s, fmt.Sprintf("/Contents%s", d["Contents"].PDFString()))
	s = append(s, fmt.Sprintf("/Type%s", d["Type"].PDFString()))
	s = append(s, fmt.Sprintf("/Filter%s", d["Filter"].PDFString()))
	s = append(s, fmt.Sprintf("/SubFilter%s", d["SubFilter"].PDFString()))
	s = append(s, ">>")
	return strings.Join(s, "")
}

func writeSigDict(ctx *model.Context, ir types.IndirectRef) error {
	// 	<<
	// 		<ByteRange, []>
	// 		<Contents, <00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000>>
	// 		<Filter, Adobe.PPKLite>
	// 		<SubFilter, adbe.pkcs7.detached>
	// 		<Type, Sig>
	// >>

	d, err := ctx.DereferenceDict(ir)
	if err != nil {
		return err
	}

	typ := d.NameEntry("Type")
	if typ == nil || *typ != "Sig" {
		return errors.New("corrupt sig dict")
	}

	f := d.NameEntry("Filter")
	if f == nil || *f != "Adobe.PPKLite" {
		return errors.Errorf("sig dict: unexpected Filter: %s", *f)
	}

	f = d.NameEntry("SubFilter")
	if f == nil || *f != "adbe.pkcs7.detached" {
		return errors.Errorf("sig dict: unexpected SubFilter: %s", *f)
	}

	objNr := ir.ObjectNumber.Value()
	genNr := ir.GenerationNumber.Value()

	// Set write-offset for this object.
	w := ctx.Write
	w.SetWriteOffset(objNr)

	written, err := writeObjectHeader(w, objNr, genNr)
	if err != nil {
		return err
	}

	// <</ByteRange[]
	w.OffsetSigByteRange = w.Offset + int64(written) + 2 + 10
	// 2 for "<<"
	// 10 for "/ByteRange"

	// [...]/Contents<00..... maxSigContentsBytes>
	w.OffsetSigContents = w.OffsetSigByteRange + 1 + 60 + 1 + 9
	// 1 for "["
	// 60 for max 60 chars within this array PDF string.
	// 1 for "]"
	// 9 for "/Contents<"

	i, err := w.WriteString(sigDictPDFString(d))
	if err != nil {
		return err
	}

	j, err := writeObjectTrailer(w)
	if err != nil {
		return err
	}

	// Write-offset for next object.
	w.Offset += int64(written + i + j)

	// Record writeOffset for first and last char of Contents.

	// Record writeOffset for ByteArray...

	return nil
}

func writeSigFieldDict(ctx *model.Context, d types.Dict, objNr, genNr int) error {
	// 	<<
	// 		<DA, (/Courier 0 Tf)>
	// 		<FT, Sig>
	// 		<Rect, [0.00 0.00 0.00 0.00]>
	// 		<Subtype, Widget>
	// 		<T, (Signature)>
	// 		<Type, Annot>
	// 		<V, (21 0 R)>
	// >>

	if err := writeDictObject(ctx, objNr, genNr, d); err != nil {
		return err
	}

	ir := d.IndirectRefEntry("V")
	if ir == nil {
		return errors.New("sig field dict: missing V")
	}

	return writeSigDict(ctx, *ir)
}

func writeBlankSignature(ctx *model.Context, d types.Dict, objNr, genNr int) error {

	// <<
	// 	<DR, <<
	// 		<Font, <<
	// 			<Courier, (19 0 R)>
	// 		>>>
	// 	>>>
	// 	<Fields, [(20 0 R)]>
	// 	<SigFlags, 3>
	// >>

	if err := writeDictObject(ctx, objNr, genNr, d); err != nil {
		return err
	}

	// Write font resource
	resDict := d.DictEntry("DR")
	fontResDict := resDict.DictEntry("Font")
	ir := fontResDict.IndirectRefEntry("Courier")
	if err := writeIndirectObject(ctx, *ir); err != nil {
		return err
	}

	// Write fields
	a := d.ArrayEntry("Fields")
	if a == nil {
		return errors.New("acroform dict: missing Fields")
	}
	for _, o := range a {
		ir, ok := o.(types.IndirectRef)
		if !ok {
			return errors.New("acroform dict fields: expecting indRef")
		}
		d, err := ctx.DereferenceDict(ir)
		if err != nil {
			return err
		}
		ft := d.NameEntry("FT")
		if ft == nil || *ft != "Sig" {
			if err := writeIndirectObject(ctx, ir); err != nil {
				return err
			}
			continue
		}
		objNr := ir.ObjectNumber.Value()
		genNr := ir.GenerationNumber.Value()
		writeSigFieldDict(ctx, d, objNr, genNr)
	}
	return nil
}

func writeAcroFormRootEntry(ctx *model.Context, d types.Dict, dictName string) error {
	o, found := d.Find("AcroForm")
	if !found || o == nil {
		return nil
	}

	if ctx.Cmd != model.ADDSIGNATURE {
		if err := writeRootEntry(ctx, d, dictName, "AcroForm", model.RootAcroForm); err != nil {
			return err
		}
		ctx.Stats.AddRootAttr(model.RootAcroForm)
		return nil
	}

	// TODO distinguish between
	// 		A) PDF is not signed      => write new Acroform with single SigField
	//		B) Acroform is not signed => add Sigfield to existing Acroform
	//      C) PDF is already signed  => add Sigfield to existing Acroform via incremental update

	// Handle A)
	indRef, ok := o.(types.IndirectRef)
	if !ok {
		return errors.New("pdfcpu: add signature: missing Acroform object")
	}

	d1, err := ctx.DereferenceDict(indRef)
	if err != nil {
		return err
	}

	objNr := indRef.ObjectNumber.Value()
	genNr := indRef.GenerationNumber.Value()

	if err := writeBlankSignature(ctx, d1, objNr, genNr); err != nil {
		return err
	}

	ctx.Stats.AddRootAttr(model.RootAcroForm)

	return nil
}
