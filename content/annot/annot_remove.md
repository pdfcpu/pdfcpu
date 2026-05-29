---
layout: default
title: "Remove Annotations"
---

# Remove Annotations

This command removes annotation from a PDF document by object number.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu annotations remove inFile [ outFile ] [ objNr | annotId | annotType]... [flags]
```

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file, use `-` to read from stdin      | yes
| outFile      | PDF output file, use `-` to write to stdout     | no
| objNr...     | one or more objNrs  | no
| annotId...   | one or more annotIds  | no
| annotType... | one or more annotTypes  | no

<br>

## Examples

Remove annotation with object number 575 as taken from the output from `pdfcpu annotations list`:
```
$ pdfcpu annotations remove test.pdf 575
writing test.pdf...
pages: all
```

<br>

Remove annotations for first 5 pages:
```
$ pdfcpu annotations remove test.pdf --pages 1-5
writing test.pdf...
pages: 1,2,3,4,5
```

<br>

Remove all annotations:
```
$ pdfcpu annotations remove test.pdf
writing test.pdf...
pages: all
```

<br>

Remove link annotations from a streamed PDF and upload the result:

```sh
$ aws s3 cp s3://acme-redaction/review.pdf - \
   | pdfcpu annotations remove - - Link \
   | aws s3 cp - s3://acme-redaction/review-no-links.pdf
```
