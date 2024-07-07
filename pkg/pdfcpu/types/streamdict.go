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

package types

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/log"

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
	//DCTImage          image.Image
	IsPageContent bool
	CSComponents  int
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
		//nil,
		false,
		0,
	}
}

// Clone returns a clone of sd.
func (sd StreamDict) Clone() Object {
	sd1 := sd
	sd1.Dict = sd.Dict.Clone().(Dict)
	pl := make([]PDFFilter, len(sd.FilterPipeline))
	for k, v := range sd.FilterPipeline {
		f := PDFFilter{}
		f.Name = v.Name
		if v.DecodeParms != nil {
			f.DecodeParms = v.DecodeParms.Clone().(Dict)
		}
		pl[k] = f
	}
	sd1.FilterPipeline = pl
	return sd1
}

// HasSoleFilterNamed returns true if sd has a
// filterPipeline with 1 filter named filterName.
func (sd StreamDict) HasSoleFilterNamed(filterName string) bool {
	fpl := sd.FilterPipeline
	if fpl == nil || len(fpl) != 1 {
		return false
	}
	return fpl[0].Name == filterName
}

func (sd StreamDict) Image() bool {
	s := sd.Type()
	if s == nil || *s != "XObject" {
		return false
	}
	s = sd.Subtype()
	if s == nil || *s != "Image" {
		return false
	}
	return true
}

type DecodeLazyObjectStreamObjectFunc func(c context.Context, s string) (Object, error)

type LazyObjectStreamObject struct {
	osd         *ObjectStreamDict
	startOffset int
	endOffset   int

	decodeFunc    DecodeLazyObjectStreamObjectFunc
	decodedObject Object
	decodedError  error
}

func NewLazyObjectStreamObject(osd *ObjectStreamDict, startOffset, endOffset int, decodeFunc DecodeLazyObjectStreamObjectFunc) Object {
	return LazyObjectStreamObject{
		osd:         osd,
		startOffset: startOffset,
		endOffset:   endOffset,

		decodeFunc: decodeFunc,
	}
}

func (l LazyObjectStreamObject) Clone() Object {
	return LazyObjectStreamObject{
		osd:         l.osd,
		startOffset: l.startOffset,
		endOffset:   l.endOffset,

		decodeFunc:    l.decodeFunc,
		decodedObject: l.decodedObject,
		decodedError:  l.decodedError,
	}
}

func (l LazyObjectStreamObject) PDFString() string {
	data, err := l.GetData()
	if err != nil {
		panic(err)
	}

	return string(data)
}

func (l LazyObjectStreamObject) String() string {
	return l.PDFString()
}

func (l *LazyObjectStreamObject) GetData() ([]byte, error) {
	if err := l.osd.Decode(); err != nil {
		return nil, err
	}

	var data []byte
	if l.endOffset == -1 {
		data = l.osd.Content[l.startOffset:]
	} else {
		data = l.osd.Content[l.startOffset:l.endOffset]
	}
	return data, nil
}

func (l *LazyObjectStreamObject) DecodedObject(c context.Context) (Object, error) {
	if l.decodedObject == nil && l.decodedError == nil {
		data, err := l.GetData()
		if err != nil {
			return nil, err
		}

		if log.ReadEnabled() {
			log.Read.Printf("parseObjectStream: objString = %s\n", string(data))
		}

		l.decodedObject, l.decodedError = l.decodeFunc(c, string(data))
		if l.decodedError != nil {
			return nil, l.decodedError
		}

		if log.ReadEnabled() {
			//log.Read.Printf("parseObjectStream: [%d] = obj %s:\n%s\n", i/2-1, objs[i-2], o)
		}
	}
	return l.decodedObject, l.decodedError
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

func parmsForFilter(d Dict) map[string]int {
	m := map[string]int{}

	if d == nil {
		return m
	}

	for k, v := range d {

		i, ok := v.(Integer)
		if ok {
			m[k] = i.Value()
			continue
		}

		// Encode boolean values: false -> 0, true -> 1
		b, ok := v.(Boolean)
		if ok {
			m[k] = 0
			if b.Value() {
				m[k] = 1
			}
			continue
		}

	}

	return m
}

// Encode applies sd's filter pipeline to sd.Content in order to produce sd.Raw.
func (sd *StreamDict) Encode() error {
	if sd.Content == nil && sd.Raw != nil {
		// Not decoded yet, no need to encode.
		return nil
	}

	// No filter specified, nothing to encode.
	if sd.FilterPipeline == nil {
		if log.TraceEnabled() {
			log.Trace.Println("encodeStream: returning uncompressed stream.")
		}
		sd.Raw = sd.Content
		streamLength := int64(len(sd.Raw))
		sd.StreamLength = &streamLength
		sd.Update("Length", Integer(streamLength))
		return nil
	}

	var b, c io.Reader
	b = bytes.NewReader(sd.Content)

	// Apply each filter in the pipeline to result of preceding filter.

	for i := len(sd.FilterPipeline) - 1; i >= 0; i-- {
		f := sd.FilterPipeline[i]
		if log.TraceEnabled() {
			if f.DecodeParms != nil {
				log.Trace.Printf("encodeStream: encoding filter:%s\ndecodeParms:%s\n", f.Name, f.DecodeParms)
			} else {
				log.Trace.Printf("encodeStream: encoding filter:%s\n", f.Name)
			}
		}

		// Make parms map[string]int
		parms := parmsForFilter(f.DecodeParms)

		fi, err := filter.NewFilter(f.Name, parms)
		if err != nil {
			return err
		}

		c, err = fi.Encode(b)
		if err != nil {
			return err
		}

		b = c
	}

	if bb, ok := c.(*bytes.Buffer); ok {
		sd.Raw = bb.Bytes()
	} else {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, c); err != nil {
			return err
		}

		sd.Raw = buf.Bytes()
	}

	streamLength := int64(len(sd.Raw))
	sd.StreamLength = &streamLength
	sd.Update("Length", Integer(streamLength))

	return nil
}

func fixParms(f PDFFilter, parms map[string]int, sd *StreamDict) error {
	if f.Name == filter.CCITTFax {
		// x/image/ccitt needs the optional decode parameter "Rows"
		// if not available we supply image "Height".
		_, ok := parms["Rows"]
		if !ok {
			ip := sd.IntEntry("Height")
			if ip == nil {
				return errors.New("pdfcpu: ccitt: \"Height\" required")
			}
			parms["Rows"] = *ip
		}
	}
	return nil
}

// Decode applies sd's filter pipeline to sd.Raw in order to produce sd.Content.
func (sd *StreamDict) Decode() error {
	_, err := sd.DecodeLength(-1)
	return err
}

func (sd *StreamDict) decodeLength(maxLen int64) ([]byte, error) {
	var b, c io.Reader
	b = bytes.NewReader(sd.Raw)

	// Apply each filter in the pipeline to result of preceding filter.
	for idx, f := range sd.FilterPipeline {

		if f.Name == filter.JPX {
			break
		}

		if f.Name == filter.DCT {
			if sd.CSComponents != 4 {
				break
			}
			// if sd.CSComponents == 4 {
			// 	// Special case where we have to do real JPG decoding.
			// 	// Another option is using a dctDecode filter using gob - performance hit?

			// 	im, err := jpeg.Decode(b)
			//  if err != nil {
			// 	 	return err
			// 	}
			// 	sd.DCTImage = im // hacky
			// 	return nil
			// }
		}

		parms := parmsForFilter(f.DecodeParms)
		if err := fixParms(f, parms, sd); err != nil {
			return nil, err
		}

		fi, err := filter.NewFilter(f.Name, parms)
		if err != nil {
			return nil, err
		}

		if maxLen >= 0 && idx == len(sd.FilterPipeline)-1 {
			c, err = fi.DecodeLength(b, maxLen)
		} else {
			c, err = fi.Decode(b)
		}
		if err != nil {
			return nil, err
		}

		//fmt.Printf("decodedStream after:%s\n%s\n", f.Name, hex.Dump(c.Bytes()))
		b = c
	}

	var data []byte
	if bb, ok := c.(*bytes.Buffer); ok {
		data = bb.Bytes()
	} else {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, c); err != nil {
			return nil, err
		}

		data = buf.Bytes()
	}

	if maxLen < 0 {
		sd.Content = data
		return data, nil
	}

	return data[:maxLen], nil
}

func (sd *StreamDict) DecodeLength(maxLen int64) ([]byte, error) {
	if sd.Content != nil {
		// This stream has already been decoded.
		if maxLen < 0 {
			return sd.Content, nil
		}

		return sd.Content[:maxLen], nil
	}

	fpl := sd.FilterPipeline

	// No filter or sole filter DTC && !CMYK or JPX - nothing to decode.
	if fpl == nil || len(fpl) == 1 && ((fpl[0].Name == filter.DCT && sd.CSComponents != 4) || fpl[0].Name == filter.JPX) {
		sd.Content = sd.Raw
		//fmt.Printf("decodedStream returning %d(#%02x)bytes: \n%s\n", len(sd.Content), len(sd.Content), hex.Dump(sd.Content))
		if maxLen < 0 {
			return sd.Content, nil
		}

		return sd.Content[:maxLen], nil
	}

	//fmt.Printf("decodedStream before:\n%s\n", hex.Dump(sd.Raw))

	return sd.decodeLength(maxLen)
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
func (osd *ObjectStreamDict) AddObject(objNumber int, pdfString string) error {
	offset := len(osd.Content)
	s := ""
	if osd.ObjCount > 0 {
		s = " "
	}
	s = s + fmt.Sprintf("%d %d", objNumber, offset)
	osd.Prolog = append(osd.Prolog, []byte(s)...)
	//pdfString := entry.Object.PDFString()
	osd.Content = append(osd.Content, []byte(pdfString)...)
	osd.ObjCount++
	if log.TraceEnabled() {
		log.Trace.Printf("AddObject end : ObjCount:%d prolog = <%s> Content = <%s>\n", osd.ObjCount, osd.Prolog, osd.Content)
	}
	return nil
}

// Finalize prepares the final content of the objectstream.
func (osd *ObjectStreamDict) Finalize() {
	osd.Content = append(osd.Prolog, osd.Content...)
	osd.FirstObjOffset = len(osd.Prolog)
	if log.TraceEnabled() {
		log.Trace.Printf("Finalize : firstObjOffset:%d Content = <%s>\n", osd.FirstObjOffset, osd.Content)
	}
}

// XRefStreamDict represents a cross reference stream dictionary.
type XRefStreamDict struct {
	StreamDict
	Size           int
	Objects        []int
	W              [3]int
	PreviousOffset *int64
}
