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

Choose on of the basic processing features:
```
pdfcpu validate [-v(erbose)|vv] [-q(uiet)] [-mode strict|relaxed] [-upw userpw] [-opw ownerpw] inFile
pdfcpu optimize [-v(erbose)|vv] [-q(uiet)] [-stats csvFile] [-upw userpw] [-opw ownerpw] inFile [outFile]
pdfcpu merge [-v(erbose)|vv] [-q(uiet)] [-mode create|append] outFile inFile...
pdfcpu split [-v(erbose)|vv] [-q(uiet)] [-mode span|bookmark] [-upw userpw] [-opw ownerpw] inFile outDir [span]
pdfcpu trim [-v(erbose)|vv] [-q(uiet)] -pages selectedPages [-upw userpw] [-opw ownerpw] inFile [outFile]
pdfcpu rotate [-v(erbose)|vv] [-q(uiet)] [-pages selectedPages] [-upw userpw] [-opw ownerpw] inFile rotation [outFile]
pdfcpu nup [-v(erbose)|vv] [-q(uiet)] [-pages selectedPages] [description] outFile n inFile|imageFiles..
pdfcpu grid [-v(erbose)|vv] [-q(uiet)] [-pages selectedPages] [description] outFile m n inFile|imageFiles...
```

<br>

## Generate Commands

Convert images (jpg, png, tiff) into PDF:
```
pdfcpu import [-v(erbose)|vv] [-q(uiet)] [description] outFile imageFile...
```

<br>

## Stamps

Manage stamps for selected pages:
```
pdfcpu stamp add    [-v(erbose)|vv] [-q(uiet)] [-pages selectedPages] [-upw userpw] [-opw ownerpw] -mode text|image|pdf string|file description inFile [outFile]
pdfcpu stamp remove [-v(erbose)|vv] [-q(uiet)] [-pages selectedPages] [-upw userpw] [-opw ownerpw] inFile [outFile]
pdfcpu stamp update [-v(erbose)|vv] [-q(uiet)] [-pages selectedPages] [-upw userpw] [-opw ownerpw] -mode text|image|pdf string|file description inFile [outFile]
```

<br>

## Watermarks

Manage watermarks for selected pages:
```
pdfcpu watermark add    [-v(erbose)|vv] [-q(uiet)] [-pages selectedPages] [-upw userpw] [-opw ownerpw] -mode text|image|pdf string|file description inFile [outFile]
pdfcpu watermark remove [-v(erbose)|vv] [-q(uiet)] [-pages selectedPages] [-upw userpw] [-opw ownerpw] inFile [outFile]
pdfcpu watermark update [-v(erbose)|vv] [-q(uiet)] [-pages selectedPages] [-upw userpw] [-opw ownerpw] -mode text|image|pdf string|file description inFile [outFile]
```

<br>

## Pages

Insert and remove pages:
```
pdfcpu pages insert [-v(erbose)|vv] [-q(uiet)] [-pages selectedPages] [-upw userpw] [-opw ownerpw] [-mode before|after] inFile [outFile]
pdfcpu pages remove [-v(erbose)|vv] [-q(uiet)]  -pages selectedPages  [-upw userpw] [-opw ownerpw] inFile [outFile]
```

<br>

## Extraction

Extract components and resources:
```
pdfcpu extract [-v(erbose)|vv] [-q(uiet)] -mode image|font|content|page|meta [-pages selectedPages] [-upw userpw] [-opw ownerpw] inFile outDir
```

<br>

## Attachments

Manage your PDF attachments:
```
pdfcpu attachments list    [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile
pdfcpu attachments add     [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile file...
pdfcpu attachments remove  [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile [file...]
pdfcpu attachments extract [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile outDir [file...]
```

<br>

## Encryption

Secure your PDFs:
```
pdfcpu encrypt [-v(erbose)|vv] [-q(uiet)] [-mode rc4|aes] [-key 40|128|256] [perm none|all] [-upw userpw] -opw ownerpw inFile [outFile]
pdfcpu decrypt [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile [outFile]
pdfcpu changeopw [-v(erbose)|vv] [-q(uiet)] [-upw userpw] inFile opwOld opwNew
pdfcpu changeupw [-v(erbose)|vv] [-q(uiet)] [-opw ownerpw] inFile upwOld upwNew
pdfcpu permissions list [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile
pdfcpu permissions set [-v(erbose)|vv] [-q(uiet)] [-perm none|all] [-upw userpw] -opw ownerpw inFile
```

<br>

## Other

### Print Supported Papersizes

```
pdfcpu paper
```

<br>

### Print Version

```
pdfcpu version
```

<br>

### Print PDF Info

```
pdfcpu info [-u(nits)] [-upw userpw] [-opw ownerpw] inFile
```

<br>

### Print List of Supported Fonts

```
pdfcpu fonts
```

<br>

### Keywords

Manage your keywords for searching:
```
pdfcpu keywords list    [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile
pdfcpu keywords add     [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile keyword...
pdfcpu keywords remove  [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile [keyword...]
```

<br>

### Properties

Manage your document properties:
```
pdfcpu properties list    [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile
pdfcpu properties add     [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile nameValuePair...
pdfcpu properties remove  [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile [name...]
```
