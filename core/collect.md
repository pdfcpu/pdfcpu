---
layout: default
---

# Collect

* Create a custom PDF page sequence.

* Arrange your pages in any order you like.

* Pages may appear multiple times.

* Have a look at some [examples](#examples).


## Usage

```
pdfcpu collect [-v(erbose)|vv] [-q(uiet)] -pages selectedPages [-upw userpw] [-opw ownerpw] inFile [outFile]
```

<br>

### Flags

| flag                                          | description     | required
|:----------------------------------------------|:----------------|:--------
| [verbose](../getting_started/common_flags.md) | turn on logging | no
| [vv](../getting_started/common_flags.md)      | verbose logging | no
| [quiet](../getting_started/common_flags.md)   | quiet mode      | no
| [pages](../getting_started/page_selection)    | page selection  | yes
| [upw](../getting_started/common_flags.md)     | user password   | no
| [opw](../getting_started/common_flags.md)     | owner password  | no

<br>

### Arguments

| name         | description         | required | default
|:-------------|:--------------------|:---------|:-
| inFile       | PDF input file      | yes
| outFile      | PDF output file     | no       | inFile

<br>

## Examples

Create a custom page collection from `in.pdf` and write the result to `out.pdf`.
Begin with 3 instances of page 1 then append the rest of the file excluding the last page:

```sh
pdfcpu collect -pages 1,1,1,2-l-1 in.pdf out.pdf
writing sequ.pdf ...
```
