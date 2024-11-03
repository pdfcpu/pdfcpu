---
layout: default
---

# Merge

Merge 2 or more PDF files into `outFile`. Have a look at some [examples](#examples).

## Usage

```
pdfcpu merge [-m(ode) create|append|zip] [-s(ort) -b(ookmarks) -d(ivider) -opt(imize)] outFile inFile...
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


<br>

### Common Flags

| name                                            | description     | values
|:------------------------------------------------|:----------------|:-------
| [v(erbose)](../getting_started/common_flags.md) | turn on logging |
| [vv](../getting_started/common_flags.md)        | verbose logging |
| [q(uiet)](../getting_started/common_flags.md)   | quiet mode      |
| [-o(ffline)](../getting_started/common_flags.md)| disable http traffic |                                 | 
| [c(onf)](../getting_started/common_flags.md)    | config dir      | $path, disable
| [opw](../getting_started/common_flags.md)       | owner password  |
| [upw](../getting_started/common_flags.md)       | user password   |
| [u(nit)](../getting_started/common_flags.md)    | display unit    | po(ints),in(ches),cm,mm

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
$ pdfcpu merge out.pdf in1.pdf in2.pdf in3.pdf
```

<br>

Merge all PDF Files in the current directory into `out.pdf` and don't create bookmarks:

```sh
$ pdfcpu merge -b=f out.pdf *.pdf
```

<br>

Merge some PDF files into an existing PDF file `out.pdf` and create divider pages between the merged docuements:

```sh
$ pdfcpu merge -mode append -divider out.pdf in1.pdf in2.pdf in3.pdf
```

<br>

Zip two files together (eg. like in 1a,1b,2a,2b..):
```sh
$ pdfcpu merge -mode zip out.pdf a.pdf b.pdf
```
