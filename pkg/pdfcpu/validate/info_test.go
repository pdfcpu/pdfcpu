/*
Copyright 2026 The pdfcpu Authors.

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
	"reflect"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type infoValidationCase struct {
	name             string
	mode             int
	object           types.Object
	wantErr          bool
	wantInfo         bool
	wantProducer     string
	wantCreationDate string
}

func infoXRefTable(mode int, object types.Object) *model.XRefTable {
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = mode
	v := model.V20

	return &model.XRefTable{
		Table: map[int]*model.XRefTableEntry{
			1: model.NewXRefTableEntryGen0(object),
		},
		Info:           types.NewIndirectRef(1, 0),
		RootDict:       types.Dict{},
		Conf:           conf,
		HeaderVersion:  &v,
		ValidationMode: mode,
		KeywordList:    types.StringSet{},
		Properties:     map[string]string{},
	}
}

func assertInfoObjectUnchanged(t *testing.T, xRefTable *model.XRefTable, before types.Object) {
	t.Helper()
	if got := xRefTable.Table[1].Object; !reflect.DeepEqual(got, before) {
		t.Fatalf("info object changed:\n got: %#v\nwant: %#v", got, before)
	}
}

func assertInfoReference(t *testing.T, xRefTable *model.XRefTable, wantPresent bool) {
	t.Helper()
	if got := xRefTable.Info != nil; got != wantPresent {
		t.Fatalf("Info present: %t, want: %t", got, wantPresent)
	}
	if wantPresent && *xRefTable.Info != *types.NewIndirectRef(1, 0) {
		t.Fatalf("Info = %v, want 1 0 R", *xRefTable.Info)
	}
}

func runInfoValidationCase(t *testing.T, tt infoValidationCase) {
	t.Helper()
	xRefTable := infoXRefTable(tt.mode, tt.object)
	before := tt.object.Clone()

	err := validateDocumentInfoObject(xRefTable)
	if got := err != nil; got != tt.wantErr {
		t.Fatalf("error = %v, want error: %t", err, tt.wantErr)
	}
	assertInfoReference(t, xRefTable, tt.wantInfo)
	if xRefTable.Producer != tt.wantProducer {
		t.Fatalf("Producer = %q, want %q", xRefTable.Producer, tt.wantProducer)
	}
	if xRefTable.CreationDate != tt.wantCreationDate {
		t.Fatalf("CreationDate = %q, want %q", xRefTable.CreationDate, tt.wantCreationDate)
	}
	assertInfoObjectUnchanged(t, xRefTable, before)
}

// TestValidateDocumentInfoEntriesCharacterizesStrictAndRelaxedMode records validation and cache behavior without
// allowing validation to rewrite source Info dictionary entries.
func TestValidateDocumentInfoEntriesCharacterizesStrictAndRelaxedMode(t *testing.T) {
	malformedUTF16BE := types.StringLiteral("\\376\\377\\330\\000\\000\\101")
	tests := []infoValidationCase{
		{
			name:         "valid strict",
			mode:         model.ValidationStrict,
			object:       types.Dict{"Producer": types.StringLiteral("producer")},
			wantInfo:     true,
			wantProducer: "producer",
		},
		{
			name:         "valid relaxed",
			mode:         model.ValidationRelaxed,
			object:       types.Dict{"Producer": types.StringLiteral("producer")},
			wantInfo:     true,
			wantProducer: "producer",
		},
		{
			name:     "noncanonical date strict",
			mode:     model.ValidationStrict,
			object:   types.Dict{"CreationDate": types.StringLiteral("20200102030405Z")},
			wantErr:  true,
			wantInfo: true,
		},
		{
			name:             "noncanonical date relaxed",
			mode:             model.ValidationRelaxed,
			object:           types.Dict{"CreationDate": types.StringLiteral("20200102030405Z")},
			wantInfo:         true,
			wantCreationDate: "D:20200102030405+00'00'",
		},
		{
			name:     "invalid date strict",
			mode:     model.ValidationStrict,
			object:   types.Dict{"CreationDate": types.StringLiteral("not-a-date")},
			wantErr:  true,
			wantInfo: true,
		},
		{
			name:     "invalid date relaxed",
			mode:     model.ValidationRelaxed,
			object:   types.Dict{"CreationDate": types.StringLiteral("not-a-date")},
			wantInfo: true,
		},
		{
			name:     "malformed UTF16BE strict",
			mode:     model.ValidationStrict,
			object:   types.Dict{"Producer": malformedUTF16BE},
			wantErr:  true,
			wantInfo: true,
		},
		{
			name:     "malformed UTF16BE relaxed",
			mode:     model.ValidationRelaxed,
			object:   types.Dict{"Producer": malformedUTF16BE},
			wantErr:  true,
			wantInfo: true,
		},
		{
			name:     "alternative Trapped strict",
			mode:     model.ValidationStrict,
			object:   types.Dict{"Trapped": types.Boolean(true)},
			wantErr:  true,
			wantInfo: true,
		},
		{
			name:     "alternative Trapped relaxed",
			mode:     model.ValidationRelaxed,
			object:   types.Dict{"Trapped": types.Boolean(true)},
			wantInfo: true,
		},
		{
			name:     "wrong custom property type strict",
			mode:     model.ValidationStrict,
			object:   types.Dict{"Custom": types.Integer(7)},
			wantErr:  true,
			wantInfo: true,
		},
		{
			name:     "wrong custom property type relaxed",
			mode:     model.ValidationRelaxed,
			object:   types.Dict{"Custom": types.Integer(7)},
			wantInfo: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runInfoValidationCase(t, tt)
		})
	}
}

// TestValidateDocumentInfoObjectReferenceRepairs records the cases where validation clears the Info reference.
func TestValidateDocumentInfoObjectReferenceRepairs(t *testing.T) {
	for _, tt := range []struct {
		name     string
		mode     int
		object   types.Object
		wantErr  bool
		wantInfo bool
	}{
		{name: "null strict", mode: model.ValidationStrict},
		{name: "null relaxed", mode: model.ValidationRelaxed},
		{
			name: "non-dictionary strict", mode: model.ValidationStrict,
			object: types.Integer(7), wantErr: true, wantInfo: true,
		},
		{name: "non-dictionary relaxed", mode: model.ValidationRelaxed, object: types.Integer(7)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			xRefTable := infoXRefTable(tt.mode, tt.object)
			err := validateDocumentInfoObject(xRefTable)
			if got := err != nil; got != tt.wantErr {
				t.Fatalf("error = %v, want error: %t", err, tt.wantErr)
			}
			assertInfoReference(t, xRefTable, tt.wantInfo)
			assertInfoObjectUnchanged(t, xRefTable, tt.object)
		})
	}
}

// TestValidateDocumentInfoPieceInfoRequirement characterizes the strict and relaxed handling of a missing ModDate.
func TestValidateDocumentInfoPieceInfoRequirement(t *testing.T) {
	for _, tt := range []struct {
		name    string
		mode    int
		wantErr bool
	}{
		{name: "strict", mode: model.ValidationStrict, wantErr: true},
		{name: "relaxed", mode: model.ValidationRelaxed},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d := types.Dict{"Producer": types.StringLiteral("producer")}
			xRefTable := infoXRefTable(tt.mode, d)
			xRefTable.RootDict["PieceInfo"] = types.Dict{}
			before := d.Clone()

			err := validateDocumentInfoObject(xRefTable)
			if got := err != nil; got != tt.wantErr {
				t.Fatalf("error = %v, want error: %t", err, tt.wantErr)
			}
			assertInfoReference(t, xRefTable, true)
			assertInfoObjectUnchanged(t, xRefTable, before)
		})
	}
}

// TestFixInfoDictClearsMetadataAlias records the repair for an Info reference pointing at catalog Metadata.
func TestFixInfoDictClearsMetadataAlias(t *testing.T) {
	for _, tt := range []struct {
		name string
		mode int
	}{
		{name: "strict", mode: model.ValidationStrict},
		{name: "relaxed", mode: model.ValidationRelaxed},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sd := types.StreamDict{Dict: types.Dict{"Type": types.Name("Metadata")}}
			xRefTable := infoXRefTable(tt.mode, sd)
			xRefTable.RootDict["Metadata"] = *xRefTable.Info
			before := sd
			before.Dict = sd.Dict.Clone().(types.Dict)

			if err := fixInfoDict(xRefTable, xRefTable.RootDict); err != nil {
				t.Fatal(err)
			}
			assertInfoReference(t, xRefTable, false)
			assertInfoObjectUnchanged(t, xRefTable, before)
		})
	}
}

func xmpMetadata(modDate, producer string) []byte {
	return []byte(`<x:xmpmeta xmlns:x="adobe:ns:meta/">` +
		`<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">` +
		`<rdf:Description xmlns:xmp="http://ns.adobe.com/xap/1.0/" xmlns:pdf="http://ns.adobe.com/pdf/1.3/">` +
		`<xmp:ModifyDate>` + modDate + `</xmp:ModifyDate>` +
		`<pdf:Producer>` + producer + `</pdf:Producer>` +
		`</rdf:Description></rdf:RDF></x:xmpmeta>`)
}

func metadataInfoXRefTable(mode int, infoModDate, xmpModDate string) (*model.XRefTable, types.Dict) {
	infoDict := types.Dict{
		"ModDate":  types.StringLiteral(infoModDate),
		"Producer": types.StringLiteral("info producer"),
	}
	xRefTable := infoXRefTable(mode, infoDict)
	metadataRef := types.NewIndirectRef(2, 0)
	xRefTable.RootDict["Metadata"] = *metadataRef
	xRefTable.Table[2] = model.NewXRefTableEntryGen0(types.StreamDict{
		Dict: types.Dict{
			"Type":    types.Name("Metadata"),
			"Subtype": types.Name("XML"),
		},
		Content: xmpMetadata(xmpModDate, "xmp producer"),
	})
	return xRefTable, infoDict
}

func validateInfoAndMetadataByAuthority(xRefTable *model.XRefTable, metadataAuthoritative bool) error {
	if metadataAuthoritative {
		if err := validateDocumentInfoObject(xRefTable); err != nil {
			return err
		}
		return validateRootMetadata(xRefTable, xRefTable.RootDict, OPTIONAL, model.V14)
	}
	if err := validateRootMetadata(xRefTable, xRefTable.RootDict, OPTIONAL, model.V14); err != nil {
		return err
	}
	return validateDocumentInfoObject(xRefTable)
}

// TestMetadataAuthorityControlsCachedInfoValues records how modification dates choose cached Info or XMP values.
func TestMetadataAuthorityControlsCachedInfoValues(t *testing.T) {
	for _, tt := range []struct {
		name                      string
		mode                      int
		infoModDate               string
		xmpModDate                string
		wantMetadataAuthoritative bool
		wantProducer              string
	}{
		{
			name: "newer XMP strict", mode: model.ValidationStrict,
			infoModDate: "D:20200101000000+00'00'", xmpModDate: "2021-01-01T00:00:00Z",
			wantMetadataAuthoritative: true, wantProducer: "xmp producer",
		},
		{
			name: "newer Info strict", mode: model.ValidationStrict,
			infoModDate: "D:20220101000000+00'00'", xmpModDate: "2021-01-01T00:00:00Z",
			wantProducer: "info producer",
		},
		{
			name: "newer XMP relaxed", mode: model.ValidationRelaxed,
			infoModDate: "D:20200101000000+00'00'", xmpModDate: "2021-01-01T00:00:00Z",
			wantMetadataAuthoritative: true, wantProducer: "xmp producer",
		},
		{
			name: "newer Info relaxed", mode: model.ValidationRelaxed,
			infoModDate: "D:20220101000000+00'00'", xmpModDate: "2021-01-01T00:00:00Z",
			wantProducer: "info producer",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			xRefTable, infoDict := metadataInfoXRefTable(tt.mode, tt.infoModDate, tt.xmpModDate)
			before := infoDict.Clone()

			metadataAuthoritative, err := metaDataModifiedAfterInfoDict(xRefTable)
			if err != nil {
				t.Fatal(err)
			}
			if metadataAuthoritative != tt.wantMetadataAuthoritative {
				t.Fatalf("metadata authoritative: %t, want: %t", metadataAuthoritative, tt.wantMetadataAuthoritative)
			}
			if err := validateInfoAndMetadataByAuthority(xRefTable, metadataAuthoritative); err != nil {
				t.Fatal(err)
			}
			if xRefTable.Producer != tt.wantProducer {
				t.Fatalf("Producer = %q, want %q", xRefTable.Producer, tt.wantProducer)
			}
			assertInfoReference(t, xRefTable, true)
			assertInfoObjectUnchanged(t, xRefTable, before)
		})
	}
}
