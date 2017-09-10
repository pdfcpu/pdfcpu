package write

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

func logWriteStats(ctx *types.PDFContext) {

	xRefTable := ctx.XRefTable

	if len(xRefTable.Table) != *xRefTable.Size {
		missing, str := xRefTable.MissingObjects()
		logXRef.Printf("%d missing objects: %s\n", missing, *str)
	}

	var nonRefObjs []int

	for i := 0; i < *xRefTable.Size; i++ {

		entry, found := xRefTable.Find(i)
		if !found || entry.Free || ctx.Write.HasWriteOffset(i) {
			continue
		}

		nonRefObjs = append(nonRefObjs, i)

	}

	// Non referenced objects
	ctx.Optimize.NonReferencedObjs = nonRefObjs
	l, str := ctx.Optimize.NonReferencedObjsString()
	logXRef.Printf("%d original empty xref entries:\n%s", l, str)

	// Duplicate font objects
	l, str = ctx.Optimize.DuplicateFontObjectsString()
	logXRef.Printf("%d original redundant font entries: %s", l, str)

	// Duplicate image objects
	l, str = ctx.Optimize.DuplicateImageObjectsString()
	logXRef.Printf("%d original redundant image entries: %s", l, str)

	// Duplicate info objects
	l, str = ctx.Optimize.DuplicateInfoObjectsString()
	logXRef.Printf("%d original redundant info entries: %s", l, str)

	// ObjectStreams
	l, str = ctx.Read.ObjectStreamsString()
	logXRef.Printf("%d original objectStream entries: %s", l, str)

	// XRefStreams
	l, str = ctx.Read.XRefStreamsString()
	logXRef.Printf("%d original xrefStream entries: %s", l, str)

	// Linearization objects
	l, str = ctx.LinearizationObjsString()
	logXRef.Printf("%d original linearization entries: %s", l, str)
}

func statsHeadLine() *string {

	hl := "name;version;author;creator;producer;src_size (bin|text);src_bin:imgs|fonts|other;dest_size (bin|text);dest_bin:imgs|fonts|other;"
	hl += "linearized;hybrid;xrefstr;objstr;pages;objs;missing;garbage;"
	hl += "R_Version;R_Extensions;R_PageLabels;R_Names;R_Dests;R_ViewerPrefs;R_PageLayout;R_PageMode;"
	hl += "R_Outlines;R_Threads;R_OpenAction;R_AA;R_URI;R_AcroForm;R_Metadata;R_StructTreeRoot;R_MarkInfo;"
	hl += "R_Lang;R_SpiderInfo;R_OutputIntents;R_PieceInfo;R_OCProperties;R_Perms;R_Legal;R_Requirements;"
	hl += "R_Collection;R_NeedsRendering;"
	hl += "P_LastModified;P_Resources;P_MediaBox;P_CropBox;P_BleedBox;P_TrimBox;P_ArtBox;"
	hl += "P_BoxColorInfo;P_Contents;P_Rotate;P_Group;P_Thumb;P_B;P_Dur;P_Trans;P_Annots;"
	hl += "P_AA;P_Metadata;P_PieceInfo;P_StructParents;P_ID;P_PZ;P_SeparationInfo;P_Tabs;"
	hl += "P_TemplateInstantiated;P_PresSteps;P_UserUnit;P_VP;\n"

	return &hl
}

func statsLine(ctx *types.PDFContext) *string {

	xRefTable := ctx.XRefTable

	version := types.VersionString(*xRefTable.HeaderVersion)
	if xRefTable.RootVersion != nil {
		version = fmt.Sprintf("%s,%s", version, types.VersionString(*xRefTable.RootVersion))
	}

	sourceFileSize := ctx.Read.FileSize
	sourceBinarySize := ctx.Read.BinaryTotalSize
	sourceNonBinarySize := sourceFileSize - sourceBinarySize

	sourceSizeStats := fmt.Sprintf("%s (%4.1f%% | %4.1f%%)",
		types.ByteSize(sourceFileSize),
		float32(sourceBinarySize)/float32(sourceFileSize)*100,
		float32(sourceNonBinarySize)/float32(sourceFileSize)*100)

	sourceBinaryImageSize := ctx.Read.BinaryImageSize + ctx.Read.BinaryImageDuplSize
	sourceBinaryFontSize := ctx.Read.BinaryFontSize + ctx.Read.BinaryFontDuplSize
	sourceBinaryOtherSize := sourceBinarySize - sourceBinaryImageSize - sourceBinaryFontSize

	sourceBinaryStats := fmt.Sprintf("%4.1f%% | %4.1f%% | %4.1f%%",
		float32(sourceBinaryImageSize)/float32(sourceBinarySize)*100,
		float32(sourceBinaryFontSize)/float32(sourceBinarySize)*100,
		float32(sourceBinaryOtherSize)/float32(sourceBinarySize)*100)

	destFileSize := ctx.Write.FileSize
	destBinarySize := ctx.Write.BinaryTotalSize
	destNonBinarySize := destFileSize - destBinarySize

	destSizeStats := fmt.Sprintf("%s (%4.1f%% | %4.1f%%)",
		types.ByteSize(destFileSize),
		float32(destBinarySize)/float32(destFileSize)*100,
		float32(destNonBinarySize)/float32(destFileSize)*100)

	destBinaryImageSize := ctx.Write.BinaryImageSize
	destBinaryFontSize := ctx.Write.BinaryFontSize
	destBinaryOtherSize := destBinarySize - destBinaryImageSize - destBinaryFontSize

	destBinaryStats := fmt.Sprintf("%4.1f%% | %4.1f%% | %4.1f%%",
		float32(destBinaryImageSize)/float32(destBinarySize)*100,
		float32(destBinaryFontSize)/float32(destBinarySize)*100,
		float32(destBinaryOtherSize)/float32(destBinarySize)*100)

	var missingObjs string
	if count, mstr := xRefTable.MissingObjects(); count > 0 {
		missingObjs = fmt.Sprintf("%d:%s", count, *mstr)
	}

	var nonreferencedObjs string
	if len(ctx.Optimize.NonReferencedObjs) > 0 {
		var s []string
		for _, o := range ctx.Optimize.NonReferencedObjs {
			s = append(s, fmt.Sprintf("%d", o))
		}
		nonreferencedObjs = fmt.Sprintf("%d:%s", len(ctx.Optimize.NonReferencedObjs), strings.Join(s, ","))
	}

	line := fmt.Sprintf("%s;%s;%s;%s;%s;%s;%s;%s;%s;%v;%v;%v;%v;%d;%d;%s;%s;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v\n",
		filepath.Base(ctx.Read.FileName),
		version,
		xRefTable.Author,
		xRefTable.Creator,
		xRefTable.Producer,
		sourceSizeStats,
		sourceBinaryStats,
		destSizeStats,
		destBinaryStats,
		ctx.Read.Linearized,
		ctx.Read.Hybrid,
		ctx.Read.UsingXRefStreams,
		ctx.Read.UsingObjectStreams,
		xRefTable.PageCount,
		*xRefTable.Size,
		missingObjs,
		nonreferencedObjs,
		xRefTable.Stats.UsesRootAttr(types.RootVersion),        // ok
		xRefTable.Stats.UsesRootAttr(types.RootExtensions),     // ok
		xRefTable.Stats.UsesRootAttr(types.RootPageLabels),     // ok
		xRefTable.Stats.UsesRootAttr(types.RootNames),          // ok
		xRefTable.Stats.UsesRootAttr(types.RootDests),          // ok
		xRefTable.Stats.UsesRootAttr(types.RootViewerPrefs),    // ok
		xRefTable.Stats.UsesRootAttr(types.RootPageLayout),     // ok
		xRefTable.Stats.UsesRootAttr(types.RootPageMode),       // ok
		xRefTable.Stats.UsesRootAttr(types.RootOutlines),       // ok
		xRefTable.Stats.UsesRootAttr(types.RootThreads),        // ok
		xRefTable.Stats.UsesRootAttr(types.RootOpenAction),     // ok
		xRefTable.Stats.UsesRootAttr(types.RootAA),             // ok
		xRefTable.Stats.UsesRootAttr(types.RootURI),            // ok
		xRefTable.Stats.UsesRootAttr(types.RootAcroForm),       // ok
		xRefTable.Stats.UsesRootAttr(types.RootMetadata),       // ok
		xRefTable.Stats.UsesRootAttr(types.RootStructTreeRoot), // ok
		xRefTable.Stats.UsesRootAttr(types.RootMarkInfo),       // ok
		xRefTable.Stats.UsesRootAttr(types.RootLang),           // ok
		xRefTable.Stats.UsesRootAttr(types.RootSpiderInfo),     // ok
		xRefTable.Stats.UsesRootAttr(types.RootOutputIntents),  // ok
		xRefTable.Stats.UsesRootAttr(types.RootPieceInfo),      // ok
		xRefTable.Stats.UsesRootAttr(types.RootOCProperties),   // ok
		xRefTable.Stats.UsesRootAttr(types.RootPerms),          // ok
		xRefTable.Stats.UsesRootAttr(types.RootLegal),          // ok
		xRefTable.Stats.UsesRootAttr(types.RootRequirements),   // ok
		xRefTable.Stats.UsesRootAttr(types.RootCollection),     // ok
		xRefTable.Stats.UsesRootAttr(types.RootNeedsRendering), // ok
		xRefTable.Stats.UsesPageAttr(types.PageLastModified),   // ok
		xRefTable.Stats.UsesPageAttr(types.PageResources),      // ok
		xRefTable.Stats.UsesPageAttr(types.PageMediaBox),       // ok
		xRefTable.Stats.UsesPageAttr(types.PageCropBox),        // ok
		xRefTable.Stats.UsesPageAttr(types.PageBleedBox),       // ok
		xRefTable.Stats.UsesPageAttr(types.PageTrimBox),        // ok
		xRefTable.Stats.UsesPageAttr(types.PageArtBox),         // ok
		xRefTable.Stats.UsesPageAttr(types.PageBoxColorInfo),
		xRefTable.Stats.UsesPageAttr(types.PageContents), // ok
		xRefTable.Stats.UsesPageAttr(types.PageRotate),
		xRefTable.Stats.UsesPageAttr(types.PageGroup), // ok
		xRefTable.Stats.UsesPageAttr(types.PageThumb), // ok
		xRefTable.Stats.UsesPageAttr(types.PageB),
		xRefTable.Stats.UsesPageAttr(types.PageDur),
		xRefTable.Stats.UsesPageAttr(types.PageTrans),
		xRefTable.Stats.UsesPageAttr(types.PageAnnots), // ok
		xRefTable.Stats.UsesPageAttr(types.PageAA),
		xRefTable.Stats.UsesPageAttr(types.PageMetadata),
		xRefTable.Stats.UsesPageAttr(types.PagePieceInfo),
		xRefTable.Stats.UsesPageAttr(types.PageStructParents),
		xRefTable.Stats.UsesPageAttr(types.PageID),
		xRefTable.Stats.UsesPageAttr(types.PagePZ),
		xRefTable.Stats.UsesPageAttr(types.PageSeparationInfo),
		xRefTable.Stats.UsesPageAttr(types.PageTabs),
		xRefTable.Stats.UsesPageAttr(types.PageTemplateInstantiated),
		xRefTable.Stats.UsesPageAttr(types.PagePresSteps),
		xRefTable.Stats.UsesPageAttr(types.PageUserUnit),
		xRefTable.Stats.UsesPageAttr(types.PageVP))

	return &line
}

// AppendStatsFile appends a stats line for this xRefTable to the configured csv file name.
func AppendStatsFile(ctx *types.PDFContext) (err error) {

	fileName := ctx.StatsFileName

	// if file does not exist, create file
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, 0600)

	if err != nil {

		if os.IsExist(err) {
			return errors.Errorf("can't open %s\n%s", fileName, err)
		}

		file, err = os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return errors.Errorf("can't create %s\n%s", fileName, err)
		}

		_, err = file.WriteString(*statsHeadLine())
		if err != nil {
			return
		}

	}

	defer func() {
		file.Close()
	}()

	_, err = file.WriteString(*statsLine(ctx))

	return
}
