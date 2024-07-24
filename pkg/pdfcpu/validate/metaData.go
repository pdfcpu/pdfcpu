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
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

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

func catalogMetaData(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) (*model.XMPMeta, error) {
	sd, err := validateMetadataStream(xRefTable, rootDict, required, sinceVersion)
	if err != nil || sd == nil {
		return nil, err
	}

	// if xRefTable.Version() < model.V20 {
	// 	return nil
	// }

	// Decode streamDict for supported filters only.
	err = sd.Decode()
	if err == filter.ErrUnsupportedFilter {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	x := model.XMPMeta{}

	if err = xml.Unmarshal(sd.Content, &x); err != nil {
		if xRefTable.ValidationMode == model.ValidationStrict {
			return nil, err
		}
		log.CLI.Println("ignoring metadata parse error")
		return nil, nil
	}

	return &x, nil
}

func validateRootMetadata(xRefTable *model.XRefTable, rootDict types.Dict, required bool, sinceVersion model.Version) error {

	if xRefTable.CatalogXMPMeta == nil {
		return nil
	}

	x := xRefTable.CatalogXMPMeta

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

	ss := strings.FieldsFunc(d.Keywords, func(c rune) bool { return c == ',' || c == ';' || c == '\r' })
	for _, s := range ss {
		keyword := strings.TrimSpace(s)
		xRefTable.KeywordList[keyword] = true
	}

	return nil
}
