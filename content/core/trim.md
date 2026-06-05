---
layout: default
title: "Trim"
---

# Trim

Generate a new PDF containing only the selected pages of `inFile`.

`trim` removes pages from the document by page selection. It does not crop page content, change page boxes, or remove white margins on a page. Use `crop` for changing the visible page area.

The output keeps the selected pages in their original order and writes them to `outFile`. If `outFile` is omitted, `inFile` is overwritten.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu trim inFile [outFile] [flags]
```

<br>

### Flags

| name                                         | description    | required
|:---------------------------------------------|:---------------|---------
| [p(ages)](/getting_started/page_selection) | selected pages | yes

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description         | required | default
|:-------------|:--------------------|:---------|:-
| inFile       | PDF input file, use `-` to read from stdin      | yes
| outFile      | PDF output file, use `-` to write to stdout     | no       | inFile

<br>

## Restrictions

The following PDF elements are not carried over into the trimmed document:

* Annotations
* Outlines
* Struct Trees
* Forms

<br>

## Examples

Get rid of unwanted blank pages:

```sh
$ pdfcpu trim --pages even test.pdf test_trimmed.pdf
```

<br>
Create a single page PDF file for a specific page number:

```sh
$ pdfcpu trim -p 1 test.pdf firstPage.pdf
```

<br>
Get rid of the catalog and trailing index:

```sh
$ pdfcpu trim book.pdf essence.pdf --pages '!2-4,!12-'
```

<br>

Trim a streamed PDF and upload the result:

```sh
$ aws s3 cp s3://acme-cases/filing.pdf - \
   | pdfcpu trim --pages 1-12 - - \
   | aws s3 cp - s3://acme-cases/filing-trimmed.pdf
```
