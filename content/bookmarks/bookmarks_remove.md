---
layout: default
title: "Remove Bookmarks"
---

# Remove Bookmarks

This command removes all bookmarks.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu bookmarks remove inFile [ outFile ] [flags]
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

 Remove all bookmarks:

```
$ pdfcpu bookmarks remove bookmarkSimple.pdf

$ pdfcpu bookmarks list bookmarkSimple.pdf
no bookmarks available
```

<br>

Remove bookmarks while reading the PDF from stdin:

```sh
$ aws s3 cp s3://acme-manuals/product-bookmarked.pdf - \
   | pdfcpu bookmarks remove - - \
   | aws s3 cp - s3://acme-manuals/product-flat.pdf
```
