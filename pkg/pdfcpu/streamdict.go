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

	"github.com/hhrutter/pdfcpu/pkg/filter"
	"github.com/hhrutter/pdfcpu/pkg/log"
	"github.com/pkg/errors"
)

// PDFFilter represents a PDF stream filter object.
type PDFFilter struct {
	Name        string
	DecodeParms Dict
}

// StreamDict represents a PDF stream dict object.
type StreamDict struct {
	Dict
	StreamOffset      int64
	StreamLength      *int64
	StreamLengthObjNr *int
	FilterPipeline    []PDFFilter
	Raw               []byte // Encoded
	Content           []byte // Decoded
	IsPageContent     bool
}

// NewStreamDict creates a new PDFStreamDict for given PDFDict, stream offset and length.
func NewStreamDict(d Dict, streamOffset int64, streamLength *int64, streamLengthObjNr *int, filterPipeline []PDFFilter) StreamDict {
	return StreamDict{
		d,
		streamOffset,
		streamLength,
		streamLengthObjNr,
		filterPipeline,
		nil,
		nil,
		false,
	}
}

// HasSoleFilterNamed returns true if there is exactly one filter defined for a stream dict.
func (sd StreamDict) HasSoleFilterNamed(filterName string) bool {

	fpl := sd.FilterPipeline
	if fpl == nil {
		return false
	}

	if len(fpl) != 1 {
		return false
	}

	soleFilter := fpl[0]

	return soleFilter.Name == filterName
}

// ObjectStreamDict represents a object stream dictionary.
type ObjectStreamDict struct {
	StreamDict
	Prolog         []byte
	ObjCount       int
	FirstObjOffset int
	ObjArray       Array
}

// NewObjectStreamDict creates a new ObjectStreamDict object.
func NewObjectStreamDict() *ObjectStreamDict {

	sd := StreamDict{Dict: NewDict()}
	sd.Insert("Type", Name("ObjStm"))
	sd.Insert("Filter", Name(filter.Flate))
	sd.FilterPipeline = []PDFFilter{{Name: filter.Flate, DecodeParms: nil}}

	return &ObjectStreamDict{StreamDict: sd}
}

// IndexedObject returns the object at given index from a ObjectStreamDict.
func (osd *ObjectStreamDict) IndexedObject(index int) (Object, error) {
	if osd.ObjArray == nil {
		return nil, errors.Errorf("IndexedObject(%d): object not available", index)
	}
	return osd.ObjArray[index], nil
}

// AddObject adds another object to this object stream.
// Relies on decoded content!
func (osd *ObjectStreamDict) AddObject(objNumber int, entry *XRefTableEntry) error {

	offset := len(osd.Content)

	s := ""
	if osd.ObjCount > 0 {
		s = " "
	}
	s = s + fmt.Sprintf("%d %d", objNumber, offset)

	osd.Prolog = append(osd.Prolog, []byte(s)...)

	pdfString := entry.Object.PDFString()
	osd.Content = append(osd.Content, []byte(pdfString)...)

	osd.ObjCount++

	log.Trace.Printf("AddObject end : ObjCount:%d prolog = <%s> Content = <%s>\n", osd.ObjCount, osd.Prolog, osd.Content)

	return nil
}

// Finalize prepares the final content of the objectstream.
func (osd *ObjectStreamDict) Finalize() {
	osd.Content = append(osd.Prolog, osd.Content...)
	osd.FirstObjOffset = len(osd.Prolog)
	log.Trace.Printf("Finalize : firstObjOffset:%d Content = <%s>\n", osd.FirstObjOffset, osd.Content)
}

// XRefStreamDict represents a cross reference stream dictionary.
type XRefStreamDict struct {
	StreamDict
	Size           int
	Objects        []int
	W              [3]int
	PreviousOffset *int64
}

// NewXRefStreamDict creates a new PDFXRefStreamDict object.
func NewXRefStreamDict(ctx *Context) *XRefStreamDict {

	sd := StreamDict{Dict: NewDict()}

	sd.Insert("Type", Name("XRef"))
	sd.Insert("Filter", Name(filter.Flate))
	sd.FilterPipeline = []PDFFilter{{Name: filter.Flate, DecodeParms: nil}}

	sd.Insert("Root", *ctx.Root)

	if ctx.Info != nil {
		sd.Insert("Info", *ctx.Info)
	}

	if ctx.ID != nil {
		sd.Insert("ID", ctx.ID)
	}

	if ctx.Encrypt != nil && ctx.EncKey != nil {
		sd.Insert("Encrypt", *ctx.Encrypt)
	}

	return &XRefStreamDict{StreamDict: sd}
}
