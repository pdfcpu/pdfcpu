---
layout: default
title: "Collect"
---

# Collect

* Create a custom PDF page sequence.

* Arrange your pages in any order you like.

* Pages may appear multiple times.

* Have a look at some [examples](#examples).


## Usage

```
pdfcpu collect inFile [outFile] --pages
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

## Examples

Create a custom page collection from `in.pdf` and write the result to `out.pdf`.
Begin with 3 instances of page 1 then append the rest of the file excluding the last page:

```sh
$ pdfcpu collect in.pdf out.pdf --pages 1,1,1,2-l-1 
writing sequ.pdf ...
```

<br>

Collect selected pages from stdin and write the result to stdout:

```sh
$ aws s3 cp s3://acme-dataroom/deck.pdf - \
   | pdfcpu collect --pages 1,3,5-7 - - \
   | aws s3 cp - s3://acme-dataroom/executive-extract.pdf
```
