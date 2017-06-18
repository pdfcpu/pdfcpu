package types

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
