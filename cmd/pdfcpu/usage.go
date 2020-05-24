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
   changeopw   change owner password
   changeupw   change user password
   collect     create custom sequence of selected pages
   decrypt     remove password protection
   encrypt     set password protection		
   extract     extract images, fonts, content, pages, metadata
   fonts       install, list supported fonts
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
   stamp       add, remove, update text, image or PDF stamps for selected pages
   trim        create trimmed version of selected pages
   validate    validate PDF against PDF 32000-1:2008 (PDF 1.7)
   version     print version
   watermark   add, remove, update text, image or PDF watermarks for selected pages

   All instantly recognizable command prefixes are supported eg. val for validation
   One letter Unix style abbreviations supported for flags and command parameters.

Use "pdfcpu help [command]" for more information about a command.`

	usageValidate     = "usage: pdfcpu validate [-v(erbose)|vv] [-q(uiet)] [-mode strict|relaxed] [-upw userpw] [-opw ownerpw] inFile"
	usageLongValidate = `Check inFile for specification compliance.

verbose, v ... turn on logging
        vv ... verbose logging
  quiet, q ... disable output
      mode ... validation mode
       upw ... user password
       opw ... owner password
    inFile ... input pdf file
		
The validation modes are:

 strict ... (default) validates against PDF 32000-1:2008 (PDF 1.7)
relaxed ... like strict but doesn't complain about common seen spec violations.`

	usageOptimize     = "usage: pdfcpu optimize [-v(erbose)|vv] [-q(uiet)] [-stats csvFile] [-upw userpw] [-opw ownerpw] inFile [outFile]"
	usageLongOptimize = `Read inFile, remove redundant page resources like embedded fonts and images and write the result to outFile.

verbose, v ... turn on logging
        vv ... verbose logging
  quiet, q ... disable output
     stats ... appends a stats line to a csv file with information about the usage of root and page entries.
               useful for batch optimization and debugging PDFs.
       upw ... user password
       opw ... owner password
    inFile ... input pdf file
   outFile ... output pdf file`

	usageSplit     = "usage: pdfcpu split [-v(erbose)|vv] [-q(uiet)] [-mode span|bookmark] [-upw userpw] [-opw ownerpw] inFile outDir [span]"
	usageLongSplit = `Generate a set of PDFs for the input file in outDir according to given span value or along bookmarks.

verbose, v ... turn on logging
        vv ... verbose logging
  quiet, q ... disable output
      mode ... split mode (defaults to span)
       upw ... user password
       opw ... owner password
    inFile ... input pdf file
    outDir ... output directory
      span ... split span in pages (default: 1) for mode "span"
      
The split modes are:

      span     ... Split into PDF files with span pages each (default).
                   span itself defaults to 1 resulting in single page PDF files.
  
      bookmark ... Split into PDF files representing sections defined by existing bookmarks.
                   span will be ignored.
                   Assumption: inFile contains an outline dictionary.`

	usageMerge     = "usage: pdfcpu merge [-v(erbose)|vv] [-q(uiet)] [-mode create|append] outFile inFile..."
	usageLongMerge = `Concatenate a sequence of PDFs/inFiles into outFile.
   
verbose, v ... turn on logging
        vv ... verbose logging
  quiet, q ... disable output
      mode ... merge mode (defaults to create)
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

	usageExtract     = "usage: pdfcpu extract [-v(erbose)|vv] [-q(uiet)] -mode image|font|content|page|meta [-pages selectedPages] [-upw userpw] [-opw ownerpw] inFile outDir"
	usageLongExtract = `Export inFile's images, fonts, content or pages into outDir.

verbose, v ... turn on logging
        vv ... verbose logging
  quiet, q ... disable output
      mode ... extraction mode
     pages ... selected pages
       upw ... user password
       opw ... owner password
    inFile ... input pdf file
    outDir ... output directory

 The extraction modes are:

  image ... extract images
   font ... extract font files (supported font types: TrueType)
content ... extract raw page content
   page ... extract single page PDFs
   meta ... extract all metadata (page selection does not apply)
   
` + usagePageSelection

	usageTrim     = "usage: pdfcpu trim [-v(erbose)|vv] [-q(uiet)] -pages selectedPages [-upw userpw] [-opw ownerpw] inFile [outFile]"
	usageLongTrim = `Generate a trimmed version of inFile for selected pages.

verbose, v ... turn on logging
        vv ... verbose logging
  quiet, q ... disable output
     pages ... selected pages
       upw ... user password
       opw ... owner password
    inFile ... input pdf file
   outFile ... output pdf file
   
` + usagePageSelection

	usageAttachList    = "pdfcpu attachments list    [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile"
	usageAttachAdd     = "pdfcpu attachments add     [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile file..."
	usageAttachRemove  = "pdfcpu attachments remove  [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile [file...]"
	usageAttachExtract = "pdfcpu attachments extract [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile outDir [file...]"

	usageAttach = "usage: " + usageAttachList +
		"\n       " + usageAttachAdd +
		"\n       " + usageAttachRemove +
		"\n       " + usageAttachExtract

	usageLongAttach = `Manage embedded file attachments.
	
verbose, v ... turn on logging
        vv ... verbose logging
  quiet, q ... disable output
       upw ... user password
       opw ... owner password
    inFile ... input pdf file
      file ... attachment
    outDir ... output directory`

	usagePortfolioList    = "pdfcpu portfolio list    [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile"
	usagePortfolioAdd     = "pdfcpu portfolio add     [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile file[,desc]..."
	usagePortfolioRemove  = "pdfcpu portfolio remove  [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile [file...]"
	usagePortfolioExtract = "pdfcpu portfolio extract [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile outDir [file...]"

	usagePortfolio = "usage: " + usagePortfolioList +
		"\n       " + usagePortfolioAdd +
		"\n       " + usagePortfolioRemove +
		"\n       " + usagePortfolioExtract

	usageLongPortfolio = `Manage portfolio entries.
	
verbose, v ... turn on logging
        vv ... verbose logging
  quiet, q ... disable output
       upw ... user password
       opw ... owner password
    inFile ... input pdf file
      file ... attachment
      desc ... description (optional)
    outDir ... output directory
    
    Adding attachments to portfolio: 
           pdfcpu portfolio add test.pdf test.mp3 test.mkv

    Adding attachments to portfolio with description: 
           pdfcpu portfolio add test.pdf 'test.mp3, Test sound file' 'test.mkv, Test video file'
    `

	usagePermList = "pdfcpu permissions list [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile"
	usagePermSet  = "pdfcpu permissions set [-v(erbose)|vv] [-q(uiet)] [-perm none|all] [-upw userpw] -opw ownerpw inFile"

	usagePerm = "usage: " + usagePermList +
		"\n       " + usagePermSet

	usageLongPerm = `Manage user access permissions.
	
verbose, v ... turn on logging
        vv ... verbose logging
  quiet, q ... disable output
      perm ... user access permissions
       upw ... user password
       opw ... owner password
    inFile ... input pdf file`

	usageEncrypt     = "usage: pdfcpu encrypt [-v(erbose)|vv] [-q(uiet)] [-mode rc4|aes] [-key 40|128|256] [perm none|all] [-upw userpw] -opw ownerpw inFile [outFile]"
	usageLongEncrypt = `Setup password protection based on user and owner password.

verbose, v ... turn on logging
        vv ... verbose logging
  quiet, q ... disable output
      mode ... algorithm (default=aes)
       key ... key length in bits (default=256)
      perm ... user access permissions
       upw ... user password
       opw ... owner password (must not be empty!)
    inFile ... input pdf file
   outFile ... output pdf file`

	usageDecrypt     = "usage: pdfcpu decrypt [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile [outFile]"
	usageLongDecrypt = `Remove password protection and reset permissions.

verbose, v ... turn on logging
        vv ... verbose logging
  quiet, q ... disable output
       upw ... user password
       opw ... owner password
    inFile ... input pdf file
   outFile ... output pdf file`

	usageChangeUserPW     = "usage: pdfcpu changeupw [-v(erbose)|vv] [-q(uiet)] [-opw ownerpw] inFile upwOld upwNew"
	usageLongChangeUserPW = `Change the user password also known as the open doc password.
	
verbose, v ... turn on logging
        vv ... verbose logging
  quiet, q ... disable output
       opw ... owner password, required unless = ""
    inFile ... input pdf file
    upwOld ... old user password
    upwNew ... new user password`

	usageChangeOwnerPW     = "usage: pdfcpu changeopw [-v(erbose)|vv] [-q(uiet)] [-upw userpw] inFile opwOld opwNew"
	usageLongChangeOwnerPW = `Change the owner password also known as the set permissions password.
	
verbose, v ... turn on logging
        vv ... verbose logging
  quiet, q ... disable output
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
   offset:           (dx dy) in user units eg. '15 20'
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

	usageStampAdd    = "pdfcpu stamp add    [-v(erbose)|vv] [-q(uiet)] [-pages selectedPages] [-upw userpw] [-opw ownerpw] -mode text|image|pdf string|file description inFile [outFile]"
	usageStampRemove = "pdfcpu stamp remove [-v(erbose)|vv] [-q(uiet)] [-pages selectedPages] [-upw userpw] [-opw ownerpw] inFile [outFile]"
	usageStampUpdate = "pdfcpu stamp update [-v(erbose)|vv] [-q(uiet)] [-pages selectedPages] [-upw userpw] [-opw ownerpw] -mode text|image|pdf string|file description inFile [outFile]"

	usageStamp = "usage: " + usageStampAdd +
		"\n       " + usageStampRemove +
		"\n       " + usageStampUpdate

	usageLongStamp = `Process stamping for selected pages. 

 verbose, v ... turn on logging
         vv ... verbose logging
   quiet, q ... disable output
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

	usageWatermarkAdd    = "pdfcpu watermark add    [-v(erbose)|vv] [-q(uiet)] [-pages selectedPages] [-upw userpw] [-opw ownerpw] -mode text|image|pdf string|file description inFile [outFile]"
	usageWatermarkRemove = "pdfcpu watermark remove [-v(erbose)|vv] [-q(uiet)] [-pages selectedPages] [-upw userpw] [-opw ownerpw] inFile [outFile]"
	usageWatermarkUpdate = "pdfcpu watermark update [-v(erbose)|vv] [-q(uiet)] [-pages selectedPages] [-upw userpw] [-opw ownerpw] -mode text|image|pdf string|file description inFile [outFile]"

	usageWatermark = "usage: " + usageWatermarkAdd +
		"\n       " + usageWatermarkRemove +
		"\n       " + usageWatermarkUpdate

	usageLongWatermark = `Process watermarking for selected pages. 

 verbose, v ... turn on logging
         vv ... verbose logging
   quiet, q ... disable output
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

	usageImportImages     = "usage: pdfcpu import [-v(erbose)|vv] [-q(uiet)] [description] outFile imageFile..."
	usageLongImportImages = `Turn image files into a PDF page sequence and write the result to outFile.
If outFile already exists the page sequence will be appended.
Each imageFile will be rendered to a separate page.
In its simplest form this converts an image into a PDF: "pdfcpu import img.pdf img.jpg"

 verbose, v ... turn on logging
         vv ... verbose logging
   quiet, q ... disable output
description ... dimensions, format, position, offset, scale factor
    outFile ... output pdf file
  imageFile ... a list of image files
  
  <description> is a comma separated configuration string containing:

  optional entries:

      (defaults: d:595 842, f:A4, pos:full, off:0 0, s:0.5 rel, dpi:72)

  dimensions: (width height) in user units eg. '400 200'

  formsize, papersize: eg. A4, Letter, Legal...
                           Please refer to "pdfcpu help paper" for a comprehensive list of defined paper sizes.
                           An appended 'L' enforces landscape mode. (eg. A3L)
                           An appended 'P' enforces portrait mode. (eg. TabloidP)

  position:    one of 'full' or the anchors: tl,tc,tr, l,c,r, bl,bc,br
  offset:      (dx dy) in user units eg. '15 20'
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

	usagePagesInsert = "pdfcpu pages insert [-v(erbose)|vv] [-q(uiet)] [-pages selectedPages] [-upw userpw] [-opw ownerpw] [-mode before|after] inFile [outFile]"
	usagePagesRemove = "pdfcpu pages remove [-v(erbose)|vv] [-q(uiet)]  -pages selectedPages  [-upw userpw] [-opw ownerpw] inFile [outFile]"

	usagePages = "usage: " + usagePagesInsert +
		"\n       " + usagePagesRemove

	usageLongPages = `Manage pages.

 verbose, v ... turn on logging
         vv ... verbose logging
   quiet, q ... disable output
      pages ... selected pages
        upw ... user password
        opw ... owner password
       mode ... before, after (default: before)
     inFile ... input pdf file
    outFile ... output pdf file

` + usagePageSelection

	usageRotate     = "usage: pdfcpu rotate [-v(erbose)|vv] [-q(uiet)] [-pages selectedPages] [-upw userpw] [-opw ownerpw] inFile rotation [outFile]"
	usageLongRotate = `Rotate selected pages by a multiple of 90 degrees. 

 verbose, v ... turn on logging
         vv ... verbose logging
   quiet, q ... disable output
      pages ... selected pages
        upw ... user password
        opw ... owner password
     inFile ... input pdf file
   rotation ... a multiple of 90 degrees for clockwise rotation
    outFile ... output pdf file

` + usagePageSelection

	usageNUp     = "usage: pdfcpu nup [-v(erbose)|vv] [-q(uiet)] [-pages selectedPages] [description] outFile n inFile|imageFiles..."
	usageLongNUp = `Rearrange existing PDF pages or images into a sequence of page grids.
This reduces the number of pages and therefore the required print time.
If the input is one imageFile a single page n-up PDF gets generated.

 verbose, v ... turn on logging
         vv ... verbose logging
   quiet, q ... disable output
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
  
        (defaults: d:595 842, f:A4, o:rd, b:on, m:3)
  
    dimensions: (width,height) in user units eg. '400 200'
    
    formsize, papersize, eg. A4, Letter, Legal...
                           Please refer to "pdfcpu help paper" for a comprehensive list of defined paper sizes.
                           Appended 'L' enforces landscape mode. (eg. A3L)
                           Appended 'P' enforces portrait mode. (eg. TabloidP)
                           Only one of dimensions or format is allowed.
    
    orientation: one of rd ... right down (=default)
                           dr ... down right
                           ld ... left down
                           dl ... down left
                           Orientation applies to PDF input files only.

    border:      on/off true/false
    
    margin:      for n-up content: int >= 0

All configuration string parameters support completion.
    
Examples: "pdfcpu nup out.pdf 4 in.pdf"
          Rearrange pages of in.pdf into 2x2 grids and write result to out.pdf using the default orientation
          and default paper size A4. in.pdf's page size will be preserved.
                                 
          "pdfcpu nup -pages=3- out.pdf 6 in.pdf"
          Rearrange selected pages of in.pdf (all pages starting with page 3) into 3x2 grids and write result to out.pdf using the default orientation
          and default paper size A4. in.pdf's page size will be preserved.

          "pdfcpu nup out.pdf 9 logo.jpg"
          Arrange instances of logo.jpg into a 3x3 grid and write result to out.pdf using the A4 default format.
          
          "pdfcpu nup 'f:Tabloid' out.pdf 4 *.jpg" 
          Rearrange all jpg files into 2x2 grids and write result to out.pdf using the Tabloid format
          and the default orientation.

` + usagePageSelection

	usageGrid     = "usage: pdfcpu grid [-v(erbose)|vv] [-q(uiet)] [-pages selectedPages] [description] outFile m n inFile|imageFiles..."
	usageLongGrid = `Rearrange PDF pages or images for enhanced browsing experience.
For a PDF inputfile each output page represents a grid of input pages.
For image inputfiles each output page shows all images laid out onto grids of given paper size. 
This command produces poster like PDF pages convenient for page and image browsing. 

 verbose, v ... turn on logging
         vv ... verbose logging
   quiet, q ... disable output
      pages ... selected pages for inFile only
description ... dimensions, format, orientation
    outFile ... output pdf file
          m ... grid columns
          n ... grid lines
     inFile ... input pdf file
 imageFiles ... input image file(s)

    <description> is a comma separated configuration string containing:

    optional entries:
  
        (defaults: d:595 842, f:A4, o:rd, b:on, m:3)
  
    dimensions: (width height) in user units eg. '400 200'

    formsize, papersize, eg. A4, Letter, Legal...
                           Please refer to "pdfcpu help paper" for a comprehensive list of defined paper sizes.
                           Appended 'L' enforces landscape mode. (eg. A3L)
                           Appended 'P' enforces portrait mode. (eg. TabloidP)
                           Only one of dimensions or format is allowed.

    orientation: one of rd ... right down (=default)
                           dr ... down right
                           ld ... left down
                           dl ... down left
                           Orientation applies to PDF input files only.

    border:      on/off true/false
    
    margin:      for content: int >= 0

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

	usageVersion     = "usage: pdfcpu version"
	usageLongVersion = "Print the pdfcpu version."

	usagePaper     = "usage: pdfcpu paper"
	usageLongPaper = "Print a list of supported paper sizes."

	usageInfo     = "usage: pdfcpu info [-u(nits)] [-upw userpw] [-opw ownerpw] inFile"
	usageLongInfo = `Print info about a PDF file.
   
units, u ... paper size display unit
     upw ... user password
     opw ... owner password
  inFile ... input pdf file
    
Possible units are:
   
points, po ... (default) points
inches, in ... inches
        cm ... centimetres
        mm ... millimetres`

	usageFontsList    = "pdfcpu fonts list"
	usageFontsInstall = "pdfcpu fonts install fontFiles..."

	usageFonts = "usage: " + usageFontsList +
		"\n       " + usageFontsInstall
	usageLongFonts = `Print a list of supported fonts (includes the 14 PDF core fonts).
Install given true type fonts (.ttf) from working directory for embedding in stamps/watermarks.`

	usageKeywordsList   = "pdfcpu keywords list    [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile"
	usageKeywordsAdd    = "pdfcpu keywords add     [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile keyword..."
	usageKeywordsRemove = "pdfcpu keywords remove  [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile [keyword...]"

	usageKeywords = "usage: " + usageKeywordsList +
		"\n       " + usageKeywordsAdd +
		"\n       " + usageKeywordsRemove

	usageLongKeywords = `Manage keywords.
	
verbose, v ... turn on logging
        vv ... verbose logging
  quiet, q ... disable output
       upw ... user password
       opw ... owner password
    inFile ... input pdf file
   keyword ... search keyword
    
    Eg. adding two keywords: 
           pdfcpu keywords add test.pdf music 'virtual instruments'
    `

	usagePropertiesList   = "pdfcpu properties list    [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile"
	usagePropertiesAdd    = "pdfcpu properties add     [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile nameValuePair..."
	usagePropertiesRemove = "pdfcpu properties remove  [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile [name...]"

	usageProperties = "usage: " + usagePropertiesList +
		"\n       " + usagePropertiesAdd +
		"\n       " + usagePropertiesRemove

	usageLongProperties = `Manage document properties.
    
   verbose, v ... turn on logging
           vv ... verbose logging
     quiet, q ... disable output
          upw ... user password
          opw ... owner password
       inFile ... input pdf file
nameValuePair ... 'name = value'
         name ... property name
     
     Eg. adding one property:   pdfcpu properties add test.pdf 'key = value'
         adding two properties: pdfcpu properties add test.pdf 'key1 = val1' 'key2 = val2'
     `
	usageCollect     = "usage: pdfcpu collect [-v(erbose)|vv] [-q(uiet)] -pages selectedPages [-upw userpw] [-opw ownerpw] inFile [outFile]"
	usageLongCollect = `Create custom sequence of selected pages. 
  
   verbose, v ... turn on logging
           vv ... verbose logging
     quiet, q ... disable output
        pages ... selected pages
          upw ... user password
          opw ... owner password
       inFile ... input pdf file
      outFile ... output pdf file
  
  ` + usagePageSelection
)
