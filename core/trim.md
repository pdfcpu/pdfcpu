---
layout: default
---

# Trim

Generate a trimmed version of `inFile` for selected pages. Have a look at some [examples](#examples).

## Usage

```
pdfcpu trim [-v(erbose)|vv] [-q(uiet)] -pages pageSelection [-upw userpw] [-opw ownerpw] inFile [outFile]
```

<br>

### Flags

| flag                             | description     | required
|:---------------------------------|:----------------|:--------
| [verbose](../getting_started/common_flags.md) | turn on logging | no
| [vv](../getting_started/common_flags.md)     | verbose logging | no
| [quiet](../getting_started/common_flags.md)   | quiet mode      | no
| [pages](../getting_started/page_selection) | page selection  | yes
| [upw](../getting_started/common_flags.md)    | user password   | no
| [opw](../getting_started/common_flags.md)    | owner password  | no

<br>

### Arguments

| name         | description         | required | default
|:-------------|:--------------------|:---------|:-
| inFile       | PDF input file      | yes
| outFile      | PDF output file     | no       | inFile_new.pdf

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
pdfcpu trim -pages even test.pdf test_trimmed.pdf
```

<br>
Create a single page PDF file for a specific page number:

```sh
pdfcpu trim -pages 1 test.pdf firstPage.pdf
```

<br>
Get rid of the catalog and trailing index:

```sh
pdfcpu trim -pages '!2-4,!12-' book.pdf essence.pdf
```