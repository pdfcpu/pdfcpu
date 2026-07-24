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

func validateMetadataStream(xRefTable *model.XRefTable, d types.Dict, required bool, sinceVersion model.Version) (*types.StreamDict, error) {
	if xRefTable.ValidationMode == model.ValidationRelaxed {
		sinceVersion = model.V10
	}

	rawEntry := d["Metadata"]
	sd, err := validateStreamDictEntry(xRefTable, d, "dict", "Metadata", required, sinceVersion, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", dictEntryContext("dict", "Metadata", rawEntry), err)
	}
	if sd == nil {
		delete(d, "Metadata")
		return nil, nil
	}

	dictName := "metaDataDict"

	if _, err = validateNameEntry(xRefTable, sd.Dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Metadata" }); err != nil {
		return nil, fmt.Errorf("%s: %s.Type: %w", objectContext(dictEntryContext("dict", "Metadata", rawEntry), rawEntry), dictName, err)
	}

	relaxed := xRefTable.ValidationMode == model.ValidationRelaxed
	subtype, err := validateNameEntry(xRefTable, sd.Dict, dictName, "Subtype", OPTIONAL, sinceVersion, func(s string) bool {
		return s == "XML" || relaxed && s == "XMP"
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %s.Subtype: %w", objectContext(dictEntryContext("dict", "Metadata", rawEntry), rawEntry), dictName, err)
	}
	if subtype != nil && relaxed && subtype.Value() == "XMP" {
		model.ShowDigestedSpecViolation("dict=" + dictName + " entry=Subtype invalid dict entry: XMP")
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
		if err != nil {
			return nil, fmt.Errorf("catalog metadata stream: %w", err)
		}
		return nil, nil
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
		return nil, fmt.Errorf("decode catalog metadata: %w", err)
	}

	x := model.XMPMeta{}

	if err = xml.Unmarshal(sd.Content, &x); err != nil {
		if xRefTable.ValidationMode == model.ValidationStrict {
			return nil, fmt.Errorf("parse catalog metadata: %w", err)
		}
		model.ShowSkipped("metadata parse error")
		return nil, nil
	}

	return &x, nil
}

func populateKeywordList(xRefTable *model.XRefTable, d model.Description) {
	ss := strings.FieldsFunc(d.Keywords, func(c rune) bool { return c == ',' || c == ';' || c == '\r' })
	for _, s := range ss {
		keyword := strings.TrimSpace(s)
		xRefTable.KeywordList[keyword] = true
	}
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

	s := strings.Join(d.Title.Alt.Entries, ", ")
	if len(s) > 0 || len(xRefTable.Title) == 0 {
		xRefTable.Title = s
	}

	s = strings.Join(d.Author.Seq.Entries, ", ")
	if len(s) > 0 || len(xRefTable.Author) == 0 {
		xRefTable.Author = s
	}

	s = strings.Join(d.Subject.Alt.Entries, ", ")
	if len(s) > 0 || len(xRefTable.Subject) == 0 {
		xRefTable.Subject = s
	}

	s = d.Creator
	if len(s) > 0 || len(xRefTable.Creator) == 0 {
		xRefTable.Creator = s
	}

	t := time.Time(d.CreationDate)
	if !t.IsZero() {
		xRefTable.CreationDate = types.DateString(t)
	}

	t = time.Time(d.ModDate)
	if !t.IsZero() {
		xRefTable.ModDate = types.DateString(t)
	}

	s = d.Producer
	if len(s) > 0 || len(xRefTable.Producer) == 0 {
		xRefTable.Producer = s
	}

	// TODO xRefTable.Trapped = d.Trapped

	populateKeywordList(xRefTable, d)

	return nil
}
