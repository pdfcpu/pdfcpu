---
layout: default
title: "Extract Pages"
---

# Extract Pages

This command is similar to [split](/core/split) with a default `span` of 1.

Use `-` as `outDir` with page mode and a single selected page to write the extracted page PDF to stdout.

## Examples

Extract all pages from `book.pdf` into single page PDFs in the current directory:
```sh
$ pdfcpu extract book.pdf . --mode page
extracting pages from book.pdf into . ...
writing ./book_1.pdf ...
writing ./book_7.pdf ...
writing ./book_11.pdf ...
writing ./book_13.pdf ...
writing ./book_16.pdf ...
```

<br>

Extracting pages 5-10 from `book.pdf` into `out`:

```sh
$ pdfcpu extract book.pdf out --mode page --pages 5-10 
pageSelection: 5-10
extracting pages from book.pdf into out ...
writing out/book_6.pdf ...
writing out/book_7.pdf ...
writing out/book_8.pdf ...
writing out/book_9.pdf ...
writing out/book_10.pdf ...
writing out/book_5.pdf ...
```

<br>

Extract a single page from an S3 PDF and upload the result:

```sh
$ aws s3 cp s3://acme-archive/contract.pdf - \
   | pdfcpu extract --mode page --pages 3 - - \
   | aws s3 cp - s3://acme-archive/pages/contract-page-3.pdf
```
