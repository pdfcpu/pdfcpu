---
layout: default
title: "Merge"
---

# Merge

Merge 2 or more PDF files into `outFile`. Have a look at some [examples](#examples).

## Usage

```
pdfcpu merge outFile inFile... [flags]
```

<br>

### Flags

| name       | description                          | default   | required
|:-----------|:-------------------------------------|:----------|:--
| m(ode)     | create, append, zip                  | create    | no
| s(ort)     | sort inFiles if present              | unsorted  | no
| b(ookmarks)| create bookmarks                     | yes       | no
| d(ivider)  | insert separator between merged docs | no        | no
| opt(imize) | optimize before writing              | yes       | no
| rmsig      | remove signatures                    | no        | no

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| outFile      | PDF output file, use `-` to write to stdout | yes
| inFile...    | at least 2 PDF input files subject to concatenation, use `-` for one stdin input in create mode | yes

<br>

## Restrictions

The following PDF elements are not carried over into the merged document:

* Struct Trees

<br>

## Examples

pdfcpu respects the order of the provided input files and merges accordingly. Merge three input files into `out.pdf` by concatenating `in3.pdf` to `in2.pdf` and the result to `in1.pdf`:

```sh
$ pdfcpu merge out.pdf in1.pdf in2.pdf in3.pdf
```

<br>

Merge all PDF Files in the current directory into `out.pdf` and don't create bookmarks:

```sh
$ pdfcpu merge out.pdf *.pdf -b=f
```

<br>

Merge some PDF files into an existing PDF file `out.pdf` and create divider pages between the merged documents:

```sh
$ pdfcpu merge --mode append -divider out.pdf in1.pdf in2.pdf in3.pdf
```

<br>

Zip two files together (eg. like in 1a,1b,2a,2b..):
```sh
$ pdfcpu merge --mode zip out.pdf a.pdf b.pdf
```

<br>

Merge local PDFs and upload the result:

```sh
$ pdfcpu merge - quarterly/*.pdf \
   | aws s3 cp - s3://acme-reports/quarterly/merged.pdf
```

<br>

Merge one PDF streamed from S3 with local files:

```sh
$ aws s3 cp s3://acme-reports/cover.pdf - \
   | pdfcpu merge - - chapter1.pdf chapter2.pdf \
   | aws s3 cp - s3://acme-reports/book.pdf
```
