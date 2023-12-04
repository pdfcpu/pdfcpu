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

   annotations   list, remove page annotations
   attachments   list, add, remove, extract embedded file attachments
   booklet       arrange pages onto larger sheets of paper to make a booklet or zine
   bookmarks     list, import, export, remove bookmarks
   boxes         list, add, remove page boundaries for selected pages
   changeopw     change owner password
   changeupw     change user password
   collect       create custom sequence of selected pages
   config        print configuration
   create        create PDF content including forms via JSON
   crop          set cropbox for selected pages
   cut           custom cut pages horizontally or vertically
   decrypt       remove password protection
   encrypt       set password protection		
   extract       extract images, fonts, content, pages or metadata
   fonts         install, list supported fonts, create cheat sheets
   form          list, remove fields, lock, unlock, reset, export, fill form via JSON or CSV
   grid          rearrange pages or images for enhanced browsing experience
   images        list images for selected pages
   import        import/convert images to PDF
   info          print file info
   keywords      list, add, remove keywords
   merge         concatenate PDFs
   ndown         cut selected pages into n pages symmetrically
   nup           rearrange pages or images for reduced number of pages
   optimize      optimize PDF by getting rid of redundant page resources
   pagelayout    list, set, reset page layout for opened document
   pagemode      list, set, reset page mode for opened document
   pages         insert, remove selected pages
   paper         print list of supported paper sizes
   permissions   list, set user access permissions
   portfolio     list, add, remove, extract portfolio entries with optional description
   poster        cut selected pages into poster using paper size or dimensions
   properties    list, add, remove document properties
   resize        scale selected pages
   rotate        rotate selected pages
   selectedpages print definition of the -pages flag
   split         split up a PDF by span or bookmark
   stamp         add, remove, update Unicode text, image or PDF stamps for selected pages
   trim          create trimmed version of selected pages
   validate      validate PDF against PDF 32000-1:2008 (PDF 1.7) + basic PDF 2.0 validation
   version       print version
   viewerpref    list, set, reset viewer preferences for opened document
   watermark     add, remove, update Unicode text, image or PDF watermarks for selected pages

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

	usageValidate = "usage: pdfcpu validate [-m(ode) strict|relaxed] [-l(inks)] inFile..." + generalFlags

	usageLongValidate = `Check inFile for specification compliance.

      mode ... validation mode
     links ... check for broken links
    inFile ... input PDF file
		
The validation modes are:

 strict ... validates against PDF 32000-1:2008 (PDF 1.7) and rudimentary against PDF 32000:2 (PDF 2.0)
relaxed ... (default) like strict but doesn't complain about common seen spec violations.`

	usageOptimize     = "usage: pdfcpu optimize [-stats csvFile] inFile [outFile]" + generalFlags
	usageLongOptimize = `Read inFile, remove redundant page resources like embedded fonts and images and write the result to outFile.

     stats ... appends a stats line to a csv file with information about the usage of root and page entries.
               useful for batch optimization and debugging PDFs.
    inFile ... input PDF file
   outFile ... output PDF file`

	usageSplit     = "usage: pdfcpu split [-m(ode) span|bookmark|page] inFile outDir [span|pageNr...]" + generalFlags
	usageLongSplit = `Generate a set of PDFs for the input file in outDir according to given span value or along bookmarks or page numbers.

      mode ... split mode (defaults to span)
    inFile ... input PDF file
    outDir ... output directory
      span ... split span in pages (default: 1) for mode "span"
    pageNr ... split before a specific page number for mode "page"
      
The split modes are:

      span     ... Split into PDF files with span pages each (default).
                   span itself defaults to 1 resulting in single page PDF files.
  
      bookmark ... Split into PDF files representing sections defined by existing bookmarks.
                   Assumption: inFile contains an outline dictionary.
                   
      page     ... Split before specific page numbers.
      
Eg. pdfcpu split test.pdf .      (= pdfcpu split -m span test.pdf . 1)
      generates:
         test_1.pdf
         test_2.pdf
         etc.

    pdfcpu split test.pdf . 2    (= pdfcpu split -m span test.pdf . 2)
      generates:
         test_1-2.pdf
         test_3-4.pdf
         etc.

    pdfcpu split -m bookmark test.pdf .
      generates:
         test_bm1Title_1-4.pdf
         test_bm2Title.5-7-pdf
         etc.

    pdfcpu split -m page test.pdf . 2 4 10
      generates:
         test_1.pdf
         test_2-3.pdf
         test_4-9.pdf
         test_10-20.pdf`

	usageMerge     = "usage: pdfcpu merge [-m(ode) create|append|zip] [ -s(ort) -b(ookmarks) -d(ivider)] outFile inFile..." + generalFlags
	usageLongMerge = `Concatenate a sequence of PDFs/inFiles into outFile.

      mode ... merge mode (defaults to create)
      sort ... sort inFiles by file name
 bookmarks ... create bookmarks
   divider ... insert blank page between merged documents
   outFile ... output PDF file
    inFile ... a list of PDF files subject to concatenation.
    
The merge modes are:

    create ... outFile will be created and possibly overwritten (default).

    append ... if outFile does not exist, it will be created (like in default mode).
               if outFile already exists, inFiles will be appended to outFile.

       zip ... zip inFile1 and inFile2 into outFile (which will be created and possibly overwritten).
               
Skip bookmark creation like so: -bookmarks=false`

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

	usageExtract     = "usage: pdfcpu extract -m(ode) i(mage)|f(ont)|c(ontent)|p(age)|m(eta) [-p(ages) selectedPages] inFile outDir" + generalFlags
	usageLongExtract = `Export inFile's images, fonts, content or pages into outDir.

      mode ... extraction mode
     pages ... Please refer to "pdfcpu selectedpages"
    inFile ... input PDF file
    outDir ... output directory

 The extraction modes are:

  image ... extract images
   font ... extract font files (supported font types: TrueType)
content ... extract raw page content
   page ... extract single page PDFs
   meta ... extract all metadata (page selection does not apply)
   
`

	usageTrim     = "usage: pdfcpu trim -p(ages) selectedPages inFile [outFile]" + generalFlags
	usageLongTrim = `Generate a trimmed version of inFile for selected pages.

     pages ... Please refer to "pdfcpu selectedpages"
    inFile ... input PDF file
   outFile ... output PDF file
   
`

	usageAttachList    = "pdfcpu attachments list    inFile"
	usageAttachAdd     = "pdfcpu attachments add     inFile file..."
	usageAttachRemove  = "pdfcpu attachments remove  inFile [file...]"
	usageAttachExtract = "pdfcpu attachments extract inFile outDir [file...]"

	usageAttach = "usage: " + usageAttachList +
		"\n       " + usageAttachAdd +
		"\n       " + usageAttachRemove +
		"\n       " + usageAttachExtract + generalFlags

	usageLongAttach = `Manage embedded file attachments.

    inFile ... input PDF file
      file ... attachment
    outDir ... output directory
    
    Remove all attachments: pdfcpu attach remove test.pdf
    `

	usagePortfolioList    = "pdfcpu portfolio list    inFile"
	usagePortfolioAdd     = "pdfcpu portfolio add     inFile file[,desc]..."
	usagePortfolioRemove  = "pdfcpu portfolio remove  inFile [file...]"
	usagePortfolioExtract = "pdfcpu portfolio extract inFile outDir [file...]"

	usagePortfolio = "usage: " + usagePortfolioList +
		"\n       " + usagePortfolioAdd +
		"\n       " + usagePortfolioRemove +
		"\n       " + usagePortfolioExtract + generalFlags

	usageLongPortfolio = `Manage portfolio entries.

    inFile ... input PDF file
      file ... attachment
      desc ... description (optional)
    outDir ... output directory
    
    Adding attachments to portfolio: 
           pdfcpu portfolio add test.pdf test.mp3 test.mkv

    Adding attachments to portfolio with description: 
           pdfcpu portfolio add test.pdf "test.mp3, Test sound file" "test.mkv, Test video file"
    `

	usagePermList = "pdfcpu permissions list [-upw userpw] [-opw ownerpw] inFile..."
	usagePermSet  = "pdfcpu permissions set [-perm none|print|all|max4Hex|max12Bits] [-upw userpw] -opw ownerpw inFile"

	usagePerm = "usage: " + usagePermList +
		"\n       " + usagePermSet + generalFlags

	usageLongPerm = `Manage user access permissions.

      perm ... user access permissions
    inFile ... input PDF file
    
   perm modes:
      
           none: 000000000000 (x000)
          print: 100000000100 (x804)
            all: 111100111100 (xF3C)
        max4Hex: x + max. 3 hex digits
      max12Bits: max. 12 binary digits

   using the permission bits:

      1:  -
      2:  -
      3:  Print (security handlers rev.2), draft print (security handlers >= rev.3)
      4:  Modify contents by operations other than controlled by bits 6, 9, 11.
      5:  Copy, extract text & graphics
      6:  Add or modify annotations, fill form fields, in conjunction with bit 4 create/mod form fields.
      7:  -
      8:  -
      9: Fill form fields (security handlers >= rev.3)
     10: Copy, extract text & graphics (security handlers >= rev.3) (unused since PDF 2.0)
     11: Assemble document (security handlers >= rev.3)
     12: Print (security handlers >= rev.3)`

	usageEncrypt     = "usage: pdfcpu encrypt [-m(ode) rc4|aes] [-key 40|128|256] [-perm none|print|all] [-upw userpw] -opw ownerpw inFile [outFile]" + generalFlags
	usageLongEncrypt = `Setup password protection based on user and owner password.

      mode ... algorithm (default=aes)
       key ... key length in bits (default=256)
      perm ... user access permissions
    inFile ... input PDF file
   outFile ... output PDF file`

	usageDecrypt     = "usage: pdfcpu decrypt [-upw userpw] [-opw ownerpw] inFile [outFile]" + generalFlags
	usageLongDecrypt = `Remove password protection and reset permissions.

    inFile ... input PDF file
   outFile ... output PDF file`

	usageChangeUserPW     = "usage: pdfcpu changeupw [-opw ownerpw] inFile upwOld upwNew" + generalFlags
	usageLongChangeUserPW = `Change the user password also known as the open doc password.

       opw ... owner password, required unless = ""
    inFile ... input PDF file
    upwOld ... old user password
    upwNew ... new user password`

	usageChangeOwnerPW     = "usage: pdfcpu changeopw [-upw userpw] inFile opwOld opwNew" + generalFlags
	usageLongChangeOwnerPW = `Change the owner password also known as the set permissions password.

       upw ... user password, required unless = ""
    inFile ... input PDF file
    opwOld ... old owner password (provide user password on initial changeopw)
    opwNew ... new owner password`

	usageStampMode = `There are 3 different kinds of stamps:

   1) text based:
      -mode text string			
         eg. pdfcpu stamp add -mode text -- "Hello gopher!" "" in.pdf out.pdf
         Use the following format strings:
               %p ... current page number
               %P ... total pages
         eg. pdfcpu stamp add -mode text -- "Page %p of %P" "sc:1.0 abs, pos:bc, rot:0" in.pdf out.pdf
   
   2) image based
      -mode image imageFileName
         supported extensions: .jpg, .jpeg, .png, .tif, .tiff, .webp
         eg. pdfcpu stamp add -mode image -- "logo.png" "" in.pdf out.pdf
         
   3) PDF based
      -mode pdf PDFFileName:page#
         Stamp selected pages of infile with one specific page of a stamp PDF file.
         Eg: pdfcpu stamp add -mode pdf -- "stamp.pdf:3" "" in.pdf out.pdf ... stamp each page of in.pdf with page 3 of stamp.pdf
           
      -mode pdf PDFFileName
         Multistamp your file, meaning apply all pages of a stamp PDF file one by one to ascending pages of inFile.
         Eg: pdfcpu stamp add -mode pdf -- "stamp.pdf" "" in.pdf out.pdf ... multistamp all pages of in.pdf with ascending pages of stamp.pdf
   
      -mode pdf PDFFileName:startPage#Src:startPage#Dest
         Customize your multistamp by by starting with startPage#Src of a stamp PDF file.
         Apply repeatedly pages of the stamp file to inFile starting at startPage#Dest.
         Eg: pdfcpu stamp add -mode pdf -- "stamp.pdf:2:3" "" in.pdf out.pdf ... multistamp starting with page 2 of stamp.pdf onto page 3 of in.pdf
   `

	usageWatermarkMode = `There are 3 different kinds of watermarks:

   1) text based:
      -mode text string			
         eg. pdfcpu watermark add -mode text -- "Hello gopher!" "" in.pdf out.pdf
         Use the following format strings:
               %p ... current page number
               %P ... total pages
         eg. pdfcpu watermark add -mode text -- "Page %p of %P" "sc:1.0 abs, pos:bc, rot:0" in.pdf out.pdf
   
   2) image based
      -mode image imageFileName
         supported extensions: .jpg, .jpeg, .png, .tif, .tiff, .webp 
         eg. pdfcpu watermark add -mode image -- "logo.png" "" in.pdf out.pdf
         
   3) PDF based
      -mode pdf PDFFileName:page#
         Watermark selected pages of infile with one specific page of a watermark PDF file.
         Eg: pdfcpu watermark add -mode pdf -- "watermark.pdf:3" "" in.pdf out.pdf ... watermark each page of in.pdf with page 3 of watermark.pdf
        
      -mode pdf PDFFileName
         Multiwatermark your file, meaning apply all pages of a watermark PDF file one by one to ascending pages of inFile.
         Eg: pdfcpu watermark add -mode pdf -- "watermark.pdf" "" in.pdf out.pdf ... multiwatermark all pages of in.pdf with ascending pages of watermark.pdf

      -mode pdf PDFFileName:startPage#Src:startPage#Dest
         Customize your multiwatermark by by starting with startPage#Src of a watermark PDF file.
         Apply repeatedly pages of the watermark file to inFile starting at startPage#Dest.
         Eg: pdfcpu watermark add -mode pdf -- "watermark.pdf:2:3" "" in.pdf out.pdf ... multiwatermark starting with page 2 of watermark.pdf onto page 3 of in.pdf

   A watermark is the first content that gets rendered for a page.
   The visibility of the watermark depends on the transparency of all layers rendered on top.
`
	usageWMDescription = `

<description> is a comma separated configuration string containing these optional entries:
   
   (defaults: "font:Helvetica, points:24, rtl:off, pos:c, off:0,0 sc:0.5 rel, rot:0, d:1, op:1, m:0 and for all colors: 0.5 0.5 0.5")

   fontname:         Please refer to "pdfcpu fonts list"

   points:           fontsize in points, in combination with absolute scaling only.

   rtl:              render right to left (on/off, true/false, t/f)

   position:         one of the anchors:

                           tl|top-left     tc|top-center      tr|top-right
                            l|left          c|center           r|right
                           bl|bottom-left  bc|bottom-center   br|bottom-right

   offset:           (dx dy) in given display unit eg. '15 20'
   
   scalefactor:      0.0 < i <= 1.0 {r|rel} | 0.0 < i {a|abs}
                    
   aligntext:        l|left, c|center, r|right, j|justified (for text watermarks only)

   fillcolor:        color value to be used when rendering text, see also rendermode
                     for backwards compatibility "color" is also accepted.
   
   strokecolor:      color value to be used when rendering text, see also rendermode
   
   backgroundcolor:  color value for visualization of the bounding box background for text.
                     "bgcolor" is also accepted. 
   
   rotation:         -180.0 <= x <= 180.0
   
   diagonal:         render along diagonal
                     1..lower left to upper right
                     2..upper left to lower right (if present overrules r!)
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

   url:              Add link annotation for stamps only (omit https://)

A color value: 3 color intensities, where 0.0 < i < 1.0, eg 1.0, 
               or the hex RGB value: #RRGGBB, eg #FF0000 = red

All configuration string parameters support completion.

e.g. "pos:bl, off: 20 5"   "rot:45"                 "op:0.5, sc:0.5 abs, rot:0"
     "d:2"                 "sc:.75 abs, points:48"  "rot:-90, scale:0.75 rel"
     "f:Courier, sc:0.75, str: 0.5 0.0 0.0, rot:20"


`

	usageStampAdd    = "pdfcpu stamp add    [-p(ages) selectedPages] -m(ode) text|image|pdf -- string|file description inFile [outFile]"
	usageStampUpdate = "pdfcpu stamp update [-p(ages) selectedPages] -m(ode) text|image|pdf -- string|file description inFile [outFile]"
	usageStampRemove = "pdfcpu stamp remove [-p(ages) selectedPages] inFile [outFile]"

	usageStamp = "usage: " + usageStampAdd +
		"\n       " + usageStampUpdate +
		"\n       " + usageStampRemove + generalFlags

	usageLongStamp = `Process stamping for selected pages. 

      pages ... Please refer to "pdfcpu selectedpages"
        upw ... user password
        opw ... owner password
       mode ... text, image, PDF
     string ... display string for text based watermarks
       file ... image or PDF file
description ... fontname, points, position, offset, scalefactor, aligntext, rotation, 
                diagonal, opacity, rendermode, strokecolor, fillcolor, bgcolor, margins, border
     inFile ... input PDF file
    outFile ... output PDF file

` + usageStampMode + usageWMDescription

	usageWatermarkAdd    = "pdfcpu watermark add    [-p(ages) selectedPages] -m(ode) text|image|pdf -- string|file description inFile [outFile]"
	usageWatermarkUpdate = "pdfcpu watermark update [-p(ages) selectedPages] -m(ode) text|image|pdf -- string|file description inFile [outFile]"
	usageWatermarkRemove = "pdfcpu watermark remove [-p(ages) selectedPages] inFile [outFile]"

	usageWatermark = "usage: " + usageWatermarkAdd +
		"\n       " + usageWatermarkUpdate +
		"\n       " + usageWatermarkRemove + generalFlags

	usageLongWatermark = `Process watermarking for selected pages. 

      pages ... Please refer to "pdfcpu selectedpages"
       mode ... text, image, PDF
     string ... display string for text based watermarks
       file ... image or PDF file
description ... fontname, points, position, offset, scalefactor, aligntext, rotation,
                diagonal, opacity, rendermode, strokecolor, fillcolor, bgcolor, margins, border
     inFile ... input PDF file
    outFile ... output PDF file

` + usageWatermarkMode + usageWMDescription

	usageImportImages     = "usage: pdfcpu import -- [description] outFile imageFile..." + generalFlags
	usageLongImportImages = `Turn image files into a PDF page sequence and write the result to outFile.
If outFile already exists the page sequence will be appended.
Each imageFile will be rendered to a separate page.
In its simplest form this converts an image into a PDF: "pdfcpu import img.pdf img.jpg"

description ... dimensions, format, position, offset, scale factor, boxes
    outFile ... output PDF file
  imageFile ... a list of image files
  
  <description> is a comma separated configuration string containing:

  optional entries:

      (defaults: "d:595 842, f:A4, pos:full, off:0 0, sc:0.5 rel, dpi:72, gray:off, sepia:off")

  dimensions:      (width height) in given display unit eg. '400 200' setting the media box

  formsize:        eg. A4, Letter, Legal...
                   Append 'L' to enforce landscape mode. (eg. A3L)
                   Append 'P' to enforce portrait mode. (eg. TabloidP)
                   Please refer to "pdfcpu paper" for a comprehensive list of defined paper sizes.
                   "papersize" is also accepted.

  position:        one of 'full' or the anchors:

                        tl|top-left     tc|top-center      tr|top-right
                         l|left          c|center           r|right
                        bl|bottom-left  bc|bottom-center   br|bottom-right

  offset:          (dx dy) in given display unit eg. '15 20'

  scalefactor:     0.0 <= x <= 1.0 followed by optional 'abs|rel' or 'a|r'

  dpi:             apply desired dpi

  gray:            Convert to grayscale (on/off, true/false, t/f)

  sepia:           Apply sepia effect (on/off, true/false, t/f)

  backgroundcolor: "bgcolor" is also accepted.
  
  Only one of dimensions or format is allowed.
  position: full => image dimensions equal page dimensions.
  
  All configuration string parameters support completion.

  e.g. "f:A5, pos:c"                              ... render the image centered on A5 with relative scaling 0.5.'
       "d:300 600, pos:bl, off:20 20, sc:1.0 abs" ... render the image anchored to bottom left corner with offset 20,20 and abs. scaling 1.0.
       "pos:full"                                 ... render the image to a page with corresponding dimensions.
       "f:A4, pos:c, dpi:300"                     ... render the image centered on A4 respecting a destination resolution of 300 dpi.
       `

	usagePagesInsert = "pdfcpu pages insert [-p(ages) selectedPages] [-m(ode) before|after] inFile [outFile]"
	usagePagesRemove = "pdfcpu pages remove  -p(ages) selectedPages  inFile [outFile]"
	usagePages       = "usage: " + usagePagesInsert +
		"\n       " + usagePagesRemove + generalFlags

	usageLongPages = `Manage pages.

      pages ... Please refer to "pdfcpu selectedpages"
       mode ... before, after (default: before)
     inFile ... input PDF file
    outFile ... output PDF file

`

	usageRotate     = "usage: pdfcpu rotate [-p(ages) selectedPages] inFile rotation [outFile]" + generalFlags
	usageLongRotate = `Rotate selected pages by a multiple of 90 degrees. 

      pages ... Please refer to "pdfcpu selectedpages"
     inFile ... input PDF file
   rotation ... a multiple of 90 degrees for clockwise rotation
    outFile ... output PDF file

`

	usageNUp     = "usage: pdfcpu nup [-p(ages) selectedPages] -- [description] outFile n inFile|imageFiles..." + generalFlags
	usageLongNUp = `Rearrange existing PDF pages or images into a sequence of page grids.
This reduces the number of pages and therefore the required print time.
If the input is one imageFile a single page n-up PDF gets generated.

      pages ... inFile only, please refer to "pdfcpu selectedpages"
description ... dimensions, format, orientation
    outFile ... output PDF file
          n ... the n-Up value (see below for details)
     inFile ... input PDF file
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
  
        (defaults: "di:595 842, form:A4, or:rd, bo:on, ma:3")
  
    dimensions:      (width,height) in given display unit eg. '400 200'
    formsize:        The output sheet size, eg. A4, Letter, Legal...
                     Append 'L' to enforce landscape mode. (eg. A3L)
                     Append 'P' to enforce portrait mode. (eg. TabloidP)
                     Only one of dimensions or format is allowed.
                     Please refer to "pdfcpu paper" for a comprehensive list of defined paper sizes.
                     "papersize" is also accepted.
    orientation:     one of rd ... right down (=default)
                            dr ... down right
                            ld ... left down
                            dl ... down left
                     Orientation applies to PDF input files only.
    border:          Print border (on/off, true/false, t/f) 
    margin:          for n-up content: float >= 0 in given display unit
    backgroundcolor: backgound color for margin > 0.
                     "bgcolor" is also accepted.

All configuration string parameters support completion.
    
Examples: pdfcpu nup out.pdf 4 in.pdf
           Rearrange pages of in.pdf into 2x2 grids and write result to out.pdf using the default orientation
           and default paper size A4. in.pdf's page size will be preserved.
                                 
          pdfcpu nup -pages=3- -- out.pdf 6 in.pdf
           Rearrange selected pages of in.pdf (all pages starting with page 3) into 3x2 grids and
           write result to out.pdf using the default orientation and default paper size A4.
           in.pdf's page size will be preserved.

          pdfcpu nup out.pdf 9 logo.jpg
           Arrange instances of logo.jpg into a 3x3 grid and write result to out.pdf using the A4 default format.
          
          pdfcpu nup -- "form:Tabloid" out.pdf 4 *.jpg 
           Rearrange all jpg files into 2x2 grids and write result to out.pdf using the Tabloid format
           and the default orientation.

`

	usageBooklet     = "usage: pdfcpu booklet [-p(ages) selectedPages] -- [description] outFile n inFile|imageFiles..." + generalFlags
	usageLongBooklet = `Arrange a sequence of pages onto larger sheets of paper for a small book or zine.

              pages       ... for inFile only, please refer to "pdfcpu selectedpages"
              description ... dimensions, formsize, border, margin
              outFile     ... output PDF file
              n           ... booklet style (2 or 4)
              inFile      ... input PDF file
              imageFiles  ... input image file(s)

There are two styles of booklet, depending on your page/input and sheet/output size:

n=2: Two of your pages fit on one side of a sheet (eg statement on letter, A5 on A4)
Assemble by printing on both sides (odd pages on the front and even pages on the back) and folding down the middle.

A variant of n=2 is a technique to bind your own hardback book.
It works best when the source PDF holding your book content has at least 128 pages.
You bind your paper in eight sheet folios each making up 32 pages of your book.
Each sheet is going to make four pages of your book, gets printed on both sides and folded in half.
For such a multi folio booklet set 'multifolio:on' and play around with 'foliosize' which defaults to 8.

n=4: Four of your pages fit on one side of a sheet (eg statement on ledger, A5 on A3, A6 on A4)
Assemble by printing on both sides, then cutting the sheets horizontally.
The sets of pages on the bottom of the sheet are rotated so that the cut side of the
paper is on the bottom of the booklet for every page. After cutting, place the bottom
set of pages after the top set of pages in the booklet. Then fold the half sheets.

                             portrait landscape
 Possible values for n: 2 ...  1x2       2x1
                        4 ...  2x2       2x2

<description> is a comma separated configuration string containing these optional entries:

   (defaults: "dim:595 842, formsize:A4, border:off, guides:off, margin:0")

   dimensions:       (width,height) of the output sheet in given display unit eg. '400 200'
   formsize:         The output sheet size, eg. A4, Letter, Legal...
                     Append 'L' to enforce landscape mode. (eg. A3L)
                     Append 'P' to enforce portrait mode. (eg. TabloidP)
                     Only one of dimensions or format is allowed.
                     Please refer to "pdfcpu paper" for a comprehensive list of defined paper sizes.
                     "papersize" is also accepted.
   multifolio:       Generate multi folio booklet (on/off, true/false, t/f) for n=2 and PDF input only.
   foliosize:        folio size for multi folio booklets only (default:8)
   border:           Print border (on/off, true/false, t/f) 
   guides:           Print folding and cutting lines (on/off, true/false, t/f)
   margin:           Apply content margin (float >= 0 in given display unit)
   backgroundcolor:  sheet backgound color for margin > 0.
                     "bgcolor" is also accepted.

All configuration string parameters support completion.

Examples: pdfcpu booklet -- "formsize:Letter" out.pdf 2 in.pdf
           Arrange pages of in.pdf 2 per sheet side (4 per sheet, back and front) onto out.pdf

          pdfcpu booklet -- "formsize:Ledger" out.pdf 4 in.pdf"
           Arrange pages of in.pdf 4 per sheet side (8 per sheet, back and front) onto out.pdf

          pdfcpu booklet -- "formsize:A4" out.pdf 2 in.pdf
           Arrange pages of in.pdf 2 per sheet side (4 per sheet, back and front) onto out.pdf

          pdfcpu booklet -- "formsize:A4, multifolio:on" hardbackbook.pdf 2 in.pdf
           Arrange pages of in.pdf 2 per sheetside as sequence of folios covering 4*foliosize pages each.
           See also: https://www.instructables.com/How-to-bind-your-own-Hardback-Book/
`

	usageGrid     = "usage: pdfcpu grid [-p(ages) selectedPages] -- [description] outFile m n inFile|imageFiles..." + generalFlags
	usageLongGrid = `Rearrange PDF pages or images for enhanced browsing experience.
For a PDF inputfile each output page represents a grid of input pages.
For image inputfiles each output page shows all images laid out onto grids of given paper size. 
This command produces poster like PDF pages convenient for page and image browsing. 

      pages ... Please refer to "pdfcpu selectedpages"
description ... dimensions, format, orientation
    outFile ... output PDF file
          m ... grid lines
          n ... grid columns
     inFile ... input PDF file
 imageFiles ... input image file(s)

    <description> is a comma separated configuration string containing:

    optional entries:
  
        (defaults: "d:595 842, form:A4, o:rd, bo:on, ma:3")
  
    dimensions:   (width height) in given display unit eg. '400 200'
    formsize:     The output sheet size, eg. A4, Letter, Legal...
                  Append 'L' to enforce landscape mode. (eg. A3L)
                  Append 'P' to enforce portrait mode. (eg. TabloidP)
                  Only one of dimensions or format is allowed.
                  Please refer to "pdfcpu paper" for a comprehensive list of defined paper sizes.
                  "papersize" is also accepted.
    orientation:  one of rd ... right down (=default)
                         dr ... down right
                         ld ... left down
                         dl ... down left
                  Orientation applies to PDF input files only.
    border:       Print border (on/off, true/false, t/f) 
    margin:       Apply content margin (float >= 0 in given display unit)

All configuration string parameters support completion.

Examples: pdfcpu grid out.pdf 1 10 in.pdf
           Rearrange pages of in.pdf into 1x10 grids and write result to out.pdf using the default orientation.
           The output page size is the result of a 1(vert)x10(hor) page grid using in.pdf's page size.

          pdfcpu grid -- "p:LegalL" out.pdf 2 2 in.pdf 
           Rearrange pages of in.pdf into 2x2 grids and write result to out.pdf using the default orientation.
           The output page size is the result of a 2(vert)x2(hor) page grid using page size Legal in landscape mode.

          pdfcpu grid -- "o:rd" out.pdf 3 2 in.pdf 
           Rearrange pages of in.pdf into 3x2 grids and write result to out.pdf using orientation 'right down'.
           The output page size is the result of a 3(vert)x2(hor) page grid using in.pdf's page size.

          pdfcpu grid -- "d:400 400" out.pdf 8 6 *.jpg
           Arrange imagefiles onto a 8x6 page grid and write result to out.pdf using a grid cell size of 400x400.

`

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

	usageVersion     = "usage: pdfcpu version"
	usageLongVersion = "Print the pdfcpu version & build info."

	usagePaper     = "usage: pdfcpu paper"
	usageLongPaper = "Print a list of supported paper sizes."

	usageConfig     = "usage: pdfcpu config"
	usageLongConfig = "Print configuration."

	usageSelectedPages     = "usage: pdfcpu selectedpages"
	usageLongSelectedPages = "Print definition of the -pages flag."

	usageInfo     = "usage: pdfcpu info [-p(ages) selectedPages] [-j(son)] inFile..." + generalFlags
	usageLongInfo = `Print info about a PDF file.
   
   pages ... Please refer to "pdfcpu selectedpages"
    json ... output JSON
  inFile ... a list of PDF input files`

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
	usageKeywordsRemove = "pdfcpu keywords remove  inFile [keyword...]"

	usageKeywords = "usage: " + usageKeywordsList +
		"\n       " + usageKeywordsAdd +
		"\n       " + usageKeywordsRemove + generalFlags

	usageLongKeywords = `Manage keywords.

    inFile ... input PDF file
   keyword ... search keyword
    
    Eg. adding two keywords: 
           pdfcpu keywords add test.pdf music 'virtual instruments'

        remove all keywords:
           pdfcpu keywords remove test.pdf
    `

	usagePropertiesList   = "pdfcpu properties list    inFile"
	usagePropertiesAdd    = "pdfcpu properties add     inFile nameValuePair..."
	usagePropertiesRemove = "pdfcpu properties remove  inFile [name...]"

	usageProperties = "usage: " + usagePropertiesList +
		"\n       " + usagePropertiesAdd +
		"\n       " + usagePropertiesRemove + generalFlags

	usageLongProperties = `Manage document properties.

       inFile ... input PDF file
nameValuePair ... 'name = value'
         name ... property name
     
     Eg. adding one property:   pdfcpu properties add test.pdf 'key = value'
         adding two properties: pdfcpu properties add test.pdf 'key1 = val1' 'key2 = val2'

         remove all properties: pdfcpu properties remove test.pdf
     `
	usageCollect     = "usage: pdfcpu collect -p(ages) selectedPages inFile [outFile]" + generalFlags
	usageLongCollect = `Create custom sequence of selected pages. 

        pages ... Please refer to "pdfcpu selectedpages"
       inFile ... input PDF file
      outFile ... output PDF file
  
  `

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
      "0.5 0.5 20 20"      absolute, top:.5 right:.5 bottom:20 left:20
      "0.5 0.5 .1 .1 abs"  absolute, top:.5 right:.5 bottom:.1 left:.1
      "0.5 0.5 .1 .1 rel"  relative, top:.5 right:.5 bottom:20 left:20
      "10"                 absolute, top,right,bottom,left:10
      "10 5"               absolute, top,bottom:10  left,right:5
      "10 5 15"            absolute, top:10 left,right:5 bottom:15
      "5%"                 relative, top,right,bottom,left:5% of parent box width/height
      ".1 .5"              absolute, top,bottom:.1  left,right:.5 
      ".1 .3 rel"          relative, top,bottom:.1=10%  left,right:.3=30%
      "-10"                absolute, top,right,bottom,left:-10 relative to parent box (for crop box the media box gets expanded)

   Anchored within parent box, use dim and optionally pos, off:
      "dim: 200 300 abs"                   centered, 200x300 display units
      "pos:c, off:0 0, dim: 200 300 abs"   centered, 200x300 display units
      "pos:tl, off:5 5, dim: 50% 50% rel"  anchored to top left corner, 50% width/height of parent box, offset by 5/5 display units
      "pos:br, off:-5 -5, dim: .5 .5 rel"  anchored to bottom right corner, 50% width/height of parent box, offset by -5/-5 display units


`

	usageCrop     = "usage: pdfcpu crop [-p(ages) selectedPages] -- description inFile [outFile]" + generalFlags
	usageLongCrop = `Set crop box for selected pages. 

        pages ... Please refer to "pdfcpu selectedpages"
  description ... crop box definition abs. or rel. to media box
       inFile ... input PDF file
      outFile ... output PDF file

Examples:
   pdfcpu crop -- "[0 0 500 500]" in.pdf ... crop a 500x500 points region located in lower left corner
   pdfcpu crop -u mm -- "20" in.pdf      ... crop relative to media box using a 20mm margin

` + usageBoxDescription

	usageBoxesList   = "pdfcpu boxes list    [-p(ages) selectedPages] -- [boxTypes] inFile"
	usageBoxesAdd    = "pdfcpu boxes add     [-p(ages) selectedPages] -- description inFile [outFile]"
	usageBoxesRemove = "pdfcpu boxes remove  [-p(ages) selectedPages] -- boxTypes inFile [outFile]"

	usageBoxes = "usage: " + usageBoxesList +
		"\n       " + usageBoxesAdd +
		"\n       " + usageBoxesRemove + generalFlags

	usageLongBoxes = `Manage page boundaries.

     boxTypes ... comma separated list of box types: m(edia), c(rop), t(rim), b(leed), a(rt)
        pages ... Please refer to "pdfcpu selectedpages"
  description ... box definitions abs. or rel. to parent box
       inFile ... input PDF file
      outFile ... output PDF file

<description> is a sequence of box definitions and assignments:

   m(edia): {box} 
    c(rop): {box} 
     a(rt): {box} | m(edia) | c(rop) | b(leed) | t(rim)
   b(leed): {box} | m(edia) | c(rop) | a(rt) | t(rim)
    t(rim): {box} | m(edia) | c(rop) | a(rt) | b(leed)

Examples: 
   pdfcpu box list in.pdf
   pdfcpu box l -- "bleed,trim" in.pdf
   pdfcpu box add -- "crop:[10 10 200 200], trim:5, bleed:trim" in.pdf
   pdfcpu box rem -- "t,b" in.pdf
     
` + usageBoxDescription

	usageAnnotsList   = "pdfcpu annotations list   [-p(ages) selectedPages] inFile"
	usageAnnotsRemove = "pdfcpu annotations remove [-p(ages) selectedPages] inFile [outFile] [objNr|annotId|annotType]..."

	usageAnnots = "usage: " + usageAnnotsList +
		"\n       " + usageAnnotsRemove + generalFlags

	usageLongAnnots = `Manage annotations.
   
      pages ... Please refer to "pdfcpu selectedpages"
     inFile ... input PDF file
      objNr ... obj# from "pdfcpu annotations list"
    annotId ... id from "pdfcpu annotations list"
  annotType ... Text, Link, FreeText, Line, Square, Circle, Polygon, PolyLine, HighLight, Underline, Squiggly, StrikeOut, Stamp,
                Caret, Ink, Popup, FileAttachment, Sound, Movie, Widget, Screen, PrinterMark, TrapNet, Watermark, 3D, Redact
   
   Examples:

      List all annotations:
         pdfcpu annot list in.pdf

      List annotation of first two pages:
         pdfcpu annot list -pages 1-2 in.pdf

      Remove all page annotations and write to out.pdf:
         pdfcpu annot remove in.pdf out.pdf
      
      Remove annotations for first 10 pages:
         pdfcpu annot remove -pages 1-10 in.pdf

      Remove annotations with obj# 37, 38 (see output of pdfcpu annot list)
         pdfcpu annot remove in.pdf 37 38

      Remove all Widget annotations and write to out.pdf:
         pdfcpu annot remove in.pdf out.pdf Widget

      Remove all Ink and Widget annotations on page 3:
         pdfcpu annot remove -pages 3 in.pdf Ink Widget

      Remove annotations by type, id and obj# and write to out.pdf:
         pdfcpu annot remove in.pdf out.pdf Link 30 Text someId
      `

	usageImagesList = "pdfcpu images list [-p(ages) selectedPages] inFile..." + generalFlags

	usageImages = "usage: " + usageImagesList

	usageLongImages = `Manage keywords.

     pages ... Please refer to "pdfcpu selectedpages"
    inFile ... input PDF file
    
    Example: pdfcpu images list -p "1-5" gallery.pdf
    `

	usageCreate     = "usage: pdfcpu create inFileJSON [inFile] outFile" + generalFlags
	usageLongCreate = `Create page content corresponding to declarations in inFileJSON.
Append new page content to existing page content in inFile and write result to outFile.
If inFile is absent outFile will be overwritten.

   inFileJSON ... input json file
       inFile ... optional input PDF file 
      outFile ... output PDF file

A minimalistic sample json:
{
   "pages": {
      "1": {
         "content": {
            "text": [
               {
                  "value": "Hello pdfcpu user!",
                  "anchor": "center",
                  "font": {
                     "name": "Helvetica",
                     "size": 12
                   }
               }
            ]
         }
      }
   }
}
   
For more info on json syntax & samples please refer to :
   pdfcpu/pkg/testdata/json/*
   pdfcpu/pkg/samples/create/*`

	usageFormListFields   = "pdfcpu form list   inFile..."
	usageFormRemoveFields = "pdfcpu form remove inFile [outFile] <fieldID|fieldName>..."
	usageFormLock         = "pdfcpu form lock   inFile [outFile] [fieldID|fieldName]..."
	usageFormUnlock       = "pdfcpu form unlock inFile [outFile] [fieldID|fieldName]..."
	usageFormReset        = "pdfcpu form reset  inFile [outFile] [fieldID|fieldName]..."
	usageFormExport       = "pdfcpu form export inFile [outFileJSON]"
	usageFormFill         = "pdfcpu form fill inFile inFileJSON [outFile]"
	usageFormMultiFill    = "pdfcpu form multifill [-m(ode) single|merge] inFile inFileData outDir [outName]"

	usageForm = "usage: " + usageFormListFields +
		"\n       " + usageFormRemoveFields +
		"\n       " + usageFormLock +
		"\n       " + usageFormUnlock +
		"\n       " + usageFormReset +
		"\n       " + usageFormExport +
		"\n\n       " + usageFormFill +
		"\n       " + usageFormMultiFill + generalFlags

	usageLongForm = `Manage PDF forms.

           inFile ... input PDF file
       inFileData ... input CSV or JSON file
       inFileJSON ... input JSON file
          outFile ... output PDF file
      outFileJSON ... output JSON file
             mode ... output mode (defaults to single)
           outDir ... output directory
          outName ... base output name
          fieldID ... as indicated by "pdfcpu form list"
        fieldName ... as indicated by "pdfcpu form list"

The output modes are:

    single ... each filled form instance gets written to a separate output file.

    merge  ... all filled form instances are merged together resulting in one output file.
               

Supported usecases:

   1) Get a list of form fields:
         "pdfcpu form list in.pdf" returns a list of form fields of in.pdf.
         Each field is identified by its name and id.
   
   2) Remove some form fields:
         "pdfcpu form remove in.pdf middleName birthPlace" removes the the two fields "middleName" and "birthPlace".
         You may supply a mixed list of field ids and field names.
      
   3) Make some or all fields read-only:
         "pdfcpu form lock in.pdf dateOfBirth" turns the field "dateOfBirth" into read-only.
         "pdfcpu from lock in.pdf" makes the form read-only.
         You may supply a mixed list of field ids and field names.
   
   4) Make some or all read-only fields writeable:
         "pdfcpu form unlock in.pdf dateOfBirth" makes the field "dateOfBirth" writeable.
         "pdfcpu form unlock in.pdf" makes all fields of in.pdf writeable.
         You may supply a mixed list of field ids and field names.
   
   5) Clear some or all fields:
         "pdfcpu form reset in.pdf firstName lastName" resets the fields "firstName" and "lastName" to its default values.
         "pdfcpu form reset in.pdf" resets the whole form of in.pdf.
         You may supply a mixed list of field ids and field names.
       
   6) Export all form fields as preparation for form filling:
         "pdfcpu form export in.pdf" exports field data into a JSON structure written to in.json.
   
   7) Fill a form with data:
         a) Export your form into in.json and edit the field values.
         b) Optionally trim down each field to id or name and value(s).
         c) "pdfcpu form fill in.pdf in.json out.pdf" fills in.pdf with form data from in.json and writes the result to out.pdf.

   or

   8) Generate a sequence of filled instances of a form:
         a) Export your form to in.json and edit the field values.
            Extend the JSON Array containing the form by using copy & paste and edit the corresponding form data.
         b) Optionally trim down each field to id or name and value(s).
         c) "pdfcpu form multifill in.pdf in.json outDir" creates a separate PDF for each filled form instance in outDir.
      or
         a) Export your form to in.json.
         b) Create a CSV file holding form instance data where each CSV line corresponds to one form data tuple.
            The first line identifies fields via id or name from in.json.
         c) "pdfcpu form multifill in.pdf in.csv outDir" creates a separate PDF for each filled form instance in outDir.

   or

   9) Generate a sequence of filled instances of a form and merge output:
         a) Export your form to in.json and edit the field values.
            Extend the JSON Array containing the form by using copy & paste and edit the corresponding form data.
         b) Optionally trim down each field to id or name and value(s).
         c) "pdfcpu form multifill -m merge in.pdf in.json outDir" creates a single output PDF in outDir.
      or
         a) Export your form to in.json.
         b) Create a CSV file holding form instance data where each CSV line corresponds to one form data tuple.
            The first line identifies fields via id or name in in.json.
         c) "pdfcpu form multifill -m merge in.pdf in.csv outDir" creates a single output PDF in outDir.


   (For syntax and details please refer to pdfcpu/pkg/api/test/form_test.go)`

	usageResize     = "usage: pdfcpu resize [-p(ages) selectedPages] -- description inFile [outFile]" + generalFlags
	usageLongResize = `Resize existing pages.

      pages ... please refer to "pdfcpu selectedpages"
description ... scalefactor, dimensions, formsize, enforce, border, bgcolor
     inFile ... input PDF file
    outFile ... output PDF file

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

      enforce:      if dimensions set only, enforce orientation (on/off, true/false, t/f).

      border:       if dimensions set only, draw content region border (on/off, true/false, t/f).

      bgcolor:      if dimensions set only, background color value for unused page regions.
   
      
   Examples: 

         pdfcpu resize "scale:2" in.pdf out.pdf
            Enlarge pages by doubling the page dimensions, keep orientation.

         pdfcpu resize -pages 1-3 -- "sc:.5" in.pdf out.pdf
            Shrink first 3 pages by cutting in half the page dimensions, keep orientation.

         pdfcpu resize -u cm -- "dim:40 0" in.pdf out.pdf
            Resize pages to width of 40 cm, keep orientation.

         pdfcpu resize "form:A4" in.pdf out.pdf
            Resize pages to A4, keep orientation.

         pdfcpu resize "f:A4P, bgcol:#d0d0d0" in.pdf out.pdf
            Resize pages to A4 and enforce orientation(here: portrait mode), apply background color.

         pdfcpu resize "dim:400 200" in.pdf out.pdf
            Resize pages to 400 x 200 points, keep orientation.

         pdfcpu resize "dim:400 200, enforce:true" in.pdf out.pdf
            Resize pages to 400 x 200 points, enforce orientation.
`
	usagePoster     = "usage: pdfcpu poster [-p(ages) selectedPages] -- description inFile outDir [outFileName]" + generalFlags
	usageLongPoster = `Create a poster using paper size.

         pages ... Please refer to "pdfcpu selectedpages"
   description ... formsize(=papersize), dimensions, scalefactor, margin, bgcolor, border
        inFile ... input PDF file
        outDir ... output directory
   outFileName ... output file name

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

         pdfcpu poster "f:A4" in.pdf outDir
            Page format is A2, the printer supports A4.
            Generate a poster(A2) via a corresponding 2x2 grid of A4 pages.
         
         pdfcpu poster "f:A4, scale:2.0" in.pdf outDir
            Page format is A2, the printer supports A4.
            Generate a poster(A0) via a corresponding 4x4 grid of A4 pages.

         pdfcpu poster -u cm -- "dim:15 10, margin:1, bgcol:DarkGray, border:on" in.pdf outDir
            Generate a poster via a corresponding grid with cell size 15x10 cm and provide a glue area of 1 cm.
            
   See also the related commands: ndown, cut`

	usageNDown     = "usage: pdfcpu ndown [-p(ages) selectedPages] -- [description] n inFile outDir [outFileName]" + generalFlags
	usageLongNDown = `Cut selected page into n pages symmetrically.

         pages ... Please refer to "pdfcpu selectedpages"
   description ... margin, bgcolor, border
             n ... the n-Down value (see below for details)
        inFile ... input PDF file
        outDir ... output directory
   outFileName ... output file name

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
            Page format is A2, the printer supports A3.
            Quick cut page into 2 equally sized pages.

         pdfcpu ndown 4 in.pdf outDir
            Page format is A2, the printer supports A4.
            Quick cut page into 4 equally (A4) sized pages.

         pdfcpu ndown -u cm -- "margin:1, bgcol:DarkGray, border:on" 4 in.pdf outDir
            Page format is A2, the printer supports A4.
            Quick cut page into 4 equally (A4) sized pages and provide a glue area of 1 cm.
            
   See also the related commands: poster, cut`

	usageCut     = "usage: pdfcpu cut [-p(ages) selectedPages] -- description inFile outDir [outFileName]" + generalFlags
	usageLongCut = `Custom cut pages horizontally or vertically.

         pages ... Please refer to "pdfcpu selectedpages"
   description ... horizontal, vertical, margin, bgcolor, border
        inFile ... input PDF file
        outDir ... output directory
   outFileName ... output file name

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

         pdfcpu cut -- "hor:.25" inFile outDir
            Apply a horizontal page cut at 0.25*height
            Results in 2 PDF pages.

         pdfcpu cut -- "hor:.25, vert:.75" inFile outDir
            Apply a horizontal page cut at 0.25*height
            Apply a vertical page cut at 0.75*width

         pdfcpu cut -- "hor:.33 .66" inFile outDir
            Has the same effect as: pdfcpu ndown 3 in.pdf outDir

         pdfcpu cut -- "hor:.5, ver:.5" inFile outDir
            Has the same effect as: pdfcpu ndown 4 in.pdf outDir
            
   See also the related commands: poster, ndown`

	usageBookmarksList   = "pdfcpu bookmarks list   inFile"
	usageBookmarksImport = "pdfcpu bookmarks import [-r(eplace)] inFile inFileJSON [outFile]"
	usageBookmarksExport = "pdfcpu bookmarks export inFile [outFileJSON]"
	usageBookmarksRemove = "pdfcpu bookmarks remove inFile [outFile]"

	usageBookmarks = "usage: " + usageBookmarksList +
		"\n       " + usageBookmarksImport +
		"\n       " + usageBookmarksExport +
		"\n       " + usageBookmarksRemove + generalFlags

	usageLongBookmarks = `Manage bookmarks.

           inFile ... input PDF file
       inFileJSON ... input JSON file
          outFile ... output PDF file
      outFileJSON ... output PDF file
`

	usagePageLayoutList  = "pdfcpu pagelayout list  inFile"
	usagePageLayoutSet   = "pdfcpu pagelayout set   inFile value"
	usagePageLayoutReset = "pdfcpu pagelayout reset inFile"

	usagePageLayout = "usage: " + usagePageLayoutList +
		"\n       " + usagePageLayoutSet +
		"\n       " + usagePageLayoutReset + generalFlags

	usageLongPageLayout = `Manage the page layout which shall be used when the document is opened:

    inFile ... input PDF file
     value ... one of:

     SinglePage     ... Display one page at a time (default)
     TwoColumnLeft  ... Display the pages in two columns, with odd- numbered pages on the left
     TwoColumnRight ... Display the pages in two columns, with odd- numbered pages on the right
     TwoPageLeft    ... Display the pages two at a time, with odd-numbered pages on the left
     TwoPageRight   ... Display the pages two at a time, with odd-numbered pages on the right
    
    Eg. set page layout:
           pdfcpu pagelayout set test.pdf TwoPageLeft

        reset page layout:
           pdfcpu pagelayout reset test.pdf
`

	usagePageModeList  = "pdfcpu pagemode list  inFile"
	usagePageModeSet   = "pdfcpu pagemode set   inFile value"
	usagePageModeReset = "pdfcpu pagemode reset inFile"

	usagePageMode = "usage: " + usagePageModeList +
		"\n       " + usagePageModeSet +
		"\n       " + usagePageModeReset + generalFlags

	usageLongPageMode = `Manage how the document shall be displayed when opened:

    inFile ... input PDF file
     value ... one of:

            UseNone ... Neither document outline nor thumbnail images visible (default)
        UseOutlines ... Document outline visible
          UseThumbs ... Thumbnail images visible
         FullScreen ... Full-screen mode, with no menu bar, window controls, or any other window visible
              UseOC ... Optional content group panel visible (since PDF 1.5)
     UseAttachments ... Attachments panel visible (since PDF 1.6)
    
    Eg. set page mode:
           pdfcpu pagemode set test.pdf UseOutlines

        reset page mode:
           pdfcpu pagemode reset test.pdf
    `

	usageViewerPreferencesList  = "pdfcpu viewerpref list [-a(ll)] [-j(son)] inFile"
	usageViewerPreferencesSet   = "pdfcpu viewerpref set                     inFile (inFileJSON | JSONstring)"
	usageViewerPreferencesReset = "pdfcpu viewerpref reset                   inFile"

	usageViewerPreferences = "usage: " + usageViewerPreferencesList +
		"\n       " + usageViewerPreferencesSet +
		"\n       " + usageViewerPreferencesReset + generalFlags

	usageLongViewerPreferences = `Manage the way the document shall be displayed on the screen and shall be printed:

              all ... output all (including default values)
             json ... output JSON
           inFile ... input PDF file
       inFileJSON ... input JSON file containing viewing preferences
       JSONstring ... JSON string containing viewing preferences
             
    
    The preferences are:

      HideToolbar           ... Hide tool bars when the document is active (default=false).
      HideMenubar           ... Hide the menu bar when the document is active (default=false).
      HideWindowUI          ... Hide user interface elements in the document’s window (default=false).
      FitWindow             ... Resize the document’s window to fit the size of the first displayed page (default=false).
      CenterWindow          ... Position the document’s window in the centre of the screen (default=false).
      DisplayDocTitle       ... true: The window’s title bar should display the document title taken from the dc:title element of the XMP metadata stream.
                                false: The title bar should display the name of the PDF file containing the document (default=false).

      NonFullScreenPageMode ... How to display the document on exiting full-screen mode:
                                    UseNone     = Neither document outline nor thumbnail images visible (=default)
                                    UseOutlines = Document outline visible
                                    UseThumbs   = Thumbnail images visible
                                    UseOC       = Optional content group panel visible

      Direction             ... The predominant logical content order for text
                                    L2R         = Left to right (=default)
                                    R2L         = Right to left (including vertical writing systems, such as Chinese, Japanese, and Korean)

      ViewArea              ... The name of the page boundary representing the area of a page that shall be displayed when viewing the document on the screen.
      ViewClip              ... The name of the page boundary to which the contents of a page shall be clipped when viewing the document on the screen.
      PrintArea             ... The name of the page boundary representing the area of a page that shall be rendered when printing the document.
      PrintClip             ... The name of the page boundary to which the contents of a page shall be clipped when printing the document.
                                    All 4 since PDF 1.4 and deprecated as of PDF 2.0                              
                                    Page Boundaries: MediaBox, CropBox(=default), TrimBox, BleedBox, ArtBox                             

      Duplex                ... The paper handling option that shall be used when printing the file from the print dialogue (since PDF 1.7):
                                    Simplex             = Print single-sided
                                    DuplexFlipShortEdge = Duplex and flip on the short edge of the sheet
                                    DuplexFlipLongEdge  = Duplex and flip on the long edge of the sheet

      PickTrayByPDFSize     ... Whether the PDF page size shall be used to select the input paper tray.

      PrintPageRange        ... The page numbers used to initialize the print dialogue box when the file is printed (since PDF 1.7).
                                The array shall contain an even number of integers to be interpreted in pairs, with each pair specifying
                                the first and last pages in a sub-range of pages to be printed. The first page of the PDF file shall be denoted by 1.
      
      NumCopies             ... The number of copies that shall be printed when the print dialog is opened for this file (since PDF 1.7).
     
      Enforce               ... Array of names of Viewer preference settings that shall be enforced by PDF processors and
                                that shall not be overridden by subsequent selections in the application user interface (since PDF 2.0).
                                    Possible values: PrintScaling

    Eg. list viewer preferences:
         pdfcpu viewerpref list test.pdf
         pdfcpu viewerpref list -all test.pdf
         pdfcpu viewerpref list -json test.pdf
         pdfcpu viewerpref list -all -json test.pdf

   reset viewer preferences:
         pdfcpu viewerpref reset test.pdf

   set printer preferences via JSON string (case agnostic):
         pdfcpu viewerpref set test.pdf "{\"HideMenuBar\": true, \"CenterWindow\": true}"
         pdfcpu viewerpref set test.pdf "{\"duplex\": \"duplexFlipShortEdge\", \"printPageRange\": [1, 4, 10, 12], \"NumCopies\": 3}"

   set viewer preferences via JSON file:
         pdfcpu viewerpref set test.pdf viewerpref.json

         and eg. viewerpref.json (each preferences is optional!):

         {
            "viewerPreferences": {
               "HideToolBar": true,
               "HideMenuBar": false,
               "HideWindowUI": false,
               "FitWindow": true,
               "CenterWindow": true,
               "DisplayDocTitle": true,
               "NonFullScreenPageMode": "UseThumbs",
               "Direction": "R2L",
               "Duplex": "Simplex",
               "PickTrayByPDFSize": false,
               "PrintPageRange": [
                  1, 4,
                  10, 20
               ],
               "NumCopies": 3,
               "Enforce": [
                  "PrintScaling"
               ]
            }
         }
   
    `
)
