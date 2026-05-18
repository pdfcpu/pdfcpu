---
layout: default
title: "Set Page Mode"
---

# Set Page Mode

This command configures the page mode that shall be used when the document is opened.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu pagemode set inFile value [ outFile ] [flags]
```

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------------------------
| inFile       | PDF input file, use `-` to read from stdin      | yes
| value        | page layout mode    | yes
| outFile      | PDF output file, use `-` to write to stdout     | no

<br>

### Page Modes

| name           | description
|:---------------|:-------------------------------------------------
| UseNone        | Neither document outline nor thumbnail images visible (default)
| UseOutlines    | Document outline visible
| UseThumbs      | Thumbnail images visible
| FullScreen     | Optional content group panel visible (since PDF 1.5)
| UseOC          | Display the pages two at a time, with odd-numbered pages on the right
| UseAttachments | Attachments panel visible (since PDF 1.6)

<br>

## Examples

Set pagemode for `test.pdf` (case agnostic):

```sh
$ pdfcpu pagemode set test.pdf usethumbs
$ pdfcpu pagemode list test.pdf
UseThumbs
```

<br>

Set page mode while reading and writing the PDF through pipes:

```sh
$ aws s3 cp s3://acme-publishing/ebook.pdf - \
   | pdfcpu pagemode set - UseOutlines - \
   | aws s3 cp - s3://acme-publishing/ebook-outlines.pdf
```
