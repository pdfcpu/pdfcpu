---
layout: default
---

# Split

Generate a set of PDF files for `inFile` in `outDir` according to given `span` value. Also check out the [extract pages](../extract/extract_pages.md) command which gives you similar functionality. Have a look at some [examples](#examples).

## Usage

```
pdfcpu split [-m(ode) span|bookmark|page] inFile outDir [span|pageNr...]
```

<br>

### Flags

| name       | required | value    | description
|:-----------|:---------|:---------|:-----------
| m(ode)     | no       | span     | Split into PDF files with span pages each (default).
|            |          | bookmark | Split into PDF files representing sections defined by existing bookmarks.
|            |          | page     | Split before specific page number(s).

<br>

### Common Flags

| name                                            | description     | values
|:------------------------------------------------|:----------------|:-------
| [v(erbose)](../getting_started/common_flags.md) | turn on logging |
| [vv](../getting_started/common_flags.md)        | verbose logging |
| [q(uiet)](../getting_started/common_flags.md)   | quiet mode      |
| [u(nit)](../getting_started/common_flags.md)    | display unit    | po(ints),in(ches),cm,mm
| [c(onf)](../getting_started/common_flags.md)       | config dir      | $path, disable
| [upw](../getting_started/common_flags.md)          | user password   |
| [opw](../getting_started/common_flags.md)          | owner password  |

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

Split a PDF file into single page PDF files in `out`:
```sh
$ pdfcpu split test.pdf out
``` 

<br>

Split a PDF file into individual PDF files for every sheet of paper. Every PDF output file in `out` spans 2 pages of the original:
```sh
$ pdfcpu split test.pdf out 2
```

<br>

Split a PDF file along its bookmarks:
```sh
$ pdfcpu split -m bookmark test.pdf out
```

<br>

Split a PDF file before pages 2,4,10:
```sh
$ pdfcpu split -m page test.pdf out 2 4 10
```




