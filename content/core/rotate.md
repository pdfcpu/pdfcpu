---
layout: default
title: "Rotate"
---

# Rotate

Rotate selected pages of `inFile` clockwise by a multiple of 90 degrees. Have a look at some [examples](#examples).

## Usage

```
pdfcpu rotate inFile rotation [outFile] [flags]
```

<br>

### Flags

| name                                         | description    | required
|:---------------------------------------------|:---------------|---------
| [p(ages)](/getting_started/page_selection) | selected pages | no

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description     | required | values
|:-------------|:----------------|:---------|:-
| inFile       | PDF input file, use `-` to read from stdin      | yes      |
| rotation     | rotation angle  | yes      | -270, -180, -90, 90, 180, 270
| outFile      | PDF output file, use `-` to write to stdout     | no       |

<br>

## Examples

Rotate all pages of a PDF file clockwise by 90 degrees:

```sh
$ pdfcpu rotate test.pdf 90
```

<br>
Rotate the first two pages counter clockwise by 90 degrees:

```sh
$ pdfcpu rotate --pages 1-2 test.pdf -90
```

<br>

Rotate streamed input and write the result to stdout:

```sh
$ aws s3 cp s3://acme-scans/batch.pdf - \
   | pdfcpu rotate --pages odd - 90 - \
   | aws s3 cp - s3://acme-scans/batch-rotated.pdf
```
