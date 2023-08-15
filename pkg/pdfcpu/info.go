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
	"fmt"
	"sort"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// handleInfoDict extracts relevant infoDict fields into the context.
func handleInfoDict(ctx *model.Context, d types.Dict) (err error) {

	for key, value := range d {

		switch key {

		case "Title":
			log.Write.Println("found Title")

		case "Author":
			log.Write.Println("found Author")
			// Record for stats.
			ctx.Author, err = ctx.DereferenceText(value)
			if err != nil {
				return err
			}
			ctx.Author = model.CSVSafeString(ctx.Author)

		case "Subject":
			log.Write.Println("found Subject")

		case "Keywords":
			log.Write.Println("found Keywords")

		case "Creator":
			log.Write.Println("found Creator")
			// Record for stats.
			ctx.Creator, err = ctx.DereferenceText(value)
			if err != nil {
				return err
			}
			ctx.Creator = model.CSVSafeString(ctx.Creator)

		case "Producer", "CreationDate", "ModDate":
			// pdfcpu will modify these as direct dict entries.
			log.Write.Printf("found %s", key)
			if indRef, ok := value.(types.IndirectRef); ok {
				// Get rid of these extra objects.
				ctx.Optimize.DuplicateInfoObjects[int(indRef.ObjectNumber)] = true
			}

		case "Trapped":
			log.Write.Println("found Trapped")

		default:
			log.Write.Printf("handleInfoDict: found out of spec entry %s %v\n", key, value)

		}
	}

	return nil
}

func ensureInfoDict(ctx *model.Context) error {

	// => 14.3.3 Document Information Dictionary

	// Optional:
	// Title                -
	// Author               -
	// Subject              -
	// Keywords             -
	// Creator              -
	// Producer		        modified by pdfcpu
	// CreationDate	        modified by pdfcpu
	// ModDate		        modified by pdfcpu
	// Trapped              -

	now := types.DateString(time.Now())

	v := "pdfcpu " + model.VersionStr

	if ctx.Info == nil {

		d := types.NewDict()
		d.InsertString("Producer", v)
		d.InsertString("CreationDate", now)
		d.InsertString("ModDate", now)

		ir, err := ctx.IndRefForNewObject(d)
		if err != nil {
			return err
		}

		ctx.Info = ir

		return nil
	}

	d, err := ctx.DereferenceDict(*ctx.Info)
	if err != nil || d == nil {
		return err
	}

	if err = handleInfoDict(ctx, d); err != nil {
		return err
	}

	d.Update("CreationDate", types.StringLiteral(now))
	d.Update("ModDate", types.StringLiteral(now))
	d.Update("Producer", types.StringLiteral(v))

	return nil
}

// Write the document info object for this PDF file.
func writeDocumentInfoDict(ctx *model.Context) error {

	log.Write.Printf("*** writeDocumentInfoDict begin: offset=%d ***\n", ctx.Write.Offset)

	// Note: The document info object is optional but pdfcpu ensures one.

	if ctx.Info == nil {
		log.Write.Printf("writeDocumentInfoObject end: No info object present, offset=%d\n", ctx.Write.Offset)
		return nil
	}

	log.Write.Printf("writeDocumentInfoObject: %s\n", *ctx.Info)

	o := *ctx.Info

	d, err := ctx.DereferenceDict(o)
	if err != nil || d == nil {
		return err
	}

	_, _, err = writeDeepObject(ctx, o)
	if err != nil {
		return err
	}

	log.Write.Printf("*** writeDocumentInfoDict end: offset=%d ***\n", ctx.Write.Offset)

	return nil
}

func appendEqualMediaAndCropBoxInfo(ss *[]string, pb model.PageBoundaries, unit string, currUnit types.DisplayUnit) {
	mb := pb.MediaBox()
	tb := pb.TrimBox()
	bb := pb.BleedBox()
	ab := pb.ArtBox()
	s := " = CropBox"

	if tb == nil || tb.Equals(*mb) {
		s += ", TrimBox"
	}
	if bb == nil || bb.Equals(*mb) {
		s += ", BleedBox"
	}
	if ab == nil || ab.Equals(*mb) {
		s += ", ArtBox"
	}

	*ss = append(*ss, fmt.Sprintf("  MediaBox (%s) %v %s", unit, mb.Format(currUnit), s))

	if tb != nil && !tb.Equals(*mb) {
		*ss = append(*ss, fmt.Sprintf("   TrimBox (%s) %v", unit, tb.Format(currUnit)))
	}
	if bb != nil && !bb.Equals(*mb) {
		*ss = append(*ss, fmt.Sprintf("  BleedBox (%s) %v", unit, bb.Format(currUnit)))
	}
	if ab != nil && !ab.Equals(*mb) {
		*ss = append(*ss, fmt.Sprintf("    ArtBox (%s) %v", unit, ab.Format(currUnit)))
	}
}

func trimBleedArtBoxString(cb, tb, bb, ab *types.Rectangle) string {
	s := ""
	if tb == nil || tb.Equals(*cb) {
		s += "= TrimBox"
	}
	if bb == nil || bb.Equals(*cb) {
		if len(s) == 0 {
			s += "= "
		} else {
			s += ", "
		}
		s += "BleedBox"
	}
	if ab == nil || ab.Equals(*cb) {
		if len(s) == 0 {
			s += "= "
		} else {
			s += ", "
		}
		s += "ArtBox"
	}
	return s
}

func appendNotEqualMediaAndCropBoxInfo(ss *[]string, pb model.PageBoundaries, unit string, currUnit types.DisplayUnit) {
	mb := pb.MediaBox()
	cb := pb.CropBox()
	tb := pb.TrimBox()
	bb := pb.BleedBox()
	ab := pb.ArtBox()

	s := trimBleedArtBoxString(cb, tb, bb, ab)
	*ss = append(*ss, fmt.Sprintf("   CropBox (%s) %v %s", unit, cb.Format(currUnit), s))

	if tb != nil && !tb.Equals(*mb) && !tb.Equals(*cb) {
		*ss = append(*ss, fmt.Sprintf("   TrimBox (%s) %v", unit, tb.Format(currUnit)))
	}
	if bb != nil && !bb.Equals(*mb) && !bb.Equals(*cb) {
		*ss = append(*ss, fmt.Sprintf("  BleedBox (%s) %v", unit, bb.Format(currUnit)))
	}
	if ab != nil && !ab.Equals(*mb) && !ab.Equals(*cb) {
		*ss = append(*ss, fmt.Sprintf("    ArtBox (%s) %v", unit, ab.Format(currUnit)))
	}
}

func appendPageBoxesInfo(ss *[]string, pb model.PageBoundaries, unit string, currUnit types.DisplayUnit, i int) {
	d := pb.CropBox().Dimensions()
	if pb.Rot%180 != 0 {
		d.Width, d.Height = d.Height, d.Width
	}
	or := "portrait"
	if d.Landscape() {
		or = "landscape"
	}
	s := fmt.Sprintf("rot=%+d orientation:%s", pb.Rot, or)
	*ss = append(*ss, fmt.Sprintf("Page %d: %s", i+1, s))
	mb := pb.MediaBox()
	cb := pb.CropBox()
	if cb == nil || mb != nil && mb.Equals(*cb) {
		appendEqualMediaAndCropBoxInfo(ss, pb, unit, currUnit)
		return
	}
	appendNotEqualMediaAndCropBoxInfo(ss, pb, unit, currUnit)
}

func pageInfo(info *PDFInfo, selectedPages types.IntSet) ([]string, error) {

	ss := []string{}

	if len(selectedPages) > 0 {
		for i, pb := range info.PageBoundaries {
			if _, found := selectedPages[i+1]; !found {
				continue
			}
			appendPageBoxesInfo(&ss, pb, info.UnitString, info.Unit, i)
		}
		return ss, nil
	}

	s := "Page size:"
	for d := range info.PageDimensions {
		dc := d.ConvertToUnit(info.Unit)
		ss = append(ss, fmt.Sprintf("%21s %.2f x %.2f %s", s, dc.Width, dc.Height, info.UnitString))
		s = ""
	}
	return ss, nil
}

type PDFInfo struct {
	FileName           string                 `json:"source,omitempty"`
	Version            string                 `json:"version"`
	PageCount          int                    `json:"pages"`
	PageBoundaries     []model.PageBoundaries `json:"-"`
	PageDimensions     map[types.Dim]bool     `json:"-"`
	Title              string                 `json:"title"`
	Author             string                 `json:"author"`
	Subject            string                 `json:"subject"`
	Producer           string                 `json:"producer"`
	Creator            string                 `json:"creator"`
	CreationDate       string                 `json:"creationDate"`
	ModificationDate   string                 `json:"modificationDate"`
	Keywords           []string               `json:"keywords"`
	Properties         map[string]string      `json:"properties"`
	Tagged             bool                   `json:"tagged"`
	Hybrid             bool                   `json:"hybrid"`
	Linearized         bool                   `json:"linearized"`
	UsingXRefStreams   bool                   `json:"usingXRefStreams"`
	UsingObjectStreams bool                   `json:"usingObjectStreams"`
	Watermarked        bool                   `json:"watermarked"`
	Thumbnails         bool                   `json:"thumbnails"`
	Form               bool                   `json:"form"`
	Signatures         bool                   `json:"signatures"`
	AppendOnly         bool                   `json:"appendOnly"`
	Outlines           bool                   `json:"bookmarks"`
	Names              bool                   `json:"names"`
	Encrypted          bool                   `json:"encrypted"`
	Permissions        int                    `json:"permissions"`
	Attachments        []model.Attachment     `json:"attachments,omitempty"`
	Unit               types.DisplayUnit      `json:"-"`
	UnitString         string                 `json:"-"`
}

func (info PDFInfo) renderKeywords(ss *[]string) error {
	for i, l := range info.Keywords {
		if i == 0 {
			*ss = append(*ss, fmt.Sprintf("%20s: %s", "Keywords", l))
			continue
		}
		*ss = append(*ss, fmt.Sprintf("%20s  %s", "", l))
	}
	return nil
}

func (info PDFInfo) renderProperties(ss *[]string) error {
	first := true
	for k, v := range info.Properties {
		if first {
			*ss = append(*ss, fmt.Sprintf("%20s: %s = %s", "Properties", k, v))
			first = false
			continue
		}
		*ss = append(*ss, fmt.Sprintf("%20s  %s = %s", "", k, v))
	}
	return nil
}

func (info PDFInfo) renderFlagsPart1(ss *[]string, separator string) {

	*ss = append(*ss, separator)

	s := "No"
	if info.Tagged {
		s = "Yes"
	}
	*ss = append(*ss, fmt.Sprintf("              Tagged: %s", s))

	s = "No"
	if info.Hybrid {
		s = "Yes"
	}
	*ss = append(*ss, fmt.Sprintf("              Hybrid: %s", s))

	s = "No"
	if info.Linearized {
		s = "Yes"
	}
	*ss = append(*ss, fmt.Sprintf("          Linearized: %s", s))

	s = "No"
	if info.UsingXRefStreams {
		s = "Yes"
	}
	*ss = append(*ss, fmt.Sprintf("  Using XRef streams: %s", s))

	s = "No"
	if info.UsingObjectStreams {
		s = "Yes"
	}
	*ss = append(*ss, fmt.Sprintf("Using object streams: %s", s))
}

func (info PDFInfo) renderFlagsPart2(ss *[]string, separator string) {

	s := "No"
	if info.Watermarked {
		s = "Yes"
	}
	*ss = append(*ss, fmt.Sprintf("         Watermarked: %s", s))

	s = "No"
	if info.Thumbnails {
		s = "Yes"
	}
	*ss = append(*ss, fmt.Sprintf("          Thumbnails: %s", s))

	s = "No"
	if info.Form {
		s = "Yes"
	}
	*ss = append(*ss, fmt.Sprintf("                Form: %s", s))
	if info.Form {
		if info.Signatures || info.AppendOnly {
			*ss = append(*ss, "     SignaturesExist: Yes")
			s = "No"
			if info.AppendOnly {
				s = "Yes"
			}
			*ss = append(*ss, fmt.Sprintf("          AppendOnly: %s", s))
		}
	}

	s = "No"
	if info.Outlines {
		s = "Yes"
	}
	*ss = append(*ss, fmt.Sprintf("            Outlines: %s", s))

	s = "No"
	if info.Names {
		s = "Yes"
	}
	*ss = append(*ss, fmt.Sprintf("               Names: %s", s))

	*ss = append(*ss, separator)

	s = "No"
	if info.Encrypted {
		s = "Yes"
	}
	*ss = append(*ss, fmt.Sprintf("%20s: %s", "Encrypted", s))
}

func (info *PDFInfo) renderFlags(ss *[]string, separator string) {
	info.renderFlagsPart1(ss, separator)
	info.renderFlagsPart2(ss, separator)
}

func (info *PDFInfo) renderPermissions(ss *[]string) {
	l := PermissionsList(info.Permissions)
	if len(l) == 1 {
		*ss = append(*ss, fmt.Sprintf("%20s: %s", "Permissions", l[0]))
	} else {
		*ss = append(*ss, fmt.Sprintf("%20s:", "Permissions"))
		*ss = append(*ss, l...)
	}
}

func (info *PDFInfo) renderAttachments(ss *[]string) {
	ss0 := []string{}
	for _, a := range info.Attachments {
		ss0 = append(ss0, a.FileName)
	}
	sort.Strings(ss0)
	*ss = append(*ss, ss0...)
}

// Info returns info about ctx.
func Info(ctx *model.Context, fileName string, selectedPages types.IntSet) (*PDFInfo, error) {

	info := &PDFInfo{FileName: fileName, Unit: ctx.Unit, UnitString: ctx.UnitString()}

	v := ctx.HeaderVersion
	if ctx.RootVersion != nil {
		v = ctx.RootVersion
	}
	info.Version = (*v).String()

	info.PageCount = ctx.PageCount

	// PageBoundaries for selected pages.
	pbs, err := ctx.PageBoundaries(selectedPages)
	if err != nil {
		return nil, err
	}
	info.PageBoundaries = pbs

	// Media box dimensions for all pages.
	pd, err := ctx.PageDims()
	if err != nil {
		return nil, err
	}
	m := map[types.Dim]bool{}
	for _, d := range pd {
		m[d] = true
	}
	info.PageDimensions = m

	info.Title = ctx.Title
	info.Subject = ctx.Subject
	info.Producer = ctx.Producer
	info.Creator = ctx.Creator
	info.CreationDate = ctx.CreationDate
	info.ModificationDate = ctx.ModDate

	kwl, err := KeywordsList(ctx.XRefTable)
	if err != nil {
		return nil, err
	}
	info.Keywords = kwl

	info.Properties = ctx.Properties
	info.Tagged = ctx.Tagged
	info.Hybrid = ctx.Read.Hybrid
	info.Linearized = ctx.Read.Linearized
	info.UsingXRefStreams = ctx.Read.UsingXRefStreams
	info.UsingObjectStreams = ctx.Read.UsingObjectStreams
	info.Watermarked = ctx.Watermarked
	info.Thumbnails = len(ctx.PageThumbs) > 0
	info.Form = ctx.Form != nil
	info.Signatures = ctx.SignatureExist
	info.AppendOnly = ctx.AppendOnly

	if ctx.E != nil {
		info.Permissions = ctx.E.P
	}

	aa, err := ctx.ListAttachments()
	if err != nil {
		return nil, err
	}
	info.Attachments = aa

	return info, nil
}

// ListInfo returns formatted info about ctx.
func ListInfo(info *PDFInfo, selectedPages types.IntSet) ([]string, error) {

	var separator = draw.HorSepLine([]int{44})

	var ss []string

	if info.FileName != "" {
		ss = append(ss, fmt.Sprintf("%20s: %s", "Source", info.FileName))
	}
	ss = append(ss, fmt.Sprintf("%20s: %s", "PDF version", info.Version))
	ss = append(ss, fmt.Sprintf("%20s: %d", "Page count", info.PageCount))

	pi, err := pageInfo(info, selectedPages)
	if err != nil {
		return nil, err
	}
	ss = append(ss, pi...)

	ss = append(ss, fmt.Sprint(separator))
	ss = append(ss, fmt.Sprintf("%20s: %s", "Title", info.Title))
	ss = append(ss, fmt.Sprintf("%20s: %s", "Author", info.Author))
	ss = append(ss, fmt.Sprintf("%20s: %s", "Subject", info.Subject))
	ss = append(ss, fmt.Sprintf("%20s: %s", "PDF Producer", info.Producer))
	ss = append(ss, fmt.Sprintf("%20s: %s", "Content creator", info.Creator))
	ss = append(ss, fmt.Sprintf("%20s: %s", "Creation date", info.CreationDate))
	ss = append(ss, fmt.Sprintf("%20s: %s", "Modification date", info.ModificationDate))

	info.renderKeywords(&ss)
	info.renderProperties(&ss)
	info.renderFlags(&ss, separator)
	info.renderPermissions(&ss)
	info.renderAttachments(&ss)

	return ss, nil
}
