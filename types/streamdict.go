package types

import (
	"fmt"

	"github.com/hhrutter/pdfcpu/log"
	"github.com/pkg/errors"
)

// PDFFilter represents a PDF stream filter object.
type PDFFilter struct {
	Name        string
	DecodeParms *PDFDict
}

// PDFStreamDict represents a PDF stream dict object.
type PDFStreamDict struct {
	PDFDict
	StreamOffset      int64
	StreamLength      *int64
	StreamLengthObjNr *int
	FilterPipeline    []PDFFilter
	Raw               []byte // Encoded
	Content           []byte // Decoded
	IsPageContent     bool
}

// NewPDFStreamDict creates a new PDFStreamDict for given PDFDict, stream offset and length.
func NewPDFStreamDict(pdfDict PDFDict, streamOffset int64, streamLength *int64, streamLengthObjNr *int,
	filterPipeline []PDFFilter) PDFStreamDict {
	return PDFStreamDict{pdfDict, streamOffset, streamLength, streamLengthObjNr, filterPipeline, nil, nil, false}
}

// HasSoleFilterNamed returns true if there is exactly one filter defined for a stream dict.
func (streamDict PDFStreamDict) HasSoleFilterNamed(filterName string) bool {

	fpl := streamDict.FilterPipeline
	if fpl == nil {
		return false
	}

	if len(fpl) != 1 {
		return false
	}

	soleFilter := fpl[0]

	return soleFilter.Name == filterName
}

// PDFObjectStreamDict represents a object stream dictionary.
type PDFObjectStreamDict struct {
	PDFStreamDict
	Prolog         []byte
	ObjCount       int
	FirstObjOffset int
	ObjArray       PDFArray
}

// NewPDFObjectStreamDict creates a new PDFObjectStreamDict object.
func NewPDFObjectStreamDict() *PDFObjectStreamDict {

	streamDict := PDFStreamDict{PDFDict: NewPDFDict()}

	streamDict.Insert("Type", PDFName("ObjStm"))
	streamDict.Insert("Filter", PDFName("FlateDecode"))

	streamDict.FilterPipeline = []PDFFilter{{Name: "FlateDecode", DecodeParms: nil}}

	return &PDFObjectStreamDict{PDFStreamDict: streamDict}
}

// IndexedObject returns the object at given index from a PDFObjectStreamDict.
func (oStreamDict *PDFObjectStreamDict) IndexedObject(index int) (PDFObject, error) {
	if oStreamDict.ObjArray == nil {
		return nil, errors.Errorf("IndexedObject(%d): object not available", index)
	}
	return oStreamDict.ObjArray[index], nil
}

// AddObject adds another object to this object stream.
// Relies on decoded content!
func (oStreamDict *PDFObjectStreamDict) AddObject(objNumber int, entry *XRefTableEntry) error {

	offset := len(oStreamDict.Content)

	s := ""
	if oStreamDict.ObjCount > 0 {
		s = " "
	}
	s = s + fmt.Sprintf("%d %d", objNumber, offset)

	oStreamDict.Prolog = append(oStreamDict.Prolog, []byte(s)...)

	var pdfString string

	switch obj := entry.Object.(type) {

	case PDFDict:
		pdfString = obj.PDFString()

	case PDFArray:
		pdfString = obj.PDFString()

	case PDFInteger:
		pdfString = obj.PDFString()

	case PDFFloat:
		pdfString = obj.PDFString()

	case PDFStringLiteral:
		pdfString = obj.PDFString()

	case PDFHexLiteral:
		pdfString = obj.PDFString()

	case PDFBoolean:
		pdfString = obj.PDFString()

	case PDFName:
		pdfString = obj.PDFString()

	default:
		return errors.Errorf("AddObject: undefined PDF object #%d\n", objNumber)

	}

	oStreamDict.Content = append(oStreamDict.Content, []byte(pdfString)...)
	oStreamDict.ObjCount++

	log.Debug.Printf("AddObject end : ObjCount:%d prolog = <%s> Content = <%s>\n", oStreamDict.ObjCount, oStreamDict.Prolog, oStreamDict.Content)

	return nil
}

// Finalize prepares the final content of the objectstream.
func (oStreamDict *PDFObjectStreamDict) Finalize() {
	oStreamDict.Content = append(oStreamDict.Prolog, oStreamDict.Content...)
	oStreamDict.FirstObjOffset = len(oStreamDict.Prolog)
	log.Debug.Printf("Finalize : firstObjOffset:%d Content = <%s>\n", oStreamDict.FirstObjOffset, oStreamDict.Content)
}

// PDFXRefStreamDict represents a cross reference stream dictionary.
type PDFXRefStreamDict struct {
	PDFStreamDict
	Size           int
	Objects        []int
	W              [3]int
	PreviousOffset *int64
}

// NewPDFXRefStreamDict creates a new PDFXRefStreamDict object.
func NewPDFXRefStreamDict(ctx *PDFContext) *PDFXRefStreamDict {

	streamDict := PDFStreamDict{PDFDict: NewPDFDict()}

	streamDict.Insert("Type", PDFName("XRef"))
	streamDict.Insert("Filter", PDFName("FlateDecode"))
	streamDict.FilterPipeline = []PDFFilter{{Name: "FlateDecode", DecodeParms: nil}}

	streamDict.Insert("Root", *ctx.Root)

	if ctx.Info != nil {
		streamDict.Insert("Info", *ctx.Info)
	}

	if ctx.ID != nil {
		streamDict.Insert("ID", *ctx.ID)
	}

	if ctx.Encrypt != nil && ctx.EncKey != nil {
		streamDict.Insert("Encrypt", *ctx.Encrypt)
	}

	return &PDFXRefStreamDict{PDFStreamDict: streamDict}
}
