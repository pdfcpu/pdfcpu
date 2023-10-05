/*
Copyright 2023 The pdfcpu Authors.

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

package validate

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

const userDateFormatNoTimeZone = "2006-01-02T15:04:05Z"
const userDateFormatNegTimeZone = "2006-01-02T15:04:05-07:00"
const userDateFormatPosTimeZone = "2006-01-02T15:04:05+07:00"

type userDate time.Time

func (ud *userDate) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	dateString := ""
	err := d.DecodeElement(&dateString, &start)
	if err != nil {
		return err
	}
	dat, err := time.Parse(userDateFormatNoTimeZone, dateString)
	if err == nil {
		*ud = userDate(dat)
		return nil
	}
	dat, err = time.Parse(userDateFormatPosTimeZone, dateString)
	if err == nil {
		*ud = userDate(dat)
		return nil
	}
	dat, err = time.Parse(userDateFormatNegTimeZone, dateString)
	if err == nil {
		*ud = userDate(dat)
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
	CreationDate userDate `xml:"http://ns.adobe.com/xap/1.0/ CreateDate"`
	ModDate      userDate `xml:"http://ns.adobe.com/xap/1.0/ ModifyDate"`
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

func validateMetadataStream(xRefTable *model.XRefTable, d types.Dict, required bool, sinceVersion model.Version) (*types.StreamDict, error) {
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V13
	}

	sd, err := validateStreamDictEntry(xRefTable, d, "dict", "Metadata", required, sinceVersion, nil)
	if err != nil || sd == nil {
		return nil, err
	}

	dictName := "metaDataDict"

	if _, err = validateNameEntry(xRefTable, sd.Dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Metadata" }); err != nil {
		return nil, err
	}

	if _, err = validateNameEntry(xRefTable, sd.Dict, dictName, "Subtype", OPTIONAL, sinceVersion, func(s string) bool { return s == "XML" }); err != nil {
		return nil, err
	}

	return sd, nil
}

func validateMetadata(xRefTable *model.XRefTable, d types.Dict, required bool, sinceVersion model.Version) error {
	// => 14.3 Metadata
	// In general, any PDF stream or dictionary may have metadata attached to it
	// as long as the stream or dictionary represents an actual information resource,
	// as opposed to serving as an implementation artifact.
	// Some PDF constructs are considered implementational, and hence may not have associated metadata.

	_, err := validateMetadataStream(xRefTable, d, required, sinceVersion)
	return err
}

func validateRootMetadata(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {
	sd, err := validateMetadataStream(xRefTable, rootDict, required, sinceVersion)
	if err != nil || sd == nil {
		return err
	}

	if xRefTable.Version() < model.V20 {
		return nil
	}

	// Decode streamDict for supported filters only.
	if err := sd.Decode(); err == filter.ErrUnsupportedFilter {
		return nil
	}
	if err != nil {
		return err
	}

	x := XMPMeta{}

	if err = xml.Unmarshal(sd.Content, &x); err != nil {
		fmt.Printf("error: %v", err)
		return err
	}

	// fmt.Printf("       Title: %v\n", x.RDF.Description.Title.Alt.Entries)
	// fmt.Printf("      Author: %v\n", x.RDF.Description.Author.Seq.Entries)
	// fmt.Printf("     Subject: %v\n", x.RDF.Description.Subject.Alt.Entries)
	// fmt.Printf("     Creator: %s\n", x.RDF.Description.Creator)
	// fmt.Printf("CreationDate: %v\n", time.Time(x.RDF.Description.CreationDate).Format(time.RFC3339Nano))
	// fmt.Printf("     ModDate: %v\n", time.Time(x.RDF.Description.ModDate).Format(time.RFC3339Nano))
	// fmt.Printf("    Producer: %s\n", x.RDF.Description.Producer)
	// fmt.Printf("     Trapped: %t\n", x.RDF.Description.Trapped)
	// fmt.Printf("    Keywords: %s\n", x.RDF.Description.Keywords)

	d := x.RDF.Description
	xRefTable.Title = strings.Join(d.Title.Alt.Entries, ", ")
	xRefTable.Author = strings.Join(d.Author.Seq.Entries, ", ")
	xRefTable.Subject = strings.Join(d.Subject.Alt.Entries, ", ")
	xRefTable.Creator = d.Creator
	xRefTable.CreationDate = time.Time(d.CreationDate).Format(time.RFC3339Nano)
	xRefTable.ModDate = time.Time(d.ModDate).Format(time.RFC3339Nano)
	xRefTable.Producer = d.Producer
	//xRefTable.Trapped = d.Trapped
	xRefTable.Keywords = d.Keywords

	return nil
}
