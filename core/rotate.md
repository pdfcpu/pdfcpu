---
layout: default
---

# Rotate

Rotate selected pages of `inFile` clockwise by a multiple of 90 degrees. Have a look at some [examples](#examples).

## Usage

```
pdfcpu rotate [-v(erbose)|vv] [-q(uiet)] [-pages pageSelection] [-upw userpw] [-opw ownerpw] inFile rotation
```

<br>

### Flags

| flag                             | description     | required | values
|:---------------------------------|:----------------|:---------|:------
| [verbose](../getting_started/common_flags.md) | turn on logging | no
| [vv](../getting_started/common_flags.md)      | verbose logging | no
| [quiet](../getting_started/common_flags.md)   | quiet mode      | no
| [pages](../getting_started/page_selection) | page selection  | no | all pages
| [upw](../getting_started/common_flags.md)     | user password   | no
| [opw](../getting_started/common_flags.md)     | owner password  | no

<br>

### Arguments

| name         | description    | required | values
|:-------------|:---------------|:---------|:-
| inFile       | PDF input file | yes
| rotation     | rotation angle | yes      | -270, -180, -90, 90, 180, 270

<br>

## Examples

Rotate all pages of a PDF file clockwise by 90 degrees:

```sh
pdfcpu rotate test.pdf 90
```

<br>
Rotate the first two pages counter clockwise by 90 degrees:

```sh
pdfcpu rotate -pages 1-2 test.pdf -90
```