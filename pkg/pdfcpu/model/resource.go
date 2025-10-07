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

package model

import (
	"fmt"
	"sort"
	"strings"

	"github.com/angel-one/pdfcpu/pkg/pdfcpu/types"
)

// FontObject represents a font used in a PDF file.
type FontObject struct {
	ResourceNames []string
	Prefix        string
	FontName      string
	FontDict      types.Dict
	Data          []byte
	Extension     string
	Embedded      bool
}

// AddResourceName adds a resourceName referring to this font.
func (fo *FontObject) AddResourceName(resourceName string) {
	for _, resName := range fo.ResourceNames {
		if resName == resourceName {
			return
		}
	}
	fo.ResourceNames = append(fo.ResourceNames, resourceName)
}

// ResourceNamesString returns a string representation of all the resource names of this font.
func (fo FontObject) ResourceNamesString() string {
	var resNames []string
	resNames = append(resNames, fo.ResourceNames...)
	return strings.Join(resNames, ",")
}

// Data returns the raw data belonging to this image object.
// func (fo FontObject) Data() []byte {
// 	return nil
// }

// SubType returns the SubType of this font.
func (fo FontObject) SubType() string {
	var subType string
	if fo.FontDict.Subtype() != nil {
		subType = *fo.FontDict.Subtype()
	}
	return subType
}

// Encoding returns the Encoding of this font.
func (fo FontObject) Encoding() string {
	encoding := "Built-in"
	pdfObject, found := fo.FontDict.Find("Encoding")
	if found {
		switch enc := pdfObject.(type) {
		case types.Name:
			encoding = enc.Value()
		default:
			encoding = "Custom"
		}
	}
	return encoding
}

func (fo FontObject) String() string {
	return fmt.Sprintf("%-10s %-30s %-10s %-20s %-8v %s\n",
		fo.Prefix, fo.FontName,
		fo.SubType(), fo.Encoding(),
		fo.Embedded, fo.ResourceNamesString())
}

// ImageObject represents an image used in a PDF file.
type ImageObject struct {
	ResourceNames map[int]string
	ImageDict     *types.StreamDict
}

// DuplicateImageObject represents a redundant image.
type DuplicateImageObject struct {
	ImageDict *types.StreamDict
	NewObjNr  int
}

// AddResourceName adds a resourceName to this imageObject's ResourceNames map.
func (io *ImageObject) AddResourceName(pageNr int, resourceName string) {
	io.ResourceNames[pageNr] = resourceName
}

// ResourceNamesString returns a string representation of the ResourceNames for this image.
func (io ImageObject) ResourceNamesString() string {
	pageNrs := make([]int, 0, len(io.ResourceNames))
	for k := range io.ResourceNames {
		pageNrs = append(pageNrs, k)
	}
	sort.Ints(pageNrs)
	var sb strings.Builder
	for i, pageNr := range pageNrs {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%d:%s", pageNr, io.ResourceNames[pageNr]))
	}
	var resNames []string
	resNames = append(resNames, sb.String())
	return strings.Join(resNames, ",")
}

var resourceTypes = types.NewStringSet([]string{"ColorSpace", "ExtGState", "Font", "Pattern", "Properties", "Shading", "XObject"})

// PageResourceNames represents the required resource names for a specific page as extracted from its content streams.
type PageResourceNames map[string]types.StringSet

// NewPageResourceNames returns initialized pageResourceNames.
func NewPageResourceNames() PageResourceNames {
	m := make(map[string]types.StringSet, len(resourceTypes))
	for k := range resourceTypes {
		m[k] = types.StringSet{}
	}
	return m
}

// Resources returns a set of all required resource names for subdict s.
func (prn PageResourceNames) Resources(s string) types.StringSet {
	return prn[s]
}

// HasResources returns true for any resource names present in resource subDict s.
func (prn PageResourceNames) HasResources(s string) bool {
	return len(prn.Resources(s)) > 0
}

// HasContent returns true in any resource names present.
func (prn PageResourceNames) HasContent() bool {
	for k := range resourceTypes {
		if prn.HasResources(k) {
			return true
		}
	}
	return false
}

func (prn PageResourceNames) String() string {
	sep := ", "
	var ss []string
	s := []string{"PageResourceNames:\n"}
	for k := range resourceTypes {
		ss = nil
		for k := range prn.Resources(k) {
			ss = append(ss, k)
		}
		s = append(s, k+": "+strings.Join(ss, sep)+"\n")
	}
	return strings.Join(s, "")
}
