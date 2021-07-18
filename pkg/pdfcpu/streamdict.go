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
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"

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

// Clone returns a clone of sd.
func (sd StreamDict) Clone() Object {
	sd1 := sd
	sd1.Dict = sd.Dict.Clone().(Dict)
	pl := make([]PDFFilter, len(sd.FilterPipeline))
	for k, v := range sd.FilterPipeline {
		f := PDFFilter{}
		f.Name = v.Name
		if f.DecodeParms != nil {
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
	// No filter specified, nothing to encode.
	if sd.FilterPipeline == nil {
		log.Trace.Println("encodeStream: returning uncompressed stream.")
		sd.Raw = sd.Content
		streamLength := int64(len(sd.Raw))
		sd.StreamLength = &streamLength
		sd.Update("Length", Integer(streamLength))
		return nil
	}

	var b, c io.Reader
	b = bytes.NewReader(sd.Content)

	// Apply each filter in the pipeline to result of preceding filter.
	for _, f := range sd.FilterPipeline {
		if f.DecodeParms != nil {
			log.Trace.Printf("encodeStream: encoding filter:%s\ndecodeParms:%s\n", f.Name, f.DecodeParms)
		} else {
			log.Trace.Printf("encodeStream: encoding filter:%s\n", f.Name)
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

	var err error
	if sd.Raw, err = ioutil.ReadAll(c); err != nil {
		return err
	}
	streamLength := int64(len(sd.Raw))
	sd.StreamLength = &streamLength
	sd.Update("Length", Integer(streamLength))

	return nil
}

// Decode applies sd's filter pipeline to sd.Raw in order to produce sd.Content.
func (sd *StreamDict) Decode() error {
	if sd.Content != nil {
		// This stream has already been decoded.
		return nil
	}

	// No filter specified, nothing to decode.
	if sd.FilterPipeline == nil {
		sd.Content = sd.Raw
		log.Trace.Printf("decodedStream returning %d(#%02x)bytes: \n%s\n", len(sd.Content), len(sd.Content), hex.Dump(sd.Content))
		return nil
	}

	//fmt.Printf("decodedStream before:\n%s\n", hex.Dump(sd.Raw))

	var b, c io.Reader
	b = bytes.NewReader(sd.Raw)

	// Apply each filter in the pipeline to result of preceding filter.
	for _, f := range sd.FilterPipeline {

		if f.DecodeParms != nil {
			log.Trace.Printf("decodeStream: decoding filter:%s\ndecodeParms:%s\n", f.Name, f.DecodeParms)
		} else {
			log.Trace.Printf("decodeStream: decoding filter:%s\n", f.Name)
		}

		// make parms map[string]int
		parms := parmsForFilter(f.DecodeParms)

		if f.Name == filter.CCITTFax {
			// x/image/ccitt needs the optional decode parameter "Rows"
			// if not available we supply the image "Height".
			_, ok := parms["Rows"]
			if !ok {
				ip := sd.IntEntry("Height")
				if ip == nil {
					return errors.New("pdfcpu: ccitt: \"Height\" required")
				}
				parms["Rows"] = *ip
			}
		}

		fi, err := filter.NewFilter(f.Name, parms)
		if err != nil {
			return err
		}

		c, err = fi.Decode(b)
		if err != nil {
			return err
		}

		//fmt.Printf("decodedStream after:%s\n%s\n", f.Name, hex.Dump(c.Bytes()))
		b = c
	}

	var err error
	if sd.Content, err = ioutil.ReadAll(c); err != nil {
		return err
	}

	return nil
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
	if ctx.Write.Increment {
		sd.Insert("Prev", Integer(*ctx.Write.OffsetPrevXRef))
	}
	return &XRefStreamDict{StreamDict: sd}
}
