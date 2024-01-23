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
	"os"
	"path/filepath"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

func logWriteStats(ctx *model.Context) {

	xRefTable := ctx.XRefTable

	if len(xRefTable.Table) != *xRefTable.Size {
		if count, mstr := xRefTable.MissingObjects(); count > 0 {
			log.Stats.Printf("%d missing objects: %s\n", count, *mstr)
		}
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
	log.Stats.Printf("%d original empty xref entries:\n%s", l, str)

	// Duplicate font objects
	l, str = ctx.Optimize.DuplicateFontObjectsString()
	log.Stats.Printf("%d original redundant font entries: %s", l, str)

	// Duplicate image objects
	l, str = ctx.Optimize.DuplicateImageObjectsString()
	log.Stats.Printf("%d original redundant image entries: %s", l, str)

	// Duplicate info objects
	l, str = ctx.Optimize.DuplicateInfoObjectsString()
	log.Stats.Printf("%d original redundant info entries: %s", l, str)

	// ObjectStreams
	l, str = ctx.Read.ObjectStreamsString()
	log.Stats.Printf("%d original objectStream entries: %s", l, str)

	// XRefStreams
	l, str = ctx.Read.XRefStreamsString()
	log.Stats.Printf("%d original xrefStream entries: %s", l, str)

	// Linearization objects
	l, str = ctx.LinearizationObjsString()
	log.Stats.Printf("%d original linearization entries: %s", l, str)
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

func statsLine(ctx *model.Context) *string {

	xRefTable := ctx.XRefTable

	version := xRefTable.HeaderVersion.String()
	if xRefTable.RootVersion != nil {
		version = fmt.Sprintf("%s,%s", version, xRefTable.RootVersion.String())
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
		xRefTable.Stats.UsesRootAttr(model.RootVersion),
		xRefTable.Stats.UsesRootAttr(model.RootExtensions),
		xRefTable.Stats.UsesRootAttr(model.RootPageLabels),
		xRefTable.Stats.UsesRootAttr(model.RootNames),
		xRefTable.Stats.UsesRootAttr(model.RootDests),
		xRefTable.Stats.UsesRootAttr(model.RootViewerPrefs),
		xRefTable.Stats.UsesRootAttr(model.RootPageLayout),
		xRefTable.Stats.UsesRootAttr(model.RootPageMode),
		xRefTable.Stats.UsesRootAttr(model.RootOutlines),
		xRefTable.Stats.UsesRootAttr(model.RootThreads),
		xRefTable.Stats.UsesRootAttr(model.RootOpenAction),
		xRefTable.Stats.UsesRootAttr(model.RootAA),
		xRefTable.Stats.UsesRootAttr(model.RootURI),
		xRefTable.Stats.UsesRootAttr(model.RootAcroForm),
		xRefTable.Stats.UsesRootAttr(model.RootMetadata),
		xRefTable.Stats.UsesRootAttr(model.RootStructTreeRoot),
		xRefTable.Stats.UsesRootAttr(model.RootMarkInfo),
		xRefTable.Stats.UsesRootAttr(model.RootLang),
		xRefTable.Stats.UsesRootAttr(model.RootSpiderInfo),
		xRefTable.Stats.UsesRootAttr(model.RootOutputIntents),
		xRefTable.Stats.UsesRootAttr(model.RootPieceInfo),
		xRefTable.Stats.UsesRootAttr(model.RootOCProperties),
		xRefTable.Stats.UsesRootAttr(model.RootPerms),
		xRefTable.Stats.UsesRootAttr(model.RootLegal),
		xRefTable.Stats.UsesRootAttr(model.RootRequirements),
		xRefTable.Stats.UsesRootAttr(model.RootCollection),
		xRefTable.Stats.UsesRootAttr(model.RootNeedsRendering),
		xRefTable.Stats.UsesPageAttr(model.PageLastModified),
		xRefTable.Stats.UsesPageAttr(model.PageResources),
		xRefTable.Stats.UsesPageAttr(model.PageMediaBox),
		xRefTable.Stats.UsesPageAttr(model.PageCropBox),
		xRefTable.Stats.UsesPageAttr(model.PageBleedBox),
		xRefTable.Stats.UsesPageAttr(model.PageTrimBox),
		xRefTable.Stats.UsesPageAttr(model.PageArtBox),
		xRefTable.Stats.UsesPageAttr(model.PageBoxColorInfo),
		xRefTable.Stats.UsesPageAttr(model.PageContents),
		xRefTable.Stats.UsesPageAttr(model.PageRotate),
		xRefTable.Stats.UsesPageAttr(model.PageGroup),
		xRefTable.Stats.UsesPageAttr(model.PageThumb),
		xRefTable.Stats.UsesPageAttr(model.PageB),
		xRefTable.Stats.UsesPageAttr(model.PageDur),
		xRefTable.Stats.UsesPageAttr(model.PageTrans),
		xRefTable.Stats.UsesPageAttr(model.PageAnnots),
		xRefTable.Stats.UsesPageAttr(model.PageAA),
		xRefTable.Stats.UsesPageAttr(model.PageMetadata),
		xRefTable.Stats.UsesPageAttr(model.PagePieceInfo),
		xRefTable.Stats.UsesPageAttr(model.PageStructParents),
		xRefTable.Stats.UsesPageAttr(model.PageID),
		xRefTable.Stats.UsesPageAttr(model.PagePZ),
		xRefTable.Stats.UsesPageAttr(model.PageSeparationInfo),
		xRefTable.Stats.UsesPageAttr(model.PageTabs),
		xRefTable.Stats.UsesPageAttr(model.PageTemplateInstantiated),
		xRefTable.Stats.UsesPageAttr(model.PagePresSteps),
		xRefTable.Stats.UsesPageAttr(model.PageUserUnit),
		xRefTable.Stats.UsesPageAttr(model.PageVP))

	return &line
}

// AppendStatsFile appends a stats line for this xRefTable to the configured csv file name.
func AppendStatsFile(ctx *model.Context) error {

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
			return err
		}

	}

	defer func() {
		file.Close()
	}()

	_, err = file.WriteString(*statsLine(ctx))

	return err
}
