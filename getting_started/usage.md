---
layout: default
---

# Usage

Use `pdfcpu` for a rundown of all commands:

```
Go-> pdfcpu
pdfcpu is a tool for PDF manipulation written in Go.

Usage:

   pdfcpu command [arguments]

The commands are:

   attachments list, add, remove, extract embedded file attachments
   booklet     arrange pages onto larger sheets of paper to make a booklet or zine
   boxes       list, add, remove page boundaries for selected pages
   changeopw   change owner password
   changeupw   change user password
   collect     create custom PDF page sequence
   crop        set crop box for selected pages
   decrypt     remove password protection
   encrypt     set password protection
   extract     extract images, fonts, content, pages, metadata
   fonts       install, list supported fonts
   grid        rearrange pages or images for enhanced browsing experience
   import      import/convert images to PDF
   info        print file info
   keywords    list, add, remove document keywords
   merge       concatenate 2 or more PDFs
   nup         rearrange pages or images for reduced number of pages
   optimize    optimize PDF by getting rid of redundant page resources
   pages       insert, remove selected pages
   paper       print list of supported paper sizes
   permissions list, set user access permissions
   portfolio   list, add, remove, extract portfolio entries
   properties  list, add, remove document properties
   rotate      rotate pages
   split       split multi-page PDF into several PDFs according to split span
   stamp       add, update, remove text, image or PDF stamps to selected pages
   trim        create trimmed version of selected pages
   validate    validate PDF against PDF 32000-1:2008 (PDF 1.7)
   version     print version
   watermark   add, update, remove text, image or PDF watermarks to selected pages

   Completion supported for all commands.
   One letter Unix style abbreviations supported for flags.

Use "pdfcpu help [command]" for more information about a command.
```

<br>

## Core Commands

The basic processing features:
```
pdfcpu validate [-m(ode) strict|relaxed] inFile
pdfcpu optimize [-stats csvFile] inFile [outFile]
pdfcpu merge    [-m(ode) create|append] [-sort] outFile inFile...
pdfcpu split    [-m(ode) span|bookmark] inFile outDir [span]
pdfcpu crop     [-p(ages) selectedPages] -- description inFile [outFile]
pdfcpu rotate   [-p(ages) selectedPages] inFile rotation [outFile]
pdfcpu trim      -p(ages) selectedPages inFile [outFile]
pdfcpu collect   -p(ages) selectedPages inFile [outFile]

```

## Stamps

Manage stamps:
```
pdfcpu stamp add    [-p(ages) selectedPages] -m(ode) text|image|pdf -- string|file description inFile [outFile]
pdfcpu stamp update [-p(ages) selectedPages] -m(ode) text|image|pdf -- string|file description inFile [outFile]
pdfcpu stamp remove [-p(ages) selectedPages] inFile [outFile]
```

<br>

## Watermarks

Manage watermarks:
```
pdfcpu watermark add    [-p(ages) selectedPages] -m(ode) text|image|pdf -- string|file description inFile [outFile]
pdfcpu watermark update [-p(ages) selectedPages] -m(ode) text|image|pdf -- string|file description inFile [outFile]
pdfcpu watermark remove [-p(ages) selectedPages] inFile [outFile]
```

<br>

## Fonts

Manage core fonts and your user fonts:

```
pdfcpu fonts list
pdfcpu fonts install fontFiles...
pdfcpu fonts cheatsheet fontFiles...
```

<br>

## Generate Commands

```
pdfcpu booklet [-p(ages) selectedPages] -- [description] outFile n inFile|imageFiles...
pdfcpu grid    [-p(ages) selectedPages] -- [description] outFile m n inFile|imageFiles...
pdfcpu import                           -- [description] outFile imageFile...
pdfcpu nup     [-p(ages) selectedPages] -- [description] outFile n inFile|imageFiles...
```

<br>

## Pages

Insert or remove pages:
```
pdfcpu pages insert [-p(ages) selectedPages] [-m(ode) before|after] inFile [outFile]
pdfcpu pages remove  -p(ages) selectedPages inFile [outFile]
```

<br>

## Extraction

Extract components and resources:
```
pdfcpu extract -m(ode) image|font|content|page|meta [-p(ages) selectedPages] inFile outDir
```

<br>

## Attachments

Manage your attachments:
```
pdfcpu attachments list    inFile
pdfcpu attachments add     inFile file...
pdfcpu attachments remove  inFile [file...]
pdfcpu attachments extract inFile outDir [file...]
```

<br>

## Portfolio

Manage your portfolios:
```
pdfcpu portfolio list    inFile
pdfcpu portfolio add     inFile file[,desc]...
pdfcpu portfolio remove  inFile [file...]
pdfcpu portfolio extract inFile outDir [file...]
```

<br>

## Encryption

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

## Print Supported Papersizes

```
pdfcpu paper
```
<br>

## Keywords

Manage your keywords for searching:
```
pdfcpu keywords list    inFile
pdfcpu keywords add     inFile keyword...
pdfcpu keywords remove  inFile [keyword...]
```

<br>

## Properties

Manage your document properties:
```
pdfcpu properties list    inFile
pdfcpu properties add     inFile nameValuePair...
pdfcpu properties remove  inFile [name...]
```

<br>

## Boxes

Manage your page boundaries:
```
pdfcpu boxes list    [-p(ages) selectedPages] -- [boxTypes] inFile
pdfcpu boxes add     [-p(ages) selectedPages] -- description inFile [outFile]
pdfcpu boxes remove  [-p(ages) selectedPages] -- boxTypes inFile [outFile]
```

<br>

## Info

Print file details:
```
pdfcpu info [-p(ages) selectedPages] inFile
```

<br>

## Print Version

```
pdfcpu version
```

