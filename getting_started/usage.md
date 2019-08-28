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
   changeopw   change owner password
   changeupw   change user password
   decrypt     remove password protection
   encrypt     set password protection
   extract     extract images, fonts, content, pages, metadata
   grid        rearrange pages or images for enhanced browsing experience
   import      import/convert images to PDF
   info        print file info
   merge       concatenate 2 or more PDFs
   nup         rearrange pages or images for reduced number of pages
   optimize    optimize PDF by getting rid of redundant page resources
   pages       insert, remove selected pages
   paper       print list of supported paper sizes
   permissions list, set user access permissions
   rotate      rotate pages
   split       split multi-page PDF into several PDFs according to split span
   stamp       add text, image or PDF stamp to selected pages
   trim        create trimmed version of selected pages
   validate    validate PDF against PDF 32000-1:2008 (PDF 1.7)
   version     print version
   watermark   add text, image or PDF watermark to selected pages

   Completion supported for all commands.
   One letter Unix style abbreviations supported for flags.

Use "pdfcpu help [command]" for more information about a command.
```

<br>

## Core Commands

Choose on of the basic processing features:
```sh
pdfcpu validate [-verbose] [-mode strict|relaxed] [-upw userpw] [-opw ownerpw] inFile
pdfcpu optimize [-verbose] [-stats csvFile] [-upw userpw] [-opw ownerpw] inFile [outFile]
pdfcpu merge [-verbose] outFile inFile...
pdfcpu split [-verbose] [-upw userpw] [-opw ownerpw] inFile outDir [span]
pdfcpu trim [-verbose] -pages pageSelection [-upw userpw] [-opw ownerpw] inFile outFile
pdfcpu rotate [-v(erbose)|vv] [-pages pageSelection] inFile rotation
pdfcpu stamp [-verbose] -pages pageSelection description inFile [outFile]
pdfcpu watermark [-verbose] -pages pageSelection description inFile [outFile]
pdfcpu nup [-v(erbose)|vv] [-pages pageSelection] [description] outFile n inFile|imageFiles...
pdfcpu grid [-v(erbose)|vv] [-pages pageSelection] [description] outFile m n inFile|imageFiles...
```

<br>

## Generate Commands

Convert images (jpg, png, tiff) into PDF:
```sh
pdfcpu import [-v(erbose)|vv] [description] outFile imageFile...
```

<br>

## Pages

Insert and remove pages:
```sh
pdfcpu pages insert [-v(erbose)|vv] [-pages pageSelection] [-upw userpw] [-opw ownerpw] inFile [outFile]
pdfcpu pages remove [-v(erbose)|vv]  -pages pageSelection  [-upw userpw] [-opw ownerpw] inFile [outFile]
```

<br>

## Extraction

Extract components and resources:
```sh
pdfcpu extract [-verbose] -mode image|font|content|page|meta [-pages pageSelection] [-upw userpw] [-opw ownerpw] inFile outDir
```

<br>

## Attachments

Manage your PDF attachments:
```sh
pdfcpu attachments list [-verbose] [-upw userpw] [-opw ownerpw] inFile
pdfcpu attachments add [-verbose] [-upw userpw] [-opw ownerpw] inFile file...
pdfcpu attachments remove [-verbose] [-upw userpw] [-opw ownerpw] inFile [file...]
pdfcpu attachments extract [-verbose] [-upw userpw] [-opw ownerpw] inFile outDir [file...]
```

<br>

## Encryption

Secure your PDFs:
```sh
pdfcpu encrypt [-verbose] [-mode rc4|aes] [-key 40|128|256] [-perm none|all] [-upw userpw] -opw ownerpw inFile [outFile]
pdfcpu decrypt [-verbose] [-upw userpw] [-opw ownerpw] inFile [outFile]
pdfcpu changeopw [-verbose] [-upw userpw] inFile opwOld opwNew
pdfcpu changeupw [-verbose] [-opw ownerpw] inFile upwOld upwNew
pdfcpu permissions add [-verbose] [-perm none|all] [-upw userpw] -opw ownerpw inFile
pdfcpu permissions list [-verbose] [-upw userpw] [-opw ownerpw] inFile
```

<br>

## Other

### Print Supported Papersizes

```sh
pdfcpu paper
```

<br>

### Print Version

```sh
pdfcpu version
```

<br>

### Print PDF Info

```sh
pdfcpu info [-u(nits)] [-upw userpw] [-opw ownerpw] inFile
```
