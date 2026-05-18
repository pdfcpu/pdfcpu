---
layout: default
title: "Reset Page Layout"
---

# Reset Page Layout

This command resets the configured page layout for a PDF file.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu pagelayout reset inFile [ outFile ] [flags]
```

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file, use `-` to read from stdin      | yes
| outFile      | PDF output file, use `-` to write to stdout     | no

<br>

## Examples

Reset page layout for `test.pdf`:
```sh
$ pdfcpu pagelayout reset test.pdf
$ pdfcpu pagelayout list test.pdf
No page layout set, PDF viewers will default to "SinglePage"
```
<br>

Reset page layout while reading and writing the PDF through pipes:

```sh
$ aws s3 cp s3://acme-publishing/ebook-spreads.pdf - \
   | pdfcpu pagelayout reset - - \
   | aws s3 cp - s3://acme-publishing/ebook-default-layout.pdf
```
