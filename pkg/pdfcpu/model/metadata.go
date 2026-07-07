/*
Copyright 2024 The pdfcpu Authors.

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

package model

import (
	"encoding/xml"
	"errors"
	"strings"
	"time"
)

// ErrMissingMetadata signals a missing required metadata buffer.
var ErrMissingMetadata = errors.New("missing metadata")

type UserDate time.Time

const userDateFormatNoTimeZone = "2006-01-02T15:04:05Z"
const userDateFormatNegTimeZone = "2006-01-02T15:04:05-07:00"
const userDateFormatPosTimeZone = "2006-01-02T15:04:05+07:00"

// UnmarshalXML unmarshals XML data.
func (ud *UserDate) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	dateString := ""
	err := d.DecodeElement(&dateString, &start)
	if err != nil {
		return err
	}
	dat, err := time.Parse(userDateFormatNoTimeZone, dateString)
	if err == nil {
		*ud = UserDate(dat)
		return nil
	}
	dat, err = time.Parse(userDateFormatPosTimeZone, dateString)
	if err == nil {
		*ud = UserDate(dat)
		return nil
	}
	dat, err = time.Parse(userDateFormatNegTimeZone, dateString)
	if err == nil {
		*ud = UserDate(dat)
		return nil
	}
	return err
}

type Alt struct {
	//XMLName xml.Name `xml:"http://www.w3.org/1999/02/22-rdf-syntax-ns# Alt"`
	Entries []string `xml:"http://www.w3.org/1999/02/22-rdf-syntax-ns# li"`
}

type Seq struct {
	//XMLName xml.Name `xml:"http://www.w3.org/1999/02/22-rdf-syntax-ns# Seq"`
	Entries []string `xml:"http://www.w3.org/1999/02/22-rdf-syntax-ns# li"`
}

type Title struct {
	//XMLName xml.Name `xml:"http://purl.org/dc/elements/1.1/ title"`
	Alt Alt `xml:"http://www.w3.org/1999/02/22-rdf-syntax-ns# Alt"`
}

type Desc struct {
	//XMLName xml.Name `xml:"http://purl.org/dc/elements/1.1/ description"`
	Alt Alt `xml:"http://www.w3.org/1999/02/22-rdf-syntax-ns# Alt"`
}

type Creator struct {
	//XMLName xml.Name `xml:"http://purl.org/dc/elements/1.1/ creator"`
	Seq Seq `xml:"http://www.w3.org/1999/02/22-rdf-syntax-ns# Seq"`
}

type Description struct {
	//XMLName      xml.Name `xml:"http://www.w3.org/1999/02/22-rdf-syntax-ns# Description"`
	Title        Title    `xml:"http://purl.org/dc/elements/1.1/ title"`
	Author       Creator  `xml:"http://purl.org/dc/elements/1.1/ creator"`
	Subject      Desc     `xml:"http://purl.org/dc/elements/1.1/ description"`
	Creator      string   `xml:"http://ns.adobe.com/xap/1.0/ CreatorTool"`
	CreationDate UserDate `xml:"http://ns.adobe.com/xap/1.0/ CreateDate"`
	ModDate      UserDate `xml:"http://ns.adobe.com/xap/1.0/ ModifyDate"`
	Producer     string   `xml:"http://ns.adobe.com/pdf/1.3/ Producer"`
	Trapped      bool     `xml:"http://ns.adobe.com/pdf/1.3/ Trapped"`
	Keywords     string   `xml:"http://ns.adobe.com/pdf/1.3/ Keywords"`
}

type RDF struct {
	XMLName     xml.Name `xml:"http://www.w3.org/1999/02/22-rdf-syntax-ns# RDF"`
	Description Description
}

type XMPMeta struct {
	XMLName xml.Name `xml:"adobe:ns:meta/ xmpmeta"`
	RDF     RDF
}

func removeTag(s, kw string) (string, bool) {
	original := s
	kwLen := len(kw)
	i := strings.Index(s, kw)
	if i < 0 {
		return s, false
	}

	j := i + kwLen

	i = strings.LastIndex(s[:i], "<")
	if i < 0 {
		return s, false
	}

	block1 := s[:i]

	s = s[j:]
	i = strings.Index(s, kw)
	if i < 0 {
		return original, false
	}

	j = i + kwLen

	block2 := s[j:]

	return block1 + block2, true
}

func removeKeywords(metadata *[]byte) (bool, error) {
	if metadata == nil {
		return false, ErrMissingMetadata
	}
	// Callers supply catalog XMP that has already passed validation. Do not reparse it
	// here; remove known tag fragments directly, treating missing or malformed fragments
	// as no-ops.
	s := string(*metadata)
	if len(s) == 0 {
		return false, nil
	}

	var removed bool
	if s, removed = removeTag(s, "Keywords>"); !removed {
		return false, nil
	}

	// Possible Acrobat bug.
	// Acrobat seems to use dc:subject for keywords but ***does not*** show the content in Subject.
	s, _ = removeTag(s, "subject>")
	*metadata = []byte(s)
	return true, nil
}

// RemoveKeywords removes keywords.
func RemoveKeywords(metadata *[]byte) error {
	_, err := removeKeywords(metadata)
	return err
}
