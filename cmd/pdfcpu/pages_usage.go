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

package main

const (
	usageLongNUp = `Rearrange existing PDF pages or images into a sequence of page grids.
This reduces the number of pages and therefore the required print time.
If the input is one imageFile a single page n-up PDF gets generated.

      pages ... inFile only, please refer to "pdfcpu selectedpages"
description ... dimensions, formsize, orientation
    outFile ... output PDF file, use - to write to stdout
          n ... the n-Up value (see below for details)
     inFile ... input PDF file, use - to read from stdin
 imageFiles ... input image file(s)

                              portrait landscape
 Supported values for n: 2 ...  1x2       2x1
                         3 ...  1x3       3x1
                         4 ...  2x2
                         8 ...  2x4       4x2
                         9 ...  3x3
                        12 ...  3x4       4x3
                        16 ...  4x4

    <description> is a comma separated configuration string containing:

    optional entries:

        (defaults: "di:595 842, form:A4, or:rd, bo:on, ma:3, enforce:on")

    dimensions:      (width,height) in given display unit eg. '400 200'

    formsize:        The output sheet size, eg. A4, Letter, Legal...
                     Append 'L' to enforce landscape mode. (eg. A3L)
                     Append 'P' to enforce portrait mode. (eg. TabloidP)
                     Only one of dimensions or formsize is allowed.
                     Please refer to "pdfcpu paper" for a comprehensive list of defined paper sizes.
                     "papersize" is also accepted.

    orientation:     one of rd ... right down (=default)
                            dr ... down right
                            ld ... left down
                            dl ... down left
                     Orientation applies to PDF input files only.

    enforce:         enforce best-fit orientation of individual content (on/off, true/false, t/f).

    border:          Print border (on/off, true/false, t/f)

    margin:          for n-up content: float >= 0 in given display unit

    backgroundcolor: background color for margin > 0.
                     "bgcolor" is also accepted.

All configuration string parameters support completion.

Examples: pdfcpu nup out.pdf 4 in.pdf
           Rearrange pages of in.pdf into 2x2 grids and write result to out.pdf using the default orientation
           and default paper size A4. in.pdf's page size will be preserved.

          pdfcpu nup out.pdf 6 in.pdf --pages=3-
           Rearrange selected pages of in.pdf (all pages starting with page 3) into 3x2 grids and
           write result to out.pdf using the default orientation and default paper size A4.
           in.pdf's page size will be preserved.

          pdfcpu nup out.pdf 9 logo.jpg
           Arrange instances of logo.jpg into a 3x3 grid and write result to out.pdf using the A4 default form size.

          pdfcpu nup 'form:Tabloid' out.pdf 4 *.jpg
           Rearrange all jpg files into 2x2 grids and write result to out.pdf using the Tabloid form size
           and the default orientation.

Pipeline example:
          aws s3 cp s3://acme-print/handout.pdf - \
             | pdfcpu nup - 4 - \
             | aws s3 cp - s3://acme-print/handout-4up.pdf
`

	usageLongBooklet = `Arrange a sequence of pages onto larger sheets of paper for a small book or zine.

      pages       ... for inFile only, please refer to "pdfcpu selectedpages"
      description ... dimensions, formsize, border, margin
      outFile     ... output PDF file, use - to write to stdout
      n           ... booklet style (2, 4, 6, 8)
      inFile      ... input PDF file, use - to read from stdin
      imageFiles  ... input image file(s)

There are several styles of booklet, depending on your page/input and sheet/output size,
the edge along which your booklet will be bound,
and your preferred method for creating the booklet.

For assembly instructions for each type, see: https://pdfcpu.io/generate/booklet

Pipeline example:
   aws s3 cp s3://acme-print/zine.pdf - \
      | pdfcpu booklet - 4 - \
      | aws s3 cp - s3://acme-print/zine-booklet.pdf

n=2: This is the simplest case and the most common for those printing at home.
Two of your pages fit on one side of a sheet (eg statement on letter, A5 on A4)
Assemble by printing on both sides (odd pages on the front and even pages on the back) and folding down the middle.

n=4: Four of your pages fit on one side of a sheet (eg statement on ledger, A5 on A3, A6 on A4).

When printing 4-up, your booklet can be bound either along the long-edge (for portrait this is the left side of the paper, for landscape the top)
or the short-edge (for portrait this is the top of the paper, for landscape the left side).
Using a different binding will change the ordering of the pages on the sheet.
You can set long or short-edge with the 'binding' option.

In 4-up printing, the sets of pages on the bottom of the sheet are rotated so that the cut side of the
paper is on the bottom of the booklet for every page (for the default portrait, long-edge binding case.
Similar rotation logic applies for the other three orientations).
Having the cut edge always on bottom makes for more uniform pages within the book and less work in trimming.

The btype=advanced is a special method for assembling, only for 4-up booklets.
Printers that are used to collating first and then cutting may prefer this method.

n=6: Six of your pages fit on one side of a sheet. This produces an unusual sized booklet.

   Only available for portrait, long-edge orientation.

n=8: Eight of your pages fit on one side of a sheet (eg A6 on A3).

   Only available for portrait, long-edge orientation.

Perfect binding is a special type of booklet. The main difference is that the binding is glued into a spine,
meaning that the pages are cut along the binding and not folded as in the other forms of booklet.
This results in a different page ordering on the sheet than the other methods. If you intend to perfect bind your booklet,
use btype=perfectbound.

There is also an option to use signatures, a bookbinding method useful for books with higher page counts.
In this method of binding, you arrange your folios (sheets folded in half) in groups of 'foliosize'.
Each group is called a signature. You then stack the signatures together to form the book.
For example, you can bind your paper in groups of eight sheets (foliosize=8), so that each signature containing 32 pages of your book.
For such a multi folio booklet set 'multifolio:on' and 'foliosize', which defaults to 8.
The last signature may be shorter, e.g. for a booklet of 120 pages with signature size=16 (foliosize=4) will have 7 complete signatures and a final signature of only 8 pages.


                             portrait landscape
 Possible values for n: 2 ...  1x2       --
                        4 ...  2x2       2x2
                        6 ...  2x3       --
                        8 ...  2x4       --

<description> is a comma separated configuration string containing these optional entries:

   (defaults: "dim:595 842, formsize:A4, btype: booklet, binding: long, multifolio: false, border:off, guides:off, margin:0")

   dimensions:       (width,height) of the output sheet in given display unit eg. '400 200'
   formsize:         The output sheet size, eg. A4, Letter, Legal...
                     Append 'L' to enforce landscape mode. (eg. A3L)
                     Append 'P' to enforce portrait mode. (eg. TabloidP)
                     Only one of dimensions or formsize is allowed.
                     Please refer to "pdfcpu paper" for a comprehensive list of defined paper sizes.
                     "papersize" is also accepted.
   btype:            The method for arranging pages into a booklet. (booklet, bookletadvanced, perfectbound)
   binding:          The edge of the paper which has the binding. (long, short)
   multifolio:       Generate multi folio booklet (on/off, true/false, t/f) for n=2 and PDF input only.
   foliosize:        folio size for multi folio booklets only (default:8)
   border:           Print border (on/off, true/false, t/f)
   guides:           Print folding and cutting lines (on/off, true/false, t/f)
   margin:           Apply content margin (float >= 0 in given display unit)
   backgroundcolor:  sheet background color for margin > 0.
                     "bgcolor" is also accepted.

All configuration string parameters support completion.

Examples:

   pdfcpu booklet 'formsize:Letter' out.pdf 2 in.pdf
      Arrange pages of in.pdf 2 per sheet side (4 per sheet, back and front) onto out.pdf

   pdfcpu booklet 'formsize:Ledger' out.pdf 4 in.pdf
      Arrange pages of in.pdf 4 per sheet side (8 per sheet, back and front) onto out.pdf

   pdfcpu booklet 'formsize:Ledger' out.pdf 6 in.pdf
      Arrange pages of in.pdf 6 per sheet side (12 per sheet, back and front) onto out.pdf

   pdfcpu booklet 'formsize:A3' out.pdf 8 in.pdf
      Arrange pages of in.pdf 8 per sheet side (16 per sheet, back and front) onto out.pdf

   pdfcpu booklet 'formsize:A3, binding:short' out.pdf 4 in.pdf
      Arrange pages of in.pdf 4 per sheet side, with short-edge binding onto out.pdf

   pdfcpu booklet 'formsize:A4, multifolio:on' hardbackbook.pdf 2 in.pdf
      Arrange pages of in.pdf 2 per sheetside as sequence of folios covering 4*foliosize pages each.
      See also: https://www.instructables.com/How-to-bind-your-own-Hardback-Book/

   pdfcpu booklet 'formsize:A4, btype:perfectbound' out.pdf 2 in.pdf
      Arrange pages of in.pdf 2 per sheet side, arranged for perfect binding, onto out.pdf

   pdfcpu booklet 'formsize:A3, btype:bookletadvanced' out.pdf 4 in.pdf
      Arrange pages of in.pdf 4 per sheet side, arranged for advanced binding, onto out.pdf
`
	usageLongGrid = `Rearrange PDF pages or images for enhanced browsing experience.
For a PDF inputfile each output page represents a grid of input pages.
For image inputfiles each output page shows all images laid out onto grids of given paper size.
This command produces poster like PDF pages convenient for page and image browsing.

      pages ... Please refer to "pdfcpu selectedpages"
description ... dimensions, formsize, orientation, enforce
    outFile ... output PDF file, use - to write to stdout
          m ... grid lines
          n ... grid columns
     inFile ... input PDF file, use - to read from stdin
 imageFiles ... input image file(s)

    <description> is a comma separated configuration string containing:

    optional entries:

        (defaults: "d:595 842, form:A4, o:rd, bo:on, ma:3, enforce:on")

    dimensions:   (width height) in given display unit eg. '400 200'

    formsize:     The output sheet size, eg. A4, Letter, Legal...
                  Append 'L' to enforce landscape mode. (eg. A3L)
                  Append 'P' to enforce portrait mode. (eg. TabloidP)
                  Only one of dimensions or formsize is allowed.
                  Please refer to "pdfcpu paper" for a comprehensive list of defined paper sizes.
                  "papersize" is also accepted.

    orientation:  one of rd ... right down (=default)
                         dr ... down right
                         ld ... left down
                         dl ... down left
                  Orientation applies to PDF input files only.

    enforce:      enforce best-fit orientation of individual content (on/off, true/false, t/f).

    border:       Print border (on/off, true/false, t/f)

    margin:       Apply content margin (float >= 0 in given display unit)

All configuration string parameters support completion.

Examples: pdfcpu grid out.pdf 1 10 in.pdf
           Rearrange pages of in.pdf into 1x10 grids and write result to out.pdf using the default orientation.
           The output page size is the result of a 1(vert)x10(hor) page grid using in.pdf's page size.

          pdfcpu grid 'p:LegalL' out.pdf 2 2 in.pdf
           Rearrange pages of in.pdf into 2x2 grids and write result to out.pdf using the default orientation.
           The output page size is the result of a 2(vert)x2(hor) page grid using page size Legal in landscape mode.

          pdfcpu grid 'o:rd' out.pdf 3 2 in.pdf
           Rearrange pages of in.pdf into 3x2 grids and write result to out.pdf using orientation 'right down'.
           The output page size is the result of a 3(vert)x2(hor) page grid using in.pdf's page size.

          pdfcpu grid 'd:400 400' out.pdf 8 6 *.jpg
           Arrange imagefiles onto a 8x6 page grid and write result to out.pdf using a grid cell size of 400x400.

Pipeline example:
          aws s3 cp s3://acme-docs/manual.pdf - \
             | pdfcpu grid - 2 2 - \
             | aws s3 cp - s3://acme-docs/manual-grid.pdf
`

	usageLongResize = `Resize existing pages.

      pages ... please refer to "pdfcpu selectedpages"
description ... scalefactor, dimensions, formsize, enforce, rotate, border, bgcolor
     inFile ... input PDF file, use - to read from stdin
    outFile ... output PDF file, use - to write to stdout

    <description> is a comma separated configuration string containing:

      scalefactor:  Resize page by scale factor.
                        Use scale < 1 to shrink pages.
                        Use scale > 1 to enlarge pages.

      formsize:     Resize page to form/paper size eg. A4, Letter, Legal...
                        Append 'L' to enforce landscape mode. (eg. A3L)
                        Append 'P' to enforce portrait mode. (eg. A4P, TabloidP)
                        Please refer to "pdfcpu paper" for a comprehensive list of defined paper sizes.
                        "papersize" is also accepted.

      dimensions:   Resize page to custom dimensions.
                        (width height) in given display unit eg. "400 200"

      enforce:      preserve the requested output-page orientation (on/off, true/false, t/f).

      rotate:       allow additional best-fit content rotation (on/off, true/false, t/f; default: on).
                        rotate:off: content scales proportionally and remains centered.
                        Existing page rotation is still normalized.

      border:       if dimensions set only, draw content region border (on/off, true/false, t/f).

      bgcolor:      if dimensions set only, background color value for unused page regions.


   Examples:

         pdfcpu resize 'scale:2' in.pdf out.pdf
            Enlarge pages by doubling the page dimensions, keep orientation.

         pdfcpu resize 'sc:.5' in.pdf out.pdf --pages 1-3
            Shrink first 3 pages by cutting in half the page dimensions, keep orientation.

         pdfcpu resize 'dim:40 0' in.pdf out.pdf -u cm
            Resize pages to width of 40 cm, keep orientation.

         pdfcpu resize 'form:A4' in.pdf out.pdf
            Resize pages to A4, keep orientation.

         pdfcpu resize 'f:A4P, bgcol:#d0d0d0' in.pdf out.pdf
            Resize pages to A4 and enforce orientation(here: portrait mode), apply background color.

         pdfcpu resize 'form:A4P, rotate:off' in.pdf out.pdf
            Keep every page A4 portrait and fit landscape content without additional turn.

         pdfcpu resize 'dim:400 200' in.pdf out.pdf
            Resize pages to 400 x 200 points, keep orientation.

         pdfcpu resize 'dim:400 200, enforce:true' in.pdf out.pdf
            Resize pages to 400 x 200 points, enforce orientation.

Pipeline example:
   aws s3 cp s3://acme-design/spec-sheet.pdf - \
      | pdfcpu resize 'form:A4' - - \
      | aws s3 cp - s3://acme-design/spec-sheet-a4.pdf
`
	usageLongPoster = `Create a poster using paper size.

         pages ... Please refer to "pdfcpu selectedpages"
   description ... formsize(=papersize), dimensions, scalefactor, margin, bgcolor, border
        inFile ... input PDF file, use - to read from stdin
        outDir ... output directory
       outFile ... output file name

   Optionally scale up your page dimensions then define the poster grid tile size via form size or dimensions.

   <description> is a comma separated configuration string containing:

      scalefactor:  Enlarge page by scale factor > 1.

      formsize:     Posterize using tiles with form/paper size eg. A4, Letter, Legal...
                        Append 'L' to enforce landscape mode. (eg. A3L)
                        Append 'P' to enforce portrait mode. (eg. A4P, TabloidP)
                        Please refer to "pdfcpu paper" for a comprehensive list of defined paper sizes.
                        "papersize" is also accepted.

      dimensions:   Posterize using tiles with custom dimensions.
                        (width height) in given display unit eg. "400 200"

      margin:       Apply margin / glue area (float >= 0 in given display unit)

      bgcolor:      color value for visualization of margin / glue area.

      border:       if margin set, draw content region border (on/off, true/false, t/f)


   Examples:

         pdfcpu poster 'f:A4' in.pdf outDir
            Page form size is A2, the printer supports A4.
            Generate a poster(A2) via a corresponding 2x2 grid of A4 pages.

         pdfcpu poster 'f:A4, scale:2.0' in.pdf outDir
            Page form size is A2, the printer supports A4.
            Generate a poster(A0) via a corresponding 4x4 grid of A4 pages.

         pdfcpu poster 'dim:15 10, margin:1, bgcol:DarkGray, border:on' in.pdf outDir -u cm
            Generate a poster via a corresponding grid with cell size 15x10 cm and provide a glue area of 1 cm.

   Pipeline example:
         aws s3 cp s3://acme-print/poster.pdf - \
            | pdfcpu poster 'dim:100 100' - /work/poster-tiles

   See also the related commands: ndown, cut`
	usageLongNDown = `Cut selected page into n pages symmetrically.

         pages ... Please refer to "pdfcpu selectedpages"
   description ... margin, bgcolor, border
             n ... the n-Down value (see below for details)
        inFile ... input PDF file, use - to read from stdin
        outDir ... output directory
       outFile ... output file name

   <description> is a comma separated configuration string containing:

      margin:       Apply margin / glue area (float >= 0 in given display unit)

      bgcolor:      color value for visualization of margin / glue area.

      border:       if margin set, draw content region border (on/off, true/false, t/f)


                                  grid Eg.
   Supported values for n: 2 ...  1x2  A1 -> 2 x A2
                           3 ...  1x3
                           4 ...  2x2  A1 -> 4 x A3
                           8 ...  2x4  A1 -> 8 x A4
                           9 ...  3x3
                          12 ...  3x4
                          16 ...  4x4  A1 -> 16 x A5


   Examples:

         pdfcpu ndown 2 in.pdf outDir
            Page form size is A2, the printer supports A3.
            Quick cut page into 2 equally sized pages.

         pdfcpu ndown 4 in.pdf outDir
            Page form size is A2, the printer supports A4.
            Quick cut page into 4 equally (A4) sized pages.

         pdfcpu ndown 'margin:1, bgcol:DarkGray, border:on' 4 in.pdf outDir -u cm
            Page format size is A2, the printer supports A4.
            Quick cut page into 4 equally (A4) sized pages and provide a glue area of 1 cm.

   Pipeline example:
         aws s3 cp s3://acme-print/poster.pdf - \
            | pdfcpu ndown 4 - /work/tiles

   See also the related commands: poster, cut`

	usageLongCut = `Custom cut pages horizontally or vertically.

         pages ... Please refer to "pdfcpu selectedpages"
   description ... horizontal, vertical, margin, bgcolor, border
        inFile ... input PDF file, use - to read from stdin
        outDir ... output directory
       outFile ... output file name

   Fine grained custom page cutting.
   Apply any number of horizontal or vertical page cuts.

   <description> is a comma separated configuration string containing:

      horizontal:   Apply horizontal page cuts at height fraction (origin top left corner)
                    A sequence of fractions separated by white space.

      vertical:     Apply vertical page cuts at width fraction (origin top left corner)
                    A sequence of fractions separated by white space.

      margin:       Apply margin / glue area (float >= 0 in given display unit)

      bgcolor:      color value for visualization of margin / glue area.

      border:       if margin set, draw content region border (on/off, true/false, t/f)


   Examples:

         pdfcpu cut 'hor:.25' inFile outDir
            Apply a horizontal page cut at 0.25*height
            Results in 2 PDF pages.

         pdfcpu cut 'hor:.25, vert:.75' inFile outDir
            Apply a horizontal page cut at 0.25*height
            Apply a vertical page cut at 0.75*width

         pdfcpu cut 'hor:.33 .66' inFile outDir
            Has the same effect as: pdfcpu ndown 3 in.pdf outDir

         pdfcpu cut 'hor:.5, ver:.5' inFile outDir
            Has the same effect as: pdfcpu ndown 4 in.pdf outDir

   Pipeline example:
         aws s3 cp s3://acme-print/poster.pdf - \
            | pdfcpu cut 'hor:.5, ver:.5' - /work/cut-pages

   See also the related commands: poster, ndown`

	usageLongZoom = `Zoom in/out of selected pages either by magnification factor or corresponding margin.

      pages ... Please refer to "pdfcpu selectedpages"
description ... factor, hmargin, vmargin, border, bgcolor
     inFile ... input PDF file, use - to read from stdin
    outFile ... output PDF file, use - to write to stdout

Examples:
   pdfcpu zoom 'factor: '   in.pdf out.pdf            ... zoom in to magnification of 200%
   pdfcpu zoom 'factor: .5' in.pdf out.pdf            ... zoom out to magnification of 50%

   pdfcpu zoom 'hmargin: -10' in.pdf out.pdf          ... zoom in to horizontal margin of -10 points
   pdfcpu zoom 'hmargin:  10' in.pdf out.pdf          ... zoom out to horizontal margin of 10 points

   pdfcpu zoom 'hmargin: -1' in.pdf out.pdf --unit cm ... zoom in to horizontal margin of -1 cm
   pdfcpu zoom 'hmargin:  1' in.pdf out.pdf --unit cm ... zoom out to horizontal margin of 1 cm

   pdfcpu zoom 'vmargin: -10' in.pdf out.pdf          ... zoom in to vertical margin of -10 points
   pdfcpu zoom 'vmargin:  10' in.pdf out.pdf          ... zoom out to vertical margin of 10 points

   pdfcpu zoom 'vmargin: -1' in.pdf out.pdf --unit cm ... zoom in to vertical margin of -1 cm
   pdfcpu zoom 'vmargin: 1, border:true, bgcolor:lightgray' in.pdf out.pdf --unit cm ... zoom out to vertical margin of 1 cm

Pipeline example:
   aws s3 cp s3://acme-presentations/deck.pdf - \
      | pdfcpu zoom 'hmargin: 10' - - \
      | aws s3 cp - s3://acme-presentations/deck-with-margin.pdf
`

	usagePageSelection = `'-p' or '--pages' selects pages for processing and is a comma separated list of expressions:

	Valid expressions are:

   even ... include even pages           odd ... include odd pages
      # ... include page #               #-# ... include page range
     !# ... exclude page #              !#-# ... exclude page range
     n# ... exclude page #              n#-# ... exclude page range

     #- ... include page # - last page    -# ... include first page - page #
    !#- ... exclude page # - last page   !-# ... exclude first page - page #
    n#- ... exclude page # - last page   n-# ... exclude first page - page #

   l-3- ... include last 3 pages         l-3 ... include page # last-3
  -l-3  ... include all, but last 3    2-l-1 ... pages 2 up to "last-1"

	n serves as an alternative for !, since ! needs to be escaped with single quotes on the cmd line.

        e.g. -3,5,7- or 4-7,!6 or 1-,!5 or odd,n1`

	usageLongPages = `Manage pages.

      pages ... Please refer to "pdfcpu selectedpages"
       mode ... before, after (default: before)
description ... dimensions, formsize
     inFile ... input PDF file, use - to read from stdin
    outFile ... output PDF file, use - to write to stdout

<description> is a comma separated configuration string containing:

  optional entries:

      (defaults: "dim:595 842, f:A4")

  dimensions:      (width height) in given display unit eg. '400 200' setting the media box

  formsize:        eg. A4, Letter, Legal...
                   Append 'L' to enforce landscape mode. (eg. A3L)
                   Append 'P' to enforce portrait mode. (eg. TabloidP)
                   Please refer to "pdfcpu paper" for a comprehensive list of defined paper sizes.
                   "papersize" is also accepted.

   All configuration string parameters support completion.

   Examples:      pdfcpu pages insert in.pdf
                   Insert one blank page before each page using the form size imposed internally by the current media box.

                  pdfcpu pages insert 'f:A5L' in.pdf --pages 3
                   Insert one blank A5 page in landscape mode before page 3.

                  pdfcpu pages insert in.pdf 'dim: 10 5' --unit cm
                   Insert one blank 10 x 5 cm separator page for all pages.

                  pdfcpu pages remove in.pdf out.pdf -p odd
                  pdfcpu pages remove in.pdf out.pdf --pages odd
                   Remove all odd pages.

Pipeline examples:
   aws s3 cp s3://acme-forms/packet.pdf - \
      | pdfcpu pages insert -p 1 - - \
      | aws s3 cp - s3://acme-forms/packet-with-cover.pdf

   aws s3 cp s3://acme-forms/packet.pdf - \
      | pdfcpu pages remove -p 2 - - \
      | aws s3 cp - s3://acme-forms/packet-without-page-2.pdf
`

	usageLongRotate = `Rotate selected pages by a multiple of 90 degrees.

      pages ... Please refer to "pdfcpu selectedpages"
     inFile ... input PDF file, use - to read from stdin
   rotation ... a multiple of 90 degrees for clockwise rotation
    outFile ... output PDF file, use - to write to stdout

Pipeline example:
   aws s3 cp s3://acme-scans/batch.pdf - \
      | pdfcpu rotate -p odd - 90 - \
      | aws s3 cp - s3://acme-scans/batch-rotated.pdf
`

	usageLongCrop = `Set crop box for selected pages.

        pages ... Please refer to "pdfcpu selectedpages"
  description ... crop box definition abs. or rel. to media box
       inFile ... input PDF file, use - to read from stdin
      outFile ... output PDF file, use - to write to stdout

Examples:
   pdfcpu crop '[0 0 500 500]' in.pdf ... crop a 500x500 points region located in lower left corner
   pdfcpu crop '20' in.pdf -u mm      ... crop relative to media box using a 20mm margin

Pipeline example:
   aws s3 cp s3://acme-print/catalog.pdf - \
      | pdfcpu crop '10' - - -u mm \
      | aws s3 cp - s3://acme-print/catalog-cropped.pdf
` + usageBoxDescription

	usageLongBoxes = `Manage page boundaries.

     boxTypes ... comma separated list of box types: m(edia), c(rop), t(rim), b(leed), a(rt)
        pages ... Please refer to "pdfcpu selectedpages"
  description ... box definitions abs. or rel. to parent box
       inFile ... input PDF file, use - to read from stdin
      outFile ... output PDF file, use - to write to stdout

<description> is a sequence of box definitions and assignments:

   m(edia): {box}
    c(rop): {box}
     a(rt): {box} | m(edia) | c(rop) | b(leed) | t(rim)
   b(leed): {box} | m(edia) | c(rop) | a(rt) | t(rim)
    t(rim): {box} | m(edia) | c(rop) | a(rt) | b(leed)

Examples:
   pdfcpu boxes list in.pdf
   pdfcpu boxes list 'bleed,trim' in.pdf
   pdfcpu boxes add 'crop:[10 10 200 200], trim:5, bleed:trim' in.pdf
   pdfcpu boxes remove 't,b' in.pdf

Pipeline examples:
   aws s3 cp s3://acme-print/ad.pdf - \
      | pdfcpu boxes list -

   aws s3 cp s3://acme-print/ad.pdf - \
      | pdfcpu boxes add 'trim:5, bleed:10' - - \
      | aws s3 cp - s3://acme-print/ad-boxes.pdf

   aws s3 cp s3://acme-print/ad-boxes.pdf - \
      | pdfcpu boxes remove 'trim,bleed' - - \
      | aws s3 cp - s3://acme-print/ad-clean-boxes.pdf
` + usageBoxDescription
)
