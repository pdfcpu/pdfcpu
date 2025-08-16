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

	if log.WriteEnabled() {
		log.Write.Println("startObjectStream begin")
	}

	objStreamDict := types.NewObjectStreamDict()

	objNr, err := ctx.InsertObject(*objStreamDict)
	if err != nil {
		return err
	}

	ctx.Write.CurrentObjStream = &objNr

	if log.WriteEnabled() {
		log.Write.Printf("startObjectStream end: %d\n", objNr)
	}

	return nil
}

func stopObjectStream(ctx *model.Context) error {
	if log.WriteEnabled() {
		log.Write.Println("stopObjectStream begin")
	}

	xRefTable := ctx.XRefTable

	if !ctx.Write.WriteToObjectStream {
		return errors.Errorf("stopObjectStream: Not writing to object stream.")
	}

	if ctx.Write.CurrentObjStream == nil {
		ctx.Write.WriteToObjectStream = false
		if log.WriteEnabled() {
			log.Write.Println("stopObjectStream end (no content)")
		}
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
	if log.WriteEnabled() {
		log.Write.Printf("stopObjectStream: objStreamDict: %s\n", osd)
	}

	if err := writeStreamDictObject(ctx, *ctx.Write.CurrentObjStream, 0, osd.StreamDict); err != nil {
		return err
	}

	// Release memory.
	osd.Raw = nil

	ctx.Write.CurrentObjStream = nil
	ctx.Write.WriteToObjectStream = false

	if log.WriteEnabled() {
		log.Write.Println("stopObjectStream end")
	}

	return nil
}

func writeToObjectStream(ctx *model.Context, objNumber, genNumber int) (ok bool, err error) {
	if log.WriteEnabled() {
		log.Write.Printf("addToObjectStream begin, obj#:%d gen#:%d\n", objNumber, genNumber)
	}

	w := ctx.Write

	if ctx.WriteXRefStream && // object streams assume an xRefStream to be generated.
		ctx.WriteObjectStream && // signal for compression into object stream is on.
		ctx.Write.WriteToObjectStream && // currently writing to object stream.
		genNumber == 0 {

		if w.CurrentObjStream == nil {
			// Create new objects stream on first write.
			if err = startObjectStream(ctx); err != nil {
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
		if err = objStreamDict.AddObject(objNumber, s); err != nil {
			return false, err
		}

		objStrEntry.Object = objStreamDict

		if log.WriteEnabled() {
			log.Write.Printf("writeObject end, obj#%d written to objectStream #%d\n", objNumber, *ctx.Write.CurrentObjStream)
		}

		if objStreamDict.ObjCount == ObjectStreamMaxObjects {
			if err = stopObjectStream(ctx); err != nil {
				return false, err
			}
			w.WriteToObjectStream = true
		}

		ok = true

	}

	if log.WriteEnabled() {
		log.Write.Printf("addToObjectStream end, obj#:%d gen#:%d\n", objNumber, genNumber)
	}

	return ok, nil
}

func writeObject(ctx *model.Context, objNumber, genNumber int, s string) error {
	if log.WriteEnabled() {
		log.Write.Printf("writeObject begin, obj#:%d gen#:%d <%s>\n", objNumber, genNumber, s)
	}

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

	if log.WriteEnabled() {
		log.Write.Printf("writeObject end, %d bytes written\n", written+i+j)
	}

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

func writeStringLiteralObject(ctx *model.Context, objNumber, genNumber int, sl types.StringLiteral) error {
	ok, err := writeToObjectStream(ctx, objNumber, genNumber)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	if ctx.EncKey != nil {
		sl1, err := encryptStringLiteral(sl, objNumber, genNumber, ctx.EncKey, ctx.AES4Strings, ctx.E.R)
		if err != nil {
			return err
		}

		sl = *sl1
	}

	return writeObject(ctx, objNumber, genNumber, sl.PDFString())
}

func writeHexLiteralObject(ctx *model.Context, objNumber, genNumber int, hl types.HexLiteral) error {
	ok, err := writeToObjectStream(ctx, objNumber, genNumber)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	if ctx.EncKey != nil {
		hl1, err := encryptHexLiteral(hl, objNumber, genNumber, ctx.EncKey, ctx.AES4Strings, ctx.E.R)
		if err != nil {
			return err
		}

		hl = *hl1
	}

	return writeObject(ctx, objNumber, genNumber, hl.PDFString())
}

func writeIntegerObject(ctx *model.Context, objNumber, genNumber int, integer types.Integer) error {
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
		if _, err := encryptDeepObject(a, objNumber, genNumber, ctx.EncKey, ctx.AES4Strings, ctx.E.R); err != nil {
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
		if log.WriteEnabled() {
			log.Write.Printf("*** handleIndirectLength: object #%d already written offset=%d ***\n", objNr, ctx.Write.Offset)
		}
	} else {
		length, err := ctx.DereferenceInteger(*ir)
		if err != nil || length == nil {
			return err
		}
		if err = writeIntegerObject(ctx, objNr, genNr, *length); err != nil {
			return err
		}
	}

	return nil
}

func writeStreamObject(ctx *model.Context, objNr, genNr int, sd types.StreamDict, pdfString string) (int, int64, int, error) {
	h, err := writeObjectHeader(ctx.Write, objNr, genNr)
	if err != nil {
		return 0, 0, 0, err
	}

	// Note: Lines that are not part of stream object data are limited to no more than 255 characters.
	if _, err = ctx.Write.WriteString(pdfString); err != nil {
		return 0, 0, 0, err
	}

	b, err := writeStream(ctx.Write, sd)
	if err != nil {
		return 0, 0, 0, err
	}

	t, err := writeObjectTrailer(ctx.Write)
	if err != nil {
		return 0, 0, 0, err
	}

	return h, b, t, nil
}

func writeStreamDictObject(ctx *model.Context, objNr, genNr int, sd types.StreamDict) error {
	if log.WriteEnabled() {
		log.Write.Printf("writeStreamDictObject begin: object #%d\n%v", objNr, sd)
	}

	var inObjStream bool

	if ctx.Write.WriteToObjectStream {
		inObjStream = true
		ctx.Write.WriteToObjectStream = false
	}

	// Sometimes a streamDicts length is a reference.
	if ir := sd.IndirectRefEntry("Length"); ir != nil {
		if err := handleIndirectLength(ctx, ir); err != nil {
			return err
		}
	}

	var err error

	// Unless the "Identity" crypt filter is used we have to encrypt.
	isXRefStreamDict := sd.Type() != nil && *sd.Type() == "XRef"
	if ctx.EncKey != nil &&
		!isXRefStreamDict &&
		!(len(sd.FilterPipeline) == 1 && sd.FilterPipeline[0].Name == "Crypt") {

		if sd.Raw, err = encryptStream(sd.Raw, objNr, genNr, ctx.EncKey, ctx.AES4Streams, ctx.E.R); err != nil {
			return err
		}

		l := int64(len(sd.Raw))
		sd.StreamLength = &l
		sd.Update("Length", types.Integer(l))
	}

	ctx.Write.SetWriteOffset(objNr)

	pdfString := sd.PDFString()

	h, b, t, err := writeStreamObject(ctx, objNr, genNr, sd, pdfString)
	if err != nil {
		return err
	}

	written := b + int64(h+len(pdfString)+t)

	ctx.Write.Offset += written
	ctx.Write.BinaryTotalSize += *sd.StreamLength

	if inObjStream {
		ctx.Write.WriteToObjectStream = true
	}

	if log.WriteEnabled() {
		log.Write.Printf("writeStreamDictObject end: object #%d written=%d\n", objNr, written)
	}

	return nil
}

func writeDirectObject(ctx *model.Context, o types.Object) error {
	switch o := o.(type) {

	case types.Dict:
		for k, v := range o {
			if ctx.WritingPages && (k == "Dest" || k == "D") {
				ctx.Dest = true
			}
			if _, _, err := writeDeepObject(ctx, v); err != nil {
				return err
			}
			ctx.Dest = false
		}
		if log.WriteEnabled() {
			log.Write.Printf("writeDirectObject: end offset=%d\n", ctx.Write.Offset)
		}

	case types.Array:
		for i, v := range o {
			if ctx.Dest && i == 0 {
				continue
			}
			if _, _, err := writeDeepObject(ctx, v); err != nil {
				return err
			}
		}
		if log.WriteEnabled() {
			log.Write.Printf("writeDirectObject: end offset=%d\n", ctx.Write.Offset)
		}

	default:
		if log.WriteEnabled() {
			log.Write.Printf("writeDirectObject: end, direct obj - nothing written: offset=%d\n%v\n", ctx.Write.Offset, o)
		}
	}

	return nil
}

func writeNullObject(ctx *model.Context, objNumber, genNumber int) error {
	// An indirect reference to nil is a corner case.
	// Still, it is an object that will be written.
	if err := writePDFNullObject(ctx, objNumber, genNumber); err != nil {
		return err
	}

	// Ensure no entry in free list.
	return ctx.UndeleteObject(objNumber)
}

func writeDeepDict(ctx *model.Context, d types.Dict, objNr, genNr int) error {

	if d.IsPage() {
		valid, err := ctx.IsObjValid(objNr, genNr)
		if err != nil {
			return err
		}
		if !valid {
			return nil
		}
	}

	if err := writeDictObject(ctx, objNr, genNr, d); err != nil {
		return err
	}

	for k, v := range d {
		if ctx.WritingPages && (k == "Dest" || k == "D") {
			ctx.Dest = true
		}
		if _, _, err := writeDeepObject(ctx, v); err != nil {
			return err
		}
		ctx.Dest = false
	}

	return nil
}

func writeDeepStreamDict(ctx *model.Context, sd *types.StreamDict, objNr, genNr int) error {
	if ctx.EncKey != nil {
		if _, err := encryptDeepObject(*sd, objNr, genNr, ctx.EncKey, ctx.AES4Strings, ctx.E.R); err != nil {
			return err
		}
	}

	if err := writeStreamDictObject(ctx, objNr, genNr, *sd); err != nil {
		return err
	}

	for _, v := range sd.Dict {
		if _, _, err := writeDeepObject(ctx, v); err != nil {
			return err
		}
	}

	return nil
}

func writeDeepArray(ctx *model.Context, a types.Array, objNr, genNr int) error {
	if err := writeArrayObject(ctx, objNr, genNr, a); err != nil {
		return err
	}

	for i, v := range a {
		if ctx.Dest && i == 0 {
			continue
		}
		if _, _, err := writeDeepObject(ctx, v); err != nil {
			return err
		}
	}

	return nil
}

func writeLazyObjectStreamObject(ctx *model.Context, objNr, genNr int, o types.LazyObjectStreamObject) error {
	data, err := o.GetData()
	if err != nil {
		return err
	}

	return writeObject(ctx, objNr, genNr, string(data))
}

func writeObjectGeneric(ctx *model.Context, o types.Object, objNr, genNr int) (err error) {
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

	case types.LazyObjectStreamObject:
		err = writeLazyObjectStreamObject(ctx, objNr, genNr, o)

	default:
		err = errors.Errorf("writeIndirectObject: undefined PDF object #%d %T\n", objNr, o)
	}

	return err
}

func writeIndirectObject(ctx *model.Context, ir types.IndirectRef) error {
	objNr := int(ir.ObjectNumber)
	genNr := int(ir.GenerationNumber)

	if ctx.Write.HasWriteOffset(objNr) {
		if log.WriteEnabled() {
			log.Write.Printf("writeIndirectObject end: object #%d already written.\n", objNr)
		}
		return nil
	}

	o, err := ctx.DereferenceForWrite(ir)
	if err != nil {
		return errors.Wrapf(err, "writeIndirectObject: unable to dereference indirect object #%d", objNr)
	}

	if log.WriteEnabled() {
		log.Write.Printf("writeIndirectObject: object #%d gets writeoffset: %d\n", objNr, ctx.Write.Offset)
	}

	if o == nil {

		if err = writeNullObject(ctx, objNr, genNr); err != nil {
			return err
		}

		if log.WriteEnabled() {
			log.Write.Printf("writeIndirectObject: end, obj#%d resolved to nil, offset=%d\n", objNr, ctx.Write.Offset)
		}

		return nil
	}

	if err := writeObjectGeneric(ctx, o, objNr, genNr); err != nil {
		return err
	}

	return err
}

func writeDeepObject(ctx *model.Context, objIn types.Object) (objOut types.Object, written bool, err error) {
	if log.WriteEnabled() {
		log.Write.Printf("writeDeepObject: begin offset=%d\n%s\n", ctx.Write.Offset, objIn)
	}

	ir, ok := objIn.(types.IndirectRef)
	if !ok {
		return objIn, written, writeDirectObject(ctx, objIn)
	}

	if err = writeIndirectObject(ctx, ir); err == nil {
		written = true
		if log.WriteEnabled() {
			log.Write.Printf("writeDeepObject: end offset=%d\n", ctx.Write.Offset)
		}
	}

	return objOut, written, err
}

func writeEntry(ctx *model.Context, d types.Dict, dictName, entryName string) (types.Object, error) {
	o, found := d.Find(entryName)
	if !found || o == nil {
		if log.WriteEnabled() {
			log.Write.Printf("writeEntry end: entry %s is nil\n", entryName)
		}
		return nil, nil
	}

	if log.WriteEnabled() {
		log.Write.Printf("writeEntry begin: dict=%s entry=%s offset=%d\n", dictName, entryName, ctx.Write.Offset)
	}

	o, _, err := writeDeepObject(ctx, o)
	if err != nil {
		return nil, err
	}

	if o == nil {
		if log.WriteEnabled() {
			log.Write.Printf("writeEntry end: dict=%s entry=%s resolved to nil, offset=%d\n", dictName, entryName, ctx.Write.Offset)
		}
		return nil, nil
	}

	if log.WriteEnabled() {
		log.Write.Printf("writeEntry end: dict=%s entry=%s offset=%d\n", dictName, entryName, ctx.Write.Offset)
	}

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
