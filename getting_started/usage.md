---
layout: default
---

# Usage

Use `pdfcpu` for a rundown of all commands:

```
$ pdfcpu
pdfcpu is a tool for PDF manipulation written in Go.

Usage:

   pdfcpu command [arguments]

The commands are:

   annotations list, remove page annotations
   attachments list, add, remove, extract embedded file attachments
   booklet     arrange pages onto larger sheets of paper to make a booklet or zine
   bookmarks   list, import, export, remove bookmarks
   boxes       list, add, remove page boundaries for selected pages
   changeopw   change owner password
   changeupw   change user password
   collect     create custom PDF page sequence
   config      print configuration
   create      create PDF content via JSON
   crop        set crop box for selected pages
   cut         custom cut pages horizontally or vertically
   decrypt     remove password protection
   encrypt     set password protection
   extract     extract images, fonts, content, pages, metadata
   fonts       install, list supported fonts
   form        list, remove fields, lock, unlock, reset, export, fill form via JSON or CSV
   grid        rearrange pages or images for enhanced browsing experience
   images      list, extract, update images for selected pages
   import      import/convert images to PDF
   info        print file info
   keywords    list, add, remove document keywords
   merge       concatenate 2 or more PDFs
   ndown       cut selected pages into n pages symmetrically
   nup         rearrange pages or images for reduced number of pages
   optimize    optimize PDF by getting rid of redundant page resources
   pagelayout  list, set, reset page layout for opened document
   pagemode    list, set, reset page mode for opened document
   pages       insert, remove selected pages
   paper       print list of supported paper sizes
   permissions list, set user access permissions
   portfolio   list, add, remove, extract portfolio entries
   poster      cut selected pages into poster using paper size or dimensions
   properties  list, add, remove document properties
   resize      resize selected pages
   rotate      rotate selected pages
   selectedpag print definition of the -pages flag
   split       split multi-page PDF into several PDFs according to split span
   stamp       add, update, remove text, image or PDF stamps to selected pages
   trim        create trimmed version of selected pages
   validate    validate PDF against PDF 32000-1:2008 (PDF 1.7) + basic validation version PDF 2.0
   version     print version
   viewerpref  list, set, reset viewer preferences for opened document
   watermark   add, update, remove text, image or PDF watermarks to selected pages
   zoom        zoom in/out of selected pages by magnification factor or corresponding margin

   Completion supported for all commands.
   One letter Unix style abbreviations supported for flags.

Use "pdfcpu help [command]" for more information about a command.
```

<br>

## [Core Commands](../core/core.md)

The basic processing features:
```
pdfcpu collect   -p(ages) selectedPages inFile [outFile]
pdfcpu crop     [-p(ages) selectedPages] -- description inFile [outFile]
pdfcpu merge    [-m(ode) create|append|zip] [ -s(ort) -b(ookmarks) -d(ivider)] outFile inFile...
pdfcpu optimize [-stats csvFile] inFile [outFile]
pdfcpu resize   [-p(ages) selectedPages] -- description inFile [outFile]
pdfcpu rotate   [-p(ages) selectedPages] inFile rotation [outFile]
pdfcpu split    [-m(ode) span|bookmark|page] inFile outDir [span|pageNr...]
pdfcpu trim      -p(ages) selectedPages inFile [outFile]
pdfcpu validate [-m(ode) strict|relaxed] [-l(inks)] inFile...
pdfcpu zoom     [-p(ages) selectedPages] -- description inFile [outFile]
```

<br>

## [Stamps](../core/stamp.md)

Manage your stamps:
```
pdfcpu stamp add    [-p(ages) selectedPages] -m(ode) text|image|pdf -- string|file description inFile [outFile]
pdfcpu stamp update [-p(ages) selectedPages] -m(ode) text|image|pdf -- string|file description inFile [outFile]
pdfcpu stamp remove [-p(ages) selectedPages] inFile [outFile]
```

<br>

## [Watermarks](../core/watermark.md)

Manage your watermarks:
```
pdfcpu watermark add    [-p(ages) selectedPages] -m(ode) text|image|pdf -- string|file description inFile [outFile]
pdfcpu watermark update [-p(ages) selectedPages] -m(ode) text|image|pdf -- string|file description inFile [outFile]
pdfcpu watermark remove [-p(ages) selectedPages] inFile [outFile]
```

<br>


## [Forms](../form/form.md)

Manage your PDF forms:

```
pdfcpu form list   inFile...
pdfcpu form remove inFile [outFile] <fieldID|fieldName>...
pdfcpu form lock   inFile [outFile] [fieldID|fieldName]...
pdfcpu form unlock inFile [outFile] [fieldID|fieldName]...
pdfcpu form reset  inFile [outFile] [fieldID|fieldName]...
pdfcpu form export inFile [outFileJSON]
```
<br>

## [Fonts](../fonts/fonts.md)

Manage core fonts and your user fonts:

```
pdfcpu fonts list
pdfcpu fonts install fontFiles...
pdfcpu fonts cheatsheet fontFiles...
```

<br>

## [Generate Commands](../generate/generate.md)

```
pdfcpu booklet [-p(ages) selectedPages] -- [description] outFile n inFile|imageFiles...
pdfcpu create  inFileJSON [inFile] outFile
pdfcpu cut     [-p(ages) selectedPages] -- description inFile outDir [outFileName]
pdfcpu grid    [-p(ages) selectedPages] -- [description] outFile m n inFile|imageFiles...
pdfcpu import                           -- [description] outFile imageFile...
pdfcpu ndown    -p(ages) selectedPages] -- [description] n inFile outDir [outFileName]
pdfcpu nup     [-p(ages) selectedPages] -- [description] outFile n inFile|imageFiles...
pdfcpu poster  [-p(ages) selectedPages] -- description inFile outDir [outFileName]
```

<br>

## [Pages](../pages/pages.md)

Insert or remove pages:
```
pdfcpu pages insert [-p(ages) selectedPages] [-m(ode) before|after] inFile [outFile]
pdfcpu pages remove  -p(ages) selectedPages inFile [outFile]
```

<br>

## [Extract](../extract/extract.md)

Extract components and resources:
```
pdfcpu extract -m(ode) image|font|content|page|meta [-p(ages) selectedPages] inFile outDir
```

<br>

## [Attachments](../attach/attach.md)

Manage your attachments:
```
pdfcpu attachments list    inFile
pdfcpu attachments add     inFile file...
pdfcpu attachments remove  inFile [file...]
pdfcpu attachments extract inFile outDir [file...]
```

<br>

## [Portfolio](../portfolio/portfolio.md)

Manage your portfolios:
```
pdfcpu portfolio list    inFile
pdfcpu portfolio add     inFile file[,desc]...
pdfcpu portfolio remove  inFile [file...]
pdfcpu portfolio extract inFile outDir [file...]
```

<br>

## [Annotations](../annot/annot.md)

Manage your annotations:
```
pdfcpu annotations list   [-p(ages) selectedPages] inFile
pdfcpu annotations remove [-p(ages) selectedPages] inFile [outFile] [objNr|annotId|annotType]...
```

<br>

## [Images](../images/images.md)

Manage your images:
```
pdfcpu images list [-p(ages) selectedPages] inFile...
pdfcpu images extract [-p(ages) selectedPages] -- inFile outDir
pdfcpu images update inFile imageFile [outFile] [ objNr | (pageNr Id) ]
```

<br>

## [Encryption](../encrypt/encrypt.md)

Secure your PDFs:
```
pdfcpu encrypt [-m(ode) rc4|aes] [-key 40|128|256] [-perm none|all] [-upw userpw] -opw ownerpw inFile [outFile]
pdfcpu decrypt [-upw userpw] [-opw ownerpw] inFile [outFile]
pdfcpu changeopw [-upw userpw] inFile opwOld opwNew
pdfcpu changeupw [-opw ownerpw] inFile upwOld upwNew
pdfcpu permissions list [-upw userpw] [-opw ownerpw] inFile
pdfcpu permissions set [-perm none|all] [-upw userpw] -opw ownerpw inFile
```

<br>

## [Print Supported Papersizes](../paper.md)

```
pdfcpu paper
```
<br>

## [Keywords](../keywords/keywords.md)

Manage your keywords for searching:
```
pdfcpu keywords list    inFile
pdfcpu keywords add     inFile keyword...
pdfcpu keywords remove  inFile [keyword...]
```

<br>

## [Properties](../properties/properties.md)

Manage your document properties:
```
pdfcpu properties list    inFile
pdfcpu properties add     inFile nameValuePair...
pdfcpu properties remove  inFile [name...]
```

<br>

## [Page Layout](../pagelayout/pagelayout.md)

Manage the page layout for your opened document:
```
pdfcpu pagelayout list  inFile
pdfcpu pagelayout set   inFile value
pdfcpu pagelayout reset inFile
```

<br>

## [Page Mode](../pagemode/pagemode.md)

Manage the page mode for your opened document:
```
pdfcpu pagemode list  inFile
pdfcpu pagemode set   inFile value
pdfcpu pagemode reset inFile
```

<br>

## [Viewer Preferences](../viewerpref/viewerpref.md)

Manage the viewer preferences for your opened document:
```
pdfcpu viewerpref list [-a(ll)] [-j(son)] inFile
pdfcpu viewerpref set                     inFile (inFileJSON | JSONstring)
pdfcpu viewerpref reset                   inFile
```

<br>

## [Bookmarks](../bookmarks/bookmarks.md)

Manage your bookmarks:
```
pdfcpu bookmarks list inFile
pdfcpu bookmarks import [-r(eplace)] inFile inFileJSON [outFile]
pdfcpu bookmarks export inFile [outFileJSON]
pdfcpu bookmarks remove inFile [outFile]
```

<br>

## [Boxes](../boxes/boxes.md)

Manage your page boundaries:
```
pdfcpu boxes list    [-p(ages) selectedPages] -- [boxTypes] inFile
pdfcpu boxes add     [-p(ages) selectedPages] -- description inFile [outFile]
pdfcpu boxes remove  [-p(ages) selectedPages] -- boxTypes inFile [outFile]
```

<br>

## [Info](../info.md)

Print file details:
```
pdfcpu info [-p(ages) selectedPages] [-j(son)] inFile...
```

<br>

## [Print definition of the -pages flag](../getting_started/page_selection.md)

```
pdfcpu selectedpages
```

<br>

## Print Version

```
pdfcpu version
```

