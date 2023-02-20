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

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

const (

	// ObjectStreamMaxObjects limits the number of objects within an object stream written.
	ObjectStreamMaxObjects = 100
)

func writeCommentLine(w *model.WriteContext, comment string) (int, error) {
	return w.WriteString(fmt.Sprintf("%%%s%s", comment, w.Eol))
}

func writeHeader(w *model.WriteContext, v model.Version) error {

	i, err := writeCommentLine(w, "PDF-"+v.String())
	if err != nil {
		return err
	}

	j, err := writeCommentLine(w, "\xe2\xe3\xcf\xD3")
	if err != nil {
		return err
	}

	w.Offset += int64(i + j)

	return nil
}

func writeTrailer(w *model.WriteContext) error {
	_, err := w.WriteString("%%EOF" + w.Eol)
	return err
}

func writeObjectHeader(w *model.WriteContext, objNumber, genNumber int) (int, error) {
	return w.WriteString(fmt.Sprintf("%d %d obj%s", objNumber, genNumber, w.Eol))
}

func writeObjectTrailer(w *model.WriteContext) (int, error) {
	return w.WriteString(fmt.Sprintf("%sendobj%s", w.Eol, w.Eol))
}

func startObjectStream(ctx *model.Context) error {

	// See 7.5.7 Object streams
	// When new object streams and compressed objects are created, they shall always be assigned new object numbers.

	log.Write.Println("startObjectStream begin")

	objStreamDict := types.NewObjectStreamDict()

	objNr, err := ctx.InsertObject(*objStreamDict)
	if err != nil {
		return err
	}

	ctx.Write.CurrentObjStream = &objNr

	log.Write.Printf("startObjectStream end: %d\n", objNr)

	return nil
}

func stopObjectStream(ctx *model.Context) error {

	log.Write.Println("stopObjectStream begin")

	xRefTable := ctx.XRefTable

	if !ctx.Write.WriteToObjectStream {
		return errors.Errorf("stopObjectStream: Not writing to object stream.")
	}

	if ctx.Write.CurrentObjStream == nil {
		ctx.Write.WriteToObjectStream = false
		log.Write.Println("stopObjectStream end (no content)")
		return nil
	}

	entry, _ := xRefTable.FindTableEntry(*ctx.Write.CurrentObjStream, 0)
	osd, _ := (entry.Object).(types.ObjectStreamDict)

	// When we are ready to write: append prolog and content
	osd.Finalize()

	// Encode objStreamDict.Content -> objStreamDict.Raw
	// and wipe (decoded) content to free up memory.
	if err := osd.StreamDict.Encode(); err != nil {
		return err
	}

	// Release memory.
	osd.Content = nil

	osd.StreamDict.Insert("First", types.Integer(osd.FirstObjOffset))
	osd.StreamDict.Insert("N", types.Integer(osd.ObjCount))

	// for each objStream execute at the end right before xRefStreamDict gets written.
	log.Write.Printf("stopObjectStream: objStreamDict: %s\n", osd)

	if err := writeStreamDictObject(ctx, *ctx.Write.CurrentObjStream, 0, osd.StreamDict); err != nil {
		return err
	}

	// Release memory.
	osd.Raw = nil

	ctx.Write.CurrentObjStream = nil
	ctx.Write.WriteToObjectStream = false

	log.Write.Println("stopObjectStream end")

	return nil
}

func writeToObjectStream(ctx *model.Context, objNumber, genNumber int) (ok bool, err error) {

	log.Write.Printf("addToObjectStream begin, obj#:%d gen#:%d\n", objNumber, genNumber)

	w := ctx.Write

	if ctx.WriteXRefStream && // object streams assume an xRefStream to be generated.
		ctx.WriteObjectStream && // signal for compression into object stream is on.
		ctx.Write.WriteToObjectStream && // currently writing to object stream.
		genNumber == 0 {

		if w.CurrentObjStream == nil {
			// Create new objects stream on first write.
			err = startObjectStream(ctx)
			if err != nil {
				return false, err
			}
		}

		objStrEntry, _ := ctx.FindTableEntry(*ctx.Write.CurrentObjStream, 0)
		objStreamDict, _ := (objStrEntry.Object).(types.ObjectStreamDict)

		// Get next free index in object stream.
		i := objStreamDict.ObjCount

		// Locate the xref table entry for the object to be added to this object stream.
		entry, _ := ctx.FindTableEntry(objNumber, genNumber)

		// Turn entry into a compressed entry located in object stream at index i.
		entry.Compressed = true
		entry.ObjectStream = ctx.Write.CurrentObjStream // !
		entry.ObjectStreamInd = &i
		w.SetWriteOffset(objNumber) // for a compressed obj this is supposed to be a fake offset. value does not matter.

		// Append to prolog & content
		s := entry.Object.PDFString()
		err = objStreamDict.AddObject(objNumber, s)
		if err != nil {
			return false, err
		}

		objStrEntry.Object = objStreamDict

		log.Write.Printf("writeObject end, obj#%d written to objectStream #%d\n", objNumber, *ctx.Write.CurrentObjStream)

		if objStreamDict.ObjCount == ObjectStreamMaxObjects {
			err = stopObjectStream(ctx)
			if err != nil {
				return false, err
			}
			w.WriteToObjectStream = true
		}

		ok = true

	}

	log.Write.Printf("addToObjectStream end, obj#:%d gen#:%d\n", objNumber, genNumber)

	return ok, nil
}

func writeObject(ctx *model.Context, objNumber, genNumber int, s string) error {

	log.Write.Printf("writeObject begin, obj#:%d gen#:%d <%s>\n", objNumber, genNumber, s)

	w := ctx.Write

	// Cleanup entry (necessary for split command)
	// TODO This is not the right place to check for an existing obj since we maybe writing NULL.
	entry, ok := ctx.FindTableEntry(objNumber, genNumber)
	if ok {
		entry.Compressed = false
	}

	// Set write-offset for this object.
	w.SetWriteOffset(objNumber)

	written, err := writeObjectHeader(w, objNumber, genNumber)
	if err != nil {
		return err
	}

	// Note: Lines that are not part of stream object data are limited to no more than 255 characters.
	i, err := w.WriteString(s)
	if err != nil {
		return err
	}

	j, err := writeObjectTrailer(w)
	if err != nil {
		return err
	}

	// Write-offset for next object.
	w.Offset += int64(written + i + j)

	log.Write.Printf("writeObject end, %d bytes written\n", written+i+j)

	return nil
}

func writePDFNullObject(ctx *model.Context, objNumber, genNumber int) error {

	return writeObject(ctx, objNumber, genNumber, "null")
}

func writeBooleanObject(ctx *model.Context, objNumber, genNumber int, boolean types.Boolean) error {

	ok, err := writeToObjectStream(ctx, objNumber, genNumber)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	return writeObject(ctx, objNumber, genNumber, boolean.PDFString())
}

func writeNameObject(ctx *model.Context, objNumber, genNumber int, name types.Name) error {

	ok, err := writeToObjectStream(ctx, objNumber, genNumber)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	return writeObject(ctx, objNumber, genNumber, name.PDFString())
}

func writeStringLiteralObject(ctx *model.Context, objNumber, genNumber int, stringLiteral types.StringLiteral) error {

	ok, err := writeToObjectStream(ctx, objNumber, genNumber)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	sl := stringLiteral

	if ctx.EncKey != nil {
		s1, err := encryptString(stringLiteral.Value(), objNumber, genNumber, ctx.EncKey, ctx.AES4Strings, ctx.E.R)
		if err != nil {
			return err
		}

		sl = types.StringLiteral(*s1)
	}

	return writeObject(ctx, objNumber, genNumber, sl.PDFString())
}

func writeHexLiteralObject(ctx *model.Context, objNumber, genNumber int, hexLiteral types.HexLiteral) error {

	ok, err := writeToObjectStream(ctx, objNumber, genNumber)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	hl := hexLiteral

	if ctx.EncKey != nil {
		s1, err := encryptString(hexLiteral.Value(), objNumber, genNumber, ctx.EncKey, ctx.AES4Strings, ctx.E.R)
		if err != nil {
			return err
		}

		hl = types.HexLiteral(*s1)
	}

	return writeObject(ctx, objNumber, genNumber, hl.PDFString())
}

func writeIntegerObject(ctx *model.Context, objNumber, genNumber int, integer types.Integer) error {

	ok, err := writeToObjectStream(ctx, objNumber, genNumber)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	return writeObject(ctx, objNumber, genNumber, integer.PDFString())
}

func writeFloatObject(ctx *model.Context, objNumber, genNumber int, float types.Float) error {

	ok, err := writeToObjectStream(ctx, objNumber, genNumber)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	return writeObject(ctx, objNumber, genNumber, float.PDFString())
}

func writeDictObject(ctx *model.Context, objNumber, genNumber int, d types.Dict) error {

	ok, err := writeToObjectStream(ctx, objNumber, genNumber)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	if ctx.EncKey != nil {
		_, err := encryptDeepObject(d, objNumber, genNumber, ctx.EncKey, ctx.AES4Strings, ctx.E.R)
		if err != nil {
			return err
		}
	}

	return writeObject(ctx, objNumber, genNumber, d.PDFString())
}

func writeArrayObject(ctx *model.Context, objNumber, genNumber int, a types.Array) error {

	ok, err := writeToObjectStream(ctx, objNumber, genNumber)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	if ctx.EncKey != nil {
		_, err := encryptDeepObject(a, objNumber, genNumber, ctx.EncKey, ctx.AES4Strings, ctx.E.R)
		if err != nil {
			return err
		}
	}

	return writeObject(ctx, objNumber, genNumber, a.PDFString())
}

func writeStream(w *model.WriteContext, sd types.StreamDict) (int64, error) {

	b, err := w.WriteString(fmt.Sprintf("%sstream%s", w.Eol, w.Eol))
	if err != nil {
		return 0, errors.Wrapf(err, "writeStream: failed to write raw content")
	}

	c, err := w.Write(sd.Raw)
	if err != nil {
		return 0, errors.Wrapf(err, "writeStream: failed to write raw content")
	}
	if int64(c) != *sd.StreamLength {
		return 0, errors.Errorf("writeStream: failed to write raw content: %d bytes written - streamlength:%d", c, *sd.StreamLength)
	}

	e, err := w.WriteString(fmt.Sprintf("%sendstream", w.Eol))
	if err != nil {
		return 0, errors.Wrapf(err, "writeStream: failed to write raw content")
	}

	written := int64(b+e) + *sd.StreamLength

	return written, nil
}

func handleIndirectLength(ctx *model.Context, ir *types.IndirectRef) error {

	objNr := int(ir.ObjectNumber)
	genNr := int(ir.GenerationNumber)

	if ctx.Write.HasWriteOffset(objNr) {
		log.Write.Printf("*** handleIndirectLength: object #%d already written offset=%d ***\n", objNr, ctx.Write.Offset)
	} else {
		length, err := ctx.DereferenceInteger(*ir)
		if err != nil || length == nil {
			return err
		}
		err = writeIntegerObject(ctx, objNr, genNr, *length)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeStreamDictObject(ctx *model.Context, objNumber, genNumber int, sd types.StreamDict) error {

	log.Write.Printf("writeStreamDictObject begin: object #%d\n%v", objNumber, sd)

	var inObjStream bool

	if ctx.Write.WriteToObjectStream {
		inObjStream = true
		ctx.Write.WriteToObjectStream = false
	}

	// Sometimes a streamDicts length is a reference.
	if ir := sd.IndirectRefEntry("Length"); ir != nil {
		err := handleIndirectLength(ctx, ir)
		if err != nil {
			return err
		}
	}

	var err error

	// Unless the "Identity" crypt filter is used we have to encrypt.
	isXRefStreamDict := sd.Type() != nil && *sd.Type() == "XRef"
	if ctx.EncKey != nil &&
		!isXRefStreamDict &&
		!(len(sd.FilterPipeline) == 1 && sd.FilterPipeline[0].Name == "Crypt") {

		sd.Raw, err = encryptStream(sd.Raw, objNumber, genNumber, ctx.EncKey, ctx.AES4Streams, ctx.E.R)
		if err != nil {
			return err
		}

		l := int64(len(sd.Raw))
		sd.StreamLength = &l
		sd.Update("Length", types.Integer(l))
	}

	ctx.Write.SetWriteOffset(objNumber)

	h, err := writeObjectHeader(ctx.Write, objNumber, genNumber)
	if err != nil {
		return err
	}

	// Note: Lines that are not part of stream object data are limited to no more than 255 characters.
	pdfString := sd.PDFString()
	_, err = ctx.Write.WriteString(pdfString)
	if err != nil {
		return err
	}

	b, err := writeStream(ctx.Write, sd)
	if err != nil {
		return err
	}

	t, err := writeObjectTrailer(ctx.Write)
	if err != nil {
		return err
	}

	written := b + int64(h+len(pdfString)+t)

	ctx.Write.Offset += written
	ctx.Write.BinaryTotalSize += *sd.StreamLength

	if inObjStream {
		ctx.Write.WriteToObjectStream = true
	}

	log.Write.Printf("writeStreamDictObject end: object #%d written=%d\n", objNumber, written)

	return nil
}

func writeDirectObject(ctx *model.Context, o types.Object) error {

	switch o := o.(type) {

	case types.Dict:
		for k, v := range o {
			if ctx.WritingPages && (k == "Dest" || k == "D") {
				ctx.Dest = true
			}
			_, _, err := writeDeepObject(ctx, v)
			if err != nil {
				return err
			}
			ctx.Dest = false
		}
		log.Write.Printf("writeDirectObject: end offset=%d\n", ctx.Write.Offset)

	case types.Array:
		for i, v := range o {
			if ctx.Dest && i == 0 {
				continue
			}
			_, _, err := writeDeepObject(ctx, v)
			if err != nil {
				return err
			}
		}
		log.Write.Printf("writeDirectObject: end offset=%d\n", ctx.Write.Offset)

	default:
		log.Write.Printf("writeDirectObject: end, direct obj - nothing written: offset=%d\n%v\n", ctx.Write.Offset, o)

	}

	return nil
}

func writeNullObject(ctx *model.Context, objNumber, genNumber int) error {

	// An indirect reference to nil is a corner case.
	// Still, it is an object that will be written.
	err := writePDFNullObject(ctx, objNumber, genNumber)
	if err != nil {
		return err
	}

	// Ensure no entry in free list.
	return ctx.UndeleteObject(objNumber)
}

func writeDeepDict(ctx *model.Context, d types.Dict, objNr, genNr int) error {

	err := writeDictObject(ctx, objNr, genNr, d)
	if err != nil {
		return err
	}

	for k, v := range d {
		if ctx.WritingPages && (k == "Dest" || k == "D") {
			ctx.Dest = true
		}
		_, _, err = writeDeepObject(ctx, v)
		if err != nil {
			return err
		}
		ctx.Dest = false
	}

	return nil
}

func writeDeepStreamDict(ctx *model.Context, sd *types.StreamDict, objNr, genNr int) error {

	if ctx.EncKey != nil {
		_, err := encryptDeepObject(*sd, objNr, genNr, ctx.EncKey, ctx.AES4Strings, ctx.E.R)
		if err != nil {
			return err
		}
	}

	err := writeStreamDictObject(ctx, objNr, genNr, *sd)
	if err != nil {
		return err
	}

	for _, v := range sd.Dict {
		_, _, err = writeDeepObject(ctx, v)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeDeepArray(ctx *model.Context, a types.Array, objNr, genNr int) error {

	err := writeArrayObject(ctx, objNr, genNr, a)
	if err != nil {
		return err
	}

	for i, v := range a {
		if ctx.Dest && i == 0 {
			continue
		}
		_, _, err = writeDeepObject(ctx, v)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeIndirectObject(ctx *model.Context, ir types.IndirectRef) (types.Object, error) {

	objNr := int(ir.ObjectNumber)
	genNr := int(ir.GenerationNumber)

	if ctx.Write.HasWriteOffset(objNr) {
		log.Write.Printf("writeIndirectObject end: object #%d already written.\n", objNr)
		return nil, nil
	}

	o, err := ctx.Dereference(ir)
	if err != nil {
		return nil, errors.Wrapf(err, "writeIndirectObject: unable to dereference indirect object #%d", objNr)
	}

	log.Write.Printf("writeIndirectObject: object #%d gets writeoffset: %d\n", objNr, ctx.Write.Offset)

	if o == nil {

		err = writeNullObject(ctx, objNr, genNr)
		if err != nil {
			return nil, err
		}

		log.Write.Printf("writeIndirectObject: end, obj#%d resolved to nil, offset=%d\n", objNr, ctx.Write.Offset)
		return nil, nil
	}

	switch o := o.(type) {

	case types.Dict:
		err = writeDeepDict(ctx, o, objNr, genNr)

	case types.StreamDict:
		err = writeDeepStreamDict(ctx, &o, objNr, genNr)

	case types.Array:
		err = writeDeepArray(ctx, o, objNr, genNr)

	case types.Integer:
		err = writeIntegerObject(ctx, objNr, genNr, o)

	case types.Float:
		err = writeFloatObject(ctx, objNr, genNr, o)

	case types.StringLiteral:
		err = writeStringLiteralObject(ctx, objNr, genNr, o)

	case types.HexLiteral:
		err = writeHexLiteralObject(ctx, objNr, genNr, o)

	case types.Boolean:
		err = writeBooleanObject(ctx, objNr, genNr, o)

	case types.Name:
		err = writeNameObject(ctx, objNr, genNr, o)

	default:
		return nil, errors.Errorf("writeIndirectObject: undefined PDF object #%d %T\n", objNr, o)

	}

	return nil, err
}

func writeDeepObject(ctx *model.Context, objIn types.Object) (objOut types.Object, written bool, err error) {

	log.Write.Printf("writeDeepObject: begin offset=%d\n%s\n", ctx.Write.Offset, objIn)

	ir, ok := objIn.(types.IndirectRef)
	if !ok {
		return objIn, written, writeDirectObject(ctx, objIn)
	}

	objOut, err = writeIndirectObject(ctx, ir)
	if err == nil {
		written = true
		log.Write.Printf("writeDeepObject: end offset=%d\n", ctx.Write.Offset)
	}

	return objOut, written, err
}

func writeEntry(ctx *model.Context, d types.Dict, dictName, entryName string) (types.Object, error) {

	o, found := d.Find(entryName)
	if !found || o == nil {
		log.Write.Printf("writeEntry end: entry %s is nil\n", entryName)
		return nil, nil
	}

	log.Write.Printf("writeEntry begin: dict=%s entry=%s offset=%d\n", dictName, entryName, ctx.Write.Offset)

	o, _, err := writeDeepObject(ctx, o)
	if err != nil {
		return nil, err
	}

	if o == nil {
		log.Write.Printf("writeEntry end: dict=%s entry=%s resolved to nil, offset=%d\n", dictName, entryName, ctx.Write.Offset)
		return nil, nil
	}

	log.Write.Printf("writeEntry end: dict=%s entry=%s offset=%d\n", dictName, entryName, ctx.Write.Offset)

	return o, nil
}

func writeFlatObject(ctx *model.Context, objNr int) error {

	e, ok := ctx.FindTableEntryLight(objNr)
	if !ok {
		return errors.Errorf("writeFlatObject: undefined PDF object #%d ", objNr)
	}

	if e.Free {
		ctx.Write.Table[objNr] = 0
		return nil
	}

	o := e.Object
	genNr := *e.Generation
	var err error

	switch o := o.(type) {

	case types.Dict:
		err = writeDictObject(ctx, objNr, genNr, o)

	case types.StreamDict:
		err = writeDeepStreamDict(ctx, &o, objNr, genNr)

	case types.Array:
		err = writeArrayObject(ctx, objNr, genNr, o)

	case types.Integer:
		err = writeIntegerObject(ctx, objNr, genNr, o)

	case types.Float:
		err = writeFloatObject(ctx, objNr, genNr, o)

	case types.StringLiteral:
		err = writeStringLiteralObject(ctx, objNr, genNr, o)

	case types.HexLiteral:
		err = writeHexLiteralObject(ctx, objNr, genNr, o)

	case types.Boolean:
		err = writeBooleanObject(ctx, objNr, genNr, o)

	case types.Name:
		err = writeNameObject(ctx, objNr, genNr, o)

	default:
		err = errors.Errorf("writeFlatObject: unexpected PDF object #%d %T\n", objNr, o)

	}

	return err
}
