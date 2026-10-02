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
	"fmt"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type linkTargetKind uint8

const (
	linkTargetURI linkTargetKind = iota + 1
	linkTargetURIReference
	linkTargetFile
	linkTargetExecutable
)

type linkSource uint8

const (
	linkSourceURIAction linkSource = iota + 1
	linkSourceRemoteGoTo
	linkSourceEmbeddedGoTo
	linkSourceLaunch
	linkSourceSubmitForm
	linkSourceImportData
	linkSourceThread
	linkSourceFileAttachment
	linkSourceAlternatePresentation
	linkSourceReferenceXObject
	linkSourceOPI
	linkSourceURLFileSpecification
	linkSourceMediaClip
	linkSourceMediaClipBaseURL
	linkSourceMovie
	linkSourceSound
	linkSourceWebCaptureURL
	linkSourceWebCaptureSource
	linkSourceWebCaptureCommand
)

var linkSourceNames = [...]string{
	"",
	"URI action",
	"GoToR action",
	"GoToE action",
	"Launch action",
	"SubmitForm action",
	"ImportData action",
	"Thread action",
	"FileAttachment annotation",
	"AlternatePresentations resource",
	"Reference XObject",
	"OPI dictionary",
	"URL file specification",
	"MediaClip data",
	"MediaClip base URL",
	"Movie annotation",
	"Sound",
	"Web Capture URL",
	"Web Capture source",
	"Web Capture command",
}

type linkTarget struct {
	kind   linkTargetKind
	source linkSource
}

const linkTargetMetadataPrefix = "pdfcpu:link:"

func validLinkTargetKind(kind linkTargetKind) bool {
	return kind >= linkTargetURI && kind <= linkTargetExecutable
}

func validLinkSource(source linkSource) bool {
	return source > 0 && int(source) < len(linkSourceNames)
}

func (source linkSource) String() string {
	if !validLinkSource(source) {
		return "unknown external reference"
	}
	return linkSourceNames[source]
}

func encodeLinkTarget(target linkTarget) string {
	return fmt.Sprintf("%s%d:%d", linkTargetMetadataPrefix, target.kind, target.source)
}

func decodeLinkTarget(s string) (linkTarget, bool) {
	if !strings.HasPrefix(s, linkTargetMetadataPrefix) {
		return linkTarget{}, false
	}
	kindText, sourceText, found := strings.Cut(strings.TrimPrefix(s, linkTargetMetadataPrefix), ":")
	if !found || strings.Contains(sourceText, ":") {
		return linkTarget{}, false
	}
	kindValue, err := strconv.ParseUint(kindText, 10, 8)
	if err != nil {
		return linkTarget{}, false
	}
	sourceValue, err := strconv.ParseUint(sourceText, 10, 8)
	if err != nil {
		return linkTarget{}, false
	}
	target := linkTarget{kind: linkTargetKind(kindValue), source: linkSource(sourceValue)}
	return target, validLinkTargetKind(target.kind) && validLinkSource(target.source) && encodeLinkTarget(target) == s
}

func linkTargetPriority(kind linkTargetKind) int {
	switch kind {
	case linkTargetURI:
		return 3
	case linkTargetURIReference:
		return 2
	}
	return 1
}

func collectLinkTargetForPage(xRefTable *model.XRefTable, page int, target string, kind linkTargetKind, source linkSource) {
	if !xRefTable.ValidateLinks || target == "" {
		return
	}
	if xRefTable.URIs == nil {
		xRefTable.URIs = map[int]map[string]string{}
	}
	if xRefTable.URIs[page] == nil {
		xRefTable.URIs[page] = map[string]string{}
	}
	if encoded, found := xRefTable.URIs[page][target]; found {
		if existing, valid := decodeLinkTarget(encoded); valid && linkTargetPriority(existing.kind) > linkTargetPriority(kind) {
			return
		}
	}
	xRefTable.URIs[page][target] = encodeLinkTarget(linkTarget{kind: kind, source: source})
}

func collectLinkTarget(xRefTable *model.XRefTable, target string, kind linkTargetKind, source linkSource) {
	collectLinkTargetForPage(xRefTable, xRefTable.CurPage, target, kind, source)
}

func collectDocumentLinkTarget(xRefTable *model.XRefTable, target string, source linkSource) {
	collectLinkTargetForPage(xRefTable, 0, target, linkTargetURI, source)
}

func fileSpecificationTarget(xRefTable *model.XRefTable, o types.Object) (string, bool) {
	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return "", false
	}
	switch o := o.(type) {
	case types.StringLiteral, types.HexLiteral:
		target, err := model.Text(o)
		return target, err == nil
	case types.Dict:
		keys := []string{"UF", "F", "DOS", "Mac", "Unix"}
		if isURLFileSpecification(xRefTable, o) {
			keys = []string{"F", "UF", "DOS", "Mac", "Unix"}
		}
		for _, key := range keys {
			target, found, err := xRefTable.DereferenceStringEntry(o, key)
			if err == nil && found && target != nil {
				return *target, true
			}
		}
	}
	return "", false
}

func isURLFileSpecification(xRefTable *model.XRefTable, o types.Object) bool {
	o, err := xRefTable.Dereference(o)
	if err != nil {
		return false
	}
	d, ok := o.(types.Dict)
	if !ok {
		return false
	}
	fs, found, err := xRefTable.DereferenceNameEntry(d, "FS")
	return err == nil && found && fs != nil && fs.Value() == "URL"
}

func isEmbeddedFileSpecification(xRefTable *model.XRefTable, o types.Object) bool {
	if isURLFileSpecification(xRefTable, o) {
		return false
	}
	o, err := xRefTable.Dereference(o)
	if err != nil {
		return false
	}
	d, ok := o.(types.Dict)
	if !ok {
		return false
	}
	rawEF, found := d.Find("EF")
	if !found {
		return false
	}
	ef, err := xRefTable.DereferenceDict(rawEF)
	if err != nil || ef == nil {
		return false
	}
	for _, key := range []string{"UF", "F", "DOS", "Mac", "Unix"} {
		target, found, err := xRefTable.DereferenceStringEntry(d, key)
		if err != nil || !found || target == nil {
			continue
		}
		rawEmbeddedFile, found := ef.Find(key)
		if !found {
			continue
		}
		embeddedFile, err := xRefTable.Dereference(rawEmbeddedFile)
		if err != nil {
			return false
		}
		_, ok := embeddedFile.(types.StreamDict)
		return ok
	}
	return false
}

func collectFileSpecificationTargetForPage(xRefTable *model.XRefTable, page int, o types.Object, kind linkTargetKind, source linkSource) {
	target, found := fileSpecificationTarget(xRefTable, o)
	if !found {
		return
	}
	if isURLFileSpecification(xRefTable, o) {
		kind = linkTargetURI
	}
	collectLinkTargetForPage(xRefTable, page, target, kind, source)
}

func collectFileSpecificationTarget(xRefTable *model.XRefTable, o types.Object, kind linkTargetKind, source linkSource) {
	collectFileSpecificationTargetForPage(xRefTable, xRefTable.CurPage, o, kind, source)
}

func collectDocumentFileSpecificationTarget(xRefTable *model.XRefTable, o types.Object, kind linkTargetKind, source linkSource) {
	collectFileSpecificationTargetForPage(xRefTable, 0, o, kind, source)
}
