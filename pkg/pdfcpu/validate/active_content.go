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
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type activeContentKind string

const (
	activeContentJavaScript activeContentKind = "JavaScript"
	activeContentSound      activeContentKind = "Sound"
	activeContentMovie      activeContentKind = "Movie"
	activeContentRendition  activeContentKind = "Rendition"
)

type activeContentOwner string

const (
	activeContentOwnerAction     activeContentOwner = "action"
	activeContentOwnerDocument   activeContentOwner = "document"
	activeContentOwnerPage       activeContentOwner = "page"
	activeContentOwnerAnnotation activeContentOwner = "annotation"
	activeContentOwnerFormField  activeContentOwner = "form field"
	activeContentOwnerNameTree   activeContentOwner = "name tree"
	activeContentOwnerOutline    activeContentOwner = "outline"
)

type activeContentSource string

const (
	activeContentSourceAction             activeContentSource = "action"
	activeContentSourceOpenAction         activeContentSource = "OpenAction"
	activeContentSourceAdditionalAction   activeContentSource = "additional action"
	activeContentSourceAnnotationAction   activeContentSource = "annotation action"
	activeContentSourceOutlineAction      activeContentSource = "outline action"
	activeContentSourceJavaScriptNameTree activeContentSource = "JavaScript name tree"
	activeContentSourceRenditionAction    activeContentSource = "Rendition action"
	activeContentSourceSoundAnnotation    activeContentSource = "Sound annotation"
	activeContentSourceMovieAnnotation    activeContentSource = "Movie annotation"
)

type activeContentFinding struct {
	kind       activeContentKind
	owner      activeContentOwner
	source     activeContentSource
	trigger    string
	pageNr     int
	objNr      int
	ownerObjNr int
	depth      int
}

func activeContentSourceDescription(f activeContentFinding) string {
	switch f.source {
	case activeContentSourceOpenAction:
		return "document OpenAction"
	case activeContentSourceAdditionalAction:
		s := fmt.Sprintf("%s additional action", f.owner)
		if f.trigger != "" {
			s += " /" + f.trigger
		}
		return s
	case activeContentSourceAnnotationAction:
		return "annotation action"
	case activeContentSourceOutlineAction:
		return "outline action"
	case activeContentSourceJavaScriptNameTree:
		return "JavaScript name tree"
	case activeContentSourceRenditionAction:
		s := fmt.Sprintf("%s Rendition action", f.owner)
		if f.trigger != "" {
			s += " /" + f.trigger
		}
		return s
	case activeContentSourceSoundAnnotation:
		return "sound annotation"
	case activeContentSourceMovieAnnotation:
		return "movie annotation"
	default:
		return string(f.owner)
	}
}

func activeContentMessage(f activeContentFinding) string {
	location := "document"
	if f.pageNr > 0 {
		location = fmt.Sprintf("page %d", f.pageNr)
	}
	details := []string{}
	if f.objNr > 0 {
		details = append(details, fmt.Sprintf("obj#:%d", f.objNr))
	}
	if f.ownerObjNr > 0 && f.ownerObjNr != f.objNr {
		details = append(details, fmt.Sprintf("owner obj#:%d", f.ownerObjNr))
	}
	if f.depth > 0 {
		details = append(details, fmt.Sprintf("Next depth:%d", f.depth))
	}
	suffix := ""
	if len(details) > 0 {
		suffix = " (" + strings.Join(details, ", ") + ")"
	}
	return fmt.Sprintf("%s: active content: %s in %s%s", location, f.kind, activeContentSourceDescription(f), suffix)
}

func addActiveContentNotice(xRefTable *model.XRefTable, f activeContentFinding) {
	notice := model.NewValidationNotice(model.NoticePhaseValidate, model.NoticeSkipped, activeContentMessage(f), nil)
	notice.PageNumber = f.pageNr
	xRefTable.AddValidationNotice(notice)
}
