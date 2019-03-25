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
   decrypt     remove password protection
   encrypt     set password protection		
   extract     extract images, fonts, content, pages, metadata
   grid        rearrange pages orimages for enhanced browsing experience
   import      import/convert images
   merge       concatenate 2 or more PDFs
   nup         rearrange pages or images for reduced number of pages
   optimize    optimize PDF by getting rid of redundant page resources
   pages       insert, remove selected pages
   paper       print list of supported paper sizes
   permissions list, add user access permissions
   rotate      rotate pages
   split       split multi-page PDF into several PDFs according to split span
   stamp       add text, image or PDF stamp to selected pages
   trim        create trimmed version with selected pages
   validate    validate PDF against PDF 32000-1:2008 (PDF 1.7)
   version     print version
   watermark   add text, image or PDF watermark to selected pages

   Completion supported for all commands.
   One letter Unix style abbreviations supported for flags.

Use "pdfcpu help [command]" for more information about a command.`

	usageValidate     = "usage: pdfcpu validate [-v(erbose)|vv] [-mode strict|relaxed] [-upw userpw] [-opw ownerpw] inFile"
	usageLongValidate = `Check inFile for specification compliance.

verbose, v ... turn on logging
        vv ... verbose logging
      mode ... validation mode
       upw ... user password
       opw ... owner password
    inFile ... input pdf file
		
The validation modes are:

 strict ... (default) validates against PDF 32000-1:2008 (PDF 1.7)
relaxed ... like strict but doesn't complain about common seen spec violations.`

	usageOptimize     = "usage: pdfcpu optimize [-v(erbose)|vv] [-stats csvFile] [-upw userpw] [-opw ownerpw] inFile [outFile]"
	usageLongOptimize = `Read inFile, remove redundant page resources like embedded fonts and images and write the result to outFile.

verbose, v ... turn on logging
        vv ... verbose logging
     stats ... appends a stats line to a csv file with information about the usage of root and page entries.
               useful for batch optimization and debugging PDFs.
       upw ... user password
       opw ... owner password
    inFile ... input pdf file
   outFile ... output pdf file (default: inFile-new.pdf)`

	usageSplit     = "usage: pdfcpu split [-v(erbose)|vv] [-upw userpw] [-opw ownerpw] inFile outDir [span]"
	usageLongSplit = `Generate a set of PDFs for the input file in outDir according to given span value.

verbose, v ... turn on logging
        vv ... verbose logging
       upw ... user password
       opw ... owner password
    inFile ... input pdf file
    outDir ... output directory
      span ... split span in pages (default: 1)`

	usageMerge     = "usage: pdfcpu merge [-v(erbose)|vv] outFile inFile..."
	usageLongMerge = `Concatenate a sequence of PDFs/inFiles into outFile.

verbose, v ... turn on logging
        vv ... verbose logging
   outFile ... output pdf file
   inFiles ... a list of at least 2 pdf files subject to concatenation.`

	usagePageSelection = `'-pages' selects pages for processing and is a comma separated list of expressions:

	Valid expressions are:

	even ... include even pages           odd ... include odd pages
  	   # ... include page #               #-# ... include page range
 	  !# ... exclude page #              !#-# ... exclude page range
 	  n# ... exclude page #              n#-# ... exclude page range

	  #- ... include page # - last page    -# ... include first page - page #
 	 !#- ... exclude page # - last page   !-# ... exclude first page - page #
 	 n#- ... exclude page # - last page   n-# ... exclude first page - page #

	n serves as an alternative for !, since ! needs to be escaped with single quotes on the cmd line.

e.g. -3,5,7- or 4-7,!6 or 1-,!5 or odd,n1`

	usageExtract     = "usage: pdfcpu extract [-v(erbose)|vv] -mode image|font|content|page|meta [-pages pageSelection] [-upw userpw] [-opw ownerpw] inFile outDir"
	usageLongExtract = `Export inFile's images, fonts, content or pages into outDir.

verbose, v ... turn on logging
        vv ... verbose logging
      mode ... extraction mode
     pages ... page selection
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

	usageTrim     = "usage: pdfcpu trim [-v(erbose)|vv] -pages pageSelection [-upw userpw] [-opw ownerpw] inFile [outFile]"
	usageLongTrim = `Generate a trimmed version of inFile for selected pages.

verbose, v ... turn on logging
        vv ... verbose logging
     pages ... page selection
       upw ... user password
       opw ... owner password
    inFile ... input pdf file
   outFile ... output pdf file (default: inFile-new.pdf)
   
` + usagePageSelection

	usageAttachList    = "pdfcpu attachments list [-v(erbose)|vv] [-upw userpw] [-opw ownerpw] inFile"
	usageAttachAdd     = "pdfcpu attachments add [-v(erbose)|vv] [-upw userpw] [-opw ownerpw] inFile file..."
	usageAttachRemove  = "pdfcpu attachments remove [-v(erbose)|vv] [-upw userpw] [-opw ownerpw] inFile [file...]"
	usageAttachExtract = "pdfcpu attachments extract [-v(erbose)|vv] [-upw userpw] [-opw ownerpw] inFile outDir [file...]"

	usageAttach = "usage: " + usageAttachList +
		"\n       " + usageAttachAdd +
		"\n       " + usageAttachRemove +
		"\n       " + usageAttachExtract

	usageLongAttach = `Manage embedded file attachments.
	
verbose, v ... turn on logging
        vv ... verbose logging
       upw ... user password
       opw ... owner password
    inFile ... input pdf file
    outDir ... output directory`

	usagePermList = "pdfcpu permissions list [-v(erbose)|vv] [-upw userpw] [-opw ownerpw] inFile"
	usagePermAdd  = "pdfcpu permissions add [-v(erbose)|vv] [-perm none|all] [-upw userpw] -opw ownerpw inFile"

	usagePerm = "usage: " + usagePermList +
		"\n       " + usagePermAdd

	usageLongPerm = `Manage user access permissions.
	
verbose, v ... turn on logging
        vv ... verbose logging
      perm ... user access permissions
       upw ... user password
       opw ... owner password
    inFile ... input pdf file`

	usageEncrypt     = "usage: pdfcpu encrypt [-v(erbose)|vv] [-mode rc4|aes] [-key 40|128] [perm none|all] [-upw userpw] [-opw ownerpw] inFile [outFile]"
	usageLongEncrypt = `Setup password protection based on user and owner password.

verbose, v ... turn on logging
        vv ... verbose logging
      mode ... algorithm (default=aes)
       key ... key length in bits (default=128)
      perm ... user access permissions
       upw ... user password
       opw ... owner password
    inFile ... input pdf file
   outFile ... output pdf file`

	usageDecrypt     = "usage: pdfcpu decrypt [-v(erbose)|vv] [-upw userpw] [-opw ownerpw] inFile [outFile]"
	usageLongDecrypt = `Remove password protection and reset permissions.

verbose, v ... turn on logging
        vv ... verbose logging
       upw ... user password
       opw ... owner password
    inFile ... input pdf file
   outFile ... output pdf file`

	usageChangeUserPW     = "usage: pdfcpu changeupw [-v(erbose)|vv] [-opw ownerpw] inFile upwOld upwNew"
	usageLongChangeUserPW = `Change the user password also known as the open doc password.
	
verbose, v ... turn on logging
        vv ... verbose logging
       opw ... owner password, required unless = ""
    inFile ... input pdf file
    upwOld ... old user password
    upwNew ... new user password`

	usageChangeOwnerPW     = "usage: pdfcpu changeopw [-v(erbose)|vv] [-upw userpw] inFile opwOld opwNew"
	usageLongChangeOwnerPW = `Change the owner password also known as the set permissions password.
	
verbose, v ... turn on logging
        vv ... verbose logging
       upw ... user password, required unless = ""
    inFile ... input pdf file
    opwOld ... old owner password (provide user password on initial changeopw)
    opwNew ... new owner password`

	usageWMDescription = `<description> is a comma separated configuration string containing:
	
    1st entry: the display string
               or an image file name with one the of extensions '.jpg', 'jpeg', .png', '.tif' or '.tiff' 
               or a PDF file name with extension '.pdf' followed by an optional page number (default=1) separated by ':'

    optional entries:

         (defaults: 'f:Helvetica, p:24, s:0.5 rel, c:0.5 0.5 0.5, r:0, d:1, o:1, m:0')

      f: fontname, a basefont, supported are: Helvetica, Times-Roman, Courier
      p: fontsize in points, in combination with absolute scaling only.
      s: scale factor, 0.0 <= x <= 1.0 followed by optional 'abs|rel' or 'a|r'.
      c: color: 3 fill color intensities, where 0.0 < i < 1.0, eg 1.0, 0.0 0.0 = red (default:0.5 0.5 0.5 = gray)
      r: rotation, where -180.0 <= x <= 180.0
      d: render along diagonal, 1..lower left to upper right, 2..upper left to lower right (if present overrules r!)
      o: opacity, where 0.0 <= x <= 1.0
      m: render mode: 0 ... fill
                      1 ... stroke
                      2 ... fill & stroke

    Only one of rotation and diagonal is allowed.

e.g. 'Draft'                                                  'logo.png'
     'Draft, d:2'                                             'logo.tif, o:0.5, s:0.5 abs, r:0'
     'Intentionally left blank, s:.75 abs, p:48'              'some.pdf, r:45' 
     'Confidental, f:Courier, s:0.75, c: 0.5 0.0 0.0, r:20'   'some.pdf:3, r:-90, s:0.75'
     
` + usagePageSelection

	usageStamp     = "usage: pdfcpu stamp [-v(erbose)|vv] [-pages pageSelection] [-upw userpw] [-opw ownerpw] description inFile [outFile]"
	usageLongStamp = `Add stamps for selected pages. 

 verbose, v ... turn on logging
         vv ... verbose logging
      pages ... page selection
        upw ... user password
        opw ... owner password
description ... font, font size, text, color, image/pdf file name, pdf page#, rotation, opacity, scale factor, render mode
     inFile ... input pdf file
    outFile ... output pdf file (default: inFile-new.pdf)

` + usageWMDescription

	usageWatermark     = "usage: pdfcpu watermark [-v(erbose)|vv] [-pages pageSelection] [-upw userpw] [-opw ownerpw] description inFile [outFile]"
	usageLongWatermark = `Add watermarks for selected pages. 

 verbose, v ... turn on logging
         vv ... verbose logging
      pages ... page selection
        upw ... user password
        opw ... owner password
description ... font, font size, text, color, image/pdf file name, pdf page#, rotation, opacity, scale factor, render mode
     inFile ... input pdf file
    outFile ... output pdf file (default: inFile-new.pdf)

` + usageWMDescription

	usageImportImages     = "usage: pdfcpu import [-v(erbose)|vv] [description] outFile imageFile..."
	usageLongImportImages = `Turn image files into a PDF page sequence and write the result to outFile.
If outFile already exists the page sequence will be appended.
Each imageFile will be rendered to a separate page.
In its simplest form this converts an image into a PDF: "pdfcpu import img.pdf img.jpg"

 verbose, v ... turn on logging
         vv ... verbose logging
description ... dimensions, format, position, offset, scale factor
    outFile ... output pdf file
  imageFile ... a list of image files
  
  <description> is a comma separated configuration string containing:

  optional entries:

      (defaults: d:595 842, f:A4, p:full, o:0 0, s:0.5 rel)

  d: dimensions (width,height) in user units eg. '400 200'

  f: form/paper size, eg. A4, Letter, Legal...
                           Please refer to "pdfcpu help paper" for a comprehensive list of defined paper sizes.
                           An appended 'L' enforces landscape mode. (eg. A3L)
                           An appended 'P' enforces portrait mode. (eg. TabloidP)

  p: position: one of 'full' or the anchors: tl,tc,tr, l,c,r, bl,bc,br
  o: offset (dx,dy) in user units eg. 15,20
  s: scale factor, 0.0 <= x <= 1.0 followed by optional 'abs|rel' or 'a|r'
  
  Only one of dimensions or format is allowed.
  position: full => image dimensions equal page dimensions.
  
  e.g. 'f:A5, p:c                            ... render the image centered on A5 with relative scaling 0.5.'
       'd:300 600, p:bl, o:20 20, s:1.0 abs' ... render the image anchored to bottom left corner with offset 20,20 and abs. scaling 1.0.
       'p:full'                              ... render the image to a page with corresponding dimensions.`

	usagePagesInsert = "pdfcpu pages insert [-v(erbose)|vv] [-pages pageSelection] [-upw userpw] [-opw ownerpw] inFile [outFile]"
	usagePagesRemove = "pdfcpu pages remove [-v(erbose)|vv]  -pages pageSelection  [-upw userpw] [-opw ownerpw] inFile [outFile]"

	usagePages = "usage: " + usagePagesInsert +
		"\n       " + usagePagesRemove

	usageLongPages = `Manage pages.

 verbose, v ... turn on logging
         vv ... verbose logging
      pages ... page selection
        upw ... user password
        opw ... owner password
     inFile ... input pdf file
    outFile ... output pdf file

` + usagePageSelection

	usageRotate     = "usage: pdfcpu rotate [-v(erbose)|vv] [-pages pageSelection] [-upw userpw] [-opw ownerpw] inFile rotation"
	usageLongRotate = `Rotate selected pages by a multiple of 90 degrees. 

 verbose, v ... turn on logging
         vv ... verbose logging
      pages ... page selection
        upw ... user password
        opw ... owner password
     inFile ... input pdf file
   rotation ... a multiple of 90 degrees for clockwise rotation.

` + usagePageSelection

	usageNUp     = "usage: pdfcpu nup [-v(erbose)|vv] [-pages pageSelection] [description] outFile n inFile|imageFiles..."
	usageLongNUp = `Rearrange existing PDF pages or images into a sequence of page grids.
This reduces the number of pages and therefore the required print time.
If the input is one imageFile a single page n-up PDF gets generated.

 verbose, v ... turn on logging
         vv ... verbose logging
      pages ... page selection for inFile only
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
  
    d: dimensions (width,height) in user units eg. '400 200'
    
    f: form/paper size, eg. A4, Letter, Legal...
                           Please refer to "pdfcpu help paper" for a comprehensive list of defined paper sizes.
                           Appended 'L' enforces landscape mode. (eg. A3L)
                           Appended 'P' enforces portrait mode. (eg. TabloidP)
                           Only one of dimensions or format is allowed.
    
    o: orientation, one of rd ... right down (=default)
                           dr ... down right
                           ld ... left down
                           dl ... down left
                           Orientation applies to PDF input files only.

    b: draw border ... on/off true/false
    
    m: margin for n-up content: int >= 0
    
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

	usageGrid     = "usage: pdfcpu grid [-v(erbose)|vv] [-pages pageSelection] [description] outFile m n inFile|imageFiles..."
	usageLongGrid = `Rearrange PDF pages or images for enhanced browsing experience.
For a PDF inputfile each output page represents a grid of input pages.
For image inputfiles each output page shows all images laid out onto grids of given paper size. 
This command produces poster like PDF pages convenient for page and image browsing. 

 verbose, v ... turn on logging
         vv ... verbose logging
      pages ... page selection for inFile only
description ... dimensions, format, orientation
    outFile ... output pdf file
          m ... grid columns
          n ... grid lines
     inFile ... input pdf file
 imageFiles ... input image file(s)

    <description> is a comma separated configuration string containing:

    optional entries:
  
        (defaults: d:595 842, f:A4, o:rd, b:on, m:3)
  
    d: dimensions (width,height) in user units eg. '400 200'

    f: form/paper size, eg. A4, Letter, Legal...
                           Please refer to "pdfcpu help paper" for a comprehensive list of defined paper sizes.
                           Appended 'L' enforces landscape mode. (eg. A3L)
                           Appended 'P' enforces portrait mode. (eg. TabloidP)
                           Only one of dimensions or format is allowed.

    o: orientation, one of rd ... right down (=default)
                           dr ... down right
                           ld ... left down
                           dl ... down left
                           Orientation applies to PDF input files only.

    b: draw border ... on/off true/false
    
    m: margin for content: int >= 0

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
	usageLongVersion = "prints the pdfcpu version"

	usagePaper     = "usage: pdfcpu paper"
	usageLongPaper = "prints a list of supported paper sizes"
)
