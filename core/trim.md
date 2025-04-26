---
layout: default
---

# Trim

Generate a trimmed version of `inFile` for selected pages. Have a look at some [examples](#examples).

## Usage

```
pdfcpu trim -p(ages) selectedPages inFile [outFile]
```

<br>

### Flags

| name                                         | description    | required
|:---------------------------------------------|:---------------|---------
| [p(ages)](../getting_started/page_selection) | selected pages | yes

<br>

### [Common Flags](../getting_started/common_flags)

### Arguments

| name         | description         | required | default
|:-------------|:--------------------|:---------|:-
| inFile       | PDF input file      | yes
| outFile      | PDF output file     | no       | inFile

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
$ pdfcpu trim -pages even test.pdf test_trimmed.pdf
```

<br>
Create a single page PDF file for a specific page number:

```sh
$ pdfcpu trim -pages 1 test.pdf firstPage.pdf
```

<br>
Get rid of the catalog and trailing index:

```sh
$ pdfcpu trim -pages "!2-4,!12-" book.pdf essence.pdf
```