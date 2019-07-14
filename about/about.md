---
layout: default
---

# About

`pdfcpu` is a PDF processor written in Go supporting encryption. Over time `pdfcpu` aims to support the standard range of PDF processing features and interesting use cases that may present themselves along the way.

## `pdfcpu` the command line tool

Use shell scripts using the `pdfcpu` CLI to build your PDF processing pipelines for batch processing. You can use `pdfcpu` to manipulate your PDF files on the command line of all major platforms:

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

## `pdfcpu` the Go library
Use the `pdfcpu` API to integrate PDF processing into your Go based backend systems.

Each operation is available file based (also used by pdfcpu's CLI):
```
func OptimizeFile(inFile, outFile string, conf *pdf.Configuration) (err error)
```

and interface based (typically using io.ReadSeeker/io.Writer):

```
func Optimize(rs io.ReadSeeker, w io.Writer, conf *pdf.Configuration) error
```
Learn more about the API including examples for all operations at [GoDoc](https://godoc.org/github.com/hhrutter/pdfcpu/pkg/api).
