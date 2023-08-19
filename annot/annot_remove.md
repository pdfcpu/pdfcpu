---
layout: default
---

# Remove Annotations

This command removes annotation from a PDF document by object number.
Have a look at some [examples](#examples).

## Usage

```
pdfcpu annotations remove [-p(ages) selectedPages] inFile [outFile] [objNr|annotId|annotType]...
```

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
| inFile       | PDF input file      | yes
| outFile      | PDF output file     | no
| objNr...     | one or more objNrs  | no
| annotId...   | one or more annotIds  | no
| annotType... | one or more annotTypes  | no

<br>

## Examples

Remove annotation with object number 575 as taken from the output from `pdfcpu annot list`:
```
$ pdfcpu annot remove test.pdf 575
writing test.pdf...
pages: all
```

<br>

Remove annotations for first 5 pages:
```
$ pdfcpu annot remove -pages 1-5 test.pdf
writing test.pdf...
pages: 1,2,3,4,5
```

<br>

Remove all annotations:
```
$ pdfcpu annot remove test.pdf
writing test.pdf...
pages: all
```