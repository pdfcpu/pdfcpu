---
layout: default
---

# Merge

Merge 2 or more PDF files into `outFile`. Have a look at some [examples](#examples).

## Usage

```
pdfcpu merge [-v(erbose)|vv] [-q(uiet)] [-mode create|append] outFile inFile...
```

<br>

### Flags

| name                             | description     | default | required
|:---------------------------------|:-----------------------------|:-------|:--
| [verbose](../getting_started/common_flags.md) | turn on logging |        | no
| [vv](../getting_started/common_flags.md)      | verbose logging |        | no
| [quiet](../getting_started/common_flags.md)   | quiet mode      |        | no
| mode                                          | create, append  | create | no


<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| outFile      | PDF output file     | yes  
| inFile...    | at least 2 PDF input files subject to concatenation | yes

<br>

## Restrictions

The following PDF elements are not carried over into the merged document:

* Outlines
* Struct Trees
* Forms

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