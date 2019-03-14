---
layout: default
---

# Usage

Use `pdfcpu` for a rundown of all commands:

```
pdfcpu is a tool for PDF manipulation written in Go.

Usage:

	pdfcpu command [arguments]

The commands are:

   validate    validate PDF against PDF 32000-1:2008 (PDF 1.7)
   optimize    optimize PDF by getting rid of redundant page resources
   split       split multi-page PDF into several PDFs according to split span
   merge       concatenate 2 or more PDFs
   extract     extract images, fonts, content, pages, metadata
   trim        create trimmed version
   stamp       add stamps
   watermark   add watermarks
   import      convert/import images to PDF
   nup         rearrange pages/images into grid page layout for reduced number of pages
   grid        rearrange pages/images into grid page layout for enhanced browsing experience
   rotate      rotate pages
   attach      list, add, remove, extract embedded file attachments
   perm        list, add user access permissions
   encrypt     set password protection
   decrypt     remove password protection
   changeupw   change user password
   changeopw   change owner password
   version     print version

   Single-letter Unix-style supported for commands and flags.

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

## Extraction

Extract components and resources:
```sh
pdfcpu extract [-verbose] -mode image|font|content|page|meta [-pages pageSelection] [-upw userpw] [-opw ownerpw] inFile outDir
```

<br>

## Attachments

Manage your PDF attachments:
```sh
pdfcpu attach list [-verbose] [-upw userpw] [-opw ownerpw] inFile
pdfcpu attach add [-verbose] [-upw userpw] [-opw ownerpw] inFile file...
pdfcpu attach remove [-verbose] [-upw userpw] [-opw ownerpw] inFile [file...]
pdfcpu attach extract [-verbose] [-upw userpw] [-opw ownerpw] inFile outDir [file...]
```

<br>

## Encryption

Secure your PDFs:
```sh
pdfcpu encrypt [-verbose] [-mode rc4|aes] [-key 40|128] [-perm none|all] [-upw userpw] [-opw ownerpw] inFile [outFile]
pdfcpu decrypt [-verbose] [-upw userpw] [-opw ownerpw] inFile [outFile]
pdfcpu changeopw [-verbose] [-upw userpw] inFile opwOld opwNew
pdfcpu changeupw [-verbose] [-opw ownerpw] inFile upwOld upwNew
pdfcpu perm add [-verbose] [-perm none|all] [-upw userpw] -opw ownerpw inFile
pdfcpu perm list [-verbose] [-upw userpw] [-opw ownerpw] inFile
```

<br>

## Other

### Print Supported Papersizes

```sh
pdfcpu help paper
```

<br>

### Print Version

```sh
pdfcpu version
```