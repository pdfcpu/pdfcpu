---
layout: default
title: "Reset Viewer Preferences"
---

# Reset Viewer Preferences

This command resets the viewer preferences for a PDF document. 

Have a look at some [examples](#examples).

## Usage

```
pdfcpu viewerpref reset inFile [ outFile ] [flags]
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

Reset the viewer preferences for `test.pdf`:

```sh
$ pdfcpu viewerpref reset test.pdf
```

<br>

Reset viewer preferences while reading and writing the PDF through pipes:

```sh
$ aws s3 cp s3://acme-print/catalog-print-ready.pdf - \
   | pdfcpu viewerpref reset - - \
   | aws s3 cp - s3://acme-print/catalog-default-viewer.pdf
```
