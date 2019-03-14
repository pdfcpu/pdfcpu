---
layout: default
---

# Split

Generate a set of PDF files for `inFile` in `outDir` according to given `span` value. Also check out the [extract pages](../extract/extract_pages.md) command which gives you similar functionality. Have a look at some [examples](#examples).

## Usage

```
pdfcpu split [-v(erbose)|vv] [-upw userpw] [-opw ownerpw] inFile outDir [span]
```

<br>

### Flags

| flag                             | description         | required
|:---------------------------------|:--------------------|:--------
| [verbose](../getting_started.md) | turn on logging     | no
| [vv](../getting_started.md)      | verbose logging     | no
| [upw](../getting_started.md)     | user password       | no
| [opw](../getting_started.md)     | owner password      | no

<br>

### Arguments

| name         | description         | required | default
|:-------------|:--------------------|:---------|:-
| inFile       | PDF input file      | yes
| outDir       | output directory    | yes
| span         | split span in pages | no       | 1

<br>

## Restrictions

The following PDF elements are not carried over into the output files:

* Annotations
* Outlines
* Struct Trees
* Forms

<br>

## Examples

Split up a PDF input file into single page PDF files in `out`:
```sh
pdfcpu split test.pdf out
``` 

<br>

Split up a PDF input file into individual PDF files for every sheet of paper. Every PDF output file in `out` spans 2 pages of the original:

```sh
pdfcpu split test.pdf out 2
```
