package write

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

func writeSignatureDict(ctx *types.PDFContext, obj interface{}) (err error) {

	logInfoWriter.Printf("*** writeSignatureDict begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeSignatureDict end: already written offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeSignatureDict end: object is nil offset=%d\n", ctx.Write.Offset)
		return
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		return errors.New("writeSignatureDict: not a dict")
	}

	// process signature dict fields.
	if dict.Type() != nil && *dict.Type() != "Sig" {
		return errors.New("writeSignatureDict: type must be \"Sig\"")
	}

	logInfoWriter.Printf("*** writeSignatureDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeAppearanceSubDict(ctx *types.PDFContext, subDict types.PDFDict) (err error) {

	logInfoWriter.Printf("*** writeAppearanceSubDict begin: offset=%d ***\n", ctx.Write.Offset)

	// dict of stream objects.
	for _, obj := range subDict.Dict {

		obj, written, err := writeObject(ctx, obj)
		if err != nil {
			return err
		}

		if written {
			continue
		}

		if obj == nil {
			continue
		}

		sd, ok := obj.(types.PDFStreamDict)
		if !ok {
			return errors.New("writeAppearanceSubDict: dereferenced sub dict entry not a stream dict")
		}

		err = writeXObjectStreamDict(ctx, sd)
		if err != nil {
			return err
		}

	}

	logInfoWriter.Printf("*** writeAppearanceSubDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeAppearanceDictEntry(ctx *types.PDFContext, obj interface{}) (err error) {

	// stream or dict
	// single appearance stream or subdict

	logInfoWriter.Printf("*** writeAppearanceDictEntry begin: offset=%d *** \n", ctx.Write.Offset)

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeAppearanceDictEntry end: offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeAppearanceDictEntry end: offset=%d\n", ctx.Write.Offset)
		return
	}

	switch obj := obj.(type) {

	case types.PDFDict:
		err = writeAppearanceSubDict(ctx, obj)

	case types.PDFStreamDict:
		err = writeXObjectStreamDict(ctx, obj)

	default:
		err = errors.New("writeAppearanceDictEntry: unsupported PDF object")

	}

	logInfoWriter.Printf("*** writeAppearanceDictEntry end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeAppearanceDict(ctx *types.PDFContext, obj interface{}) (err error) {

	logInfoWriter.Printf("*** writeAppearanceDict begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeAppearanceDictEntry end: offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeAppearanceDictEntry end: offset=%d\n", ctx.Write.Offset)
		return
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		return errors.New("writeAppearanceDict: not a dict")
	}

	// N, required, stream or dict, the annotation's normal appearance
	obj, ok = dict.Find("N")
	if !ok {
		if ctx.XRefTable.ValidationMode == types.ValidationStrict {
			return errors.New("writeAppearanceDict: missing required entry \"N\"")
		}
	} else {
		err = writeAppearanceDictEntry(ctx, obj)
		if err != nil {
			return
		}
	}

	// R, optional, stream or dict, the annotation's rollover appearance
	if obj, ok = dict.Find("R"); ok {
		err = writeAppearanceDictEntry(ctx, obj)
		if err != nil {
			return
		}
	}

	// D, optional, stream or dict, the annotation's down appearance
	if obj, ok = dict.Find("D"); ok {
		err = writeAppearanceDictEntry(ctx, obj)
		if err != nil {
			return
		}
	}

	logInfoWriter.Printf("*** writeAppearanceDict end: offset=%d **\n", ctx.Write.Offset)

	return
}

func writeAcroFieldDictEntries(ctx *types.PDFContext, dict types.PDFDict, terminalNode bool, inFieldType *types.PDFName) (outFieldType *types.PDFName, err error) {

	logInfoWriter.Printf("*** writeAcroFieldDictEntries begin: offset=%d ***\n", ctx.Write.Offset)

	// FT, name, required for terminal fields, Btn,Tx,Ch,Sig
	fieldType := dict.PDFNameEntry("FT")
	if terminalNode && fieldType == nil && inFieldType == nil {
		return nil, errors.New("writeAcroFieldDictEntries: missing field type")
	}

	logInfoWriter.Printf("writeAcroFieldDictEntries moving on, inFieldType=%v outFieldType=%v", inFieldType, outFieldType)

	if fieldType != nil {

		outFieldType = fieldType
		logInfoWriter.Printf("writeAcroFieldDictEntries moving on 2, inFieldType=%v outFieldType=%v", inFieldType, outFieldType)

		switch *fieldType {

		case "Btn": // Button field

			//	<DA, (/ZaDb 0 Tf 0 g)>
			//	<FT, Btn>
			//	<Ff, 49152>
			//	<Kids, [(257 0 R) (256 0 R) (255 0 R)]>
			//	<T, (Art)>

			//log.Fatalln("writeFieldDictEntries: Btn not supported.")

		case "Tx": // Text field
			//log.Fatalln("writeFieldDictEntries: Tx not supported.")

		case "Ch": // Choice field
			return nil, errors.New("writeAcroFieldDictEntries: \"Ch\" not supported")

		case "Sig": // Signature field

			if _, ok := dict.Find("Lock"); ok {
				return nil, errors.New("writeAcroFieldDictEntries: \"Lock\" not supported")
			}

			if _, ok := dict.Find("SV"); ok {
				return nil, errors.New("writeAcroFieldDictEntries: \"SV\" not supported")
			}

			// V, optional, signature dictionary containing the signature and specifying various attributes of the signature field.
			if obj, ok := dict.Find("V"); ok {
				err = writeSignatureDict(ctx, obj)
				if err != nil {
					return
				}
			}

			// DV, optional, defaultvalue, like V
			if obj, ok := dict.Find("DV"); ok {
				err = writeSignatureDict(ctx, obj)
				if err != nil {
					return
				}
			}

			if obj, ok := dict.Find("AP"); ok {
				err = writeAppearanceDict(ctx, obj)
				if err != nil {
					return
				}
			}

		default:
			return nil, errors.Errorf("writeAcroFieldDictEntries: unknown fieldType:%s\n", fieldType)
		}
	}

	// Parent, dict, required for kid fields.
	_, _, err = writeIndRefEntry(ctx, dict, "acroFieldDict", "Parent", OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// T, optional, text string

	// TU, optional, text string, since V1.3

	// TM, optional, text string, since V1.3

	// Ff, optional, integer

	// V, optional, various

	// DV, optional, various

	// AA, optional, dict, since V1.2
	_, err = writeAdditionalActions(ctx, &dict, "acroFieldDict", "AA", OPTIONAL, types.V12, "fieldOrAnnot")
	if err != nil {
		return
	}

	logInfoWriter.Printf("*** writeAcroFieldDictEntries end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeAcroFieldDict(ctx *types.PDFContext, indRef *types.PDFIndirectRef, inFieldType *types.PDFName) (err error) {

	objNr := int(indRef.ObjectNumber)

	logInfoWriter.Printf("*** writeAcroFieldDict begin: obj#:%d offset=%d ***\n", objNr, ctx.Write.Offset)

	obj, written, err := writeIndRef(ctx, *indRef)
	if err != nil {
		return err
	}

	if written {
		logInfoWriter.Println("writeAcroFieldDict: already written.")
		return nil
	}

	if obj == nil {
		logInfoWriter.Println("writeAcroFieldDict: is nil")
		return nil
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		return errors.New("writeAcroFieldDict: corrupt field dict: dereferenced obj must be dict")
	}

	if pdfObject, ok := dict.Find("Kids"); ok {

		// dict represents a non terminal field.
		if dict.Subtype() != nil && *dict.Subtype() == "Widget" {
			return errors.New("writeAcroFieldDict: non terminal field can not be widget annotation")
		}

		// Write field entries.
		xinFieldType, err := writeAcroFieldDictEntries(ctx, dict, false, inFieldType)
		if err != nil {
			return err
		}

		// Recurse over kids.
		obj, written, err := writeObject(ctx, pdfObject)
		if err != nil {
			return err
		}

		if written || obj == nil {
			return nil
		}

		arr, ok := obj.(types.PDFArray)
		if !ok {
			return errors.New("writeAcroFieldDict: corrupt kids array: must be array")
		}

		for _, value := range arr {

			indRef, ok := value.(types.PDFIndirectRef)
			if !ok {
				return errors.New("writeAcroFieldDict: corrupt kids array: entries must be indirect reference")
			}

			err = writeAcroFieldDict(ctx, &indRef, xinFieldType)
			if err != nil {
				return err
			}

		}

	} else {

		// dict represents a terminal field.

		if dict.Subtype() == nil || *dict.Subtype() != "Widget" {
			return errors.New("writeAcroFieldDict: terminal field must be widget annotation")
		}

		// Write field entries.
		_, err = writeAcroFieldDictEntries(ctx, dict, true, inFieldType)
		if err != nil {
			return
		}

		// Write widget annotation.
		err = writeAnnotationDict(ctx, dict)
		if err != nil {
			return
		}
	}

	logInfoWriter.Printf("*** writeAcroFieldDict end: obj#:%d offset=%d ***\n", indRef.ObjectNumber, ctx.Write.Offset)

	return
}

func writeAcroFormFields(ctx *types.PDFContext, pdfObject interface{}) (err error) {

	logInfoWriter.Printf("*** writeAcroFormFields begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeObject(ctx, pdfObject)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeAcroFormFields end: already written offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writeAcroFormFields end: is nil offset=%d\n", ctx.Write.Offset)
		return
	}

	arr, ok := obj.(types.PDFArray)
	if !ok {
		return errors.New("writeAcroFormFields: corrupt form field array: obj must be array")
	}

	for _, value := range arr {

		indRef, ok := value.(types.PDFIndirectRef)
		if !ok {
			return errors.New("writeAcroFormFields: corrupt form field array entry")
		}

		err = writeAcroFieldDict(ctx, &indRef, nil)
		if err != nil {
			return
		}

	}

	logInfoWriter.Printf("*** writeAcroFormFields end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeAcroFormEntryCO(ctx *types.PDFContext, obj interface{}, sinceVersion types.PDFVersion) (err error) {
	// since V1.3
	// Array of indRefs to field dicts with calculation actions.
	// => 12.6.3 Trigger Events

	logInfoWriter.Printf("*** writeAcroFormEntryCO begin: offset=%d ***\n", ctx.Write.Offset)

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeAcroFormEntryCO: unsupported in version %s.\n", ctx.VersionString())
	}

	var (
		arr  *types.PDFArray
		dict *types.PDFDict
	)

	arr, _, err = writeArray(ctx, obj)
	if err != nil {
		return
	}

	for _, obj := range *arr {

		dict, err = ctx.DereferenceDict(obj)
		if err != nil || dict == nil {
			return
		}

		err = writeAnnotationDict(ctx, *dict)
		if err != nil {
			return
		}

	}

	logInfoWriter.Printf("*** writeAcroFormEntryCO end: offset=%d ***\n", ctx.Write.Offset)

	return
}

// TODO implement
func writeAcroFormEntryXFA(ctx *types.PDFContext, obj interface{}, sinceVersion types.PDFVersion) (err error) {

	logInfoWriter.Printf("*** writeAcroFormEntryXFA begin: offset=%d ***\n", ctx.Write.Offset)

	err = errors.New("*** writeAcroFormEntryXFA unsupported ***")

	logInfoWriter.Printf("*** writeAcroFormEntryXFA end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writeAcroForm(ctx *types.PDFContext, rootDict types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 12.7.2 Interactive Form Dictionary

	logInfoWriter.Printf("*** writeAcroForm begin: offset=%d ***\n", ctx.Write.Offset)

	dict, written, err := writeDictEntry(ctx, rootDict, "rootDict", "AcroForm", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writeAcroForm end: dict already written. offset=%d\n", ctx.Write.Offset)
		return
	}

	if dict == nil {
		logInfoWriter.Printf("writeAcroForm end: dict is nil.\n")
		return
	}

	// Version check
	if ctx.Version() < sinceVersion {
		return errors.Errorf("writeAcroForm: unsupported in version %s.\n", ctx.VersionString())
	}

	// Fields, required, array of indirect references
	obj, ok := dict.Find("Fields")
	if !ok {
		return errors.New("writeAcroForm: missing required entry \"Fields\"")
	}

	err = writeAcroFormFields(ctx, obj)
	if err != nil {
		return
	}

	dictName := "acroFormDict"

	// NeedAppearances: optional, boolean
	_, _, err = writeBooleanEntry(ctx, *dict, dictName, "NeedAppearances", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// SigFlags: optional, since 1.3, integer
	_, _, err = writeIntegerEntry(ctx, *dict, dictName, "SigFlags", OPTIONAL, types.V13, nil)
	if err != nil {
		return
	}

	// CO: array
	// TODO since 1.3, required if any fields in the document have additional-actions dictionaries containing a C entry.ObjectStream
	if obj, ok := dict.Find("CO"); ok {
		err = writeAcroFormEntryCO(ctx, obj, types.V13)
		if err != nil {
			return
		}
	}

	// DR, optional, resource dict
	if obj, ok := dict.Find("DR"); ok {
		_, err = writeResourceDict(ctx, obj)
		if err != nil {
			return
		}
	}

	// DA: optional, string
	_, _, err = writeStringEntry(ctx, *dict, dictName, "DA", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Q: optional, integer
	_, _, err = writeIntegerEntry(ctx, *dict, dictName, "Q", OPTIONAL, types.V10, func(i int) bool { return i >= 0 && i <= 2 })
	if err != nil {
		return
	}

	// TODO XFA: optional, since 1.5, stream or array
	if obj, ok := dict.Find("XFA"); ok {
		err = writeAcroFormEntryXFA(ctx, obj, types.V15)
		if err != nil {
			return
		}
	}

	ctx.Stats.AddRootAttr(types.RootAcroForm)

	logInfoWriter.Printf("*** writeAcroForm end: offset=%d ***\n", ctx.Write.Offset)

	return
}
