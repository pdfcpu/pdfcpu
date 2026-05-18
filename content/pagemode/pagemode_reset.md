---
layout: default
title: "Reset Page Mode"
---

# Reset Page Mode

This command resets the configured page mode for a PDF file.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu pagemode reset inFile [ outFile ] [flags]
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

Reset page mode for `test.pdf`:

```sh
$ pdfcpu pagemode reset test.pdf
$ pdfcpu pagemode list test.pdf
No page mode set, PDF viewers will default to "UseNone"
```

<br>

Reset page mode while reading and writing the PDF through pipes:

```sh
$ aws s3 cp s3://acme-publishing/ebook-outlines.pdf - \
   | pdfcpu pagemode reset - - \
   | aws s3 cp - s3://acme-publishing/ebook-default-mode.pdf
```
