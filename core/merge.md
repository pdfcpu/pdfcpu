---
layout: default
---

# Merge

Merge 2 or more PDF files into `outFile`. Have a look at some [examples](#examples).

## Usage

```
pdfcpu merge [-m(ode) create|append] [-s(ort) -b(ookmarks)] outFile inFile...
```

<br>

### Flags

| name     | description     | default | required
|:---------|:----------------|:--------|:--
| m(ode)   | create, append  | create  | no
| sort     | sort inFiles if present | unsorted | no
| bookmarks| create bookmarks | yes   | no


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

| name         | description         | required
|:-------------|:--------------------|:--------
| outFile      | PDF output file     | yes  
| inFile...    | at least 2 PDF input files subject to concatenation | yes

<br>

## Restrictions

The following PDF elements are not carried over into the merged document:

* Struct Trees

<br>

## Examples

pdfcpu respects the order of the provided input files and merges accordingly. Merge three input files into `out.pdf` by concatenating `in3.pdf` to `in2.pdf` and the result to `in1.pdf`:

```sh
pdfcpu merge out.pdf in1.pdf in2.pdf in3.pdf
```

<br>

Merge all PDF Files in the current directory into `out.pdf`:

```sh
pdfcpu merge out.pdf *.pdf
```

<br>

Merge some PDF files into an existing PDF file `out.pdf`:

```sh
pdfcpu merge -mode append out.pdf in1.pdf in2.pdf in3.pdf
```