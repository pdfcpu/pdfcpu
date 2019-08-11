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

import "github.com/pdfcpu/pdfcpu/pkg/log"

// The PDF root object fields.
const (
	RootVersion = iota
	RootExtensions
	RootPageLabels
	RootNames
	RootDests
	RootViewerPrefs
	RootPageLayout
	RootPageMode
	RootOutlines
	RootThreads
	RootOpenAction
	RootAA
	RootURI
	RootAcroForm
	RootMetadata
	RootStructTreeRoot
	RootMarkInfo
	RootLang
	RootSpiderInfo
	RootOutputIntents
	RootPieceInfo
	RootOCProperties
	RootPerms
	RootLegal
	RootRequirements
	RootCollection
	RootNeedsRendering
)

// The PDF page object fields.
const (
	PageLastModified = iota
	PageResources
	PageMediaBox
	PageCropBox
	PageBleedBox
	PageTrimBox
	PageArtBox
	PageBoxColorInfo
	PageContents
	PageRotate
	PageGroup
	PageThumb
	PageB
	PageDur
	PageTrans
	PageAnnots
	PageAA
	PageMetadata
	PagePieceInfo
	PageStructParents
	PageID
	PagePZ
	PageSeparationInfo
	PageTabs
	PageTemplateInstantiated
	PagePresSteps
	PageUserUnit
	PageVP
)

// PDFStats is a container for stats.
type PDFStats struct {
	// Used root attributes
	rootAttrs IntSet
	// Used page attributes
	pageAttrs IntSet
}

// NewPDFStats returns a new PDFStats object.
func NewPDFStats() PDFStats {
	return PDFStats{rootAttrs: IntSet{}, pageAttrs: IntSet{}}
}

// AddRootAttr adds the occurrence of a field with given name to the rootAttrs set.
func (stats PDFStats) AddRootAttr(name int) {
	stats.rootAttrs[name] = true
}

// UsesRootAttr returns true if a field with given name is contained in the rootAttrs set.
func (stats PDFStats) UsesRootAttr(name int) bool {
	return stats.rootAttrs[name]
}

// AddPageAttr adds the occurrence of a field with given name to the pageAttrs set.
func (stats PDFStats) AddPageAttr(name int) {
	stats.pageAttrs[name] = true
}

// UsesPageAttr returns true if a field with given name is contained in the pageAttrs set.
func (stats PDFStats) UsesPageAttr(name int) bool {
	return stats.pageAttrs[name]
}

// ValidationTimingStats prints processing time stats for validation.
func ValidationTimingStats(dur1, dur2, dur float64) {
	log.Stats.Println("Timing:")
	log.Stats.Printf("read                 : %6.3fs  %4.1f%%\n", dur1, dur1/dur*100)
	log.Stats.Printf("validate             : %6.3fs  %4.1f%%\n", dur2, dur2/dur*100)
	log.Stats.Printf("total processing time: %6.3fs\n\n", dur)
}

// TimingStats prints processing time stats for an operation.
func TimingStats(op string, durRead, durVal, durOpt, durWrite, durTotal float64) {
	log.Stats.Println("Timing:")
	log.Stats.Printf("read                 : %6.3fs  %4.1f%%\n", durRead, durRead/durTotal*100)
	log.Stats.Printf("validate             : %6.3fs  %4.1f%%\n", durVal, durVal/durTotal*100)
	log.Stats.Printf("optimize             : %6.3fs  %4.1f%%\n", durOpt, durOpt/durTotal*100)
	log.Stats.Printf("%-21s: %6.3fs  %4.1f%%\n", op, durWrite, durWrite/durTotal*100)
	log.Stats.Printf("total processing time: %6.3fs\n\n", durTotal)
}
