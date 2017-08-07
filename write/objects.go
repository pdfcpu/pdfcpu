package write

import (
	"fmt"

	"github.com/hhrutter/pdflib/filter"
	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

const (

	// REQUIRED is used for required dict entries.
	REQUIRED = true

	// OPTIONAL is used for optional dict entries.
	OPTIONAL = false

	// ObjectStreamMaxObjects limits the number of objects within an object stream written.
	ObjectStreamMaxObjects = 100
)

func writeCommentLine(w *types.WriteContext, comment string) (int, error) {
	return w.WriteString(fmt.Sprintf("%%%s%s", comment, w.Eol))
}

func writeHeader(w *types.WriteContext, v types.PDFVersion) error {

	i, err := writeCommentLine(w, "PDF-"+types.VersionString(v))
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

func writeTrailer(w *types.WriteContext) (int, error) {
	return w.WriteString("%%EOF")
}

func writeObjectHeader(w *types.WriteContext, objNumber, genNumber int) (int, error) {
	return w.WriteString(fmt.Sprintf("%d %d obj%s", objNumber, genNumber, w.Eol))
}

func writeObjectTrailer(w *types.WriteContext) (int, error) {
	return w.WriteString(fmt.Sprintf("%sendobj%s", w.Eol, w.Eol))
}

func startObjectStream(ctx *types.PDFContext) (err error) {

	// See 7.5.7 Object streams
	// When new object streams and compressed objects are created, they shall always be assigned new object numbers,
	// not old ones taken from the free list.

	logDebugWriter.Println("startObjectStream begin")

	xRefTable := ctx.XRefTable
	objStreamDict := types.NewPDFObjectStreamDict()
	xRefTableEntry := types.NewXRefTableEntryGen0()
	xRefTableEntry.Object = *objStreamDict

	objNumber, ok := xRefTable.InsertNew(*xRefTableEntry)
	if !ok {
		return errors.Errorf("startObjectStream: Problem inserting entry for %d", objNumber)
	}

	ctx.Write.CurrentObjStream = &objNumber

	logDebugWriter.Println("startObjectStream end")

	return
}

func stopObjectStream(ctx *types.PDFContext) (err error) {

	logDebugWriter.Println("stopObjectStream begin")

	xRefTable := ctx.XRefTable

	if !ctx.Write.WriteToObjectStream {
		err = errors.Errorf("stopObjectStream: Not writing to object stream.")
		return
	}

	if ctx.Write.CurrentObjStream == nil {
		ctx.Write.WriteToObjectStream = false
		logDebugWriter.Println("stopObjectStream end (no content)")
		return
	}

	entry, _ := xRefTable.FindTableEntry(*ctx.Write.CurrentObjStream, 0)
	objStreamDict, _ := (entry.Object).(types.PDFObjectStreamDict)

	// When we are ready to write: append prolog and content
	objStreamDict.Finalize()

	// Encode objStreamDict.Content -> objStreamDict.Raw
	// and wipe (decoded) content to free up memory.
	err = filter.EncodeStream(&objStreamDict.PDFStreamDict)
	if err != nil {
		return
	}

	// Release memory.
	objStreamDict.Content = nil

	objStreamDict.PDFStreamDict.Insert("First", types.PDFInteger(objStreamDict.FirstObjOffset))
	objStreamDict.PDFStreamDict.Insert("N", types.PDFInteger(objStreamDict.ObjCount))

	// for each objStream execute at the end right before xRefStreamDict gets written.
	logDebugWriter.Printf("stopObjectStream: objStreamDict: %s\n", objStreamDict)

	err = writePDFStreamDictObject(ctx, *ctx.Write.CurrentObjStream, 0, objStreamDict.PDFStreamDict)
	if err != nil {
		return
	}

	// Release memory.
	objStreamDict.Raw = nil

	ctx.Write.CurrentObjStream = nil
	ctx.Write.WriteToObjectStream = false

	logDebugWriter.Println("stopObjectStream end")

	return
}

func writePDFObject(ctx *types.PDFContext, objNumber, genNumber int, s string) (err error) {

	logInfoWriter.Printf("writePDFObject begin, obj#:%d gen#:%d\n", objNumber, genNumber)

	w := ctx.Write

	if ctx.WriteXRefStream && // object streams assume an xRefStream to be generated.
		ctx.WriteObjectStream && // signal for compression into object stream is on.
		ctx.Write.WriteToObjectStream && // currently writing to object stream.
		genNumber == 0 {

		if w.CurrentObjStream == nil {
			// Create new objects stream on first write.
			err = startObjectStream(ctx)
			if err != nil {
				return
			}
		}

		xRefTable := ctx.XRefTable

		objStrEntry, _ := xRefTable.FindTableEntry(*ctx.Write.CurrentObjStream, 0)
		objStreamDict, _ := (objStrEntry.Object).(types.PDFObjectStreamDict)

		// Get next free index in object stream.
		i := objStreamDict.ObjCount

		// Locate the xref table entry for the object to be added to this object stream.
		entry, _ := xRefTable.FindTableEntry(objNumber, genNumber)

		// Turn entry into a compressed entry located in object stream at index i.
		entry.Compressed = true
		entry.ObjectStream = ctx.Write.CurrentObjStream // !
		entry.ObjectStreamInd = &i
		w.SetWriteOffset(objNumber) // for a compressed obj this is supposed to be a fake offset. value does not matter.

		// Append to prolog & content
		err = objStreamDict.AddObject(objNumber, entry)
		if err != nil {
			return
		}

		objStrEntry.Object = objStreamDict

		logInfoWriter.Printf("writePDFObject end, obj#%d written to objectStream #%d\n", objNumber, *ctx.Write.CurrentObjStream)

		if objStreamDict.ObjCount == ObjectStreamMaxObjects {
			err = stopObjectStream(ctx)
			if err != nil {
				return
			}
			w.WriteToObjectStream = true
		}

		return
	}

	// Set write-offset for this object.
	w.SetWriteOffset(objNumber)

	written, err := writeObjectHeader(w, objNumber, genNumber)
	if err != nil {
		return
	}

	// Note: Lines that are not part of stream object data are limited to no more than 255 characters.
	i, err := w.WriteString(s)
	if err != nil {
		return
	}

	j, err := writeObjectTrailer(w)
	if err != nil {
		return
	}

	// Write-offset for next object.
	w.Offset += int64(written + i + j)

	logInfoWriter.Printf("writePDFObject end, %d bytes written\n", written)

	return
}

func writePDFNullObject(ctx *types.PDFContext, objNumber, genNumber int) error {
	return writePDFObject(ctx, objNumber, genNumber, "null")
}

func writePDFBooleanObject(ctx *types.PDFContext, objNumber, genNumber int, boolean types.PDFBoolean) error {
	return writePDFObject(ctx, objNumber, genNumber, boolean.PDFString())
}

func writePDFNameObject(ctx *types.PDFContext, objNumber, genNumber int, name types.PDFName) error {
	return writePDFObject(ctx, objNumber, genNumber, name.PDFString())
}

func writePDFStringLiteralObject(ctx *types.PDFContext, objNumber, genNumber int, stringLiteral types.PDFStringLiteral) error {
	return writePDFObject(ctx, objNumber, genNumber, stringLiteral.PDFString())
}

func writePDFHexLiteralObject(ctx *types.PDFContext, objNumber, genNumber int, hexLiteral types.PDFHexLiteral) error {
	return writePDFObject(ctx, objNumber, genNumber, hexLiteral.PDFString())
}

func writePDFIntegerObject(ctx *types.PDFContext, objNumber, genNumber int, integer types.PDFInteger) error {
	return writePDFObject(ctx, objNumber, genNumber, integer.PDFString())
}

func writePDFFloatObject(ctx *types.PDFContext, objNumber, genNumber int, float types.PDFFloat) error {
	return writePDFObject(ctx, objNumber, genNumber, float.PDFString())
}

func writePDFDictObject(ctx *types.PDFContext, objNumber, genNumber int, dict types.PDFDict) error {
	return writePDFObject(ctx, objNumber, genNumber, dict.PDFString())
}

func writePDFArrayObject(ctx *types.PDFContext, objNumber, genNumber int, array types.PDFArray) error {
	return writePDFObject(ctx, objNumber, genNumber, array.PDFString())
}

func writeStream(w *types.WriteContext, streamDict types.PDFStreamDict) (int64, error) {

	b, err := w.WriteString(fmt.Sprintf("%sstream%s", w.Eol, w.Eol))
	if err != nil {
		return 0, errors.Wrapf(err, "writeStream: failed to write raw content")
	}

	c, err := w.WriteString(string(streamDict.Raw))
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

func writePDFStreamDictObject(ctx *types.PDFContext, objNumber, genNumber int, streamDict types.PDFStreamDict) error {

	logInfoWriter.Printf("writePDFStreamDictObject begin: object #%d\n", objNumber)

	xRefTable := ctx.XRefTable

	var inObjStream bool
	if ctx.Write.WriteToObjectStream == true {
		inObjStream = true
		ctx.Write.WriteToObjectStream = false
	}

	// Sometimes a streamDicts length is a reference.
	if indRef := streamDict.IndirectRefEntry("Length"); indRef != nil {

		objNumber := int(indRef.ObjectNumber)
		genNumber := int(indRef.GenerationNumber)

		if ctx.Write.HasWriteOffset(objNumber) {
			logInfoWriter.Printf("*** writePDFStreamDictObject: object #%d already written offset=%d ***\n", objNumber, ctx.Write.Offset)
		} else {
			length, err := xRefTable.DereferenceInteger(*indRef)
			if err != nil || length == nil {
				return err
			}
			err = writePDFIntegerObject(ctx, objNumber, genNumber, *length)
			if err != nil {
				return err
			}
		}

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

	logInfoWriter.Printf("writePDFStreamDictObject end: object #%d written=%d\n", objNumber, written)

	return nil
}

// writeIndRef writes an indirect referenced object to the PDFdestination.
//
// 1) The object is already written.										=> nil, true, nil
// 2) The object cannot be dereferenced in the xreftable of the PDFSource. 	=> nil, false, err
// 3) The object is nil.													=> nil, false, nil
// 4) The object is undefined.												=> obj, false, err
// 5) A low level write error.												=> nil, false, err
// 6) Successful write.														=> obj, false, nil
// func writeIndRef(ctx *types.PDFContext, indRef types.PDFIndirectRef) (obj interface{}, written bool, err error) {

// 	logInfoWriter.Printf("writeIndRef: begin offset=%d\n", ctx.Write.Offset)

// 	xRefTable := ctx.XRefTable
// 	objNumber := int(indRef.ObjectNumber)
// 	genNumber := int(indRef.GenerationNumber)

// 	if ctx.Write.HasWriteOffset(objNumber) {
// 		logInfoWriter.Printf("writeIndRef end: object #%d already written.\n", objNumber)
// 		written = true
// 		return
// 	}

// 	obj, err = xRefTable.Dereference(indRef)
// 	if err != nil {
// 		err = errors.Wrapf(err, "writeIndRef: unable to dereference indirect object #%d", objNumber)
// 		return
// 	}

// 	logInfoWriter.Printf("writeIndRef: object #%d gets writeoffset: %d\n", objNumber, ctx.Write.Offset)

// 	if obj == nil {
// 		logInfoWriter.Printf("writeIndRef end: object #%d is nil.\n", objNumber)
// 		ctx.UndeleteObject(objNumber)
// 		err = writePDFNullObject(ctx, objNumber, genNumber)
// 		logInfoWriter.Printf("writeIndRef: end offset=%d\n", ctx.Write.Offset)
// 		return
// 	}

// 	switch obj := obj.(type) {

// 	case types.PDFDict:
// 		err = writePDFDictObject(ctx, objNumber, genNumber, obj)

// 	case types.PDFStreamDict:
// 		err = writePDFStreamDictObject(ctx, objNumber, genNumber, obj)

// 	case types.PDFArray:
// 		err = writePDFArrayObject(ctx, objNumber, genNumber, obj)

// 	case types.PDFInteger:
// 		err = writePDFIntegerObject(ctx, objNumber, genNumber, obj)

// 	case types.PDFFloat:
// 		err = writePDFFloatObject(ctx, objNumber, genNumber, obj)

// 	case types.PDFStringLiteral:
// 		err = writePDFStringLiteralObject(ctx, objNumber, genNumber, obj)

// 	case types.PDFHexLiteral:
// 		err = writePDFHexLiteralObject(ctx, objNumber, genNumber, obj)

// 	case types.PDFBoolean:
// 		err = writePDFBooleanObject(ctx, objNumber, genNumber, obj)

// 	case types.PDFName:
// 		err = writePDFNameObject(ctx, objNumber, genNumber, obj)

// 	default:
// 		err = errors.Errorf("writeIndRef: undefined PDF object #%d\n", objNumber)

// 	}

// 	logInfoWriter.Printf("writeIndRef: end offset=%d\n", ctx.Write.Offset)

// 	return
// }

// writeObject writes an object to a PDFDestination.

// 1) The object is not an indirect reference, no need for writing.			=> obj, false, nil
// 2) The object is already written.										=> nil, true, nil
// 3) The object cannot be dereferenced in the xreftable of the PDFSource. 	=> nil, false, err
// 4) The object is nil.													=> nil, false, nil
// 5) The object is undefined.												=> obj, false, err
// 6) A low level write error.												=> nil, false, err
// 7) Successful write.														=> obj, false, nil
// func writeObject(ctx *types.PDFContext, objIn interface{}) (objOut interface{}, written bool, err error) {

// 	//logDebugWriter.Printf("writeObject: begin offset=%d\n", ctx.Write.Offset)
// 	offsetOld := ctx.Write.Offset

// 	indRef, ok := objIn.(types.PDFIndirectRef)
// 	if !ok {
// 		//logDebugWriter.Printf("writeObject: end, direct obj - nothing written: offset=%d\n", ctx.Write.Offset)
// 		objOut = objIn
// 		return
// 	}

// 	objOut, written, err = writeIndRef(ctx, indRef)

// 	logDebugWriter.Printf("writeObject: #%d offsetOld=%d offsetNew=%d\n", int(indRef.ObjectNumber), offsetOld, ctx.Write.Offset)

// 	return
// }

func writeDeepObject(ctx *types.PDFContext, objIn interface{}) (objOut interface{}, written bool, err error) {

	logDebugWriter.Printf("writeDeepObject: begin offset=%d\n", ctx.Write.Offset)

	indRef, ok := objIn.(types.PDFIndirectRef)
	if !ok {

		switch obj := objIn.(type) {

		case types.PDFDict:
			for _, v := range obj.Dict {
				_, _, err = writeDeepObject(ctx, v)
				if err != nil {
					return
				}
			}
			logDebugWriter.Printf("writeDeepObject: end offset=%d\n", ctx.Write.Offset)

		case types.PDFArray:
			for _, v := range obj {
				_, _, err = writeDeepObject(ctx, v)
				if err != nil {
					return
				}
			}
			logDebugWriter.Printf("writeDeepObject: end offset=%d\n", ctx.Write.Offset)

		default:
			logDebugWriter.Printf("writeDeepObject: end, direct obj - nothing written: offset=%d\n%v\n", ctx.Write.Offset, objIn)

		}

		objOut = objIn
		return
	}

	objNumber := int(indRef.ObjectNumber)
	genNumber := int(indRef.GenerationNumber)

	if ctx.Write.HasWriteOffset(objNumber) {
		logDebugWriter.Printf("writeDeepObject end: object #%d already written.\n", objNumber)
		return
	}

	obj, err := ctx.Dereference(indRef)
	if err != nil {
		err = errors.Wrapf(err, "writeDeepObject: unable to dereference indirect object #%d", objNumber)
		return
	}

	logDebugWriter.Printf("writeDeepObject: object #%d gets writeoffset: %d\n", objNumber, ctx.Write.Offset)

	if obj == nil {

		// An indirect reference to nil is a corner case.
		// Still, it is an object that will be written.
		err = writePDFNullObject(ctx, objNumber, genNumber)
		if err != nil {
			return
		}

		// Ensure no entry in free list.
		err = ctx.UndeleteObject(objNumber)
		if err != nil {
			return
		}

		written = true

		logDebugWriter.Printf("writeDeepObject: end, obj#%d resolved to nil, offset=%d\n", objNumber, ctx.Write.Offset)

		return
	}

	switch obj := obj.(type) {

	case types.PDFDict:
		err = writePDFDictObject(ctx, objNumber, genNumber, obj)
		if err != nil {
			return
		}
		for _, v := range obj.Dict {
			_, _, err = writeDeepObject(ctx, v)
			if err != nil {
				return
			}
		}

	case types.PDFStreamDict:
		err = writePDFStreamDictObject(ctx, objNumber, genNumber, obj)
		if err != nil {
			return
		}
		for _, v := range obj.Dict {
			_, _, err = writeDeepObject(ctx, v)
			if err != nil {
				return
			}
		}

	case types.PDFArray:
		err = writePDFArrayObject(ctx, objNumber, genNumber, obj)
		if err != nil {
			return
		}
		for _, v := range obj {
			_, _, err = writeDeepObject(ctx, v)
			if err != nil {
				return
			}
		}

	case types.PDFInteger:
		err = writePDFIntegerObject(ctx, objNumber, genNumber, obj)

	case types.PDFFloat:
		err = writePDFFloatObject(ctx, objNumber, genNumber, obj)

	case types.PDFStringLiteral:
		err = writePDFStringLiteralObject(ctx, objNumber, genNumber, obj)

	case types.PDFHexLiteral:
		err = writePDFHexLiteralObject(ctx, objNumber, genNumber, obj)

	case types.PDFBoolean:
		err = writePDFBooleanObject(ctx, objNumber, genNumber, obj)

	case types.PDFName:
		err = writePDFNameObject(ctx, objNumber, genNumber, obj)

	default:
		err = errors.Errorf("writeDeepObject: undefined PDF object #%d\n", objNumber)

	}

	if err == nil {
		objOut = obj
		written = true
		logDebugWriter.Printf("writeDeepObject: end offset=%d\n", ctx.Write.Offset)
	}

	return
}

func writeEntry(ctx *types.PDFContext, dict *types.PDFDict, dictName, entryName string) (obj interface{}, err error) {

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		logDebugWriter.Printf("writeEntry end: entry %s is nil\n", entryName)
		return
	}

	logInfoWriter.Printf("writeEntry begin: dict=%s entry=%s offset=%d\n", dictName, entryName, ctx.Write.Offset)

	obj, _, err = writeDeepObject(ctx, obj)
	if err != nil {
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeEntry end: dict=%s entry=%s resolved to nil, offset=%d\n", dictName, entryName, ctx.Write.Offset)
		return
	}

	logInfoWriter.Printf("writeEntry end: dict=%s entry=%s offset=%d\n", dictName, entryName, ctx.Write.Offset)

	return
}
