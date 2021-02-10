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

package main

const (
	usage = `pdfcpu is a tool for PDF manipulation written in Go. 
	
Usage:
	
   pdfcpu command [arguments]
   
The commands are:

   attachments list, add, remove, extract embedded file attachments
   booklet     arrange pages onto larger sheets of paper to make a booklet or zine
   boxes       list, add, remove page boundaries for selected pages
   changeopw   change owner password
   changeupw   change user password
   collect     create custom sequence of selected pages
   crop        set cropbox for selected pages
   decrypt     remove password protection
   encrypt     set password protection		
   extract     extract images, fonts, content, pages or metadata
   fonts       install, list supported fonts, create cheat sheets
   grid        rearrange pages or images for enhanced browsing experience
   import      import/convert images to PDF
   info        print file info
   keywords    list, add, remove keywords
   merge       concatenate PDFs
   nup         rearrange pages or images for reduced number of pages
   optimize    optimize PDF by getting rid of redundant page resources
   pages       insert, remove selected pages
   paper       print list of supported paper sizes
   permissions list, set user access permissions
   portfolio   list, add, remove, extract portfolio entries with optional description
   properties  list, add, remove document properties
   rotate      rotate pages
   split       split up a PDF by span or bookmark
   stamp       add, remove, update Unicode text, image or PDF stamps for selected pages
   trim        create trimmed version of selected pages
   validate    validate PDF against PDF 32000-1:2008 (PDF 1.7)
   version     print version
   watermark   add, remove, update Unicode text, image or PDF watermarks for selected pages

   All instantly recognizable command prefixes are supported eg. val for validation
   One letter Unix style abbreviations supported for flags and command parameters.

Use "pdfcpu help [command]" for more information about a command.`

	generalFlags = `
   
common flags: -v(erbose)  ... turn on logging
              -vv         ... verbose logging
              -q(uiet)    ... disable output
              -c(onf)     ... set or disable config dir: $path|disable
              -opw        ... owner password
              -upw        ... user password
              -u(nit)     ... display unit: po(ints) ... points
                                            in(ches) ... inches
                                                  cm ... centimetres
                                                  mm ... millimetres`

	usageValidate = "usage: pdfcpu validate [-m(ode) strict|relaxed] inFile" + generalFlags

	usageLongValidate = `Check inFile for specification compliance.

      mode ... validation mode
    inFile ... input pdf file
		
The validation modes are:

 strict ... (default) validates against PDF 32000-1:2008 (PDF 1.7)
relaxed ... like strict but doesn't complain about common seen spec violations.`

	usageOptimize     = "usage: pdfcpu optimize [-stats csvFile] inFile [outFile]" + generalFlags
	usageLongOptimize = `Read inFile, remove redundant page resources like embedded fonts and images and write the result to outFile.

     stats ... appends a stats line to a csv file with information about the usage of root and page entries.
               useful for batch optimization and debugging PDFs.
    inFile ... input pdf file
   outFile ... output pdf file`

	usageSplit     = "usage: pdfcpu split [-m(ode) span|bookmark] inFile outDir [span]" + generalFlags
	usageLongSplit = `Generate a set of PDFs for the input file in outDir according to given span value or along bookmarks.

      mode ... split mode (defaults to span)
    inFile ... input pdf file
    outDir ... output directory
      span ... split span in pages (default: 1) for mode "span"
      
The split modes are:

      span     ... Split into PDF files with span pages each (default).
                   span itself defaults to 1 resulting in single page PDF files.
  
      bookmark ... Split into PDF files representing sections defined by existing bookmarks.
                   span will be ignored.
                   Assumption: inFile contains an outline dictionary.`

	usageMerge     = "usage: pdfcpu merge [-m(ode) create|append] [-sort] outFile inFile..." + generalFlags
	usageLongMerge = `Concatenate a sequence of PDFs/inFiles into outFile.

      mode ... merge mode (defaults to create)
      sort ... sort inFiles by file name
   outFile ... output pdf file
    inFile ... a list of pdf files subject to concatenation.
    
The merge modes are:

    create ... outFile will be created and possibly overwritten (default).

    append ... if outFile does not exist, it will be created (like in default mode).
               if outFile already exists, inFiles will be appended to outFile.`

	usagePageSelection = `'-pages' selects pages for processing and is a comma separated list of expressions:

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

	usageExtract     = "usage: pdfcpu extract -m(ode) image|font|content|page|meta [-p(ages) selectedPages] inFile outDir" + generalFlags
	usageLongExtract = `Export inFile's images, fonts, content or pages into outDir.

      mode ... extraction mode
     pages ... selected pages
    inFile ... input pdf file
    outDir ... output directory

 The extraction modes are:

  image ... extract images
   font ... extract font files (supported font types: TrueType)
content ... extract raw page content
   page ... extract single page PDFs
   meta ... extract all metadata (page selection does not apply)
   
` + usagePageSelection

	usageTrim     = "usage: pdfcpu trim -p(ages) selectedPages inFile [outFile]" + generalFlags
	usageLongTrim = `Generate a trimmed version of inFile for selected pages.

     pages ... selected pages
    inFile ... input pdf file
   outFile ... output pdf file
   
` + usagePageSelection

	usageAttachList    = "pdfcpu attachments list    inFile"
	usageAttachAdd     = "pdfcpu attachments add     inFile file..."
	usageAttachRemove  = "pdfcpu attachments remove  inFile [file...]"
	usageAttachExtract = "pdfcpu attachments extract inFile outDir [file...]" + generalFlags

	usageAttach = "usage: " + usageAttachList +
		"\n       " + usageAttachAdd +
		"\n       " + usageAttachRemove +
		"\n       " + usageAttachExtract

	usageLongAttach = `Manage embedded file attachments.

    inFile ... input pdf file
      file ... attachment
    outDir ... output directory`

	usagePortfolioList    = "pdfcpu portfolio list    inFile"
	usagePortfolioAdd     = "pdfcpu portfolio add     inFile file[,desc]..."
	usagePortfolioRemove  = "pdfcpu portfolio remove  inFile [file...]"
	usagePortfolioExtract = "pdfcpu portfolio extract inFile outDir [file...]" + generalFlags

	usagePortfolio = "usage: " + usagePortfolioList +
		"\n       " + usagePortfolioAdd +
		"\n       " + usagePortfolioRemove +
		"\n       " + usagePortfolioExtract

	usageLongPortfolio = `Manage portfolio entries.

    inFile ... input pdf file
      file ... attachment
      desc ... description (optional)
    outDir ... output directory
    
    Adding attachments to portfolio: 
           pdfcpu portfolio add test.pdf test.mp3 test.mkv

    Adding attachments to portfolio with description: 
           pdfcpu portfolio add test.pdf 'test.mp3, Test sound file' 'test.mkv, Test video file'
    `

	usagePermList = "pdfcpu permissions list [-upw userpw] [-opw ownerpw] inFile"
	usagePermSet  = "pdfcpu permissions set [-perm none|all] [-upw userpw] -opw ownerpw inFile" + generalFlags

	usagePerm = "usage: " + usagePermList +
		"\n       " + usagePermSet

	usageLongPerm = `Manage user access permissions.

      perm ... user access permissions
    inFile ... input pdf file`

	usageEncrypt     = "usage: pdfcpu encrypt [-m(ode) rc4|aes] [-key 40|128|256] [-perm none|all] [-upw userpw] -opw ownerpw inFile [outFile]" + generalFlags
	usageLongEncrypt = `Setup password protection based on user and owner password.

      mode ... algorithm (default=aes)
       key ... key length in bits (default=256)
      perm ... user access permissions
    inFile ... input pdf file
   outFile ... output pdf file`

	usageDecrypt     = "usage: pdfcpu decrypt [-upw userpw] [-opw ownerpw] inFile [outFile]" + generalFlags
	usageLongDecrypt = `Remove password protection and reset permissions.

    inFile ... input pdf file
   outFile ... output pdf file`

	usageChangeUserPW     = "usage: pdfcpu changeupw [-opw ownerpw] inFile upwOld upwNew" + generalFlags
	usageLongChangeUserPW = `Change the user password also known as the open doc password.

       opw ... owner password, required unless = ""
    inFile ... input pdf file
    upwOld ... old user password
    upwNew ... new user password`

	usageChangeOwnerPW     = "usage: pdfcpu changeopw [-upw userpw] inFile opwOld opwNew" + generalFlags
	usageLongChangeOwnerPW = `Change the owner password also known as the set permissions password.

       upw ... user password, required unless = ""
    inFile ... input pdf file
    opwOld ... old owner password (provide user password on initial changeopw)
    opwNew ... new owner password`

	usageWMMode = `There are 3 different kinds:

   1) text based:
      -mode text string			
         eg. mode -text "Hello gopher!"
   
   2) image based
      -mode image imageFileName
         supported extensions: '.jpg', 'jpeg', .png', '.tif', '.tiff' 
         eg. mode -image logo.png
         
   3) PDF based
      -mode pdf pdfFileName[:page#]
         eg. pdfcpu stamp add mode -pdf 'stamp.pdf:3' '' in.pdf out.pdf ... stamp each page of in.pdf with page 3 of stamp.pdf
         Omit page# for multistamping:
         eg. pdfcpu stamp add mode -pdf 'stamp.pdf' '' in.pdf out.pdf   ... stamp each page of in.pdf with corresponding page of stamp.pdf
   `
	usageWMDescription = `

<description> is a comma separated configuration string containing these optional entries:
	
   (defaults: 'font:Helvetica, points:24, pos:c, off:0,0 s:0.5 rel, rot:0, d:1, op:1, m:0 and for all colors: 0.5 0.5 0.5')

   fontname:         Please refer to "pdfcpu fonts list"
   points:           fontsize in points, in combination with absolute scaling only.
   position:         one of the anchors: tl,tc,tr, l,c,r, bl,bc,br
                     Reliable with non rotated pages only!
   offset:           (dx dy) in given display unit eg. '15 20'
   scalefactor:      0.0 < i <= 1.0 {r|rel} | 0.0 < i {a|abs}
   aligntext:        l..left, c..center, r..right, j..justified (for text watermarks only)
   fillcolor:        color value to be used when rendering text, see also rendermode
                     for backwards compatibility "color" is also accepted.
   strokecolor:      color value to be used when rendering text, see also rendermode
   backgroundcolor:  color value for visualization of the bounding box background for text.
                     "bgcolor" is also accepted. 
   rotation:         -180.0 <= x <= 180.0
   diagonal:         render along diagonal, 1..lower left to upper right, 2..upper left to lower right (if present overrules r!)
                     Only one of rotation and diagonal is allowed!
   opacity:          where 0.0 <= x <= 1.0

   mode, rendermode: 0 ... fill (applies fill color)
                     1 ... stroke (applies stroke color)
                     2 ... fill & stroke (applies both fill and stroke colors)

   margins:          Set bounding box margins for text (requires background color) i >= 0
                     i       ... set all four margins
                     i j     ... set top/bottom margins to i
                                 set left/right margins to j
                     i j k   ... set top margin to i
                                 set left/right margins to j
                                 set bottom margins to k
                     i j k l ... set top, right, bottom, left margins

   border:           Set bounding box border for text (requires background color)
                     i {color} {round}
                     i     ... border width > 0
                     color ... border color
                     round ... set round bounding box corners

A color value: 3 color intensities, where 0.0 < i < 1.0, eg 1.0, 
               or the hex RGB value: #RRGGBB, eg #FF0000 = red

All configuration string parameters support completion.

e.g. 'pos:bl, off: 20 5'   'rot:45'                 'op:0.5, s:0.5 abs, rot:0'
     'd:2'                 's:.75 abs, points:48'   'rot:-90, scale:0.75 rel'
     'f:Courier, s:0.75, c: 0.5 0.0 0.0, rot:20'  
  
     
` + usagePageSelection

	usageStampAdd    = "pdfcpu stamp add    [-p(ages) selectedPages] -m(ode) text|image|pdf string|file description inFile [outFile]"
	usageStampUpdate = "pdfcpu stamp update [-p(ages) selectedPages] -m(ode) text|image|pdf string|file description inFile [outFile]"
	usageStampRemove = "pdfcpu stamp remove [-p(ages) selectedPages] inFile [outFile]" + generalFlags

	usageStamp = "usage: " + usageStampAdd +
		"\n       " + usageStampUpdate +
		"\n       " + usageStampRemove

	usageLongStamp = `Process stamping for selected pages. 

      pages ... selected pages
        upw ... user password
        opw ... owner password
       mode ... text, image, pdf
     string ... display string for text based watermarks
       file ... image or pdf file
description ... fontname, points, position, offset, scalefactor, aligntext, rotation, 
                diagonal, opacity, rendermode, strokecolor, fillcolor, bgcolor, margins, border
     inFile ... input pdf file
    outFile ... output pdf file

` + usageWMMode + usageWMDescription

	usageWatermarkAdd    = "pdfcpu watermark add    [-p(ages) selectedPages] -m(ode) text|image|pdf string|file description inFile [outFile]"
	usageWatermarkUpdate = "pdfcpu watermark update [-p(ages) selectedPages] -m(ode) text|image|pdf string|file description inFile [outFile]"
	usageWatermarkRemove = "pdfcpu watermark remove [-p(ages) selectedPages] inFile [outFile]" + generalFlags

	usageWatermark = "usage: " + usageWatermarkAdd +
		"\n       " + usageWatermarkUpdate +
		"\n       " + usageWatermarkRemove

	usageLongWatermark = `Process watermarking for selected pages. 

      pages ... selected pages
       mode ... text, image, pdf
     string ... display string for text based watermarks
       file ... image or pdf file
description ... fontname, points, position, offset, scalefactor, aligntext, rotation,
                diagonal, opacity, rendermode, strokecolor, fillcolor, bgcolor, margins, border
     inFile ... input pdf file
    outFile ... output pdf file

` + usageWMMode + usageWMDescription

	usageImportImages     = "usage: pdfcpu import [description] outFile imageFile..." + generalFlags
	usageLongImportImages = `Turn image files into a PDF page sequence and write the result to outFile.
If outFile already exists the page sequence will be appended.
Each imageFile will be rendered to a separate page.
In its simplest form this converts an image into a PDF: "pdfcpu import img.pdf img.jpg"

description ... dimensions, format, position, offset, scale factor, boxes
    outFile ... output pdf file
  imageFile ... a list of image files
  
  <description> is a comma separated configuration string containing:

  optional entries:

      (defaults: d:595 842, f:A4, pos:full, off:0 0, s:0.5 rel, dpi:72)

  dimensions: (width height) in given display unit eg. '400 200' setting the media box

  formsize, papersize: eg. A4, Letter, Legal...
                           Please refer to "pdfcpu help paper" for a comprehensive list of defined paper sizes.
                           Append 'L' to enforce landscape mode. (eg. A3L)
                           Append 'P' to enforce portrait mode. (eg. TabloidP)

  position:    one of 'full' or the anchors: tl,tc,tr, l,c,r, bl,bc,br
  offset:      (dx dy) in given display unit eg. '15 20'
  scalefactor: 0.0 <= x <= 1.0 followed by optional 'abs|rel' or 'a|r'
  dpi:         apply desired dpi
  
  Only one of dimensions or format is allowed.
  position: full => image dimensions equal page dimensions.
  
  All configuration string parameters support completion.

  e.g. 'f:A5, pos:c                              ... render the image centered on A5 with relative scaling 0.5.'
       'd:300 600, pos:bl, off:20 20, s:1.0 abs' ... render the image anchored to bottom left corner with offset 20,20 and abs. scaling 1.0.
       'pos:full'                                ... render the image to a page with corresponding dimensions.
       'f:A4, pos:c, dpi:300'                    ... render the image centered on A4 respecting a destination resolution of 300 dpi.
       `

	usagePagesInsert = "pdfcpu pages insert [-p(ages) selectedPages] [-m(ode) before|after] inFile [outFile]"
	usagePagesRemove = "pdfcpu pages remove  -p(ages) selectedPages  inFile [outFile]" + generalFlags

	usagePages = "usage: " + usagePagesInsert +
		"\n       " + usagePagesRemove

	usageLongPages = `Manage pages.

      pages ... selected pages
       mode ... before, after (default: before)
     inFile ... input pdf file
    outFile ... output pdf file

` + usagePageSelection

	usageRotate     = "usage: pdfcpu rotate [-p(ages) selectedPages] inFile rotation [outFile]" + generalFlags
	usageLongRotate = `Rotate selected pages by a multiple of 90 degrees. 

      pages ... selected pages
     inFile ... input pdf file
   rotation ... a multiple of 90 degrees for clockwise rotation
    outFile ... output pdf file

` + usagePageSelection

	usageNUp     = "usage: pdfcpu nup [-p(ages) selectedPages] [description] outFile n inFile|imageFiles..." + generalFlags
	usageLongNUp = `Rearrange existing PDF pages or images into a sequence of page grids.
This reduces the number of pages and therefore the required print time.
If the input is one imageFile a single page n-up PDF gets generated.

      pages ... selected pages for inFile only
description ... dimensions, format, orientation
    outFile ... output pdf file
          n ... the n-Up value (see below for details)
     inFile ... input pdf file
 imageFiles ... input image file(s)

                             portrait landscape
 Possible values for n: 2 ...  1x2       2x1
                        3 ...  1x3       3x1
                        4 ...  2x2
                        8 ...  2x4       4x2
                        9 ...  3x3
                       12 ...  3x4       4x3
                       16 ...  4x4

    <description> is a comma separated configuration string containing:

    optional entries:
  
        (defaults: di:595 842, fo:A4, or:rd, bo:on, ma:3)
  
    dimensions:      (width,height) in given display unit eg. '400 200'
    formsize:        The output sheet size, eg. A4, Letter, Legal...
                     Append 'L' to enforce landscape mode. (eg. A3L)
                     Append 'P' to enforce portrait mode. (eg. TabloidP)
                     Only one of dimensions or format is allowed.
                     Please refer to "pdfcpu help paper" for a comprehensive list of defined paper sizes.
                     "papersize" is also accepted.
    orientation:     one of rd ... right down (=default)
                            dr ... down right
                            ld ... left down
                            dl ... down left
                     Orientation applies to PDF input files only.
    border:          on/off true/false
    margin:          for n-up content: float >= 0 in given display unit
    backgroundcolor: backgound color for margin > 0.
                     "bgcolor" is also accepted.

All configuration string parameters support completion.
    
Examples: "pdfcpu nup out.pdf 4 in.pdf"
          Rearrange pages of in.pdf into 2x2 grids and write result to out.pdf using the default orientation
          and default paper size A4. in.pdf's page size will be preserved.
                                 
          "pdfcpu nup -pages=3- out.pdf 6 in.pdf"
          Rearrange selected pages of in.pdf (all pages starting with page 3) into 3x2 grids and
          write result to out.pdf using the default orientation and default paper size A4.
          in.pdf's page size will be preserved.

          "pdfcpu nup out.pdf 9 logo.jpg"
          Arrange instances of logo.jpg into a 3x3 grid and write result to out.pdf using the A4 default format.
          
          "pdfcpu nup 'f:Tabloid' out.pdf 4 *.jpg" 
          Rearrange all jpg files into 2x2 grids and write result to out.pdf using the Tabloid format
          and the default orientation.

` + usagePageSelection

	usageBooklet     = "usage: pdfcpu booklet [-p(ages) selectedPages] [description] outFile n inFile|imageFiles..." + generalFlags
	usageLongBooklet = `Arrange a sequence of pages onto larger sheets of paper for a small book or zine.

              pages       ... selected pages for inFile only
              description ... dimensions, formsize, border, margin
              outFile     ... output pdf file
              n           ... the n-Up value: 2 or 4
              inFile      ... input pdf file
              imageFiles  ... input image file(s)

There are two styles of booklet, depending on your page/input and sheet/output size:

n=2: Two of your pages fit on one side of a sheet (eg statement on letter, A5 on A4)
Assemble by printing on both sides (odd pages on the front and even pages on the back) and folding down the middle.

n=4: Four of your pages fit on one side of a sheet (eg statement on ledger, A5 on A3, A6 on A4)
Assemble by printing on both sides, then cutting the sheets horizontally.
The sets of pages on the bottom of the sheet are rotated so that the cut side of the
paper is on the bottom of the booklet for every page. After cutting, place the bottom
set of pages after the top set of pages in the booklet. Then fold the half sheets.

                             portrait landscape
 Possible values for n: 2 ...  1x2       2x1
                        4 ...  2x2       2x2

<description> is a comma separated configuration string containing these optional entries:

   (defaults: 'dim:595 842, formsize:A4, border:off, guides:off, margin:0')

   dimensions:       (width,height) of the output sheet in given display unit eg. '400 200'
   formsize:         The output sheet size, eg. A4, Letter, Legal...
                     Append 'L' to enforce landscape mode. (eg. A3L)
                     Append 'P' to enforce portrait mode. (eg. TabloidP)
                     Only one of dimensions or format is allowed.
                     Please refer to "pdfcpu help paper" for a comprehensive list of defined paper sizes.
                     "papersize" is also accepted.
   border:           on/off true/false
   guides:           on/off true/false prints folding and cutting lines
   margin:           for n-up content: float >= 0 in given display unit
   backgroundcolor:  sheet backgound color for margin > 0.
                     "bgcolor" is also accepted.

All configuration string parameters support completion.

Examples: "pdfcpu booklet 'formsize:Letter' out.pdf 2 in.pdf"
           Arrange pages of in.pdf 2 per sheet side (4 per sheet, back and front) onto out.pdf

          "pdfcpu booklet 'formsize:Ledger' out.pdf 4 in.pdf"
           Arrange pages of in.pdf 4 per sheet side (8 per sheet, back and front) onto out.pdf

           "pdfcpu booklet 'formsize:A4' out.pdf 2 in.pdf"
           Arrange pages of in.pdf 2 per sheet side (4 per sheet, back and front) onto out.pdf
`

	usageGrid     = "usage: pdfcpu grid [-p(ages) selectedPages] [description] outFile m n inFile|imageFiles..." + generalFlags
	usageLongGrid = `Rearrange PDF pages or images for enhanced browsing experience.
For a PDF inputfile each output page represents a grid of input pages.
For image inputfiles each output page shows all images laid out onto grids of given paper size. 
This command produces poster like PDF pages convenient for page and image browsing. 

      pages ... selected pages for inFile only
description ... dimensions, format, orientation
    outFile ... output pdf file
          m ... grid columns
          n ... grid lines
     inFile ... input pdf file
 imageFiles ... input image file(s)

    <description> is a comma separated configuration string containing:

    optional entries:
  
        (defaults: d:595 842, f:A4, o:rd, bo:on, m:3)
  
    dimensions:   (width height) in given display unit eg. '400 200'
    formsize:     The output sheet size, eg. A4, Letter, Legal...
                  Append 'L' to enforce landscape mode. (eg. A3L)
                  Append 'P' to enforce portrait mode. (eg. TabloidP)
                  Only one of dimensions or format is allowed.
                  Please refer to "pdfcpu help paper" for a comprehensive list of defined paper sizes.
                  "papersize" is also accepted.
    orientation:  one of rd ... right down (=default)
                         dr ... down right
                         ld ... left down
                         dl ... down left
                  Orientation applies to PDF input files only.
    border:       on/off true/false
    margin:       for content: float >= 0 in given display unit

All configuration string parameters support completion.

Examples: "pdfcpu grid out.pdf 1 10 in.pdf"
          Rearrange pages of in.pdf into 1x10 grids and write result to out.pdf using the default orientation.
          The output page size is the result of a 1(hor)x10(vert) page grid using in.pdf's page size.

          "pdfcpu grid 'LegalL' out.pdf 2 2 in.pdf" 
          Rearrange pages of in.pdf into 2x2 grids and write result to out.pdf using the default orientation.
          The output page size is the result of a 2(hor)x2(vert) page grid using page size Legal in landscape mode.

          "pdfcpu grid 'o:rd' out.pdf 3 2 in.pdf" 
          Rearrange pages of in.pdf into 3x2 grids and write result to out.pdf using orientation 'right down'.
          The output page size is the result of a 3(hor)x2(vert) page grid using in.pdf's page size.

          "pdfcpu grid 'd:400 400' out.pdf 6 8 *.jpg"
          Arrange imagefiles onto a 6x8 page grid and write result to out.pdf using a grid cell size of 400x400.

` + usagePageSelection

	paperSizes = `This is a list of predefined paper sizes:
   
   ISO 216:1975 A:
      4A0, 2A0, A0, A1, A2, A3, A4, A5, A6, A7, A8, A9, A10
   
   ISO 216:1975 B:
      B0+, B0, B1+, B1, B2+, B2, B3, B4, B5, B6, B7, B8, B9, B10
   
   ISO 269:1985 C:
      C0, C1, C2, C3, C4, C5, C6, C7, C8, C9, C10 
   
   ISO 217:2013 untrimmed:
      RA0, RA1, RA2, RA3, RA4, SRA0, SRA1, SRA2, SRA3, SRA4, SRA1+, SRA2+, SRA3+, SRA3++
   
   American:
      SuperB(=B+),
      Tabloid (=ANSIB, DobleCarta), Ledger(=ANSIB, DobleCarta),
      Legal, GovLegal(=Oficio, Folio),
      Letter (=ANSIA, Carta, AmericanQuarto), GovLetter, Executive,
      HalfLetter (=Memo, Statement, Stationary),
      JuniorLegal (=IndexCard),
      Photo
   
   ANSI/ASME Y14.1:
      ANSIA (=Letter, Carta, AmericanQuarto),
      ANSIB (=Ledger, Tabloid, DobleCarta),
      ANSIC, ANSID, ANSIE, ANSIF
   
   ANSI/ASME Y14.1 Architectural series:
      ARCHA (=ARCH1),
      ARCHB (=ARCH2, ExtraTabloide),
      ARCHC (=ARCH3),
      ARCHD (=ARCH4),
      ARCHE (=ARCH6),
      ARCHE1 (=ARCH5),
      ARCHE2,
      ARCHE3
   
   American uncut:
      Bond, Book, Cover, Index, NewsPrint (=Tissue), Offset (=Text)
   
   English uncut:
      Crown, DoubleCrown, Quad, Demy, DoubleDemy, Medium, Royal, SuperRoyal,
      DoublePott, DoublePost, Foolscap, DoubleFoolscap   
   
   F4

   China GB/T 148-1997 D Series:
      D0, D1, D2, D3, D4, D5, D6,
      RD0, RD1, RD2, RD3, RD4, RD5, RD6

   Japan:
   
   B-series variant:
      JIS-B0, JIS-B1, JIS-B2, JIS-B3, JIS-B4, JIS-B5, JIS-B6,
      JIS-B7, JIS-B8, JIS-B9, JIS-B10, JIS-B11, JIS-B12
   
   Shirokuban4, Shirokuban5, Shirokuban6
   Kiku4, Kiku5
   AB, B40, Shikisen`

	usageVersion     = "usage: pdfcpu version [-v(erbose)|vv]"
	usageLongVersion = "Print the pdfcpu version."

	usagePaper     = "usage: pdfcpu paper"
	usageLongPaper = "Print a list of supported paper sizes."

	usageInfo     = "usage: pdfcpu info [-p(ages) selectedPages] inFile" + generalFlags
	usageLongInfo = `Print info about a PDF file.
   
   pages ... selected pages
  inFile ... input pdf file`

	usageFontsList       = "pdfcpu fonts list"
	usageFontsInstall    = "pdfcpu fonts install fontFiles..."
	usageFontsCheatSheet = "pdfcpu fonts cheatsheet fontFiles..."

	usageFonts = "usage: " + usageFontsList +
		"\n       " + usageFontsInstall +
		"\n       " + usageFontsCheatSheet
	usageLongFonts = `Print a list of supported fonts (includes the 14 PDF core fonts).
Install given True Type fonts(.ttf) or True Type collections(.ttc) for usage in stamps/watermarks.
Create single page PDF cheat sheets in current dir.`

	usageKeywordsList   = "pdfcpu keywords list    inFile"
	usageKeywordsAdd    = "pdfcpu keywords add     inFile keyword..."
	usageKeywordsRemove = "pdfcpu keywords remove  inFile [keyword...]" + generalFlags

	usageKeywords = "usage: " + usageKeywordsList +
		"\n       " + usageKeywordsAdd +
		"\n       " + usageKeywordsRemove

	usageLongKeywords = `Manage keywords.

    inFile ... input pdf file
   keyword ... search keyword
    
    Eg. adding two keywords: 
           pdfcpu keywords add test.pdf music 'virtual instruments'
    `

	usagePropertiesList   = "pdfcpu properties list    inFile"
	usagePropertiesAdd    = "pdfcpu properties add     inFile nameValuePair..."
	usagePropertiesRemove = "pdfcpu properties remove  inFile [name...]" + generalFlags

	usageProperties = "usage: " + usagePropertiesList +
		"\n       " + usagePropertiesAdd +
		"\n       " + usagePropertiesRemove

	usageLongProperties = `Manage document properties.

       inFile ... input pdf file
nameValuePair ... 'name = value'
         name ... property name
     
     Eg. adding one property:   pdfcpu properties add test.pdf 'key = value'
         adding two properties: pdfcpu properties add test.pdf 'key1 = val1' 'key2 = val2'
     `
	usageCollect     = "usage: pdfcpu collect -pages selectedPages inFile [outFile]" + generalFlags
	usageLongCollect = `Create custom sequence of selected pages. 

        pages ... selected pages
       inFile ... input pdf file
      outFile ... output pdf file
  
  ` + usagePageSelection

	usageBoxDescription = `
box:

   A rectangular region in user space describing one of:

      media box:  boundaries of the physical medium on which the page is to be printed.
       crop box:  region to which the contents of the page shall be clipped (cropped) when displayed or printed.
      bleed box:  region to which the contents of the page shall be clipped when output in a production environment.
       trim box:  intended dimensions of the finished page after trimming.
        art box:  extent of the page’s meaningful content as intended by the page’s creator.
   
   Please refer to the PDF Specification 14.11.2 Page Boundaries for details.

   All values are in given display unit (po, in, mm, cm)

   General rules:
      The media box is mandatory and serves as default for the crop box and is its parent box.
      The crop box serves as default for art box, bleed box and trim box and is their parent box.

   Arbitrary rectangular region in user space:
      [0 10 200 150]       lower left corner at (0/10), upper right corner at (200/150)
                           or xmin:0 ymin:10 xmax:200 ymax:150

   Expressed as margins within parent box:
      '0.5 0.5 20 20'      absolute, top:.5 right:.5 bottom:20 left:20
      '0.5 0.5 .1 .1 abs'  absolute, top:.5 right:.5 bottom:.1 left:.1
      '0.5 0.5 .1 .1 rel'  relative, top:.5 right:.5 bottom:20 left:20
      '10'                 absolute, top,right,bottom,left:10
      '10 5'               absolute, top,bottom:10  left,right:5
      '10 5 15'            absolute, top:10 left,right:5 bottom:15
      '5%'                 relative, top,right,bottom,left:5% of parent box width/height
      '.1 .5'              absolute, top,bottom:.1  left,right:.5 
      '.1 .3 rel'          relative, top,bottom:.1=10%  left,right:.3=30%
      '-10'                absolute, top,right,bottom,left:-10 relative to parent box (for crop box the media box gets expanded)

   Anchored within parent box, use dim and optionally pos, off:
      'dim: 200 300 abs'                   centered, 200x300 display units
      'pos:c, off:0 0, dim: 200 300 abs'   centered, 200x300 display units
      'pos:tl, off:5 5, dim: 50% 50% rel'  anchored to top left corner, 50% width/height of parent box, offset by 5/5 display units
      'pos:br, off:-5 -5, dim: .5 .5 rel'  anchored to bottom right corner, 50% width/height of parent box, offset by -5/-5 display units


`

	usageCrop     = "usage: pdfcpu crop [-p(ages) selectedPages] description inFile [outFile]" + generalFlags
	usageLongCrop = `Set crop box for selected pages. 

        pages ... selected pages
  description ... crop box definition abs. or rel. to media box
       inFile ... input pdf file
      outFile ... output pdf file

Examples:
   pdfcpu crop '[0 0 500 500]' in.pdf ... crop a 500x500 points region located in lower left corner
   pdfcpu crop -u mm '20' in.pdf      ... crop relative to media box using a 20mm margin

` + usageBoxDescription + usagePageSelection

	usageBoxesList   = "pdfcpu boxes list    [-p(ages) selectedPages] '[boxTypes]' inFile"
	usageBoxesAdd    = "pdfcpu boxes add     [-p(ages) selectedPages] description  inFile [outFile]"
	usageBoxesRemove = "pdfcpu boxes remove  [-p(ages) selectedPages] 'boxTypes'   inFile [outFile]" + generalFlags

	usageBoxes = "usage: " + usageBoxesList +
		"\n       " + usageBoxesAdd +
		"\n       " + usageBoxesRemove

	usageLongBoxes = `Manage page boundaries.

     boxTypes ... comma separated list of box types: m(edia), c(rop), t(rim), b(leed), a(rt)
  description ... box definitions abs. or rel. to parent box
       inFile ... input pdf file
      outFile ... output pdf file

<description> is a sequence of box definitions and assignments:

   m(edia): {box} 
    c(rop): {box} 
     a(rt): {box} | m(edia) | c(rop) | b(leed) | t(rim)
   b(leed): {box} | m(edia) | c(rop) | a(rt) | t(rim)
    t(rim): {box} | m(edia) | c(rop) | a(rt) | b(leed)
Examples: 
   pdfcpu box list in.pdf
   pdfcpu box l 'bleed,trim' in.pdf
   pdfcpu box add 'crop:[10 10 200 200], trim:5, bleed:trim' in.pdf
   pdfcpu box rem 't,b' in.pdf
     
` + usageBoxDescription + usagePageSelection
)
