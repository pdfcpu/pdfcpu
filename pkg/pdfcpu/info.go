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
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/log"
)

func csvSafeString(s string) string {
	return strings.Replace(s, ";", ",", -1)
}

// handleInfoDict extracts relevant infoDict fields into the context.
func (ctx *Context) handleInfoDict(d Dict) (err error) {

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
			ctx.Author = csvSafeString(ctx.Author)

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
			ctx.Creator = csvSafeString(ctx.Creator)

		case "Producer", "CreationDate", "ModDate":
			// pdfcpu will modify these as direct dict entries.
			log.Write.Printf("found %s", key)
			if indRef, ok := value.(IndirectRef); ok {
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

func (ctx *Context) ensureInfoDict() error {

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

	now := DateString(time.Now())

	v := "pdfcpu " + VersionStr

	if ctx.Info == nil {

		d := NewDict()
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

	if err = ctx.handleInfoDict(d); err != nil {
		return err
	}

	d.Update("CreationDate", StringLiteral(now))
	d.Update("ModDate", StringLiteral(now))
	d.Update("Producer", StringLiteral(v))

	return nil
}

// Write the document info object for this PDF file.
func (ctx *Context) writeDocumentInfoDict() error {

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

func appendEqualMediaAndCropBoxInfo(ss *[]string, pb PageBoundaries, unit string, currUnit DisplayUnit) {
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

func trimBleedArtBoxString(cb, tb, bb, ab *Rectangle) string {
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

func appendNotEqualMediaAndCropBoxInfo(ss *[]string, pb PageBoundaries, unit string, currUnit DisplayUnit) {
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

func appendPageBoxesInfo(ss *[]string, pb PageBoundaries, unit string, currUnit DisplayUnit, i int) {
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

func (ctx *Context) pageInfo(selectedPages IntSet) ([]string, error) {
	unit := ctx.unit()
	if len(selectedPages) > 0 {
		// TODO ctx.PageBoundaries(selectedPages)
		pbs, err := ctx.PageBoundaries()
		if err != nil {
			return nil, err
		}
		ss := []string{}
		for i, pb := range pbs {
			if _, found := selectedPages[i+1]; !found {
				continue
			}
			appendPageBoxesInfo(&ss, pb, unit, ctx.Unit, i)
		}
		return ss, nil
	}

	pd, err := ctx.PageDims()
	if err != nil {
		return nil, err
	}

	m := map[Dim]bool{}
	for _, d := range pd {
		m[d] = true
	}

	ss := []string{}
	s := "Page size:"
	for d := range m {
		dc := ctx.convertToUnit(d)
		ss = append(ss, fmt.Sprintf("%21s %.2f x %.2f %s", s, dc.Width, dc.Height, unit))
		s = ""
	}

	return ss, nil
}

// InfoDigest returns info about ctx.
func (ctx *Context) InfoDigest(selectedPages IntSet) ([]string, error) {
	var separator = "............................................"
	var ss []string
	v := ctx.HeaderVersion
	if ctx.RootVersion != nil {
		v = ctx.RootVersion
	}
	ss = append(ss, fmt.Sprintf("%20s: %s", "PDF version", v))
	ss = append(ss, fmt.Sprintf("%20s: %d", "Page count", ctx.PageCount))

	pi, err := ctx.pageInfo(selectedPages)
	if err != nil {
		return nil, err
	}
	ss = append(ss, pi...)

	ss = append(ss, fmt.Sprintf(separator))
	ss = append(ss, fmt.Sprintf("%20s: %s", "Title", ctx.Title))
	ss = append(ss, fmt.Sprintf("%20s: %s", "Author", ctx.Author))
	ss = append(ss, fmt.Sprintf("%20s: %s", "Subject", ctx.Subject))
	ss = append(ss, fmt.Sprintf("%20s: %s", "PDF Producer", ctx.Producer))
	ss = append(ss, fmt.Sprintf("%20s: %s", "Content creator", ctx.Creator))
	ss = append(ss, fmt.Sprintf("%20s: %s", "Creation date", ctx.CreationDate))
	ss = append(ss, fmt.Sprintf("%20s: %s", "Modification date", ctx.ModDate))

	if err := ctx.addKeywordsToInfoDigest(&ss); err != nil {
		return nil, err
	}

	if err := ctx.addPropertiesToInfoDigest(&ss); err != nil {
		return nil, err
	}

	ss = append(ss, separator)

	s := "No"
	if ctx.Tagged {
		s = "Yes"
	}
	ss = append(ss, fmt.Sprintf("              Tagged: %s", s))

	s = "No"
	if ctx.Read.Hybrid {
		s = "Yes"
	}
	ss = append(ss, fmt.Sprintf("              Hybrid: %s", s))

	s = "No"
	if ctx.Read.Linearized {
		s = "Yes"
	}
	ss = append(ss, fmt.Sprintf("          Linearized: %s", s))

	s = "No"
	if ctx.Read.UsingXRefStreams {
		s = "Yes"
	}
	ss = append(ss, fmt.Sprintf("  Using XRef streams: %s", s))

	s = "No"
	if ctx.Read.UsingObjectStreams {
		s = "Yes"
	}
	ss = append(ss, fmt.Sprintf("Using object streams: %s", s))

	s = "No"
	if ctx.Watermarked {
		s = "Yes"
	}
	ss = append(ss, fmt.Sprintf("         Watermarked: %s", s))

	s = "No"
	if len(ctx.PageThumbs) > 0 {
		s = "Yes"
	}
	ss = append(ss, fmt.Sprintf("          Thumbnails: %s", s))

	s = "No"
	if ctx.AcroForm != nil {
		s = "Yes"
	}
	ss = append(ss, fmt.Sprintf("            Acroform: %s", s))
	if ctx.AcroForm != nil {
		if ctx.SignatureExist {
			ss = append(ss, "     SignaturesExist: Yes")
			s = "No"
			if ctx.AppendOnly {
				s = "Yes"
			}
			ss = append(ss, fmt.Sprintf("          AppendOnly: %s", s))
		}
	}

	ss = append(ss, separator)

	s = "No"
	if ctx.Encrypt != nil {
		s = "Yes"
	}
	ss = append(ss, fmt.Sprintf("%20s: %s", "Encrypted", s))

	ctx.addPermissionsToInfoDigest(&ss)

	if err := ctx.addAttachmentsToInfoDigest(&ss); err != nil {
		return nil, err
	}

	return ss, nil
}
