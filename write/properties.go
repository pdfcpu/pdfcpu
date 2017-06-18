package write

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

func writePropertiesDict(ctx *types.PDFContext, dict types.PDFDict) (err error) {

	// see 14.6.2
	// a dictionary containing private information meaningful to the conforming writer creating marked content.

	// anything possible +

	// empty dict ok
	// Optional Metadata entry ok
	// Optional Contents entry ok
	// Optional Resources entry ok

	// Optional content group /OCG see 8.11.2
	// Optional content membership dict. /OCMD see 8.11.2.2
	// Optional MCID integer entry
	// Optional Alt since 1.5 see 14.9.3
	// Optional ActualText since 1.5 see 14.9.4
	// Optional E see since 1.4 14.9.5
	// Optional Lang string RFC 3066 see 14.9.2

	logInfoWriter.Printf("*** writePropertiesDict begin: offset=%d ***\n", ctx.Write.Offset)

	// Relaxed to V1.3
	_, err = writeMetadata(ctx, dict, OPTIONAL, types.V13)
	if err != nil {
		return
	}

	for key, val := range dict.Dict {

		logInfoWriter.Printf("writePropertiesDict: key=%s val=%v\n", key, val)

		switch key {

		case "Metadata":
			logInfoWriter.Printf("writePropertiesDict: recognized key \"%s\"\n", key)
			// see above

		case "Contents":
			logInfoWriter.Printf("writePropertiesDict: recognized key \"%s\"\n", key)
			//indRef, ok := val.(PDFIndirectRef)
			//if !ok {
			//	log.Fatalf("writePropertiesDict: currupt entry \"%s\"\n", key)
			//}
			_, _, err = writeStreamDict(ctx, val)
			if err != nil {
				return
			}

		case "Resources":
			logInfoWriter.Printf("writePropertiesDict: recognized key \"%s\"\n", key)
			_, err = writeResourceDict(ctx, val)
			if err != nil {
				return
			}

		case "OCG":
			return errors.Errorf("writePropertiesDict: recognized unsupported key \"%s\"\n", key)

		case "OCMD":
			return errors.Errorf("writePropertiesDict: recognized unsupported key \"%s\"\n", key)

		//case "MCID": -> default
		//case "Alt": -> default
		//case "ActualText": -> default
		//case "E": -> default
		//case "Lang": -> default

		default:
			logInfoWriter.Printf("writePropertiesDict: processing unrecognized key \"%s\"\n", key)
			_, _, err = writeObject(ctx, val)
			if err != nil {
				return
			}
		}

	}

	logInfoWriter.Printf("*** writePropertiesDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}

func writePropertiesResourceDict(ctx *types.PDFContext, obj interface{}) (err error) {

	logInfoWriter.Printf("*** writePropertiesResourceDict begin: offset=%d ***\n", ctx.Write.Offset)

	obj, written, err := writeObject(ctx, obj)
	if err != nil {
		return
	}

	if written {
		logInfoWriter.Printf("writePropertiesResourceDict end: object already written. offset=%d\n", ctx.Write.Offset)
		return
	}

	if obj == nil {
		logInfoWriter.Printf("writePropertiesResourceDict end: object is nil. offset=%d\n", ctx.Write.Offset)
		return
	}

	// Version check
	if ctx.Version() < types.V12 {
		return errors.Errorf("writePropertiesResourceDict: unsupported in version %s.\n", ctx.VersionString())
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		return errors.New("writePropertiesResourceDict: corrupt dict")
	}

	// Iterate over properties resource dict
	for _, obj := range dict.Dict {

		obj, written, err = writeObject(ctx, obj)
		if written {
			logInfoWriter.Printf("writePropertiesResourceDict end: resource object already written. offset=%d\n", ctx.Write.Offset)
			continue
		}

		if obj == nil {
			logInfoWriter.Printf("writePropertiesResourceDict end: resource object is nil. offset=%d\n", ctx.Write.Offset)
			continue
		}

		dict, ok := obj.(types.PDFDict)
		if !ok {
			return errors.New("writePropertiesResourceDict: corrupt properties resource dict")
		}

		// Process propDict
		err = writePropertiesDict(ctx, dict)
		if err != nil {
			return
		}
	}

	logInfoWriter.Printf("*** writePropertiesResourceDict end: offset=%d ***\n", ctx.Write.Offset)

	return
}
