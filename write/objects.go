package write

import (
	"fmt"

	"github.com/hhrutter/pdflib/types"
	"github.com/hhrutter/pdflib/validate"
	"github.com/pkg/errors"
)

func writeCommentLine(w *types.WriteContext, comment string) (int, error) {
	return w.WriteString(fmt.Sprintf("%%%s%s", comment, eol))
}

func writeHeader(ctx *types.PDFContext) error {

	v := ctx.XRefTable.Version()

	i, err := writeCommentLine(ctx.Write, "PDF-"+types.VersionString(v))
	if err != nil {
		return err
	}

	j, err := writeCommentLine(ctx.Write, "\xe2\xe3\xcf\xD3")
	if err != nil {
		return err
	}

	ctx.Write.Offset += int64(i + j)

	return nil
}

func writeTrailer(ctx *types.PDFContext) (int, error) {
	return ctx.Write.WriteString("%%EOF")
}

func writeObjectHeader(w *types.WriteContext, objNumber, genNumber int) (int, error) {
	return w.WriteString(fmt.Sprintf("%d %d obj%s", objNumber, genNumber, eol))
}

func writeObjectTrailer(w *types.WriteContext) (int, error) {
	return w.WriteString(fmt.Sprintf("%sendobj%s", eol, eol))
}

func writePDFObject(ctx *types.PDFContext, objNumber, genNumber int, s string) (err error) {

	logInfoWriter.Printf("writePDFObject begin, obj#:%d gen#:%d\n", objNumber, genNumber)

	if ctx.WriteXRefStream && // object streams assume an xRefStream to be generated.
		ctx.WriteObjectStream && // signal for compression into object stream is on.
		ctx.Write.WriteToObjectStream && // currently writing to object stream.
		genNumber == 0 {

		if ctx.Write.CurrentObjStream == nil {
			// Create new objects stream on first write.
			err = startObjectStream(ctx)
			if err != nil {
				return
			}
		}

		xRefTable := ctx.XRefTable

		objStrEntry, _ := xRefTable.FindTableEntry(*ctx.Write.CurrentObjStream, 0)
		objStreamDict, _ := (objStrEntry.Object).(types.PDFObjectStreamDict)

		i := objStreamDict.ObjCount
		entry, _ := xRefTable.FindTableEntry(objNumber, genNumber)
		entry.Compressed = true
		entry.ObjectStream = ctx.Write.CurrentObjStream // !
		entry.ObjectStreamInd = &i
		ctx.Write.SetWriteOffset(objNumber) // for a compressed obj this is supposed to be a fake offset. value does not matter.

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
			ctx.Write.WriteToObjectStream = true
		}

		return
	}

	// Set write-offset for this object.
	ctx.Write.SetWriteOffset(objNumber)

	written, err := writeObjectHeader(ctx.Write, objNumber, genNumber)
	if err != nil {
		return
	}

	i, err := ctx.Write.Writer.WriteString(s)
	if err != nil {
		return
	}

	j, err := writeObjectTrailer(ctx.Write)
	if err != nil {
		return
	}

	// Write-offset for next object.
	ctx.Write.Offset += int64(written + i + j)

	// TODO max 255 chars per line!
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

	b, err := w.WriteString(fmt.Sprintf("%sstream%s", eol, eol))
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

	// Stream dicts are not to be written to object streams.
	// Save pointer to object number of current object stream.
	//var saveObjStrPtr *int
	//if ctx.Write.CurrentObjStream != nil {
	//	saveObjStrPtr = ctx.Write.CurrentObjStream
	//	ctx.Write.CurrentObjStream = nil
	//}

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

	// TODO max 255 chars per line!
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

	// Restore pointer to object number of current object stream.
	//if saveObjStrPtr != nil {
	//	ctx.Write.CurrentObjStream = saveObjStrPtr
	//}

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
func writeIndRef(ctx *types.PDFContext, indRef types.PDFIndirectRef) (obj interface{}, written bool, err error) {

	logInfoWriter.Printf("writeIndRef: begin offset=%d\n", ctx.Write.Offset)

	xRefTable := ctx.XRefTable
	objNumber := int(indRef.ObjectNumber)
	genNumber := int(indRef.GenerationNumber)

	if ctx.Write.HasWriteOffset(objNumber) {
		logInfoWriter.Printf("writeIndRef end: object #%d already written.\n", objNumber)
		written = true
		return
	}

	obj, err = xRefTable.Dereference(indRef)
	if err != nil {
		err = errors.Wrapf(err, "writeIndRef: unable to dereference indirect object #%d", objNumber)
		return
	}

	logInfoWriter.Printf("writeIndRef: object #%d gets writeoffset: %d\n", objNumber, ctx.Write.Offset)

	if obj == nil {
		logInfoWriter.Printf("writeIndRef end: object #%d is nil.\n", objNumber)
		err = writePDFNullObject(ctx, objNumber, genNumber)
		logInfoWriter.Printf("writeIndRef: end offset=%d\n", ctx.Write.Offset)
		return
	}

	switch obj := obj.(type) {

	case types.PDFDict:
		err = writePDFDictObject(ctx, objNumber, genNumber, obj)

	case types.PDFStreamDict:
		err = writePDFStreamDictObject(ctx, objNumber, genNumber, obj)

	case types.PDFArray:
		err = writePDFArrayObject(ctx, objNumber, genNumber, obj)

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
		err = errors.Errorf("writeIndRef: undefined PDF object #%d\n", objNumber)

	}

	logInfoWriter.Printf("writeIndRef: end offset=%d\n", ctx.Write.Offset)

	return
}

// writeObject writes an object to a PDFDestination.
//
// 1) The object is not an indirect reference, no need for writing.			=> obj, false, nil
// 2) The object is already written.										=> nil, true, nil
// 3) The object cannot be dereferenced in the xreftable of the PDFSource. 	=> nil, false, err
// 4) The object is nil.													=> nil, false, nil
// 5) The object is undefined.												=> obj, false, err
// 6) A low level write error.												=> nil, false, err
// 7) Successful write.														=> obj, false, nil
func writeObject(ctx *types.PDFContext, objIn interface{}) (objOut interface{}, written bool, err error) {

	//logDebugWriter.Printf("writeObject: begin offset=%d\n", ctx.Write.Offset)
	offsetOld := ctx.Write.Offset

	indRef, ok := objIn.(types.PDFIndirectRef)
	if !ok {
		//logDebugWriter.Printf("writeObject: end, direct obj - nothing written: offset=%d\n", ctx.Write.Offset)
		objOut = objIn
		return
	}

	objOut, written, err = writeIndRef(ctx, indRef)

	logDebugWriter.Printf("writeObject: #%d offsetOld=%d offsetNew=%d\n", int(indRef.ObjectNumber), offsetOld, ctx.Write.Offset)

	return
}

func writeString(ctx *types.PDFContext, obj interface{}, validate func(string) bool) (s *string, written bool, err error) {

	logInfoWriter.Printf("writeString begin: offset=%d\n", ctx.Write.Offset)

	obj, written, err = writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written || obj == nil {
		return
	}

	var str string

	switch obj := obj.(type) {

	case types.PDFStringLiteral:
		str = obj.Value()

	case types.PDFHexLiteral:
		str = obj.Value()

	default:
		err = errors.Errorf("writeString: invalid type: %v", obj)
		return
	}

	// Validation
	if validate != nil && !validate(str) {
		err = errors.Errorf("writeString: invalid string: %s", str)
		return
	}

	s = &str

	logInfoWriter.Printf("writeString end: offset=%d\n", ctx.Write.Offset)

	return
}

func writeTextString(ctx *types.PDFContext, obj interface{}, validate func(string) bool) (s *string, written bool, err error) {

	logInfoWriter.Printf("writeTextString begin: offset=%d\n", ctx.Write.Offset)

	obj, written, err = writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written || obj == nil {
		return
	}

	var str string

	switch obj := obj.(type) {

	case types.PDFStringLiteral:
		str, err = types.StringLiteralToString(obj.Value())
		if err != nil {
			return nil, false, err
		}

	case types.PDFHexLiteral:
		str, err = types.HexLiteralToString(obj.Value())
		if err != nil {
			return nil, false, err
		}

	default:
		err = errors.Errorf("writeTextString: invalid type: %v", obj)
		return
	}

	// Validation
	if validate != nil && !validate(str) {
		err = errors.Errorf("writeTextString: invalid string: %s", str)
		return
	}

	s = &str

	logInfoWriter.Printf("writeTextString end: offset=%d\n", ctx.Write.Offset)

	return
}

func writeInteger(ctx *types.PDFContext, obj interface{}, validate func(int) bool) (ip *types.PDFInteger, written bool, err error) {

	logInfoWriter.Printf("writeInteger begin: offset=%d\n", ctx.Write.Offset)

	obj, written, err = writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		return
	}

	if obj == nil {
		err = errors.New("writeInteger: missing object")
		return
	}

	i, ok := obj.(types.PDFInteger)
	if !ok {
		err = errors.New("writeInteger: invalid type")
		return
	}

	// Validation
	if validate != nil && !validate(i.Value()) {
		err = errors.Errorf("writeInteger: invalid integer: %s\n", i)
		return
	}

	ip = &i

	logInfoWriter.Printf("writeInteger end: offset=%d\n", ctx.Write.Offset)

	return
}

func writeFloat(ctx *types.PDFContext, obj interface{}, validate func(float64) bool) (fp *types.PDFFloat, written bool, err error) {

	// TODO written irrelevant?

	logInfoWriter.Printf("writeFloat begin: offset=%d\n", ctx.Write.Offset)

	obj, written, err = writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		return
	}

	if obj == nil {
		err = errors.New("writeFloat: missing object")
		return
	}

	f, ok := obj.(types.PDFFloat)
	if !ok {
		err = errors.New("writeFloat: invalid type")
		return
	}

	// Validation
	if validate != nil && !validate(f.Value()) {
		err = errors.Errorf("writeFloat: invalid float: %s\n", f)
		return
	}

	fp = &f

	logInfoWriter.Printf("writeFloat end: offset=%d\n", ctx.Write.Offset)

	return
}

func writeNumber(ctx *types.PDFContext, obj interface{}) (n interface{}, written bool, err error) {

	// TODO written irrelevant?

	logInfoWriter.Printf("writeNumber begin: offset=%d\n", ctx.Write.Offset)

	n, written, err = writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		return
	}

	if obj == nil {
		err = errors.New("writeNumber: missing object")
		return
	}

	switch n.(type) {

	case types.PDFInteger:
		// no further processing.

	case types.PDFFloat:
		// no further processing.

	default:
		err = errors.New("writeNumber: invalid type")

	}

	logInfoWriter.Printf("writeNumber end: offset=%d\n", ctx.Write.Offset)

	return
}

func writeName(ctx *types.PDFContext, obj interface{}, validate func(string) bool) (namep *types.PDFName, written bool, err error) {

	// TODO written irrelevant?

	logInfoWriter.Printf("writeName begin: offset=%d\n", ctx.Write.Offset)

	obj, written, err = writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		return
	}

	if obj == nil {
		err = errors.New("writeName: missing object")
		return
	}

	name, ok := obj.(types.PDFName)
	if !ok {
		err = errors.New("writeName: invalid type")
		return
	}

	// Validation
	if validate != nil && !validate(name.String()) {
		err = errors.Errorf("writeName: invalid name: %s\n", name)
		return
	}

	namep = &name

	logInfoWriter.Printf("writeName end: offset=%d\n", ctx.Write.Offset)

	return
}

func writeDict(ctx *types.PDFContext, obj interface{}) (dictp *types.PDFDict, written bool, err error) {

	logInfoWriter.Printf("writeDict begin: offset=%d\n", ctx.Write.Offset)

	obj, written, err = writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		return
	}

	if obj == nil {
		err = errors.New("writeDict: missing object")
		return
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		err = errors.New("writeDict: invalid type")
		return
	}

	dictp = &dict

	logInfoWriter.Printf("writeDict end: offset=%d\n", ctx.Write.Offset)

	return
}

func writeStreamDict(ctx *types.PDFContext, obj interface{}) (streamDictp *types.PDFStreamDict, written bool, err error) {

	logInfoWriter.Printf("writeStreamDict begin: offset=%d\n", ctx.Write.Offset)

	obj, written, err = writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		return
	}

	if obj == nil {
		err = errors.New("writeStreamDict: missing object")
		return
	}

	streamDict, ok := obj.(types.PDFStreamDict)
	if !ok {
		err = errors.New("writeDict: invalid type")
		return
	}

	streamDictp = &streamDict

	logInfoWriter.Printf("writeStreamDict endobj: offset=%d\n", ctx.Write.Offset)

	return
}

func writeArray(ctx *types.PDFContext, obj interface{}) (arrp *types.PDFArray, written bool, err error) {

	logInfoWriter.Printf("writeArray begin: offset=%d\n", ctx.Write.Offset)

	obj, written, err = writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		return
	}

	if obj == nil {
		err = errors.New("writeArray: missing object")
		return
	}

	arr, ok := obj.(types.PDFArray)
	if !ok {
		err = errors.New("writeArray: invalid type")
		return
	}

	arrp = &arr

	logInfoWriter.Printf("writeArray end: offset=%d\n", ctx.Write.Offset)

	return
}

func writeNameArray(ctx *types.PDFContext, obj interface{}) (arrp *types.PDFArray, written bool, err error) {

	logInfoWriter.Printf("writeNameArray begin: offset=%d\n", ctx.Write.Offset)

	arrp, written, err = writeArray(ctx, obj)
	if err != nil {
		return
	}

	if written || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		obj, written, err = writeObject(ctx, obj)
		if err != nil {
			return
		}

		if written || obj == nil {
			continue
		}

		_, ok := obj.(types.PDFName)
		if !ok {
			err = errors.Errorf("writeNumberArray: invalid type at index %d\n", i)
			return
		}

	}

	logInfoWriter.Printf("writeNameArray end: offset=%d\n", ctx.Write.Offset)

	return
}

func writeIntegerArray(ctx *types.PDFContext, arr types.PDFArray) (arrp *types.PDFArray, written bool, err error) {

	logInfoWriter.Printf("writeIntegerArray begin: offset=%d\n", ctx.Write.Offset)

	arrp, written, err = writeArray(ctx, arr)
	if err != nil {
		return
	}

	if written || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		obj, written, err = writeObject(ctx, obj)
		if err != nil {
			return
		}

		if written || obj == nil {
			continue
		}

		switch obj.(type) {

		case types.PDFInteger:
			// no further processing.

		default:
			err = errors.Errorf("writeIntegerArray: invalid type at index %d\n", i)
		}

	}

	logInfoWriter.Printf("writeIntegerArray end: offset=%d\n", ctx.Write.Offset)

	return
}

func writeNumberArray(ctx *types.PDFContext, obj interface{}) (arrp *types.PDFArray, written bool, err error) {

	logInfoWriter.Printf("writeNumberArray begin: offset=%d\n", ctx.Write.Offset)

	arrp, written, err = writeArray(ctx, obj)
	if err != nil {
		return
	}

	if written || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		obj, written, err = writeObject(ctx, obj)
		if err != nil {
			return
		}

		if written || obj == nil {
			continue
		}

		switch obj.(type) {

		case types.PDFInteger:
			// no further processing.

		case types.PDFFloat:
			// no further processing.

		default:
			err = errors.Errorf("writeNumberArray: invalid type at index %d\n", i)
			return
		}

	}

	logInfoWriter.Printf("writeNumberArray end: offset=%d\n", ctx.Write.Offset)

	return
}

func writeAnyEntry(ctx *types.PDFContext, dict types.PDFDict, entryName string, required bool) (written bool, err error) {

	logInfoWriter.Printf("writeAnyEntry begin: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	entry, found := dict.Find(entryName)
	if !found || entry == nil {
		if required {
			err = errors.Errorf("writeAnyEntry: missing required entry: %s", entryName)
			return
		}
		logInfoWriter.Printf("writeAnyEntry end: entry %s not found or nil\n", entryName)
		return
	}

	indRef, ok := entry.(types.PDFIndirectRef)
	if !ok {
		logInfoWriter.Printf("writeAnyEntry end: offset=%d\n", ctx.Write.Offset)
		return
	}

	objNumber := int(indRef.ObjectNumber)
	genNumber := int(indRef.GenerationNumber)

	if ctx.Write.HasWriteOffset(objNumber) {
		logInfoWriter.Printf("*** writeAnyEntry end: object #%d already written. offset=%d ***\n", objNumber, ctx.Write.Offset)
		written = true
		return
	}

	obj, err := ctx.Dereference(indRef)
	if err != nil {
		return written, errors.Wrapf(err, "writeAnyEntry: unable to dereference object #%d", objNumber)
	}

	if obj == nil {
		return written, errors.Errorf("writeAnyEntry end: entry %s is nil", entryName)
	}

	logInfoWriter.Printf("writeAnyEntry: object #%d gets writeoffset: %d\n", objNumber, ctx.Write.Offset)

	switch obj := obj.(type) {

	case types.PDFDict:
		err = writePDFDictObject(ctx, objNumber, genNumber, obj)

	case types.PDFStreamDict:
		err = writePDFStreamDictObject(ctx, objNumber, genNumber, obj)

	case types.PDFArray:
		err = writePDFArrayObject(ctx, objNumber, genNumber, obj)

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
		err = errors.Errorf("writeAnyEntry: unsupported entry: %s", entryName)

	}

	logDebugWriter.Printf("writeAnyEntry: new offset = %d\n", ctx.Write.Offset)
	logInfoWriter.Printf("writeAnyEntry end: offset=%d\n", ctx.Write.Offset)

	return
}

func writeBooleanEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(bool) bool) (boolp *types.PDFBoolean, written bool, err error) {

	logInfoWriter.Printf("writeBooleanEntry begin: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("writeBooleanEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeBooleanEntry end: entry %s is nil\n", entryName)
		return
	}

	obj, written, err = writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeBooleanEntry end: entry %s already written\n", entryName)
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("writeBooleanEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeBooleanEntry end: entry %s is nil\n", entryName)
		return
	}

	b, ok := obj.(types.PDFBoolean)
	if !ok {
		err = errors.Errorf("writeBooleanEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		err = errors.Errorf("writeBooleanEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, ctx.VersionString())
		return
	}

	// Validation
	if validate != nil && !validate(b.Value()) {
		err = errors.Errorf("writeBooleanEntry: dict=%s entry=%s invalid name dict entry", dictName, entryName)
		return
	}

	boolp = &b

	logInfoWriter.Printf("writeBooleanEntry end: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	return
}

func writeStringEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(string) bool) (s *string, written bool, err error) {

	logInfoWriter.Printf("writeStringEntry begin: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("writeStringEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeStringEntry end: entry %s is nil\n", entryName)
		return
	}

	obj, written, err = writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeStringEntry end: entry %s already written\n", entryName)
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("writeStringEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeStringEntry end: optional entry %s is nil\n", entryName)
		return
	}

	var str string

	switch obj := obj.(type) {

	case types.PDFStringLiteral:
		str = obj.Value()

	case types.PDFHexLiteral:
		str = obj.Value()

	default:
		err = errors.Errorf("writeStringEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		err = errors.Errorf("writeStringEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, ctx.VersionString())
		return
	}

	// Validation
	if validate != nil && !validate(str) {
		err = errors.Errorf("writeStringEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
		return
	}

	s = &str

	logInfoWriter.Printf("writeStringEntry end: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	return
}

func writeDateEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion) (s *types.PDFStringLiteral, written bool, err error) {

	logInfoWriter.Printf("writeDateEntry begin: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("writeDateEntry: missing required entry: %s", entryName)
			return
		}
		logInfoWriter.Printf("writeDateEntry end: entry %s is nil\n", entryName)
		return
	}

	obj, written, err = writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeDateEntry end: entry %s already written\n", entryName)
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("writeDateEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeDateEntry end: optional entry %s is nil\n", entryName)
		return
	}

	date, ok := obj.(types.PDFStringLiteral)
	if !ok {
		err = errors.Errorf("writeDateEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		err = errors.Errorf("writeDateEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, ctx.VersionString())
		return
	}

	// Validation
	if ok := validate.Date(date.Value()); !ok {
		err = errors.Errorf("writeDateEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
		return
	}

	s = &date

	logInfoWriter.Printf("writeDateEntry end: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	return
}

func writeFloatEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(float64) bool) (fp *types.PDFFloat, written bool, err error) {

	logInfoWriter.Printf("writeFloatEntry begin: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("writeIntegerEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeFloatEntry end: entry %s is nil\n", entryName)
		return
	}

	obj, written, err = writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeFloatEntry end: entry %s already written\n", entryName)
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("writeFloatEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeFloatEntry end: optional entry %s is nil\n", entryName)
		return
	}

	f, ok := obj.(types.PDFFloat)
	if !ok {
		err = errors.Errorf("writeFloatEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		err = errors.Errorf("writeFloatEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, ctx.VersionString())
		return
	}

	// Validation
	if validate != nil && !validate(f.Value()) {
		err = errors.Errorf("writeFloatEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
		return
	}

	fp = &f

	logInfoWriter.Printf("writeFloatEntry end: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	return
}

func writeIntegerEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(int) bool) (ip *types.PDFInteger, written bool, err error) {

	logInfoWriter.Printf("writeIntegerEntry begin: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("writeIntegerEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeIntegerEntry end: optional entry %s not found or nil\n", entryName)
		return
	}

	obj, written, err = writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeIntegerEntry end: entry %s already written\n", entryName)
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("writeIntegerEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeIntegerEntry end: optional entry %s is nil\n", entryName)
		return
	}

	i, ok := obj.(types.PDFInteger)
	if !ok {
		err = errors.Errorf("writeIntegerEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		err = errors.Errorf("writeIntegerEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, ctx.VersionString())
		return
	}

	// Validation
	if validate != nil && !validate(i.Value()) {
		err = errors.Errorf("writeIntegerEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
		return
	}

	ip = &i

	logInfoWriter.Printf("writeIntegerEntry end: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	return
}

func writeNumberEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion) (obj interface{}, written bool, err error) {

	logInfoWriter.Printf("writeNumberEntry begin: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("writeNumberEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeNumberEntry end: optional entry %s not found or nil\n", entryName)
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		err = errors.Errorf("writeNumberEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, ctx.VersionString())
		return
	}

	obj, written, err = writeNumber(ctx, obj)

	logInfoWriter.Printf("writeNumberEntry end: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	return
}

func writeNameEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(string) bool) (namep *types.PDFName, written bool, err error) {

	logInfoWriter.Printf("writeNameEntry begin: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("writeNameEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeNameEntry end: optional entry %s not found or nil\n", entryName)
		return
	}

	obj, written, err = writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeNameEntry end: entry %s already written\n", entryName)
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("writeNameEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeNameEntry end: optional entry %s is nil\n", entryName)
		return
	}

	name, ok := obj.(types.PDFName)
	if !ok {
		err = errors.Errorf("writeNameEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		err = errors.Errorf("writeNameEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, ctx.VersionString())
		return
	}

	// Validation
	if validate != nil && !validate(name.String()) {
		err = errors.Errorf("writeNameEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
		return
	}

	namep = &name

	logInfoWriter.Printf("writeNameEntry end: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	return
}

func writeDictEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFDict) bool) (dictp *types.PDFDict, written bool, err error) {

	logInfoWriter.Printf("writeDictEntry begin: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("writeDictEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeDictEntry end: optional entry %s not found or nil\n", entryName)
		return
	}

	obj, written, err = writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeDictEntry end: entry %s already written\n", entryName)
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("writeDictEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeDictEntry end: optional entry %s is nil\n", entryName)
		return
	}

	d, ok := obj.(types.PDFDict)
	if !ok {
		err = errors.Errorf("writeDictEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		err = errors.Errorf("writeDictEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, ctx.VersionString())
		return
	}

	// Validation
	if validate != nil && !validate(d) {
		err = errors.Errorf("writeDictEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	dictp = &d

	logInfoWriter.Printf("writeDictEntry end: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	return
}

func writeStreamDictEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFStreamDict) bool) (sdp *types.PDFStreamDict, written bool, err error) {

	logInfoWriter.Printf("writeStreamDictEntry begin: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("writeStreamDictEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeStreamDictEntry end: optional entry %s not found or nil\n", entryName)
		return
	}

	obj, written, err = writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeStreamDictEntry end: entry %s already written\n", entryName)
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("writeStreamDictEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeStreamDictEntry end: optional entry %s is nil\n", entryName)
		return
	}

	sd, ok := obj.(types.PDFStreamDict)
	if !ok {
		err = errors.Errorf("writeStreamDictEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		err = errors.Errorf("writeStreamDictEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, ctx.VersionString())
		return
	}

	// Validation
	if validate != nil && !validate(sd) {
		err = errors.Errorf("writeStreamDictEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	sdp = &sd

	logInfoWriter.Printf("writeStreamDictEntry end: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	return
}

func writeFunctionEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion) (written bool, err error) {

	logInfoWriter.Printf("writeFunctionEntry begin: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("writeFunctionEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeFunctionEntry end: optional entry %s not found or nil\n", entryName)
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		err = errors.Errorf("writeFunctionEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, ctx.VersionString())
		return
	}

	written, err = writeFunction(ctx, obj)
	if err != nil {
		return
	}

	logInfoWriter.Printf("writeFunctionEntry end: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	return
}

func writeArrayEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFArray) bool) (arrp *types.PDFArray, written bool, err error) {

	logInfoWriter.Printf("writeArrayEntry begin: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("writeArrayEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeArrayEntry end: optional entry %s not found or nil\n", entryName)
		return
	}

	obj, written, err = writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeArrayEntry end: entry %s already written\n", entryName)
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("writeArrayEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoWriter.Printf("writeArrayEntry end: optional entry %s is nil\n", entryName)
		return
	}

	arr, ok := obj.(types.PDFArray)
	if !ok {
		err = errors.Errorf("writeArrayEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		err = errors.Errorf("writeArrayEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, ctx.VersionString())
		return
	}

	// Validation
	if validate != nil && !validate(arr) {
		err = errors.Errorf("writeArrayEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	arrp = &arr

	logInfoWriter.Printf("writeArrayEntry end: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	return
}

// Write an array of indRefs. All referenced objects must already be written.
func writeIndRefArrayEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFArray) bool) (arrp *types.PDFArray, written bool, err error) {

	logInfoWriter.Printf("writeIndRefArrayEntry begin: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	arrp, written, err = writeArrayEntry(ctx, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil {
		return
	}

	if written || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		indRef, ok := obj.(types.PDFIndirectRef)
		if !ok {
			err = errors.Errorf("writeIndRefArrayEntry: invalid type at index %d\n", i)
			return
		}

		objNumber := int(indRef.ObjectNumber)
		if !ctx.Write.HasWriteOffset(objNumber) {
			err = errors.Errorf("writeIndRefArrayEntry: entry at index %d has not been written.\n", i)
			return
		}

	}

	logInfoWriter.Printf("writeIndRefArrayEntry end: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	return
}

func writeStringArrayEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFArray) bool) (arrp *types.PDFArray, written bool, err error) {

	logInfoWriter.Printf("writeStringArrayEntry begin: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	arrp, written, err = writeArrayEntry(ctx, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil {
		return
	}

	if written || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		obj, written, err = writeObject(ctx, obj)
		if err != nil {
			return
		}

		if written || obj == nil {
			continue
		}

		switch obj.(type) {

		case types.PDFStringLiteral:
			// no further processing.

		case types.PDFHexLiteral:
			// no further processing

		default:
			err = errors.Errorf("writeStringArrayEntry: invalid type at index %d\n", i)
			return
		}

	}

	logInfoWriter.Printf("writeStringArrayEntry end: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	return
}

// TODO validate does not validate array.
func writeNameArrayEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(string) bool) (arrp *types.PDFArray, written bool, err error) {

	logInfoWriter.Printf("writeNameArrayEntry begin: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	arrp, written, err = writeArrayEntry(ctx, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil {
		return
	}

	if written || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		obj, written, err = writeObject(ctx, obj)
		if err != nil {
			return
		}

		if written || obj == nil {
			continue
		}

		name, ok := obj.(types.PDFName)
		if !ok {
			err = errors.Errorf("writeNameArrayEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
			return
		}

		if validate != nil && !validate(name.String()) {
			err = errors.Errorf("writeNameArrayEntry: dict=%s entry=%s invalid entry at index %d\n", dictName, entryName, i)
			return
		}

	}

	logInfoWriter.Printf("writeNameArrayEntry end: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	return
}

func writeBooleanArrayEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFArray) bool) (arrp *types.PDFArray, written bool, err error) {

	logInfoWriter.Printf("writeBooleanArrayEntry begin: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	arrp, written, err = writeArrayEntry(ctx, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil {
		return
	}

	if written || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		obj, written, err = writeObject(ctx, obj)
		if err != nil {
			return
		}

		if written || obj == nil {
			continue
		}

		_, ok := obj.(types.PDFBoolean)
		if !ok {
			err = errors.Errorf("writeBooleanArrayEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
			return
		}

	}

	logInfoWriter.Printf("writeBooleanArrayEntry end: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	return
}

func writeIntegerArrayEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFArray) bool) (arrp *types.PDFArray, written bool, err error) {

	logInfoWriter.Printf("writeIntegerArrayEntry begin: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	arrp, written, err = writeArrayEntry(ctx, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil {
		return
	}

	if written || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		obj, written, err = writeObject(ctx, obj)
		if err != nil {
			return
		}

		if written || obj == nil {
			continue
		}

		_, ok := obj.(types.PDFInteger)
		if !ok {
			err = errors.Errorf("writeIntegerArrayEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
			return
		}

	}

	logInfoWriter.Printf("writeIntegerArrayEntry end: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	return
}

func writeNumberArrayEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFArray) bool) (arrp *types.PDFArray, written bool, err error) {

	logInfoWriter.Printf("writeNumberArrayEntry begin: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	arrp, written, err = writeArrayEntry(ctx, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil {
		return
	}

	if written || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		obj, written, err = writeObject(ctx, obj)
		if err != nil {
			return
		}

		if written || obj == nil {
			continue
		}

		switch obj.(type) {

		case types.PDFInteger:
			// no further processing.

		case types.PDFFloat:
			// no further processing.

		default:
			err = errors.Errorf("writeNumberArrayEntry: invalid type at index %d\n", i)
			return
		}

	}

	logInfoWriter.Printf("writeNumberArrayEntry end: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	return
}

func writeFunctionArrayEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFArray) bool) (arrp *types.PDFArray, written bool, err error) {

	logInfoWriter.Printf("writeFunctionArrayEntry begin: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	arrp, written, err = writeArrayEntry(ctx, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil {
		return
	}

	if written || arrp == nil {
		return
	}

	for _, obj := range *arrp {
		_, err = writeFunction(ctx, obj)
		if err != nil {
			return
		}
	}

	logInfoWriter.Printf("writeFunctionArrayEntry end: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	return
}

func writeRectangleEntry(ctx *types.PDFContext, dict types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFArray) bool) (arrp *types.PDFArray, written bool, err error) {

	logInfoWriter.Printf("writeRectangleEntry begin: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	arrp, written, err = writeNumberArrayEntry(ctx, dict, dictName, entryName, required, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 4 })
	if err != nil {
		return
	}

	if written || arrp == nil {
		return
	}

	if validate != nil && !validate(*arrp) {
		err = errors.Errorf("writeRectangleEntry: dict=%s entry=%s invalid rectangle entry", dictName, entryName)
		return
	}

	logInfoWriter.Printf("writeRectangleEntry end: entry=%s offset=%d\n", entryName, ctx.Write.Offset)

	return
}
