package pdfcpu

import (
	"fmt"

	"github.com/hhrutter/pdfcpu/log"
	"github.com/pkg/errors"
)

const (

	// ObjectStreamMaxObjects limits the number of objects within an object stream written.
	ObjectStreamMaxObjects = 100
)

func writeCommentLine(w *WriteContext, comment string) (int, error) {
	return w.WriteString(fmt.Sprintf("%%%s%s", comment, w.Eol))
}

func writeHeader(w *WriteContext, v PDFVersion) error {

	i, err := writeCommentLine(w, "PDF-"+VersionString(v))
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

func writeTrailer(w *WriteContext) (int, error) {
	return w.WriteString("%%EOF")
}

func writeObjectHeader(w *WriteContext, objNumber, genNumber int) (int, error) {
	return w.WriteString(fmt.Sprintf("%d %d obj%s", objNumber, genNumber, w.Eol))
}

func writeObjectTrailer(w *WriteContext) (int, error) {
	return w.WriteString(fmt.Sprintf("%sendobj%s", w.Eol, w.Eol))
}

func startObjectStream(ctx *PDFContext) error {

	// See 7.5.7 Object streams
	// When new object streams and compressed objects are created, they shall always be assigned new object numbers.

	log.Debug.Println("startObjectStream begin")

	objStreamDict := NewPDFObjectStreamDict()

	objNr, err := ctx.InsertObject(*objStreamDict)
	if err != nil {
		return err
	}

	ctx.Write.CurrentObjStream = &objNr

	log.Debug.Printf("startObjectStream end: %d\n", objNr)

	return nil
}

func stopObjectStream(ctx *PDFContext) error {

	log.Debug.Println("stopObjectStream begin")

	xRefTable := ctx.XRefTable

	if !ctx.Write.WriteToObjectStream {
		return errors.Errorf("stopObjectStream: Not writing to object stream.")
	}

	if ctx.Write.CurrentObjStream == nil {
		ctx.Write.WriteToObjectStream = false
		log.Debug.Println("stopObjectStream end (no content)")
		return nil
	}

	entry, _ := xRefTable.FindTableEntry(*ctx.Write.CurrentObjStream, 0)
	objStreamDict, _ := (entry.Object).(PDFObjectStreamDict)

	// When we are ready to write: append prolog and content
	objStreamDict.Finalize()

	// Encode objStreamDict.Content -> objStreamDict.Raw
	// and wipe (decoded) content to free up memory.
	err := encodeStream(&objStreamDict.PDFStreamDict)
	if err != nil {
		return err
	}

	// Release memory.
	objStreamDict.Content = nil

	objStreamDict.PDFStreamDict.Insert("First", PDFInteger(objStreamDict.FirstObjOffset))
	objStreamDict.PDFStreamDict.Insert("N", PDFInteger(objStreamDict.ObjCount))

	// for each objStream execute at the end right before xRefStreamDict gets written.
	log.Debug.Printf("stopObjectStream: objStreamDict: %s\n", objStreamDict)

	err = writePDFStreamDictObject(ctx, *ctx.Write.CurrentObjStream, 0, objStreamDict.PDFStreamDict)
	if err != nil {
		return err
	}

	// Release memory.
	objStreamDict.Raw = nil

	ctx.Write.CurrentObjStream = nil
	ctx.Write.WriteToObjectStream = false

	log.Debug.Println("stopObjectStream end")

	return nil
}

func writeToObjectStream(ctx *PDFContext, objNumber, genNumber int) (ok bool, err error) {

	log.Debug.Printf("addToObjectStream begin, obj#:%d gen#:%d\n", objNumber, genNumber)

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
		objStreamDict, _ := (objStrEntry.Object).(PDFObjectStreamDict)

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
		err = objStreamDict.AddObject(objNumber, entry)
		if err != nil {
			return false, err
		}

		objStrEntry.Object = objStreamDict

		log.Debug.Printf("writePDFObject end, obj#%d written to objectStream #%d\n", objNumber, *ctx.Write.CurrentObjStream)

		if objStreamDict.ObjCount == ObjectStreamMaxObjects {
			err = stopObjectStream(ctx)
			if err != nil {
				return false, err
			}
			w.WriteToObjectStream = true
		}

		ok = true

	}

	log.Debug.Printf("addToObjectStream end, obj#:%d gen#:%d\n", objNumber, genNumber)

	return ok, nil
}

func writePDFObject(ctx *PDFContext, objNumber, genNumber int, s string) error {

	log.Debug.Printf("writePDFObject begin, obj#:%d gen#:%d <%s>\n", objNumber, genNumber, s)

	w := ctx.Write

	// Cleanup entry (nexessary for split command)
	entry, _ := ctx.FindTableEntry(objNumber, genNumber)
	entry.Compressed = false

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

	log.Debug.Printf("writePDFObject end, %d bytes written\n", written+i+j)

	return nil
}

func writePDFNullObject(ctx *PDFContext, objNumber, genNumber int) error {

	ok, err := writeToObjectStream(ctx, objNumber, genNumber)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	return writePDFObject(ctx, objNumber, genNumber, "null")
}

func writePDFBooleanObject(ctx *PDFContext, objNumber, genNumber int, boolean PDFBoolean) error {

	ok, err := writeToObjectStream(ctx, objNumber, genNumber)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	return writePDFObject(ctx, objNumber, genNumber, boolean.PDFString())
}

func writePDFNameObject(ctx *PDFContext, objNumber, genNumber int, name PDFName) error {

	ok, err := writeToObjectStream(ctx, objNumber, genNumber)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	return writePDFObject(ctx, objNumber, genNumber, name.PDFString())
}

func writePDFStringLiteralObject(ctx *PDFContext, objNumber, genNumber int, stringLiteral PDFStringLiteral) error {

	ok, err := writeToObjectStream(ctx, objNumber, genNumber)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	sl := stringLiteral

	if ctx.EncKey != nil {
		s1, err := encryptString(ctx.AES4Strings, stringLiteral.Value(), objNumber, genNumber, ctx.EncKey)
		if err != nil {
			return err
		}

		sl = PDFStringLiteral(*s1)
	}

	return writePDFObject(ctx, objNumber, genNumber, sl.PDFString())
}

func writePDFHexLiteralObject(ctx *PDFContext, objNumber, genNumber int, hexLiteral PDFHexLiteral) error {

	ok, err := writeToObjectStream(ctx, objNumber, genNumber)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	hl := hexLiteral

	if ctx.EncKey != nil {
		s1, err := encryptString(ctx.AES4Strings, hexLiteral.Value(), objNumber, genNumber, ctx.EncKey)
		if err != nil {
			return err
		}

		hl = PDFHexLiteral(*s1)
	}

	return writePDFObject(ctx, objNumber, genNumber, hl.PDFString())
}

func writePDFIntegerObject(ctx *PDFContext, objNumber, genNumber int, integer PDFInteger) error {

	ok, err := writeToObjectStream(ctx, objNumber, genNumber)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	return writePDFObject(ctx, objNumber, genNumber, integer.PDFString())
}

func writePDFFloatObject(ctx *PDFContext, objNumber, genNumber int, float PDFFloat) error {

	ok, err := writeToObjectStream(ctx, objNumber, genNumber)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	return writePDFObject(ctx, objNumber, genNumber, float.PDFString())
}

func writePDFDictObject(ctx *PDFContext, objNumber, genNumber int, dict PDFDict) error {

	ok, err := writeToObjectStream(ctx, objNumber, genNumber)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	if ctx.EncKey != nil {
		_, err := encryptDeepObject(dict, objNumber, genNumber, ctx.EncKey, ctx.AES4Strings)
		if err != nil {
			return err
		}
	}

	return writePDFObject(ctx, objNumber, genNumber, dict.PDFString())
}

func writePDFArrayObject(ctx *PDFContext, objNumber, genNumber int, array PDFArray) error {

	ok, err := writeToObjectStream(ctx, objNumber, genNumber)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	if ctx.EncKey != nil {
		_, err := encryptDeepObject(array, objNumber, genNumber, ctx.EncKey, ctx.AES4Strings)
		if err != nil {
			return err
		}
	}

	return writePDFObject(ctx, objNumber, genNumber, array.PDFString())
}

func writeStream(w *WriteContext, streamDict PDFStreamDict) (int64, error) {

	b, err := w.WriteString(fmt.Sprintf("%sstream%s", w.Eol, w.Eol))
	if err != nil {
		return 0, errors.Wrapf(err, "writeStream: failed to write raw content")
	}

	c, err := w.Write(streamDict.Raw)
	if err != nil {
		return 0, errors.Wrapf(err, "writeStream: failed to write raw content")
	}
	if int64(c) != *streamDict.StreamLength {
		return 0, errors.Errorf("writeStream: failed to write raw content: %d bytes written - streamlength:%d", c, *streamDict.StreamLength)
	}

	e, err := w.WriteString("endstream")
	if err != nil {
		return 0, errors.Wrapf(err, "writeStream: failed to write raw content")
	}

	written := int64(b+e) + *streamDict.StreamLength

	return written, nil
}

func handleIndirectLength(ctx *PDFContext, indRef *PDFIndirectRef) error {

	objNumber := int(indRef.ObjectNumber)
	genNumber := int(indRef.GenerationNumber)

	if ctx.Write.HasWriteOffset(objNumber) {
		log.Debug.Printf("*** handleIndirectLength: object #%d already written offset=%d ***\n", objNumber, ctx.Write.Offset)
	} else {
		length, err := ctx.DereferenceInteger(*indRef)
		if err != nil || length == nil {
			return err
		}
		err = writePDFIntegerObject(ctx, objNumber, genNumber, *length)
		if err != nil {
			return err
		}
	}

	return nil
}

func writePDFStreamDictObject(ctx *PDFContext, objNumber, genNumber int, streamDict PDFStreamDict) error {

	log.Debug.Printf("writePDFStreamDictObject begin: object #%d\n%v", objNumber, streamDict)

	var inObjStream bool
	if ctx.Write.WriteToObjectStream == true {
		inObjStream = true
		ctx.Write.WriteToObjectStream = false
	}

	// Sometimes a streamDicts length is a reference.
	if indRef := streamDict.IndirectRefEntry("Length"); indRef != nil {
		err := handleIndirectLength(ctx, indRef)
		if err != nil {
			return err
		}
	}

	var err error

	// Unless the "Identity" crypt filter is used we have to encrypt.
	isXRefStreamDict := streamDict.Type() != nil && *streamDict.Type() == "XRef"
	if ctx.EncKey != nil &&
		!isXRefStreamDict &&
		!(len(streamDict.FilterPipeline) == 1 && streamDict.FilterPipeline[0].Name == "Crypt") {

		streamDict.Raw, err = encryptStream(ctx.AES4Streams, streamDict.Raw, objNumber, genNumber, ctx.EncKey)
		if err != nil {
			return err
		}

		l := int64(len(streamDict.Raw))
		streamDict.StreamLength = &l
		streamDict.Update("Length", PDFInteger(l))
	}

	ctx.Write.SetWriteOffset(objNumber)

	h, err := writeObjectHeader(ctx.Write, objNumber, genNumber)
	if err != nil {
		return err
	}

	// Note: Lines that are not part of stream object data are limited to no more than 255 characters.
	pdfString := streamDict.PDFString()
	_, err = ctx.Write.WriteString(pdfString)
	if err != nil {
		return err
	}

	b, err := writeStream(ctx.Write, streamDict)
	if err != nil {
		return err
	}

	t, err := writeObjectTrailer(ctx.Write)
	if err != nil {
		return err
	}

	written := b + int64(h+len(pdfString)+t)

	ctx.Write.Offset += written
	ctx.Write.BinaryTotalSize += *streamDict.StreamLength

	if inObjStream {
		ctx.Write.WriteToObjectStream = true
	}

	log.Debug.Printf("writePDFStreamDictObject end: object #%d written=%d\n", objNumber, written)

	return nil
}

func writeDirectObject(ctx *PDFContext, o PDFObject) error {

	switch o := o.(type) {

	case PDFDict:
		for _, v := range o.Dict {
			_, _, err := writeDeepObject(ctx, v)
			if err != nil {
				return err
			}
		}
		log.Debug.Printf("writeDirectObject: end offset=%d\n", ctx.Write.Offset)

	case PDFArray:
		for _, v := range o {
			_, _, err := writeDeepObject(ctx, v)
			if err != nil {
				return err
			}
		}
		log.Debug.Printf("writeDirectObject: end offset=%d\n", ctx.Write.Offset)

	default:
		log.Debug.Printf("writeDirectObject: end, direct obj - nothing written: offset=%d\n%v\n", ctx.Write.Offset, o)

	}

	return nil
}

func writeNullObject(ctx *PDFContext, objNumber, genNumber int) error {

	// An indirect reference to nil is a corner case.
	// Still, it is an object that will be written.
	err := writePDFNullObject(ctx, objNumber, genNumber)
	if err != nil {
		return err
	}

	// Ensure no entry in free list.
	return ctx.UndeleteObject(objNumber)
}

func writeDeepPDFDict(ctx *PDFContext, d PDFDict, objNr, genNr int) error {

	err := writePDFDictObject(ctx, objNr, genNr, d)
	if err != nil {
		return err
	}

	for _, v := range d.Dict {
		_, _, err = writeDeepObject(ctx, v)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeDeepPDFStreamDict(ctx *PDFContext, sd *PDFStreamDict, objNr, genNr int) error {

	if ctx.EncKey != nil {
		_, err := encryptDeepObject(*sd, objNr, genNr, ctx.EncKey, ctx.AES4Strings)
		if err != nil {
			return err
		}
	}

	err := writePDFStreamDictObject(ctx, objNr, genNr, *sd)
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

func writeDeepPDFArray(ctx *PDFContext, arr PDFArray, objNr, genNr int) error {

	err := writePDFArrayObject(ctx, objNr, genNr, arr)
	if err != nil {
		return err
	}

	for _, v := range arr {
		_, _, err = writeDeepObject(ctx, v)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeIndirectObject(ctx *PDFContext, indRef PDFIndirectRef) (PDFObject, error) {

	objNumber := int(indRef.ObjectNumber)
	genNumber := int(indRef.GenerationNumber)

	if ctx.Write.HasWriteOffset(objNumber) {
		log.Debug.Printf("writeIndirectObject end: object #%d already written.\n", objNumber)
		return nil, nil
	}

	o, err := ctx.Dereference(indRef)
	if err != nil {
		return nil, errors.Wrapf(err, "writeIndirectObject: unable to dereference indirect object #%d", objNumber)
	}

	log.Debug.Printf("writeIndirectObject: object #%d gets writeoffset: %d\n", objNumber, ctx.Write.Offset)

	if o == nil {

		err = writeNullObject(ctx, objNumber, genNumber)
		if err != nil {
			return nil, err
		}

		log.Debug.Printf("writeIndirectObject: end, obj#%d resolved to nil, offset=%d\n", objNumber, ctx.Write.Offset)
		return nil, nil
	}

	switch obj := o.(type) {

	case PDFDict:
		err = writeDeepPDFDict(ctx, obj, objNumber, genNumber)

	case PDFStreamDict:
		err = writeDeepPDFStreamDict(ctx, &obj, objNumber, genNumber)

	case PDFArray:
		err = writeDeepPDFArray(ctx, obj, objNumber, genNumber)

	case PDFInteger:
		err = writePDFIntegerObject(ctx, objNumber, genNumber, obj)

	case PDFFloat:
		err = writePDFFloatObject(ctx, objNumber, genNumber, obj)

	case PDFStringLiteral:
		err = writePDFStringLiteralObject(ctx, objNumber, genNumber, obj)

	case PDFHexLiteral:
		err = writePDFHexLiteralObject(ctx, objNumber, genNumber, obj)

	case PDFBoolean:
		err = writePDFBooleanObject(ctx, objNumber, genNumber, obj)

	case PDFName:
		err = writePDFNameObject(ctx, objNumber, genNumber, obj)

	default:
		return nil, errors.Errorf("writeIndirectObject: undefined PDF object #%d %T\n", objNumber, o)

	}

	return nil, err
}

func writeDeepObject(ctx *PDFContext, objIn PDFObject) (objOut PDFObject, written bool, err error) {

	log.Debug.Printf("writeDeepObject: begin offset=%d\n%s\n", ctx.Write.Offset, objIn)

	indRef, ok := objIn.(PDFIndirectRef)
	if !ok {
		//err = writeDirectObject(ctx, objIn)
		//objOut = objIn
		return objIn, written, writeDirectObject(ctx, objIn)
	}

	objOut, err = writeIndirectObject(ctx, indRef)
	if err == nil {
		written = true
		log.Debug.Printf("writeDeepObject: end offset=%d\n", ctx.Write.Offset)
	}

	return objOut, written, err
}

func writeEntry(ctx *PDFContext, dict *PDFDict, dictName, entryName string) (PDFObject, error) {

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		log.Debug.Printf("writeEntry end: entry %s is nil\n", entryName)
		return nil, nil
	}

	log.Debug.Printf("writeEntry begin: dict=%s entry=%s offset=%d\n", dictName, entryName, ctx.Write.Offset)

	obj, _, err := writeDeepObject(ctx, obj)
	if err != nil {
		return nil, err
	}

	if obj == nil {
		log.Debug.Printf("writeEntry end: dict=%s entry=%s resolved to nil, offset=%d\n", dictName, entryName, ctx.Write.Offset)
		return nil, nil
	}

	log.Debug.Printf("writeEntry end: dict=%s entry=%s offset=%d\n", dictName, entryName, ctx.Write.Offset)

	return obj, nil
}
